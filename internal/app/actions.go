package app

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/action"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"strings"
	"time"
)

// Mute will apply a mute to the players steam id. Mutes are propagated to the servers immediately.
// If duration set to 0, the value of config.DefaultExpiration() will be used.
func (g gbans) Mute(args action.MuteRequest, pi *model.PlayerInfo) error {
	target, err := args.Target.SID64()
	if err != nil {
		return errors.Errorf("Failed to get steam id from: %s", args.Target)
	}
	source, errSrc := args.Source.SID64()
	if errSrc != nil {
		return errors.Errorf("Failed to get steam id from: %s", args.Source)
	}
	duration, errDur := args.Duration.Value()
	if errDur != nil {
		return errDur
	}
	until := config.DefaultExpiration()
	if duration > 0 {
		until = config.Now().Add(duration)
	}
	b := model.NewBannedPerson()
	if err2 := g.db.GetBanBySteamID(g.ctx, target, false, &b); err2 != nil && err2 != store.ErrNoResult {
		log.Errorf("Error getting b from db: %v", err2)
		return errors.New("Internal DB Error")
	} else if err2 != nil {
		b = model.BannedPerson{
			Ban:    model.NewBan(target, source, duration),
			Person: model.NewPerson(target),
		}
		b.Ban.BanType = model.NoComm
	}
	if b.Ban.BanType == model.Banned {
		return errors.New("Person is already banned")
	}
	b.Ban.BanType = model.NoComm
	b.Ban.Reason = model.Custom
	b.Ban.ReasonText = args.Reason
	b.Ban.ValidUntil = until
	if err3 := g.db.SaveBan(g.ctx, &b.Ban); err3 != nil {
		log.Errorf("Failed to save b: %v", err3)
		return err3
	}

	if errF := g.Find(target.String(), "", pi); errF != nil {
		return nil
	}
	if pi.InGame {
		log.Infof("Gagging in-game Player")
		query.RCON(g.ctx, []model.Server{*pi.Server},
			fmt.Sprintf(`sm_gag "#%s"`, string(steamid.SID64ToSID(target))),
			fmt.Sprintf(`sm_mute "#%s"`, string(steamid.SID64ToSID(target))))
	}
	log.Infof("Gagged Player successfully")
	return nil
}

// Unban will set the current ban to now, making it expired.
// Returns true, nil if the ban exists, and was successfully banned.
// Returns false, nil if the ban does not exist.
func (g gbans) Unban(args action.UnbanRequest) (bool, error) {
	target, errTar := args.Target.SID64()
	if errTar != nil {
		return false, errTar
	}
	//source, errSrc := args.Origin.SID64()
	//if errSrc != nil {
	//	return false, errSrc
	//}
	b := model.NewBannedPerson()
	err := g.db.GetBanBySteamID(g.ctx, target, false, &b)
	if err != nil {
		if err == store.ErrNoResult {
			return false, nil
		}
		return false, err
	}
	b.Ban.ValidUntil = config.Now()
	if err2 := g.db.SaveBan(g.ctx, &b.Ban); err2 != nil {
		return false, errors.Wrapf(err2, "Failed to save unban")
	}
	log.Infof("Player unbanned: %v", target)
	return true, nil
}

