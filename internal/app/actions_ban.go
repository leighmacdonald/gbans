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
	"strconv"
	"time"
)

func newBaseBanOpts(source model.SteamIDProvider, target model.StringSID, duration model.Duration,
	reason model.Reason, reasonText string, modNote string, origin model.Origin, opts *model.BaseBanOpts) error {
	sourceSid, errSource := source.SID64()
	if errSource != nil {
		return errors.Wrapf(errSource, "Failed to parse source id")
	}
	var targetSid = steamid.SID64(0)
	if string(target) != "0" {
		newTargetSid, errTargetSid := target.SID64()
		if errTargetSid != nil {
			return errors.New("Invalid target id")
		}
		targetSid = newTargetSid
	}
	durationActual, errDuration := duration.Value()
	if errDuration != nil {
		return errors.Wrapf(errDuration, "Unable to determine expiration")
	}
	if reason == model.Custom && reasonText == "" {
		return errors.New("Custom reason cannot be empty")
	}
	opts.TargetId = targetSid
	opts.SourceId = sourceSid
	opts.Duration = durationActual
	opts.ModNote = modNote
	opts.Reason = reason
	opts.ReasonText = reasonText
	opts.Origin = origin
	opts.Deleted = false
	opts.BanType = model.Banned
	opts.IsEnabled = true
	return nil
}

func NewBanSteam(source model.SteamIDProvider, target model.StringSID, duration model.Duration,
	reason model.Reason, reasonText string, modNote string, origin model.Origin, reportId int64, banSteam *model.BanSteam) error {
	var opts model.BanSteamOpts
	errBaseOpts := newBaseBanOpts(source, target, duration, reason, reasonText, modNote, origin, &opts.BaseBanOpts)
	if errBaseOpts != nil {
		return errBaseOpts
	}
	if reportId < 0 {
		return errors.New("Invalid report ID")
	}
	opts.ReportId = int(reportId)
	banSteam.Apply(opts)
	banSteam.ReportId = opts.ReportId
	banSteam.BanID = opts.BanId
	return nil
}

