package usecase

import (
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type banUsecase struct {
	banRepo        domain.BanRepository
	personUsecase  domain.PersonUsecase
	configUsecase  domain.ConfigUsecase
	discordUsecase domain.DiscordUsecase
	stateUsecase   domain.StateUsecase
	log            *zap.Logger
}

func NewBanUsecase(log *zap.Logger, br domain.BanRepository, pr domain.PersonUsecase, cu domain.ConfigUsecase, du domain.DiscordUsecase, su domain.StateUsecase) domain.BanUsecase {
	return &banUsecase{log: log, banRepo: br, personUsecase: pr, configUsecase: cu, discordUsecase: du}
}
func (s *banUsecase) GetBanBySteamID(ctx context.Context, sid64 steamid.SID64, bannedPerson *domain.BannedSteamPerson, deletedOk bool) error {
	return s.banRepo.GetBanBySteamID(ctx, sid64, bannedPerson, deletedOk)
}

func (s *banUsecase) GetBanByBanID(ctx context.Context, banID int64, bannedPerson *domain.BannedSteamPerson, deletedOk bool) error {
	return s.banRepo.GetBanByBanID(ctx, banID, bannedPerson, deletedOk)
}

func (s *banUsecase) GetBanByLastIP(ctx context.Context, lastIP net.IP, bannedPerson *domain.BannedSteamPerson, deletedOk bool) error {
	return s.banRepo.GetBanByLastIP(ctx, lastIP, bannedPerson, deletedOk)
}

func (s *banUsecase) SaveBan(ctx context.Context, ban *domain.BanSteam) error {
	return s.banRepo.SaveBan(ctx, ban)
}

func (s *banUsecase) BanASN(ctx context.Context, banASN *domain.BanASN) error {
	var existing domain.BanASN
	if errGetExistingBan := s.banRepo.GetBanASN(ctx, banASN.ASNum, &existing); errGetExistingBan != nil {
		if !errors.Is(errGetExistingBan, errs.ErrNoResult) {
			return errors.Join(errGetExistingBan, domain.ErrFailedFetchBan)
		}
	}

	if errSave := s.banRepo.SaveBanASN(ctx, banASN); errSave != nil {
		return errors.Join(errSave, domain.ErrSaveBan)
	}
	// TODO Kick all Current players matching
	return nil
}

// BanCIDR adds a new network to the banned network list. It will accept any Valid CIDR format.
// It accepts an optional steamid to associate a particular user with the network ban. Any active players
// that fall within the range will be kicked immediately.
// If duration is 0, the value of Config.DefaultExpiration() will be used.
func (s *banUsecase) BanCIDR(ctx context.Context, banNet *domain.BanCIDR) error {
	// TODO
	// _, err2 := db.GetBanNetByAddress(ctx, net.ParseIP(cidrStr))
	// if err2 != nil && err2 != db.ErrNoResult {
	//	return "", errCommandFailed
	// }
	// if err2 == nil {
	//	return "", consts.ErrDuplicateBan
	// }
	if banNet.CIDR == "" {
		return domain.ErrCIDRMissing
	}

	_, realCIDR, errCIDR := net.ParseCIDR(banNet.CIDR)
	if errCIDR != nil {
		return errors.Join(errCIDR, errs.ErrInvalidIP)
	}

	if errSaveBanNet := s.banRepo.SaveBanNet(ctx, banNet); errSaveBanNet != nil {
		return errors.Join(errSaveBanNet, domain.ErrSaveBan)
	}

	var author domain.Person
	if err := s.personUsecase.GetOrCreatePersonBySteamID(ctx, banNet.SourceID, &author); err != nil {
		return errors.Join(err, domain.ErrFetchSource)
	}

	var target domain.Person
	if err := s.personUsecase.GetOrCreatePersonBySteamID(ctx, banNet.TargetID, &target); err != nil {
		return errors.Join(err, domain.ErrFetchTarget)
	}

	conf := s.configUsecase.Config()

	s.discordUsecase.SendPayload(domain.ChannelModLog, discord.BanCIDRResponse(realCIDR, author, conf.ExtURL(author), target, banNet))

	go func(_ *net.IPNet, reason domain.Reason) {
		foundPlayers := s.stateUsecase.Find("", "", nil, realCIDR)

		if len(foundPlayers) == 0 {
			return
		}

		for _, player := range foundPlayers {
			if errKick := s.stateUsecase.Kick(ctx, player.Player.SID, reason); errKick != nil {
				s.log.Error("Failed to kick player", zap.Error(errKick))
			}
		}
	}(realCIDR, banNet.Reason)

	return nil
}

// BanSteam will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of Config.DefaultExpiration() will be used.
func (s *banUsecase) BanSteam(ctx context.Context, banSteam *domain.BanSteam) error {
	if !banSteam.TargetID.Valid() {
		return errors.Join(errs.ErrInvalidSID, errs.ErrTargetID)
	}

	existing := domain.NewBannedPerson()

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
		var report domain.Report
		if errReport := app.db.GetReport(ctx, reportId, &report); errReport != nil {
			return errors.Join(errReport, errGetBanReport)
		}

		report.ReportStatus = domain.ClosedWithAction
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

	var target domain.Person
	if err := app.db.GetOrCreatePersonBySteamID(ctx, banSteam.TargetID, &target); err != nil {
		return errors.Join(err, errFetchPerson)
	}

	// TODO mute player currently in-game w/o kicking
	if banSteam.BanType == domain.Banned {
		if errKick := state.Kick(ctx, app.State(), banSteam.TargetID, banSteam.Reason); errKick != nil && !errors.Is(errKick, errs.ErrPlayerNotFound) {
			app.log.Error("Failed to kick player", zap.Error(errKick),
				zap.Int64("sid64", banSteam.TargetID.Int64()))
		}

		app.SendPayload(app.Config().Discord.LogChannelID, discord.KickPlayerEmbed(target))
	} else if banSteam.BanType == domain.NoComm {
		if errSilence := state.Silence(ctx, app.State(), banSteam.TargetID, banSteam.Reason); errSilence != nil && !errors.Is(errSilence, errs.ErrPlayerNotFound) {
			app.log.Error("Failed to silence player", zap.Error(errSilence),
				zap.Int64("sid64", banSteam.TargetID.Int64()))
		}
		app.SendPayload(app.Config().Discord.LogChannelID, discord.SilenceEmbed(target))
	}

	return nil
}

func (s *banUsecase) BanSteamGroup(ctx context.Context, banGroup *domain.BanGroup) error {
	//TODO implement me
	panic("implement me")
}

// Unban will set the Current ban to now, making it expired.
// Returns true, nil if the ban exists, and was successfully banned.
// Returns false, nil if the ban does not exist.
func (s *banUsecase) Unban(ctx context.Context, targetSID steamid.SID64, reason string) (bool, error) {
	bannedPerson := domain.NewBannedPerson()
	errGetBan := s.banRepo.GetBanBySteamID(ctx, targetSID, &bannedPerson, false)

	if errGetBan != nil {
		if errors.Is(errGetBan, errs.ErrNoResult) {
			return false, nil
		}

		return false, errors.Join(errGetBan, domain.ErrFailedFetchBan)
	}

	bannedPerson.Deleted = true
	bannedPerson.UnbanReasonText = reason

	if errSave := s.banRepo.SaveBan(ctx, &bannedPerson.BanSteam); errSave != nil {
		return false, errors.Join(errSave, domain.ErrSaveBan)
	}

	var person domain.Person
	if err := s.personUsecase.GetPersonBySteamID(ctx, targetSID, &person); err != nil {
		return false, errors.Join(err, domain.ErrFetchPerson)
	}

	s.discordUsecase.SendPayload(domain.ChannelModLog, discord.UnbanMessage(person))

	// env.Log().Info("Player unbanned", zap.Int64("sid64", targetSID.Int64()), zap.String("reason", reason))

	return true, nil
}
func (s *banUsecase) UnbanASN(ctx context.Context, asnNum string) (bool, error) {
	asNum, errConv := strconv.ParseInt(asnNum, 10, 64)
	if errConv != nil {
		return false, errors.Join(errConv, domain.ErrParseASN)
	}

	var banASN domain.BanASN
	if errGetBanASN := s.banRepo.GetBanASN(ctx, asNum, &banASN); errGetBanASN != nil {
		return false, errors.Join(errGetBanASN, domain.ErrFetchASNBan)
	}

	if errDrop := s.banRepo.DropBanASN(ctx, &banASN); errDrop != nil {
		return false, errors.Join(errDrop, domain.ErrDropASNBan)
	}

	asnNetworks, errGetASNRecords := app.db.GetASNRecordsByNum(ctx, asNum)
	if errGetASNRecords != nil {
		if errors.Is(errGetASNRecords, errs.ErrNoResult) {
			return false, errors.Join(errGetASNRecords, domain.ErrUnknownASN)
		}

		return false, errors.Join(errGetASNRecords, domain.ErrFetchASN)
	}

	s.discordUsecase.SendPayload(domain.ChannelModLog, discord.UnbanASNMessage(asNum, asnNetworks))

	return true, nil
}
func (s *banUsecase) DropBan(ctx context.Context, ban *domain.BanSteam, hardDelete bool) error {
	return s.banRepo.DropBan(ctx, ban, hardDelete)
}

func (s *banUsecase) GetBansSteam(ctx context.Context, filter domain.SteamBansQueryFilter) ([]domain.BannedSteamPerson, int64, error) {
	return s.banRepo.GetBansSteam(ctx, filter)
}

func (s *banUsecase) GetExpiredBans(ctx context.Context) ([]domain.BanSteam, error) {
	return s.banRepo.GetExpiredBans(ctx)
}

func (s *banUsecase) GetBansOlderThan(ctx context.Context, filter domain.QueryFilter, since time.Time) ([]domain.BanSteam, error) {
	return s.banRepo.GetBansOlderThan(ctx, filter, since)
}

func (s *banUsecase) GetBanGroup(ctx context.Context, groupID steamid.GID, banGroup *domain.BanGroup) error {
	return s.banRepo.GetBanGroup(ctx, groupID, banGroup)
}

func (s *banUsecase) GetBanGroupByID(ctx context.Context, banGroupID int64, banGroup *domain.BanGroup) error {
	return s.banRepo.GetBanGroupByID(ctx, banGroupID, banGroup)
}

func (s *banUsecase) GetBanGroups(ctx context.Context, filter domain.GroupBansQueryFilter) ([]domain.BannedGroupPerson, int64, error) {
	return s.banRepo.GetBanGroups(ctx, filter)
}

func (s *banUsecase) SaveBanGroup(ctx context.Context, banGroup *domain.BanGroup) error {
	//TODO implement me
	panic("implement me")
}

func (s *banUsecase) DropBanGroup(ctx context.Context, banGroup *domain.BanGroup) error {
	//TODO implement me
	panic("implement me")
}

func (s *banUsecase) GetBanASN(ctx context.Context, asNum int64, banASN *domain.BanASN) error {
	//TODO implement me
	panic("implement me")
}

func (s *banUsecase) GetBansASN(ctx context.Context, filter domain.ASNBansQueryFilter) ([]domain.BannedASNPerson, int64, error) {
	//TODO implement me
	panic("implement me")
}

func (s *banUsecase) SaveBanASN(ctx context.Context, banASN *domain.BanASN) error {
	//TODO implement me
	panic("implement me")
}

func (s *banUsecase) DropBanASN(ctx context.Context, banASN *domain.BanASN) error {
	//TODO implement me
	panic("implement me")
}

func (s *banUsecase) GetSteamIDsAtIP(ctx context.Context, ipNet *net.IPNet) (steamid.Collection, error) {
	//TODO implement me
	panic("implement me")
}

func (s *banUsecase) GetBanNetByAddress(ctx context.Context, ipAddr net.IP) ([]domain.BanCIDR, error) {
	//TODO implement me
	panic("implement me")
}

func (s *banUsecase) GetBanNetByID(ctx context.Context, netID int64, banNet *domain.BanCIDR) error {
	//TODO implement me
	panic("implement me")
}

func (s *banUsecase) GetBansNet(ctx context.Context, filter domain.CIDRBansQueryFilter) ([]domain.BannedCIDRPerson, int64, error) {
	//TODO implement me
	panic("implement me")
}

func (s *banUsecase) SaveBanNet(ctx context.Context, banNet *domain.BanCIDR) error {
	//TODO implement me
	panic("implement me")
}

func (s *banUsecase) DropBanNet(ctx context.Context, banNet *domain.BanCIDR) error {
	//TODO implement me
	panic("implement me")
}

func (s *banUsecase) GetExpiredNetBans(ctx context.Context) ([]domain.BanCIDR, error) {
	//TODO implement me
	panic("implement me")
}

func (s *banUsecase) GetExpiredASNBans(ctx context.Context) ([]domain.BanASN, error) {
	//TODO implement me
	panic("implement me")
}

// IsOnIPWithBan checks if the address matches an existing user who is currently banned already. This
// function will always fail-open and allow players in if an error occurs.
func (s *banUsecase) IsOnIPWithBan(ctx context.Context, steamID steamid.SID64, address net.IP) (bool, error) {
	existing := domain.NewBannedPerson()
	if errMatch := s.GetBanByLastIP(ctx, address, &existing, false); errMatch != nil {
		if errors.Is(errMatch, errs.ErrNoResult) {
			return false, nil
		}

		return false, errMatch
	}

	duration, errDuration := util.ParseUserStringDuration("10y")
	if errDuration != nil {
		return false, errDuration
	}

	existing.BanSteam.ValidUntil = time.Now().Add(duration)

	if errSave := s.SaveBan(ctx, &existing.BanSteam); errSave != nil {
		env.Log().Error("Could not update previous ban.", zap.Error(errSave))

		return false
	}

	var newBan domain.BanSteam
	if errNewBan := domain.NewBanSteam(ctx,
		domain.StringSID(env.Config().General.Owner.String()),
		domain.StringSID(steamID.String()), duration, domain.Evading, domain.Evading.String(),
		"Connecting from same IP as banned player", domain.System,
		0, domain.Banned, false, &newBan); errNewBan != nil {
		env.Log().Error("Could not create evade ban", zap.Error(errDuration))

		return false
	}

	if errSave := env.BanSteam(ctx, &newBan); errSave != nil {
		env.Log().Error("Could not save evade ban", zap.Error(errSave))

		return false
	}

	return true
}
