// Package app is the main application and entry point. It implements the action.Executor and io.Closer interfaces.
package app

import (
	"context"
	"fmt"
	"net"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/steamid/v3/extra"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"github.com/rumblefrog/go-a2s"
	"go.uber.org/zap"
)

// BuildVersion holds the current git revision, as of build time.
var BuildVersion = "master" //nolint:gochecknoglobals

type App struct {
	conf                 *config.Config
	bot                  *discord.Bot
	db                   *store.Store
	log                  *zap.Logger
	logFileChan          chan *model.LogFilePayload
	warningChan          chan newUserWarning
	notificationChan     chan NotificationPayload
	stateCollector       *state.ServerStateCollector
	bannedGroupMembers   map[steamid.GID]steamid.Collection
	bannedGroupMembersMu *sync.RWMutex
	patreon              *PatreonManager
	eb                   *eventBroadcaster
	wordFilters          *wordFilters
	mc                   *metricCollector
	serverState          map[int]ServerDetails
	stateMu              *sync.RWMutex
	msl                  []state.ServerLocation
}

func New(conf *config.Config, database *store.Store, bot *discord.Bot, logger *zap.Logger) App {
	eb := newEventBroadcaster()

	application := App{
		bot:                  bot,
		eb:                   eb,
		db:                   database,
		conf:                 conf,
		log:                  logger,
		logFileChan:          make(chan *model.LogFilePayload, 10),
		warningChan:          make(chan newUserWarning),
		notificationChan:     make(chan NotificationPayload, 5),
		bannedGroupMembers:   map[steamid.GID]steamid.Collection{},
		bannedGroupMembersMu: &sync.RWMutex{},
		patreon:              NewPatreonManager(logger, conf, database),
		wordFilters:          newWordFilters(),
		mc:                   newMetricCollector(),
		serverState:          map[int]ServerDetails{},
		stateMu:              &sync.RWMutex{},
	}

	if errReg := application.registerDiscordHandlers(); errReg != nil {
		panic(errReg)
	}

	application.stateCollector = state.NewServerStateCollector(logger,
		application.onA2SUpdate,
		application.onPlayerUpdate,
		application.onMSLUpdate)

	// bot.SetOnConnect(func() {
	//	_ = SendNotification(ctx, &conf, app.NotificationPayload{
	//		MinPerms: consts.PAdmin, Severity: consts.SeverityInfo, Message: "Discord connected",
	//	})
	// })
	// bot.SetOnDisconnect(func() {
	//	_ = SendNotification(ctx, &conf, app.NotificationPayload{
	//		MinPerms: consts.PAdmin, Severity: consts.SeverityInfo, Message: "Discord disconnected",
	//	})
	// })

	return application
}

func (app *App) onPlayerUpdate(serverID int, newState extra.Status) {
	app.stateMu.Lock()
	defer app.stateMu.Unlock()
	server := app.serverState[serverID]
	server.PlayerCount = newState.PlayersCount
	server.MaxPlayers = newState.PlayersMax
	if newState.ServerName != "" && newState.ServerName != server.Name {
		server.Name = newState.ServerName
	}
	server.Version = newState.Version
	server.Edicts = newState.Edicts
	server.Tags = newState.Tags
	if newState.Map != "" && newState.Map != server.Map {
		server.Map = newState.Map
	}
	server.Players = newState.Players
	app.serverState[serverID] = server
}

func (app *App) onA2SUpdate(serverID int, newState *a2s.ServerInfo) {
	app.stateMu.Lock()
	defer app.stateMu.Unlock()
	server := app.serverState[serverID]
	if newState.Map != "" && newState.Map != server.Map {
		server.Map = newState.Map
	}
	if newState.Name != "" && newState.Name != server.Name {
		server.Name = newState.Name
	}
	server.Protocol = newState.Protocol
	server.Folder = newState.Folder
	server.Game = newState.Game
	server.AppID = newState.ID
	server.PlayerCount = int(newState.MaxPlayers)
	server.MaxPlayers = int(newState.MaxPlayers)
	server.Bots = int(newState.Bots)
	server.ServerType = newState.ServerType.String()
	server.ServerOS = newState.ServerOS.String()
	server.VAC = newState.VAC
	server.Version = newState.Version
	if newState.SourceTV != nil {
		server.STVPort = newState.SourceTV.Port
		server.STVName = newState.SourceTV.Name
	}
	app.serverState[serverID] = server
}

