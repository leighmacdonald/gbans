// Package app is the main application and entry point. It implements the action.Executor and io.Closer interfaces.
package app

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/action"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/event"
	"github.com/leighmacdonald/gbans/internal/external"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/web"
	"github.com/leighmacdonald/gbans/internal/web/ws"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strconv"
	"sync"
	"time"
)

var (
	// BuildVersion holds the current git revision, as of build time
	BuildVersion = "master"
)

// gbans is the main application struct.
// It implements the action.Executor interface
type gbans struct {
	// Top-level background context
	ctx context.Context
	// Holds ephemeral user warning state for things such as word filters
	warnings   map[steamid.SID64][]userWarning
	warningsMu *sync.RWMutex
	// When a server posts log entries they are sent through here
	logRawQueue    chan ws.LogPayload
	bot            discord.ChatBot
	db             store.Store
	web            web.WebHandler
	serversState   map[string]model.ServerState
	serversStateMu *sync.RWMutex
	l              *log.Entry
}

// New instantiates a new application
func New(ctx context.Context) (*gbans, error) {
	application := &gbans{
		ctx:            ctx,
		warnings:       map[steamid.SID64][]userWarning{},
		warningsMu:     &sync.RWMutex{},
		serversStateMu: &sync.RWMutex{},
		logRawQueue:    make(chan ws.LogPayload, 50),
		l:              log.WithFields(log.Fields{"module": "app"}),
	}
	dbStore, se := store.New(config.DB.DSN)
	if se != nil {
		return nil, errors.Wrapf(se, "Failed to setup store")
	}
	discordBot, be := discord.New(application, dbStore)
	if be != nil {
		return nil, errors.Wrapf(be, "Failed to setup bot")
	}
	webService, we := web.New(application.logRawQueue, dbStore, discordBot, application)
	if we != nil {
		return nil, errors.Wrapf(we, "Failed to setup web")
	}

	application.db = dbStore
	application.bot = discordBot
	application.web = webService

	return application, nil
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
func (g *gbans) Start() {
	// Load in the external network block / ip ban lists to memory if enabled
	if config.Net.Enabled {
		initNetBans()
	} else {
		g.l.Warnf("External Network ban lists not enabled")
	}

	defer func() {
		if errC := g.Close(); errC != nil {
			g.l.Errorf("Returned closing error: %v", errC)
		}
	}()

	// Start the discord service
	if config.Discord.Enabled {
		g.initDiscord()
	} else {
		g.l.Warnf("Discord bot not enabled")
	}

	// Start the background goroutine workers
	g.initWorkers()

	// Load the filtered word set into memory
	if config.Filter.Enabled {
		g.initFilters()
	}

	// Start the HTTP server
	if err := g.web.ListenAndServe(); err != nil {
		g.l.Errorf("Error shutting down service: %v", err)
	}
}

// Close cleans up the application and closes connections
func (g *gbans) Close() error {
	return g.db.Close()
}

// warnWorker will periodically flush out warning older than `config.General.WarningTimeout`
func (g *gbans) warnWorker() {
	t := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-t.C:
			now := config.Now()
			g.warningsMu.Lock()
			for k := range g.warnings {
				for i, w := range g.warnings[k] {
					if now.Sub(w.CreatedOn) > config.General.WarningTimeout {
						if len(g.warnings[k]) > 1 {
							g.warnings[k] = append(g.warnings[k][:i], g.warnings[k][i+1])
						} else {
							g.warnings[k] = nil
						}
					}
					if len(g.warnings[k]) == 0 {
						delete(g.warnings, k)
					}
				}
			}
			g.warningsMu.Unlock()
		case <-g.ctx.Done():
			g.l.Debugf("warnWorker shutting down")
			return
		}
	}
}

// logWriter handles tak
func (g *gbans) logWriter() {
	const (
		freq = time.Second * 10
	)
	var logCache []model.ServerEvent
	events := make(chan model.ServerEvent, 100)
	if err := event.RegisterConsumer(events, []logparse.MsgType{logparse.Any}); err != nil {
		g.l.Warnf("logWriter Tried to register duplicate reader channel")
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
			if errI := g.db.BatchInsertServerLogs(g.ctx, logCache); errI != nil {
				g.l.Errorf("Failed to batch insert logs: %v", errI)
			}
			logCache = nil
		case <-g.ctx.Done():
			g.l.Debugf("logWriter shuttings down")
			return
		}
	}
}

