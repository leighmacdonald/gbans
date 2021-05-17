package query

import (
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/pkg/errors"
	"github.com/rumblefrog/go-a2s"
	log "github.com/sirupsen/logrus"
	"sync"
)

func A2SInfo(servers []model.Server) map[string]*a2s.ServerInfo {
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

func a2sQuery(server model.Server) (*a2s.ServerInfo, error) {
	client, err := a2s.NewClient(server.Addr())
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create a2s client")
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.WithFields(log.Fields{"Server": server.ServerName}).Errorf("Failed to close a2s client: %v", err)
		}
	}()
	info, err := client.QueryInfo() // QueryInfo, QueryPlayer, QueryRules
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to query Server info")
	}
	return info, nil
}