// Ban will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of config.DefaultExpiration() will be used.
func (g gbans) Ban(args action.BanRequest, b *model.Ban) error {
	target, errTar := args.Target.SID64()
	if errTar != nil {
		return errTar
	}
	source, errSrc := args.Source.SID64()
	if errSrc != nil {
		return errSrc
	}
	duration, errDur := args.Duration.Value()
	if errDur != nil {
		return errDur
	}
	existing := model.NewBannedPerson()
	err := g.db.GetBanBySteamID(g.ctx, target, false, &existing)
	if existing.Ban.BanID > 0 && existing.Ban.BanType == model.Banned {
		return store.ErrDuplicate
	}
	if err != nil && err != store.ErrNoResult {
		return errors.Wrapf(err, "Failed to get b")
	}
	until := config.DefaultExpiration()
	if duration.Seconds() != 0 {
		until = config.Now().Add(duration)
	}
	b.SteamID = target
	b.AuthorID = source
	b.BanType = model.Banned
	b.Reason = model.Custom
	b.ReasonText = args.Reason
	b.Note = ""
	b.ValidUntil = until
	b.Source = model.System
	b.CreatedOn = config.Now()
	b.UpdatedOn = config.Now()

	if err2 := g.db.SaveBan(g.ctx, b); err2 != nil {
		return err2
	}
	go func() {
		banNotice := &discordgo.MessageEmbed{
			URL:   fmt.Sprintf("https://steamcommunity.com/profiles/%d", b.SteamID),
			Type:  discordgo.EmbedTypeRich,
			Title: fmt.Sprintf("User Banned (#%d)", b.BanID),
			Color: 10038562,
		}
		banNotice.Fields = append(banNotice.Fields, &discordgo.MessageEmbedField{
			Name:   "STEAM",
			Value:  string(steamid.SID64ToSID(b.SteamID)),
			Inline: true,
		})
		banNotice.Fields = append(banNotice.Fields, &discordgo.MessageEmbedField{
			Name:   "STEAM3",
			Value:  string(steamid.SID64ToSID3(b.SteamID)),
			Inline: true,
		})
		banNotice.Fields = append(banNotice.Fields, &discordgo.MessageEmbedField{
			Name:   "SID64",
			Value:  b.SteamID.String(),
			Inline: true,
		})
		banNotice.Fields = append(banNotice.Fields, &discordgo.MessageEmbedField{
			Name:   "Expires In",
			Value:  config.FmtDuration(b.ValidUntil),
			Inline: false,
		})
		banNotice.Fields = append(banNotice.Fields, &discordgo.MessageEmbedField{
			Name:   "Expires At",
			Value:  config.FmtTimeShort(b.ValidUntil),
			Inline: false,
		})
		if config.Discord.PublicLogChannelEnable {
			if errPLC := g.bot.SendEmbed(config.Discord.PublicLogChannelId, banNotice); errPLC != nil {
				log.Errorf("Failed to send ban notice to public channel: %v", errPLC)
			}
		}
	}()
	go func() {
		ipAddr := ""
		// kick the user if they currently are playing on a server
		pi := model.NewPlayerInfo()
		_ = g.Find(target.String(), "", &pi)
		if pi.Valid && pi.InGame {
			ipAddr = pi.Player.IP.String()
			if _, errR := query.ExecRCON(*pi.Server,
				fmt.Sprintf("sm_kick #%d %s", pi.Player.UserID, args.Reason)); errR != nil {
				log.Errorf("Faied to kick user afeter b: %v", errR)
			}
			p := model.NewPerson(pi.Player.SID)
			if errG := g.db.GetOrCreatePersonBySteamID(g.ctx, pi.Player.SID, &p); errG != nil {
				log.Errorf("Failed to fetch banned player: %v", errG)
			}
			p.IPAddr = net.ParseIP(ipAddr)
			if errS := g.db.SavePerson(g.ctx, &p); errS != nil {
				log.Errorf("Failed to update banned player up: %v", errS)
			}
		}
	}()
	return nil
}

// BanNetwork adds a new network to the banned network list. It will accept any Valid CIDR format.
// It accepts an optional steamid to associate a particular user with the network ban. Any active players
// that fall within the range will be kicked immediately.
// If duration is 0, the value of config.DefaultExpiration() will be used.
func (g gbans) BanNetwork(args action.BanNetRequest, banNet *model.BanNet) error {
	target, errTar := args.Target.SID64()
	if errTar != nil {
		return errTar
	}
	source, errSrc := args.Source.SID64()
	if errSrc != nil {
		return errSrc
	}
	duration, errDur := args.Duration.Value()
	if errDur != nil {
		return errDur
	}
	until := config.DefaultExpiration()
	if duration.Seconds() != 0 {
		until = config.Now().Add(duration)
	}
	_, cidr, errCidr := net.ParseCIDR(args.CIDR)
	if errCidr != nil {
		return errors.Wrapf(errCidr, "Failed to parse CIDR address")
	}
	// TODO
	//_, err2 := store.GetBanNet(ctx, net.ParseIP(cidrStr))
	//if err2 != nil && err2 != store.ErrNoResult {
	//	return "", errCommandFailed
	//}
	//if err2 == nil {
	//	return "", consts.ErrDuplicateBan
	//}

	banNet.SteamID = target
	banNet.AuthorID = source
	banNet.CIDR = cidr
	banNet.Source = model.System
	banNet.Reason = args.Reason
	banNet.CreatedOn = config.Now()
	banNet.UpdatedOn = config.Now()
	banNet.ValidUntil = until

	if err := g.db.SaveBanNet(g.ctx, banNet); err != nil {
		return err
	}
	go func() {
		var pi model.PlayerInfo
		if errPI := g.FindPlayerByCIDR(cidr, &pi); errPI != nil {
			return
		}
		if pi.Player != nil && pi.Server != nil {
			_, err2 := query.ExecRCON(*pi.Server,
				fmt.Sprintf(`gb_kick "#%s" %s`, string(steamid.SID64ToSID(pi.Player.SID)), banNet.Reason))
			if err2 != nil {
				log.Errorf("Failed to query for ban request: %v", err2)
				return
			}
		}
	}()

	return nil
}

// Kick will kick the steam id from all servers.
func (g gbans) Kick(args action.KickRequest, pi *model.PlayerInfo) error {
	target, errTar := args.Target.SID64()
	if errTar != nil {
		return errTar
	}
	//source, errSrc := args.Origin.SID64()
	//if errSrc != nil {
	//	return nil, errSrc
	//}
	// kick the user if they currently are playing on a server
	var foundPI model.PlayerInfo
	_ = g.Find(target.String(), "", &foundPI)
	if pi.Valid && pi.InGame {
		if _, errR := query.ExecRCON(*pi.Server, fmt.Sprintf("sm_kick #%d %s", pi.Player.UserID, args.Reason)); errR != nil {
			log.Errorf("Faied to kick user afeter ban: %v", errR)
		}
	}
	pi = &foundPI
	return nil
}

