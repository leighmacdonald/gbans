package query

import (
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/pkg/errors"
	"github.com/rumblefrog/go-a2s"
	log "github.com/sirupsen/logrus"
)

func A2SQueryServer(server model.Server) (*a2s.ServerInfo, error) {
	client, errClient := a2s.NewClient(server.Addr())
	if errClient != nil {
		return nil, errors.Wrapf(errClient, "Failed to create a2s client")
	}
	defer func() {
		if errClose := client.Close(); errClose != nil {
			log.WithFields(log.Fields{"server": server.ServerName}).Errorf("Failed to close a2s client: %v", errClose)
		}
	}()
	info, errQuery := client.QueryInfo() // QueryInfo, QueryPlayer, QueryRules
	if errQuery != nil {
		return nil, errors.Wrapf(errQuery, "Failed to query server info")
	}
	return info, nil
}
