package app

import (
	"context"
	"errors"
	"net"
	"strconv"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"go.uber.org/zap"
)

// SetSteam is used to associate a discord user with either steam id. This is used
// instead of requiring users to link their steam account to discord itself. It also
// means the discord does not require more privileged intents.
func (app *App) SetSteam(ctx context.Context, sid64 steamid.SID64, discordID string) error {
	newPerson := model.NewPerson(sid64)
	if errGetPerson := app.db.GetOrCreatePersonBySteamID(ctx, sid64, &newPerson); errGetPerson != nil || !sid64.Valid() {
		return errs.ErrInvalidSID
	}

	if (newPerson.DiscordID) != "" {
		return errDiscordAlreadyLinked
	}

	newPerson.DiscordID = discordID
	if errSavePerson := app.db.SavePerson(ctx, &newPerson); errSavePerson != nil {
		return errors.Join(errSavePerson, errSaveChanges)
	}

	// env.Log().Info("Discord steamid set", zap.Int64("sid64", sid64.Int64()), zap.String("discordId", discordID))

	return nil
}

// FilterAdd creates a new chat filter using a regex pattern.
func (app *App) FilterAdd(ctx context.Context, filter *model.Filter) error {
	if errSave := app.db.SaveFilter(ctx, filter); errSave != nil {
		if errors.Is(errSave, errs.ErrDuplicate) {
			return errs.ErrDuplicate
		}

		// env.Log().Error("Error saving filter word", zap.Error(errSave))

		return errors.Join(errSave, errSaveChanges)
	}

	filter.Init()

	app.wordFilters.Add(filter)

	app.SendPayload(app.Config().Discord.LogChannelID, discord.FilterAddMessage(*filter))

	return nil
}

// BanSteam will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of Config.DefaultExpiration() will be used.
func (app *App) BanSteam(ctx context.Context, banSteam *model.BanSteam) error {
	if !banSteam.TargetID.Valid() {
		return errors.Join(errs.ErrInvalidSID, errs.ErrTargetID)
	}

	existing := model.NewBannedPerson()

	errGetExistingBan := app.db.GetBanBySteamID(ctx, banSteam.TargetID, &existing, false)

	if existing.BanID > 0 {
		return errs.ErrDuplicate
	}

	if errGetExistingBan != nil && !errors.Is(errGetExistingBan, errs.ErrNoResult) {
		return errors.Join(errGetExistingBan, errFailedFetchBan)
	}

	if errSave := app.db.SaveBan(ctx, banSteam); errSave != nil {
		return errors.Join(errSave, errSaveBan)
	}

	updateAppealState := func(reportId int64) error {
		var report model.Report
		if errReport := app.db.GetReport(ctx, reportId, &report); errReport != nil {
			return errors.Join(errReport, errGetBanReport)
		}

		report.ReportStatus = model.ClosedWithAction
		if errSaveReport := app.db.SaveReport(ctx, &report); errSaveReport != nil {
			return errors.Join(errSaveReport, errReportStateUpdate)
		}

		return nil
	}

	// Close the report if the ban was attached to one
	if banSteam.ReportID > 0 {
		if errRep := updateAppealState(banSteam.ReportID); errRep != nil {
			return errRep
		}
	}

	var target model.Person
	if err := app.db.GetOrCreatePersonBySteamID(ctx, banSteam.TargetID, &target); err != nil {
		return errors.Join(err, errFetchPerson)
	}

	// TODO mute player currently in-game w/o kicking
	if banSteam.BanType == model.Banned {
		if errKick := state.Kick(ctx, app.State(), banSteam.TargetID, banSteam.Reason); errKick != nil && !errors.Is(errKick, errs.ErrPlayerNotFound) {
			app.log.Error("Failed to kick player", zap.Error(errKick),
				zap.Int64("sid64", banSteam.TargetID.Int64()))
		}

		app.SendPayload(app.Config().Discord.LogChannelID, discord.KickPlayerEmbed(target))
	} else if banSteam.BanType == model.NoComm {
		if errSilence := state.Silence(ctx, app.State(), banSteam.TargetID, banSteam.Reason); errSilence != nil && !errors.Is(errSilence, errs.ErrPlayerNotFound) {
			app.log.Error("Failed to silence player", zap.Error(errSilence),
				zap.Int64("sid64", banSteam.TargetID.Int64()))
		}
		app.SendPayload(app.Config().Discord.LogChannelID, discord.SilenceEmbed(target))
	}

	return nil
}

// BanASN will ban all network ranges associated with the requested ASN.
func (app *App) BanASN(ctx context.Context, banASN *model.BanASN) error {
	var existing model.BanASN
	if errGetExistingBan := app.db.GetBanASN(ctx, banASN.ASNum, &existing); errGetExistingBan != nil {
		if !errors.Is(errGetExistingBan, errs.ErrNoResult) {
			return errors.Join(errGetExistingBan, errFailedFetchBan)
		}
	}

	if errSave := app.db.SaveBanASN(ctx, banASN); errSave != nil {
		return errors.Join(errSave, errSaveBan)
	}
	// TODO Kick all Current players matching
	return nil
}

