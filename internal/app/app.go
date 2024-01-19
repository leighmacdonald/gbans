// Package app is the main application and entry point. It implements the action.Executor and io.Closer interfaces.
package app

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/match"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/s3"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	BuildVersion = "master" //nolint:gochecknoglobals
	BuildCommit  = ""       //nolint:gochecknoglobals
	BuildDate    = ""       //nolint:gochecknoglobals
)

type App struct {
	conf             config.Config
	confMu           sync.RWMutex
	discord          ChatBot
	db               store.Store
	log              *zap.Logger
	logFileChan      chan *logFilePayload
	notificationChan chan NotificationPayload
	state            *serverStateCollector
	steamGroups      *steamGroupMemberships
	patreon          *patreonManager
	eb               *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]
	wordFilters      *wordFilters
	warningTracker   *WarningTracker
	mc               *metricCollector
	assetStore       s3.AssetStore
	logListener      *logparse.UDPLogListener
	activityTracker  *activityTracker
	netBlock         *NetworkBlocker
	weaponMap        fp.MutexMap[logparse.Weapon, int]
	chatLogger       *chatLogger
	matchSummarizer  *match.Summarizer
}

type ChatBot interface {
	SendPayload(payload discord.Payload)
	RegisterHandler(cmd discord.Cmd, handler discord.CommandHandler) error
}

type CommandHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error)

func New(conf config.Config, database store.Store, bot ChatBot, logger *zap.Logger, assetStore s3.AssetStore) *App {
	eventBroadcaster := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()
	filters := newWordFilters()
	matchUUIDMap := fp.NewMutexMap[int, uuid.UUID]()
	application := &App{
		discord:          bot,
		eb:               eventBroadcaster,
		db:               database,
		conf:             conf,
		assetStore:       assetStore,
		log:              logger,
		logFileChan:      make(chan *logFilePayload, 10),
		notificationChan: make(chan NotificationPayload, 5),
		steamGroups:      newSteamGroupMemberships(logger, database),
		patreon:          newPatreonManager(logger, conf, database),
		wordFilters:      filters,
		mc:               newMetricCollector(),
		state:            newServerStateCollector(logger),
		activityTracker:  newForumActivity(logger),
		netBlock:         NewNetworkBlocker(),
		weaponMap:        fp.NewMutexMap[logparse.Weapon, int](),
	}

	application.setConfig(conf)

	application.warningTracker = newWarningTracker(logger, database, conf.Filter,
		onWarningHandler(application),
		onWarningExceeded(application))

	application.chatLogger = newChatLogger(logger, database, eventBroadcaster, filters, application.warningTracker, matchUUIDMap)

	application.matchSummarizer = match.NewSummarizer(logger, eventBroadcaster, matchUUIDMap, application.onMatchComplete)

	if conf.Discord.Enabled {
		if errReg := application.registerDiscordHandlers(); errReg != nil {
			panic(errReg)
		}
	}

	return application
}

func (app *App) bot() ChatBot { // nolint:ireturn
	return app.discord
}

func (app *App) config() config.Config {
	app.confMu.RLock()
	defer app.confMu.RUnlock()

	return app.conf
}

func (app *App) startWorkers(ctx context.Context) {
	go app.patreon.updater(ctx)
	go app.banSweeper(ctx)
	go app.profileUpdater(ctx)
	go app.warningTracker.start(ctx)
	go app.logReader(ctx, app.config().Debug.WriteUnhandledLogEvents)
	go app.initLogSrc(ctx)
	go logMetricsConsumer(ctx, app.mc, app.eb, app.log)
	go app.matchSummarizer.Start(ctx)
	go app.chatLogger.start(ctx)
	go app.playerConnectionWriter(ctx)
	go app.steamGroups.start(ctx)
	go cleanupTasks(ctx, app.db, app.log)
	go app.showReportMeta(ctx)
	go app.notificationSender(ctx)
	go app.demoCleaner(ctx)
	go app.stateUpdater(ctx)
	go app.activityTracker.start(ctx)
}

func (app *App) setConfig(conf config.Config) {
	app.confMu.Lock()
	defer app.confMu.Unlock()

	app.conf = conf
}

