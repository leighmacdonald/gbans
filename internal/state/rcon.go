package state

import (
	"context"
	"errors"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"net"
)

var ErrCommandFailed = errors.New("rcon command failed")

type Executor interface {
	ExecServer(ctx context.Context, serverID int, cmd string) (string, error)
	ExecRaw(ctx context.Context, addr string, password string, cmd string) (string, error)
	OnFindExec(ctx context.Context, name string, steamID steamid.SID64, ip net.IP, cidr *net.IPNet, onFoundCmd func(info domain.PlayerServerInfo) string) error
}
