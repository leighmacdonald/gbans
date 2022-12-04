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
)

func newBaseBanOpts(source model.SteamIDProvider, target model.StringSID, duration model.Duration,
	reason model.Reason, reasonText string, modNote string, origin model.Origin,
	banType model.BanType, opts *model.BaseBanOpts) error {
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
	if !(banType == model.Banned || banType == model.NoComm) {
		return errors.New("New ban must be ban or nocomm")
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
	opts.BanType = banType
	opts.IsEnabled = true
	return nil
}

func NewBanSteam(source model.SteamIDProvider, target model.StringSID, duration model.Duration,
	reason model.Reason, reasonText string, modNote string, origin model.Origin, reportId int64, banType model.BanType,
	banSteam *model.BanSteam) error {
	var opts model.BanSteamOpts
	errBaseOpts := newBaseBanOpts(source, target, duration, reason, reasonText, modNote, origin, banType, &opts.BaseBanOpts)
	if errBaseOpts != nil {
		return errBaseOpts
	}
	if reportId < 0 {
		return errors.New("Invalid report ID")
	}
	opts.ReportId = reportId
	banSteam.Apply(opts)
	banSteam.ReportId = opts.ReportId
	banSteam.BanID = opts.BanId
	return nil
}

// BanSteam will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of config.DefaultExpiration() will be used.
func (app *App) BanSteam(ctx context.Context, database store.Store, banSteam *model.BanSteam) error {
	if !banSteam.TargetId.Valid() {
		return errors.Wrap(consts.ErrInvalidSID, "Invalid target steam id")
	}
	existing := model.NewBannedPerson()
	errGetExistingBan := database.GetBanBySteamID(ctx, banSteam.TargetId, &existing, false)
	if existing.Ban.BanID > 0 {
		return store.ErrDuplicate
	}
	if errGetExistingBan != nil && errGetExistingBan != store.ErrNoResult {
		return errors.Wrapf(errGetExistingBan, "Failed to get ban")
	}

	if errSave := database.SaveBan(ctx, banSteam); errSave != nil {
		return errors.Wrap(errSave, "Failed to save ban")
	}
	var updateAppealState = func(reportId int64) error {
		var report model.Report
		if errReport := database.GetReport(ctx, reportId, &report); errReport != nil {
			log.WithFields(log.Fields{
				"report_id": reportId,
			}).Errorf("Failed to get associated report for ban")
			return errors.New("Failed to get report")
		}
		report.ReportStatus = model.ClosedWithAction
		if errSaveReport := database.SaveReport(ctx, &report); errSaveReport != nil {
			log.WithFields(log.Fields{
				"report_id": reportId,
			}).Errorf("Failed to set appeal state for ban")
			return errors.New("Failed to update report state")
		}
		return nil
	}
	// Close the report if the ban was attached to one
	if banSteam.ReportId > 0 {
		if errRep := updateAppealState(banSteam.ReportId); errRep != nil {
			return errRep
		}
		log.WithFields(log.Fields{
			"report_id": banSteam.ReportId,
		}).Infof("Report state set to closed")
	}

	go func() {
		var title string
		var colour int
		if banSteam.BanType == model.NoComm {
			title = fmt.Sprintf("User Muted (#%d)", banSteam.BanID)
			colour = int(orange)
		} else {
			title = fmt.Sprintf("User Banned (#%d)", banSteam.BanID)
			colour = int(red)
		}
		banNotice := &discordgo.MessageEmbed{
			URL:   fmt.Sprintf("https://steamcommunity.com/profiles/%d", banSteam.TargetId),
			Type:  discordgo.EmbedTypeRich,
			Title: title,
			Color: colour,
		}
		addFieldsSteamID(banNotice, banSteam.TargetId)
		expIn := "Permanent"
		expAt := "Permanent"
		if banSteam.ValidUntil.Year()-config.Now().Year() < 5 {
			expIn = config.FmtDuration(banSteam.ValidUntil)
			expAt = config.FmtTimeShort(banSteam.ValidUntil)
		}
		addField(banNotice, "Expires In", expIn)
		addField(banNotice, "Expires At", expAt)
		app.sendDiscordPayload(discordPayload{channelId: config.Discord.PublicLogChannelId, embed: banNotice})
	}()
	// TODO mute player currently in-game w/o kicking
	if banSteam.BanType == model.Banned {
		if errKick := app.Kick(ctx, database, model.System,
			model.StringSID(banSteam.TargetId.String()),
			model.StringSID(banSteam.SourceId.String()),
			banSteam.Reason, nil); errKick != nil {
			log.Errorf("failed to kick player: %v", errKick)
		}
	} else if banSteam.BanType == model.NoComm {
		if errSilence := app.Silence(ctx, model.System,
			model.StringSID(banSteam.TargetId.String()),
			model.StringSID(banSteam.SourceId.String()),
			banSteam.Reason, nil); errSilence != nil {
			log.Errorf("failed to silence player: %v", errSilence)
		}
	}

	return nil
}

func NewBanASN(source model.SteamIDProvider, target model.StringSID, duration model.Duration,
	reason model.Reason, reasonText string, modNote string, origin model.Origin, ASNum int64, banType model.BanType, banASN *model.BanASN) error {
	var opts model.BanASNOpts
	errBaseOpts := newBaseBanOpts(source, target, duration, reason, reasonText, modNote, origin, banType, &opts.BaseBanOpts)
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
func (app *App) BanASN(ctx context.Context, database store.Store, banASN *model.BanASN) error {
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
	reason model.Reason, reasonText string, modNote string, origin model.Origin, cidr string,
	banType model.BanType, banCIDR *model.BanCIDR) error {
	var opts model.BanCIDROpts
	if errBaseOpts := newBaseBanOpts(source, target, duration, reason, reasonText, modNote, origin,
		banType, &opts.BaseBanOpts); errBaseOpts != nil {
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
func (app *App) BanCIDR(ctx context.Context, database store.Store, banNet *model.BanCIDR) error {
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
		if errFindPI := app.FindPlayerByCIDR(ctx, banNet.CIDR, &playerInfo); errFindPI != nil {
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
	banType model.BanType, banGroup *model.BanGroup) error {
	var opts model.BanSteamGroupOpts
	errBaseOpts := newBaseBanOpts(source, target, duration, reason, reasonText, modNote, origin, banType, &opts.BaseBanOpts)
	if errBaseOpts != nil {
		return errBaseOpts
	}
	// TODO validate gid here w/fetch?
	opts.GroupId = groupId
	opts.GroupName = groupName
	return banGroup.Apply(opts)
}

func (app *App) BanSteamGroup(ctx context.Context, database store.Store, banGroup *model.BanGroup) error {
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
func (app *App) Unban(ctx context.Context, target steamid.SID64, reason string) (bool, error) {
	bannedPerson := model.NewBannedPerson()
	errGetBan := app.store.GetBanBySteamID(ctx, target, &bannedPerson, false)
	if errGetBan != nil {
		if errGetBan == store.ErrNoResult {
			return false, nil
		}
		return false, errGetBan
	}
	bannedPerson.Ban.Deleted = true
	bannedPerson.Ban.UnbanReasonText = reason
	if errSaveBan := app.store.SaveBan(ctx, &bannedPerson.Ban); errSaveBan != nil {
		return false, errors.Wrapf(errSaveBan, "Failed to save unban")
	}
	log.Infof("Player unbanned: %v", target)

	unbanNotice := &discordgo.MessageEmbed{
		URL:   fmt.Sprintf("https://steamcommunity.com/profiles/%d", bannedPerson.Ban.TargetId),
		Type:  discordgo.EmbedTypeRich,
		Title: fmt.Sprintf("User Unbanned: %s (#%d)", bannedPerson.Person.PersonaName, bannedPerson.Ban.BanID),
		Color: int(green),
	}
	addFieldsSteamID(unbanNotice, bannedPerson.Person.SteamID)
	addField(unbanNotice, "Reason", reason)

	app.sendDiscordPayload(discordPayload{
		channelId: config.Discord.ModLogChannelId,
		embed:     unbanNotice,
	})

	return true, nil
}

// UnbanASN will remove an existing ASN ban
func (app *App) UnbanASN(ctx context.Context, database store.Store, asnNum string) (bool, error) {
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
