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
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// BuildVersion holds the current git revision, as of build time
	BuildVersion = "master"
	gCtx         context.Context
	// Holds ephemeral user warning state for things such as word filters
	warnings   map[steamid.SID64][]userWarning
	warningsMu *sync.RWMutex
	// When a server posts log entries they are sent through here
	logRawQueue chan web.LogPayload
)

type warnReason int

const (
	warnLanguage warnReason = iota
)

type userWarning struct {
	WarnReason warnReason
	CreatedOn  time.Time
}

// shutdown cleans up the application and closes connections
func shutdown() {
	store.Close()
}

// Start is the main application entry point
func Start() {
	actChan := make(chan *action.Action)
	action.Register(actChan)
	go actionExecutor(gCtx, actChan)
	// Load in the external network block / ip ban lists to memory if enabled
	if config.Net.Enabled {
		initNetBans()
	} else {
		log.Warnf("External Network ban lists not enabled")
	}

	// Setup the storage backend
	initStore()
	defer shutdown()

	// Start the discord service
	if config.Discord.Enabled {
		initDiscord()
	} else {
		log.Warnf("Discord bot not enabled")
	}

	// Start the background goroutine workers
	initWorkers(gCtx)

	// Load the filtered word set into memory
	if config.Filter.Enabled {
		initFilters()
	}

	if config.General.Mode == config.Debug {
		go func() {
			// TODO remove
			f := golib.FindFile("test_data/log_sup_med_1.log", "gbans")
			b, _ := ioutil.ReadFile(f)
			rows := strings.Split(string(b), "\r\n")
			t := time.NewTicker(time.Millisecond * 200)
			servers, _ := store.GetServers(context.Background())
			serverId := "lo-1"
			if len(servers) > 0 {
				serverId = servers[0].ServerName
			}
			i := 0
			for {
				<-t.C
				logRawQueue <- web.LogPayload{
					ServerName: serverId,
					Message:    rows[i],
				}
				i++
				if i == len(rows) {
					i = 0
				}
			}
		}()
	}

	// Start the HTTP server
	web.Start(gCtx, logRawQueue)
}

type actionFn = func(context.Context, *action.Action)

// actionExecutor is the action message request handler for any actions that are requested
//
// Each request is executed under its own goroutine concurrently. There should be no expectations
// of results being completed in sequential order unless
func actionExecutor(ctx context.Context, actChan chan *action.Action) {
	var actionMap = map[action.Type]actionFn{
		action.Mute:                  onActionMute,
		action.Kick:                  onActionKick,
		action.Ban:                   onActionBan,
		action.Unban:                 onActionUnban,
		action.BanNet:                onActionBanNet,
		action.Find:                  onActionFind,
		action.CheckFilter:           onActionCheckFilter,
		action.AddFilter:             onActionAddFilter,
		action.DelFilter:             onActionDelFilter,
		action.GetPersonByID:         onActionGetPersonByID,
		action.GetOrCreatePersonByID: onActionGetOrCreatePersonByID,
		action.SetSteamID:            onActionSetSteamID,
		action.Say:                   onActionSay,
		action.CSay:                  onActionCSay,
		action.PSay:                  onActionPSay,
		action.FindByCIDR:            onActionFindByCIDR,
		action.GetBan:                onActionGetBan,
		action.GetBanNet:             onActionGetBanNet,
		action.GetHistoryIP:          onActionGetHistoryIP,
		action.GetHistoryChat:        onActionGetHistoryChat,
		action.GetASNRecord:          onActionGetASNRecord,
		action.GetLocationRecord:     onActionGetLocationRecord,
		action.GetProxyRecord:        onActionGetProxyRecord,
		action.Servers:               onActionServers,
		action.ServerByName:          onActionServerByName,
	}
	for {
		select {
		case <-ctx.Done():
			return
		case act := <-actChan:
			fn, found := actionMap[act.Type]
			if found {
				go fn(ctx, act)
			}
		}
	}
}