func (app *App) onMSLUpdate(newState []state.ServerLocation) {
	app.stateMu.Lock()
	defer app.stateMu.Unlock()
	app.msl = newState
}

func (app *App) state() ServerDetailsCollection {
	app.stateMu.RLock()
	defer app.stateMu.RUnlock()

	var curState []ServerDetails //nolint:prealloc
	for _, s := range app.serverState {
		curState = append(curState, s)
	}

	sort.SliceStable(curState, func(i, j int) bool {
		return curState[i].Name < curState[j].Name
	})

	return curState
}

type userWarning struct {
	WarnReason    store.Reason
	Message       string
	Matched       string
	MatchedFilter *store.Filter
	CreatedOn     time.Time
}

func firstTimeSetup(ctx context.Context, conf *config.Config, db *store.Store) error {
	if !conf.General.Owner.Valid() {
		return errors.New("Configured owner is not a valid steam64")
	}
	localCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var owner store.Person
	if errRootUser := db.GetPersonBySteamID(localCtx, conf.General.Owner, &owner); errRootUser != nil {
		if !errors.Is(errRootUser, store.ErrNoResult) {
			return errors.Wrapf(errRootUser, "Failed first time setup")
		}
		newOwner := store.NewPerson(conf.General.Owner)
		newOwner.PermissionLevel = consts.PAdmin
		if errSave := db.SavePerson(localCtx, &newOwner); errSave != nil {
			return errors.Wrap(errSave, "Failed to create admin user")
		}
		newsEntry := store.NewsEntry{
			Title:       "Welcome to gbans",
			BodyMD:      "This is an *example* **news** entry.",
			IsPublished: true,
			CreatedOn:   time.Now(),
			UpdatedOn:   time.Now(),
		}
		if errSave := db.SaveNewsArticle(localCtx, &newsEntry); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample news entry")
		}
		server := store.NewServer("server-1", "127.0.0.1", 27015)
		server.CC = "jp"
		server.RCON = "example_rcon"
		server.Latitude = 35.652832
		server.Longitude = 139.839478
		server.ServerNameLong = "Example Server"
		server.LogSecret = 12345678
		server.Region = "asia"
		if errSave := db.SaveServer(localCtx, &server); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample server entry")
		}
		var page wiki.Page
		page.BodyMD = "# Welcome to the wiki"
		page.UpdatedOn = time.Now()
		page.CreatedOn = time.Now()
		page.Revision = 1
		page.Slug = wiki.RootSlug
		if errSave := db.SaveWikiPage(localCtx, &page); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample wiki entry")
		}
	}

	return nil
}

func (app *App) Init(ctx context.Context) error {
	if setupErr := firstTimeSetup(ctx, app.conf, app.db); setupErr != nil {
		app.log.Fatal("Failed to do first time setup", zap.Error(setupErr))
	}

	// Load in the external network block / ip ban lists to memory if enabled
	if app.conf.NetBans.Enabled {
		if errNetBans := initNetBans(ctx, app.conf); errNetBans != nil {
			return errors.Wrap(errNetBans, "Failed to load net bans")
		}
	} else {
		app.log.Warn("External Network ban lists not enabled")
	}

	// Start the background goroutine workers
	app.startWorkers(ctx)

	// Load the filtered word set into memory
	if app.conf.Filter.Enabled {
		if errFilter := app.initFilters(ctx); errFilter != nil {
			return errors.Wrap(errFilter, "Failed to load filters")
		}
		app.log.Info("Loaded filter list", zap.Int("count", len(app.wordFilters.wordFilters)))
	}

	return nil
}

