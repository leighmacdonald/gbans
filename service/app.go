package service

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/bot"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/gbans/store"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

func Start(database string, addr string) {
	store.Init(database)
	servers := []model.Server{
		{
			ServerName:     "default",
			Address:        "172.16.1.22",
			Port:           27015,
			RCON:           "testpass",
			Password:       "test_auth",
			TokenCreatedOn: time.Now().Unix(),
		},
	}
	for _, s := range servers {
		if err := store.SaveServer(&s); err != nil && err != model.ErrDuplicate {
			log.Errorf("Failed to add default server: %v", err)
		}
	}
	dur, _ := time.ParseDuration("0s")
	if err := Ban(context.Background(), "STEAM_0:0:431710372", "STEAM_0:1:61934148", dur,
		net.ParseIP("172.16.1.22"), model.Banned, model.Racism, "bad words!"); err != nil && err != model.ErrDuplicate {
		log.Errorf("Failed to add test ban: %v", err)
	}
	startHTTP(addr)
	if config.Discord.Enabled {
		if config.Discord.Token != "" {
			bot.Start(config.Discord.Token)
		} else {
			log.Fatalf("Discord enabled, but bot token invalid")
		}
	}
}

func Ban(ctx context.Context, sidStr string, author string, duration time.Duration, ip net.IP,
	banType model.BanType, reason model.Reason, reasonText string) error {
	sid := steamid.StringToSID64(sidStr)
	if !sid.Valid() {
		return errors.Errorf("Failed to get steam id from; %s", sidStr)
	}
	aid := steamid.StringToSID64(author)
	if !aid.Valid() {
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
		IP:         ip.String(),
		Note:       "naughty",
		Until:      until,
		CreatedOn:  time.Now().Unix(),
		UpdatedOn:  time.Now().Unix(),
	}
	if err := store.SaveBan(&ban); err != nil {
		return store.DBErr(err)
	}
	servers, err := store.GetServers()
	if err != nil {
		log.Errorf("Failed to get server for ban propogation")
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
