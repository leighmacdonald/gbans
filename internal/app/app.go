// Package app is the main application and entry point. It implements the action.Executor and io.Closer interfaces.
package app

import (
	"context"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/event"
	"github.com/leighmacdonald/gbans/internal/external"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	// BuildVersion holds the current git revision, as of build time
	BuildVersion   = "master"
	ctx            context.Context
	warnings       map[steamid.SID64][]userWarning
	warningsMu     *sync.RWMutex
	logPayloadChan chan model.LogPayload
	serversState   map[string]*model.ServerState
	serversStateMu *sync.RWMutex
	discordSendMsg chan discordPayload
)

func init() {
	rand.Seed(time.Now().Unix())
	serversStateMu = &sync.RWMutex{}
	ctx = context.Background()
	warnings = map[steamid.SID64][]userWarning{}
	warningsMu = &sync.RWMutex{}
	logPayloadChan = make(chan model.LogPayload, 100000)
	discordSendMsg = make(chan discordPayload)
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
func Start() error {
	dbStore, dbErr := store.New(config.DB.DSN)
	if dbErr != nil {
		return errors.Wrapf(dbErr, "Failed to setup store")
	}
	defer func() {
		if errClose := dbStore.Close(); errClose != nil {
			log.Errorf("Error cleanly closing app: %v", errClose)
		}
	}()

	webService, errWeb := NewWeb(dbStore, discordSendMsg)
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
		go initDiscord(dbStore, discordSendMsg)
	} else {
		log.Warnf("discord bot not enabled")
	}

	// Start the background goroutine workers
	initWorkers(dbStore, discordSendMsg)

	// Load the filtered word set into memory
	if config.Filter.Enabled {
		initFilters(dbStore)
	}

	// Start & block, listening on the HTTP server
	if errHttpListen := webService.ListenAndServe(); errHttpListen != nil {
		return errors.Wrapf(errHttpListen, "Error shutting down service")
	}
	return nil
}

