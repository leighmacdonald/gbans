package app

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// BanSteam will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of config.DefaultExpiration() will be used.
func BanSteam(ctx context.Context, conf *config.Config, banSteam *store.BanSteam) error {
	if !banSteam.TargetID.Valid() {
		return errors.Wrap(consts.ErrInvalidSID, "Invalid target steam id")
	}
	existing := store.NewBannedPerson()
	errGetExistingBan := store.GetBanBySteamID(ctx, banSteam.TargetID, &existing, false)
	if existing.Ban.BanID > 0 {
		return store.ErrDuplicate
	}
	if errGetExistingBan != nil && !errors.Is(errGetExistingBan, store.ErrNoResult) {
		return errors.Wrapf(errGetExistingBan, "Failed to get ban")
	}

	if errSave := store.SaveBan(ctx, banSteam); errSave != nil {
		return errors.Wrap(errSave, "Failed to save ban")
	}

	updateAppealState := func(reportId int64) error {
		var report store.Report
		if errReport := store.GetReport(ctx, reportId, &report); errReport != nil {
			return errors.Wrap(errReport, "Failed to get associated report for ban")
		}
		report.ReportStatus = store.ClosedWithAction
		if errSaveReport := store.SaveReport(ctx, &report); errSaveReport != nil {
			return errors.Wrap(errSaveReport, "Failed to update report state")
		}

		return nil
	}

	// Close the report if the ban was attached to one
	if banSteam.ReportID > 0 {
		if errRep := updateAppealState(banSteam.ReportID); errRep != nil {
			return errRep
		}

		logger.Info("Report state set to closed", zap.Int64("report_id", banSteam.ReportID))
	}

	go func() {
		var title string
		var colour int
		if banSteam.BanType == store.NoComm {
			title = fmt.Sprintf("User Muted (#%d)", banSteam.BanID)
			colour = int(discord.Orange)
		} else {
			title = fmt.Sprintf("User Banned (#%d)", banSteam.BanID)
			colour = int(discord.Red)
		}
		banNotice := &discordgo.MessageEmbed{
			URL:   fmt.Sprintf("https://steamcommunity.com/profiles/%d", banSteam.TargetID),
			Type:  discordgo.EmbedTypeRich,
			Title: title,
			Color: colour,
		}
		discord.AddFieldsSteamID(banNotice, banSteam.TargetID)
		expIn := "Permanent"
		expAt := "Permanent"
		if banSteam.ValidUntil.Year()-config.Now().Year() < 5 {
			expIn = config.FmtDuration(banSteam.ValidUntil)
			expAt = config.FmtTimeShort(banSteam.ValidUntil)
		}
		discord.AddField(banNotice, "Expires In", expIn)
		discord.AddField(banNotice, "Expires At", expAt)
		discord.SendPayload(discord.Payload{ChannelID: conf.Discord.PublicLogChannelID, Embed: banNotice})
	}()
	// TODO mute player currently in-game w/o kicking
	if banSteam.BanType == store.Banned {
		if errKick := Kick(ctx, store.System,
			banSteam.TargetID,
			banSteam.SourceID,
			banSteam.Reason); errKick != nil {
			logger.Error("Failed to kick player", zap.Error(errKick),
				zap.Int64("sid64", banSteam.TargetID.Int64()))
		}
	} else if banSteam.BanType == store.NoComm {
		if errSilence := Silence(ctx, store.System,
			banSteam.TargetID,
			banSteam.SourceID,
			banSteam.Reason); errSilence != nil {
			logger.Error("Failed to silence player", zap.Error(errSilence),
				zap.Int64("sid64", banSteam.TargetID.Int64()))
		}
	}

	return nil
}

// BanASN will ban all network ranges associated with the requested ASN.
func BanASN(ctx context.Context, banASN *store.BanASN) error {
	var existing store.BanASN
	if errGetExistingBan := store.GetBanASN(ctx, banASN.ASNum, &existing); errGetExistingBan != nil {
		if !errors.Is(errGetExistingBan, store.ErrNoResult) {
			return errors.Wrapf(errGetExistingBan, "Failed trying to fetch existing asn ban")
		}
	}
	if errSave := store.SaveBanASN(ctx, banASN); errSave != nil {
		return errSave
	}
	// TODO Kick all current players matching
	return nil
}

