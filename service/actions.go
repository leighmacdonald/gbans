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
	until := config.DefaultExpiration()
	if duration > 0 {
		until = config.Now().Add(duration)
	}
	ban := model.Ban{
		SteamID:    sid,
		AuthorID:   author,
		BanType:    model.NoComm,
		Reason:     reason,
		ReasonText: reasonText,
		Source:     0,
		ValidUntil: until,
		CreatedOn:  config.Now(),
		UpdatedOn:  config.Now(),
	}
	if err := SaveBan(&ban); err != nil {
		return DBErr(err)
	}
	servers, err := getServers()
	if err != nil {
		log.Errorf("Failed to get server for ban propagation")
	}
	queryRCON(ctx, servers,
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
		if err == errNoResult {
			return errors.Wrapf(err, "Player is not banned")
		} else {
			return err
		}
	}
	ban.ValidUntil = time.Now().UTC()
	if err := SaveBan(&ban); err != nil {
		return errors.Wrapf(err, "Failed to save unban")
	}
	log.Infof("Player unbanned: %v", sid.Int64())
	return nil
}

func BanPlayer(ctx context.Context, sid steamid.SID64, author steamid.SID64, duration time.Duration,
	reason model.Reason, reasonText string, source model.BanSource) (*model.Ban, error) {
	if !sid.Valid() {
		return nil, errors.Errorf("Invalid steam id from: %s", sid.String())
	}
	if !author.Valid() {
		return nil, errors.Errorf("Invalid steam id (author) from: %s", author)
	}

	existing, err := GetBan(sid)
	if err != nil {
		if err != errNoResult {
			return nil, errors.Wrapf(err, "Failed to get ban")
		}
	} else {
		return nil, errors.Wrapf(err, "Ban exists for steamid: %d :: %v", sid, existing)
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
		ValidUntil: until,
		Source:     source,
		CreatedOn:  config.Now(),
		UpdatedOn:  config.Now(),
	}
	if err := SaveBan(&ban); err != nil {
		return nil, DBErr(err)
	}
	servers, err2 := getServers()
	if err2 != nil {
		log.Errorf("Failed to get server for ban propagation")
	}
	queryRCON(ctx, servers, `gb_kick "#%s" %s`, string(steamid.SID64ToSID(sid)), reasonText)
	return &ban, nil
}

func BanNetwork(ctx context.Context, cidr *net.IPNet, sid steamid.SID64, author steamid.SID64, duration time.Duration,
	reason model.Reason, reasonText string, source model.BanSource) (*model.BanNet, error) {
	if !author.Valid() {
		return nil, errors.Errorf("Failed to get steam id from: %s", author)
	}
	until := config.DefaultExpiration()
	if duration.Seconds() != 0 {
		until = config.Now().Add(duration)
	}
	banNet := model.BanNet{
		CIDR:       cidr,
		Source:     source,
		Reason:     reasonText,
		CreatedOn:  config.Now(),
		UpdatedOn:  config.Now(),
		ValidUntil: until,
	}
	if err := saveBanNet(&banNet); err != nil {
		return nil, DBErr(err)
	}
	p, server, err := findPlayerByCIDR(cidr)
	if err != nil && err != errUnknownID {
		return nil, err
	}
	if err != errUnknownID && p != nil && server != nil {
		resp, err := execServerRCON(*server, fmt.Sprintf(`gb_kick "#%s" %s`, string(steamid.SID64ToSID(p.SID)), reasonText))
		if err != nil {
			return nil, err
		}
		log.Debugf("RCON: %s", resp)
	}
	return &banNet, nil
}