// BanSteam will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of config.DefaultExpiration() will be used.
func BanSteam(ctx context.Context, database store.Store, banSteam *model.BanSteam, botSendMessageChan chan discordPayload) error {
	if !banSteam.TargetId.Valid() {
		return errors.Wrap(consts.ErrInvalidSID, "Invalid target steam id")
	}
	existing := model.NewBannedPerson()
	errGetExistingBan := database.GetBanBySteamID(ctx, banSteam.TargetId, &existing, false)
	if existing.Ban.BanID > 0 && existing.Ban.BanType == model.Banned {
		return store.ErrDuplicate
	}
	if errGetExistingBan != nil && errGetExistingBan != store.ErrNoResult {
		return errors.Wrapf(errGetExistingBan, "Failed to get ban")
	}

	if errSave := database.SaveBan(ctx, banSteam); errSave != nil {
		return errors.Wrap(errSave, "Failed to save ban")
	}
	go func(payloadChan chan discordPayload) {
		banNotice := &discordgo.MessageEmbed{
			URL:   fmt.Sprintf("https://steamcommunity.com/profiles/%d", banSteam.TargetId),
			Type:  discordgo.EmbedTypeRich,
			Title: fmt.Sprintf("User Banned (#%d)", banSteam.BanID),
			Color: 10038562,
		}
		addFieldsSteamID(banNotice, banSteam.TargetId)
		expIn := "Permanent"
		expAt := "Permanent"
		if banSteam.ValidUntil.Year()-time.Now().Year() < 5 {
			expIn = config.FmtDuration(banSteam.ValidUntil)
			expAt = config.FmtTimeShort(banSteam.ValidUntil)
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
		sendDiscordPayload(payloadChan, discordPayload{channelId: config.Discord.PublicLogChannelId, embed: banNotice})
	}(botSendMessageChan)
	// TODO mute player currently in-game w/o kicking
	if banSteam.BanType == model.Banned {
		if errKick := Kick(ctx, database, model.System,
			model.StringSID(banSteam.TargetId.String()),
			model.StringSID(banSteam.SourceId.String()),
			banSteam.Reason, nil); errKick != nil {
			log.Errorf("failed to kick player: %v", errKick)
		}
	}

	return nil
}

func NewBanASN(source model.SteamIDProvider, target model.StringSID, duration model.Duration,
	reason model.Reason, reasonText string, modNote string, origin model.Origin, ASNum int64, banASN *model.BanASN) error {
	var opts model.BanASNOpts
	errBaseOpts := newBaseBanOpts(source, target, duration, reason, reasonText, modNote, origin, &opts.BaseBanOpts)
	if errBaseOpts != nil {
		return errBaseOpts
	}
	// Valid public ASN ranges
	// https://www.iana.org/assignments/as-numbers/as-numbers.xhtml
	ranges := []struct {
		start int64
		end   int64
	}{
		{1, 23455},
		{23457, 64495},
		{131072, 4199999999},
	}
	ok := false
	for _, r := range ranges {
		if ASNum >= r.start && ASNum <= r.end {
			ok = true
			break
		}
	}
	if !ok {
		return errors.New("Invalid asn")
	}
	opts.ASNum = ASNum
	return banASN.Apply(opts)
}

// BanASN will ban all network ranges associated with the requested ASN
func BanASN(ctx context.Context, database store.Store, banASN *model.BanASN) error {
	var existing model.BanASN
	if errGetExistingBan := database.GetBanASN(ctx, banASN.ASNum, &existing); errGetExistingBan != nil {
		if !errors.Is(errGetExistingBan, store.ErrNoResult) {
			return errors.Wrapf(errGetExistingBan, "Failed trying to fetch existing asn ban")
		}
	}
	if errSave := database.SaveBanASN(ctx, banASN); errSave != nil {
		return errSave
	}
	// TODO Kick all current players matching
	return nil
}

func NewBanCIDR(source model.SteamIDProvider, target model.StringSID, duration model.Duration,
	reason model.Reason, reasonText string, modNote string, origin model.Origin, cidr string, banCIDR *model.BanCIDR) error {
	var opts model.BanCIDROpts
	if errBaseOpts := newBaseBanOpts(source, target, duration, reason, reasonText, modNote, origin, &opts.BaseBanOpts); errBaseOpts != nil {
		return errBaseOpts
	}
	_, parsedNetwork, errParse := net.ParseCIDR(cidr)
	if errParse != nil {
		return errors.Wrap(errParse, "Failed to parse cidr address")
	}
	opts.CIDR = parsedNetwork
	return banCIDR.Apply(opts)
}

// BanCIDR adds a new network to the banned network list. It will accept any Valid CIDR format.
// It accepts an optional steamid to associate a particular user with the network ban. Any active players
// that fall within the range will be kicked immediately.
// If duration is 0, the value of config.DefaultExpiration() will be used.
func BanCIDR(ctx context.Context, database store.Store, banNet *model.BanCIDR) error {
	// TODO
	//_, err2 := store.GetBanNetByAddress(ctx, net.ParseIP(cidrStr))
	//if err2 != nil && err2 != store.ErrNoResult {
	//	return "", errCommandFailed
	//}
	//if err2 == nil {
	//	return "", consts.ErrDuplicateBan
	//}
	if banNet.CIDR == nil {
		return errors.New("CIDR unset")
	}
	if errSaveBanNet := database.SaveBanNet(ctx, banNet); errSaveBanNet != nil {
		return errSaveBanNet
	}
	go func() {
		var playerInfo model.PlayerInfo
		if errFindPI := FindPlayerByCIDR(ctx, database, banNet.CIDR, &playerInfo); errFindPI != nil {
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

func NewBanSteamGroup(source model.SteamIDProvider, target model.StringSID, duration model.Duration,
	reason model.Reason, reasonText string, modNote string, origin model.Origin, groupId steamid.GID, groupName string,
	banGroup *model.BanGroup) error {
	var opts model.BanSteamGroupOpts
	errBaseOpts := newBaseBanOpts(source, target, duration, reason, reasonText, modNote, origin, &opts.BaseBanOpts)
	if errBaseOpts != nil {
		return errBaseOpts
	}
	// TODO validate gid here w/fetch?
	opts.GroupId = groupId
	opts.GroupName = groupName
	return banGroup.Apply(opts)
}

func BanSteamGroup(ctx context.Context, database store.Store, banGroup *model.BanGroup) error {
	members, membersErr := steamweb.GetGroupMembers(ctx, banGroup.GroupId)
	if membersErr != nil {
		return errors.Wrapf(membersErr, "Failed to validate group")
	}
	if errSaveBanGroup := database.SaveBanGroup(ctx, banGroup); errSaveBanGroup != nil {
		return errSaveBanGroup
	}
	log.WithFields(log.Fields{
		"gid":     banGroup.GroupId.String(),
		"members": len(members),
	}).Infof("Steam group banned")
	return nil
}

// Unban will set the current ban to now, making it expired.
// Returns true, nil if the ban exists, and was successfully banned.
// Returns false, nil if the ban does not exist.
func Unban(ctx context.Context, database store.Store, target steamid.SID64, reason string) (bool, error) {
	bannedPerson := model.NewBannedPerson()
	errGetBan := database.GetBanBySteamID(ctx, target, &bannedPerson, false)
	if errGetBan != nil {
		if errGetBan == store.ErrNoResult {
			return false, nil
		}
		return false, errGetBan
	}
	bannedPerson.Ban.Deleted = true
	bannedPerson.Ban.UnbanReasonText = reason
	if errSaveBan := database.SaveBan(ctx, &bannedPerson.Ban); errSaveBan != nil {
		return false, errors.Wrapf(errSaveBan, "Failed to save unban")
	}
	log.Infof("Player unbanned: %v", target)
	go func() {
		unbanNotice := &discordgo.MessageEmbed{
			URL:   fmt.Sprintf("https://steamcommunity.com/profiles/%d", bannedPerson.Ban.TargetId),
			Type:  discordgo.EmbedTypeRich,
			Title: fmt.Sprintf("User Unbanned: %s (#%d)", bannedPerson.Person.PersonaName, bannedPerson.Ban.BanID),
			Color: int(green),
		}
		addFieldsSteamID(unbanNotice, bannedPerson.Person.SteamID)
		addField(unbanNotice, "Reason", reason)
		sendDiscordPayload(discordSendMsg, discordPayload{
			channelId: config.Discord.ModLogChannelId,
			embed:     unbanNotice,
		})
	}()
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
