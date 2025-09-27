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

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/datetime"
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

const Permanent = "Permanent"

// Opts defines common ban options that apply to all types to varying degrees
// It should not be instantiated directly, but instead use one of the composites that build
// upon it.
type Opts struct {
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

func (opts *Opts) Validate() error {
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
	AppealState AppealState `json:"appeal_state"`
	// Name is the name at time of banning.
	Name string `json:"name"`

	// Deleted is used for soft-deletes
	Deleted   bool `json:"deleted"`
	IsEnabled bool `json:"is_enabled"`
	// ValidUntil is when the ban will be no longer valid. 0 denotes forever
	ValidUntil time.Time `json:"valid_until" `
	CreatedOn  time.Time `json:"created_on"`
	UpdatedOn  time.Time `json:"updated_on"`

	SourcePersonaname string `json:"source_personaname"`
	SourceAvatarhash  string `json:"source_avatarhash"`
	TargetPersonaname string `json:"target_personaname"`
	TargetAvatarhash  string `json:"target_avatarhash"`
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
	Reasons       []ban.Reason
	Personaname   string
	CIDR          string
	CIDROnly      bool
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
	repo    Repository
	persons *person.Persons
	config  *config.Configuration
	state   *servers.State
	reports Reports
	tfAPI   *thirdparty.TFAPI
	notif   notification.Notifications
}

func NewBans(repository Repository, person *person.Persons,
	config *config.Configuration, reports Reports, state *servers.State,
	tfAPI *thirdparty.TFAPI, notif notification.Notifications,
) Bans {
	return Bans{
		repo:    repository,
		persons: person,
		config:  config,
		reports: reports,
		state:   state,
		tfAPI:   tfAPI,
		notif:   notif,
	}
}

func (s Bans) UpdateCache(ctx context.Context) error {
	bans, errBans := s.repo.Query(ctx, QueryOpts{GroupsOnly: true})
	if errBans != nil {
		return errBans
	}

	if err := s.repo.TruncateCache(ctx); err != nil {
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

		if err := s.repo.InsertCache(ctx, ban.TargetID, list); err != nil {
			return err
		}
	}

	return nil
}

func (s Bans) Query(ctx context.Context, opts QueryOpts) ([]Ban, error) {
	return s.repo.Query(ctx, opts)
}

func (s Bans) QueryOne(ctx context.Context, opts QueryOpts) (Ban, error) {
	// TODO FIXME
	results, errResults := s.repo.Query(ctx, opts)
	if errResults != nil {
		return Ban{}, errResults
	}

	if len(results) == 0 {
		return Ban{}, database.ErrNoResult
	}

	return results[0], nil
}

func (s Bans) Stats(ctx context.Context, stats *Stats) error {
	return s.repo.Stats(ctx, stats)
}

func (s Bans) Save(ctx context.Context, ban *Ban) error {
	oldState := Open
	if ban.BanID > 0 {
		existing, errExisting := s.QueryOne(ctx, QueryOpts{
			BanID:      ban.BanID,
			EvadeOk:    true,
			Deleted:    true,
			LatestOnly: true,
		})
		if errExisting != nil {
			slog.Error("Failed to get existing ban", log.ErrAttr(errExisting))

			return errExisting
		}

		oldState = existing.AppealState
	}

	if err := s.repo.Save(ctx, ban); err != nil {
		return err
	}

	if oldState != ban.AppealState {
		s.notif.Send <- notification.NewSiteGroup(
			[]permission.Privilege{permission.Moderator, permission.Admin},
			notification.Info,
			fmt.Sprintf("Ban appeal state changed: %s -> %s", oldState, ban.AppealState),
			ban.Path())

		s.notif.Send <- notification.NewSiteUser(
			[]steamid.SteamID{ban.TargetID},
			notification.Info,
			fmt.Sprintf("Your mute/ban appeal status has changed: %s -> %s", oldState, ban.AppealState),
			ban.Path())
	}

	return nil
}

// Create will ban the steam id from all servers. Players are immediately kicked from servers
// once executed. If duration is 0, the value of Config.DefaultExpiration() will be used.
func (s Bans) Create(ctx context.Context, opts Opts) (Ban, error) {
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

	target, err := s.persons.GetOrCreatePersonBySteamID(ctx, nil, opts.TargetID)
	if err != nil {
		return newBan, errors.Join(err, ErrFetchPerson)
	}

	if errSave := s.repo.Save(ctx, &newBan); errSave != nil {
		return newBan, errors.Join(errSave, ErrSaveBan)
	}

	expIn := Permanent
	expAt := Permanent

	if newBan.ValidUntil.Year()-time.Now().Year() < 5 {
		expIn = datetime.FmtDuration(newBan.ValidUntil)
		expAt = datetime.FmtTimeShort(newBan.ValidUntil)
	}

	s.notif.Send <- notification.NewDiscord(s.config.Config().Discord.BanLogChannelID,
		CreateResponse(newBan))

	s.notif.Send <- notification.NewSiteUserWithAuthor(
		[]permission.Privilege{permission.Moderator, permission.Admin},
		notification.Info,
		fmt.Sprintf("User banned (steam): %s Duration: %s Author: %s",
			newBan.Name, expIn, author.GetName()),
		newBan.Path(),
		author,
	)

	s.notif.Send <- notification.NewSiteUser(
		[]steamid.SteamID{newBan.TargetID},
		notification.Warn,
		fmt.Sprintf("You have been %s, Reason: %s, Duration: %s, Ends: %s", newBan.BanType, newBan.Reason.String(), expIn, expAt),
		newBan.Path(),
	)

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
			s.notif.Send <- notification.NewDiscord(s.config.Config().Discord.KickLogChannelID, KickPlayerEmbed(target))
		}
	case ban.NoComm:
		if errSilence := s.state.Silence(ctx, newBan.TargetID, newBan.Reason.String()); errSilence != nil && !errors.Is(errSilence, domain.ErrPlayerNotFound) {
			slog.Error("Failed to silence player", log.ErrAttr(errSilence),
				slog.Int64("sid64", newBan.TargetID.Int64()))
		} else {
			s.notif.Send <- notification.NewDiscord(s.config.Config().Discord.KickLogChannelID, MuteMessage(target.GetSteamID()))
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

	if errSave := s.repo.Save(ctx, &playerBan); errSave != nil {
		return false, errors.Join(errSave, ErrSaveBan)
	}

	person, err := s.persons.GetPersonBySteamID(ctx, nil, targetSID)
	if err != nil {
		return false, errors.Join(err, ErrFetchPerson)
	}

	s.notif.Send <- notification.NewDiscord(s.config.Config().Discord.BanLogChannelID, UnbanMessage(s.config.ExtURL(person), person))

	s.notif.Send <- notification.NewSiteGroupNotificationWithAuthor(
		[]permission.Privilege{permission.Moderator, permission.Admin},
		notification.Info,
		fmt.Sprintf("A user has been unbanned: %s, Reason: %s", person.GetName(), reason),
		person.Path(),
		author,
	)

	s.notif.Send <- notification.NewSiteUser(
		[]steamid.SteamID{person.SteamID},
		notification.Info,
		"You have been unmuted/unbanned",
		person.Path(),
	)

	return true, nil
}

func (s Bans) Delete(ctx context.Context, ban *Ban, hardDelete bool) error {
	return s.repo.Delete(ctx, ban, hardDelete)
}

func (s Bans) Expired(ctx context.Context) ([]Ban, error) {
	return s.repo.ExpiredBans(ctx)
}

func (s Bans) GetOlderThan(ctx context.Context, filter query.Filter, since time.Time) ([]Ban, error) {
	return s.repo.GetOlderThan(ctx, filter, since)
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
		return false, errors.Join(errDuration, ban.ErrDuration)
	}

	existing.Note += " Previous expiry: " + existing.ValidUntil.Format(time.DateTime)
	existing.ValidUntil = time.Now().Add(dur.ToTimeDuration())

	if errSave := s.Save(ctx, &existing); errSave != nil {
		slog.Error("Could not update previous ban.", log.ErrAttr(errSave))

		return false, errSave
	}

	config := s.config.Config()
	owner := steamid.New(config.Owner)

	req := Opts{
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

	if err := s.repo.TruncateCache(ctx); err != nil {
		return err
	}

	client := httphelper.NewClient()

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

		if err := s.repo.InsertCache(ctx, groupID, list.Members.SteamID64); err != nil {
			return err
		}
	}

	return nil
}

func (s Bans) GetMembersList(ctx context.Context, parentID int64, list *MembersList) error {
	return s.repo.GetMembersList(ctx, parentID, list)
}

func (s Bans) SaveMembersList(ctx context.Context, list *MembersList) error {
	return s.repo.SaveMembersList(ctx, list)
}
