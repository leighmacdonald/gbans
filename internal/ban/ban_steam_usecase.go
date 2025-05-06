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
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/datetime"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type banSteamUsecase struct {
	banRepo       domain.BanSteamRepository
	persons       domain.PersonUsecase
	config        domain.ConfigUsecase
	notifications domain.NotificationUsecase
	state         domain.StateUsecase
	reports       domain.ReportUsecase
}

func NewBanSteamUsecase(repository domain.BanSteamRepository, person domain.PersonUsecase,
	config domain.ConfigUsecase, notifications domain.NotificationUsecase, reports domain.ReportUsecase, state domain.StateUsecase,
) domain.BanSteamUsecase {
	return &banSteamUsecase{
		banRepo:       repository,
		persons:       person,
		config:        config,
		notifications: notifications,
		reports:       reports,
		state:         state,
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

		friends, errFriends := steamweb.GetFriendList(ctx, httphelper.NewHTTPClient(), ban.TargetID)
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
	oldState := domain.Open
	if ban.BanID > 0 {
		existing, errExisting := s.GetByBanID(ctx, ban.BanID, true, true)
		if errExisting != nil {
			slog.Error("Failed to get existing ban", log.ErrAttr(errExisting))

			return errExisting
		}

		oldState = existing.AppealState
	}

	if err := s.banRepo.Save(ctx, ban); err != nil {
		return err
	}

	if oldState != ban.AppealState {
		s.notifications.Enqueue(ctx, domain.NewSiteGroupNotification(
			[]domain.Privilege{domain.PModerator, domain.PAdmin},
			domain.SeverityInfo,
			fmt.Sprintf("Ban appeal state changed: %s -> %s", oldState, ban.AppealState),
			ban.Path()))

		s.notifications.Enqueue(ctx, domain.NewSiteUserNotification(
			[]steamid.SteamID{ban.TargetID},
			domain.SeverityInfo,
			fmt.Sprintf("Your mute/ban appeal status has changed: %s -> %s", oldState, ban.AppealState),
			ban.Path()))
	}

	return nil
}

// Ban will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of Config.DefaultExpiration() will be used.
func (s banSteamUsecase) Ban(ctx context.Context, curUser domain.UserProfile, origin domain.Origin, req domain.RequestBanSteamCreate) (domain.BannedSteamPerson, error) {
	var (
		ban domain.BannedSteamPerson
		sid = curUser.GetSteamID()
	)

	// srcds sourced bans provide a source_id to id the admin
	if sourceID, ok := req.SourceSteamID(ctx); ok {
		origin = domain.InGame
		sid = sourceID
	}

	duration, errDuration := datetime.CalcDuration(req.Duration, req.ValidUntil)
	if errDuration != nil {
		return ban, errDuration
	}

	author, errAuthor := s.persons.GetPersonBySteamID(ctx, nil, sid)
	if errAuthor != nil {
		return ban, errAuthor
	}

	targetID, targetIDOk := req.TargetSteamID(ctx)
	if !targetIDOk {
		return ban, domain.ErrTargetID
	}

	var banSteam domain.BanSteam
	if errBanSteam := domain.NewBanSteam(author.SteamID, targetID, duration, req.Reason, req.ReasonText, req.Note,
		origin, req.ReportID, req.BanType, req.IncludeFriends, req.EvadeOk, &banSteam,
	); errBanSteam != nil {
		return ban, errBanSteam
	}

	if !banSteam.TargetID.Valid() {
		return ban, errors.Join(domain.ErrInvalidSID, domain.ErrTargetID)
	}

	existing, errGetExistingBan := s.banRepo.GetBySteamID(ctx, banSteam.TargetID, false, true)
	if errGetExistingBan != nil && !errors.Is(errGetExistingBan, domain.ErrNoResult) {
		return ban, errors.Join(errGetExistingBan, domain.ErrGetBan)
	}

	if existing.BanID > 0 {
		return ban, domain.ErrDuplicate
	}

	if errSave := s.banRepo.Save(ctx, &banSteam); errSave != nil {
		return ban, errors.Join(errSave, domain.ErrSaveBan)
	}

	bannedPerson, errBannedPerson := s.banRepo.GetByBanID(ctx, banSteam.BanID, false, true)
	if errBannedPerson != nil {
		return ban, errors.Join(errBannedPerson, domain.ErrSaveBan)
	}

	expIn := "Permanent"
	expAt := "Permanent"

	if banSteam.ValidUntil.Year()-time.Now().Year() < 5 {
		expIn = datetime.FmtDuration(banSteam.ValidUntil)
		expAt = datetime.FmtTimeShort(banSteam.ValidUntil)
	}

	s.notifications.Enqueue(ctx, domain.NewDiscordNotification(domain.ChannelBanLog, discord.BanSteamResponse(bannedPerson)))

	siteMsg := domain.NewSiteUserNotificationWithAuthor(
		[]domain.Privilege{domain.PModerator, domain.PAdmin},
		domain.SeverityInfo,
		fmt.Sprintf("User banned (steam): %s Duration: %s Author: %s",
			bannedPerson.TargetPersonaname, expIn, bannedPerson.SourcePersonaname),
		banSteam.Path(),
		author.ToUserProfile(),
	)

	s.notifications.Enqueue(ctx, siteMsg)

	s.notifications.Enqueue(ctx, domain.NewSiteUserNotification(
		[]steamid.SteamID{bannedPerson.TargetID},
		domain.SeverityWarn,
		fmt.Sprintf("You have been %s, Reason: %s, Duration: %s, Ends: %s", bannedPerson.BanType, bannedPerson.Reason.String(), expIn, expAt),
		banSteam.Path(),
	))

	// Close the report if the ban was attached to one
	if banSteam.ReportID > 0 {
		if _, errSaveReport := s.reports.SetReportStatus(ctx, banSteam.ReportID, curUser, domain.ClosedWithAction); errSaveReport != nil {
			return ban, errors.Join(errSaveReport, domain.ErrReportStateUpdate)
		}
	}

	target, err := s.persons.GetOrCreatePersonBySteamID(ctx, nil, banSteam.TargetID)
	if err != nil {
		return ban, errors.Join(err, domain.ErrFetchPerson)
	}

	switch banSteam.BanType {
	case domain.Banned:
		if errKick := s.state.Kick(ctx, banSteam.TargetID, banSteam.Reason); errKick != nil && !errors.Is(errKick, domain.ErrPlayerNotFound) {
			slog.Error("Failed to kick player", log.ErrAttr(errKick),
				slog.Int64("sid64", banSteam.TargetID.Int64()))
		} else {
			s.notifications.Enqueue(ctx, domain.NewDiscordNotification(domain.ChannelKickLog, discord.KickPlayerEmbed(target)))
		}
	case domain.NoComm:
		if errSilence := s.state.Silence(ctx, banSteam.TargetID, banSteam.Reason); errSilence != nil && !errors.Is(errSilence, domain.ErrPlayerNotFound) {
			slog.Error("Failed to silence player", log.ErrAttr(errSilence),
				slog.Int64("sid64", banSteam.TargetID.Int64()))
		} else {
			s.notifications.Enqueue(ctx, domain.NewDiscordNotification(domain.ChannelKickLog, discord.MuteMessage(bannedPerson)))
		}
	default:
		return ban, domain.ErrInvalidBanType
	}

	return s.GetByBanID(ctx, banSteam.BanID, false, true)
}

// Unban will set the Current ban to now, making it expired.
// Returns true, nil if the ban exists, and was successfully banned.
// Returns false, nil if the ban does not exist.
func (s banSteamUsecase) Unban(ctx context.Context, targetSID steamid.SteamID, reason string, author domain.UserProfile) (bool, error) {
	bannedPerson, errGetBan := s.banRepo.GetBySteamID(ctx, targetSID, false, true)

	if errGetBan != nil {
		if errors.Is(errGetBan, domain.ErrNoResult) {
			return false, nil
		}

		return false, errors.Join(errGetBan, domain.ErrGetBan)
	}

	bannedPerson.Deleted = true
	bannedPerson.UnbanReasonText = reason

	if errSave := s.banRepo.Save(ctx, &bannedPerson.BanSteam); errSave != nil {
		return false, errors.Join(errSave, domain.ErrSaveBan)
	}

	person, err := s.persons.GetPersonBySteamID(ctx, nil, targetSID)
	if err != nil {
		return false, errors.Join(err, domain.ErrFetchPerson)
	}

	s.notifications.Enqueue(ctx, domain.NewDiscordNotification(domain.ChannelBanLog, discord.UnbanMessage(s.config, person)))

	s.notifications.Enqueue(ctx, domain.NewSiteGroupNotificationWithAuthor(
		[]domain.Privilege{domain.PModerator, domain.PAdmin},
		domain.SeverityInfo,
		fmt.Sprintf("A user has been unbanned: %s, Reason: %s", bannedPerson.TargetPersonaname, reason),
		bannedPerson.Path(),
		author,
	))

	s.notifications.Enqueue(ctx, domain.NewSiteUserNotification(
		[]steamid.SteamID{bannedPerson.TargetID},
		domain.SeverityInfo,
		"You have been unmuted/unbanned",
		bannedPerson.Path(),
	))

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
func (s banSteamUsecase) CheckEvadeStatus(ctx context.Context, curUser domain.UserProfile, steamID steamid.SteamID, address netip.Addr) (bool, error) {
	existing, errMatch := s.GetByLastIP(ctx, address, false, false)
	if errMatch != nil {
		if errors.Is(errMatch, domain.ErrNoResult) {
			return false, nil
		}

		return false, errMatch
	}

	if existing.BanType == domain.NoComm {
		// Currently we do not ban for mute evasion.
		// TODO make this configurable
		return false, errMatch
	}

	duration, errDuration := datetime.ParseUserStringDuration("10y")
	if errDuration != nil {
		return false, errDuration
	}

	existing.Note += " Previous expiry: " + existing.ValidUntil.Format(time.DateTime)
	existing.ValidUntil = time.Now().Add(duration)

	if errSave := s.Save(ctx, &existing.BanSteam); errSave != nil {
		slog.Error("Could not update previous ban.", log.ErrAttr(errSave))

		return false, errSave
	}

	config := s.config.Config()
	owner := steamid.New(config.Owner)

	req := domain.RequestBanSteamCreate{
		SourceIDField:  domain.SourceIDField{SourceID: owner.String()},
		TargetIDField:  domain.TargetIDField{TargetID: steamID.String()},
		Duration:       "10y",
		BanType:        domain.Banned,
		Reason:         domain.Evading,
		Note:           fmt.Sprintf("Connecting from same IP as banned player.\n\nEvasion of: [#%d](%s)", existing.BanID, config.ExtURL(existing)),
		IncludeFriends: false,
		EvadeOk:        false,
	}

	_, errSave := s.Ban(ctx, curUser, domain.System, req)
	if errSave != nil {
		if errors.Is(errSave, domain.ErrDuplicate) {
			// Already banned
			return true, nil
		}

		slog.Error("Could not save evade ban", log.ErrAttr(errSave))

		return false, errSave
	}

	return true, nil
}