// warnWorker will periodically flush out warning older than `config.General.WarningTimeout`
func warnWorker() {
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

// logWriter handles writing log events to the database. It does it in batches for performance
// reasons.
func logWriter(database store.StatStore) {
	const (
		writeFrequency = time.Second * 5
	)
	var logCache []model.ServerEvent
	serverEventChan := make(chan model.ServerEvent, 100000)
	if errRegister := event.RegisterConsumer(serverEventChan, []logparse.EventType{logparse.Any}); errRegister != nil {
		log.Warnf("logWriter Tried to register duplicate reader channel")
	}
	writeTicker := time.NewTicker(writeFrequency)
	var writeLogs = func() {
		if len(logCache) == 0 {
			return
		}
		if errInsert := database.BatchInsertServerLogs(ctx, logCache); errInsert != nil {
			log.Errorf("Failed to batch insert logs: %v", errInsert)
		}
		logCache = nil
	}
	for {
		select {
		case serverEvent := <-serverEventChan:
			if serverEvent.EventType == logparse.IgnoredMsg {
				continue
			}
			logCache = append(logCache, serverEvent)
			if len(logCache) >= 500 {
				writeLogs()
			}
		case <-writeTicker.C:
			writeLogs()
		case <-ctx.Done():
			log.Debugf("logWriter shuttings down")
			return
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
		select {
		case <-ticker.C:
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
}

// logReader is the fan-out orchestrator for game log events
// Registering receivers can be accomplished with RegisterLogEventReader
func logReader(db store.Store) {
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

	getPlayer := func(id string, playerCache map[string]any, person *model.Person) error {
		sid1Str, ok := playerCache[id]
		if ok {
			s := steamid.SID3ToSID64(steamid.SID3(sid1Str.(string)))
			if errGetPerson := db.GetOrCreatePersonBySteamID(ctx, s, person); errGetPerson != nil {
				return errors.Wrapf(errGetPerson, "Failed to load player1 %s: %s", sid1Str, errGetPerson.Error())
			}
			return nil
		}
		return nil
	}

	getServer := func(serverName string, s *model.Server) error {
		return db.GetServerByName(ctx, serverName, s)
	}

	playerStateCache := newPlayerCache()
	for {
		select {
		case payload := <-logPayloadChan:
			var serverEvent model.ServerEvent
			errLogServerEvent := logToServerEvent(payload, playerStateCache, &serverEvent, getPlayer, getServer)
			if errLogServerEvent != nil {
				log.Errorf("Failed to parse: %v", errLogServerEvent)
				continue
			}
			if serverEvent.EventType == logparse.UnknownMsg {
				if _, errWrite := file.WriteString(payload.Message + "\n"); errWrite != nil {
					log.Errorf("Failed to write debug log: %v", errWrite)
				}
			}
			event.Emit(serverEvent)
		case <-ctx.Done():
			log.Debugf("logReader shutting down")
			return
		}
	}
}

type getPlayerFn func(id string, v map[string]any, person *model.Person) error
type getServerFn func(serverName string, server *model.Server) error

func logToServerEvent(payload model.LogPayload, playerStateCache *playerCache,
	event *model.ServerEvent, getPlayer getPlayerFn, getServer getServerFn) error {
	parseResult := logparse.Parse(payload.Message)
	var server model.Server
	if errServer := getServer(payload.ServerName, &server); errServer != nil {
		return errors.Wrapf(errServer, "Failed to get server for log message: %parseResult", payload.Message)
	}
	event.Server = &server
	event.EventType = parseResult.MsgType
	var playerSource model.Person
	errGetSourcePlayer := getPlayer("sid", parseResult.Values, &playerSource)
	if errGetSourcePlayer != nil {
	} else {
		event.Source = &playerSource
	}
	var playerTarget model.Person
	if errGetTargetPlayer := getPlayer("sid2", parseResult.Values, &playerTarget); errGetTargetPlayer != nil {
	} else {
		event.Target = &playerTarget
	}
	aposValue, aposFound := parseResult.Values["attacker_position"]
	if aposFound {
		var attackerPosition logparse.Pos
		if errParsePOS := logparse.ParsePOS(aposValue.(string), &attackerPosition); errParsePOS != nil {
			log.Warnf("Failed to parse attacker position: %p", errParsePOS)
		}
		event.AttackerPOS = attackerPosition
		delete(parseResult.Values, "attacker_position")
	}
	vposValue, vposFound := parseResult.Values["victim_position"]
	if vposFound {
		var victimPosition logparse.Pos
		if errParsePOS := logparse.ParsePOS(vposValue.(string), &victimPosition); errParsePOS != nil {
			log.Warnf("Failed to parse victim position: %parseResult", errParsePOS)
		}
		event.VictimPOS = victimPosition
		delete(parseResult.Values, "victim_position")
	}
	asValue, asFound := parseResult.Values["assister_position"]
	if asFound {
		var assisterPosition logparse.Pos
		if errParsePOS := logparse.ParsePOS(asValue.(string), &assisterPosition); errParsePOS != nil {
			log.Warnf("Failed to parse assister position: %parseResult", errParsePOS)
		}
		event.AssisterPOS = assisterPosition
		delete(parseResult.Values, "assister_position")
	}

	critType, critTypeFound := parseResult.Values["crit"]
	if critTypeFound {
		event.Crit = critType.(logparse.CritType)
		delete(parseResult.Values, "crit")
	}

	weapon := logparse.UnknownWeapon
	weaponValue, weaponFound := parseResult.Values["weapon"]
	if weaponFound {
		weapon = logparse.ParseWeapon(weaponValue.(string))
	}
	event.Weapon = weapon

	var class logparse.PlayerClass
	classValue, classFound := parseResult.Values["class"]
	if classFound {
		if !logparse.ParsePlayerClass(classValue.(string), &class) {
			class = logparse.Spectator
		}
		delete(parseResult.Values, "class")
	} else if event.Source != nil {
		class = playerStateCache.getClass(event.Source.SteamID)
	}
	event.PlayerClass = class

	var damage int64
	dmgValue, dmgFound := parseResult.Values["damage"]
	if dmgFound {
		parsedDamage, errParseDamage := strconv.ParseInt(dmgValue.(string), 10, 32)
		if errParseDamage != nil {
			log.Warnf("failed to parse damage value: %parseResult", errParseDamage)
		}
		damage = parsedDamage
		delete(parseResult.Values, "damage")
	}
	event.Damage = damage

	var realDamage int64
	realDmgValue, realDmgFound := parseResult.Values["realdamage"]
	if realDmgFound {
		parsedRealDamage, errParseRealDamage := strconv.ParseInt(realDmgValue.(string), 10, 32)
		if errParseRealDamage != nil {
			log.Warnf("failed to parse damage value: %parseResult", errParseRealDamage)
		}
		realDamage = parsedRealDamage
		delete(parseResult.Values, "realdamage")
	}
	event.RealDamage = realDamage

	var item logparse.PickupItem
	itemValue, itemFound := parseResult.Values["item"]
	if itemFound {
		if !logparse.ParsePickupItem(itemValue.(string), &item) {
			item = 0
		}
	}
	event.Item = item

	var team logparse.Team
	teamValue, teamFound := parseResult.Values["team"]
	if teamFound {
		if !logparse.ParseTeam(teamValue.(string), &team) {
			team = 0
		}
	} else {
		if event.Source != nil {
			team = playerStateCache.getTeam(event.Source.SteamID)
		}
	}
	event.Team = team

	healingValue, healingFound := parseResult.Values["healing"]
	if healingFound {
		healingP, errParseHealing := strconv.ParseInt(healingValue.(string), 10, 32)
		if errParseHealing != nil {
			log.Warnf("failed to parse healing value: %parseResult", errParseHealing)
		}
		event.Healing = healingP
	}

	createdOnValue, createdOnFound := parseResult.Values["created_on"]
	if !createdOnFound {
		return errors.New("created_on missing")
	}

	event.CreatedOn = createdOnValue.(time.Time)

	// Remove keys that get mapped to actual schema columns
	for _, key := range []string{
		"created_on", "item", "weapon", "healing",
		"name", "pid", "sid", "team",
		"name2", "pid2", "sid2", "team2"} {
		delete(parseResult.Values, key)
	}
	event.MetaData = parseResult.Values
	switch parseResult.MsgType {
	case logparse.SpawnedAs:
		playerStateCache.setClass(event.Source.SteamID, event.PlayerClass)
	case logparse.JoinedTeam:
		playerStateCache.setTeam(event.Source.SteamID, event.Team)
	}
	return nil
}

// addWarning records a user warning into memory. This is not persistent, so application
// restarts will wipe the user's history.
//
// Warning are flushed once they reach N age as defined by `config.General.WarningTimeout
func addWarning(database store.Store, sid64 steamid.SID64, reason warnReason, botSendMessageChan chan discordPayload) {
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
			errBan = Ban(database, options, &ban, botSendMessageChan)
		case config.Ban:
			options.banType = model.Banned
			errBan = Ban(database, options, &ban, botSendMessageChan)
		case config.Kick:
			var playerInfo model.PlayerInfo
			errBan = Kick(database, model.System, model.Target(sid64.String()),
				model.Target(config.General.Owner.String()), warnReasonString(reason), &playerInfo)
		}
		if errBan != nil {
			log.WithFields(log.Fields{"action": config.General.WarningExceededAction}).
				Errorf("Failed to apply warning action: %v", errBan)
		}
	}
}

func initFilters(database store.FilterStore) {
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

func initWorkers(database store.Store, botSendMessageChan chan discordPayload) {
	go banSweeper(database)
	go mapChanger(database, time.Second*5)
	go serverStateUpdater(database)
	go profileUpdater(database)
	go warnWorker()
	go logReader(database)
	go logWriter(database)
	go filterWorker(database, botSendMessageChan)
	go initLogSrc(database)
	go logMetricsConsumer()
}

func initLogSrc(database store.Store) {
	logSrc, errLogSrc := newRemoteSrcdsLogSource(config.Log.SrcdsLogAddr, database, logPayloadChan)
	if errLogSrc != nil {
		log.Fatalf("Failed to setup udp log src: %v", errLogSrc)
	}
	logSrc.start()
}

func initDiscord(database store.Store, botSendMessageChan chan discordPayload) {
	if config.Discord.Token != "" {
		session, sessionErr := NewDiscord(database)
		if sessionErr != nil {
			log.Fatalf("Failed to setup session: %v", sessionErr)
		}
		events := make(chan model.ServerEvent)
		if len(config.Discord.LogChannelID) > 0 {
			if errRegister := event.RegisterConsumer(events, []logparse.EventType{logparse.Say, logparse.SayTeam}); errRegister != nil {
				log.Warnf("Error registering discord log event reader")
			}
		}
		go func() {
			for {
				select {
				case payload := <-botSendMessageChan:
					if errSend := session.SendEmbed(payload.channelId, payload.message); errSend != nil {
						log.Errorf("Failed to send discord payload: %v", errSend)
					}
				}
			}
		}()
		if errSessionStart := session.Start(ctx, config.Discord.Token, events); errSessionStart != nil {
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
