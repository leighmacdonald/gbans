package ban

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	banDomain "github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
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
	banRepo BanRepository
	persons person.PersonUsecase
	config  *config.ConfigUsecase
	state   *servers.StateUsecase
	reports ReportUsecase
	tfAPI   *thirdparty.TFAPI
}

func NewBanUsecase(repository BanRepository, person person.PersonUsecase,
	config *config.ConfigUsecase, reports ReportUsecase, state *servers.StateUsecase,
	tfAPI *thirdparty.TFAPI,
) BanUsecase {
	return BanUsecase{
		banRepo: repository,
		persons: person,
		config:  config,
		reports: reports,
		state:   state,
		tfAPI:   tfAPI,
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

func (s *BanUsecase) Query(ctx context.Context, opts QueryOpts) ([]Ban, error) {
	return s.banRepo.Query(ctx, opts)
}

func (s *BanUsecase) QueryOne(ctx context.Context, opts QueryOpts) (Ban, error) {
	// TODO FIXME
	results, errResults := s.banRepo.Query(ctx, opts)
	if errResults != nil {
		return Ban{}, errResults
	}

	if len(results) == 0 {
		return Ban{}, database.ErrNoResult
	}

	return results[0], nil
}

func (s *BanUsecase) Stats(ctx context.Context, stats *Stats) error {
	return s.banRepo.Stats(ctx, stats)
}

func (s *BanUsecase) Save(ctx context.Context, ban *Ban) error {
	// oldState := Open
	// if ban.BanID > 0 {
	// 	existing, errExisting := s.QueryOne(ctx, QueryOpts{
	// 		BanID:      ban.BanID,
	// 		EvadeOk:    true,
	// 		Deleted:    true,
	// 		LatestOnly: true,
	// 	})
	// 	if errExisting != nil {
	// 		slog.Error("Failed to get existing ban", log.ErrAttr(errExisting))

	// 		return errExisting
	// 	}

	// 	oldState = existing.AppealState
	// }

	if err := s.banRepo.Save(ctx, ban); err != nil {
		return err
	}

	// if oldState != ban.AppealState {
	// 	s.notifications.Enqueue(ctx, notification.NewSiteGroupNotification(
	// 		[]permission.Privilege{permission.PModerator, permission.PAdmin},
	// 		notification.SeverityInfo,
	// 		fmt.Sprintf("Ban appeal state changed: %s -> %s", oldState, ban.AppealState),
	// 		ban.Path()))

	// 	s.notifications.Enqueue(ctx, notification.NewSiteUserNotification(
	// 		[]steamid.SteamID{ban.TargetID},
	// 		notification.SeverityInfo,
	// 		fmt.Sprintf("Your mute/ban appeal status has changed: %s -> %s", oldState, ban.AppealState),
	// 		ban.Path()))
	// }

	return nil
}

var ErrBanOptsInvalid = errors.New("invalid ban options")

// Ban will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of Config.DefaultExpiration() will be used.
func (s *BanUsecase) Ban(ctx context.Context, opts BanOpts) (Ban, error) {
	if errValidate := opts.Validate(); errValidate != nil {
		return Ban{}, errValidate
	}

	newBan := Ban{
		IncludeFriends: opts.IncludeFriends,
		LastIP:         opts.LastIP,
		EvadeOk:        opts.EvadeOk,
		BanType:        opts.BanType,
		Reason:         opts.Reason,
	}

	if opts.CIDR != "" {
		// TODO dont ban too many people, limit to /24 ?
		_, _, errCIDR := net.ParseCIDR(opts.CIDR)
		if errCIDR != nil {
			return newBan, fmt.Errorf("%w: Invalid CIDR", ErrBanOptsInvalid)
		}
		newBan.CIDR = opts.CIDR
	} else if opts.ASNum != 0 {

	}

	author, errAuthor := s.persons.GetOrCreatePersonBySteamID(ctx, nil, opts.SourceID)
	if errAuthor != nil {
		return newBan, errAuthor
	}

	_, err := s.persons.GetOrCreatePersonBySteamID(ctx, nil, opts.TargetID)
	if err != nil {
		return newBan, errors.Join(err, ErrFetchPerson)
	}

	existing, errGetExistingBan := s.QueryOne(ctx, QueryOpts{TargetID: opts.TargetID, EvadeOk: true})
	if errGetExistingBan != nil && !errors.Is(errGetExistingBan, database.ErrNoResult) {
		return newBan, errors.Join(errGetExistingBan, ErrGetBan)
	}

	if existing.BanID > 0 {
		return newBan, database.ErrDuplicate // TODO better error
	}

	if errSave := s.banRepo.Save(ctx, &newBan); errSave != nil {
		return newBan, errors.Join(errSave, ErrSaveBan)
	}

	// bannedPerson, errBannedPerson := s.QueryOne(ctx, QueryOpts{BanID: newBan.BanID, EvadeOk: true})
	// if errBannedPerson != nil {
	// 	return newBan, errors.Join(errBannedPerson, ErrSaveBan)
	// }

	// expIn := "Permanent"
	// expAt := "Permanent"

	// if banSteam.ValidUntil.Year()-time.Now().Year() < 5 {
	// 	expIn = datetime.FmtDuration(banSteam.ValidUntil)
	// 	expAt = datetime.FmtTimeShort(banSteam.ValidUntil)
	// }

	// s.notifications.Enqueue(ctx, notification.NewDiscordNotification(discord.ChannelBanLog, discord.BanSteamResponse(bannedPerson)))

	// siteMsg := notification.NewSiteUserNotificationWithAuthor(
	// 	[]permission.Privilege{permission.PModerator, permission.PAdmin},
	// 	notification.SeverityInfo,
	// 	fmt.Sprintf("User banned (steam): %s Duration: %s Author: %s",
	// 		bannedPerson.TargetPersonaname, expIn, bannedPerson.SourcePersonaname),
	// 	banSteam.Path(),
	// 	author.ToUserProfile(),
	// )

	// s.notifications.Enqueue(ctx, siteMsg)

	// s.notifications.Enqueue(ctx, notification.NewSiteUserNotification(
	// 	[]steamid.SteamID{bannedPerson.TargetID},
	// 	notification.SeverityWarn,
	// 	fmt.Sprintf("You have been %s, Reason: %s, Duration: %s, Ends: %s", bannedPerson.BanType, bannedPerson.Reason.String(), expIn, expAt),
	// 	banSteam.Path(),
	// ))

	// Close the report if the ban was attached to one
	if newBan.ReportID > 0 {
		if _, errSaveReport := s.reports.SetReportStatus(ctx, newBan.ReportID, author, ClosedWithAction); errSaveReport != nil {
			return newBan, errors.Join(errSaveReport, ErrReportStateUpdate)
		}
	}

	switch newBan.BanType {
	case banDomain.Banned:
		if errKick := s.state.Kick(ctx, newBan.TargetID, newBan.Reason.String()); errKick != nil && !errors.Is(errKick, domain.ErrPlayerNotFound) {
			slog.Error("Failed to kick player", log.ErrAttr(errKick),
				slog.Int64("sid64", newBan.TargetID.Int64()))
		} else {
			// s.notifications.Enqueue(ctx, notification.NewDiscordNotification(domain.ChannelKickLog, message.KickPlayerEmbed(target)))
		}
	case banDomain.NoComm:
		if errSilence := s.state.Silence(ctx, newBan.TargetID, newBan.Reason.String()); errSilence != nil && !errors.Is(errSilence, domain.ErrPlayerNotFound) {
			slog.Error("Failed to silence player", log.ErrAttr(errSilence),
				slog.Int64("sid64", newBan.TargetID.Int64()))
		} else {
			// s.notifications.Enqueue(ctx, notification.NewDiscordNotification(domain.ChannelKickLog, message.MuteMessage(bannedPerson)))
		}
	default:
		return newBan, ErrInvalidBanType
	}

	return s.QueryOne(ctx, QueryOpts{BanID: newBan.BanID, EvadeOk: true})
}

// Unban will set the Current ban to now, making it expired.
// Returns true, nil if the ban exists, and was successfully banned.
// Returns false, nil if the ban does not exist.
func (s *BanUsecase) Unban(ctx context.Context, targetSID steamid.SteamID, reason string, author domain.PersonInfo) (bool, error) {
	playerBan, errGetBan := s.QueryOne(ctx, QueryOpts{TargetID: targetSID, EvadeOk: true})
	if errGetBan != nil {
		if errors.Is(errGetBan, database.ErrNoResult) {
			return false, nil
		}

		return false, errors.Join(errGetBan, ErrGetBan)
	}

	playerBan.Deleted = true
	playerBan.UnbanReasonText = reason

	if errSave := s.banRepo.Save(ctx, &playerBan); errSave != nil {
		return false, errors.Join(errSave, ErrSaveBan)
	}

	// person, err := s.persons.GetPersonBySteamID(ctx, nil, targetSID)
	// if err != nil {
	// 	return false, errors.Join(err, ErrFetchPerson)
	// }

	// s.notifications.Enqueue(ctx, notification.NewDiscordNotification(domain.ChannelBanLog, message.UnbanMessage(s.config, person)))

	// s.notifications.Enqueue(ctx, notification.NewSiteGroupNotificationWithAuthor(
	// 	[]permission.Privilege{permission.PModerator, permission.PAdmin},
	// 	notification.SeverityInfo,
	// 	fmt.Sprintf("A user has been unbanned: %s, Reason: %s", bannedPerson.TargetPersonaname, reason),
	// 	bannedPerson.Path(),
	// 	author,
	// ))

	// s.notifications.Enqueue(ctx, notification.NewSiteUserNotification(
	// 	[]steamid.SteamID{bannedPerson.TargetID},
	// 	notification.SeverityInfo,
	// 	"You have been unmuted/unbanned",
	// 	bannedPerson.Path(),
	// ))

	return true, nil
}

func (s *BanUsecase) Delete(ctx context.Context, ban *Ban, hardDelete bool) error {
	return s.banRepo.Delete(ctx, ban, hardDelete)
}

func (s *BanUsecase) Expired(ctx context.Context) ([]Ban, error) {
	return s.banRepo.ExpiredBans(ctx)
}

func (s *BanUsecase) GetOlderThan(ctx context.Context, filter domain.QueryFilter, since time.Time) ([]Ban, error) {
	return s.banRepo.GetOlderThan(ctx, filter, since)
}

// CheckEvadeStatus checks if the address matches an existing user who is currently banned already. This
// function will always fail-open and allow players in if an error occurs.
func (s *BanUsecase) CheckEvadeStatus(ctx context.Context, steamID steamid.SteamID, address netip.Addr) (bool, error) {
	existing, errMatch := s.QueryOne(ctx, QueryOpts{CIDR: address.String()})
	if errMatch != nil {
		if errors.Is(errMatch, database.ErrNoResult) {
			return false, nil
		}

		return false, errMatch
	}

	if existing.BanType == banDomain.NoComm {
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

	if errSave := s.Save(ctx, &existing); errSave != nil {
		slog.Error("Could not update previous ban.", log.ErrAttr(errSave))

		return false, errSave
	}

	config := s.config.Config()
	owner := steamid.New(config.Owner)

	req := BanOpts{
		SourceID: owner,
		TargetID: steamID,
		Origin:   banDomain.System,
		Duration: time.Hour * 24 * 365,
		BanType:  banDomain.Banned,
		Reason:   banDomain.Evading,
		Note:     fmt.Sprintf("Connecting from same IP as banned player.\n\nEvasion of: [#%d](%s)", existing.BanID, config.ExtURL(existing)),
	}

	_, errSave := s.Ban(ctx, req)
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

func (s *BanUsecase) UpdateGroupCache(ctx context.Context) error {
	groups, errGroups := s.Query(ctx, QueryOpts{GroupsOnly: true})
	if errGroups != nil {
		return errGroups
	}

	if err := s.banRepo.TruncateCache(ctx); err != nil {
		return err
	}

	client := httphelper.NewHTTPClient()

	for _, group := range groups {
		listURL := fmt.Sprintf("https://steamcommunity.com/gid/%d/memberslistxml/?xml=1", group.TargetID.Int64())

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
