package service

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/pkg/errors"
	"github.com/rumblefrog/go-a2s"
	log "github.com/sirupsen/logrus"
	"strings"
	"sync"
	"time"
)

func a2sQuery(server model.Server) (*a2s.ServerInfo, error) {
	client, err := a2s.NewClient(server.Addr())
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create a2s client")
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.WithFields(log.Fields{"server": server.ServerName}).Errorf("Failed to close a2s client: %v", err)
		}
	}()
	info, err := client.QueryInfo() // QueryInfo, QueryPlayer, QueryRules
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to query server info")
	}
	return info, nil
}

func queryRCON(ctx context.Context, servers []model.Server, commands ...string) map[string]string {
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
				log.WithFields(log.Fields{"server": server.ServerName}).Debugf("RCON: %s", command)
				resp, err := conn.Exec(sanitizeRCONCommand(command))
				if err != nil {
					log.Errorf("Failed to exec rcon command %s: %v", server.ServerName, err)
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

// sanitizeRCONCommand is a very basic check for injection of additional commands
// using `;` as a command separator. This will just return the first part of the command
func sanitizeRCONCommand(s string) string {
	p := strings.SplitN(s, ";", 1)
	return p[0]
}

func execServerRCON(server model.Server, cmd string) (string, error) {
	r, err := rcon.Dial(context.Background(), server.Addr(), server.RCON, time.Second*10)
	if err != nil {
		return "", errors.Errorf("Failed to dial server: %s", server.ServerName)
	}
	resp, err2 := r.Exec(sanitizeRCONCommand(cmd))
	if err2 != nil {
		return "", errors.Errorf("Failed to exec command: %v", err)
	}
	return resp, nil
}

func queryA2SInfo(servers []model.Server) map[string]*a2s.ServerInfo {
	responses := make(map[string]*a2s.ServerInfo)
	mu := &sync.RWMutex{}
	wg := &sync.WaitGroup{}
	for _, s := range servers {
		wg.Add(1)
		go func(server model.Server) {
			defer wg.Done()
			resp, err := a2sQuery(server)
			if err != nil {
				log.Errorf("A2S: %v", err)
				return
			}
			mu.Lock()
			responses[server.ServerName] = resp
			mu.Unlock()
		}(s)
	}
	wg.Wait()
	return responses
}
