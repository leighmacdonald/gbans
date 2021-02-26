package service

import (
	"context"
	"encoding/gob"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/external"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/gbans/util"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/rumblefrog/go-a2s"
	log "github.com/sirupsen/logrus"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	BuildVersion  = "master"
	router        *gin.Engine
	templates     map[string]*template.Template
	routes        map[Route]string
	ctx           context.Context
	serverStateMu *sync.RWMutex
	serverState   map[string]ServerState
	warnings      map[steamid.SID64][]UserWarning
	warningsMu    *sync.RWMutex
	httpServer    *http.Server
)

type WarnReason int

const (
	warnLanguage WarnReason = iota
)

type UserWarning struct {
	WarnReason WarnReason
	CreatedOn  time.Time
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
		}
	}
}

// addWarning records a user warning into memory. This is not persistent, so application
// restarts will wipe the users history.
//
// Warning are flushed once they reach N age as defined by `config.General.WarningTimeout
func addWarning(sid64 steamid.SID64, reason WarnReason) {
	warningsMu.Lock()
	defer warningsMu.Unlock()
	_, found := warnings[sid64]
	if !found {
		warnings[sid64] = []UserWarning{}
	}
	warnings[sid64] = append(warnings[sid64], UserWarning{
		WarnReason: reason,
		CreatedOn:  config.Now(),
	})
	if len(warnings[sid64]) >= config.General.WarningLimit {
		if err := BanPlayer(ctx, sid64, config.General.Owner, 0, model.WarningsExceeded,
			"Warning limit exceeded", model.System); err != nil {
			log.Errorf("Failed to ban player after too many warnings: %s", err)
		}
	}
}

type gameType string

const (
	unknown gameType = "Unknown"
	tf2     gameType = "Team Fortress 2"
	cs      gameType = "Counter-Strike"
	csgo    gameType = "Counter-Strike: Global Offensive"
)

type ServerState struct {
	Addr     string
	Port     int
	Slots    int
	GameType gameType
	A2SInfo  *a2s.ServerInfo
	extra.Status
	// TODO Find better way to track this
	Alive bool
}

func (s ServerState) OS() template.HTML {
	switch s.A2SInfo.ServerOS {
	case a2s.ServerOS_Linux:
		return "linux"
	case a2s.ServerOS_Windows:
		return "windows"
	case a2s.ServerOS_Mac:
		return "mac"
	default:
		return "unknown"
	}
}

func (s ServerState) VacStatus() template.HTML {
	if s.A2SInfo.VAC {
		return "on"
	}
	return "off"
}

func init() {
	gob.Register(Flash{})
	warningsMu = &sync.RWMutex{}
	warnings = make(map[steamid.SID64][]UserWarning)
	templates = make(map[string]*template.Template)
	serverState = make(map[string]ServerState)
	serverStateMu = &sync.RWMutex{}
	ctx = context.Background()
	router = gin.New()

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
	// Load the HTML templated into memory
	initTemplates()

	// Watch for template changes and reload so we dont have to recompile on every edit
	if config.General.Mode == "debug" {
		rx := regexp.MustCompile(`.+?\.gohtml$`)
		go util.WatchDir(ctx, "templates", func(path string) error {
			if !strings.HasPrefix(path, "_") && rx.MatchString(path) {
				log.Debugf("Template modified: %s", path)
				initTemplates()
			}
			return nil
		})
	}

	// Setup the HTTP router
	initRouter()
	// Setup the storage backend
	initStore()
	defer Close()

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
	words, err := GetFilteredWords()
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
	go banSweeper()
	go serverStateUpdater()
	go profileUpdater()
	go warnWorker()
}

func initDiscord() {
	if config.Discord.Token != "" {
		go StartDiscord(ctx, config.Discord.Token, config.Discord.ModChannels)
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

func queryInt(c *gin.Context, key string) int {
	valStr := c.Query(key)
	val, err := strconv.ParseInt(valStr, 10, 32)
	if err != nil {
		log.Panicf("Failed to parse query (Use a validator): \"%v\"", valStr)
	}
	return int(val)
}
