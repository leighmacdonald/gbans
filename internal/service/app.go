package service

import (
	"context"
	"embed"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/external"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/rumblefrog/go-a2s"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"
)

var (
	// BuildVersion holds the current git revision, as of build time
	BuildVersion      = "master"
	router            *gin.Engine
	gCtx              context.Context
	serverStateMu     *sync.RWMutex
	serverStates      map[string]serverState
	warnings          map[steamid.SID64][]userWarning
	warningsMu        *sync.RWMutex
	httpServer        *http.Server
	logRawQueue       chan LogPayload
	logEventReaders   []chan logEvent
	logEventReadersMu *sync.RWMutex
	//go:embed dist
	content embed.FS

	lgr = log.New()
)

type warnReason int

const (
	warnLanguage warnReason = iota
)

type userWarning struct {
	WarnReason warnReason
	CreatedOn  time.Time
}

// registerLogEventReader will register a channel to receive new log events as they come in
func registerLogEventReader(r chan logEvent) error {
	logEventReadersMu.Lock()
	defer logEventReadersMu.Unlock()
	for _, c := range logEventReaders {
		if c == r {
			return errDuplicate
		}
	}
	logEventReaders = append(logEventReaders, r)
	return nil
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

type logEvent struct {
	Type     logparse.MsgType
	Event    map[string]string
	Server   model.Server
	Player1  *model.Person
	Player2  *model.Person
	Assister *model.Person
	RawEvent string
}

func logWriter(ctx context.Context) {
	events := make(chan logEvent)
	if err := registerLogEventReader(events); err != nil {
		log.Warnf("logWriter Tried to register duplicate reader channel")
	}
	for {
		select {
		case evt := <-events:
			if err := insertLog(model.NewServerLog(evt.Server.ServerID, evt.Type, evt.Event)); err != nil {
				log.Errorf("Failed to insert log: %v", err)
				continue
			}
		case <-ctx.Done():
			log.Debugf("logWriter shuttings down")
			return
		}
	}
}

func logReader(ctx context.Context, logRows chan LogPayload, readers ...chan logEvent) {
	for _, reader := range readers {
		if err := registerLogEventReader(reader); err != nil {
			log.Warnf("Tried to register duplicate log event reader")
		}
	}
	getPlayer := func(id string, v map[string]string) *model.Person {
		sid1Str, ok := v[id]
		if ok {
			p, err := GetOrCreatePersonBySteamID(steamid.SID3ToSID64(steamid.SID3(sid1Str)))
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
			s, e := getServerByName(raw.ServerName)
			if e != nil {
				log.Errorf("Failed to get server for log message: %v", e)
				continue
			}
			le := logEvent{
				Type:     v.MsgType,
				Event:    v.Values,
				Server:   s,
				Player1:  getPlayer("sid", v.Values),
				Player2:  getPlayer("sid2", v.Values),
				RawEvent: raw.Message,
			}
			for _, reader := range logEventReaders {
				reader <- le
			}
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
	_, found := warnings[sid64]
	if !found {
		warnings[sid64] = []userWarning{}
	}
	warnings[sid64] = append(warnings[sid64], userWarning{
		WarnReason: reason,
		CreatedOn:  config.Now(),
	})
	if len(warnings[sid64]) >= config.General.WarningLimit {
		if _, err := BanPlayer(gCtx, sid64, config.General.Owner, 0, model.WarningsExceeded,
			"Warning limit exceeded", model.System); err != nil {
			log.Errorf("Failed to ban player after too many warnings: %s", err)
		}
	}
}

type gameType string

const (
	//unknown gameType = "Unknown"
	tf2 gameType = "team Fortress 2"
	//cs      gameType = "Counter-Strike"
	//csgo    gameType = "Counter-Strike: Global Offensive"
)

type serverState struct {
	Addr     string
	Port     int
	Slots    int
	GameType gameType
	A2SInfo  *a2s.ServerInfo
	extra.Status
	// TODO Find better way to track this
	Alive bool
}

func init() {
	warningsMu = &sync.RWMutex{}
	warnings = make(map[steamid.SID64][]userWarning)
	serverStates = make(map[string]serverState)
	serverStateMu = &sync.RWMutex{}
	// Global background context. This is passed into the functions that use it as a parameter.
	// This should not be implicitly referenced anywhere to help testing
	gCtx = context.Background()
	router = gin.New()
	logRawQueue = make(chan LogPayload)
	logEventReadersMu = &sync.RWMutex{}
}

// shutdown cleans up the application and closes connections
func shutdown() {
	db.Close()
}

// Start is the main application entry point
//
func Start() {
	// Load in the external network block / ip ban lists to memory if enabled
	if config.Net.Enabled {
		initNetBans()
	} else {
		log.Warnf("External Network ban lists not enabled")
	}

	// Setup the HTTP router
	initRouter()

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
	initWorkers()

	// Load the filtered word set into memory
	if config.Filter.Enabled {
		initFilters()
	}

	// Start the HTTP server
	initHTTP()

	if config.Relay.Enabled {
		initRelay()
	}
}

func initRelay() {

}

func initFilters() {
	// TODO load external lists via http
	words, err := getFilteredWords()
	if err != nil {
		log.Fatal("Failed to load word list")
	}
	util.ImportFilteredWords(words)
	log.Debugf("Loaded %d filtered words", len(words))
}

func initStore() {
	Init(config.DB.DSN)
}

func initWorkers() {
	go banSweeper(gCtx)
	go serverStateUpdater(gCtx)
	go profileUpdater(gCtx)
	go warnWorker(gCtx)
	go logReader(gCtx, logRawQueue)
	go logWriter(gCtx)
}

func initDiscord() {
	if config.Discord.Token != "" {
		go startDiscord(gCtx, config.Discord.Token)
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
