// Package app is the main application and entry point. It implements the action.Executor and io.Closer interfaces.
package app

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"gopkg.in/mxpv/patreon-go.v1"
	"net"
	"os"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/event"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"github.com/rumblefrog/go-a2s"
	log "github.com/sirupsen/logrus"
)

type App struct {
	logFileChan chan *LogFilePayload
	// Current known state of the servers rcon status command
	serverStateStatus   map[string]extra.Status
	serverStateStatusMu *sync.RWMutex
	// Current known state of the servers a2s server info query
	serverStateA2S     map[string]a2s.ServerInfo
	serverStateA2SMu   *sync.RWMutex
	masterServerList   []model.ServerLocation
	masterServerListMu *sync.RWMutex
	discordSendMsg     chan discordPayload
	warningChan        chan newUserWarning
	notificationChan   chan notificationPayload
	serverStateMu      *sync.RWMutex
	serverState        model.ServerStateCollection

	bannedGroupMembers   map[steamid.GID]steamid.Collection
	bannedGroupMembersMu *sync.RWMutex
	ctx                  context.Context
	store                store.Store
	patreon              *patreon.Client
	patreonMu            *sync.RWMutex
	patreonCampaigns     []patreon.Campaign
	patreonPledges       []patreon.Pledge
	patreonUsers         map[string]*patreon.User
}

var (
	// BuildVersion holds the current git revision, as of build time
	BuildVersion = "master"
)

type userWarning struct {
	WarnReason model.Reason
	CreatedOn  time.Time
}

type discordPayload struct {
	channelId string
	embed     *discordgo.MessageEmbed
}

func New(ctx context.Context) *App {
	app := App{
		logFileChan:          make(chan *LogFilePayload, 10),
		serverStateStatus:    map[string]extra.Status{},
		serverStateStatusMu:  &sync.RWMutex{},
		serverStateA2S:       map[string]a2s.ServerInfo{},
		serverStateA2SMu:     &sync.RWMutex{},
		masterServerList:     []model.ServerLocation{},
		masterServerListMu:   &sync.RWMutex{},
		discordSendMsg:       make(chan discordPayload, 5),
		warningChan:          make(chan newUserWarning),
		notificationChan:     make(chan notificationPayload, 5),
		serverStateMu:        &sync.RWMutex{},
		serverState:          model.ServerStateCollection{},
		bannedGroupMembers:   map[steamid.GID]steamid.Collection{},
		bannedGroupMembersMu: &sync.RWMutex{},
		patreonMu:            &sync.RWMutex{},
		ctx:                  ctx,
	}
	return &app
}

func firstTimeSetup(ctx context.Context, db store.Store) error {
	if !config.General.Owner.Valid() {
		return errors.New("Configured owner is not a valid steam64")
	}
	localCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var owner model.Person
	if errRootUser := db.GetPersonBySteamID(localCtx, config.General.Owner, &owner); errRootUser != nil {
		if !errors.Is(errRootUser, store.ErrNoResult) {
			return errors.Wrapf(errRootUser, "Failed first time setup")
		}
		newOwner := model.NewPerson(config.General.Owner)
		newOwner.PermissionLevel = model.PAdmin
		if errSave := db.SavePerson(localCtx, &newOwner); errSave != nil {
			return errors.Wrap(errSave, "Failed to create admin user")
		}
		newsEntry := model.NewsEntry{
			Title:       "Welcome to gbans",
			BodyMD:      "This is an *example* **news** entry.",
			IsPublished: true,
			CreatedOn:   time.Now(),
			UpdatedOn:   time.Now(),
		}
		if errSave := db.SaveNewsArticle(localCtx, &newsEntry); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample news entry")
		}
		server := model.NewServer("server-1", "127.0.0.1", 27015)
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
	}
	return nil
}

