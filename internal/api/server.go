package api

// ServerStore API

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

// IsOnIPWithBan checks if the address matches an existing user who is currently banned already. This
// function will always fail-open and allow players in if an error occurs.
func IsOnIPWithBan(ctx context.Context, env Env, steamID steamid.SID64, address net.IP) bool {
	existing := domain.NewBannedPerson()
	if errMatch := env.Store().GetBanByLastIP(ctx, address, &existing, false); errMatch != nil {
		if errors.Is(errMatch, errs.ErrNoResult) {
			return false
		}

		env.Log().Error("Could not load player by ip", zap.Error(errMatch))

		return false
	}

	duration, errDuration := util.ParseUserStringDuration("10y")
	if errDuration != nil {
		env.Log().Error("Could not parse ban duration", zap.Error(errDuration))

		return false
	}

	existing.BanSteam.ValidUntil = time.Now().Add(duration)

	if errSave := env.Store().SaveBan(ctx, &existing.BanSteam); errSave != nil {
		env.Log().Error("Could not update previous ban.", zap.Error(errSave))

		return false
	}

	var newBan domain.BanSteam
	if errNewBan := domain.NewBanSteam(ctx,
		domain.StringSID(env.Config().General.Owner.String()),
		domain.StringSID(steamID.String()), duration, domain.Evading, domain.Evading.String(),
		"Connecting from same IP as banned player", domain.System,
		0, domain.Banned, false, &newBan); errNewBan != nil {
		env.Log().Error("Could not create evade ban", zap.Error(errDuration))

		return false
	}

	if errSave := env.BanSteam(ctx, &newBan); errSave != nil {
		env.Log().Error("Could not save evade ban", zap.Error(errSave))

		return false
	}

	return true
}
