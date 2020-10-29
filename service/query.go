package service

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/rcon/rcon"
	"github.com/pkg/errors"
	"github.com/rumblefrog/go-a2s"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

func A2SQuery(server model.Server) (*a2s.ServerInfo, error) {
	client, err := a2s.NewClient(server.Addr())

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create a2s client")
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Errorf("Failed to close a2s client: %v", err)
		}
	}()
	info, err := client.QueryInfo() // QueryInfo, QueryPlayer, QueryRules
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to query server info")
	}
	return info, nil
}

func QueryRCON(ctx context.Context, servers []model.Server, commands ...string) map[string]string {
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
				resp, err := conn.Exec(command)
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

func QueryA2SInfo(ctx context.Context, servers []model.Server) map[string]*a2s.ServerInfo {
	responses := make(map[string]*a2s.ServerInfo)
	mu := &sync.RWMutex{}
	wg := &sync.WaitGroup{}
	for _, s := range servers {
		wg.Add(1)
		go func(server model.Server) {
			defer wg.Done()
			resp, err := A2SQuery(server)
			if err != nil {
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
