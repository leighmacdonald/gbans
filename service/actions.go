package service

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

func MutePlayer(ctx context.Context, sid steamid.SID64, author steamid.SID64, duration time.Duration,
	reason model.Reason, reasonText string) error {
	if !sid.Valid() {
		return errors.Errorf("Failed to get steam id from: %s", sid.String())
	}
	if !author.Valid() {
		return errors.Errorf("Failed to get steam id from: %s", author)
	}
	untile := config.DefaultExpiration()
	if duration > 0 {
		untile = config.Now().Add(duration)
	}
	ban := model.Ban{
		SteamID:    sid,
		AuthorID:   author,
		BanType:    model.NoComm,
		Reason:     reason,
		ReasonText: reasonText,
		Source:     0,
		Until:      untile,
		CreatedOn:  config.Now(),
		UpdatedOn:  config.Now(),
	}
	if err := SaveBan(&ban); err != nil {
		return DBErr(err)
	}
	servers, err := GetServers()
	if err != nil {
		log.Errorf("Failed to get server for ban propagation")
	}
	QueryRCON(ctx, servers,
		fmt.Sprintf(`sm_gag "#%s""`, string(steamid.SID64ToSID(sid))),
		fmt.Sprintf(`sm_kick "#%s""`, string(steamid.SID64ToSID(sid))))
	return nil
}

// UnbanPlayer will set the current ban to now, making it expired.
func UnbanPlayer(ctx context.Context, sid steamid.SID64) error {
	if !sid.Valid() {
		return errors.Errorf("Invalid steam id from: %s", sid.String())
	}
	ban, err := GetBan(sid)
	if err != nil {
		if err == ErrNoResult {
			return errors.Wrapf(err, "Player is not banned")
		} else {
			return err
		}
	}
	ban.Until = time.Now().UTC()
	if err := SaveBan(&ban); err != nil {
		return errors.Wrapf(err, "Failed to save unban")
	}
	log.Infof("Player unbanned: %v", sid.Int64())
	return nil
}

func BanPlayer(ctx context.Context, sid steamid.SID64, author steamid.SID64, duration time.Duration,
	reason model.Reason, reasonText string, source model.BanSource) error {
	if !sid.Valid() {
		return errors.Errorf("Invalid steam id from: %s", sid.String())
	}
	if !author.Valid() {
		return errors.Errorf("Invalid steam id (author) from: %s", author)
	}

	_, err := GetBan(sid)
	if err != nil && err == ErrNoResult {
		return errors.Wrapf(err, "Failed to lookup existing ban")
	}
	if err != ErrNoResult {
		return errors.Wrapf(ErrDuplicate, "Existing ban")
	}
	until := config.DefaultExpiration()
	if duration.Seconds() != 0 {
		until = config.Now().Add(duration)
	}
	ban := model.Ban{
		SteamID:    sid,
		AuthorID:   author,
		BanType:    model.Banned,
		Reason:     reason,
		ReasonText: reasonText,
		Note:       "",
		Until:      until,
		Source:     source,
		CreatedOn:  config.Now(),
		UpdatedOn:  config.Now(),
	}
	if err := SaveBan(&ban); err != nil {
		return DBErr(err)
	}
	servers, err := GetServers()
	if err != nil {
		log.Errorf("Failed to get server for ban propagation")
	}
	QueryRCON(ctx, servers, `gb_kick "#%s" %s`, string(steamid.SID64ToSID(sid)), reasonText)
	return nil
}

func BanIP(ctx context.Context, ip net.IPNet, author steamid.SID64, duration time.Duration,
	banType model.BanType, reason model.Reason, reasonText string, source model.BanSource) error {
	if !author.Valid() {
		return errors.Errorf("Failed to get steam id from: %s", author)
	}
	until := config.DefaultExpiration()
	if duration.Seconds() != 0 {
		until = config.Now().Add(duration)
	}
	banNet := model.BanNet{
		CIDR:      ip.String(),
		Source:    source,
		Reason:    reasonText,
		CreatedOn: config.Now(),
		UpdatedOn: config.Now(),
		Until:     until,
	}
	if err := SaveBanNet(&banNet); err != nil {
		return DBErr(err)
	}
	p, server, err := findPlayerByCIDR(ip.String())
	if err != nil && err != errUnknownID {
		return err
	}
	if err != errUnknownID && p != nil && server != nil {
		resp, err := execServerRCON(*server, fmt.Sprintf(`gb_kick "#%s" %s`, string(steamid.SID64ToSID(p.SID)), reasonText))
		if err != nil {
			return err
		}
		log.Debugf("RCON: %s", resp)
	}
	return err
}
