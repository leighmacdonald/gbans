package app

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// BanSteam will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of config.DefaultExpiration() will be used.
func (app *App) BanSteam(ctx context.Context, banSteam *store.BanSteam) error {
	if !banSteam.TargetID.Valid() {
		return errors.Wrap(consts.ErrInvalidSID, "Invalid target steam id")
	}

	existing := store.NewBannedPerson()

	errGetExistingBan := app.db.GetBanBySteamID(ctx, banSteam.TargetID, &existing, false)

	if existing.BanID > 0 {
		return store.ErrDuplicate
	}

	if errGetExistingBan != nil && !errors.Is(errGetExistingBan, store.ErrNoResult) {
		return errors.Wrapf(errGetExistingBan, "Failed to get ban")
	}

	if errSave := app.db.SaveBan(ctx, banSteam); errSave != nil {
		return errors.Wrap(errSave, "Failed to save ban")
	}

	updateAppealState := func(reportId int64) error {
		var report store.Report
		if errReport := app.db.GetReport(ctx, reportId, &report); errReport != nil {
			return errors.Wrap(errReport, "Failed to get associated report for ban")
		}

		report.ReportStatus = store.ClosedWithAction
		if errSaveReport := app.db.SaveReport(ctx, &report); errSaveReport != nil {
			return errors.Wrap(errSaveReport, "Failed to update report state")
		}

		return nil
	}

	// Close the report if the ban was attached to one
	if banSteam.ReportID > 0 {
		if errRep := updateAppealState(banSteam.ReportID); errRep != nil {
			return errRep
		}

		app.log.Info("Report state set to closed", zap.Int64("report_id", banSteam.ReportID))
	}

	if app.config().Discord.Enabled {
		go func() {
			var (
				conf   = app.config()
				title  string
				colour int
			)

			if banSteam.BanType == store.NoComm {
				title = fmt.Sprintf("User Muted (#%d)", banSteam.BanID)
				colour = conf.Discord.ColourWarn
			} else {
				title = fmt.Sprintf("User Banned (#%d)", banSteam.BanID)
				colour = conf.Discord.ColourError
			}

			msgEmbed := discord.NewEmbed(conf, title)
			msgEmbed.Embed().
				SetColor(colour).
				SetURL(fmt.Sprintf("https://steamcommunity.com/profiles/%s", banSteam.TargetID))

			msgEmbed.AddFieldsSteamID(banSteam.TargetID)

			expIn := "Permanent"
			expAt := "Permanent"

			if banSteam.ValidUntil.Year()-time.Now().Year() < 5 {
				expIn = util.FmtDuration(banSteam.ValidUntil)
				expAt = util.FmtTimeShort(banSteam.ValidUntil)
			}

			msgEmbed.
				Embed().
				AddField("Expires In", expIn).
				AddField("Expires At", expAt)
			app.discord.SendPayload(discord.Payload{
				ChannelID: conf.Discord.PublicLogChannelID, Embed: msgEmbed.Embed().Truncate().MessageEmbed,
			})
		}()
	}
	// TODO mute player currently in-game w/o kicking
	if banSteam.BanType == store.Banned {
		if errKick := app.Kick(ctx, store.System,
			banSteam.TargetID,
			banSteam.SourceID,
			banSteam.Reason); errKick != nil && !errors.Is(errKick, consts.ErrPlayerNotFound) {
			app.log.Error("Failed to kick player", zap.Error(errKick),
				zap.Int64("sid64", banSteam.TargetID.Int64()))
		}
	} else if banSteam.BanType == store.NoComm {
		if errSilence := app.Silence(ctx, store.System,
			banSteam.TargetID,
			banSteam.SourceID,
			banSteam.Reason); errSilence != nil && !errors.Is(errSilence, consts.ErrPlayerNotFound) {
			app.log.Error("Failed to silence player", zap.Error(errSilence),
				zap.Int64("sid64", banSteam.TargetID.Int64()))
		}
	}

	return nil
}

