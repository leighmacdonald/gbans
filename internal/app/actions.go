package app

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

type ActionHandlers struct{}

// MutePlayer will apply a mute to the players steam id. Mutes are propagated to the servers immediately.
// If duration set to 0, the value of config.DefaultExpiration() will be used.
func (a ActionHandlers) MutePlayer(ctx context.Context, sid steamid.SID64, author steamid.SID64, duration time.Duration,
	reason model.Reason, reasonText string) (*model.PlayerInfo, error) {
	if !sid.Valid() {
		return nil, errors.Errorf("Failed to get steam id from: %s", sid.String())
	}
	if !author.Valid() {
		return nil, errors.Errorf("Failed to get steam id from: %s", author)
	}
	until := config.DefaultExpiration()
	if duration > 0 {
		until = config.Now().Add(duration)
	}
	ban, err2 := store.GetBanBySteamID(ctx, sid, false)
	if err2 != nil && err2 != store.ErrNoResult {
		log.Errorf("Error getting ban from db: %v", err2)
		return nil, errors.New("Internal DB Error")
	} else if err2 != nil {
		ban = &model.BannedPerson{
			Ban:    model.NewBan(sid, 0, duration),
			Person: model.NewPerson(sid),
		}
		ban.Ban.BanType = model.NoComm
	}
	if ban.Ban.BanType == model.Banned {
		return nil, errors.New("Person is already banned")
	}
	ban.Ban.BanType = model.NoComm
	ban.Ban.Reason = reason
	ban.Ban.ReasonText = reasonText
	ban.Ban.ValidUntil = until
	if err3 := store.SaveBan(ctx, ban.Ban); err3 != nil {
		log.Errorf("Failed to save ban: %v", err3)
		return nil, err3
	}
	pi := a.FindPlayer(ctx, sid.String(), "")
	if pi.InGame {
		log.Infof("Gagging in-game Player")
		query.RCON(ctx, []model.Server{*pi.Server},
			fmt.Sprintf(`sm_gag "#%s"`, string(steamid.SID64ToSID(sid))),
			fmt.Sprintf(`sm_mute "#%s"`, string(steamid.SID64ToSID(sid))))
	}
	log.Infof("Gagged Player successfully")
	return nil, nil
}

// UnbanPlayer will set the current ban to now, making it expired.
func (a ActionHandlers) UnbanPlayer(ctx context.Context, sid steamid.SID64, _ steamid.SID64, _ string) error {
	if !sid.Valid() {
		return errors.Errorf("Invalid steam id from: %s", sid.String())
	}
	ban, err := store.GetBanBySteamID(ctx, sid, false)
	if err != nil {
		if err == store.ErrNoResult {
			return errors.Wrapf(err, "Player is not banned")
		} else {
			return err
		}
	}
	ban.Ban.ValidUntil = config.Now()
	if err2 := store.SaveBan(ctx, ban.Ban); err2 != nil {
		return errors.Wrapf(err2, "Failed to save unban")
	}
	log.Infof("Player unbanned: %v", sid.Int64())
	return nil
}

// BanPlayer will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of config.DefaultExpiration() will be used.
func (a ActionHandlers) BanPlayer(ctx context.Context, sid steamid.SID64, author steamid.SID64, duration time.Duration,
	reason model.Reason, reasonText string, source model.BanSource) (*model.Ban, error) {
	if !sid.Valid() {
		return nil, errors.Errorf("Invalid steam id from: %s", sid.String())
	}
	if !author.Valid() {
		return nil, errors.Errorf("Invalid steam id (author) from: %s", author)
	}
	existing, err := store.GetBanBySteamID(ctx, sid, false)
	if existing != nil && existing.Ban.BanID > 0 && existing.Ban.BanType == model.Banned {
		return nil, store.ErrDuplicate
	}
	if err != nil && err != store.ErrNoResult {
		return nil, errors.Wrapf(err, "Failed to get ban")
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
	if err2 := store.SaveBan(ctx, &ban); err2 != nil {
		return nil, err2
	}
	go func() {
		ipAddr := ""
		// Kick the user if they currently are playing on a Server
		pi := a.FindPlayer(ctx, sid.String(), "")
		if pi.Valid && pi.InGame {
			ipAddr = pi.Player.IP.String()
			if _, errR := query.ExecRCON(*pi.Server, fmt.Sprintf("sm_kick #%d %s", pi.Player.UserID, reasonText)); errR != nil {
				log.Errorf("Faied to kick user afeter ban: %v", errR)
			}
		}
		// Update the profile, setting their IP
		if _, e := a.GetOrCreateProfileBySteamID(ctx, sid, ipAddr); e != nil {
			log.Errorf("Failed to update banned user profile: %v", e)
		}
	}()
	return &ban, nil
}

// BanNetwork adds a new network to the banned network list. It will accept any Valid CIDR format.
// It accepts an optional steamid to associate a particular user with the network ban. Any active players
// that fall within the range will be kicked immediately.
// If duration is 0, the value of config.DefaultExpiration() will be used.
func (a ActionHandlers) BanNetwork(ctx context.Context, cidr *net.IPNet, _ steamid.SID64, author steamid.SID64, duration time.Duration,
	_ model.Reason, reasonText string, source model.BanSource) (*model.BanNet, error) {
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
	if err := store.SaveBanNet(ctx, &banNet); err != nil {
		return nil, err
	}
	p, server, err := a.FindPlayerByCIDR(ctx, cidr)
	if err != nil && err != consts.ErrUnknownID {
		return nil, err
	}
	if err != consts.ErrUnknownID && p != nil && server != nil {
		resp, err2 := query.ExecRCON(*server, fmt.Sprintf(`gb_kick "#%s" %s`, string(steamid.SID64ToSID(p.SID)), reasonText))
		if err2 != nil {
			return nil, err2
		}
		log.Debugf("RCON: %s", resp)
	}
	return &banNet, nil
}

// KickPlayer will kick the steam id from all servers.
func (a ActionHandlers) KickPlayer(ctx context.Context, sid steamid.SID64, author steamid.SID64,
	_ model.Reason, reasonText string, _ model.BanSource) (*model.PlayerInfo, error) {
	if !sid.Valid() {
		return nil, errors.Errorf("Invalid steam id from: %s", sid.String())
	}
	if !author.Valid() {
		return nil, errors.Errorf("Invalid steam id (author) from: %s", author)
	}
	ipAddr := ""
	// Kick the user if they currently are playing on a Server
	pi := a.FindPlayer(ctx, sid.String(), "")
	if pi.Valid && pi.InGame {
		ipAddr = pi.Player.IP.String()
		if _, errR := query.ExecRCON(*pi.Server, fmt.Sprintf("sm_kick #%d %s", pi.Player.UserID, reasonText)); errR != nil {
			log.Errorf("Faied to kick user afeter ban: %v", errR)
		}
	}
	// Update the profile, setting their IP
	if _, e := a.GetOrCreateProfileBySteamID(ctx, sid, ipAddr); e != nil {
		log.Errorf("Failed to update banned user profile: %v", e)
	}
	return &pi, nil
}
