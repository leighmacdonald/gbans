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
	"go.uber.org/zap"
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
func (app *App) BanSteam(ctx context.Context, banSteam *model.BanSteam) error {
	if !banSteam.TargetId.Valid() {
		return errors.Wrap(consts.ErrInvalidSID, "Invalid target steam id")
	}
	existing := model.NewBannedPerson()
	errGetExistingBan := app.store.GetBanBySteamID(ctx, banSteam.TargetId, &existing, false)
	if existing.Ban.BanID > 0 {
		return store.ErrDuplicate
	}
	if errGetExistingBan != nil && errGetExistingBan != store.ErrNoResult {
		return errors.Wrapf(errGetExistingBan, "Failed to get ban")
	}

	if errSave := app.store.SaveBan(ctx, banSteam); errSave != nil {
		return errors.Wrap(errSave, "Failed to save ban")
	}
	var updateAppealState = func(reportId int64) error {
		var report model.Report
		if errReport := app.store.GetReport(ctx, reportId, &report); errReport != nil {
			return errors.Wrap(errReport, "Failed to get associated report for ban")
		}
		report.ReportStatus = model.ClosedWithAction
		if errSaveReport := app.store.SaveReport(ctx, &report); errSaveReport != nil {
			return errors.Wrap(errSaveReport, "Failed to update report state")
		}
		return nil
	}
	// Close the report if the ban was attached to one
	if banSteam.ReportId > 0 {
		if errRep := updateAppealState(banSteam.ReportId); errRep != nil {
			return errRep
		}
		app.logger.Info("Report state set to closed", zap.Int64("report_id", banSteam.ReportId))
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
		addFieldsSteamID(banNotice, app.logger, banSteam.TargetId)
		expIn := "Permanent"
		expAt := "Permanent"
		if banSteam.ValidUntil.Year()-config.Now().Year() < 5 {
			expIn = config.FmtDuration(banSteam.ValidUntil)
			expAt = config.FmtTimeShort(banSteam.ValidUntil)
		}
		addField(banNotice, app.logger, "Expires In", expIn)
		addField(banNotice, app.logger, "Expires At", expAt)
		app.sendDiscordPayload(discordPayload{channelId: config.Discord.PublicLogChannelId, embed: banNotice})
	}()
	// TODO mute player currently in-game w/o kicking
	if banSteam.BanType == model.Banned {
		if errKick := app.Kick(ctx, model.System,
			model.StringSID(banSteam.TargetId.String()),
			model.StringSID(banSteam.SourceId.String()),
			banSteam.Reason, nil); errKick != nil {
			app.logger.Error("Failed to kick player", zap.Error(errKick),
				zap.Int64("sid64", banSteam.TargetId.Int64()))
		}
	} else if banSteam.BanType == model.NoComm {
		if errSilence := app.Silence(ctx, model.System,
			model.StringSID(banSteam.TargetId.String()),
			model.StringSID(banSteam.SourceId.String()),
			banSteam.Reason, nil); errSilence != nil {
			app.logger.Error("Failed to silence player", zap.Error(errSilence),
				zap.Int64("sid64", banSteam.TargetId.Int64()))
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
func (app *App) BanASN(ctx context.Context, banASN *model.BanASN) error {
	var existing model.BanASN
	if errGetExistingBan := app.store.GetBanASN(ctx, banASN.ASNum, &existing); errGetExistingBan != nil {
		if !errors.Is(errGetExistingBan, store.ErrNoResult) {
			return errors.Wrapf(errGetExistingBan, "Failed trying to fetch existing asn ban")
		}
	}
	if errSave := app.store.SaveBanASN(ctx, banASN); errSave != nil {
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
func (app *App) BanCIDR(ctx context.Context, banNet *model.BanCIDR) error {
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
	if errSaveBanNet := app.store.SaveBanNet(ctx, banNet); errSaveBanNet != nil {
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
				app.logger.Error("Failed to query for ban request", zap.Error(errExecRCON))
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

func (app *App) BanSteamGroup(ctx context.Context, banGroup *model.BanGroup) error {
	members, membersErr := steamweb.GetGroupMembers(ctx, banGroup.GroupId)
	if membersErr != nil {
		return errors.Wrapf(membersErr, "Failed to validate group")
	}
	if errSaveBanGroup := app.store.SaveBanGroup(ctx, banGroup); errSaveBanGroup != nil {
		return errSaveBanGroup
	}
	app.logger.Info("Steam group banned", zap.Int64("gid64", banGroup.GroupId.Int64()),
		zap.Int("members", len(members)))
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
	app.logger.Info("Player unbanned", zap.Int64("sid64", target.Int64()), zap.String("reason", reason))

	unbanNotice := &discordgo.MessageEmbed{
		URL:   fmt.Sprintf("https://steamcommunity.com/profiles/%d", bannedPerson.Ban.TargetId),
		Type:  discordgo.EmbedTypeRich,
		Title: fmt.Sprintf("User Unbanned: %s (#%d)", bannedPerson.Person.PersonaName, bannedPerson.Ban.BanID),
		Color: int(green),
	}
	addFieldsSteamID(unbanNotice, app.logger, bannedPerson.Person.SteamID)
	addField(unbanNotice, app.logger, "Reason", reason)

	app.sendDiscordPayload(discordPayload{
		channelId: config.Discord.ModLogChannelId,
		embed:     unbanNotice,
	})

	return true, nil
}

// UnbanASN will remove an existing ASN ban
func (app *App) UnbanASN(ctx context.Context, asnNum string) (bool, error) {
	asNum, errConv := strconv.ParseInt(asnNum, 10, 64)
	if errConv != nil {
		return false, errConv
	}
	var banASN model.BanASN
	if errGetBanASN := app.store.GetBanASN(ctx, asNum, &banASN); errGetBanASN != nil {
		return false, errGetBanASN
	}
	if errDrop := app.store.DropBanASN(ctx, &banASN); errDrop != nil {
		app.logger.Error("Failed to drop ASN ban", zap.Error(errDrop))
		return false, errDrop
	}
	app.logger.Info("ASN unbanned", zap.Int64("ASN", asNum))
	return true, nil
}