// BanCIDR adds a new network to the banned network list. It will accept any Valid CIDR format.
// It accepts an optional steamid to associate a particular user with the network ban. Any active players
// that fall within the range will be kicked immediately.
// If duration is 0, the value of Config.DefaultExpiration() will be used.
func (app *App) BanCIDR(ctx context.Context, banNet *model.BanCIDR) error {
	// TODO
	// _, err2 := db.GetBanNetByAddress(ctx, net.ParseIP(cidrStr))
	// if err2 != nil && err2 != db.ErrNoResult {
	//	return "", errCommandFailed
	// }
	// if err2 == nil {
	//	return "", consts.ErrDuplicateBan
	// }
	if banNet.CIDR == "" {
		return ErrCIDRMissing
	}

	_, realCIDR, errCIDR := net.ParseCIDR(banNet.CIDR)
	if errCIDR != nil {
		return errors.Join(errCIDR, errs.ErrInvalidIP)
	}

	if errSaveBanNet := app.db.SaveBanNet(ctx, banNet); errSaveBanNet != nil {
		return errors.Join(errSaveBanNet, errSaveBan)
	}

	var author model.Person
	if err := app.Store().GetOrCreatePersonBySteamID(ctx, banNet.SourceID, &author); err != nil {
		return errors.Join(err, errFetchSource)
	}

	var target model.Person
	if err := app.Store().GetOrCreatePersonBySteamID(ctx, banNet.TargetID, &target); err != nil {
		return errors.Join(err, errFetchTarget)
	}

	conf := app.Config()

	app.SendPayload(conf.Discord.LogChannelID, discord.BanCIDRResponse(realCIDR, author, conf.ExtURL(author), target, banNet))

	go func(_ *net.IPNet, reason model.Reason) {
		foundPlayers := app.state.Find("", "", nil, realCIDR)

		if len(foundPlayers) == 0 {
			return
		}

		for _, player := range foundPlayers {
			if errKick := state.Kick(ctx, app.state, player.Player.SID, reason); errKick != nil {
				app.log.Error("Failed to kick player", zap.Error(errKick))
			}
		}
	}(realCIDR, banNet.Reason)

	return nil
}

func (app *App) BanSteamGroup(ctx context.Context, banGroup *model.BanGroup) error {
	members, membersErr := steamweb.GetGroupMembers(ctx, banGroup.GroupID)
	if membersErr != nil || len(members) == 0 {
		return errors.Join(membersErr, errGroupValidate)
	}

	if errSave := app.db.SaveBanGroup(ctx, banGroup); errSave != nil {
		return errors.Join(errSave, errSaveBanGroup)
	}

	// env.Log().Info("Steam group banned", zap.Int64("gid64", banGroup.GroupID.Int64()),	zap.Int("members", len(members)))

	return nil
}

// Unban will set the Current ban to now, making it expired.
// Returns true, nil if the ban exists, and was successfully banned.
// Returns false, nil if the ban does not exist.
func (app *App) Unban(ctx context.Context, targetSID steamid.SID64, reason string) (bool, error) {
	bannedPerson := model.NewBannedPerson()
	errGetBan := app.db.GetBanBySteamID(ctx, targetSID, &bannedPerson, false)

	if errGetBan != nil {
		if errors.Is(errGetBan, errs.ErrNoResult) {
			return false, nil
		}

		return false, errors.Join(errGetBan, errFailedFetchBan)
	}

	bannedPerson.Deleted = true
	bannedPerson.UnbanReasonText = reason

	if errSave := app.db.SaveBan(ctx, &bannedPerson.BanSteam); errSave != nil {
		return false, errors.Join(errSave, errSaveBan)
	}

	conf := app.Config()

	var person model.Person
	if err := app.db.GetPersonBySteamID(ctx, targetSID, &person); err != nil {
		return false, errors.Join(err, errFetchPerson)
	}

	app.SendPayload(conf.Discord.LogChannelID, discord.UnbanMessage(person))

	// env.Log().Info("Player unbanned", zap.Int64("sid64", targetSID.Int64()), zap.String("reason", reason))

	return true, nil
}

// UnbanASN will remove an existing ASN ban.
func (app *App) UnbanASN(ctx context.Context, asnNum string) (bool, error) {
	asNum, errConv := strconv.ParseInt(asnNum, 10, 64)
	if errConv != nil {
		return false, errors.Join(errConv, errParseASN)
	}

	var banASN model.BanASN
	if errGetBanASN := app.Store().GetBanASN(ctx, asNum, &banASN); errGetBanASN != nil {
		return false, errors.Join(errGetBanASN, errFetchASNBan)
	}

	if errDrop := app.Store().DropBanASN(ctx, &banASN); errDrop != nil {
		app.log.Error("Failed to drop ASN ban", zap.Error(errDrop))

		return false, errors.Join(errDrop, errDropASNBan)
	}

	app.log.Info("ASN unbanned", zap.Int64("ASN", asNum))

	asnNetworks, errGetASNRecords := app.db.GetASNRecordsByNum(ctx, asNum)
	if errGetASNRecords != nil {
		if errors.Is(errGetASNRecords, errs.ErrNoResult) {
			return false, errors.Join(errGetASNRecords, errUnknownASN)
		}

		return false, errors.Join(errGetASNRecords, errFetchASN)
	}

	app.SendPayload(app.conf.Discord.LogChannelID, discord.UnbanASNMessage(asNum, asnNetworks))

	return true, nil
}
