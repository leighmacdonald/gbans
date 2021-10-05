package query

import (
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/pkg/errors"
	"github.com/rumblefrog/go-a2s"
	log "github.com/sirupsen/logrus"
)

func A2SQueryServer(server model.Server) (*a2s.ServerInfo, error) {
	client, err := a2s.NewClient(server.Addr())
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create a2s client")
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.WithFields(log.Fields{"server": server.ServerName}).Errorf("Failed to close a2s client: %v", err)
		}
	}()
	info, err2 := client.QueryInfo() // QueryInfo, QueryPlayer, QueryRules
	if err2 != nil {
		return nil, errors.Wrapf(err2, "Failed to query server info")
	}
	return info, nil
}
