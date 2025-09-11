package ban

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/person/permission"
	"github.com/leighmacdonald/gbans/internal/report"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/datetime"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type BansQueryFilter struct {
	domain.TargetIDField
	Deleted bool `json:"deleted"`
}

type BanUsecase struct {
	banRepo       *BanRepository
	persons       *person.PersonUsecase
	config        *config.ConfigUsecase
	notifications notification.NotificationUsecase
	state         state.StateUsecase
	reports       report.ReportUsecase
	tfAPI         *thirdparty.TFAPI
}

func NewBanUsecase(repository *BanRepository, person *person.PersonUsecase,
	config *config.ConfigUsecase, notifications notification.NotificationUsecase, reports report.ReportUsecase, state state.StateUsecase,
	tfAPI *thirdparty.TFAPI,
) *BanUsecase {
	return &BanUsecase{
		banRepo:       repository,
		persons:       person,
		config:        config,
		notifications: notifications,
		reports:       reports,
		state:         state,
		tfAPI:         tfAPI,
	}
}

func (s *BanUsecase) UpdateCache(ctx context.Context) error {
	bans, errBans := s.banRepo.Query(ctx, QueryOpts{})
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

		friends, errFriends := s.tfAPI.Friends(ctx, ban.TargetID)
		if errFriends != nil {
			continue
		}

		var list []int64
		for _, friend := range friends {
			sid := steamid.New(friend.SteamId)
			list = append(list, sid.Int64())
		}

		if err := s.banRepo.InsertCache(ctx, ban.TargetID, list); err != nil {
			return err
		}
	}

	return nil
}

func (s *BanUsecase) Query(ctx context.Context, opts QueryOpts) ([]BannedPerson, error) {
	// TODO FIXME
	return s.banRepo.Get(ctx, BansQueryFilter{})
}

func (s *BanUsecase) Stats(ctx context.Context, stats *Stats) error {
	return s.banRepo.Stats(ctx, stats)
}

func (s *BanUsecase) Save(ctx context.Context, ban *Ban) error {
	oldState := Open
	if ban.BanID > 0 {
		existing, errExisting := s.banRepo.Query(ctx, QueryOpts{
			BanID:   ban.BanID,
			EvadeOk: true,
			Deleted: true,
		})
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
		s.notifications.Enqueue(ctx, notification.NewSiteGroupNotification(
			[]permission.Privilege{permission.PModerator, permission.PAdmin},
			notification.SeverityInfo,
			fmt.Sprintf("Ban appeal state changed: %s -> %s", oldState, ban.AppealState),
			ban.Path()))

		s.notifications.Enqueue(ctx, notification.NewSiteUserNotification(
			[]steamid.SteamID{ban.TargetID},
			notification.SeverityInfo,
			fmt.Sprintf("Your mute/ban appeal status has changed: %s -> %s", oldState, ban.AppealState),
			ban.Path()))
	}

	return nil
}

