// Package app is the main application and entry point. It implements the action.Executor and io.Closer interfaces.
package app

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"math/rand"
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

var (
	// BuildVersion holds the current git revision, as of build time
	BuildVersion = "master"
	logFileChan  chan *LogFilePayload
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
	serverStateMu      *sync.RWMutex
	serverState        model.ServerStateCollection

	bannedGroupMembers   map[steamid.GID]steamid.Collection
	bannedGroupMembersMu *sync.RWMutex
)

func init() {
	rand.Seed(time.Now().Unix())
	serverStateMu = &sync.RWMutex{}
	serverStateA2SMu = &sync.RWMutex{}
	serverStateStatusMu = &sync.RWMutex{}
	masterServerListMu = &sync.RWMutex{}
	logFileChan = make(chan *LogFilePayload, 10)
	discordSendMsg = make(chan discordPayload, 5)
	warningChan = make(chan newUserWarning)
	serverStateA2S = map[string]a2s.ServerInfo{}
	serverStateStatus = map[string]extra.Status{}

	bannedGroupMembers = map[steamid.GID]steamid.Collection{}
	bannedGroupMembersMu = &sync.RWMutex{}
}

type userWarning struct {
	WarnReason model.Reason
	CreatedOn  time.Time
}

type discordPayload struct {
	channelId string
	embed     *discordgo.MessageEmbed
}

// Start is the main application entry point
func Start(ctx context.Context) error {
	dbStore, dbErr := store.New(ctx, config.DB.DSN)
	if dbErr != nil {
		return errors.Wrapf(dbErr, "Failed to setup store")
	}
	defer func() {
		if errClose := dbStore.Close(); errClose != nil {
			log.Errorf("Error cleanly closing app: %v", errClose)
		}
	}()

	webService, errWeb := NewWeb(dbStore, discordSendMsg, logFileChan)
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
		go initDiscord(ctx, dbStore, discordSendMsg)
	} else {
		log.Warnf("discord bot not enabled")
	}

	// Start the background goroutine workers
	initWorkers(ctx, dbStore, discordSendMsg, logFileChan, warningChan)

	// Load the filtered word set into memory
	if config.Filter.Enabled {
		initFilters(ctx, dbStore)
	}

	// Start & block, listening on the HTTP server
	if errHttpListen := webService.ListenAndServe(ctx); errHttpListen != nil {
		return errors.Wrapf(errHttpListen, "Error shutting down service")
	}
	return nil
}

type newUserWarning struct {
	SteamId steamid.SID64
	userWarning
}

// warnWorker handles tracking and applying warnings based on incoming events
func warnWorker(ctx context.Context, newWarnings chan newUserWarning,
	botSendMessageChan chan discordPayload, database store.Store) {
	warnings := map[steamid.SID64][]userWarning{}
	eventChan := make(chan model.ServerEvent)
	if errRegister := event.Consume(eventChan, []logparse.EventType{logparse.Say, logparse.SayTeam}); errRegister != nil {
		log.Fatalf("Failed to register event reader: %v", errRegister)
	}
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case newWarn := <-newWarnings:
			if !newWarn.SteamId.Valid() {
				continue
			}
			_, found := warnings[newWarn.SteamId]
			if !found {
				warnings[newWarn.SteamId] = []userWarning{}
			}
			warnings[newWarn.SteamId] = append(warnings[newWarn.SteamId], newWarn.userWarning)
			if len(warnings[newWarn.SteamId]) >= config.General.WarningLimit {
				log.Errorf("Warn limit exceeded (%d): %d", newWarn.SteamId, len(warnings[newWarn.SteamId]))
				var errBan error
				var banSteam model.BanSteam

				if errOpts := NewBanSteam(model.StringSID(config.General.Owner.String()),
					model.StringSID(newWarn.SteamId.String()),
					model.Duration(config.General.WarningExceededDuration.String()),
					newWarn.WarnReason,
					newWarn.WarnReason.String(),
					"Automatic warning ban",
					model.System,
					0,
					model.Banned,
					&banSteam); errOpts != nil {
					log.Errorf("Failed to create warning ban: %v", errOpts)
					return
				}

				switch config.General.WarningExceededAction {
				case config.Gag:
					banSteam.BanType = model.NoComm
					errBan = BanSteam(ctx, database, &banSteam, botSendMessageChan)
				case config.Ban:
					banSteam.BanType = model.Banned
					errBan = BanSteam(ctx, database, &banSteam, botSendMessageChan)
				case config.Kick:
					var playerInfo model.PlayerInfo
					errBan = Kick(ctx, database, model.System, model.StringSID(newWarn.SteamId.String()),
						model.StringSID(config.General.Owner.String()), newWarn.WarnReason, &playerInfo)
				}

				if errBan != nil {
					log.WithFields(log.Fields{"action": config.General.WarningExceededAction}).
						Errorf("Failed to apply warning action: %v", errBan)
				}
			}
		case serverEvent := <-eventChan:
			msg, found := serverEvent.MetaData["msg"].(string)
			if !found {
				continue
			}
			matchedWord, matchedFilter := findFilteredWordMatch(msg)
			if matchedFilter != nil {
				warningChan <- newUserWarning{
					SteamId: serverEvent.Source.SteamID,
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
		case <-ticker.C:
			now := config.Now()
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
		case <-ctx.Done():
			log.Debugf("warnWorker shutting down")
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

func playerMessageWriter(ctx context.Context, database store.Store) {
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
		case <-ctx.Done():
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
				lCtx, cancel := context.WithTimeout(ctx, time.Second*5)
				if errChat := database.AddChatHistory(lCtx, &msg); errChat != nil {
					log.Errorf("Failed to add chat history: %v", errChat)
				}
				cancel()
				log.WithFields(log.Fields{"msg": msg}).Tracef("Saved message")
			}
		}
	}
}