// warnWorker will periodically flush out warning older than `config.General.WarningTimeout`
func warnWorker(ctx context.Context) {
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

// logWriter handles tak
func logWriter(ctx context.Context) {
	const (
		freq = time.Second * 10
	)
	var logCache []model.ServerEvent
	events := make(chan model.ServerEvent, 100)
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
			if errI := store.BatchInsertServerLogs(ctx, logCache); errI != nil {
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
func logReader(ctx context.Context, logRows chan web.LogPayload) {
	getPlayer := func(id string, v map[string]string) *model.Person {
		sid1Str, ok := v[id]
		if ok {
			p, err := store.GetOrCreatePersonBySteamID(ctx, steamid.SID3ToSID64(steamid.SID3(sid1Str)))
			if err != nil {
				log.Errorf("Failed to load player1 %s: %s", sid1Str, err.Error())
				return nil
			}
			return p
		}
		return nil
	}
	for {
		select {
		case raw := <-logRows:
			v := logparse.Parse(raw.Message)
			s, e := store.GetServerByName(ctx, raw.ServerName)
			if e != nil {
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
				var aspv logparse.Pos
				if err := logparse.NewPosFromString(asValue, &aspv); err != nil {
					log.Warnf("Failed to parse assister position: %v", err)
				}
				aspos = aspv
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
				damageP, err := strconv.ParseInt(dmgValue, 10, 64)
				if err != nil {
					log.Warnf("failed to parse damage value: %v", err)
				}
				damage = int(damageP)
			}
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
				Source:      getPlayer("sid", v.Values),
				Target:      getPlayer("sid2", v.Values),
				PlayerClass: class,
				Weapon:      weapon,
				Damage:      damage,
				AttackerPOS: apos,
				VictimPOS:   vpos,
				AssisterPOS: aspos,
				CreatedOn:   config.Now(),
			}

			event.Emit(se)
		case <-ctx.Done():
			log.Debugf("logReader shutting down")
			return
		}
	}
}

// addWarning records a user warning into memory. This is not persistent, so application
// restarts will wipe the users history.
//
// Warning are flushed once they reach N age as defined by `config.General.WarningTimeout
func addWarning(sid64 steamid.SID64, reason warnReason) {
	warningsMu.Lock()
	defer warningsMu.Unlock()
	const msg = "Warning limit exceeded"
	_, found := warnings[sid64]
	if !found {
		warnings[sid64] = []userWarning{}
	}
	warnings[sid64] = append(warnings[sid64], userWarning{
		WarnReason: reason,
		CreatedOn:  config.Now(),
	})
	if len(warnings[sid64]) >= config.General.WarningLimit {
		var act action.Action
		switch config.General.WarningExceededAction {
		case config.Gag:
			act = action.NewMute(action.Core, sid64.String(), config.General.Owner.String(), msg,
				config.General.WarningExceededDuration.String())
		case config.Ban:
			act = action.NewBan(action.Core, sid64.String(), config.General.Owner.String(), msg,
				config.General.WarningExceededDuration.String())
		case config.Kick:
			act = action.NewKick(action.Core, sid64.String(), config.General.Owner.String(), msg)
		}
		res := <-act.Enqueue().Done()
		if res.Err != nil {
			log.Errorf("Failed to ban Player after too many warnings: %v", res.Err)
		}
	}
}

func init() {
	warningsMu = &sync.RWMutex{}
	warnings = make(map[steamid.SID64][]userWarning)
	// Global background context. This is passed into the functions that use it as a parameter.
	// This should not be implicitly referenced anywhere to help testing
	gCtx = context.Background()
	logRawQueue = make(chan web.LogPayload, 50)
}

func initFilters() {
	// TODO load external lists via http
	c, cancel := context.WithTimeout(gCtx, time.Second*15)
	defer cancel()
	words, err := store.GetFilters(c)
	if err != nil {
		log.Fatal("Failed to load word list")
	}
	importFilteredWords(words)
	log.Debugf("Loaded %d filtered words", len(words))
}

func initStore() {
	store.Init(config.DB.DSN)
}

func initWorkers(ctx context.Context) {
	go banSweeper(ctx)
	go serverStateUpdater(ctx)
	go profileUpdater(ctx)
	go warnWorker(ctx)
	go logReader(ctx, logRawQueue)
	go logWriter(ctx)
	go filterWorker(ctx)
	//go state.LogMeter(ctx)
}

func initDiscord() {
	if config.Discord.Token != "" {
		events := make(chan model.ServerEvent)
		if err := event.RegisterConsumer(events, []logparse.MsgType{logparse.Say, logparse.SayTeam}); err != nil {
			log.Warnf("Error registering discord log event reader")
		}
		go discord.Start(gCtx, config.Discord.Token, events)
	} else {
		log.Fatalf("Discord enabled, but bot token invalid")
	}
}

func initNetBans() {
	for _, list := range config.Net.Sources {
		if err := external.Import(list); err != nil {
			log.Errorf("Failed to import list: %v", err)
		}
	}
}
