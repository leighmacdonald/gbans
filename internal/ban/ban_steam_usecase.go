package ban

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type banSteamUsecase struct {
	banRepo        domain.BanSteamRepository
	personUsecase  domain.PersonUsecase
	configUsecase  domain.ConfigUsecase
	discordUsecase domain.DiscordUsecase
	stateUsecase   domain.StateUsecase
	reportUsecase  domain.ReportUsecase
	log            *zap.Logger
	friends        *SteamFriends
}

func NewBanSteamUsecase(log *zap.Logger, repository domain.BanSteamRepository, personUsecase domain.PersonUsecase,
	configUsecase domain.ConfigUsecase, discordUsecase domain.DiscordUsecase, groupUsecase domain.BanGroupUsecase,
	reportUsecase domain.ReportUsecase, stateUsecase domain.StateUsecase,
) domain.BanSteamUsecase {
	bu := &banSteamUsecase{log: log, banRepo: repository, personUsecase: personUsecase, configUsecase: configUsecase, discordUsecase: discordUsecase, reportUsecase: reportUsecase, stateUsecase: stateUsecase}
	friendTracker := NewSteamFriends(log, bu, groupUsecase)

	bu.friends = friendTracker

	return bu
}

func (s *banSteamUsecase) Stats(ctx context.Context, stats *domain.Stats) error {
	return s.banRepo.Stats(ctx, stats)
}

func (s *banSteamUsecase) IsFriendBanned(steamID steamid.SID64) (steamid.SID64, bool) {
	return s.friends.IsMember(steamID)
}

func (s *banSteamUsecase) GetBySteamID(ctx context.Context, sid64 steamid.SID64, deletedOk bool) (domain.BannedSteamPerson, error) {
	return s.banRepo.GetBySteamID(ctx, sid64, deletedOk)
}

func (s *banSteamUsecase) GetByBanID(ctx context.Context, banID int64, deletedOk bool) (domain.BannedSteamPerson, error) {
	return s.banRepo.GetByBanID(ctx, banID, deletedOk)
}

func (s *banSteamUsecase) GetByLastIP(ctx context.Context, lastIP net.IP, deletedOk bool) (domain.BannedSteamPerson, error) {
	return s.banRepo.GetByLastIP(ctx, lastIP, deletedOk)
}

func (s *banSteamUsecase) Save(ctx context.Context, ban *domain.BanSteam) error {
	return s.banRepo.Save(ctx, ban)
}

// Ban will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of Config.DefaultExpiration() will be used.
func (s *banSteamUsecase) Ban(ctx context.Context, curUser domain.PersonInfo, banSteam *domain.BanSteam) error {
	if !banSteam.TargetID.Valid() {
		return errors.Join(domain.ErrInvalidSID, domain.ErrTargetID)
	}

	existing, errGetExistingBan := s.banRepo.GetBySteamID(ctx, banSteam.TargetID, false)

	if existing.BanID > 0 {
		return domain.ErrDuplicate
	}

	if errGetExistingBan != nil && !errors.Is(errGetExistingBan, domain.ErrNoResult) {
		return errors.Join(errGetExistingBan, domain.ErrFailedFetchBan)
	}

	if errSave := s.banRepo.Save(ctx, banSteam); errSave != nil {
		return errors.Join(errSave, domain.ErrSaveBan)
	}

	updateAppealState := func(reportId int64) error {
		report, errReport := s.reportUsecase.GetReport(ctx, curUser, reportId)
		if errReport != nil {
			return errors.Join(errReport, domain.ErrGetBanReport)
		}

		report.ReportStatus = domain.ClosedWithAction
		if errSaveReport := s.reportUsecase.SaveReport(ctx, &report); errSaveReport != nil {
			return errors.Join(errSaveReport, domain.ErrReportStateUpdate)
		}

		return nil
	}

	// Close the report if the ban was attached to one
	if banSteam.ReportID > 0 {
		if errRep := updateAppealState(banSteam.ReportID); errRep != nil {
			return errRep
		}
	}

	target, err := s.personUsecase.GetOrCreatePersonBySteamID(ctx, banSteam.TargetID)
	if err != nil {
		return errors.Join(err, domain.ErrFetchPerson)
	}

	// TODO mute player currently in-game w/o kicking
	if banSteam.BanType == domain.Banned {
		if errKick := s.stateUsecase.Kick(ctx, banSteam.TargetID, banSteam.Reason); errKick != nil && !errors.Is(errKick, domain.ErrPlayerNotFound) {
			s.log.Error("Failed to kick player", zap.Error(errKick),
				zap.Int64("sid64", banSteam.TargetID.Int64()))
		}

		s.discordUsecase.SendPayload(domain.ChannelModLog, discord.KickPlayerEmbed(target))
	} else if banSteam.BanType == domain.NoComm {
		if errSilence := s.stateUsecase.Silence(ctx, banSteam.TargetID, banSteam.Reason); errSilence != nil && !errors.Is(errSilence, domain.ErrPlayerNotFound) {
			s.log.Error("Failed to silence player", zap.Error(errSilence),
				zap.Int64("sid64", banSteam.TargetID.Int64()))
		}

		s.discordUsecase.SendPayload(domain.ChannelModLog, discord.SilenceEmbed(target))
	}

	return nil
}

