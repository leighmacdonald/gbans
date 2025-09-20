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
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/sosodev/duration"
)

var (
	ErrFetchPerson        = errors.New("failed to fetch/create person")
	ErrInvalidBanOpts     = errors.New("invalid ban options")
	ErrGetBan             = errors.New("failed to load existing ban")
	ErrSaveBan            = errors.New("failed to save ban")
	ErrInvalidBanType     = errors.New("invalid ban type")
	ErrInvalidBanDuration = errors.New("invalid ban duration")
	ErrInvalidBanReason   = errors.New("custom reason cannot be empty")
	ErrInvalidASN         = errors.New("invalid asn, out of range")
	ErrInvalidCIDR        = errors.New("failed to parse CIDR address")
)

// BanOpts defines common ban options that apply to all types to varying degrees
// It should not be instantiated directly, but instead use one of the composites that build
// upon it.
type BanOpts struct {
	TargetID steamid.SteamID `json:"target_id" validate:"required,steamid"`
	SourceID steamid.SteamID `json:"source_id" validate:"required,steamid"`
	// ISO8601
	Duration   *duration.Duration `json:"duration" validate:"required,duration"`
	BanType    ban.Type           `json:"ban_type" validate:"required"`
	Reason     ban.Reason         `json:"reason" validate:"required"`
	ReasonText string             `json:"reason_text" validate:"required"`
	Origin     ban.Origin         `json:"origin" validate:"required"`
	ReportID   int64              `json:"report_id" validate:"gte=1"`
	CIDR       *string            `json:"cidr" validate:"cidrv4"`
	EvadeOk    bool               `json:"evade_ok"`
	Name       string             `json:"name"`
	DemoName   string             `json:"demo_name"`
	DemoTick   int                `json:"demo_tick" validate:"gte=1"`
	Note       string             `json:"note"`
}

func (opts *BanOpts) Validate() error {
	if opts.Duration.ToTimeDuration() <= 0 {
		return fmt.Errorf("%w: %w", ErrInvalidBanOpts, ErrInvalidBanDuration)
	}

	if opts.Reason == ban.Custom && len(opts.ReasonText) < 3 {
		return fmt.Errorf("%w: Custom reason must be at least 3 characters", ErrInvalidBanOpts)
	}

	if opts.CIDR != nil {
		// TODO dont ban too many people, limit to /24 ?
		_, _, errCIDR := net.ParseCIDR(*opts.CIDR)
		if errCIDR != nil {
			return fmt.Errorf("%w: Invalid CIDR", ErrInvalidBanOpts)
		}
	}

	return nil
}

// BanBase provides a common struct shared between all ban types, it should not be used
// directly.
type Ban struct {
	// SteamID is the steamID of the banned person
	TargetID steamid.SteamID `json:"target_id"`
	SourceID steamid.SteamID `json:"source_id"`
	BanID    int64           `json:"ban_id"`
	ReportID int64           `json:"report_id"`
	LastIP   *string         `json:"last_ip"`
	EvadeOk  bool            `json:"evade_ok"`

	// Reason defines the overall ban classification
	BanType ban.Type `json:"ban_type"`
	// Reason defines the overall ban classification
	Reason ban.Reason `json:"reason"`
	// ReasonText is returned to the client when kicked trying to join the server
	ReasonText      string `json:"reason_text"`
	UnbanReasonText string `json:"unban_reason_text"`
	// Note is a supplementary note added by admins that is hidden from normal view
	Note        string      `json:"note"`
	Origin      ban.Origin  `json:"origin"`
	CIDR        *string     `json:"cidr"`
	Name        string      `json:"name"`
	AppealState AppealState `json:"appeal_state"`

	// Deleted is used for soft-deletes
	Deleted   bool `json:"deleted"`
	IsEnabled bool `json:"is_enabled"`
	// ValidUntil is when the ban will be no longer valid. 0 denotes forever
	ValidUntil time.Time `json:"valid_until" `
	CreatedOn  time.Time `json:"created_on"`
	UpdatedOn  time.Time `json:"updated_on"`
}

func (b Ban) IsGroup() bool {
	return b.TargetID.Int64() >= int64(steamid.BaseGID)
}

func (b Ban) Path() string {
	return fmt.Sprintf("/ban/%d", b.BanID)
}

func (b Ban) String() string {
	return fmt.Sprintf("SID: %d Origin: %s Reason: %s Type: %v", b.TargetID.Int64(), b.Origin, b.ReasonText, b.BanType)
}

