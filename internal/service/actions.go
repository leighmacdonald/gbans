package service

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

// MutePlayer will apply a mute to the players steam id. Mutes are propagated to the servers immediately.
// If duration set to 0, the value of config.DefaultExpiration() will be used.
func MutePlayer(ctx context.Context, sid steamid.SID64, author steamid.SID64, duration time.Duration,
	reason model.Reason, reasonText string) (*playerInfo, error) {
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
	ban, err2 := getBanBySteamID(ctx, sid, false)
	if err2 != nil && dbErr(err2) != errNoResult {
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
	if err3 := saveBan(ctx, ban.Ban); err3 != nil {
		log.Errorf("Failed to save ban: %v", err3)
		return nil, dbErr(err3)
	}
	pi := findPlayer(ctx, sid.String(), "")
	if pi.inGame {
		log.Infof("Gagging in-game player")
		queryRCON(ctx, []model.Server{*pi.server},
			fmt.Sprintf(`sm_gag "#%s"`, string(steamid.SID64ToSID(sid))),
			fmt.Sprintf(`sm_mute "#%s"`, string(steamid.SID64ToSID(sid))))
	}
	log.Infof("Gagged player successfully")
	return nil, nil
}

// UnbanPlayer will set the current ban to now, making it expired.
func UnbanPlayer(ctx context.Context, sid steamid.SID64, _ steamid.SID64, _ string) error {
	if !sid.Valid() {
		return errors.Errorf("Invalid steam id from: %s", sid.String())
	}
	ban, err := getBanBySteamID(ctx, sid, false)
	if err != nil {
		if err == errNoResult {
			return errors.Wrapf(err, "Player is not banned")
		} else {
			return err
		}
	}
	ban.Ban.ValidUntil = config.Now()
	if err2 := saveBan(ctx, ban.Ban); err2 != nil {
		return errors.Wrapf(err2, "Failed to save unban")
	}
	log.Infof("Player unbanned: %v", sid.Int64())
	return nil
}

// BanPlayer will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of config.DefaultExpiration() will be used.
func BanPlayer(ctx context.Context, sid steamid.SID64, author steamid.SID64, duration time.Duration,
	reason model.Reason, reasonText string, source model.BanSource) (*model.Ban, error) {
	if !sid.Valid() {
		return nil, errors.Errorf("Invalid steam id from: %s", sid.String())
	}
	if !author.Valid() {
		return nil, errors.Errorf("Invalid steam id (author) from: %s", author)
	}
	existing, err := getBanBySteamID(ctx, sid, false)
	if existing != nil && existing.Ban.BanID > 0 && existing.Ban.BanType == model.Banned {
		return nil, errDuplicate
	}
	if err != nil && err != errNoResult {
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
	if err2 := saveBan(ctx, &ban); err2 != nil {
		return nil, dbErr(err2)
	}
	go func() {
		ipAddr := ""
		// Kick the user if they currently are playing on a server
		pi := findPlayer(ctx, sid.String(), "")
		if pi.valid && pi.inGame {
			ipAddr = pi.player.IP.String()
			if _, errR := execServerRCON(*pi.server, fmt.Sprintf("sm_kick #%d %s", pi.player.UserID, reasonText)); errR != nil {
				log.Errorf("Faied to kick user afeter ban: %v", errR)
			}
		}
		// Update the profile, setting their IP
		if _, e := getOrCreateProfileBySteamID(ctx, sid, ipAddr); e != nil {
			log.Errorf("Failed to update banned user profile: %v", e)
		}
	}()
	return &ban, nil
}

// BanNetwork adds a new network to the banned network list. It will accept any valid CIDR format.
// It accepts an optional steamid to associate a particular user with the network ban. Any active players
// that fall within the range will be kicked immediately.
// If duration is 0, the value of config.DefaultExpiration() will be used.
func BanNetwork(ctx context.Context, cidr *net.IPNet, _ steamid.SID64, author steamid.SID64, duration time.Duration,
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
	if err := saveBanNet(ctx, &banNet); err != nil {
		return nil, dbErr(err)
	}
	p, server, err := findPlayerByCIDR(ctx, cidr)
	if err != nil && err != errUnknownID {
		return nil, err
	}
	if err != errUnknownID && p != nil && server != nil {
		resp, err2 := execServerRCON(*server, fmt.Sprintf(`gb_kick "#%s" %s`, string(steamid.SID64ToSID(p.SID)), reasonText))
		if err2 != nil {
			return nil, err2
		}
		log.Debugf("RCON: %s", resp)
	}
	return &banNet, nil
}

// KickPlayer will kick the steam id from all servers.
func KickPlayer(ctx context.Context, sid steamid.SID64, author steamid.SID64,
	_ model.Reason, reasonText string, _ model.BanSource) (*playerInfo, error) {
	if !sid.Valid() {
		return nil, errors.Errorf("Invalid steam id from: %s", sid.String())
	}
	if !author.Valid() {
		return nil, errors.Errorf("Invalid steam id (author) from: %s", author)
	}
	ipAddr := ""
	// Kick the user if they currently are playing on a server
	pi := findPlayer(ctx, sid.String(), "")
	if pi.valid && pi.inGame {
		ipAddr = pi.player.IP.String()
		if _, errR := execServerRCON(*pi.server, fmt.Sprintf("sm_kick #%d %s", pi.player.UserID, reasonText)); errR != nil {
			log.Errorf("Faied to kick user afeter ban: %v", errR)
		}
	}
	// Update the profile, setting their IP
	if _, e := getOrCreateProfileBySteamID(ctx, sid, ipAddr); e != nil {
		log.Errorf("Failed to update banned user profile: %v", e)
	}
	return &pi, nil
}
