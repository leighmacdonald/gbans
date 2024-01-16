// Package app is the main application and entry point. It implements the action.Executor and io.Closer interfaces.
package app

import (
	"context"
	"fmt"
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
	db               *store.Store
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
	assetStore       AssetStore
	logListener      *logparse.UDPLogListener
	matchUUIDMap     fp.MutexMap[int, uuid.UUID]
	activityMu       *sync.RWMutex
	activity         []forumActivity
	netBlock         *NetworkBlocker
}

type ChatBot interface {
	SendPayload(payload discord.Payload)
	RegisterHandler(cmd discord.Cmd, handler discord.CommandHandler) error
}

type CommandHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error)

func New(conf config.Config, database *store.Store, bot ChatBot, logger *zap.Logger, assetStore AssetStore) *App {
	application := &App{
		discord:          bot,
		eb:               fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent](),
		db:               database,
		conf:             conf,
		assetStore:       assetStore,
		log:              logger,
		logFileChan:      make(chan *logFilePayload, 10),
		notificationChan: make(chan NotificationPayload, 5),
		steamGroups:      newSteamGroupMemberships(logger, database),
		matchUUIDMap:     fp.NewMutexMap[int, uuid.UUID](),
		patreon:          newPatreonManager(logger, conf, database),
		wordFilters:      newWordFilters(),
		mc:               newMetricCollector(),
		state:            newServerStateCollector(logger),
		activityMu:       &sync.RWMutex{},
		netBlock:         NewNetworkBlocker(),
	}

	application.setConfig(conf)

	application.warningTracker = newWarningTracker(logger, database, conf.Filter,
		onWarningHandler(application),
		onWarningExceeded(application))

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

func (app *App) setConfig(conf config.Config) {
	app.confMu.Lock()
	defer app.confMu.Unlock()

	app.conf = conf
}

func (app *App) initLogAddress() {
	conf := app.config()

	if conf.Debug.AddRCONLogAddress == "" {
		return
	}

	time.Sleep(time.Second * 60)

	app.state.connectionsMu.RLock()
	defer app.state.connectionsMu.RUnlock()

	for _, server := range app.state.connections {
		if server.RemoteConsole == nil {
			continue
		}

		_, errExec := server.Exec(fmt.Sprintf("logaddress_add %s", conf.Debug.AddRCONLogAddress))
		if errExec != nil {
			app.log.Error("Failed to set logaddress")
		}
	}
}

func firstTimeSetup(ctx context.Context, conf config.Config, database *store.Store) error {
	if !conf.General.Owner.Valid() {
		return errors.New("Configured owner is not a valid steam64")
	}

	localCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var owner store.Person

	if errRootUser := database.GetPersonBySteamID(localCtx, conf.General.Owner, &owner); errRootUser != nil {
		if !errors.Is(errRootUser, store.ErrNoResult) {
			return errors.Wrapf(errRootUser, "Failed first time setup")
		}

		newOwner := store.NewPerson(conf.General.Owner)
		newOwner.PermissionLevel = consts.PAdmin

		if errSave := database.SavePerson(localCtx, &newOwner); errSave != nil {
			return errors.Wrap(errSave, "Failed to create admin user")
		}

		newsEntry := store.NewsEntry{
			Title:       "Welcome to gbans",
			BodyMD:      "This is an *example* **news** entry.",
			IsPublished: true,
			CreatedOn:   time.Now(),
			UpdatedOn:   time.Now(),
		}

		if errSave := database.SaveNewsArticle(localCtx, &newsEntry); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample news entry")
		}

		server := store.NewServer("server-1", "127.0.0.1", 27015)
		server.CC = "jp"
		server.RCON = "example_rcon"
		server.Latitude = 35.652832
		server.Longitude = 139.839478
		server.Name = "Example Server"
		server.LogSecret = 12345678
		server.Region = "asia"

		if errSave := database.SaveServer(localCtx, &server); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample server entry")
		}

		page := wiki.Page{
			Slug:      wiki.RootSlug,
			BodyMD:    "# Welcome to the wiki",
			Revision:  1,
			CreatedOn: time.Now(),
			UpdatedOn: time.Now(),
		}

		if errSave := database.SaveWikiPage(localCtx, &page); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample wiki entry")
		}
	}

	if errWeapons := database.LoadWeapons(ctx); errWeapons != nil {
		return errors.Wrap(errWeapons, "Failed to load weapons")
	}

	return nil
}