type BansQueryFilter struct {
	domain.TargetIDField
	Deleted bool `json:"deleted"`
}

type NewBanMessage struct {
	Message string `json:"message"`
}

type RequestUnban struct {
	UnbanReasonText string `json:"unban_reason_text"`
}

type QueryOpts struct {
	SourceID steamid.SteamID
	// TargetID can represent a SteamID or a group ID. They both use steamID formats, just in a different numberspace
	TargetID      steamid.SteamID
	GroupsOnly    bool
	BanID         int64
	Deleted       bool
	EvadeOk       bool
	Personaname   string
	CIDR          string
	IncludeGroups bool
	LatestOnly    bool
}

type Stats struct {
	BansTotal     int `json:"bans_total"`
	BansDay       int `json:"bans_day"`
	BansWeek      int `json:"bans_week"`
	BansMonth     int `json:"bans_month"`
	Bans3Month    int `json:"bans3_month"`
	Bans6Month    int `json:"bans6_month"`
	BansYear      int `json:"bans_year"`
	BansCIDRTotal int `json:"bans_cidr_total"`
	AppealsOpen   int `json:"appeals_open"`
	AppealsClosed int `json:"appeals_closed"`
	FilteredWords int `json:"filtered_words"`
	ServersAlive  int `json:"servers_alive"`
	ServersTotal  int `json:"servers_total"`
}

type Bans struct {
	banRepo BanRepository
	persons *person.Persons
	config  *config.Configuration
	state   *servers.State
	reports Reports
	tfAPI   *thirdparty.TFAPI
}

func NewBans(repository BanRepository, person *person.Persons,
	config *config.Configuration, reports Reports, state *servers.State,
	tfAPI *thirdparty.TFAPI,
) Bans {
	return Bans{
		banRepo: repository,
		persons: person,
		config:  config,
		reports: reports,
		state:   state,
		tfAPI:   tfAPI,
	}
}

func (s Bans) UpdateCache(ctx context.Context) error {
	bans, errBans := s.banRepo.Query(ctx, QueryOpts{GroupsOnly: true})
	if errBans != nil {
		return errBans
	}

	if err := s.banRepo.TruncateCache(ctx); err != nil {
		return err
	}

	for idx, ban := range bans {
		if ban.Deleted || ban.ValidUntil.Before(time.Now()) {
			continue
		}

		if idx > 0 {
			// Not sure what the rate limit is, but be generous for groups.
			time.Sleep(time.Second * 5)
		}

		groupInfo, err := s.tfAPI.SteamGroup(ctx, ban.TargetID)
		if err != nil {
			return err
		}

		var list []int64
		for _, member := range groupInfo.Members {
			sid := steamid.New(member.SteamId)
			list = append(list, sid.Int64())
		}

		if err := s.banRepo.InsertCache(ctx, ban.TargetID, list); err != nil {
			return err
		}
	}

	return nil
}

func (s Bans) Query(ctx context.Context, opts QueryOpts) ([]Ban, error) {
	return s.banRepo.Query(ctx, opts)
}

