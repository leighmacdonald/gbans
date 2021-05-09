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
	ban, err2 := getBanBySteamID(ctx, sid, false)
	if err2 != nil && dbErr(err2) != errNoResult {
		log.Errorf("Error getting ban from db: %v", err2)
		return errors.New("Internal DB Error")
	} else if err2 != nil {
		ban = &model.BannedPerson{
			Ban:    model.NewBan(sid, 0, duration),
			Person: model.NewPerson(sid),
		}
	}
	if ban.Ban.BanType == model.Banned {
		return errors.New("Person is already banned")
	}
	ban.Ban.BanType = model.NoComm
	ban.Ban.Reason = reason
	ban.Ban.ReasonText = reasonText
	ban.Ban.ValidUntil = until
	if err3 := saveBan(ctx, ban.Ban); err3 != nil {
		log.Errorf("Failed to save ban: %v", err3)
		return dbErr(err3)
	}
	servers, err := getServers(ctx)
	if err != nil {
		log.Errorf("Failed to get server for ban propagation")
	} else {
		queryRCON(ctx, servers,
			fmt.Sprintf(`sm_gag "#%s"`, string(steamid.SID64ToSID(sid))),
			fmt.Sprintf(`sm_mute "#%s"`, string(steamid.SID64ToSID(sid))))

		//if pi.inGame {
		//	resp, err4 := execServerRCON(*pi.server, fmt.Sprintf(`sm_gag "#%s"`, steamid.SID64ToSID3(pi.sid)))
		//	if err4 != nil {
		//		log.Errorf("Failed to gag active user: %v", err4)
		//	} else {
		//		if strings.Contains(resp, "[SM] Gagged") {
		//			var dStr string
		//			if duration.Seconds() == 0 {
		//				dStr = "Forever"
		//			} else {
		//				dStr = duration.String()
		//			}
		//			return "", errors.Errorf("Person gagged successfully for: %s", dStr)
		//		} else {
		//			return "", errors.New("Failed to gag player in-game")
		//		}
		//	}
		//}

	}
	return nil
}

// UnbanPlayer will set the current ban to now, making it expired.
func UnbanPlayer(ctx context.Context, sid steamid.SID64) error {
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
	if existing != nil && existing.Ban.BanID > 0 {
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
