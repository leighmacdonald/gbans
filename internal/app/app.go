// Package app is the main application and entry point. It implements the action.Executor and io.Closer interfaces.
package app

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/event"
	"github.com/leighmacdonald/gbans/internal/external"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"github.com/rumblefrog/go-a2s"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"sync"
	"time"
)

var (
	// BuildVersion holds the current git revision, as of build time
	BuildVersion = "master"
	warnings     map[steamid.SID64][]userWarning
	warningsMu   *sync.RWMutex
	logFileChan  chan *LogFilePayload
	// Current known state of the servers rcon status command
	serverStateStatus   map[string]extra.Status
	serverStateStatusMu *sync.RWMutex
	// Current known state of the servers a2s server info query
	serverStateA2S   map[string]a2s.ServerInfo
	serverStateA2SMu *sync.RWMutex
	discordSendMsg   chan discordPayload
	serverStateMu    *sync.RWMutex
	serverState      model.ServerStateCollection
)

func init() {
	rand.Seed(time.Now().Unix())
	serverStateMu = &sync.RWMutex{}
	serverStateA2SMu = &sync.RWMutex{}
	serverStateStatusMu = &sync.RWMutex{}

	warnings = map[steamid.SID64][]userWarning{}
	warningsMu = &sync.RWMutex{}
	logFileChan = make(chan *LogFilePayload, 10)
	discordSendMsg = make(chan discordPayload)
	serverStateA2S = map[string]a2s.ServerInfo{}
	serverStateStatus = map[string]extra.Status{}
}

type warnReason int

const (
	warnLanguage warnReason = iota
)

func warnReasonString(reason warnReason) string {
	switch reason {
	case warnLanguage:
		return "Language"
	default:
		return "Unset"
	}
}

type userWarning struct {
	WarnReason warnReason
	CreatedOn  time.Time
}

type discordPayload struct {
	channelId string
	message   *discordgo.MessageEmbed
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
	initWorkers(ctx, dbStore, discordSendMsg, logFileChan)

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

// warnWorker will periodically flush out warning older than `config.General.WarningTimeout`
func warnWorker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			now := config.Now()
			warningsMu.Lock()
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
			warningsMu.Unlock()
		case <-ctx.Done():
			log.Debugf("warnWorker shutting down")
			return
		}
	}
}

func matchSummarizer(ctx context.Context, db store.Store) {
	eventChan := make(chan model.ServerEvent)
	if errReg := event.Consume(eventChan, []logparse.EventType{logparse.Any}); errReg != nil {
		log.Warnf("logWriter Tried to register duplicate reader channel")
	}
	match := model.NewMatch()
	var curServer model.Server
	for {
		select {
		case evt := <-eventChan:
			if match.ServerId == 0 && evt.Server.ServerID > 0 {
				curServer = evt.Server
				match.ServerId = curServer.ServerID
			}
			// Apply the update before any secondary side effects trigger
			if errApply := match.Apply(evt); errApply != nil {
				log.Tracef("Error applying event: %v", errApply)
			}
			switch evt.EventType {
			case logparse.WGameOver:
				if errSave := db.MatchSave(ctx, &match); errSave != nil {
					log.Errorf("Failed to save match: %v", errSave)
				}
				sendDiscordNotif(curServer, &match)
				match = model.NewMatch()
				log.Debugf("New match summary created")
			}
		}
	}
}

func sendDiscordNotif(server model.Server, match *model.Match) {
	embed := &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeRich,
		Title:       fmt.Sprintf("Match results - %s - %s", server.ServerNameShort, match.MapName),
		Description: "Match results",
		Color:       int(green),
		URL:         fmt.Sprintf("https://gbans.uncletopia.com/match/%d", match.MatchID),
	}
	redScore := 0
	bluScore := 0
	for _, round := range match.Rounds {
		redScore += round.Score.Red
		bluScore += round.Score.Blu
	}

	addFieldInline(embed, "Red Score", fmt.Sprintf("%d", redScore))
	addFieldInline(embed, "Blu Score", fmt.Sprintf("%d", bluScore))
	found := 0
	for _, team := range []logparse.Team{logparse.RED, logparse.BLU} {
		teamStats, statsFound := match.TeamSums[team]
		if statsFound {
			addFieldInline(embed, fmt.Sprintf("%s Kills", team.String()), fmt.Sprintf("%d", teamStats.Kills))
			addFieldInline(embed, fmt.Sprintf("%s Damage", team.String()), fmt.Sprintf("%d", teamStats.Damage))
			addFieldInline(embed, fmt.Sprintf("%s Ubers", team.String()), fmt.Sprintf("%d", teamStats.Charges))
			addFieldInline(embed, fmt.Sprintf("%s Drops", team.String()), fmt.Sprintf("%d", teamStats.Drops))
			found++
		}
	}
	if found == 2 {
		log.Debugf("Sending discord summary")
		select {
		case discordSendMsg <- discordPayload{channelId: config.Discord.LogChannelID, message: embed}:
		default:
			log.Warnf("Cannot send discord payload, channel full")
		}
	}
}