func firstTimeSetup(ctx context.Context, conf config.Config, database store.Store, weaponMap fp.MutexMap[logparse.Weapon, int]) error {
	if !conf.General.Owner.Valid() {
		return errors.New("Configured owner is not a valid steam64")
	}

	localCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var owner model.Person

	if errRootUser := store.GetPersonBySteamID(localCtx, database, conf.General.Owner, &owner); errRootUser != nil {
		if !errors.Is(errRootUser, store.ErrNoResult) {
			return errors.Wrapf(errRootUser, "Failed first time setup")
		}

		newOwner := model.NewPerson(conf.General.Owner)
		newOwner.PermissionLevel = consts.PAdmin

		if errSave := store.SavePerson(localCtx, database, &newOwner); errSave != nil {
			return errors.Wrap(errSave, "Failed to create admin user")
		}

		newsEntry := model.NewsEntry{
			Title:       "Welcome to gbans",
			BodyMD:      "This is an *example* **news** entry.",
			IsPublished: true,
			CreatedOn:   time.Now(),
			UpdatedOn:   time.Now(),
		}

		if errSave := store.SaveNewsArticle(localCtx, database, &newsEntry); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample news entry")
		}

		server := model.NewServer("server-1", "127.0.0.1", 27015)
		server.CC = "jp"
		server.RCON = "example_rcon"
		server.Latitude = 35.652832
		server.Longitude = 139.839478
		server.Name = "Example Server"
		server.LogSecret = 12345678
		server.Region = "asia"

		if errSave := store.SaveServer(localCtx, database, &server); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample server entry")
		}

		page := wiki.Page{
			Slug:      wiki.RootSlug,
			BodyMD:    "# Welcome to the wiki",
			Revision:  1,
			CreatedOn: time.Now(),
			UpdatedOn: time.Now(),
		}

		if errSave := store.SaveWikiPage(localCtx, database, &page); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample wiki entry")
		}
	}

	if errWeapons := store.LoadWeapons(ctx, database, weaponMap); errWeapons != nil {
		return errors.Wrap(errWeapons, "Failed to load weapons")
	}

	return nil
}

func (app *App) Init(ctx context.Context) error {
	app.log.Info("Starting gbans...",
		zap.String("version", BuildVersion),
		zap.String("commit", BuildCommit),
		zap.String("date", BuildDate))

	if setupErr := firstTimeSetup(ctx, app.conf, app.db, app.weaponMap); setupErr != nil {
		app.log.Fatal("Failed to do first time setup", zap.Error(setupErr))
	}

	// start the background goroutine workers
	app.startWorkers(ctx)

	// Load the filtered word set into memory
	if app.config().Filter.Enabled {
		if errFilter := app.LoadFilters(ctx); errFilter != nil {
			return errors.Wrap(errFilter, "Failed to load filters")
		}

		app.log.Info("Loaded filter list", zap.Int("count", len(app.wordFilters.wordFilters)))
	}

	if errBlocklist := app.loadNetBlocks(ctx); errBlocklist != nil {
		app.log.Error("Could not load CIDR block list", zap.Error(errBlocklist))
	}

	return nil
}

func (app *App) loadNetBlocks(ctx context.Context) error {
	sources, errSource := store.GetCIDRBlockSources(ctx, app.db)
	if errSource != nil {
		return errors.Wrap(errSource, "Failed to load block sources")
	}

	var total atomic.Int64

	waitGroup := sync.WaitGroup{}

	for _, source := range sources {
		if !source.Enabled {
			continue
		}

		waitGroup.Add(1)

		go func(src model.CIDRBlockSource) {
			defer waitGroup.Done()

			count, errAdd := app.netBlock.AddRemoteSource(ctx, src.Name, src.URL)
			if errAdd != nil {
				app.log.Error("Could not load remote source URL")
			}

			total.Add(count)
		}(source)
	}

	waitGroup.Wait()

	app.netBlock.Lock()
	_, netBlock, _ := net.ParseCIDR("192.168.0.0/24")
	app.netBlock.blocks["local"] = []*net.IPNet{netBlock}
	app.netBlock.Unlock()

	whitelists, errWhitelists := store.GetCIDRBlockWhitelists(ctx, app.db)
	if errWhitelists != nil {
		if !errors.Is(errWhitelists, store.ErrNoResult) {
			return errors.Wrap(errWhitelists, "Failed to load cidr block whitelists")
		}
	}

	for _, whitelist := range whitelists {
		app.netBlock.AddWhitelist(whitelist.CIDRBlockWhitelistID, whitelist.Address)
	}

	app.log.Info("Loaded cidr block lists",
		zap.Int64("cidr_blocks", total.Load()), zap.Int("whitelisted", len(whitelists)))

	return nil
}

