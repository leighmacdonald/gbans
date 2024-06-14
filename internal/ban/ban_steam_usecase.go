package ban

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"time"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/datetime"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type banSteamUsecase struct {
	banRepo domain.BanSteamRepository
	persons domain.PersonUsecase
	config  domain.ConfigUsecase
	discord domain.DiscordUsecase
	state   domain.StateUsecase
	reports domain.ReportUsecase
}

func NewBanSteamUsecase(repository domain.BanSteamRepository, person domain.PersonUsecase,
	config domain.ConfigUsecase, discord domain.DiscordUsecase, reports domain.ReportUsecase, state domain.StateUsecase,
) domain.BanSteamUsecase {
	return &banSteamUsecase{
		banRepo: repository,
		persons: person,
		config:  config,
		discord: discord,
		reports: reports,
		state:   state,
	}
}

func (s banSteamUsecase) UpdateCache(ctx context.Context) error {
	bans, errBans := s.Get(ctx, domain.SteamBansQueryFilter{Deleted: false})
	if errBans != nil {
		return errBans
	}

	if err := s.banRepo.TruncateCache(ctx); err != nil {
		return err
	}

	for _, ban := range bans {
		if !ban.IncludeFriends || ban.Deleted || ban.ValidUntil.Before(time.Now()) {
			continue
		}

		friends, errFriends := steamweb.GetFriendList(ctx, ban.TargetID)
		if errFriends != nil {
			continue
		}

		var list []int64
		for _, friend := range friends {
			list = append(list, friend.SteamID.Int64())
		}

		if err := s.banRepo.InsertCache(ctx, ban.TargetID, list); err != nil {
			return err
		}
	}

	return nil
}

func (s banSteamUsecase) Stats(ctx context.Context, stats *domain.Stats) error {
	return s.banRepo.Stats(ctx, stats)
}

func (s banSteamUsecase) GetBySteamID(ctx context.Context, sid64 steamid.SteamID, deletedOk bool, evadeOK bool) (domain.BannedSteamPerson, error) {
	return s.banRepo.GetBySteamID(ctx, sid64, deletedOk, evadeOK)
}

func (s banSteamUsecase) GetByBanID(ctx context.Context, banID int64, deletedOk bool, evadeOK bool) (domain.BannedSteamPerson, error) {
	return s.banRepo.GetByBanID(ctx, banID, deletedOk, evadeOK)
}

func (s banSteamUsecase) GetByLastIP(ctx context.Context, lastIP netip.Addr, deletedOk bool, evadeOK bool) (domain.BannedSteamPerson, error) {
	return s.banRepo.GetByLastIP(ctx, lastIP, deletedOk, evadeOK)
}

func (s banSteamUsecase) Save(ctx context.Context, ban *domain.BanSteam) error {
	return s.banRepo.Save(ctx, ban)
}