// BanCIDR adds a new network to the banned network list. It will accept any Valid CIDR format.
// It accepts an optional steamid to associate a particular user with the network ban. Any active players
// that fall within the range will be kicked immediately.
// If duration is 0, the value of config.DefaultExpiration() will be used.
func BanCIDR(ctx context.Context, banNet *store.BanCIDR) error {
	// TODO
	// _, err2 := store.GetBanNetByAddress(ctx, net.ParseIP(cidrStr))
	// if err2 != nil && err2 != store.ErrNoResult {
	//	return "", errCommandFailed
	// }
	// if err2 == nil {
	//	return "", consts.ErrDuplicateBan
	// }
	if banNet.CIDR == nil {
		return errors.New("CIDR unset")
	}
	if errSaveBanNet := store.SaveBanNet(ctx, banNet); errSaveBanNet != nil {
		return errSaveBanNet
	}
	go func(_ *net.IPNet, reason store.Reason) {
		foundPlayers, found := state.Find(state.FindOpts{CIDR: banNet.CIDR})
		if !found {
			return
		}
		for _, player := range foundPlayers {
			if errKick := Kick(ctx, store.System, player.Player.SID, banNet.SourceID, reason); errKick != nil {
				logger.Error("Failed to kick player", zap.Error(errKick))
			}
		}
	}(banNet.CIDR, banNet.Reason)

	return nil
}

func BanSteamGroup(ctx context.Context, banGroup *store.BanGroup) error {
	members, membersErr := steamweb.GetGroupMembers(ctx, banGroup.GroupID)
	if membersErr != nil {
		return errors.Wrapf(membersErr, "Failed to validate group")
	}
	if errSaveBanGroup := store.SaveBanGroup(ctx, banGroup); errSaveBanGroup != nil {
		return errSaveBanGroup
	}
	logger.Info("Steam group banned", zap.Int64("gid64", banGroup.GroupID.Int64()),
		zap.Int("members", len(members)))
	return nil
}

// Unban will set the current ban to now, making it expired.
// Returns true, nil if the ban exists, and was successfully banned.
// Returns false, nil if the ban does not exist.
func Unban(ctx context.Context, conf *config.Config, target steamid.SID64, reason string) (bool, error) {
	bannedPerson := store.NewBannedPerson()
	errGetBan := store.GetBanBySteamID(ctx, target, &bannedPerson, false)
	if errGetBan != nil {
		if errors.Is(errGetBan, store.ErrNoResult) {
			return false, nil
		}
		return false, errGetBan
	}
	bannedPerson.Ban.Deleted = true
	bannedPerson.Ban.UnbanReasonText = reason
	if errSaveBan := store.SaveBan(ctx, &bannedPerson.Ban); errSaveBan != nil {
		return false, errors.Wrapf(errSaveBan, "Failed to save unban")
	}
	logger.Info("Player unbanned", zap.Int64("sid64", target.Int64()), zap.String("reason", reason))

	unbanNotice := &discordgo.MessageEmbed{
		URL:   fmt.Sprintf("https://steamcommunity.com/profiles/%d", bannedPerson.Ban.TargetID),
		Type:  discordgo.EmbedTypeRich,
		Title: fmt.Sprintf("User Unbanned: %s (#%d)", bannedPerson.Person.PersonaName, bannedPerson.Ban.BanID),
		Color: int(discord.Green),
	}
	discord.AddFieldsSteamID(unbanNotice, bannedPerson.Person.SteamID)
	discord.AddField(unbanNotice, "Reason", reason)

	discord.SendPayload(discord.Payload{
		ChannelID: conf.Discord.ModLogChannelID,
		Embed:     unbanNotice,
	})

	return true, nil
}

// UnbanASN will remove an existing ASN ban.
func UnbanASN(ctx context.Context, asnNum string) (bool, error) {
	asNum, errConv := strconv.ParseInt(asnNum, 10, 64)
	if errConv != nil {
		return false, errConv
	}
	var banASN store.BanASN
	if errGetBanASN := store.GetBanASN(ctx, asNum, &banASN); errGetBanASN != nil {
		return false, errGetBanASN
	}
	if errDrop := store.DropBanASN(ctx, &banASN); errDrop != nil {
		logger.Error("Failed to drop ASN ban", zap.Error(errDrop))
		return false, errDrop
	}
	logger.Info("ASN unbanned", zap.Int64("ASN", asNum))
	return true, nil
}
