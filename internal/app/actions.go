package app

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
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
func Unban(db store.Store, target steamid.SID64) (bool, error) {
	b := model.NewBannedPerson()
	err := db.GetBanBySteamID(ctx, target, false, &b)
	if err != nil {
		if err == store.ErrNoResult {
			return false, nil
		}
		return false, err
	}
	b.Ban.ValidUntil = config.Now()
	if err2 := db.SaveBan(ctx, &b.Ban); err2 != nil {
		return false, errors.Wrapf(err2, "Failed to save unban")
	}
	log.Infof("Player unbanned: %v", target)
	return true, nil
}

// UnbanASN will remove an existing ASN ban
func UnbanASN(ctx context.Context, db store.Store, asnNum string) (bool, error) {
	asNum, errConv := strconv.ParseInt(asnNum, 10, 64)
	if errConv != nil {
		return false, errConv
	}
	var ba model.BanASN
	if err := db.GetBanASN(ctx, asNum, &ba); err != nil {
		return false, err
	}
	if errDrop := db.DropBanASN(ctx, &ba); errDrop != nil {
		log.Errorf("Failed to drop ASN ban: %v", errDrop)
		return false, errDrop
	}
	log.Infof("ASN unbanned: %d", asNum)
	return true, nil
}

type banOpts struct {
	target   model.Target
	author   model.Target
	duration model.Duration
	banType  model.BanType
	reason   string
	origin   model.Origin
}

// Ban will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of config.DefaultExpiration() will be used.
func Ban(db store.Store, opts banOpts, ban *model.Ban, botSendMessageChan chan discordPayload) error {
	existing := model.NewBannedPerson()
	sid, errSid := opts.target.SID64()
	if errSid != nil {
		return errSid
	}
	aid, errAid := opts.author.SID64()
	if errAid != nil {
		return errAid
	}
	err := db.GetBanBySteamID(ctx, sid, false, &existing)
	if existing.Ban.BanID > 0 && existing.Ban.BanType == model.Banned {
		return store.ErrDuplicate
	}
	if err != nil && err != store.ErrNoResult {
		return errors.Wrapf(err, "Failed to get ban")
	}
	until := config.DefaultExpiration()
	duration, errDuration := opts.duration.Value()
	if errDuration != nil {
		return errDuration
	}
	if duration.Seconds() != 0 {
		until = config.Now().Add(duration)
	}
	ban.SteamID = sid
	ban.AuthorID = aid
	ban.BanType = opts.banType
	ban.Reason = model.Custom
	ban.ReasonText = opts.reason
	ban.Note = ""
	ban.ValidUntil = until
	ban.Source = opts.origin
	ban.CreatedOn = config.Now()
	ban.UpdatedOn = config.Now()

	if err2 := db.SaveBan(ctx, ban); err2 != nil {
		return err2
	}
	go func(dp chan discordPayload) {
		banNotice := &discordgo.MessageEmbed{
			URL:   fmt.Sprintf("https://steamcommunity.com/profiles/%d", ban.SteamID),
			Type:  discordgo.EmbedTypeRich,
			Title: fmt.Sprintf("User Banned (#%d)", ban.BanID),
			Color: 10038562,
		}
		banNotice.Fields = append(banNotice.Fields, &discordgo.MessageEmbedField{
			Name:   "STEAM",
			Value:  string(steamid.SID64ToSID(ban.SteamID)),
			Inline: true,
		})
		banNotice.Fields = append(banNotice.Fields, &discordgo.MessageEmbedField{
			Name:   "STEAM3",
			Value:  string(steamid.SID64ToSID3(ban.SteamID)),
			Inline: true,
		})
		banNotice.Fields = append(banNotice.Fields, &discordgo.MessageEmbedField{
			Name:   "SID64",
			Value:  ban.SteamID.String(),
			Inline: true,
		})
		banNotice.Fields = append(banNotice.Fields, &discordgo.MessageEmbedField{
			Name:   "Expires In",
			Value:  config.FmtDuration(ban.ValidUntil),
			Inline: false,
		})
		banNotice.Fields = append(banNotice.Fields, &discordgo.MessageEmbedField{
			Name:   "Expires At",
			Value:  config.FmtTimeShort(ban.ValidUntil),
			Inline: false,
		})
		if config.Discord.PublicLogChannelEnable {
			select {
			case discordSendMsg <- discordPayload{channelId: config.Discord.PublicLogChannelId, message: banNotice}:
			default:
				log.Warnf("Cannot send discord payload, channel full")
			}
		}
	}(botSendMessageChan)

	if errKick := Kick(db, model.System, opts.target, opts.author, opts.reason, nil); errKick != nil {
		log.Errorf("failed to kick player: %v", errKick)
	}

	return nil
}