// Start is the main application entry point
func (app *App) Start() error {
	dbStore, dbErr := store.New(app.ctx, config.DB.DSN)
	if dbErr != nil {
		return errors.Wrapf(dbErr, "Failed to setup store")
	}
	defer func() {
		if errClose := dbStore.Close(); errClose != nil {
			log.Errorf("Error cleanly closing app: %v", errClose)
		}
	}()
	app.store = dbStore

	if setupErr := firstTimeSetup(app.ctx, app.store); setupErr != nil {
		log.WithError(setupErr).Fatalf("Failed to do first time setup")
	}

	patreonClient, errPatreon := thirdparty.NewPatreonClient(app.ctx, dbStore)
	if errPatreon == nil {
		app.patreon = patreonClient
	}

	webService, errWeb := NewWeb(app)
	if errWeb != nil {
		return errors.Wrapf(errWeb, "Failed to setup web")
	}

	// Load in the external network block / ip ban lists to memory if enabled
	if config.Net.Enabled {
		initNetBans()
	} else {
		log.Warnf("External Network ban lists not enabled")
	}

	// Start the discord service
	if config.Discord.Enabled {
		go app.initDiscord(app.ctx, dbStore, app.discordSendMsg)
	} else {
		log.Warnf("discord bot not enabled")
	}

	// Start the background goroutine workers
	app.initWorkers()

	// Load the filtered word set into memory
	if config.Filter.Enabled {
		initFilters(app.ctx, dbStore)
	}

	if errSend := app.sendNotification(notificationPayload{
		minPerms: model.PAdmin,
		sids:     nil,
		severity: model.SeverityInfo,
		message:  "App start",
		link:     "",
	}); errSend != nil {
		log.Error(errSend)
	}
	// Start & block, listening on the HTTP server
	if errHttpListen := webService.ListenAndServe(app.ctx); errHttpListen != nil {
		return errors.Wrapf(errHttpListen, "Error shutting down service")
	}
	return nil
}

type newUserWarning struct {
	ServerEvent model.ServerEvent
	Message     string
	userWarning
}