type newUserWarning struct {
	ServerEvent model.ServerEvent
	Message     string
	userWarning
}

// warnWorker handles tracking and applying warnings based on incoming events.
func (app *App) warnWorker(ctx context.Context, conf *config.Config) {
	warnings := map[steamid.SID64][]userWarning{}
	eventChan := make(chan model.ServerEvent)
	if errRegister := app.eb.Consume(eventChan, []logparse.EventType{logparse.Say, logparse.SayTeam}); errRegister != nil {
		app.log.Fatal("Failed to register event reader", zap.Error(errRegister))
	}
	log := app.log.Named("warnWorker")
	ticker := time.NewTicker(1 * time.Second)
	warningHandler := func() {
		for {
			select {
			case now := <-ticker.C:
				for steamID := range warnings {
					for warnIdx, warning := range warnings[steamID] {
						if now.Sub(warning.CreatedOn) > conf.General.WarningTimeout {
							if len(warnings[steamID]) > 1 {
								warnings[steamID] = append(warnings[steamID][:warnIdx], warnings[steamID][warnIdx+1])
							} else {
								delete(warnings, steamID)
							}
						}
					}
				}
			case newWarn := <-app.warningChan:
				evt, ok := newWarn.ServerEvent.Event.(logparse.SayEvt)
				if !ok {
					continue
				}
				if !evt.SID.Valid() {
					continue
				}
				newWarn.MatchedFilter.TriggerCount++
				if errSave := app.db.SaveFilter(ctx, newWarn.MatchedFilter); errSave != nil {
					log.Error("Failed to update filter trigger count", zap.Error(errSave))
				}
				log.Info("User triggered word filter",
					zap.String("matched", newWarn.Matched),
					zap.String("message", newWarn.Message),
					zap.Int64("filter_id", newWarn.MatchedFilter.FilterID))
				var person store.Person
				if personErr := app.PersonBySID(ctx, evt.SID, &person); personErr != nil {
					log.Error("Failed to get person for warning", zap.Error(personErr))

					continue
				}
				if newWarn.MatchedFilter.IsEnabled {
					_, found := warnings[evt.SID]
					if !found {
						warnings[evt.SID] = []userWarning{}
					}
					warnings[evt.SID] = append(warnings[evt.SID], newWarn.userWarning)
				}

				title := fmt.Sprintf("Language Warning (#%d/%d)", len(warnings[evt.SID]), conf.General.WarningLimit)
				if !newWarn.MatchedFilter.IsEnabled {
					title = "[DISABLED] Language Warning"
				}
				warnNotice := &discordgo.MessageEmbed{
					URL:   conf.ExtURL("/profiles/%d", evt.SID),
					Type:  discordgo.EmbedTypeRich,
					Title: title,
					Color: int(discord.Green),
					Image: &discordgo.MessageEmbedImage{URL: person.AvatarFull},
				}
				discord.AddField(warnNotice, "Matched", newWarn.Matched)
				discord.AddField(warnNotice, "Message", newWarn.userWarning.Message)
				if newWarn.MatchedFilter.IsEnabled {
					if len(warnings[evt.SID]) > conf.General.WarningLimit {
						log.Info("Warn limit exceeded",
							zap.Int64("sid64", evt.SID.Int64()),
							zap.Int("count", len(warnings[evt.SID])))
						var errBan error
						var banSteam store.BanSteam
						if errNewBan := store.NewBanSteam(ctx, store.StringSID(conf.General.Owner.String()),
							store.StringSID(evt.SID.String()),
							store.Duration(conf.General.WarningExceededDurationValue),
							newWarn.WarnReason,
							"",
							"Automatic warning ban",
							store.System,
							0,
							store.NoComm,
							&banSteam); errNewBan != nil {
							log.Error("Failed to create warning ban", zap.Error(errNewBan))

							continue
						}
						switch conf.General.WarningExceededAction {
						case config.Gag:
							banSteam.BanType = store.NoComm
							errBan = app.BanSteam(ctx, &banSteam)
						case config.Ban:
							banSteam.BanType = store.Banned
							errBan = app.BanSteam(ctx, &banSteam)
						case config.Kick:
							errBan = app.Kick(ctx, store.System, evt.SID, conf.General.Owner, newWarn.WarnReason)
						}
						if errBan != nil {
							log.Error("Failed to apply warning action",
								zap.Error(errBan),
								zap.String("action", string(conf.General.WarningExceededAction)))
						}
						discord.AddField(warnNotice, "Name", person.PersonaName)
						expIn := "Permanent"
						expAt := "Permanent"
						if banSteam.ValidUntil.Year()-config.Now().Year() < 5 {
							expIn = config.FmtDuration(banSteam.ValidUntil)
							expAt = config.FmtTimeShort(banSteam.ValidUntil)
						}
						discord.AddField(warnNotice, "Expires In", expIn)
						discord.AddField(warnNotice, "Expires At", expAt)
					} else {
						msg := fmt.Sprintf("[WARN #%d] Please refrain from using slurs/toxicity (see: rules & MOTD). "+
							"Further offenses will result in mutes/bans", len(warnings[evt.SID]))
						if errPSay := app.PSay(ctx, "", evt.SID, msg); errPSay != nil {
							log.Error("Failed to send user warning psay message", zap.Error(errPSay))
						}
					}
				}
				discord.AddField(warnNotice, "Pattern", newWarn.MatchedFilter.Pattern)
				discord.AddFieldsSteamID(warnNotice, evt.SID)
				discord.AddFieldInt64Inline(warnNotice, "Filter ID", newWarn.MatchedFilter.FilterID)
				discord.AddFieldInline(warnNotice, "Server", newWarn.ServerEvent.Server.ServerName)
				app.bot.SendPayload(discord.Payload{
					ChannelID: conf.Discord.ModLogChannelID,
					Embed:     warnNotice,
				})

			case <-ctx.Done():
				return
			}
		}
	}

	go warningHandler()

	for {
		select {
		case serverEvent := <-eventChan:
			evt, ok := serverEvent.Event.(logparse.SayEvt)
			if !ok {
				log.Error("Got invalid type?")

				continue
			}
			if evt.Msg == "" {
				continue
			}
			matchedWord, matchedFilter := app.wordFilters.findFilteredWordMatch(evt.Msg)
			if matchedFilter != nil {
				app.warningChan <- newUserWarning{
					ServerEvent: serverEvent,
					userWarning: userWarning{
						WarnReason:    store.Language,
						Message:       evt.Msg,
						Matched:       matchedWord,
						MatchedFilter: matchedFilter,
						CreatedOn:     config.Now(),
					},
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (app *App) matchSummarizer(ctx context.Context) {
	log := app.log.Named("matchSum")
	eventChan := make(chan model.ServerEvent)
	if errReg := app.eb.Consume(eventChan, []logparse.EventType{logparse.Any}); errReg != nil {
		log.Error("logWriter Tried to register duplicate reader channel", zap.Error(errReg))
	}
	matches := map[int]logparse.Match{}

	var curServer store.Server
	for {
		select {
		case evt := <-eventChan:
			match, found := matches[evt.Server.ServerID]
			if !found && evt.EventType != logparse.MapLoad {
				// Wait for new map
				continue
			}
			if evt.EventType == logparse.LogStart {
				log.Info("New match created (new game)", zap.String("server", evt.Server.ServerName))
				matches[evt.Server.ServerID] = logparse.NewMatch(log, evt.Server.ServerID, evt.Server.ServerNameLong)
			}
			// Apply the update before any secondary side effects trigger
			if errApply := match.Apply(evt.Results); errApply != nil {
				log.Error("Error applying event",
					zap.String("server", evt.Server.ServerName),
					zap.Error(errApply))
			}
			switch evt.EventType {
			case logparse.LogStop:
				fallthrough
			case logparse.WGameOver:
				go func(completeMatch logparse.Match) {
					if errSave := app.db.MatchSave(ctx, &completeMatch); errSave != nil {
						log.Error("Failed to save match",
							zap.String("server", evt.Server.ServerName), zap.Error(errSave))
					} else {
						sendDiscordMatchResults(curServer, completeMatch, app.conf, app.bot)
					}
				}(match)
				delete(matches, evt.Server.ServerID)
			}
		case <-ctx.Done():
			return
		}
	}
}

func sendDiscordMatchResults(server store.Server, match logparse.Match, conf *config.Config, bot *discord.Bot) {
	embed := &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeRich,
		Title:       fmt.Sprintf("Match #%d - %s - %s", match.MatchID, server.ServerName, match.MapName),
		Description: "Match results",
		Color:       int(discord.Green),
		URL:         conf.ExtURL("/log/%d", match.MatchID),
	}
	redScore := 0
	bluScore := 0
	for _, round := range match.Rounds {
		redScore += round.Score.Red
		bluScore += round.Score.Blu
	}

	found := 0
	for _, teamStats := range match.TeamSums {
		discord.AddFieldInline(embed, fmt.Sprintf("%s Kills", teamStats.Team.String()), fmt.Sprintf("%d", teamStats.Kills))
		discord.AddFieldInline(embed, fmt.Sprintf("%s Damage", teamStats.Team.String()), fmt.Sprintf("%d", teamStats.Damage))
		discord.AddFieldInline(embed, fmt.Sprintf("%s Ubers/Drops", teamStats.Team.String()), fmt.Sprintf("%d/%d", teamStats.Charges, teamStats.Drops))
		found++
	}
	discord.AddFieldInline(embed, "Red Score", fmt.Sprintf("%d", redScore))
	discord.AddFieldInline(embed, "Blu Score", fmt.Sprintf("%d", bluScore))
	discord.AddFieldInline(embed, "Duration", fmt.Sprintf("%.2f Minutes", time.Since(match.CreatedOn).Minutes()))
	bot.SendPayload(discord.Payload{ChannelID: conf.Discord.LogChannelID, Embed: embed})
}

func playerMessageWriter(ctx context.Context, eb *eventBroadcaster, logger *zap.Logger, db *store.Store) {
	log := logger.Named("playerMessageWriter")
	serverEventChan := make(chan model.ServerEvent)
	if errRegister := eb.Consume(serverEventChan, []logparse.EventType{
		logparse.Say,
		logparse.SayTeam,
	}); errRegister != nil {
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
				e, ok := evt.Event.(logparse.SayEvt)
				if !ok {
					continue
				}
				if e.Msg == "" {
					log.Warn("Empty person message body, skipping")

					continue
				}
				msg := store.PersonMessage{
					SteamID:     e.SID,
					PersonaName: e.Name,
					ServerName:  evt.Server.ServerNameLong,
					ServerID:    evt.Server.ServerID,
					Body:        e.Msg,
					Team:        evt.EventType == logparse.SayTeam,
					CreatedOn:   e.CreatedOn,
				}
				lCtx, cancel := context.WithTimeout(ctx, time.Second*5)
				if errChat := db.AddChatHistory(lCtx, &msg); errChat != nil {
					log.Error("Failed to add chat history", zap.Error(errChat))
				}
				cancel()
				log.Debug("Saved user chat message", zap.String("message", msg.Body))
			}
		}
	}
}

func playerConnectionWriter(ctx context.Context, eb *eventBroadcaster, db *store.Store, logger *zap.Logger) {
	log := logger.Named("playerConnectionWriter")
	serverEventChan := make(chan model.ServerEvent)
	if errRegister := eb.Consume(serverEventChan, []logparse.EventType{logparse.Connected}); errRegister != nil {
		log.Warn("logWriter Tried to register duplicate reader channel", zap.Error(errRegister))

		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-serverEventChan:
			e, ok := evt.Event.(logparse.ConnectedEvt)
			if !ok {
				continue
			}
			if e.Address == "" {
				log.Warn("Empty person message body, skipping")

				continue
			}
			parsedAddr := net.ParseIP(e.Address)
			if parsedAddr == nil {
				log.Warn("Received invalid address", zap.String("addr", e.Address))

				continue
			}
			conn := store.PersonConnection{
				IPAddr:      parsedAddr,
				SteamID:     e.SID,
				PersonaName: e.Name,
				CreatedOn:   e.CreatedOn,
			}
			lCtx, cancel := context.WithTimeout(ctx, time.Second*5)
			if errChat := db.AddConnectionHistory(lCtx, &conn); errChat != nil {
				log.Error("Failed to add connection history", zap.Error(errChat))
			}
			cancel()
		}
	}
}

// logReader is the fan-out orchestrator for game log events
// Registering receivers can be accomplished with RegisterLogEventReader.
func (app *App) logReader(ctx context.Context, writeUnhandled bool) {
	log := app.log.Named("logReader")
	var file *os.File
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
	parser := logparse.New()
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
				serverEvent := model.ServerEvent{
					Server:  logFile.Server,
					Results: parseResult,
				}
				if serverEvent.EventType == logparse.IgnoredMsg {
					ignored++

					continue
				} else if serverEvent.EventType == logparse.UnknownMsg {
					unknown++
					if writeUnhandled {
						if _, errWrite := file.WriteString(logLine + "\n"); errWrite != nil {
							log.Error("Failed to write debug log", zap.Error(errWrite))
						}
					}
				}
				app.eb.Emit(serverEvent)
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

func (app *App) initFilters(ctx context.Context) error {
	// TODO load external lists via http
	localCtx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()
	words, errGetFilters := app.db.GetFilters(localCtx)
	if errGetFilters != nil {
		if errors.Is(errGetFilters, store.ErrNoResult) {
			return nil
		}

		return errors.Wrap(errGetFilters, "Failed to fetch filters")
	}
	app.wordFilters.importFilteredWords(words)

	return nil
}

func (app *App) startWorkers(ctx context.Context) {
	go app.patreon.updater(ctx)
	go app.banSweeper(ctx)
	// go profileUpdater(ctx)
	go app.warnWorker(ctx, app.conf)
	go app.logReader(ctx, app.conf.Debug.WriteUnhandledLogEvents)
	go app.initLogSrc(ctx)
	go logMetricsConsumer(ctx, app.mc, app.eb, app.log)
	go app.matchSummarizer(ctx)
	go playerMessageWriter(ctx, app.eb, app.log, app.db)
	go playerConnectionWriter(ctx, app.eb, app.db, app.log)
	go app.steamGroupMembershipUpdater(ctx)
	go app.localStatUpdater(ctx)
	go cleanupTasks(ctx, app.db, app.log)
	go app.showReportMeta(ctx)
	go app.notificationSender(ctx)
	go demoCleaner(ctx, app.db, app.log)
	go app.stateUpdater(ctx)
}

// UDP log sink.
func (app *App) initLogSrc(ctx context.Context) {
	logSrc, errLogSrc := newRemoteSrcdsLogSource(app.log, app.db, app.conf.Log.SrcdsLogAddr, app.eb)
	if errLogSrc != nil {
		app.log.Fatal("Failed to setup udp log src", zap.Error(errLogSrc))
	}
	logSrc.start(ctx)
}

// func SendUserNotification(pl NotificationPayload) {
//	select {
//	case notificationChan <- pl:
//	default:
//		logger.Error("Failed to write user notification payload, channel full")
//	}
// }

func initNetBans(ctx context.Context, conf *config.Config) error {
	for _, banList := range conf.NetBans.Sources {
		if _, errImport := thirdparty.Import(ctx, banList, conf.NetBans.CachePath, conf.NetBans.MaxAge); errImport != nil {
			return errors.Wrap(errImport, "Failed to import net bans")
		}
	}

	return nil
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