// BanASN will ban all network ranges associated with the requested ASN.
func (app *App) BanASN(ctx context.Context, banASN *store.BanASN) error {
	var existing store.BanASN
	if errGetExistingBan := app.db.GetBanASN(ctx, banASN.ASNum, &existing); errGetExistingBan != nil {
		if !errors.Is(errGetExistingBan, store.ErrNoResult) {
			return errors.Wrapf(errGetExistingBan, "Failed trying to fetch existing asn ban")
		}
	}

	if errSave := app.db.SaveBanASN(ctx, banASN); errSave != nil {
		return errors.Wrap(errSave, "Failed to save ban")
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
	// _, err2 := store.GetBanNetByAddress(ctx, net.ParseIP(cidrStr))
	// if err2 != nil && err2 != store.ErrNoResult {
	//	return "", errCommandFailed
	// }
	// if err2 == nil {
	//	return "", consts.ErrDuplicateBan
	// }
	if banNet.CIDR == "" {
		return errors.New("IP unset")
	}

	_, realCIDR, errCIDR := net.ParseCIDR(banNet.CIDR)
	if errCIDR != nil {
		return errors.Wrap(errCIDR, "Invalid IP")
	}

	if errSaveBanNet := app.db.SaveBanNet(ctx, banNet); errSaveBanNet != nil {
		return errors.Wrap(errSaveBanNet, "Failed to save ban")
	}

	go func(_ *net.IPNet, reason store.Reason) {
		state := app.state.current()
		foundPlayers := state.find(findOpts{CIDR: realCIDR})

		if len(foundPlayers) == 0 {
			return
		}

		for _, player := range foundPlayers {
			if errKick := app.Kick(ctx, store.System, player.Player.SID, banNet.SourceID, reason); errKick != nil {
				app.log.Error("Failed to kick player", zap.Error(errKick))
			}
		}
	}(realCIDR, banNet.Reason)

	conf := app.config()

	msgEmbed := discord.NewEmbed(conf, "IP Banned Successfully")
	msgEmbed.Embed().
		SetColor(conf.Discord.ColourSuccess).
		AddField("cidr", realCIDR.String()).
		AddField("net_id", fmt.Sprintf("%d", banNet.NetID)).
		AddField("Reason", banNet.Reason.String())

	var author store.Person
	if err := app.db.GetOrCreatePersonBySteamID(ctx, banNet.SourceID, &author); err != nil {
		return errors.Wrap(err, "Failed to get author")
	}

	var target store.Person
	if err := app.db.GetOrCreatePersonBySteamID(ctx, banNet.TargetID, &target); err != nil {
		return errors.Wrap(err, "Failed to get target")
	}

	msgEmbed.AddTargetPerson(target)
	msgEmbed.AddAuthorPersonInfo(author)

	app.discord.SendPayload(discord.Payload{
		ChannelID: conf.Discord.LogChannelID,
		Embed:     msgEmbed.Embed().Truncate().MessageEmbed,
	})

	return nil
}

func (app *App) BanSteamGroup(ctx context.Context, banGroup *store.BanGroup) error {
	members, membersErr := steamweb.GetGroupMembers(ctx, banGroup.GroupID)
	if membersErr != nil {
		return errors.Wrapf(membersErr, "Failed to validate group")
	}

	if errSaveBanGroup := app.db.SaveBanGroup(ctx, banGroup); errSaveBanGroup != nil {
		return errors.Wrapf(errSaveBanGroup, "Failed to save banned group")
	}

	app.log.Info("Steam group banned", zap.Int64("gid64", banGroup.GroupID.Int64()),
		zap.Int("members", len(members)))

	return nil
}

// Unban will set the current ban to now, making it expired.
// Returns true, nil if the ban exists, and was successfully banned.
// Returns false, nil if the ban does not exist.
func (app *App) Unban(ctx context.Context, targetSID steamid.SID64, reason string) (bool, error) {
	bannedPerson := store.NewBannedPerson()
	errGetBan := app.db.GetBanBySteamID(ctx, targetSID, &bannedPerson, false)

	if errGetBan != nil {
		if errors.Is(errGetBan, store.ErrNoResult) {
			return false, nil
		}

		return false, errors.Wrapf(errGetBan, "Failed to get ban")
	}

	bannedPerson.Deleted = true
	bannedPerson.UnbanReasonText = reason

	if errSaveBan := app.db.SaveBan(ctx, &bannedPerson.BanSteam); errSaveBan != nil {
		return false, errors.Wrapf(errSaveBan, "Failed to save unban")
	}

	app.log.Info("Player unbanned", zap.Int64("sid64", targetSID.Int64()), zap.String("reason", reason))

	conf := app.config()

	msgEmbed := discord.
		NewEmbed(conf, "User Unbanned Successfully")
	msgEmbed.Embed().
		SetColor(conf.Discord.ColourSuccess).
		SetURL(conf.ExtURL(bannedPerson.BanSteam)).
		AddField("ban_id", fmt.Sprintf("%d", bannedPerson.BanID)).AddField("Reason", reason)

	var target store.Person
	if err := app.db.GetPersonBySteamID(ctx, targetSID, &target); err != nil {
		return false, errors.Wrap(err, "Failed to get target")
	}

	msgEmbed.AddTargetPerson(target)

	msgEmbed.AddFieldsSteamID(bannedPerson.TargetID)

	app.discord.SendPayload(discord.Payload{
		ChannelID: conf.Discord.LogChannelID,
		Embed:     msgEmbed.Embed().Truncate().MessageEmbed,
	})

	return true, nil
}

// UnbanASN will remove an existing ASN ban.
func (app *App) UnbanASN(ctx context.Context, asnNum string) (bool, error) {
	asNum, errConv := strconv.ParseInt(asnNum, 10, 64)
	if errConv != nil {
		return false, errors.Wrapf(errConv, "Failed to parse int")
	}

	var banASN store.BanASN
	if errGetBanASN := app.db.GetBanASN(ctx, asNum, &banASN); errGetBanASN != nil {
		return false, errors.Wrapf(errGetBanASN, "Failed to get asn ban")
	}

	if errDrop := app.db.DropBanASN(ctx, &banASN); errDrop != nil {
		app.log.Error("Failed to drop ASN ban", zap.Error(errDrop))

		return false, errors.Wrap(errDrop, "Failed to drop asn ban")
	}

	app.log.Info("ASN unbanned", zap.Int64("ASN", asNum))

	conf := app.config()

	msgEmbed := discord.NewEmbed(conf, "ASN Unbanned Successfully")
	msgEmbed.Embed().
		SetColor(conf.Discord.ColourSuccess).
		AddField("asn", asnNum).
		AddField("ban_asn_id", fmt.Sprintf("%d", banASN.BanASNId)).
		AddField("Reason", banASN.Reason.String())

	if banASN.TargetID.Valid() {
		var target store.Person
		if err := app.db.GetOrCreatePersonBySteamID(ctx, banASN.TargetID, &target); err != nil {
			return false, errors.Wrap(err, "Failed to get target")
		}

		msgEmbed.AddTargetPerson(target)
	}

	app.discord.SendPayload(discord.Payload{
		ChannelID: conf.Discord.LogChannelID,
		Embed:     msgEmbed.Embed().Truncate().MessageEmbed,
	})

	return true, nil
}
