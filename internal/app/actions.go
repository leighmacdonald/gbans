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
	"github.com/leighmacdonald/gbans/pkg/util"
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
func Unban(ctx context.Context, database store.Store, target steamid.SID64) (bool, error) {
	bannedPerson := model.NewBannedPerson()
	errGetBan := database.GetBanBySteamID(ctx, target, false, &bannedPerson)
	if errGetBan != nil {
		if errGetBan == store.ErrNoResult {
			return false, nil
		}
		return false, errGetBan
	}
	bannedPerson.Ban.ValidUntil = config.Now()
	if errSaveBan := database.SaveBan(ctx, &bannedPerson.Ban); errSaveBan != nil {
		return false, errors.Wrapf(errSaveBan, "Failed to save unban")
	}
	log.Infof("Player unbanned: %v", target)
	return true, nil
}

// UnbanASN will remove an existing ASN ban
func UnbanASN(ctx context.Context, database store.Store, asnNum string) (bool, error) {
	asNum, errConv := strconv.ParseInt(asnNum, 10, 64)
	if errConv != nil {
		return false, errConv
	}
	var banASN model.BanASN
	if errGetBanASN := database.GetBanASN(ctx, asNum, &banASN); errGetBanASN != nil {
		return false, errGetBanASN
	}
	if errDrop := database.DropBanASN(ctx, &banASN); errDrop != nil {
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
	modNote  string
}

// Ban will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of config.DefaultExpiration() will be used.
func Ban(ctx context.Context, database store.Store, opts banOpts, ban *model.Ban, botSendMessageChan chan discordPayload) error {
	existing := model.NewBannedPerson()
	targetSid64, errSid := opts.target.SID64()
	if errSid != nil {
		return errSid
	}
	authorSid64, errAid := opts.author.SID64()
	if errAid != nil {
		return errAid
	}
	errGetExistingBan := database.GetBanBySteamID(ctx, targetSid64, false, &existing)
	if existing.Ban.BanID > 0 && existing.Ban.BanType == model.Banned {
		return store.ErrDuplicate
	}
	if errGetExistingBan != nil && errGetExistingBan != store.ErrNoResult {
		return errors.Wrapf(errGetExistingBan, "Failed to get ban")
	}
	until := config.DefaultExpiration()
	duration, errDuration := opts.duration.Value()
	if errDuration != nil {
		return errDuration
	}
	if duration.Seconds() != 0 {
		until = config.Now().Add(duration)
	}
	ban.SteamID = targetSid64
	ban.AuthorID = authorSid64
	ban.BanType = opts.banType
	ban.Reason = model.Custom
	ban.ReasonText = opts.reason
	ban.Note = opts.modNote
	ban.ValidUntil = until
	ban.Source = opts.origin
	ban.CreatedOn = config.Now()
	ban.UpdatedOn = config.Now()

	if errSave := database.SaveBan(ctx, ban); errSave != nil {
		return errSave
	}
	go func(payload chan discordPayload) {
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
		expIn := "Permanent"
		expAt := "Permanent"
		if ban.ValidUntil.Year()-time.Now().Year() < 5 {
			expIn = config.FmtDuration(ban.ValidUntil)
			expAt = config.FmtTimeShort(ban.ValidUntil)
		}
		banNotice.Fields = append(banNotice.Fields, &discordgo.MessageEmbedField{
			Name:   "Expires In",
			Value:  expIn,
			Inline: false,
		})
		banNotice.Fields = append(banNotice.Fields, &discordgo.MessageEmbedField{
			Name:   "Expires At",
			Value:  expAt,
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

	if errKick := Kick(ctx, database, model.System, opts.target, opts.author, opts.reason, nil); errKick != nil {
		log.Errorf("failed to kick player: %v", errKick)
	}

	return nil
}

type banASNOpts struct {
	banOpts
	asNum int64
}

// BanASN will ban all network ranges associated with the requested ASN
func BanASN(database store.Store, opts banASNOpts, banASN *model.BanASN) error {
	until := config.DefaultExpiration()
	targetSid64, errSid := opts.target.SID64()
	if errSid != nil {
		return errSid
	}
	authorSid64, errAid := opts.author.SID64()
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
	banASN.TargetID = targetSid64
	banASN.AuthorID = authorSid64
	banASN.ValidUntil = until
	banASN.Reason = opts.reason
	banASN.ASNum = opts.asNum
	if errSave := database.SaveBanASN(context.TODO(), banASN); errSave != nil {
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
func BanNetwork(ctx context.Context, database store.Store, opts banNetworkOpts, banNet *model.BanNet) error {
	until := config.DefaultExpiration()
	targetSid64, errSid := opts.target.SID64()
	if errSid != nil {
		return errSid
	}
	authorSid64, errAid := opts.author.SID64()
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
	_, network, errParseCIDR := net.ParseCIDR(opts.cidr)
	if errParseCIDR != nil {
		return errors.Wrapf(errParseCIDR, "Failed to parse CIDR address")
	}
	// TODO
	//_, err2 := store.GetBanNet(ctx, net.ParseIP(cidrStr))
	//if err2 != nil && err2 != store.ErrNoResult {
	//	return "", errCommandFailed
	//}
	//if err2 == nil {
	//	return "", consts.ErrDuplicateBan
	//}

	banNet.SteamID = targetSid64
	banNet.AuthorID = authorSid64
	banNet.CIDR = network
	banNet.Source = model.System
	banNet.Reason = opts.reason
	banNet.CreatedOn = config.Now()
	banNet.UpdatedOn = config.Now()
	banNet.ValidUntil = until

	if errSaveBanNet := database.SaveBanNet(ctx, banNet); errSaveBanNet != nil {
		return errSaveBanNet
	}
	go func() {
		var playerInfo model.PlayerInfo
		if errFindPI := FindPlayerByCIDR(ctx, database, network, &playerInfo); errFindPI != nil {
			return
		}
		if playerInfo.Player != nil && playerInfo.Server != nil {
			_, errExecRCON := query.ExecRCON(ctx, *playerInfo.Server,
				fmt.Sprintf(`gb_kick "#%s" %s`, string(steamid.SID64ToSID(playerInfo.Player.SID)), banNet.Reason))
			if errExecRCON != nil {
				log.Errorf("Failed to query for ban request: %v", errExecRCON)
				return
			}
		}
	}()

	return nil
}

// Kick will kick the steam id from whatever server it is connected to.
func Kick(ctx context.Context, database store.Store, origin model.Origin, target model.Target, author model.Target, reason string, playerInfo *model.PlayerInfo) error {
	authorSid64, errAid := author.SID64()
	if errAid != nil {
		return errAid
	}
	// kick the user if they currently are playing on a server
	var foundPI model.PlayerInfo
	if errFind := Find(ctx, database, target, "", &foundPI); errFind != nil {
		return errFind
	}
	if foundPI.Valid && foundPI.InGame {
		rconResponse, errExecRCON := query.ExecRCON(ctx, *foundPI.Server, fmt.Sprintf("sm_kick #%d %s", foundPI.Player.UserID, reason))
		if errExecRCON != nil {
			log.Errorf("Faied to kick user afeter ban: %v", errExecRCON)
			return errExecRCON
		}
		log.Debugf("RCON response: %s", rconResponse)
	}
	if playerInfo != nil {
		*playerInfo = foundPI
	}
	log.WithFields(log.Fields{"origin": origin, "target": target, "author": util.SanitizeLog(authorSid64.String())}).
		Infof("User kicked")
	return nil
}

// SetSteam is used to associate a discord user with either steam id. This is used
// instead of requiring users to link their steam account to discord itself. It also
// means the bot does not require more privileged intents.
func SetSteam(ctx context.Context, database store.Store, sid64 steamid.SID64, discordId string) error {
	newPerson := model.NewPerson(sid64)
	if errGetPerson := database.GetOrCreatePersonBySteamID(ctx, sid64, &newPerson); errGetPerson != nil || !sid64.Valid() {
		return consts.ErrInvalidSID
	}
	if (newPerson.DiscordID) != "" {
		return errors.Errorf("Discord account already linked to steam account: %d", newPerson.SteamID.Int64())
	}
	newPerson.DiscordID = discordId
	if errSavePerson := database.SavePerson(ctx, &newPerson); errSavePerson != nil {
		return consts.ErrInternal
	}
	log.WithFields(log.Fields{"sid64": sid64, "discordId": discordId}).Infof("Discord steamid set")
	return nil
}

// Say is used to send a message to the server via sm_say
func Say(ctx context.Context, database store.Store, author steamid.SID64, serverName string, message string) error {
	var server model.Server
	if errGetServer := database.GetServerByName(ctx, serverName, &server); errGetServer != nil {
		return errors.Errorf("Failed to fetch server: %s", serverName)
	}
	msg := fmt.Sprintf(`sm_say %s`, message)
	rconResponse, errExecRCON := query.ExecRCON(ctx, server, msg)
	if errExecRCON != nil {
		return errExecRCON
	}
	responsePieces := strings.Split(rconResponse, "\n")
	if len(responsePieces) < 2 {
		return errors.Errorf("Invalid response")
	}
	log.WithFields(log.Fields{"author": author, "server": serverName, "msg": message}).
		Infof("Server message sent")
	return nil
}

// CSay is used to send a centered message to the server via sm_csay
func CSay(ctx context.Context, database store.Store, author steamid.SID64, serverName string, message string) error {
	var (
		servers []model.Server
		err     error
	)
	if serverName == "*" {
		servers, err = database.GetServers(ctx, false)
		if err != nil {
			return errors.Wrapf(err, "Failed to fetch servers")
		}
	} else {
		var server model.Server
		if errS := database.GetServerByName(ctx, serverName, &server); errS != nil {
			return errors.Wrapf(errS, "Failed to fetch server: %s", serverName)
		}
		servers = append(servers, server)
	}
	msg := fmt.Sprintf(`sm_csay %s`, message)
	// TODO check response
	_ = query.RCON(ctx, servers, msg)
	log.WithFields(log.Fields{"author": author, "server": serverName, "msg": message}).
		Infof("Server center message sent")
	return nil
}

// PSay is used to send a private message to a player
func PSay(ctx context.Context, database store.Store, author steamid.SID64, target model.Target, message string) error {
	var playerInfo model.PlayerInfo
	// TODO check resp
	_ = Find(ctx, database, target, "", &playerInfo)
	if !playerInfo.Valid || !playerInfo.InGame {
		return consts.ErrUnknownID
	}
	msg := fmt.Sprintf(`sm_psay %d "%s"`, playerInfo.Player.UserID, message)
	_, errExecRCON := query.ExecRCON(ctx, *playerInfo.Server, msg)
	if errExecRCON != nil {
		return errors.Errorf("Failed to exec psay command: %v", errExecRCON)
	}
	log.WithFields(log.Fields{"author": author, "server": playerInfo.Server.ServerNameShort, "msg": message, "target": playerInfo.Player.SID}).
		Infof("Private message sent")
	return nil
}

// FilterAdd creates a new chat filter using a regex pattern
func FilterAdd(ctx context.Context, database store.Store, pattern string) (model.Filter, error) {
	rx, errCompile := regexp.Compile(pattern)
	if errCompile != nil {
		return model.Filter{}, errors.Wrapf(errCompile, "Invalid regex format")
	}
	filter := model.Filter{Pattern: rx, CreatedOn: config.Now()}
	if errSave := database.SaveFilter(ctx, &filter); errSave != nil {
		if errSave == store.ErrDuplicate {
			return filter, store.ErrDuplicate
		}
		log.Errorf("Error saving filter word: %v", errCompile)
		return filter, consts.ErrInternal
	}
	return filter, nil
}

// FilterDel removed and existing chat filter
func FilterDel(ctx context.Context, database store.Store, filterId int) (bool, error) {
	var filter model.Filter
	if errGetFilter := database.GetFilterByID(ctx, filterId, &filter); errGetFilter != nil {
		return false, errGetFilter
	}
	if errDropFilter := database.DropFilter(ctx, &filter); errDropFilter != nil {
		return false, errDropFilter
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
func PersonBySID(ctx context.Context, database store.Store, sid steamid.SID64, ipAddr string, person *model.Person) error {
	if errGetPerson := database.GetPersonBySteamID(ctx, sid, person); errGetPerson != nil && errGetPerson != store.ErrNoResult {
		return errGetPerson
	}
	if person.UpdatedOn == person.CreatedOn || time.Since(person.UpdatedOn) > 15*time.Second {
		summary, errSummary := steamweb.PlayerSummaries(steamid.Collection{person.SteamID})
		if errSummary != nil || len(summary) != 1 {
			log.Errorf("Failed to fetch updated profile: %v", errSummary)
			return nil
		}
		var sum = summary[0]
		person.PlayerSummary = &sum
		person.UpdatedOn = config.Now()
		if errSave := database.SavePerson(ctx, person); errSave != nil {
			log.Errorf("Failed to save updated profile: %v", errSummary)
			return nil
		}
		if errGetPersonBySid64 := database.GetPersonBySteamID(ctx, sid, person); errGetPersonBySid64 != nil && errGetPersonBySid64 != store.ErrNoResult {
			return errGetPersonBySid64
		}
	}
	return nil
}

// ResolveSID is just a simple helper for calling steamid.ResolveSID64 with a timeout
func ResolveSID(ctx context.Context, sidStr string) (steamid.SID64, error) {
	localCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	return steamid.ResolveSID64(localCtx, sidStr)
}
