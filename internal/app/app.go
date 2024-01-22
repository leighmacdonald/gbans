package app

import (
	"context"
	"errors"
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
	"github.com/leighmacdonald/gbans/internal/activity"
	"github.com/leighmacdonald/gbans/internal/api"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/s3"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/steamid/v3/steamid"
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
	discord          *Bot
	db               store.Stores
	log              *zap.Logger
	logFileChan      chan *logFilePayload
	notificationChan chan NotificationPayload
	state            *state.Collector
	steamGroups      *SteamGroupMemberships
	patreon          *PatreonManager
	eb               *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]
	wordFilters      *WordFilters
	warningTracker   *Tracker
	mc               *metricCollector
	assetStore       s3.AssetStore
	logListener      *logparse.UDPLogListener
	activityTracker  *activity.Tracker
	netBlock         *Blocker
	weaponMap        fp.MutexMap[logparse.Weapon, int]
	chatLogger       *chatLogger
	matchSummarizer  *Summarizer
}

type CommandHandler func(ctx context.Context, s *discordgo.Session, m *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error)

func New(conf config.Config, database store.Stores, bot *Bot, logger *zap.Logger, assetStore s3.AssetStore) *App {
	eventBroadcaster := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()
	filters := NewWordFilters()
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
		steamGroups:      NewSteamGroupMemberships(logger, database),
		patreon:          NewPatreonManager(logger, conf),
		wordFilters:      filters,
		mc:               newMetricCollector(),
		state:            state.NewCollector(logger),
		activityTracker:  activity.NewTracker(logger),
		netBlock:         NewBlocker(),
		weaponMap:        fp.NewMutexMap[logparse.Weapon, int](),
	}

	application.setConfig(conf)
	application.warningTracker = NewTracker(logger, database, conf.Filter, onWarningHandler(application), onWarningExceeded(application))
	application.chatLogger = newChatLogger(logger, database, eventBroadcaster, filters, application.warningTracker, matchUUIDMap)
	application.matchSummarizer = NewSummarizer(logger, eventBroadcaster, matchUUIDMap, onMatchComplete(application))

	if conf.Discord.Enabled {
		if errReg := RegisterDiscordHandlers(application); errReg != nil {
			panic(errReg)
		}
	}

	return application
}

func (app *App) Activity() *activity.Tracker {
	return app.activityTracker
}

func (app *App) SendPayload(channelID string, message *discordgo.MessageEmbed) {
	app.discord.SendPayload(channelID, message)
}

func (app *App) Version() model.BuildInfo {
	return model.BuildInfo{
		BuildVersion: BuildVersion,
		Commit:       BuildCommit,
		Date:         BuildDate,
	}
}

func (app *App) EventBroadcaster() *fp.Broadcaster[logparse.EventType, logparse.ServerEvent] {
	return app.eb
}

func (app *App) Config() config.Config {
	app.confMu.RLock()
	defer app.confMu.RUnlock()

	return app.conf
}

func (app *App) Warnings() model.Warnings {
	return app.warningTracker
}

func (app *App) State() *state.Collector {
	return app.state
}

func (app *App) Patreon() model.Patreon {
	return app.patreon
}

func (app *App) NetBlocks() model.NetBLocker {
	return app.netBlock
}

func (app *App) Store() store.Stores {
	return app.db
}

func (app *App) Groups() model.Groups {
	return app.steamGroups
}

func (app *App) WordFilters() *WordFilters {
	return app.wordFilters
}

func (app *App) Log() *zap.Logger {
	return app.log
}

func (app *App) Assets() s3.AssetStore {
	return app.assetStore
}

func (app *App) WeaponMap() fp.MutexMap[logparse.Weapon, int] {
	return app.weaponMap
}