// Ban will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of Config.DefaultExpiration() will be used.
func (s banSteamUsecase) Ban(ctx context.Context, curUser domain.PersonInfo, banSteam *domain.BanSteam) error {
	if !banSteam.TargetID.Valid() {
		return errors.Join(domain.ErrInvalidSID, domain.ErrTargetID)
	}

	existing, errGetExistingBan := s.banRepo.GetBySteamID(ctx, banSteam.TargetID, false, true)

	if existing.BanID > 0 {
		return domain.ErrDuplicate
	}

	if errGetExistingBan != nil && !errors.Is(errGetExistingBan, domain.ErrNoResult) {
		return errors.Join(errGetExistingBan, domain.ErrFailedFetchBan)
	}

	if errSave := s.banRepo.Save(ctx, banSteam); errSave != nil {
		return errors.Join(errSave, domain.ErrSaveBan)
	}

	s.discord.SendPayload(domain.ChannelBanLog, discord.BanSteamResponse(*banSteam, curUser))

	updateAppealState := func(reportId int64) error {
		report, errReport := s.reports.GetReport(ctx, curUser, reportId)
		if errReport != nil {
			return errors.Join(errReport, domain.ErrGetBanReport)
		}

		report.ReportStatus = domain.ClosedWithAction
		if errSaveReport := s.reports.SaveReport(ctx, &report.Report); errSaveReport != nil {
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

	target, err := s.persons.GetOrCreatePersonBySteamID(ctx, banSteam.TargetID)
	if err != nil {
		return errors.Join(err, domain.ErrFetchPerson)
	}

	// TODO mute player currently in-game w/o kicking
	if banSteam.BanType == domain.Banned {
		if errKick := s.state.Kick(ctx, banSteam.TargetID, banSteam.Reason); errKick != nil && !errors.Is(errKick, domain.ErrPlayerNotFound) {
			slog.Error("Failed to kick player", log.ErrAttr(errKick),
				slog.Int64("sid64", banSteam.TargetID.Int64()))
		}

		s.discord.SendPayload(domain.ChannelBanLog, discord.KickPlayerEmbed(target))
	} else if banSteam.BanType == domain.NoComm {
		if errSilence := s.state.Silence(ctx, banSteam.TargetID, banSteam.Reason); errSilence != nil && !errors.Is(errSilence, domain.ErrPlayerNotFound) {
			slog.Error("Failed to silence player", log.ErrAttr(errSilence),
				slog.Int64("sid64", banSteam.TargetID.Int64()))
		}

		s.discord.SendPayload(domain.ChannelBanLog, discord.SilenceEmbed(target))
	}

	return nil
}

// Unban will set the Current ban to now, making it expired.
// Returns true, nil if the ban exists, and was successfully banned.
// Returns false, nil if the ban does not exist.
func (s banSteamUsecase) Unban(ctx context.Context, targetSID steamid.SteamID, reason string) (bool, error) {
	bannedPerson, errGetBan := s.banRepo.GetBySteamID(ctx, targetSID, false, true)

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

	person, err := s.persons.GetPersonBySteamID(ctx, targetSID)
	if err != nil {
		return false, errors.Join(err, domain.ErrFetchPerson)
	}

	s.discord.SendPayload(domain.ChannelBanLog, discord.UnbanMessage(s.config, person))

	return true, nil
}

func (s banSteamUsecase) Delete(ctx context.Context, ban *domain.BanSteam, hardDelete bool) error {
	return s.banRepo.Delete(ctx, ban, hardDelete)
}

func (s banSteamUsecase) Get(ctx context.Context, filter domain.SteamBansQueryFilter) ([]domain.BannedSteamPerson, error) {
	return s.banRepo.Get(ctx, filter)
}

func (s banSteamUsecase) Expired(ctx context.Context) ([]domain.BanSteam, error) {
	return s.banRepo.ExpiredBans(ctx)
}

func (s banSteamUsecase) GetOlderThan(ctx context.Context, filter domain.QueryFilter, since time.Time) ([]domain.BanSteam, error) {
	return s.banRepo.GetOlderThan(ctx, filter, since)
}

// CheckEvadeStatus checks if the address matches an existing user who is currently banned already. This
// function will always fail-open and allow players in if an error occurs.
func (s banSteamUsecase) CheckEvadeStatus(ctx context.Context, curUser domain.PersonInfo, steamID steamid.SteamID, address netip.Addr) (bool, error) {
	existing, errMatch := s.GetByLastIP(ctx, address, false, false)
	if errMatch != nil {
		if errors.Is(errMatch, domain.ErrNoResult) {
			return false, nil
		}

		return false, errMatch
	}

	if existing.BanType == domain.NoComm {
		return false, errMatch
	}

	duration, errDuration := datetime.ParseUserStringDuration("10y")
	if errDuration != nil {
		return false, errDuration
	}

	existing.Note += " Previous expiry: " + existing.BanSteam.ValidUntil.Format(time.DateTime)
	existing.BanSteam.ValidUntil = time.Now().Add(duration)

	if errSave := s.Save(ctx, &existing.BanSteam); errSave != nil {
		slog.Error("Could not update previous ban.", log.ErrAttr(errSave))

		return false, errSave
	}

	var newBan domain.BanSteam
	if errNewBan := domain.NewBanSteam(steamid.New(s.config.Config().Owner),
		steamID, duration, domain.Evading, domain.Evading.String(),
		"Connecting from same IP as banned player. ", domain.System,
		0, domain.Banned, false, false, &newBan); errNewBan != nil {
		slog.Error("Could not create evade ban", log.ErrAttr(errNewBan))

		return false, errNewBan
	}

	config := s.config.Config()

	newBan.Note += fmt.Sprintf("\nEvasion of: [#%d](%s)", existing.BanID, config.ExtURL(existing))

	if errSave := s.Ban(ctx, curUser, &newBan); errSave != nil {
		if errors.Is(errSave, domain.ErrDuplicate) {
			// Already banned
			return true, nil
		}

		slog.Error("Could not save evade ban", log.ErrAttr(errSave))

		return false, errSave
	}

	return true, nil
}