// Unban will set the Current ban to now, making it expired.
// Returns true, nil if the ban exists, and was successfully banned.
// Returns false, nil if the ban does not exist.
func (s *banSteamUsecase) Unban(ctx context.Context, targetSID steamid.SID64, reason string) (bool, error) {
	bannedPerson, errGetBan := s.banRepo.GetBySteamID(ctx, targetSID, false)

	if errGetBan != nil {
		if errors.Is(errGetBan, domain.ErrNoResult) {
			return false, nil
		}

		return false, errors.Join(errGetBan, domain.ErrFailedFetchBan)
	}

	bannedPerson.Deleted = true
	bannedPerson.UnbanReasonText = reason

	if errSave := s.banRepo.Save(ctx, &bannedPerson.BanSteam); errSave != nil {
		return false, errors.Join(errSave, domain.ErrSaveBan)
	}

	person, err := s.personUsecase.GetPersonBySteamID(ctx, targetSID)
	if err != nil {
		return false, errors.Join(err, domain.ErrFetchPerson)
	}

	s.discordUsecase.SendPayload(domain.ChannelModLog, discord.UnbanMessage(person))

	// env.Log().Info("Player unbanned", zap.Int64("sid64", targetSID.Int64()), zap.String("reason", reason))

	return true, nil
}

func (s *banSteamUsecase) Delete(ctx context.Context, ban *domain.BanSteam, hardDelete bool) error {
	return s.banRepo.Delete(ctx, ban, hardDelete)
}

func (s *banSteamUsecase) Get(ctx context.Context, filter domain.SteamBansQueryFilter) ([]domain.BannedSteamPerson, int64, error) {
	return s.banRepo.Get(ctx, filter)
}

func (s *banSteamUsecase) Expired(ctx context.Context) ([]domain.BanSteam, error) {
	return s.banRepo.ExpiredBans(ctx)
}

func (s *banSteamUsecase) GetOlderThan(ctx context.Context, filter domain.QueryFilter, since time.Time) ([]domain.BanSteam, error) {
	return s.banRepo.GetOlderThan(ctx, filter, since)
}

// IsOnIPWithBan checks if the address matches an existing user who is currently banned already. This
// function will always fail-open and allow players in if an error occurs.
func (s *banSteamUsecase) IsOnIPWithBan(ctx context.Context, curUser domain.PersonInfo, steamID steamid.SID64, address net.IP) (bool, error) {
	existing, errMatch := s.GetByLastIP(ctx, address, false)
	if errMatch != nil {
		if errors.Is(errMatch, domain.ErrNoResult) {
			return false, nil
		}

		return false, errMatch
	}

	duration, errDuration := util.ParseUserStringDuration("10y")
	if errDuration != nil {
		return false, errDuration
	}

	existing.BanSteam.ValidUntil = time.Now().Add(duration)

	if errSave := s.Save(ctx, &existing.BanSteam); errSave != nil {
		s.log.Error("Could not update previous ban.", zap.Error(errSave))

		return false, errSave
	}

	var newBan domain.BanSteam
	if errNewBan := domain.NewBanSteam(ctx,
		domain.StringSID(s.configUsecase.Config().General.Owner.String()),
		domain.StringSID(steamID.String()), duration, domain.Evading, domain.Evading.String(),
		"Connecting from same IP as banned player", domain.System,
		0, domain.Banned, false, &newBan); errNewBan != nil {
		s.log.Error("Could not create evade ban", zap.Error(errDuration))

		return false, errNewBan
	}

	if errSave := s.Ban(ctx, curUser, &newBan); errSave != nil {
		s.log.Error("Could not save evade ban", zap.Error(errSave))

		return false, errSave
	}

	return true, nil
}