func (g gbans) SetSteam(args action.SetSteamIDRequest) (bool, error) {
	sid, err := steamid.ResolveSID64(g.ctx, string(args.Target))
	if err != nil || !sid.Valid() {
		return false, consts.ErrInvalidSID
	}
	var p model.Person
	if errP := g.db.GetOrCreatePersonBySteamID(g.ctx, sid, &p); errP != nil || !sid.Valid() {
		return false, consts.ErrInvalidSID
	}
	if (p.DiscordID) != "" {
		return false, errors.Errorf("Discord account already linked to steam account: %d", p.SteamID.Int64())
	}
	p.DiscordID = args.DiscordID
	if errS := g.db.SavePerson(g.ctx, &p); errS != nil {
		return false, consts.ErrInternal
	}
	return true, nil
}

func (g gbans) Say(args action.SayRequest) error {
	var server model.Server
	if err := g.db.GetServerByName(g.ctx, args.Server, &server); err != nil {
		return errors.Errorf("Failed to fetch server: %s", args.Server)
	}
	msg := fmt.Sprintf(`sm_say %s`, args.Message)
	resp, err2 := query.ExecRCON(server, msg)
	if err2 != nil {
		return err2
	}
	rp := strings.Split(resp, "\n")
	if len(rp) < 2 {
		return errors.Errorf("Invalid response")
	}
	return nil
}

func (g gbans) CSay(args action.CSayRequest) error {
	var (
		servers []model.Server
		err     error
	)
	if args.Server == "*" {
		servers, err = g.db.GetServers(g.ctx, false)
		if err != nil {
			return errors.Wrapf(err, "Failed to fetch servers")
		}
	} else {
		var server model.Server
		if errS := g.db.GetServerByName(g.ctx, args.Server, &server); errS != nil {
			return errors.Wrapf(errS, "Failed to fetch server: %s", args.Server)
		}
		servers = append(servers, server)
	}
	msg := fmt.Sprintf(`sm_csay %s`, args.Message)
	_ = query.RCON(g.ctx, servers, msg)
	return nil
}

func (g gbans) PSay(args action.PSayRequest) error {
	var pi model.PlayerInfo
	_ = g.Find(string(args.Target), "", &pi)
	if !pi.Valid || !pi.InGame {
		return consts.ErrUnknownID
	}
	msg := fmt.Sprintf(`sm_psay %d "%s"`, pi.Player.UserID, args.Message)
	_, err := query.ExecRCON(*pi.Server, msg)
	if err != nil {
		return errors.Errorf("Failed to exec psay command: %v", err)
	}
	return nil
}

//func (g Gbans) filterAdd(args action.FilterAddRequest) (*model.Filter, error) {
//	f, err := g.db.InsertFilter(g.ctx, args.Filter)
//	if err != nil {
//		if err == store.ErrDuplicate {
//			return nil, store.ErrDuplicate
//		}
//		log.Errorf("Error saving filter word: %v", err)
//		return nil, consts.ErrInternal
//	}
//	return f, nil
//}
//
//func (g Gbans) filterDel(ctx context.Context, args action.FilterDelRequest) (bool, error) {
//	var filter model.Filter
//	if err := g.db.GetFilterByID(ctx, args.FilterID, &filter); err != nil {
//		return false, err
//	}
//	if err2 := g.db.DropFilter(ctx, &filter); err2 != nil {
//		return false, err2
//	}
//	return true, nil
//}
//
//func (g Gbans) filterCheck(ctx context.Context, args action.FilterCheckRequest) ([]*model.Filter, error) {
//	return nil, errors.New("unimplemented")
//}

func (g gbans) PersonBySID(sid steamid.SID64, ipAddr string, p *model.Person) error {
	if err := g.db.GetPersonBySteamID(g.ctx, sid, p); err != nil && err != store.ErrNoResult {
		return err
	}
	if p.UpdatedOn == p.CreatedOn || time.Since(p.UpdatedOn) > 15*time.Second {
		s, errW := steamweb.PlayerSummaries(steamid.Collection{p.SteamID})
		if errW != nil || len(s) != 1 {
			log.Errorf("Failed to fetch updated profile: %v", errW)
			return nil
		}
		var sum = s[0]
		p.PlayerSummary = &sum
		p.UpdatedOn = time.Now()
		if err := g.db.SavePerson(g.ctx, p); err != nil {
			log.Errorf("Failed to save updated profile: %v", errW)
			return nil
		}
		if err := g.db.GetPersonBySteamID(g.ctx, sid, p); err != nil && err != store.ErrNoResult {
			return err
		}
	}
	return nil
}

func (g gbans) ResolveSID(sidStr string) (steamid.SID64, error) {
	c, cancel := context.WithTimeout(g.ctx, time.Second*5)
	defer cancel()
	return steamid.ResolveSID64(c, sidStr)
}