type banASNOpts struct {
	banOpts
	asNum int64
}

// BanASN will ban all network ranges associated with the requested ASN
func BanASN(db store.Store, opts banASNOpts, banASN *model.BanASN) error {
	until := config.DefaultExpiration()
	sid, errSid := opts.target.SID64()
	if errSid != nil {
		return errSid
	}
	aid, errAid := opts.author.SID64()
	if errAid != nil {
		return errAid
	}
	duration, errDur := opts.duration.Value()
	if errDur != nil {
		return errDur
	}
	if duration != 0 {
		until = config.Now().Add(duration)
	}
	banASN.Origin = opts.origin
	banASN.TargetID = sid
	banASN.AuthorID = aid
	banASN.ValidUntil = until
	banASN.Reason = opts.reason
	banASN.ASNum = opts.asNum
	if errSave := db.SaveBanASN(context.TODO(), banASN); errSave != nil {
		return errSave
	}
	// TODO Kick all current players matching
	return nil
}

type banNetworkOpts struct {
	banOpts
	cidr string
}

// BanNetwork adds a new network to the banned network list. It will accept any Valid CIDR format.
// It accepts an optional steamid to associate a particular user with the network ban. Any active players
// that fall within the range will be kicked immediately.
// If duration is 0, the value of config.DefaultExpiration() will be used.
func BanNetwork(db store.Store, opts banNetworkOpts, banNet *model.BanNet) error {
	until := config.DefaultExpiration()
	sid, errSid := opts.target.SID64()
	if errSid != nil {
		return errSid
	}
	aid, errAid := opts.author.SID64()
	if errAid != nil {
		return errAid
	}
	duration, errDur := opts.duration.Value()
	if errDur != nil {
		return errDur
	}
	if duration.Seconds() != 0 {
		until = config.Now().Add(duration)
	}
	_, cidr, errCidr := net.ParseCIDR(opts.cidr)
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

	banNet.SteamID = sid
	banNet.AuthorID = aid
	banNet.CIDR = cidr
	banNet.Source = model.System
	banNet.Reason = opts.reason
	banNet.CreatedOn = config.Now()
	banNet.UpdatedOn = config.Now()
	banNet.ValidUntil = until

	if err := db.SaveBanNet(ctx, banNet); err != nil {
		return err
	}
	go func() {
		var pi model.PlayerInfo
		if errPI := FindPlayerByCIDR(db, cidr, &pi); errPI != nil {
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
func Kick(db store.Store, origin model.Origin, target model.Target, author model.Target, reason string, pi *model.PlayerInfo) error {
	aid, errAid := author.SID64()
	if errAid != nil {
		return errAid
	}
	// kick the user if they currently are playing on a server
	var foundPI model.PlayerInfo
	if errF := Find(db, target, "", &foundPI); errF != nil {
		return errF
	}
	if foundPI.Valid && foundPI.InGame {
		resp, errR := query.ExecRCON(*foundPI.Server, fmt.Sprintf("sm_kick #%d %s", foundPI.Player.UserID, reason))
		if errR != nil {
			log.Errorf("Faied to kick user afeter ban: %v", errR)
			return errR
		}
		log.Debugf("RCON response: %s", resp)
	}
	if pi != nil {
		*pi = foundPI
	}
	log.WithFields(log.Fields{"origin": origin, "target": target, "author": aid.String()}).
		Infof("User kicked")
	return nil
}

// SetSteam is used to associate a discord user with either steam id. This is used
// instead of requiring users to link their steam account to discord itself. It also
// means the bot does not require more privileged intents.
func SetSteam(db store.Store, sid steamid.SID64, discordId string) error {
	p := model.NewPerson(sid)
	if errP := db.GetOrCreatePersonBySteamID(ctx, sid, &p); errP != nil || !sid.Valid() {
		return consts.ErrInvalidSID
	}
	if (p.DiscordID) != "" {
		return errors.Errorf("Discord account already linked to steam account: %d", p.SteamID.Int64())
	}
	p.DiscordID = discordId
	if errS := db.SavePerson(ctx, &p); errS != nil {
		return consts.ErrInternal
	}
	log.WithFields(log.Fields{"sid": sid, "did": discordId}).Infof("Discord steamid set")
	return nil
}

// Say is used to send a message to the server via sm_say
func Say(db store.Store, author steamid.SID64, serverName string, message string) error {
	var server model.Server
	if err := db.GetServerByName(ctx, serverName, &server); err != nil {
		return errors.Errorf("Failed to fetch server: %s", serverName)
	}
	msg := fmt.Sprintf(`sm_say %s`, message)
	resp, err2 := query.ExecRCON(server, msg)
	if err2 != nil {
		return err2
	}
	rp := strings.Split(resp, "\n")
	if len(rp) < 2 {
		return errors.Errorf("Invalid response")
	}
	log.WithFields(log.Fields{"author": author, "server": serverName, "msg": message}).
		Infof("Server message sent")
	return nil
}

// CSay is used to send a centered message to the server via sm_csay
func CSay(db store.Store, author steamid.SID64, serverName string, message string) error {
	var (
		servers []model.Server
		err     error
	)
	if serverName == "*" {
		servers, err = db.GetServers(ctx, false)
		if err != nil {
			return errors.Wrapf(err, "Failed to fetch servers")
		}
	} else {
		var server model.Server
		if errS := db.GetServerByName(ctx, serverName, &server); errS != nil {
			return errors.Wrapf(errS, "Failed to fetch server: %s", serverName)
		}
		servers = append(servers, server)
	}
	msg := fmt.Sprintf(`sm_csay %s`, message)
	_ = query.RCON(ctx, servers, msg)
	log.WithFields(log.Fields{"author": author, "server": serverName, "msg": message}).
		Infof("Server center message sent")
	return nil
}

// PSay is used to send a private message to a player
func PSay(db store.Store, author steamid.SID64, target model.Target, message string) error {
	var pi model.PlayerInfo
	_ = Find(db, target, "", &pi)
	if !pi.Valid || !pi.InGame {
		return consts.ErrUnknownID
	}
	msg := fmt.Sprintf(`sm_psay %d "%s"`, pi.Player.UserID, message)
	_, err := query.ExecRCON(*pi.Server, msg)
	if err != nil {
		return errors.Errorf("Failed to exec psay command: %v", err)
	}
	log.WithFields(log.Fields{"author": author, "server": pi.Server.ServerName, "msg": message, "target": pi.Player.SID}).
		Infof("Private message sent")
	return nil
}

// FilterAdd creates a new chat filter using a regex pattern
func FilterAdd(db store.Store, pattern string) (model.Filter, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return model.Filter{}, errors.Wrapf(err, "Invalid regex format")
	}
	filter := model.Filter{Pattern: re, CreatedOn: config.Now()}
	if errSave := db.SaveFilter(ctx, &filter); errSave != nil {
		if errSave == store.ErrDuplicate {
			return filter, store.ErrDuplicate
		}
		log.Errorf("Error saving filter word: %v", err)
		return filter, consts.ErrInternal
	}
	return filter, nil
}

// FilterDel removed and existing chat filter
func FilterDel(ctx context.Context, db store.Store, filterId int) (bool, error) {
	var filter model.Filter
	if err := db.GetFilterByID(ctx, filterId, &filter); err != nil {
		return false, err
	}
	if err2 := db.DropFilter(ctx, &filter); err2 != nil {
		return false, err2
	}
	return true, nil
}

// FilterCheck can be used to check if a phrase will match any filters
func FilterCheck(message string) []model.Filter {
	if message == "" {
		return nil
	}
	words := strings.Split(strings.ToLower(message), " ")
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
func ContainsFilteredWord(body string) (bool, model.Filter) {
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
func PersonBySID(db store.Store, sid steamid.SID64, ipAddr string, p *model.Person) error {
	if err := db.GetPersonBySteamID(ctx, sid, p); err != nil && err != store.ErrNoResult {
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
		p.UpdatedOn = config.Now()
		if err := db.SavePerson(ctx, p); err != nil {
			log.Errorf("Failed to save updated profile: %v", errW)
			return nil
		}
		if err := db.GetPersonBySteamID(ctx, sid, p); err != nil && err != store.ErrNoResult {
			return err
		}
	}
	return nil
}

// ResolveSID is just a simple helper for calling steamid.ResolveSID64
func ResolveSID(sidStr string) (steamid.SID64, error) {
	c, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	return steamid.ResolveSID64(c, sidStr)
}
