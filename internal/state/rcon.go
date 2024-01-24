package state

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

var ErrCommandFailed = errors.New("rcon command failed")

type Executor interface {
	ExecServer(ctx context.Context, serverID int, cmd string) (string, error)
	ExecRaw(ctx context.Context, addr string, password string, cmd string) (string, error)
	OnFindExec(ctx context.Context, name string, steamID steamid.SID64, ip net.IP, cidr *net.IPNet, onFoundCmd func(info domain.PlayerServerInfo) string) error
}

// Kick will kick the steam id from whatever server it is connected to.
func Kick(ctx context.Context, executor Executor, target steamid.SID64, reason domain.Reason) error {
	if !target.Valid() {
		return errs.ErrInvalidTargetSID
	}

	if errExec := executor.OnFindExec(ctx, "", target, nil, nil, func(info domain.PlayerServerInfo) string {
		return fmt.Sprintf("sm_kick #%d %s", info.Player.UserID, reason.String())
	}); errExec != nil {
		return errors.Join(errExec, ErrCommandFailed)
	}

	return nil
}

// Silence will gag & mute a player.
func Silence(ctx context.Context, executor Executor, target steamid.SID64, reason domain.Reason,
) error {
	if !target.Valid() {
		return errs.ErrInvalidTargetSID
	}

	var (
		users   []string
		usersMu = &sync.RWMutex{}
	)

	if errExec := executor.OnFindExec(ctx, "", target, nil, nil, func(info domain.PlayerServerInfo) string {
		usersMu.Lock()
		users = append(users, info.Player.Name)
		usersMu.Unlock()

		return fmt.Sprintf(`sm_silence "#%s" %s`, steamid.SID64ToSID(info.Player.SID), reason.String())
	}); errExec != nil {
		return errors.Join(errExec, fmt.Errorf("%w: sm_silence", ErrCommandFailed))
	}

	return nil
}

// Say is used to send a message to the server via sm_say.
func Say(ctx context.Context, executor Executor, serverID int, message string) error {
	_, errExec := executor.ExecServer(ctx, serverID, fmt.Sprintf(`sm_say %s`, message))

	return errors.Join(errExec, fmt.Errorf("%w: sm_say", ErrCommandFailed))
}

// CSay is used to send a centered message to the server via sm_csay.
func CSay(ctx context.Context, executor Executor, serverID int, message string) error {
	_, errExec := executor.ExecServer(ctx, serverID, fmt.Sprintf(`sm_csay %s`, message))

	return errors.Join(errExec, fmt.Errorf("%w: sm_csay", ErrCommandFailed))
}

// PSay is used to send a private message to a player.
func PSay(ctx context.Context, executor Executor, target steamid.SID64, message string) error {
	if !target.Valid() {
		return errs.ErrInvalidTargetSID
	}

	if errExec := executor.OnFindExec(ctx, "", target, nil, nil, func(info domain.PlayerServerInfo) string {
		return fmt.Sprintf(`sm_psay "#%s" "%s"`, steamid.SID64ToSID(target), message)
	}); errExec != nil {
		return errors.Join(errExec, fmt.Errorf("%w: sm_psay", ErrCommandFailed))
	}

	return nil
}
