package service

import (
	"context"
	"fmt"
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
	Listen(addr)
}

func Ban(sidStr string, author string, duration time.Duration, ip net.IP, reason model.Reason, reasonText string) error {
	sid := steamid.StringToSID64(sidStr)
	if !sid.Valid() {
		return errors.Errorf("Failed to get steam id from; %s", sidStr)
	}
	aid := steamid.StringToSID64(author)
	if !aid.Valid() {
		return errors.Errorf("Failed to get steam id from; %s", sidStr)
	}
	ban := model.Ban{
		SteamID:    sid,
		AuthorID:   aid,
		Reason:     reason,
		ReasonText: reasonText,
		IP:         ip.String(),
		Note:       "",
		CreatedOn:  time.Now().Unix(),
		UpdatedOn:  time.Now().Unix(),
	}
	if err := store.SaveBan(&ban); err != nil {
		return errors.Wrapf(err, "failed to save ban to database: %v", err)
	}
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