func (app *App) StartHTTP(ctx context.Context) error {
	app.log.Info("Service status changed", zap.String("state", "ready"))
	defer app.log.Info("Service status changed", zap.String("state", "stopped"))

	if app.config().General.Mode == config.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	httpServer := newHTTPServer(ctx, app)

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)

		defer cancel()

		if errShutdown := httpServer.Shutdown(shutdownCtx); errShutdown != nil { //nolint:contextcheck
			app.log.Error("Error shutting down http service", zap.Error(errShutdown))
		}
	}()

	errServe := httpServer.ListenAndServe()
	if errServe != nil && !errors.Is(errServe, http.ErrServerClosed) {
		return errors.Wrap(errServe, "HTTP listener returned error")
	}

	return nil
}

func (app *App) playerConnectionWriter(ctx context.Context) {
	log := app.log.Named("playerConnectionWriter")

	serverEventChan := make(chan logparse.ServerEvent)
	if errRegister := app.eb.Consume(serverEventChan, logparse.Connected); errRegister != nil {
		log.Warn("logWriter Tried to register duplicate reader channel", zap.Error(errRegister))

		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-serverEventChan:
			newServerEvent, ok := evt.Event.(logparse.ConnectedEvt)
			if !ok {
				continue
			}

			if newServerEvent.Address == "" {
				log.Warn("Empty person message body, skipping")

				continue
			}

			parsedAddr := net.ParseIP(newServerEvent.Address)
			if parsedAddr == nil {
				log.Warn("Received invalid address", zap.String("addr", newServerEvent.Address))

				continue
			}

			// Maybe ignore these and wait for connect call to create?
			var person model.Person
			if errPerson := PersonBySID(ctx, app.db, newServerEvent.SID, &person); errPerson != nil {
				log.Error("Failed to load person", zap.Error(errPerson))

				continue
			}

			conn := model.PersonConnection{
				IPAddr:      parsedAddr,
				SteamID:     newServerEvent.SID,
				PersonaName: strings.ToValidUTF8(newServerEvent.Name, "_"),
				CreatedOn:   newServerEvent.CreatedOn,
				ServerID:    evt.ServerID,
			}

			lCtx, cancel := context.WithTimeout(ctx, time.Second*5)
			if errChat := store.AddConnectionHistory(lCtx, app.db, &conn); errChat != nil {
				log.Error("Failed to add connection history", zap.Error(errChat))
			}

			cancel()
		}
	}
}

type logFilePayload struct {
	ServerID   int
	ServerName string
	Lines      []string
	Map        string
}

// logReader is the fan-out orchestrator for game log events
// Registering receivers can be accomplished with app.eb.Broadcaster.
func (app *App) logReader(ctx context.Context, writeUnhandled bool) {
	var (
		log  = app.log.Named("logReader")
		file *os.File
	)

	if writeUnhandled {
		var errCreateFile error
		file, errCreateFile = os.Create("./unhandled_messages.log")

		if errCreateFile != nil {
			log.Fatal("Failed to open debug message log", zap.Error(errCreateFile))
		}

		defer func() {
			if errClose := file.Close(); errClose != nil {
				log.Error("Failed to close unhandled_messages.log", zap.Error(errClose))
			}
		}()
	}

	parser := logparse.NewLogParser()

	// playerStateCache := newPlayerCache(app.logger)
	for {
		select {
		case logFile := <-app.logFileChan:
			emitted := 0
			failed := 0
			unknown := 0
			ignored := 0

			for _, logLine := range logFile.Lines {
				parseResult, errParse := parser.Parse(logLine)
				if errParse != nil {
					continue
				}

				newServerEvent := logparse.ServerEvent{
					ServerName: logFile.ServerName,
					ServerID:   logFile.ServerID,
					Results:    parseResult,
				}

				if newServerEvent.EventType == logparse.IgnoredMsg {
					ignored++

					continue
				} else if newServerEvent.EventType == logparse.UnknownMsg {
					unknown++
					if writeUnhandled {
						if _, errWrite := file.WriteString(logLine + "\n"); errWrite != nil {
							log.Error("Failed to write debug log", zap.Error(errWrite))
						}
					}
				}

				app.eb.Emit(newServerEvent.EventType, newServerEvent)
				emitted++
			}

			log.Debug("Completed emitting logfile events",
				zap.Int("ok", emitted), zap.Int("failed", failed),
				zap.Int("unknown", unknown), zap.Int("ignored", ignored))
		case <-ctx.Done():
			log.Debug("logReader shutting down")

			return
		}
	}
}

