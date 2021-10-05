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
	"regexp"
	"strconv"
	"strings"
	"time"
)

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

// UnbanASN will remove an existing ASN ban
func (g gbans) UnbanASN(ctx context.Context, args action.UnbanASNRequest) (bool, error) {
	asNum, errConv := strconv.ParseInt(args.ASNum, 10, 64)
	if errConv != nil {
		return false, errConv
	}
	var ba model.BanASN
	if err := g.db.GetBanASN(g.ctx, asNum, &ba); err != nil {
		return false, err
	}
	if errDrop := g.db.DropBanASN(ctx, &ba); errDrop != nil {
		log.Errorf("Failed to drop ASN ban: %v", errDrop)
		return false, errDrop
	}
	log.Infof("ASN unbanned: %d", asNum)
	return true, nil
}

// Ban will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of config.DefaultExpiration() will be used.
func (g gbans) Ban(args action.BanRequest, b *model.Ban) error {
	target, errTar := args.Target.SID64()
	if errTar != nil {
		return errTar
	}
	source, errSrc := args.Author.SID64()
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
	b.BanType = args.BanType
	b.Reason = model.Custom
	b.ReasonText = args.Reason
	b.Note = ""
	b.ValidUntil = until
	b.Source = args.Origin
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
	ipAddr := ""
	// kick the user if they currently are playing on a server
	pi := model.NewPlayerInfo()
	_ = g.Find(target.String(), "", &pi)
	if pi.Valid && pi.InGame {
		switch args.BanType {
		case model.NoComm:
			{
				log.Infof("Gagging in-game Player")
				query.RCON(g.ctx, []model.Server{*pi.Server},
					fmt.Sprintf(`sm_gag "#%s"`, string(steamid.SID64ToSID(target))),
					fmt.Sprintf(`sm_mute "#%s"`, string(steamid.SID64ToSID(target))))
			}
		case model.Banned:
			{
				log.Infof("Banning and kicking in-game Player")
				ipAddr = pi.Player.IP.String()
				if _, errR := query.ExecRCON(*pi.Server,
					fmt.Sprintf("sm_kick #%d %s", pi.Player.UserID, args.Reason)); errR != nil {
					log.Errorf("Faied to kick user after ban: %v", errR)
				}
			}
		}
		p := model.NewPerson(pi.Player.SID)
		if errG := g.db.GetOrCreatePersonBySteamID(g.ctx, pi.Player.SID, &p); errG != nil {
			log.Errorf("Failed to fetch banned player: %v", errG)
		}
		p.IPAddr = net.ParseIP(ipAddr)
		if errS := g.db.SavePerson(g.ctx, &p); errS != nil {
			log.Errorf("Failed to update banned player ip: %v", errS)
		}
	}
	return nil
}

// BanASN will ban all network ranges associated with the requested ASN
func (g gbans) BanASN(args action.BanASNRequest, banASN *model.BanASN) error {
	target, errTar := args.Target.SID64()
	if errTar != nil {
		return errTar
	}
	author, errSrc := args.Author.SID64()
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
	banASN.Origin = args.Origin
	banASN.TargetID = target
	banASN.AuthorID = author
	banASN.ValidUntil = until
	banASN.Reason = args.Reason
	banASN.ASNum = args.ASNum
	if errSave := g.db.SaveBanASN(context.TODO(), banASN); errSave != nil {
		return errSave
	}
	// TODO Kick all current players matching
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
	source, errSrc := args.Author.SID64()
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

// Kick will kick the steam id from whatever server it is connected to.
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
	if errF := g.Find(target.String(), "", &foundPI); errF != nil {
		return errF
	}

	if foundPI.Valid && foundPI.InGame {
		resp, errR := query.ExecRCON(*foundPI.Server, fmt.Sprintf("sm_kick #%d %s", foundPI.Player.UserID, args.Reason))
		if errR != nil {
			log.Errorf("Faied to kick user afeter ban: %v", errR)
			return errR
		}
		log.Debugf("RCON response: %s", resp)
	}
	*pi = foundPI

	return nil
}

// SetSteam is used to associate a discord user with either steam id. This is used
// instead of requiring users to link their steam account to discord itself. It also
// means the bot does not require more priviledges intents.
func (g gbans) SetSteam(args action.SetSteamIDRequest) (bool, error) {
	sid, err := steamid.ResolveSID64(g.ctx, string(args.Target))
	if err != nil || !sid.Valid() {
		return false, consts.ErrInvalidSID
	}
	p := model.NewPerson(sid)
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

// Say is used to send a message to the server via sm_say
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

// CSay is used to send a centered message to the server via sm_csay
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

// PSay is used to send a private message to a player
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

// FilterAdd creates a new chat filter using a regex pattern
func (g gbans) FilterAdd(args action.FilterAddRequest) (model.Filter, error) {
	re, err := regexp.Compile(args.Filter)
	if err != nil {
		return model.Filter{}, errors.Wrapf(err, "Invalid regex format")
	}
	filter := model.Filter{Pattern: re, CreatedOn: config.Now()}
	if errSave := g.db.SaveFilter(g.ctx, &filter); errSave != nil {
		if errSave == store.ErrDuplicate {
			return filter, store.ErrDuplicate
		}
		log.Errorf("Error saving filter word: %v", err)
		return filter, consts.ErrInternal
	}
	return filter, nil
}

// FilterDel removed and existing chat filter
func (g gbans) FilterDel(ctx context.Context, args action.FilterDelRequest) (bool, error) {
	var filter model.Filter
	if err := g.db.GetFilterByID(ctx, args.FilterID, &filter); err != nil {
		return false, err
	}
	if err2 := g.db.DropFilter(ctx, &filter); err2 != nil {
		return false, err2
	}
	return true, nil
}

// FilterCheck can be used to check if a phrase will match any filters
func (g gbans) FilterCheck(args action.FilterCheckRequest) []model.Filter {
	if args.Message == "" {
		return nil
	}
	words := strings.Split(strings.ToLower(args.Message), " ")
	wordFiltersMu.RLock()
	defer wordFiltersMu.RUnlock()
	var found []model.Filter
	for _, filter := range wordFilters {
		for _, word := range words {
			if filter.Match(word) {
				found = append(found, filter)
			}
		}
	}
	return found
}

// ContainsFilteredWord checks to see if the body of text contains a known filtered word
// It will only return the first matched filter found.
func (g gbans) ContainsFilteredWord(body string) (bool, model.Filter) {
	if body == "" {
		return false, model.Filter{}
	}
	words := strings.Split(strings.ToLower(body), " ")
	wordFiltersMu.RLock()
	defer wordFiltersMu.RUnlock()
	for _, filter := range wordFilters {
		for _, word := range words {
			if filter.Match(word) {
				return true, filter
			}
		}
	}
	return false, model.Filter{}
}

// PersonBySID fetches the person from the database, updating the PlayerSummary if it out of date
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

// ResolveSID is just a simple helper for calling steamid.ResolveSID64
func (g gbans) ResolveSID(sidStr string) (steamid.SID64, error) {
	c, cancel := context.WithTimeout(g.ctx, time.Second*5)
	defer cancel()
	return steamid.ResolveSID64(c, sidStr)
}