func (app *App) Init(ctx context.Context) error {
	app.log.Info("Starting gbans...",
		zap.String("version", BuildVersion),
		zap.String("commit", BuildCommit),
		zap.String("date", BuildDate))

	if setupErr := firstTimeSetup(ctx, app.conf, app.db); setupErr != nil {
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
	sources, errSource := app.db.GetCIDRBlockSources(ctx)
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

		go func(src store.CIDRBlockSource) {
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

	whitelists, errWhitelists := app.db.GetCIDRBlockWhitelists(ctx)
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

type newUserWarning struct {
	userMessage store.PersonMessage
	userWarning
}

func (app *App) chatRecorder(ctx context.Context) {
	var (
		log             = app.log.Named("chatRecorder")
		serverEventChan = make(chan logparse.ServerEvent)
	)

	if errRegister := app.eb.Consume(serverEventChan, logparse.Say, logparse.SayTeam); errRegister != nil {
		log.Warn("logWriter Tried to register duplicate reader channel", zap.Error(errRegister))

		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-serverEventChan:
			switch evt.EventType {
			case logparse.Say:
				fallthrough
			case logparse.SayTeam:
				newServerEvent, ok := evt.Event.(logparse.SayEvt)
				if !ok {
					continue
				}

				if newServerEvent.Msg == "" {
					log.Warn("Empty person message body, skipping")

					continue
				}

				var author store.Person
				if errPerson := app.db.GetPersonBySteamID(ctx, newServerEvent.SID, &author); errPerson != nil {
					log.Error("Failed to add chat history, could not get author", zap.Error(errPerson))

					continue
				}

				matchID, _ := app.matchUUIDMap.Get(evt.ServerID)

				msg := store.PersonMessage{
					SteamID:     newServerEvent.SID,
					PersonaName: strings.ToValidUTF8(newServerEvent.Name, "_"),
					ServerName:  evt.ServerName,
					ServerID:    evt.ServerID,
					Body:        strings.ToValidUTF8(newServerEvent.Msg, "_"),
					Team:        newServerEvent.Team,
					CreatedOn:   newServerEvent.CreatedOn,
					MatchID:     matchID,
				}

				if errChat := app.db.AddChatHistory(ctx, &msg); errChat != nil {
					log.Error("Failed to add chat history", zap.Error(errChat))

					continue
				}

				// app.incomingGameChat <- msg

				go func(userMsg store.PersonMessage) {
					if msg.ServerName == "localhost-1" {
						log.Debug("Chat message",
							zap.Int64("id", msg.PersonMessageID),
							zap.String("server", evt.ServerName),
							zap.String("name", newServerEvent.Name),
							zap.String("steam_id", newServerEvent.SID.String()),
							zap.Bool("team", msg.Team),
							zap.String("message", msg.Body))
					}

					matchedWord, matchedFilter := app.wordFilters.findFilteredWordMatch(userMsg.Body)
					if matchedFilter != nil {
						if errSaveMatch := app.db.AddMessageFilterMatch(ctx, userMsg.PersonMessageID, matchedFilter.FilterID); errSaveMatch != nil {
							log.Error("Failed to save message match status", zap.Error(errSaveMatch))
						}

						app.warningTracker.warningChan <- newUserWarning{
							userMessage: userMsg,
							userWarning: userWarning{
								WarnReason:    store.Language,
								Message:       userMsg.Body,
								Matched:       matchedWord,
								MatchedFilter: matchedFilter,
								CreatedOn:     time.Now(),
								Personaname:   userMsg.PersonaName,
								Avatar:        userMsg.AvatarHash,
								ServerName:    userMsg.ServerName,
								ServerID:      userMsg.ServerID,
								SteamID:       userMsg.SteamID.String(),
							},
						}
					}
				}(msg)
			}
		}
	}
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
			var person store.Person
			if errPerson := app.PersonBySID(ctx, newServerEvent.SID, &person); errPerson != nil {
				log.Error("Failed to load person", zap.Error(errPerson))

				continue
			}

			conn := store.PersonConnection{
				IPAddr:      parsedAddr,
				SteamID:     newServerEvent.SID,
				PersonaName: strings.ToValidUTF8(newServerEvent.Name, "_"),
				CreatedOn:   newServerEvent.CreatedOn,
				ServerID:    evt.ServerID,
			}

			lCtx, cancel := context.WithTimeout(ctx, time.Second*5)
			if errChat := app.db.AddConnectionHistory(lCtx, &conn); errChat != nil {
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

	words, count, errGetFilters := app.db.GetFilters(localCtx, store.FiltersQueryFilter{})
	if errGetFilters != nil {
		if errors.Is(errGetFilters, store.ErrNoResult) {
			return nil
		}

		return errors.Wrap(errGetFilters, "Failed to fetch filters")
	}

	app.wordFilters.importFilteredWords(words)

	app.log.Debug("Loaded word filters", zap.Int64("count", count))

	return nil
}

func (app *App) startWorkers(ctx context.Context) {
	go app.patreon.updater(ctx)
	go app.banSweeper(ctx)
	go app.profileUpdater(ctx)
	go app.warningTracker.start(ctx)
	go app.logReader(ctx, app.config().Debug.WriteUnhandledLogEvents)
	go app.initLogSrc(ctx)
	go logMetricsConsumer(ctx, app.mc, app.eb, app.log)
	go app.matchSummarizer(ctx)
	go app.chatRecorder(ctx)
	go app.playerConnectionWriter(ctx)
	go app.steamGroups.start(ctx)
	go cleanupTasks(ctx, app.db, app.log)
	go app.showReportMeta(ctx)
	go app.notificationSender(ctx)
	go app.demoCleaner(ctx)
	go app.stateUpdater(ctx)
	go app.forumActivityUpdater(ctx)
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

	servers, _, errServers := app.db.GetServers(serversCtx, store.ServerQueryFilter{
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

// PersonBySID fetches the person from the database, updating the PlayerSummary if it out of date.
func (app *App) PersonBySID(ctx context.Context, sid steamid.SID64, person *store.Person) error {
	if errGetPerson := app.db.GetOrCreatePersonBySteamID(ctx, sid, person); errGetPerson != nil {
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
			app.log.Warn("Failed to update profile summary", zap.Error(errSummaries), zap.Int64("sid", sid.Int64()))
			// return errors.Errorf("Failed to fetch Player summary for %d", sid)
		}

		vac, errBans := thirdparty.FetchPlayerBans(ctx, app.log, steamid.Collection{sid})
		if errBans != nil || len(vac) != 1 {
			app.log.Warn("Failed to update ban status", zap.Error(errBans), zap.Int64("sid", sid.Int64()))
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
	if errSavePerson := app.db.SavePerson(ctx, person); errSavePerson != nil {
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
		sids, errIds := app.db.GetSteamIdsAbove(ctx, notification.MinPerms)
		if errIds != nil {
			return errors.Wrap(errIds, "Failed to fetch steamids for notification")
		}

		notification.Sids = append(notification.Sids, sids...)
	}

	uniqueIds := fp.Uniq(notification.Sids)

	people, errPeople := app.db.GetPeopleBySteamID(ctx, uniqueIds)
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
		if errSend := app.db.SendNotification(ctx, sid, notification.Severity,
			notification.Message, notification.Link); errSend != nil {
			app.log.Error("Failed to send notification", zap.Error(errSend))

			break
		}
	}

	return nil
}

func (app *App) currentActiveUsers() []forumActivity {
	app.activityMu.RLock()
	defer app.activityMu.RUnlock()

	return app.activity
}

// isOnIPWithBan checks if the address matches an existing user who is currently banned already. This
// function will always fail-open and allow players in if an error occurs.
func (app *App) isOnIPWithBan(ctx context.Context, steamID steamid.SID64, address net.IP) bool {
	existing := store.NewBannedPerson()
	if errMatch := app.db.GetBanByLastIP(ctx, address, &existing, false); errMatch != nil {
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

	if errSave := app.db.SaveBan(ctx, &existing.BanSteam); errSave != nil {
		app.log.Error("Could not update previous ban.", zap.Error(errSave))

		return false
	}

	conf := app.config()

	var newBan store.BanSteam
	if errNewBan := store.NewBanSteam(ctx,
		store.StringSID(conf.General.Owner.String()),
		store.StringSID(steamID.String()), duration, store.Evading, store.Evading.String(),
		"Connecting from same IP as banned player", store.System,
		0, store.Banned, false, &newBan); errNewBan != nil {
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
// func validateLink(ctx context.Context, database store.Store, sourceID action.Author, target *action.Author) error {
//	var p model.Person
//	if errGetPerson := database.GetPersonByDiscordID(ctx, string(sourceID), &p); errGetPerson != nil {
//		if errGetPerson == store.ErrNoResult {
//			return consts.ErrUnlinkedAccount
//		}
//		return consts.ErrInternal
//	}
//	*target = action.Author(p.SteamID.String())
//	return nil
// }