func playerStateWriter(ctx context.Context, database store.Store) {
	serverEventChan := make(chan model.ServerEvent)
	if errRegister := event.Consume(serverEventChan, []logparse.EventType{
		logparse.Connected,
		logparse.Disconnected,
		logparse.Say,
		logparse.SayTeam,
	}); errRegister != nil {
		log.Warnf("logWriter Tried to register duplicate reader channel")
	}
	for {
		select {
		case evt := <-serverEventChan:
			switch evt.EventType {
			case logparse.Connected:
			case logparse.Disconnected:

			}
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

			for _, logLine := range logFile.Lines {
				var serverEvent model.ServerEvent
				errLogServerEvent := logToServerEvent(ctx, logFile.Server, logLine, db, playerStateCache, &serverEvent)
				if errLogServerEvent != nil {
					log.Errorf("Failed to parse: %v", errLogServerEvent)
					continue
				}
				if serverEvent.EventType == logparse.UnknownMsg {
					if _, errWrite := file.WriteString(logLine + "\n"); errWrite != nil {
						log.Errorf("Failed to write debug log: %v", errWrite)
					}
				}
				event.Emit(serverEvent)
				emitted++
			}
			log.WithFields(log.Fields{"count": emitted}).Debugf("Completed emitting logfile events")
		case <-ctx.Done():
			log.Debugf("logReader shutting down")
			return
		}
	}
}

// addWarning records a user warning into memory. This is not persistent, so application
// restarts will wipe the user's history.
//
// Warning are flushed once they reach N age as defined by `config.General.WarningTimeout
func addWarning(ctx context.Context, database store.Store, sid64 steamid.SID64, reason warnReason, botSendMessageChan chan discordPayload) {
	warningsMu.Lock()
	_, found := warnings[sid64]
	if !found {
		warnings[sid64] = []userWarning{}
	}
	warnings[sid64] = append(warnings[sid64], userWarning{
		WarnReason: reason,
		CreatedOn:  config.Now(),
	})
	warningsMu.Unlock()
	if len(warnings[sid64]) >= config.General.WarningLimit {
		var ban model.Ban
		log.Errorf("Warn limit exceeded (%d): %d", sid64, len(warnings[sid64]))
		var errBan error
		options := banOpts{
			target:   model.Target(sid64.String()),
			author:   model.Target(config.General.Owner.String()),
			duration: model.Duration(config.General.WarningExceededDuration.String()),
			reason:   warnReasonString(reason),
			origin:   model.System,
		}
		switch config.General.WarningExceededAction {
		case config.Gag:
			options.banType = model.NoComm
			errBan = Ban(ctx, database, options, &ban, botSendMessageChan)
		case config.Ban:
			options.banType = model.Banned
			errBan = Ban(ctx, database, options, &ban, botSendMessageChan)
		case config.Kick:
			var playerInfo model.PlayerInfo
			errBan = Kick(ctx, database, model.System, model.Target(sid64.String()),
				model.Target(config.General.Owner.String()), warnReasonString(reason), &playerInfo)
		}
		if errBan != nil {
			log.WithFields(log.Fields{"action": config.General.WarningExceededAction}).
				Errorf("Failed to apply warning action: %v", errBan)
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

func initWorkers(ctx context.Context, database store.Store, botSendMessageChan chan discordPayload, logFileC chan *LogFilePayload) {
	go banSweeper(ctx, database)
	go mapChanger(ctx, database, time.Second*5)

	freq, errDuration := time.ParseDuration(config.General.ServerStatusUpdateFreq)
	if errDuration != nil {
		log.Fatalf("Failed to parse server_status_update_freq: %v", errDuration)
	}
	go serverA2SStatusUpdater(ctx, database, freq)
	go serverRCONStatusUpdater(ctx, database, freq)
	go serverStateRefresher(ctx, database)
	go profileUpdater(ctx, database)
	go warnWorker(ctx)
	go logReader(ctx, logFileC, database)
	go filterWorker(ctx, database, botSendMessageChan)
	//go initLogSrc(ctx, database)
	go logMetricsConsumer(ctx)
	go matchSummarizer(ctx, database)
}

// UDP log sink
//func initLogSrc(ctx context.Context, database store.Store) {
//	logSrc, errLogSrc := newRemoteSrcdsLogSource(ctx, config.Log.SrcdsLogAddr, database, logPayloadChan)
//	if errLogSrc != nil {
//		log.Fatalf("Failed to setup udp log src: %v", errLogSrc)
//	}
//	logSrc.start()
//}

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
					if errSend := session.SendEmbed(payload.channelId, payload.message); errSend != nil {
						log.Errorf("Failed to send discord payload: %v", errSend)
					}
				}
			}
		}()
		if errSessionStart := session.Start(ctx, config.Discord.Token); errSessionStart != nil {
			log.Errorf("discord returned error: %v", errSessionStart)
		}
	} else {
		log.Fatalf("discord enabled, but bot token invalid")
	}
}

func initNetBans() {
	for _, banList := range config.Net.Sources {
		if errImport := external.Import(banList); errImport != nil {
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
