// Package app is the main application and entry point. It implements the action.Executor and io.Closer interfaces.
package app

import (
	"context"
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
	logRawQueue    chan LogPayload
	gameLogSource  *remoteSrcdsLogSource
	bot            ChatBot
	db             store.Store
	webHandler     WebHandler
	serversState   map[string]model.ServerState
	serversStateMu *sync.RWMutex
)

func init() {
	rand.Seed(time.Now().Unix())
	serversStateMu = &sync.RWMutex{}
	ctx = context.Background()
	warnings = map[steamid.SID64][]userWarning{}
	warningsMu = &sync.RWMutex{}
	logRawQueue = make(chan LogPayload, 1000)
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

// Start is the main application entry point
func Start() error {
	dbStore, se := store.New(config.DB.DSN)
	if se != nil {
		return errors.Wrapf(se, "Failed to setup store")
	}
	db = dbStore

	discordBot, be := NewDiscord()
	if be != nil {
		return errors.Wrapf(be, "Failed to setup bot")
	}
	bot = discordBot

	logSrc, errLogSrc := newRemoteSrcdsLogSource(config.Log.SrcdsLogAddr, dbStore, logRawQueue)
	if errLogSrc != nil {
		return errors.Wrapf(errLogSrc, "Failed to setup udp log src")
	}
	gameLogSource = logSrc

	webService, we := NewWeb()
	if we != nil {
		return errors.Wrapf(we, "Failed to setup web")
	}
	webHandler = webService

	// Load in the external network block / ip ban lists to memory if enabled
	if config.Net.Enabled {
		initNetBans()
	} else {
		log.Warnf("External Network ban lists not enabled")
	}

	defer func() {
		if errC := Close(); errC != nil {
			log.Errorf("Returned closing error: %v", errC)
		}
	}()

	// Start the discord service
	if config.Discord.Enabled {
		initDiscord()
	} else {
		log.Warnf("discord bot not enabled")
	}

	// Start the background goroutine workers
	initWorkers()

	// Load the filtered word set into memory
	if config.Filter.Enabled {
		initFilters()
	}

	// Start the HTTP server
	if err := webHandler.ListenAndServe(); err != nil {
		return errors.Wrapf(err, "Error shutting down service")
	}
	return nil
}

// Close cleans up the application and closes connections
func Close() error {
	return db.Close()
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
func logWriter() {
	const (
		freq = time.Second * 10
	)
	var logCache []model.ServerEvent
	events := make(chan model.ServerEvent, 1000)
	if err := event.RegisterConsumer(events, []logparse.MsgType{logparse.Any}); err != nil {
		log.Warnf("logWriter Tried to register duplicate reader channel")
	}
	t := time.NewTicker(freq)
	for {
		select {
		case evt := <-events:
			logCache = append(logCache, evt)
		case <-t.C:
			if len(logCache) == 0 {
				continue
			}
			if errI := db.BatchInsertServerLogs(ctx, logCache); errI != nil {
				log.Errorf("Failed to batch insert logs: %v", errI)
			}
			logCache = nil
		case <-ctx.Done():
			log.Debugf("logWriter shuttings down")
			return
		}
	}
}

// logReader is the fan-out orchestrator for game log events
// Registering receivers can be accomplished with RegisterLogEventReader
func logReader() {
	getPlayer := func(id string, v map[string]string) *model.Person {
		sid1Str, ok := v[id]
		if ok {
			s := steamid.SID3ToSID64(steamid.SID3(sid1Str))
			p := model.NewPerson(s)
			if err := db.GetOrCreatePersonBySteamID(ctx, s, &p); err != nil {
				log.Errorf("Failed to load player1 %s: %s", sid1Str, err.Error())
				return nil
			}
			return &p
		}
		return nil
	}
	for {
		select {
		case raw := <-logRawQueue:
			v := logparse.Parse(raw.Message)
			var s model.Server
			if e := db.GetServerByName(ctx, raw.ServerName, &s); e != nil {
				log.Errorf("Failed to get server for log message: %v", e)
				continue
			}
			var (
				apos, vpos, aspos logparse.Pos
			)
			aposValue, aposFound := v.Values["attacker_position"]
			if aposFound {
				var apv logparse.Pos
				if err := logparse.NewPosFromString(aposValue, &apv); err != nil {
					log.Warnf("Failed to parse attacker position: %v", err)
				}
				apos = apv
			}
			vposValue, vposFound := v.Values["victim_position"]
			if vposFound {
				var vpv logparse.Pos
				if err := logparse.NewPosFromString(vposValue, &vpv); err != nil {
					log.Warnf("Failed to parse victim position: %v", err)
				}
				vpos = vpv
			}
			asValue, asFound := v.Values["assister_position"]
			if asFound {
				var asPosValue logparse.Pos
				if err := logparse.NewPosFromString(asValue, &asPosValue); err != nil {
					log.Warnf("Failed to parse assister position: %v", err)
				}
				aspos = asPosValue
			}
			var weapon logparse.Weapon
			weaponValue, weaponFound := v.Values["weapon"]
			if weaponFound {
				weapon = logparse.WeaponFromString(weaponValue)
			}
			var class logparse.PlayerClass
			classValue, classFound := v.Values["class"]
			if classFound {
				if !logparse.ParsePlayerClass(classValue, &class) {
					class = logparse.Spectator
				}
			}
			extra := ""
			extraValue, extraFound := v.Values["msg"]
			if extraFound {
				extra = extraValue
			}
			var damage int
			dmgValue, dmgFound := v.Values["damage"]
			if dmgFound {
				damageP, err := strconv.ParseInt(dmgValue, 10, 32)
				if err != nil {
					log.Warnf("failed to parse damage value: %v", err)
				}
				damage = int(damageP)
			}
			se := model.ServerEvent{
				Server:      &s,
				EventType:   v.MsgType,
				Source:      getPlayer("sid", v.Values),
				Target:      getPlayer("sid2", v.Values),
				PlayerClass: class,
				Weapon:      weapon,
				Damage:      damage,
				AttackerPOS: apos,
				VictimPOS:   vpos,
				AssisterPOS: aspos,
				CreatedOn:   config.Now(),
				Extra:       extra,
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
func addWarning(sid64 steamid.SID64, reason warnReason) {
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
			err = Ban(bo, &ban)
		case config.Ban:
			bo.banType = model.Banned
			err = Ban(bo, &ban)
		case config.Kick:
			var pi model.PlayerInfo
			err = Kick(model.System, model.Target(sid64.String()),
				model.Target(config.General.Owner.String()), warnReasonString(reason), &pi)
		}
		if err != nil {
			log.WithFields(log.Fields{"action": config.General.WarningExceededAction}).
				Errorf("Failed to apply warning action: %v", err)
		}
	}
}

func initFilters() {
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

func initWorkers() {
	go banSweeper()
	go mapChanger(time.Second * 5)
	go serverStateUpdater()
	go profileUpdater()
	go warnWorker()
	go logReader()
	go logWriter()
	go filterWorker()
	go initLogSrc()
	go logMetricsConsumer()
}

func initLogSrc() {
	gameLogSource.start()
}

func initDiscord() {
	if config.Discord.Token != "" {
		events := make(chan model.ServerEvent)
		if len(config.Discord.LogChannelID) > 0 {
			if err := event.RegisterConsumer(events, []logparse.MsgType{logparse.Say, logparse.SayTeam}); err != nil {
				log.Warnf("Error registering discord log event reader")
			}
		}
		go func() {
			if errBS := bot.Start(ctx, config.Discord.Token, events); errBS != nil {
				log.Errorf("discord returned error: %v", errBS)
			}
		}()
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