// warnWorker handles tracking and applying warnings based on incoming events
func (app *App) warnWorker() {
	warnings := map[steamid.SID64][]userWarning{}
	eventChan := make(chan model.ServerEvent)
	if errRegister := event.Consume(eventChan, []logparse.EventType{logparse.Say, logparse.SayTeam}); errRegister != nil {
		log.Fatalf("Failed to register event reader: %v", errRegister)
	}
	ticker := time.NewTicker(1 * time.Second)

	warningHandler := func() {
		for {
			select {
			case now := <-ticker.C:
				for steamId := range warnings {
					for warnIdx, warning := range warnings[steamId] {
						if now.Sub(warning.CreatedOn) > config.General.WarningTimeout {
							if len(warnings[steamId]) > 1 {
								warnings[steamId] = append(warnings[steamId][:warnIdx], warnings[steamId][warnIdx+1])
							} else {
								warnings[steamId] = nil
							}
						}
						if len(warnings[steamId]) == 0 {
							delete(warnings, steamId)
						}
					}
				}
			case newWarn := <-app.warningChan:
				steamId := newWarn.ServerEvent.Source.SteamID
				if !steamId.Valid() {
					continue
				}
				_, found := warnings[steamId]
				if !found {
					warnings[steamId] = []userWarning{}
				}
				warnings[steamId] = append(warnings[steamId], newWarn.userWarning)

				warnNotice := &discordgo.MessageEmbed{
					URL:   config.ExtURL("/profiles/%d", steamId),
					Type:  discordgo.EmbedTypeRich,
					Title: fmt.Sprintf("Language Warning (#%d/%d)", len(warnings[steamId]), config.General.WarningLimit),
					Color: int(orange),
					Image: &discordgo.MessageEmbedImage{URL: newWarn.ServerEvent.Source.AvatarFull},
				}
				addField(warnNotice, "Server", newWarn.ServerEvent.Server.ServerNameShort)
				addField(warnNotice, "Message", newWarn.Message)
				if len(warnings[steamId]) > config.General.WarningLimit {
					log.Infof("Warn limit exceeded (%d): %d", steamId, len(warnings[steamId]))
					var errBan error
					var banSteam model.BanSteam
					if errOpts := NewBanSteam(model.StringSID(config.General.Owner.String()),
						model.StringSID(steamId.String()),
						model.Duration(config.General.WarningExceededDurationValue),
						newWarn.WarnReason,
						"",
						"Automatic warning ban",
						model.System,
						0,
						model.NoComm,
						&banSteam); errOpts != nil {
						log.Errorf("Failed to create warning ban: %v", errOpts)
						return
					}

					switch config.General.WarningExceededAction {
					case config.Gag:
						banSteam.BanType = model.NoComm
						errBan = app.BanSteam(app.ctx, app.store, &banSteam)
					case config.Ban:
						banSteam.BanType = model.Banned
						errBan = app.BanSteam(app.ctx, app.store, &banSteam)
					case config.Kick:
						var playerInfo model.PlayerInfo
						errBan = app.Kick(app.ctx, app.store, model.System, model.StringSID(steamId.String()),
							model.StringSID(config.General.Owner.String()), newWarn.WarnReason, &playerInfo)
					}
					if errBan != nil {
						log.WithFields(log.Fields{"action": config.General.WarningExceededAction}).
							Errorf("Failed to apply warning action: %v", errBan)
					}
					addField(warnNotice, "Name", newWarn.ServerEvent.Source.PersonaName)
					expIn := "Permanent"
					expAt := "Permanent"
					if banSteam.ValidUntil.Year()-config.Now().Year() < 5 {
						expIn = config.FmtDuration(banSteam.ValidUntil)
						expAt = config.FmtTimeShort(banSteam.ValidUntil)
					}
					addField(warnNotice, "Expires In", expIn)
					addField(warnNotice, "Expires At", expAt)
				} else {
					msg := fmt.Sprintf("[WARN #%d] Please refrain from using slurs/toxicity (see: rules & MOTD). "+
						"Further offenses will result in mutes/bans", len(warnings[steamId]))
					if errPSay := app.PSay(app.ctx, app.store, 0, model.StringSID(steamId.String()), msg, &newWarn.ServerEvent.Server); errPSay != nil {
						log.WithError(errPSay).Errorf("Failed to send user warning psay message")
					}
					addFieldsSteamID(warnNotice, steamId)
					addField(warnNotice, "message", newWarn.Message)
				}
				app.sendDiscordPayload(discordPayload{
					channelId: config.Discord.ModLogChannelId,
					embed:     warnNotice,
				})
			case <-app.ctx.Done():
				return
			}
		}
	}

	go warningHandler()

	for {
		select {
		case serverEvent := <-eventChan:
			msg, found := serverEvent.MetaData["msg"].(string)
			if !found {
				continue
			}
			matchedWord, matchedFilter := findFilteredWordMatch(msg)
			if matchedFilter != nil {
				app.warningChan <- newUserWarning{
					ServerEvent: serverEvent,
					userWarning: userWarning{
						WarnReason: model.Language,
						CreatedOn:  config.Now(),
					},
				}
				log.WithFields(log.Fields{
					"word":        fmt.Sprintf("||%s||", matchedWord),
					"msg":         msg,
					"filter_id":   matchedFilter.WordID,
					"filter_name": matchedFilter.FilterName,
				}).Infof("User triggered word filter")

			}
		case <-app.ctx.Done():
			return
		}
	}
}

