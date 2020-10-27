package service

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/bot"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/external"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/gbans/store"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"html/template"
	"net"
	"sync"
	"time"
)

var (
	BuildVersion = "master"
	router       *gin.Engine
	templates    map[string]*template.Template
	routes       map[Route]string
	ctx          context.Context
)

func init() {
	templates = make(map[string]*template.Template)
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
	// Setup the HTTP router
	initRouter()
	// Setup the storage backend
	initStore()
	// Start the discord service
	if config.Discord.Enabled {
		initDiscord()
	} else {
		log.Warnf("Discord bot not enabled")
	}
	// Start the background goroutine workers
	initWorkers()

	// Start the HTTP server
	initHTTP()
}
func initStore() {
	store.Init(config.DB.Path)
}
func initWorkers() {
	go banSweeper()
}

func initDiscord() {
	if config.Discord.Token != "" {
		go bot.Start(ctx, config.Discord.Token, config.Discord.ModChannels)
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

func banSweeper() {
	log.Debug("Ban sweeper routine started")
	ticker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-ticker.C:
			bans, err := store.GetExpiredBans()
			if err != nil {
				log.Warnf("Failed to get expired bans")
			} else {
				for _, ban := range bans {
					if err := store.DropBan(ban); err != nil {
						log.Errorf("Failed to drop expired ban: %v", err)
					} else {
						log.Infof("Ban expired: %v", ban)
					}
				}
			}
			netBans, err := store.GetExpiredNetBans()
			if err != nil {
				log.Warnf("Failed to get expired bans")
			} else {
				for _, ban := range netBans {
					if err := store.DropNetBan(ban); err != nil {
						log.Errorf("Failed to drop expired network ban: %v", err)
					} else {
						log.Infof("Network ban expired: %v", ban)
					}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func Ban(ctx context.Context, sidStr string, author string, duration time.Duration, ip net.IP,
	banType model.BanType, reason model.Reason, reasonText string, source model.BanSource) error {
	sid, err := steamid.StringToSID64(sidStr)
	if err != nil || !sid.Valid() {
		return errors.Errorf("Failed to get steam id from; %s", sidStr)
	}
	aid, err := steamid.StringToSID64(author)
	if err != nil || !aid.Valid() {
		return errors.Errorf("Failed to get steam id from; %s", sidStr)
	}
	var until int64
	if duration.Seconds() != 0 {
		until = time.Now().Add(duration).Unix()
	}
	ban := model.Ban{
		SteamID:    sid,
		AuthorID:   aid,
		BanType:    banType,
		Reason:     reason,
		ReasonText: reasonText,
		Note:       "naughty",
		Until:      until,
		Source:     source,
		CreatedOn:  time.Now().Unix(),
		UpdatedOn:  time.Now().Unix(),
	}
	if err := store.SaveBan(&ban); err != nil {
		return store.DBErr(err)
	}
	servers, err := store.GetServers()
	if err != nil {
		log.Errorf("Failed to get server for ban propagation")
	}
	ExecRCON(ctx, servers, "gb_kick ")
	return nil
}

func ExecRCON(ctx context.Context, servers []model.Server, commands ...string) {
	timeout := time.Second * 10
	wg := &sync.WaitGroup{}
	for _, s := range servers {
		wg.Add(1)
		go func(server model.Server) {
			defer wg.Done()
			lCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			addr := fmt.Sprintf("%s:%d", server.Address, server.Port)
			conn, err := rcon.Dial(lCtx, addr, server.RCON, timeout)
			if err != nil {
				log.Errorf("Failed to connect to server %s: %v", server.ServerName, err)
				return
			}
			for _, command := range commands {
				resp, err := conn.Exec(command)
				if err != nil {
					log.Errorf("Failed to exec rcon command %s: %v", server.ServerName, err)
				}
				log.Debugf("rcon %s: %s", server.ServerName, resp)
			}
		}(s)
	}
	wg.Wait()
}