// logReader is the fan-out orchestrator for game log events
// Registering receivers can be accomplished with RegisterLogEventReader
func (g *gbans) logReader() {
	getPlayer := func(id string, v map[string]string) *model.Person {
		sid1Str, ok := v[id]
		if ok {
			s := steamid.SID3ToSID64(steamid.SID3(sid1Str))
			p := model.NewPerson(s)
			if err := g.db.GetOrCreatePersonBySteamID(g.ctx, s, &p); err != nil {
				g.l.Errorf("Failed to load player1 %s: %s", sid1Str, err.Error())
				return nil
			}
			return &p
		}
		return nil
	}
	for {
		select {
		case raw := <-g.logRawQueue:
			v := logparse.Parse(raw.Message)
			var s model.Server
			if e := g.db.GetServerByName(g.ctx, raw.ServerName, &s); e != nil {
				g.l.Errorf("Failed to get server for log message: %v", e)
				continue
			}
			var (
				apos, vpos, aspos logparse.Pos
			)
			aposValue, aposFound := v.Values["attacker_position"]
			if aposFound {
				var apv logparse.Pos
				if err := logparse.NewPosFromString(aposValue, &apv); err != nil {
					g.l.Warnf("Failed to parse attacker position: %v", err)
				}
				apos = apv
			}
			vposValue, vposFound := v.Values["victim_position"]
			if vposFound {
				var vpv logparse.Pos
				if err := logparse.NewPosFromString(vposValue, &vpv); err != nil {
					g.l.Warnf("Failed to parse victim position: %v", err)
				}
				vpos = vpv
			}
			asValue, asFound := v.Values["assister_position"]
			if asFound {
				var asPosValue logparse.Pos
				if err := logparse.NewPosFromString(asValue, &asPosValue); err != nil {
					g.l.Warnf("Failed to parse assister position: %v", err)
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
			var damage int
			dmgValue, dmgFound := v.Values["damage"]
			if dmgFound {
				damageP, err := strconv.ParseInt(dmgValue, 10, 32)
				if err != nil {
					g.l.Warnf("failed to parse damage value: %v", err)
				}
				damage = int(damageP)
			}
			source := getPlayer("sid", v.Values)
			target := getPlayer("sid2", v.Values)
			for _, k := range []string{
				"", "pid", "pid2", "sid", "sid2", "team", "team2", "name", "name2",
				"date", "time", "weapon", "damage", "class",
				"attacker_position", "victim_position", "assister_position",
			} {
				delete(v.Values, k)
			}
			se := model.ServerEvent{
				Server:      &s,
				EventType:   v.MsgType,
				Source:      source,
				Target:      target,
				PlayerClass: class,
				Weapon:      weapon,
				Damage:      damage,
				AttackerPOS: apos,
				VictimPOS:   vpos,
				AssisterPOS: aspos,
				CreatedOn:   config.Now(),
			}

			event.Emit(se)
		case <-g.ctx.Done():
			g.l.Debugf("logReader shutting down")
			return
		}
	}
}

// addWarning records a user warning into memory. This is not persistent, so application
// restarts will wipe the user's history.
//
// Warning are flushed once they reach N age as defined by `config.General.WarningTimeout
func (g *gbans) addWarning(sid64 steamid.SID64, reason warnReason) {
	g.warningsMu.Lock()
	_, found := g.warnings[sid64]
	if !found {
		g.warnings[sid64] = []userWarning{}
	}
	g.warnings[sid64] = append(g.warnings[sid64], userWarning{
		WarnReason: reason,
		CreatedOn:  config.Now(),
	})
	g.warningsMu.Unlock()
	if len(g.warnings[sid64]) >= config.General.WarningLimit {
		var pi model.PlayerInfo
		g.l.Errorf("Warn limit exceeded (%d): %d", sid64, len(g.warnings[sid64]))
		var err error
		switch config.General.WarningExceededAction {
		case config.Gag:
			err = g.Mute(action.NewMute(model.System, sid64.String(), config.General.Owner.String(), warnReasonString(reason),
				config.General.WarningExceededDuration.String()), &pi)
		case config.Ban:
			var ban model.Ban
			err = g.Ban(action.NewBan(model.System, sid64.String(), config.General.Owner.String(), warnReasonString(reason),
				config.General.WarningExceededDuration.String()), &ban)
		case config.Kick:
			err = g.Kick(action.NewKick(model.System, sid64.String(), config.General.Owner.String(), warnReasonString(reason)), &pi)
		}
		if err != nil {
			log.WithFields(log.Fields{"action": config.General.WarningExceededAction}).Errorf("Failed to apply warning action: %v", err)
		}
	}
}

func (g *gbans) initFilters() {
	// TODO load external lists via http
	c, cancel := context.WithTimeout(g.ctx, time.Second*15)
	defer cancel()
	words, err := g.db.GetFilters(c)
	if err != nil {
		g.l.Fatal("Failed to load word list")
	}
	importFilteredWords(words)
	g.l.Debugf("Loaded %d filtered words", len(words))
}

func (g *gbans) initWorkers() {
	go g.banSweeper()
	go g.mapChanger(time.Second * 5)
	go g.serverStateUpdater()
	go g.profileUpdater()
	go g.warnWorker()
	go g.logReader()
	go g.logWriter()
	go g.filterWorker()
	//go state.LogMeter(ctx)
}

func (g *gbans) initDiscord() {
	if config.Discord.Token != "" {
		events := make(chan model.ServerEvent)
		if err := event.RegisterConsumer(events, []logparse.MsgType{logparse.Say, logparse.SayTeam}); err != nil {
			g.l.Warnf("Error registering discord log event reader")
		}
		go func() {
			if errBS := g.bot.Start(g.ctx, config.Discord.Token, events); errBS != nil {
				g.l.Errorf("DiscordClient returned error: %v", errBS)
			}
		}()
	} else {
		g.l.Fatalf("Discord enabled, but bot token invalid")
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