//
//func matchSummarizer(ctx context.Context, db store.Store) {
//	eventChan := make(chan model.ServerEvent)
//	if errReg := event.Consume(eventChan, []logparse.EventType{logparse.Any}); errReg != nil {
//		log.Warnf("logWriter Tried to register duplicate reader channel")
//	}
//	match := model.NewMatch()
//
//	var reset = func() {
//		match = model.NewMatch()
//		log.Debugf("New match summary created")
//	}
//
//	var curServer model.Server
//	for {
//		// TODO reset on match start incase of stale data
//		select {
//		case evt := <-eventChan:
//			if match.ServerId == 0 && evt.Server.ServerID > 0 {
//				curServer = evt.Server
//				match.ServerId = curServer.ServerID
//			}
//			switch evt.EventType {
//			case logparse.MapLoad:
//				reset()
//			}
//			// Apply the update before any secondary side effects trigger
//			if errApply := match.Apply(evt); errApply != nil {
//				log.Tracef("Error applying event: %v", errApply)
//			}
//			switch evt.EventType {
//			case logparse.WGameOver:
//				if errSave := db.MatchSave(ctx, &match); errSave != nil {
//					log.Errorf("Failed to save match: %v", errSave)
//				} else {
//					sendDiscordNotif(curServer, &match)
//				}
//				reset()
//			}
//		case <-ctx.Done():
//			return
//		}
//	}
//}
//
//func sendDiscordNotif(server model.Server, match *model.Match) {
//	embed := &discordgo.MessageEmbed{
//		Type:        discordgo.EmbedTypeRich,
//		Title:       fmt.Sprintf("Match #%d - %s - %s", match.MatchID, server.ServerNameShort, match.MapName),
//		Description: "Match results",
//		Color:       int(green),
//		URL:         config.ExtURL("/log/%d", match.MatchID),
//	}
//	redScore := 0
//	bluScore := 0
//	for _, round := range match.Rounds {
//		redScore += round.Score.Red
//		bluScore += round.Score.Blu
//	}
//
//	found := 0
//	for _, teamStats := range match.TeamSums {
//		addFieldInline(embed, fmt.Sprintf("%s Kills", teamStats.Team.String()), fmt.Sprintf("%d", teamStats.Kills))
//		addFieldInline(embed, fmt.Sprintf("%s Damage", teamStats.Team.String()), fmt.Sprintf("%d", teamStats.Damage))
//		addFieldInline(embed, fmt.Sprintf("%s Ubers/Drops", teamStats.Team.String()), fmt.Sprintf("%d/%d", teamStats.Charges, teamStats.Drops))
//		found++
//	}
//	addFieldInline(embed, "Red Score", fmt.Sprintf("%d", redScore))
//	addFieldInline(embed, "Blu Score", fmt.Sprintf("%d", bluScore))
//	if found == 2 {
//		log.Debugf("Sending discord summary")
//		select {
//		case discordSendMsg <- discordPayload{channelId: config.Discord.LogChannelID, embed: embed}:
//		default:
//			log.Warnf("Cannot send discord payload, channel full")
//		}
//	}
//}

func (app *App) playerMessageWriter() {
	serverEventChan := make(chan model.ServerEvent)
	if errRegister := event.Consume(serverEventChan, []logparse.EventType{
		logparse.Say,
		logparse.SayTeam,
	}); errRegister != nil {
		log.Warnf("logWriter Tried to register duplicate reader channel")
		return
	}
	for {
		select {
		case <-app.ctx.Done():
			return
		case evt := <-serverEventChan:
			switch evt.EventType {
			case logparse.Say:
				fallthrough
			case logparse.SayTeam:
				body := evt.GetValueString("msg")
				if body == "" {
					log.Warnf("Empty person message body, skipping")
					continue
				}
				msg := model.PersonMessage{
					SteamId:     evt.Source.SteamID,
					PersonaName: evt.Source.PersonaName,
					ServerName:  evt.Server.ServerNameLong,
					ServerId:    evt.Server.ServerID,
					Body:        body,
					Team:        evt.EventType == logparse.SayTeam,
					CreatedOn:   evt.CreatedOn,
				}
				lCtx, cancel := context.WithTimeout(app.ctx, time.Second*5)
				if errChat := app.store.AddChatHistory(lCtx, &msg); errChat != nil {
					log.Errorf("Failed to add chat history: %v", errChat)
				}
				cancel()
				log.WithFields(log.Fields{"msg": msg}).Tracef("Saved message")
			}
		}
	}
}

func (app *App) playerConnectionWriter() {
	serverEventChan := make(chan model.ServerEvent)
	if errRegister := event.Consume(serverEventChan, []logparse.EventType{logparse.Connected}); errRegister != nil {
		log.Warnf("logWriter Tried to register duplicate reader channel")
		return
	}
	for {
		select {
		case <-app.ctx.Done():
			return
		case evt := <-serverEventChan:
			addr := evt.GetValueString("address")
			if addr == "" {
				log.Warnf("Empty person message body, skipping")
				continue
			}
			parsedAddr := net.ParseIP(addr)
			if parsedAddr == nil {
				log.Warnf("Received invalid address: %s", addr)
				continue
			}
			msg := model.PersonConnection{
				IPAddr:      parsedAddr,
				SteamId:     evt.Source.SteamID,
				PersonaName: evt.Source.PersonaName,
				CreatedOn:   evt.CreatedOn,
			}
			lCtx, cancel := context.WithTimeout(app.ctx, time.Second*5)
			if errChat := app.store.AddConnectionHistory(lCtx, &msg); errChat != nil {
				log.Errorf("Failed to add connection history: %v", errChat)
			}
			cancel()
		}
	}
}

