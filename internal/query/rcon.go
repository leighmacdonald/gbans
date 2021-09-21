package query

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strings"
	"sync"
	"time"
)

func ExecRCON(server model.Server, cmd string) (string, error) {
	r, err := rcon.Dial(context.Background(), server.Addr(), server.RCON, time.Second*5)
	if err != nil {
		return "", errors.Errorf("Failed to dial server: %s (%v)", server.ServerName, err)
	}
	resp, err2 := r.Exec(sanitizeRCONCommand(cmd))
	if err2 != nil {
		return "", errors.Errorf("Failed to exec command: %v", err2)
	}
	return resp, nil
}

func RCON(ctx context.Context, servers []model.Server, commands ...string) map[string]string {
	responses := make(map[string]string)
	mu := &sync.RWMutex{}
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
				resp, errR := conn.Exec(sanitizeRCONCommand(command))
				if errR != nil {
					log.Debugf("Failed to exec rcon command %s: %v", server.ServerName, errR)
				}
				mu.Lock()
				responses[server.ServerName] = resp
				mu.Unlock()
			}
		}(s)
	}
	wg.Wait()
	return responses
}

func GetServerStatus(server model.Server) (extra.Status, error) {
	resp, err := ExecRCON(server, "status")
	if err != nil {
		log.Debugf("Failed to exec rcon command: %v", err)
		return extra.Status{}, err
	}
	status, err2 := extra.ParseStatus(resp, true)
	if err2 != nil {
		log.Errorf("Failed to parse status output: %v", err2)
		return extra.Status{}, err2
	}
	return status, nil
}

// sanitizeRCONCommand is a very basic check for injection of additional commands
// using `;` as a command separator. This will just return the first part of the command
func sanitizeRCONCommand(s string) string {
	p := strings.SplitN(s, ";", 1)
	return p[0]
}
