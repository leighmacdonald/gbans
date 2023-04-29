package app

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/discordutil"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net"
	"strconv"
)

// BanSteam will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of config.DefaultExpiration() will be used.
func (app *App) BanSteam(ctx context.Context, banSteam *store.BanSteam) error {
	if !banSteam.TargetId.Valid() {
		return errors.Wrap(consts.ErrInvalidSID, "Invalid target steam id")
	}
	existing := store.NewBannedPerson()
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
		var report store.Report
		if errReport := app.store.GetReport(ctx, reportId, &report); errReport != nil {
			return errors.Wrap(errReport, "Failed to get associated report for ban")
		}
		report.ReportStatus = store.ClosedWithAction
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
		if banSteam.BanType == store.NoComm {
			title = fmt.Sprintf("User Muted (#%d)", banSteam.BanID)
			colour = int(discordutil.Orange)
		} else {
			title = fmt.Sprintf("User Banned (#%d)", banSteam.BanID)
			colour = int(discordutil.Red)
		}
		banNotice := &discordgo.MessageEmbed{
			URL:   fmt.Sprintf("https://steamcommunity.com/profiles/%d", banSteam.TargetId),
			Type:  discordgo.EmbedTypeRich,
			Title: title,
			Color: colour,
		}
		discordutil.AddFieldsSteamID(banNotice, app.logger, banSteam.TargetId)
		expIn := "Permanent"
		expAt := "Permanent"
		if banSteam.ValidUntil.Year()-config.Now().Year() < 5 {
			expIn = config.FmtDuration(banSteam.ValidUntil)
			expAt = config.FmtTimeShort(banSteam.ValidUntil)
		}
		discordutil.AddField(banNotice, app.logger, "Expires In", expIn)
		discordutil.AddField(banNotice, app.logger, "Expires At", expAt)
		app.SendDiscordPayload(discordutil.Payload{ChannelId: config.Discord.PublicLogChannelId, Embed: banNotice})
	}()
	// TODO mute player currently in-game w/o kicking
	if banSteam.BanType == store.Banned {
		if errKick := app.Kick(ctx, store.System,
			banSteam.TargetId,
			banSteam.SourceId,
			banSteam.Reason); errKick != nil {
			app.logger.Error("Failed to kick player", zap.Error(errKick),
				zap.Int64("sid64", banSteam.TargetId.Int64()))
		}
	} else if banSteam.BanType == store.NoComm {
		if errSilence := app.Silence(ctx, store.System,
			banSteam.TargetId,
			banSteam.SourceId,
			banSteam.Reason); errSilence != nil {
			app.logger.Error("Failed to silence player", zap.Error(errSilence),
				zap.Int64("sid64", banSteam.TargetId.Int64()))
		}
	}

	return nil
}

// BanASN will ban all network ranges associated with the requested ASN
func (app *App) BanASN(ctx context.Context, banASN *store.BanASN) error {
	var existing store.BanASN
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

// BanCIDR adds a new network to the banned network list. It will accept any Valid CIDR format.
// It accepts an optional steamid to associate a particular user with the network ban. Any active players
// that fall within the range will be kicked immediately.
// If duration is 0, the value of config.DefaultExpiration() will be used.
func (app *App) BanCIDR(ctx context.Context, banNet *store.BanCIDR) error {
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
	go func(n *net.IPNet, reason store.Reason) {
		foundPlayers, found := state.Find(state.FindOpts{CIDR: banNet.CIDR})
		if !found {
			return
		}
		for _, player := range foundPlayers {
			if errKick := app.Kick(ctx, store.System, player.Player.SID, banNet.SourceId, reason); errKick != nil {
				app.logger.Error("Failed to kick player", zap.Error(errKick))
			}
		}
	}(banNet.CIDR, banNet.Reason)

	return nil
}

func (app *App) BanSteamGroup(ctx context.Context, banGroup *store.BanGroup) error {
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
	bannedPerson := store.NewBannedPerson()
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
		Color: int(discordutil.Green),
	}
	discordutil.AddFieldsSteamID(unbanNotice, app.logger, bannedPerson.Person.SteamID)
	discordutil.AddField(unbanNotice, app.logger, "Reason", reason)

	app.SendDiscordPayload(discordutil.Payload{
		ChannelId: config.Discord.ModLogChannelId,
		Embed:     unbanNotice,
	})

	return true, nil
}

// UnbanASN will remove an existing ASN ban
func (app *App) UnbanASN(ctx context.Context, asnNum string) (bool, error) {
	asNum, errConv := strconv.ParseInt(asnNum, 10, 64)
	if errConv != nil {
		return false, errConv
	}
	var banASN store.BanASN
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