func playerConnectionWriter(ctx context.Context, database store.Store) {
	serverEventChan := make(chan model.ServerEvent)
	if errRegister := event.Consume(serverEventChan, []logparse.EventType{logparse.Connected}); errRegister != nil {
		log.Warnf("logWriter Tried to register duplicate reader channel")
		return
	}
	for {
		select {
		case <-ctx.Done():
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
			lCtx, cancel := context.WithTimeout(ctx, time.Second*5)
			if errChat := database.AddConnectionHistory(lCtx, &msg); errChat != nil {
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
func logReader(ctx context.Context, logFileChan chan *LogFilePayload, db store.Store) {
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
		case logFile := <-logFileChan:
			emitted := 0
			failed := 0
			unknown := 0
			ignored := 0
			for _, logLine := range logFile.Lines {
				var serverEvent model.ServerEvent
				errLogServerEvent := logToServerEvent(ctx, logFile.Server, logLine, db, playerStateCache, &serverEvent)
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
		case <-ctx.Done():
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

func initWorkers(ctx context.Context, database store.Store, botSendMessageChan chan discordPayload,
	logFileC chan *LogFilePayload, warningChan chan newUserWarning) {

	freq, errDuration := time.ParseDuration(config.General.ServerStatusUpdateFreq)
	if errDuration != nil {
		log.Fatalf("Failed to parse server_status_update_freq: %v", errDuration)
	}

	masterUpdateFreq, errParseMasterUpdateFreq := time.ParseDuration(config.General.ServerStatusUpdateFreq)
	if errParseMasterUpdateFreq != nil {
		log.Fatalf("Failed to parse master_server_status_update_freq: %v", errParseMasterUpdateFreq)
	}

	go banSweeper(ctx, database)
	go mapChanger(ctx, database, time.Second*300)
	go serverA2SStatusUpdater(ctx, database, freq)
	go serverRCONStatusUpdater(ctx, database, freq)
	go serverStateRefresher(ctx, database, freq)
	go profileUpdater(ctx, database)
	go warnWorker(ctx, warningChan, botSendMessageChan, database)
	go logReader(ctx, logFileC, database)
	go initLogSrc(ctx, database)
	go logMetricsConsumer(ctx)
	//go matchSummarizer(ctx, database)
	go playerMessageWriter(ctx, database)
	go playerConnectionWriter(ctx, database)
	go steamGroupMembershipUpdater(ctx, database)
	go masterServerListUpdater(ctx, database, masterUpdateFreq)
}

// UDP log sink
func initLogSrc(ctx context.Context, database store.Store) {
	logSrc, errLogSrc := newRemoteSrcdsLogSource(ctx, config.Log.SrcdsLogAddr, database)
	if errLogSrc != nil {
		log.Fatalf("Failed to setup udp log src: %v", errLogSrc)
	}
	logSrc.start(database)
}

func initDiscord(ctx context.Context, database store.Store, botSendMessageChan chan discordPayload) {
	if config.Discord.Token != "" {
		session, sessionErr := NewDiscord(ctx, database)
		if sessionErr != nil {
			log.Fatalf("Failed to setup session: %v", sessionErr)
		}
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case payload := <-botSendMessageChan:
					if !session.Ready {
						continue
					}
					if errSend := session.SendEmbed(payload.channelId, payload.embed); errSend != nil {
						log.Errorf("Failed to send discord payload: %v", errSend)
					}
				}
			}
		}()
		l := log.StandardLogger()
		l.AddHook(NewDiscordLogHook(botSendMessageChan))
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
