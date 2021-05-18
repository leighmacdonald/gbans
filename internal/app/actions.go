package app

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/action"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
)

// mute will apply a mute to the players steam id. Mutes are propagated to the servers immediately.
// If duration set to 0, the value of config.DefaultExpiration() will be used.
func mute(ctx context.Context, args *action.MuteRequest) (*model.PlayerInfo, error) {
	target, err := args.Target.SID64()
	if err != nil {
		return nil, errors.Errorf("Failed to get steam id from: %s", args.Target)
	}
	source, errSrc := args.Source.SID64()
	if errSrc != nil {
		return nil, errors.Errorf("Failed to get steam id from: %s", args.Source)
	}
	duration, errDur := args.Duration.Value()
	if errDur != nil {
		return nil, errDur
	}
	until := config.DefaultExpiration()
	if duration > 0 {
		until = config.Now().Add(duration)
	}
	b, err2 := store.GetBanBySteamID(ctx, target, false)
	if err2 != nil && err2 != store.ErrNoResult {
		log.Errorf("Error getting b from db: %v", err2)
		return nil, errors.New("Internal DB Error")
	} else if err2 != nil {
		b = &model.BannedPerson{
			Ban:    model.NewBan(target, source, duration),
			Person: model.NewPerson(target),
		}
		b.Ban.BanType = model.NoComm
	}
	if b.Ban.BanType == model.Banned {
		return nil, errors.New("Person is already banned")
	}
	b.Ban.BanType = model.NoComm
	b.Ban.Reason = model.Custom
	b.Ban.ReasonText = args.Reason
	b.Ban.ValidUntil = until
	if err3 := store.SaveBan(ctx, b.Ban); err3 != nil {
		log.Errorf("Failed to save b: %v", err3)
		return nil, err3
	}
	pi := FindPlayer(ctx, target.String(), "")
	if pi.InGame {
		log.Infof("Gagging in-game Player")
		query.RCON(ctx, []model.Server{*pi.Server},
			fmt.Sprintf(`sm_gag "#%s"`, string(steamid.SID64ToSID(target))),
			fmt.Sprintf(`sm_mute "#%s"`, string(steamid.SID64ToSID(target))))
	}
	log.Infof("Gagged Player successfully")
	return nil, nil
}

// unban will set the current ban to now, making it expired.
// Returns true, nil if the ban exists, and was successfully banned.
// Returns false, nil if the ban does not exist.
func unban(ctx context.Context, args *action.UnbanRequest) (bool, error) {
	target, errTar := args.Target.SID64()
	if errTar != nil {
		return false, errTar
	}
	//source, errSrc := args.Source.SID64()
	//if errSrc != nil {
	//	return false, errSrc
	//}
	b, err := store.GetBanBySteamID(ctx, target, false)
	if err != nil {
		if err == store.ErrNoResult {
			return false, nil
		} else {
			return false, err
		}
	}
	b.Ban.ValidUntil = config.Now()
	if err2 := store.SaveBan(ctx, b.Ban); err2 != nil {
		return false, errors.Wrapf(err2, "Failed to save unban")
	}
	log.Infof("Player unbanned: %v", target)
	return true, nil
}