// Ban will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of Config.DefaultExpiration() will be used.
func (s *BanUsecase) Ban(ctx context.Context, curUser person.UserProfile, origin Origin, req BanOpts) (BannedPerson, error) {
	var (
		ban BannedPerson
		sid = curUser.GetSteamID()
	)

	// if banNet.CIDR == "" {
	// 	return domain.ErrCIDRMissing
	// }

	// _, realCIDR, errCIDR := net.ParseCIDR(banNet.CIDR)
	// if errCIDR != nil {
	// 	return errors.Join(errCIDR, domain.ErrNetworkInvalidIP)
	// }

	// srcds sourced bans provide a source_id to id the admin
	if sourceID, ok := req.SourceSteamID(ctx); ok {
		origin = InGame
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

	var banSteam Ban
	if errBanSteam := NewBan(author.SteamID, targetID, duration, req.Reason, req.ReasonText, req.Note,
		origin, req.ReportID, req.BanType, req.IncludeFriends, req.EvadeOk, &banSteam,
	); errBanSteam != nil {
		return ban, errBanSteam
	}

	if !banSteam.TargetID.Valid() {
		return ban, errors.Join(domain.ErrInvalidSID, domain.ErrTargetID)
	}

	existing, errGetExistingBan := s.banRepo.GetBySteamID(ctx, banSteam.TargetID, false, true)
	if errGetExistingBan != nil && !errors.Is(errGetExistingBan, database.ErrNoResult) {
		return ban, errors.Join(errGetExistingBan, domain.ErrGetBan)
	}

	if existing.BanID > 0 {
		return ban, database.ErrDuplicate // TODO better error
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

	s.notifications.Enqueue(ctx, notification.NewDiscordNotification(discord.ChannelBanLog, discord.BanSteamResponse(bannedPerson)))

	siteMsg := notification.NewSiteUserNotificationWithAuthor(
		[]permission.Privilege{permission.PModerator, permission.PAdmin},
		notification.SeverityInfo,
		fmt.Sprintf("User banned (steam): %s Duration: %s Author: %s",
			bannedPerson.TargetPersonaname, expIn, bannedPerson.SourcePersonaname),
		banSteam.Path(),
		author.ToUserProfile(),
	)

	s.notifications.Enqueue(ctx, siteMsg)

	s.notifications.Enqueue(ctx, notification.NewSiteUserNotification(
		[]steamid.SteamID{bannedPerson.TargetID},
		notification.SeverityWarn,
		fmt.Sprintf("You have been %s, Reason: %s, Duration: %s, Ends: %s", bannedPerson.BanType, bannedPerson.Reason.String(), expIn, expAt),
		banSteam.Path(),
	))

	// Close the report if the ban was attached to one
	if banSteam.ReportID > 0 {
		if _, errSaveReport := s.reports.SetReportStatus(ctx, banSteam.ReportID, curUser, ClosedWithAction); errSaveReport != nil {
			return ban, errors.Join(errSaveReport, domain.ErrReportStateUpdate)
		}
	}

	target, err := s.persons.GetOrCreatePersonBySteamID(ctx, nil, banSteam.TargetID)
	if err != nil {
		return ban, errors.Join(err, domain.ErrFetchPerson)
	}

	switch banSteam.BanType {
	case Banned:
		if errKick := s.state.Kick(ctx, banSteam.TargetID, banSteam.Reason); errKick != nil && !errors.Is(errKick, domain.ErrPlayerNotFound) {
			slog.Error("Failed to kick player", log.ErrAttr(errKick),
				slog.Int64("sid64", banSteam.TargetID.Int64()))
		} else {
			s.notifications.Enqueue(ctx, notification.NewDiscordNotification(domain.ChannelKickLog, discord.KickPlayerEmbed(target)))
		}
	case NoComm:
		if errSilence := s.state.Silence(ctx, banSteam.TargetID, banSteam.Reason); errSilence != nil && !errors.Is(errSilence, domain.ErrPlayerNotFound) {
			slog.Error("Failed to silence player", log.ErrAttr(errSilence),
				slog.Int64("sid64", banSteam.TargetID.Int64()))
		} else {
			s.notifications.Enqueue(ctx, notification.NewDiscordNotification(domain.ChannelKickLog, discord.MuteMessage(bannedPerson)))
		}
	default:
		return ban, ErrInvalidBanType
	}

	return s.GetByBanID(ctx, banSteam.BanID, false, true)
}

// Unban will set the Current ban to now, making it expired.
// Returns true, nil if the ban exists, and was successfully banned.
// Returns false, nil if the ban does not exist.
func (s *BanUsecase) Unban(ctx context.Context, targetSID steamid.SteamID, reason string, author person.UserProfile) (bool, error) {
	bannedPerson, errGetBan := s.banRepo.GetBySteamID(ctx, targetSID, false, true)

	if errGetBan != nil {
		if errors.Is(errGetBan, database.ErrNoResult) {
			return false, nil
		}

		return false, errors.Join(errGetBan, domain.ErrGetBan)
	}

	bannedPerson.Deleted = true
	bannedPerson.UnbanReasonText = reason

	if errSave := s.banRepo.Save(ctx, &bannedPerson.Ban); errSave != nil {
		return false, errors.Join(errSave, domain.ErrSaveBan)
	}

	person, err := s.persons.GetPersonBySteamID(ctx, nil, targetSID)
	if err != nil {
		return false, errors.Join(err, domain.ErrFetchPerson)
	}

	s.notifications.Enqueue(ctx, notification.NewDiscordNotification(domain.ChannelBanLog, discord.UnbanMessage(s.config, person)))

	s.notifications.Enqueue(ctx, notification.NewSiteGroupNotificationWithAuthor(
		[]permission.Privilege{permission.PModerator, permission.PAdmin},
		notification.SeverityInfo,
		fmt.Sprintf("A user has been unbanned: %s, Reason: %s", bannedPerson.TargetPersonaname, reason),
		bannedPerson.Path(),
		author,
	))

	s.notifications.Enqueue(ctx, notification.NewSiteUserNotification(
		[]steamid.SteamID{bannedPerson.TargetID},
		notification.SeverityInfo,
		"You have been unmuted/unbanned",
		bannedPerson.Path(),
	))

	return true, nil
}

func (s *BanUsecase) Delete(ctx context.Context, ban *Ban, hardDelete bool) error {
	return s.banRepo.Delete(ctx, ban, hardDelete)
}

func (s *BanUsecase) Get(ctx context.Context, filter BansQueryFilter) ([]BannedSteamPerson, error) {
	return s.banRepo.Get(ctx, filter)
}

func (s *BanUsecase) Expired(ctx context.Context) ([]Ban, error) {
	return s.banRepo.ExpiredBans(ctx)
}

func (s *BanUsecase) GetOlderThan(ctx context.Context, filter domain.QueryFilter, since time.Time) ([]Ban, error) {
	return s.banRepo.GetOlderThan(ctx, filter, since)
}

// CheckEvadeStatus checks if the address matches an existing user who is currently banned already. This
// function will always fail-open and allow players in if an error occurs.
func (s *BanUsecase) CheckEvadeStatus(ctx context.Context, curUser person.UserProfile, steamID steamid.SteamID, address netip.Addr) (bool, error) {
	existing, errMatch := s.GetByLastIP(ctx, address, false, false)
	if errMatch != nil {
		if errors.Is(errMatch, database.ErrNoResult) {
			return false, nil
		}

		return false, errMatch
	}

	if existing.BanType == NoComm {
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

	if errSave := s.Save(ctx, &existing.Ban); errSave != nil {
		slog.Error("Could not update previous ban.", log.ErrAttr(errSave))

		return false, errSave
	}

	config := s.config.Config()
	owner := steamid.New(config.Owner)

	req := domain.RequestBanCreate{
		SourceIDField:  domain.SourceIDField{SourceID: owner.String()},
		TargetIDField:  domain.TargetIDField{TargetID: steamID.String()},
		Duration:       "10y",
		BanType:        Banned,
		Reason:         Evading,
		Note:           fmt.Sprintf("Connecting from same IP as banned player.\n\nEvasion of: [#%d](%s)", existing.BanID, config.ExtURL(existing)),
		IncludeFriends: false,
		EvadeOk:        false,
	}

	_, errSave := s.Ban(ctx, curUser, System, req)
	if errSave != nil {
		if errors.Is(errSave, database.ErrDuplicate) {
			// Already banned
			return true, nil
		}

		slog.Error("Could not save evade ban", log.ErrAttr(errSave))

		return false, errSave
	}

	return true, nil
}

func (s *BanUsecase) UpdateCache(ctx context.Context) error {
	groups, errGroups := s.Get(ctx, domain.GroupBansQueryFilter{Deleted: false})
	if errGroups != nil {
		return errGroups
	}

	if err := s.repository.TruncateCache(ctx); err != nil {
		return err
	}

	client := httphelper.NewHTTPClient()

	for _, group := range groups {
		listURL := fmt.Sprintf("https://steamcommunity.com/gid/%d/memberslistxml/?xml=1", group.GroupID.Int64())

		req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
		if errReq != nil {
			return errors.Join(errReq, domain.ErrRequestCreate)
		}

		resp, errResp := client.Do(req)
		if errResp != nil {
			return errors.Join(errResp, domain.ErrRequestPerform)
		}

		var list SteamGroupInfo

		decoder := xml.NewDecoder(resp.Body)
		if err := decoder.Decode(&list); err != nil {
			_ = resp.Body.Close()

			return errors.Join(err, domain.ErrRequestDecode)
		}

		_ = resp.Body.Close()

		groupID := steamid.New(list.GroupID64)
		if !groupID.Valid() {
			return domain.ErrInvalidSID
		}

		for _, member := range list.Members.SteamID64 {
			steamID := steamid.New(member)
			if !steamID.Valid() {
				continue
			}

			// Statisfy FK
			_, errCreate := s.persons.GetOrCreatePersonBySteamID(ctx, nil, steamID)
			if errCreate != nil {
				return errCreate
			}
		}

		if err := s.banRepo.InsertCache(ctx, groupID, list.Members.SteamID64); err != nil {
			return err
		}
	}

	return nil
}

func (s *BanUsecase) GetMembersList(ctx context.Context, parentID int64, list *MembersList) error {
	return s.banRepo.GetMembersList(ctx, parentID, list)
}

func (s *BanUsecase) SaveMembersList(ctx context.Context, list *MembersList) error {
	return s.banRepo.SaveMembersList(ctx, list)
}