type playerEventState struct {
	team      logparse.Team
	class     logparse.PlayerClass
	updatedAt time.Time
}

type playerCache struct {
	*sync.RWMutex
	state map[steamid.SID64]playerEventState
}

func newPlayerCache() *playerCache {
	pc := playerCache{
		RWMutex: &sync.RWMutex{},
		state:   map[steamid.SID64]playerEventState{},
	}
	go pc.cleanupWorker()
	return &pc
}

func (cache *playerCache) setTeam(sid steamid.SID64, team logparse.Team) {
	cache.Lock()
	defer cache.Unlock()
	state, found := cache.state[sid]
	if !found {
		state = playerEventState{}
	}
	state.team = team
	state.updatedAt = config.Now()
	cache.state[sid] = state
}

func (cache *playerCache) setClass(sid steamid.SID64, class logparse.PlayerClass) {
	cache.Lock()
	defer cache.Unlock()
	state, found := cache.state[sid]
	if !found {
		state = playerEventState{}
	}
	state.class = class
	state.updatedAt = config.Now()
	cache.state[sid] = state
}

func (cache *playerCache) getClass(sid steamid.SID64) logparse.PlayerClass {
	cache.RLock()
	defer cache.RUnlock()
	state, found := cache.state[sid]
	if !found {
		return logparse.Spectator
	}
	return state.class
}

func (cache *playerCache) getTeam(sid steamid.SID64) logparse.Team {
	cache.RLock()
	defer cache.RUnlock()
	state, found := cache.state[sid]
	if !found {
		return logparse.SPEC
	}
	return state.team
}

func (cache *playerCache) cleanupWorker() {
	ticker := time.NewTicker(20 * time.Second)
	for {
		<-ticker.C
		now := config.Now()
		cache.Lock()
		for steamId, state := range cache.state {
			if now.Sub(state.updatedAt) > time.Hour {
				delete(cache.state, steamId)
				log.WithFields(log.Fields{"sid": steamId}).Debugf("Player cache expired")
			}
		}
		cache.Unlock()
	}
}

type LogFilePayload struct {
	Server model.Server
	Lines  []string
	Map    string
}

// logReader is the fan-out orchestrator for game log events
// Registering receivers can be accomplished with RegisterLogEventReader
func (app *App) logReader() {
	var file *os.File
	if config.Debug.WriteUnhandledLogEvents {
		var errCreateFile error
		file, errCreateFile = os.Create("./unhandled_messages.log")
		if errCreateFile != nil {
			log.Panicf("Failed to open debug message log: %v", errCreateFile)
		}
		defer func() {
			if errClose := file.Close(); errClose != nil {
				log.Errorf("Failed to close unhandled_messages.log: %v", errClose)
			}
		}()
	}
	playerStateCache := newPlayerCache()
	for {
		select {
		case logFile := <-app.logFileChan:
			emitted := 0
			failed := 0
			unknown := 0
			ignored := 0
			for _, logLine := range logFile.Lines {
				var serverEvent model.ServerEvent
				errLogServerEvent := logToServerEvent(app.ctx, logFile.Server, logLine, app.store, playerStateCache, &serverEvent)
				if errLogServerEvent != nil {
					log.Errorf("Failed to parse: %v", errLogServerEvent)
					failed++
					continue
				}
				if serverEvent.EventType == logparse.IgnoredMsg {
					ignored++
					continue
				} else if serverEvent.EventType == logparse.UnknownMsg {
					unknown++
					if config.Debug.WriteUnhandledLogEvents {
						if _, errWrite := file.WriteString(logLine + "\n"); errWrite != nil {
							log.Errorf("Failed to write debug log: %v", errWrite)
						}
					}
				}
				event.Emit(serverEvent)
				emitted++
			}
			log.WithFields(log.Fields{"ok": emitted, "failed": failed, "unknown": unknown, "ignored": ignored}).
				Debugf("Completed emitting logfile events")
		case <-app.ctx.Done():
			log.Trace("logReader shutting down")
			return
		}
	}
}