// ban will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of config.DefaultExpiration() will be used.
func ban(ctx context.Context, args *action.BanRequest) (*model.Ban, error) {
	target, errTar := args.Target.SID64()
	if errTar != nil {
		return nil, errTar
	}
	source, errSrc := args.Source.SID64()
	if errSrc != nil {
		return nil, errSrc
	}
	duration, errDur := args.Duration.Value()
	if errDur != nil {
		return nil, errDur
	}
	existing, err := store.GetBanBySteamID(ctx, target, false)
	if existing != nil && existing.Ban.BanID > 0 && existing.Ban.BanType == model.Banned {
		return nil, store.ErrDuplicate
	}
	if err != nil && err != store.ErrNoResult {
		return nil, errors.Wrapf(err, "Failed to get b")
	}
	until := config.DefaultExpiration()
	if duration.Seconds() != 0 {
		until = config.Now().Add(duration)
	}
	b := model.Ban{
		SteamID:    target,
		AuthorID:   source,
		BanType:    model.Banned,
		Reason:     model.Custom,
		ReasonText: args.Reason,
		Note:       "",
		ValidUntil: until,
		Source:     model.System,
		CreatedOn:  config.Now(),
		UpdatedOn:  config.Now(),
	}
	if err2 := store.SaveBan(ctx, &b); err2 != nil {
		return nil, err2
	}
	go func() {
		ipAddr := ""
		// kick the user if they currently are playing on a Server
		pi := FindPlayer(ctx, target.String(), "")
		if pi.Valid && pi.InGame {
			ipAddr = pi.Player.IP.String()
			if _, errR := query.ExecRCON(*pi.Server,
				fmt.Sprintf("sm_kick #%d %s", pi.Player.UserID, args.Reason)); errR != nil {
				log.Errorf("Faied to kick user afeter b: %v", errR)
			}
		}
		updateRequest := action.NewProfile(target.String(), ipAddr)
		updateRequest.Enqueue().Done()
	}()
	return &b, nil
}

// banNetwork adds a new network to the banned network list. It will accept any Valid CIDR format.
// It accepts an optional steamid to associate a particular user with the network ban. Any active players
// that fall within the range will be kicked immediately.
// If duration is 0, the value of config.DefaultExpiration() will be used.
func banNetwork(ctx context.Context, args *action.BanNetRequest) (*model.BanNet, error) {
	target, errTar := args.Target.SID64()
	if errTar != nil {
		return nil, errTar
	}
	source, errSrc := args.Source.SID64()
	if errSrc != nil {
		return nil, errSrc
	}
	duration, errDur := args.Duration.Value()
	if errDur != nil {
		return nil, errDur
	}
	until := config.DefaultExpiration()
	if duration.Seconds() != 0 {
		until = config.Now().Add(duration)
	}
	_, cidr, errCidr := net.ParseCIDR(args.CIDR)
	if errCidr != nil {
		return nil, errors.Wrapf(errCidr, "Failed to parse CIDR address")
	}
	// TODO
	//_, err2 := store.GetBanNet(ctx, net.ParseIP(cidrStr))
	//if err2 != nil && err2 != store.ErrNoResult {
	//	return "", errCommandFailed
	//}
	//if err2 == nil {
	//	return "", consts.ErrDuplicateBan
	//}

	banNet := model.BanNet{
		SteamID:    target,
		AuthorID:   source,
		CIDR:       cidr,
		Source:     model.System,
		Reason:     args.Reason,
		CreatedOn:  config.Now(),
		UpdatedOn:  config.Now(),
		ValidUntil: until,
	}
	if err := store.SaveBanNet(ctx, &banNet); err != nil {
		return nil, err
	}
	go func() {
		fc := action.NewFindByCIDR(cidr)
		res := <-fc.Enqueue().Done()
		if res.Err != nil {
			if res.Err != consts.ErrUnknownID {
				log.Debugf("No active players found matching: %s", cidr)
				return
			}
			log.Errorf("Error finding player: %v", res.Err)
			return
		}
		pi, ok := res.Value.(model.PlayerInfo)
		if !ok {
			log.Errorf("Failed casing player info")
			return
		}
		if res.Err != consts.ErrUnknownID && pi.Player != nil && pi.Server != nil {
			_, err2 := query.ExecRCON(*pi.Server,
				fmt.Sprintf(`gb_kick "#%s" %s`, string(steamid.SID64ToSID(pi.Player.SID)), banNet.Reason))
			if err2 != nil {
				log.Errorf("Failed to query for ban request: %v", err2)
				return
			}
		}
	}()

	return &banNet, nil
}