func (s Bans) QueryOne(ctx context.Context, opts QueryOpts) (Ban, error) {
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

func (s Bans) Stats(ctx context.Context, stats *Stats) error {
	return s.banRepo.Stats(ctx, stats)
}

func (s Bans) Save(ctx context.Context, ban *Ban) error {
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

// Create will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of Config.DefaultExpiration() will be used.
func (s Bans) Create(ctx context.Context, opts BanOpts) (Ban, error) {
	if errValidate := opts.Validate(); errValidate != nil {
		return Ban{}, errValidate
	}

	newBan := Ban{
		// TODO set last ip
		// LastIP:     opts.LastIP,
		EvadeOk:    opts.EvadeOk,
		BanType:    opts.BanType,
		Reason:     opts.Reason,
		TargetID:   opts.TargetID,
		SourceID:   opts.SourceID,
		ReportID:   opts.ReportID,
		CIDR:       opts.CIDR,
		ReasonText: opts.ReasonText,
		Note:       opts.Note,
		Origin:     opts.Origin,
		Name:       opts.Name,
	}

	if opts.Duration.ToTimeDuration() > 0 {
		newBan.ValidUntil = time.Now().Add(opts.Duration.ToTimeDuration())
	}

	author, errAuthor := s.persons.GetOrCreatePersonBySteamID(ctx, nil, opts.SourceID)
	if errAuthor != nil {
		return newBan, errAuthor
	}

	_, err := s.persons.GetOrCreatePersonBySteamID(ctx, nil, opts.TargetID)
	if err != nil {
		return newBan, errors.Join(err, ErrFetchPerson)
	}

	// existing, errGetExistingBan := s.QueryOne(ctx, QueryOpts{TargetID: opts.TargetID, EvadeOk: true})
	// if errGetExistingBan != nil && !errors.Is(errGetExistingBan, database.ErrNoResult) {
	// 	return newBan, errors.Join(errGetExistingBan, ErrGetBan)
	// }

	// if existing.BanID > 0 {
	// 	return newBan, database.ErrDuplicate // TODO better error
	// }

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
	case ban.Banned:
		if errKick := s.state.Kick(ctx, newBan.TargetID, newBan.Reason.String()); errKick != nil && !errors.Is(errKick, domain.ErrPlayerNotFound) {
			slog.Error("Failed to kick player", log.ErrAttr(errKick),
				slog.Int64("sid64", newBan.TargetID.Int64()))
		} else {
			// s.notifications.Enqueue(ctx, notification.NewDiscordNotification(domain.ChannelKickLog, message.KickPlayerEmbed(target)))
		}
	case ban.NoComm:
		if errSilence := s.state.Silence(ctx, newBan.TargetID, newBan.Reason.String()); errSilence != nil && !errors.Is(errSilence, domain.ErrPlayerNotFound) {
			slog.Error("Failed to silence player", log.ErrAttr(errSilence),
				slog.Int64("sid64", newBan.TargetID.Int64()))
		} else {
			// s.notifications.Enqueue(ctx, notification.NewDiscordNotification(domain.ChannelKickLog, message.MuteMessage(bannedPerson)))
		}
	}

	return s.QueryOne(ctx, QueryOpts{BanID: newBan.BanID, EvadeOk: true})
}

// Unban will set the Current ban to now, making it expired.
// Returns true, nil if the ban exists, and was successfully banned.
// Returns false, nil if the ban does not exist.
func (s Bans) Unban(ctx context.Context, targetSID steamid.SteamID, reason string, author domain.PersonInfo) (bool, error) {
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

func (s Bans) Delete(ctx context.Context, ban *Ban, hardDelete bool) error {
	return s.banRepo.Delete(ctx, ban, hardDelete)
}

func (s Bans) Expired(ctx context.Context) ([]Ban, error) {
	return s.banRepo.ExpiredBans(ctx)
}

func (s Bans) GetOlderThan(ctx context.Context, filter query.Filter, since time.Time) ([]Ban, error) {
	return s.banRepo.GetOlderThan(ctx, filter, since)
}

// CheckEvadeStatus checks if the address matches an existing user who is currently banned already. This
// function will always fail-open and allow players in if an error occurs.
func (s Bans) CheckEvadeStatus(ctx context.Context, steamID steamid.SteamID, address netip.Addr) (bool, error) {
	existing, errMatch := s.QueryOne(ctx, QueryOpts{CIDR: address.String()})
	if errMatch != nil {
		if errors.Is(errMatch, database.ErrNoResult) {
			return false, nil
		}

		return false, errMatch
	}

	if existing.BanType == ban.NoComm {
		// Currently we do not ban for mute evasion.
		// TODO make this configurable
		return false, errMatch
	}

	dur, errDuration := duration.Parse("P10Y")
	if errDuration != nil {
		return false, errDuration
	}

	existing.Note += " Previous expiry: " + existing.ValidUntil.Format(time.DateTime)
	existing.ValidUntil = time.Now().Add(dur.ToTimeDuration())

	if errSave := s.Save(ctx, &existing); errSave != nil {
		slog.Error("Could not update previous ban.", log.ErrAttr(errSave))

		return false, errSave
	}

	config := s.config.Config()
	owner := steamid.New(config.Owner)

	req := BanOpts{
		SourceID: owner,
		TargetID: steamID,
		Origin:   ban.System,
		Duration: dur,
		BanType:  ban.Banned,
		Reason:   ban.Evading,
		Note:     fmt.Sprintf("Connecting from same IP as banned player.\n\nEvasion of: [#%d](%s)", existing.BanID, config.ExtURL(existing)),
	}

	_, errSave := s.Create(ctx, req)
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

func (s Bans) UpdateGroupCache(ctx context.Context) error {
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

func (s Bans) GetMembersList(ctx context.Context, parentID int64, list *MembersList) error {
	return s.banRepo.GetMembersList(ctx, parentID, list)
}

func (s Bans) SaveMembersList(ctx context.Context, list *MembersList) error {
	return s.banRepo.SaveMembersList(ctx, list)
}