func (app *App) LoadFilters(ctx context.Context) error {
	// TODO load external lists via http
	localCtx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	words, count, errGetFilters := store.GetFilters(localCtx, app.db, store.FiltersQueryFilter{})
	if errGetFilters != nil {
		if errors.Is(errGetFilters, store.ErrNoResult) {
			return nil
		}

		return errors.Wrap(errGetFilters, "Failed to fetch filters")
	}

	app.wordFilters.importFilters(words)

	app.log.Debug("Loaded word filters", zap.Int64("count", count))

	return nil
}

// UDP log sink.
func (app *App) initLogSrc(ctx context.Context) {
	logSrc, errLogSrc := logparse.NewUDPLogListener(app.log, app.config().Log.SrcdsLogAddr,
		func(eventType logparse.EventType, event logparse.ServerEvent) {
			app.eb.Emit(event.EventType, event)
		})

	if errLogSrc != nil {
		app.log.Fatal("Failed to setup udp log src", zap.Error(errLogSrc))
	}

	app.logListener = logSrc

	// TODO run on server config changes
	go app.updateSrcdsLogSecrets(ctx)

	app.logListener.Start(ctx)
}

func (app *App) updateSrcdsLogSecrets(ctx context.Context) {
	newSecrets := map[int]logparse.ServerIDMap{}
	serversCtx, cancelServers := context.WithTimeout(ctx, time.Second*5)

	defer cancelServers()

	servers, _, errServers := store.GetServers(serversCtx, app.db, store.ServerQueryFilter{
		IncludeDisabled: false,
		QueryFilter:     store.QueryFilter{Deleted: false},
	})
	if errServers != nil {
		app.log.Error("Failed to update srcds log secrets", zap.Error(errServers))

		return
	}

	for _, server := range servers {
		newSecrets[server.LogSecret] = logparse.ServerIDMap{
			ServerID:   server.ServerID,
			ServerName: server.ShortName,
		}
	}

	app.logListener.SetSecrets(newSecrets)
}

// PersonBySID fetches the person from the database, updating the PlayerSummary if it out of date or if
// the player does not already exist.
func PersonBySID(ctx context.Context, database store.Store, sid steamid.SID64, person *model.Person) error {
	if errGetPerson := store.GetOrCreatePersonBySteamID(ctx, database, sid, person); errGetPerson != nil {
		return errors.Wrapf(errGetPerson, "Failed to get person instance: %s", sid)
	}

	if person.IsNew || time.Since(person.UpdatedOnSteam) > time.Hour*24*30 {
		summaries, errSummaries := steamweb.PlayerSummaries(ctx, steamid.Collection{sid})
		if errSummaries != nil {
			return errors.Wrapf(errSummaries, "Failed to get Player summary: %v", errSummaries)
		}

		if len(summaries) > 0 {
			s := summaries[0]
			person.PlayerSummary = &s
		} else {
			return errors.New("Failed to update profile summary")
		}

		vac, errBans := thirdparty.FetchPlayerBans(ctx, steamid.Collection{sid})
		if errBans != nil || len(vac) != 1 {
			return errors.Wrap(errBans, "Failed to update ban status")
		} else {
			person.CommunityBanned = vac[0].CommunityBanned
			person.VACBans = vac[0].NumberOfVACBans
			person.GameBans = vac[0].NumberOfGameBans
			person.EconomyBan = steamweb.EconBanNone
			person.CommunityBanned = vac[0].CommunityBanned
			person.DaysSinceLastBan = vac[0].DaysSinceLastBan
		}

		person.UpdatedOnSteam = time.Now()
	}

	person.SteamID = sid
	if errSavePerson := store.SavePerson(ctx, database, person); errSavePerson != nil {
		return errors.Wrapf(errSavePerson, "Failed to save person")
	}

	return nil
}

// resolveSID is just a simple helper for calling steamid.ResolveSID64 with a timeout.
func resolveSID(ctx context.Context, sidStr string) (steamid.SID64, error) {
	localCtx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	sid64, errString := steamid.StringToSID64(sidStr)
	if errString == nil && sid64.Valid() {
		return sid64, nil
	}

	sid, errResolve := steamid.ResolveSID64(localCtx, sidStr)
	if errResolve != nil {
		return "", errors.Wrap(errResolve, "Failed to resolve vanity")
	}

	return sid, nil
}

