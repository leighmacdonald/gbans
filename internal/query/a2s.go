package query

import (
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/pkg/errors"
	"github.com/rumblefrog/go-a2s"
	"go.uber.org/zap"
	"time"
)

func A2SQueryServer(logger *zap.Logger, addr string) (*a2s.ServerInfo, error) {
	client, errClient := a2s.NewClient(addr, a2s.TimeoutOption(time.Second*5))
	if errClient != nil {
		return nil, errors.Wrapf(errClient, "Failed to create a2s client")
	}
	defer util.LogCloser(client, logger)
	info, errQuery := client.QueryInfo() // QueryInfo, QueryPlayer, QueryRules
	if errQuery != nil {
		return nil, errors.Wrapf(errQuery, "Failed to query server info")
	}
	return info, nil
}