// kick will kick the steam id from all servers.
func kick(ctx context.Context, args *action.KickRequest) (*model.PlayerInfo, error) {
	target, errTar := args.Target.SID64()
	if errTar != nil {
		return nil, errTar
	}
	//source, errSrc := args.Source.SID64()
	//if errSrc != nil {
	//	return nil, errSrc
	//}
	ipAddr := ""
	// kick the user if they currently are playing on a Server
	pi := FindPlayer(ctx, target.String(), "")
	if pi.Valid && pi.InGame {
		ipAddr = pi.Player.IP.String()

		if _, errR := query.ExecRCON(*pi.Server, fmt.Sprintf("sm_kick #%d %s", pi.Player.UserID, args.Reason)); errR != nil {
			log.Errorf("Faied to kick user afeter ban: %v", errR)
		}
	}
	pr := action.NewProfile(target.String(), ipAddr)
	pr.EnqueueIgnore()
	return &pi, nil
}

func setSteam(ctx context.Context, args *action.SetSteamIDRequest) (bool, error) {
	sid, err := steamid.ResolveSID64(ctx, string(args.Target))
	if err != nil || !sid.Valid() {
		return false, consts.ErrInvalidSID
	}
	p, errP := store.GetOrCreatePersonBySteamID(ctx, sid)
	if errP != nil || !sid.Valid() {
		return false, consts.ErrInvalidSID
	}
	if (p.DiscordID) != "" {
		return false, errors.Errorf("Discord account already linked to steam account: %d", p.SteamID.Int64())
	}
	p.DiscordID = args.DiscordID
	if errS := store.SavePerson(ctx, p); errS != nil {
		return false, consts.ErrInternal
	}
	return true, nil
}

func say(ctx context.Context, args *action.SayRequest) (bool, error) {
	server, err := store.GetServerByName(ctx, args.Server)
	if err != nil {
		return false, errors.Errorf("Failed to fetch server: %s", args.Server)
	}
	msg := fmt.Sprintf(`sm_say %s`, args.Message)
	resp, err2 := query.ExecRCON(server, msg)
	if err2 != nil {
		return false, err2
	}
	rp := strings.Split(resp, "\n")
	if len(rp) < 2 {
		return false, errors.Errorf("Invalid response")
	}
	return true, nil
}

func csay(ctx context.Context, args *action.CSayRequest) (bool, error) {
	var (
		servers []model.Server
		err     error
	)
	if args.Server == "*" {
		servers, err = store.GetServers(ctx)
		if err != nil {
			return false, errors.Wrapf(err, "Failed to fetch servers")
		}
	} else {
		server, errS := store.GetServerByName(ctx, args.Server)
		if errS != nil {
			return false, errors.Wrapf(errS, "Failed to fetch server: %s", args.Server)
		}
		servers = append(servers, server)
	}
	msg := fmt.Sprintf(`sm_csay %s`, args.Message)
	_ = query.RCON(ctx, servers, msg)
	return true, nil
}

func psay(ctx context.Context, args *action.PSayRequest) (bool, error) {
	pi := FindPlayer(ctx, string(args.Target), "")
	if !pi.Valid || !pi.InGame {
		return false, consts.ErrUnknownID
	}
	msg := fmt.Sprintf(`sm_psay %d "%s"`, pi.Player.UserID, args.Message)
	_, err := query.ExecRCON(*pi.Server, msg)
	if err != nil {
		return false, errors.Errorf("Failed to exec psay command: %v", err)
	}
	return true, nil
}

func filterAdd(ctx context.Context, args *action.FilterAddRequest) (*model.Filter, error) {
	f, err := store.InsertFilter(ctx, args.Filter)
	if err != nil {
		if err == store.ErrDuplicate {
			return nil, store.ErrDuplicate
		}
		log.Errorf("Error saving filter word: %v", err)
		return nil, consts.ErrInternal
	}
	return f, nil
}

func filterDel(ctx context.Context, args *action.FilterDelRequest) (bool, error) {
	filter, err := store.GetFilterByID(ctx, args.FilterID)
	if err != nil {
		return false, err
	}
	if err2 := store.DropFilter(ctx, filter); err2 != nil {
		return false, err2
	}
	return true, nil
}

func filterCheck(ctx context.Context, args *action.FilterCheckRequest) ([]*model.Filter, error) {
	return nil, errors.New("unimplemented")
}