func initFilters(ctx context.Context, database store.FilterStore) {
	// TODO load external lists via http
	localCtx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()
	words, errGetFilters := database.GetFilters(localCtx)
	if errGetFilters != nil {
		log.Fatal("Failed to load word list")
	}
	importFilteredWords(words)
	log.WithFields(log.Fields{"count": len(words), "list": "local", "type": "words"}).Debugf("Loaded blocklist")
}

func (app *App) initWorkers() {
	statusUpdateFreq, errDuration := time.ParseDuration(config.General.ServerStatusUpdateFreq)
	if errDuration != nil {
		log.Fatalf("Failed to parse server_status_update_freq: %v", errDuration)
	}
	masterUpdateFreq, errParseMasterUpdateFreq := time.ParseDuration(config.General.ServerStatusUpdateFreq)
	if errParseMasterUpdateFreq != nil {
		log.Fatalf("Failed to parse master_server_status_update_freq: %v", errParseMasterUpdateFreq)
	}
	go app.patreonUpdater()
	go app.banSweeper()
	go app.serverA2SStatusUpdater(statusUpdateFreq)
	go app.serverRCONStatusUpdater(statusUpdateFreq)
	go app.serverStateRefresher(statusUpdateFreq)
	go app.profileUpdater()
	go app.warnWorker()
	go app.logReader()
	go app.initLogSrc()
	go app.logMetricsConsumer()
	//go matchSummarizer(ctx, database)
	go app.playerMessageWriter()
	go app.playerConnectionWriter()
	go app.steamGroupMembershipUpdater()
	go app.localStatUpdater()
	go app.masterServerListUpdater(masterUpdateFreq)
	go app.cleanupTasks()
	go app.showReportMeta()
	go app.notificationSender()
	go app.demoCleaner()
}

// UDP log sink
func (app *App) initLogSrc() {
	logSrc, errLogSrc := newRemoteSrcdsLogSource(app.ctx, config.Log.SrcdsLogAddr, app.store)
	if errLogSrc != nil {
		log.Fatalf("Failed to setup udp log src: %v", errLogSrc)
	}
	logSrc.start(app.store)
}

func (app *App) sendUserNotification(pl notificationPayload) {
	select {
	case app.notificationChan <- pl:
	default:
		log.Error("Failed to write user notification payload, channel full")
	}
}

func (app *App) initDiscord(ctx context.Context, database store.Store, botSendMessageChan chan discordPayload) {
	if config.Discord.Token != "" {
		session, sessionErr := NewDiscord(ctx, app, database)
		if sessionErr != nil {
			log.Fatalf("Failed to setup session: %v", sessionErr)
		}
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case payload := <-botSendMessageChan:
					if !session.Connected.Load() {
						continue
					}
					if errSend := session.SendEmbed(payload.channelId, payload.embed); errSend != nil {
						log.Errorf("Failed to send discord payload: %v", errSend)
					}
				}
			}
		}()
		//l := log.StandardLogger()
		//l.AddHook(NewDiscordLogHook(botSendMessageChan))
		if errSessionStart := session.Start(ctx, config.Discord.Token); errSessionStart != nil {
			log.Errorf("discord returned error: %v", errSessionStart)
		}
	} else {
		log.Fatalf("discord enabled, but bot token invalid")
	}
}

func initNetBans() {
	for _, banList := range config.Net.Sources {
		if errImport := thirdparty.Import(banList); errImport != nil {
			log.Errorf("Failed to import banList: %v", errImport)
		}
	}
}

// validateLink is used in the case of discord origin actions that require mapping the
// discord member ID to a SteamID so that we can track its use and apply permissions, etc.
//
// This function will replace the discord member id value in the target field with
// the found SteamID, if any.
//func validateLink(ctx context.Context, database store.Store, sourceID action.Author, target *action.Author) error {
//	var p model.Person
//	if errGetPerson := database.GetPersonByDiscordID(ctx, string(sourceID), &p); errGetPerson != nil {
//		if errGetPerson == store.ErrNoResult {
//			return consts.ErrUnlinkedAccount
//		}
//		return consts.ErrInternal
//	}
//	*target = action.Author(p.SteamID.String())
//	return nil
//}
