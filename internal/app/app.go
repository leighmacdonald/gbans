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
	logRawQueue    chan model.LogPayload
	serversState   map[string]model.ServerState
	serversStateMu *sync.RWMutex
	discordSendMsg chan discordPayload
)

func init() {
	rand.Seed(time.Now().Unix())
	serversStateMu = &sync.RWMutex{}
	ctx = context.Background()
	warnings = map[steamid.SID64][]userWarning{}
	warningsMu = &sync.RWMutex{}
	logRawQueue = make(chan model.LogPayload, 10000)
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
	dbStore, se := store.New(config.DB.DSN)
	if se != nil {
		return errors.Wrapf(se, "Failed to setup store")
	}
	defer func() {
		if errClose := dbStore.Close(); errClose != nil {
			log.Errorf("Error cleanly closing app: %v", errClose)
		}
	}()

	webService, we := NewWeb(dbStore, discordSendMsg)
	if we != nil {
		return errors.Wrapf(we, "Failed to setup web")
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
	if err := webService.ListenAndServe(); err != nil {
		return errors.Wrapf(err, "Error shutting down service")
	}
	return nil
}

// warnWorker will periodically flush out warning older than `config.General.WarningTimeout`
func warnWorker() {
	t := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-t.C:
			now := config.Now()
			warningsMu.Lock()
			for k := range warnings {
				for i, w := range warnings[k] {
					if now.Sub(w.CreatedOn) > config.General.WarningTimeout {
						if len(warnings[k]) > 1 {
							warnings[k] = append(warnings[k][:i], warnings[k][i+1])
						} else {
							warnings[k] = nil
						}
					}
					if len(warnings[k]) == 0 {
						delete(warnings, k)
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
func logWriter(db store.StatStore) {
	const (
		freq = time.Second * 10
	)
	var logCache []model.ServerEvent
	events := make(chan model.ServerEvent, 1000)
	if err := event.RegisterConsumer(events, []logparse.EventType{logparse.Any}); err != nil {
		log.Warnf("logWriter Tried to register duplicate reader channel")
	}
	t := time.NewTicker(freq)
	var f = func() {
		if len(logCache) == 0 {
			return
		}
		if errI := db.BatchInsertServerLogs(ctx, logCache); errI != nil {
			log.Errorf("Failed to batch insert logs: %v", errI)
		}
		logCache = nil
	}
	for {
		select {
		case evt := <-events:
			if evt.EventType != logparse.IgnoredMsg {
				continue
			}
			logCache = append(logCache, evt)
			// Update immediately if we have enough volume
			if len(logCache) >= 1000 {
				t.Stop()
				f()
				t.Reset(freq)
			}
		case <-t.C:
			f()
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

func (c playerCache) setTeam(sid steamid.SID64, team logparse.Team) {
	c.Lock()
	defer c.Unlock()
	s, found := c.state[sid]
	if !found {
		s = playerEventState{}
	}
	s.team = team
	s.updatedAt = config.Now()
	c.state[sid] = s
}

func (c playerCache) setClass(sid steamid.SID64, class logparse.PlayerClass) {
	c.Lock()
	defer c.Unlock()
	s, found := c.state[sid]
	if !found {
		s = playerEventState{}
	}
	s.class = class
	s.updatedAt = config.Now()
	c.state[sid] = s
}

func (c playerCache) getClass(sid steamid.SID64) logparse.PlayerClass {
	c.RLock()
	defer c.RUnlock()
	pc, found := c.state[sid]
	if !found {
		return logparse.Spectator
	}
	return pc.class
}

func (c playerCache) getTeam(sid steamid.SID64) logparse.Team {
	c.RLock()
	defer c.RUnlock()
	pc, found := c.state[sid]
	if !found {
		return logparse.SPEC
	}
	return pc.team
}

func (c playerCache) cleanupWorker() {
	t := time.NewTicker(20 * time.Second)
	for {
		select {
		case <-t.C:
			now := config.Now()
			c.Lock()
			for k, v := range c.state {
				if now.Sub(v.updatedAt) > time.Hour {
					delete(c.state, k)
					log.WithFields(log.Fields{"sid": k}).Debugf("Player cache expired")
				}
			}
			c.Unlock()
		}
	}
}

// logReader is the fan-out orchestrator for game log events
// Registering receivers can be accomplished with RegisterLogEventReader
func logReader(db store.Store) {
	getPlayer := func(id string, v map[string]any) *model.Person {
		sid1Str, ok := v[id]
		if ok {
			s := steamid.SID3ToSID64(steamid.SID3(sid1Str.(string)))
			p := model.NewPerson(s)
			if err := db.GetOrCreatePersonBySteamID(ctx, s, &p); err != nil {
				log.Errorf("Failed to load player1 %s: %s", sid1Str, err.Error())
				return nil
			}
			return &p
		}
		return nil
	}
	var f *os.File
	if config.Debug.WriteUnhandledLogEvents {
		var errOf error
		f, errOf = os.Create("./unhandled_messages.log")
		if errOf != nil {
			log.Panicf("Failed to open debug message log: %v", errOf)
		}
		defer f.Close()
	}
	playerStateCache := newPlayerCache()
	for {
		select {
		case raw := <-logRawQueue:
			v := logparse.Parse(raw.Message)
			var s model.Server
			if e := db.GetServerByName(ctx, raw.ServerName, &s); e != nil {
				log.Errorf("Failed to get server for log message: %v", e)
				continue
			}
			var source = getPlayer("sid", v.Values)
			var target = getPlayer("sid2", v.Values)
			var (
				apos, vpos, aspos logparse.Pos
			)
			aposValue, aposFound := v.Values["attacker_position"]
			if aposFound {
				var apv logparse.Pos
				if err := logparse.ParsePOS(aposValue.(string), &apv); err != nil {
					log.Warnf("Failed to parse attacker position: %v", err)
				}
				apos = apv
				delete(v.Values, "attacker_position")
			}
			vposValue, vposFound := v.Values["victim_position"]
			if vposFound {
				var vpv logparse.Pos
				if err := logparse.ParsePOS(vposValue.(string), &vpv); err != nil {
					log.Warnf("Failed to parse victim position: %v", err)
				}
				vpos = vpv
				delete(v.Values, "victim_position")
			}
			asValue, asFound := v.Values["assister_position"]
			if asFound {
				var asPosValue logparse.Pos
				if err := logparse.ParsePOS(asValue.(string), &asPosValue); err != nil {
					log.Warnf("Failed to parse assister position: %v", err)
				}
				aspos = asPosValue
				delete(v.Values, "assister_position")
			}
			weapon := logparse.UnknownWeapon
			weaponValue, weaponFound := v.Values["weapon"]
			if weaponFound {
				weapon = logparse.ParseWeapon(weaponValue.(string))
			}
			var class logparse.PlayerClass
			classValue, classFound := v.Values["class"]
			if classFound {
				if !logparse.ParsePlayerClass(classValue.(string), &class) {
					class = logparse.Spectator
				}
				delete(v.Values, "class")
			} else if source != nil {
				class = playerStateCache.getClass(source.SteamID)
			}
			var damage int64
			dmgValue, dmgFound := v.Values["realdamage"]
			if !dmgFound {
				dmgValue, dmgFound = v.Values["damage"]
			}
			if dmgFound {
				damageP, err := strconv.ParseInt(dmgValue.(string), 10, 32)
				if err != nil {
					log.Warnf("failed to parse damage value: %v", err)
				}
				damage = damageP
				delete(v.Values, "realdamage")
				delete(v.Values, "damage")
			}
			var item logparse.PickupItem
			itemValue, itemFound := v.Values["item"]
			if itemFound {
				if !logparse.ParsePickupItem(itemValue.(string), &item) {
					item = 0
				}
			}

			var team logparse.Team
			teamValue, teamFound := v.Values["team"]
			if teamFound {
				if !logparse.ParseTeam(teamValue.(string), &team) {
					team = 0
				}
			} else {
				if source != nil {
					team = playerStateCache.getTeam(source.SteamID)
				}
			}

			var healing int64
			healingValue, healingFound := v.Values["healing"]
			if healingFound {
				healingP, err := strconv.ParseInt(healingValue.(string), 10, 32)
				if err != nil {
					log.Warnf("failed to parse healing value: %v", err)
				}
				healing = healingP
			}
			// Remove keys that get mapped to actual schema columns
			for _, k := range []string{
				"time", "date", "item", "weapon", "healing",
				"name", "pid", "sid", "team",
				"name2", "pid2", "sid2", "team2"} {
				delete(v.Values, k)
			}
			se := model.ServerEvent{
				Server:      &s,
				EventType:   v.MsgType,
				Source:      source,
				Target:      target,
				PlayerClass: class,
				Team:        team,
				Item:        item,
				Weapon:      weapon,
				Damage:      damage,
				AttackerPOS: apos,
				VictimPOS:   vpos,
				AssisterPOS: aspos,
				Healing:     healing,
				CreatedOn:   config.Now(),
				MetaData:    v.Values,
			}
			switch v.MsgType {
			case logparse.SpawnedAs:
				playerStateCache.setClass(se.Source.SteamID, se.PlayerClass)
			case logparse.JoinedTeam:
				playerStateCache.setTeam(se.Source.SteamID, se.Team)
			}
			if v.MsgType == logparse.UnknownMsg {
				if _, errWrite := f.WriteString(raw.Message + "\n"); errWrite != nil {
					log.Errorf("Failed to write debug log: %v", errWrite)
				}
			}
			event.Emit(se)
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
func addWarning(db store.Store, sid64 steamid.SID64, reason warnReason, botSendMessageChan chan discordPayload) {
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
		var err error
		bo := banOpts{
			target:   model.Target(sid64.String()),
			author:   model.Target(config.General.Owner.String()),
			duration: model.Duration(config.General.WarningExceededDuration.String()),
			reason:   warnReasonString(reason),
			origin:   model.System,
		}
		switch config.General.WarningExceededAction {
		case config.Gag:
			bo.banType = model.NoComm
			err = Ban(db, bo, &ban, botSendMessageChan)
		case config.Ban:
			bo.banType = model.Banned
			err = Ban(db, bo, &ban, botSendMessageChan)
		case config.Kick:
			var pi model.PlayerInfo
			err = Kick(db, model.System, model.Target(sid64.String()),
				model.Target(config.General.Owner.String()), warnReasonString(reason), &pi)
		}
		if err != nil {
			log.WithFields(log.Fields{"action": config.General.WarningExceededAction}).
				Errorf("Failed to apply warning action: %v", err)
		}
	}
}

func initFilters(db store.FilterStore) {
	// TODO load external lists via http
	c, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()
	words, err := db.GetFilters(c)
	if err != nil {
		log.Fatal("Failed to load word list")
	}
	importFilteredWords(words)
	log.WithFields(log.Fields{"count": len(words), "list": "local", "type": "words"}).Debugf("Loaded blocklist")
}

func initWorkers(db store.Store, botSendMessageChan chan discordPayload) {
	go banSweeper(db)
	go mapChanger(db, time.Second*5)
	go serverStateUpdater(db)
	go profileUpdater(db)
	go warnWorker()
	go logReader(db)
	go logWriter(db)
	go filterWorker(db, botSendMessageChan)
	go initLogSrc(db)
	go logMetricsConsumer()
}

func initLogSrc(db store.Store) {
	logSrc, errLogSrc := newRemoteSrcdsLogSource(config.Log.SrcdsLogAddr, db, logRawQueue)
	if errLogSrc != nil {
		log.Fatalf("Failed to setup udp log src: %v", errLogSrc)
	}
	logSrc.start()
}

func initDiscord(db store.Store, botSendMessageChan chan discordPayload) {
	if config.Discord.Token != "" {
		bot, be := NewDiscord(db)
		if be != nil {
			log.Fatalf("Failed to setup bot: %v", be)
		}
		events := make(chan model.ServerEvent)
		if len(config.Discord.LogChannelID) > 0 {
			if err := event.RegisterConsumer(events, []logparse.EventType{logparse.Say, logparse.SayTeam}); err != nil {
				log.Warnf("Error registering discord log event reader")
			}
		}
		go func() {
			for {
				select {
				case m := <-botSendMessageChan:
					if errSend := bot.SendEmbed(m.channelId, m.message); errSend != nil {
						log.Errorf("Failed to send discord message: %v", errSend)
					}
				}
			}
		}()
		if errBS := bot.Start(ctx, config.Discord.Token, events); errBS != nil {
			log.Errorf("discord returned error: %v", errBS)
		}
	} else {
		log.Fatalf("discord enabled, but bot token invalid")
	}
}

func initNetBans() {
	for _, list := range config.Net.Sources {
		if err := external.Import(list); err != nil {
			log.Errorf("Failed to import list: %v", err)
		}
	}
}

// validateLink is used in the case of discord origin actions that require mapping the
// discord member ID to a SteamID so that we can track its use and apply permissions, etc.
//
// This function will replace the discord member id value in the target field with
// the found SteamID, if any.
//func validateLink(ctx context.Context, db store.Store, sourceID action.Author, target *action.Author) error {
//	var p model.Person
//	if err := db.GetPersonByDiscordID(ctx, string(sourceID), &p); err != nil {
//		if err == store.ErrNoResult {
//			return consts.ErrUnlinkedAccount
//		}
//		return consts.ErrInternal
//	}
//	*target = action.Author(p.SteamID.String())
//	return nil
//}