type NotificationHandler struct{}

type NotificationPayload struct {
	MinPerms consts.Privilege
	Sids     steamid.Collection
	Severity consts.NotificationSeverity
	Message  string
	Link     string
}

func (app *App) SendNotification(ctx context.Context, notification NotificationPayload) error {
	// Collect all required ids
	if notification.MinPerms >= consts.PUser {
		sids, errIds := store.GetSteamIdsAbove(ctx, app.db, notification.MinPerms)
		if errIds != nil {
			return errors.Wrap(errIds, "Failed to fetch steamids for notification")
		}

		notification.Sids = append(notification.Sids, sids...)
	}

	uniqueIds := fp.Uniq(notification.Sids)

	people, errPeople := store.GetPeopleBySteamID(ctx, app.db, uniqueIds)
	if errPeople != nil && !errors.Is(errPeople, store.ErrNoResult) {
		return errors.Wrap(errPeople, "Failed to fetch people for notification")
	}

	var discordIds []string

	for _, p := range people {
		if p.DiscordID != "" {
			discordIds = append(discordIds, p.DiscordID)
		}
	}

	go func(ids []string, payload NotificationPayload) {
		for _, discordID := range ids {
			msgEmbed := discord.NewEmbed(app.config(), "Notification", payload.Message)
			if payload.Link != "" {
				msgEmbed.Embed().SetURL(payload.Link)
			}

			app.discord.SendPayload(discord.Payload{ChannelID: discordID, Embed: msgEmbed.Embed().Truncate().MessageEmbed})
		}
	}(discordIds, notification)

	// Broadcast to
	for _, sid := range uniqueIds {
		// Todo, prep stmt at least.
		if errSend := store.SendNotification(ctx, app.db, sid, notification.Severity,
			notification.Message, notification.Link); errSend != nil {
			app.log.Error("Failed to send notification", zap.Error(errSend))

			break
		}
	}

	return nil
}

// isOnIPWithBan checks if the address matches an existing user who is currently banned already. This
// function will always fail-open and allow players in if an error occurs.
func (app *App) isOnIPWithBan(ctx context.Context, steamID steamid.SID64, address net.IP) bool {
	existing := model.NewBannedPerson()
	if errMatch := store.GetBanByLastIP(ctx, app.db, address, &existing, false); errMatch != nil {
		if errors.Is(errMatch, store.ErrNoResult) {
			return false
		}

		app.log.Error("Could not load player by ip", zap.Error(errMatch))

		return false
	}

	duration, errDuration := util.ParseUserStringDuration("10y")
	if errDuration != nil {
		app.log.Error("Could not parse ban duration", zap.Error(errDuration))

		return false
	}

	existing.BanSteam.ValidUntil = time.Now().Add(duration)

	if errSave := store.SaveBan(ctx, app.db, &existing.BanSteam); errSave != nil {
		app.log.Error("Could not update previous ban.", zap.Error(errSave))

		return false
	}

	conf := app.config()

	var newBan model.BanSteam
	if errNewBan := model.NewBanSteam(ctx,
		model.StringSID(conf.General.Owner.String()),
		model.StringSID(steamID.String()), duration, model.Evading, model.Evading.String(),
		"Connecting from same IP as banned player", model.System,
		0, model.Banned, false, &newBan); errNewBan != nil {
		app.log.Error("Could not create evade ban", zap.Error(errDuration))

		return false
	}

	if errSave := app.BanSteam(ctx, &newBan); errSave != nil {
		app.log.Error("Could not save evade ban", zap.Error(errSave))

		return false
	}

	return true
}

// validateLink is used in the case of discord origin actions that require mapping the
// discord member ID to a SteamID so that we can track its use and apply permissions, etc.
//
// This function will replace the discord member id value in the target field with
// the found SteamID, if any.
// func validateLink(ctx context.Context, database db.Database, sourceID action.Author, target *action.Author) error {
//	var p model.Person
//	if errGetPerson := database.GetPersonByDiscordID(ctx, string(sourceID), &p); errGetPerson != nil {
//		if errGetPerson == db.ErrNoResult {
//			return consts.ErrUnlinkedAccount
//		}
//		return consts.ErrInternal
//	}
//	*target = action.Author(p.SteamID.String())
//	return nil
// }