func (app *App) startWorkers(ctx context.Context) {
	go app.patreon.Start(ctx)
	go app.banSweeper(ctx)
	go app.profileUpdater(ctx)
	go app.warningTracker.Start(ctx)
	go app.logReader(ctx, app.Config().Debug.WriteUnhandledLogEvents)
	go app.initLogSrc(ctx)
	go logMetricsConsumer(ctx, app.mc, app.eb, app.log)
	go app.matchSummarizer.Start(ctx)
	go app.chatLogger.start(ctx)
	go app.playerConnectionWriter(ctx)
	go app.steamGroups.Start(ctx)
	go cleanupTasks(ctx, app.db, app.log)
	go app.showReportMeta(ctx)
	go app.notificationSender(ctx)
	go app.demoCleaner(ctx)
	go app.state.Start(ctx, func() config.Config {
		return app.Config()
	}, func() state.ServerStore {
		return app.Store()
	})
	go app.activityTracker.Start(ctx)
}

func (app *App) setConfig(conf config.Config) {
	app.confMu.Lock()
	defer app.confMu.Unlock()

	app.conf = conf
}

func firstTimeSetup(ctx context.Context, conf config.Config, database store.Stores, weaponMap fp.MutexMap[logparse.Weapon, int]) error {
	if !conf.General.Owner.Valid() {
		return errors.New("Configured owner is not a valid steam64")
	}

	localCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var owner model.Person

	if errRootUser := database.GetPersonBySteamID(localCtx, conf.General.Owner, &owner); errRootUser != nil {
		if !errors.Is(errRootUser, errs.ErrNoResult) {
			return errors.Join(errRootUser, errors.New("Failed first time setup"))
		}

		newOwner := model.NewPerson(conf.General.Owner)
		newOwner.PermissionLevel = model.PAdmin

		if errSave := database.SavePerson(localCtx, &newOwner); errSave != nil {
			return errors.Join(errSave, errors.New("Failed to create admin user"))
		}

		newsEntry := model.NewsEntry{
			Title:       "Welcome to gbans",
			BodyMD:      "This is an *example* **news** entry.",
			IsPublished: true,
			CreatedOn:   time.Now(),
			UpdatedOn:   time.Now(),
		}

		if errSave := database.SaveNewsArticle(localCtx, &newsEntry); errSave != nil {
			return errors.Join(errSave, errors.New("Failed to create sample news entry"))
		}

		server := model.NewServer("server-1", "127.0.0.1", 27015)
		server.CC = "jp"
		server.RCON = "example_rcon"
		server.Latitude = 35.652832
		server.Longitude = 139.839478
		server.Name = "Example ServerStore"
		server.LogSecret = 12345678
		server.Region = "asia"

		if errSave := database.SaveServer(localCtx, &server); errSave != nil {
			return errors.Join(errSave, errors.New("Failed to create sample server entry"))
		}

		page := wiki.Page{
			Slug:      wiki.RootSlug,
			BodyMD:    "# Welcome to the wiki",
			Revision:  1,
			CreatedOn: time.Now(),
			UpdatedOn: time.Now(),
		}

		if errSave := database.SaveWikiPage(localCtx, &page); errSave != nil {
			return errors.Join(errSave, errors.New("Failed to create sample wiki entry"))
		}
	}

	if errWeapons := database.LoadWeapons(ctx, weaponMap); errWeapons != nil {
		return errors.Join(errWeapons, errors.New("Failed to load weapons"))
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
	if app.Config().Filter.Enabled {
		count, errFilter := app.LoadFilters(ctx)
		if errFilter != nil {
			return errors.Join(errFilter, errors.New("Failed to load filters"))
		}

		app.log.Info("Loaded filter list", zap.Int64("count", count))
	}

	if errBlocklist := app.loadNetBlocks(ctx); errBlocklist != nil {
		app.log.Error("Could not load CIDR block list", zap.Error(errBlocklist))
	}

	return nil
}

func (app *App) loadNetBlocks(ctx context.Context) error {
	sources, errSource := app.db.GetCIDRBlockSources(ctx)
	if errSource != nil {
		return errors.Join(errSource, errors.New("Failed to load block sources"))
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

	whitelists, errWhitelists := app.db.GetCIDRBlockWhitelists(ctx)
	if errWhitelists != nil {
		if !errors.Is(errWhitelists, errs.ErrNoResult) {
			return errors.Join(errWhitelists, errors.New("Failed to load cidr block whitelists"))
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
	app.log.Info("Service status changed", zap.String("State", "ready"))
	defer app.log.Info("Service status changed", zap.String("State", "stopped"))

	if app.Config().General.Mode == config.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	httpServer := api.New(ctx, app)

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
		return errors.Join(errServe, errors.New("HTTP listener returned error"))
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
				log.Warn("Empty Person message body, skipping")

				continue
			}

			parsedAddr := net.ParseIP(newServerEvent.Address)
			if parsedAddr == nil {
				log.Warn("Received invalid address", zap.String("addr", newServerEvent.Address))

				continue
			}

			// Maybe ignore these and wait for connect call to create?
			var person model.Person
			if errPerson := app.Store().GetOrCreatePersonBySteamID(ctx, newServerEvent.SID, &person); errPerson != nil {
				log.Error("Failed to load Person", zap.Error(errPerson))

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

func (app *App) LoadFilters(ctx context.Context) (int64, error) {
	// TODO load external lists via http
	localCtx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	words, count, errGetFilters := app.db.GetFilters(localCtx, model.FiltersQueryFilter{})
	if errGetFilters != nil {
		if errors.Is(errGetFilters, errs.ErrNoResult) {
			return 0, nil
		}

		return 0, errors.Join(errGetFilters, errors.New("Failed to fetch filters"))
	}

	app.wordFilters.Import(words)

	app.log.Debug("Loaded word filters", zap.Int64("count", count))

	return count, nil
}

// UDP log sink.
func (app *App) initLogSrc(ctx context.Context) {
	logSrc, errLogSrc := logparse.NewUDPLogListener(app.log, app.Config().Log.SrcdsLogAddr,
		func(eventType logparse.EventType, event logparse.ServerEvent) {
			app.eb.Emit(event.EventType, event)
		})

	if errLogSrc != nil {
		app.log.Fatal("Failed to setup udp log src", zap.Error(errLogSrc))
	}

	app.logListener = logSrc

	// TODO run on server Config changes
	go app.updateSrcdsLogSecrets(ctx)

	app.logListener.Start(ctx)
}

func (app *App) updateSrcdsLogSecrets(ctx context.Context) {
	newSecrets := map[int]logparse.ServerIDMap{}
	serversCtx, cancelServers := context.WithTimeout(ctx, time.Second*5)

	defer cancelServers()

	servers, _, errServers := app.db.GetServers(serversCtx, model.ServerQueryFilter{
		IncludeDisabled: false,
		QueryFilter:     model.QueryFilter{Deleted: false},
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

type NotificationHandler struct{}

type NotificationPayload struct {
	MinPerms model.Privilege
	Sids     steamid.Collection
	Severity model.NotificationSeverity
	Message  string
	Link     string
}

func (app *App) SendNotification(ctx context.Context, notification NotificationPayload) error {
	// Collect all required ids
	if notification.MinPerms >= model.PUser {
		sids, errIds := app.db.GetSteamIdsAbove(ctx, notification.MinPerms)
		if errIds != nil {
			return errors.Join(errIds, errors.New("Failed to fetch steamids for notification"))
		}

		notification.Sids = append(notification.Sids, sids...)
	}

	uniqueIds := fp.Uniq(notification.Sids)

	people, errPeople := app.db.GetPeopleBySteamID(ctx, uniqueIds)
	if errPeople != nil && !errors.Is(errPeople, errs.ErrNoResult) {
		return errors.Join(errPeople, errors.New("Failed to fetch people for notification"))
	}

	var discordIds []string

	for _, p := range people {
		if p.DiscordID != "" {
			discordIds = append(discordIds, p.DiscordID)
		}
	}

	//go func(ids []string, payload NotificationPayload) {
	//	for _, discordID := range ids {
	//		msgEmbed := discord.NewEmbed(app.Config(), "Notification", payload.Message)
	//		if payload.Link != "" {
	//			msgEmbed.Embed().SetURL(payload.Link)
	//		}
	//
	//		app.SendPayload(discordID, msgEmbed.Embed().Truncate().MessageEmbed)
	//	}
	//}(discordIds, notification)

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

// validateLink is used in the case of discord origin actions that require mapping the
// discord member ID to a SteamID so that we can track its use and apply permissions, etc.
//
// This function will replace the discord member id value in the target field with
// the found SteamID, if any.
// func validateLink(ctx context.Context, database db.postgreStore, sourceID action.Author, target *action.Author) error {
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
