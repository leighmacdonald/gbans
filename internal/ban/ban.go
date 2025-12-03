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
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/datetime"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/sosodev/duration"
)

var (
	ErrFetchPerson        = errors.New("failed to fetch/create person")
	ErrInvalidBanOpts     = errors.New("invalid ban options")
	ErrGetBan             = errors.New("failed to load existing ban")
	ErrSaveBan            = errors.New("failed to save ban")
	ErrInvalidBanDuration = errors.New("invalid ban duration")
	ErrUnbanFailed        = errors.New("failed to perform unban")
	ErrPersonSource       = errors.New("failed to load source person")
	ErrPersonTarget       = errors.New("failed to load target person")
	ErrDuplicateBan       = errors.New("duplicate ban")
	ErrBanDoesNotExist    = errors.New("ban does not exist")
)

const maxCIDRHosts = 256 * 256

type Config struct {
	BDEnabled      bool   `json:"bd_enabled"`
	ValveEnabled   bool   `json:"valve_enabled"`
	AuthorizedKeys string `json:"authorized_keys"`
}

// Origin defines the origin of the ban or action.
type Origin int

const (
	// System is an automatic ban triggered by the service.
	System Origin = iota
	// Bot is a ban using the discord bot interface.
	Bot
	// Web is a ban using the web-ui.
	Web
	// InGame is a ban using the sourcemod plugin.
	InGame
)

func (s Origin) String() string {
	switch s {
	case System:
		return "System"
	case Bot:
		return "Bot"
	case Web:
		return "Web"
	case InGame:
		return "In-Game"
	default:
		return "Unknown"
	}
}

const Permanent = "Permanent"

// Opts defines common ban options that apply to all types to varying degrees
// It should not be instantiated directly, but instead use one of the composites that build
// upon it.
type Opts struct {
	TargetID steamid.SteamID `json:"target_id" binding:"required,steamid"`
	SourceID steamid.SteamID `json:"source_id" binding:"required,steamid"`
	// ISO8601
	Duration   *duration.Duration `json:"duration" binding:"required,duration"`
	BanType    bantype.Type       `json:"ban_type" binding:"required"`
	Reason     reason.Reason      `json:"reason" binding:"required"`
	ReasonText string             `json:"reason_text"`
	Origin     Origin             `json:"origin" binding:"oneof=0 1 2 3"`
	ReportID   int64              `json:"report_id" binding:"omitempty,gt=0"`
	CIDR       *string            `json:"cidr"  binding:"omitnil,cidrv4"`
	EvadeOk    bool               `json:"evade_ok"`
	Name       string             `json:"name" binding:"max=36"`
	DemoName   string             `json:"demo_name" binding:"omitempty,max=256"`
	DemoTick   int                `json:"demo_tick" binding:"omitempty"`
	Note       string             `json:"note" binding:"omitempty,max=100000"`
}

func (opts *Opts) Validate() error {
	if opts.Duration.ToTimeDuration() <= 0 {
		return fmt.Errorf("%w: %w", ErrInvalidBanOpts, ErrInvalidBanDuration)
	}

	if opts.Reason == reason.Custom && len(opts.ReasonText) < 3 {
		return fmt.Errorf("%w: Custom reason must be at least 3 characters", ErrInvalidBanOpts)
	}

	if opts.CIDR != nil {
		_, ipnet, errCIDR := net.ParseCIDR(*opts.CIDR)
		if errCIDR != nil {
			return fmt.Errorf("%w: Invalid CIDR", ErrInvalidBanOpts)
		}

		if count := AddressCount(ipnet); count > maxCIDRHosts {
			return fmt.Errorf("%w: Invalid CIDR, too many hosts: %d", ErrInvalidBanOpts, count)
		}
	}

	return nil
}

// AddressCount returns the number of distinct host addresses within the given
// CIDR range.
func AddressCount(network *net.IPNet) uint64 {
	prefixLen, bits := network.Mask.Size()

	return 1 << (uint64(bits) - uint64(prefixLen)) //nolint:gosec
}

type Ban struct {
	// SteamID is the steamID of the banned person
	TargetID steamid.SteamID `json:"target_id"`
	SourceID steamid.SteamID `json:"source_id"`
	BanID    int64           `json:"ban_id"`
	ReportID int64           `json:"report_id"`
	LastIP   *string         `json:"last_ip"`
	EvadeOk  bool            `json:"evade_ok"`

	// Reason defines the overall ban classification
	BanType bantype.Type `json:"ban_type"`
	// Reason defines the overall ban classification
	Reason reason.Reason `json:"reason"`
	// ReasonText is returned to the client when kicked trying to join the server
	ReasonText      string `json:"reason_text"`
	UnbanReasonText string `json:"unban_reason_text"`
	// Note is a supplementary note added by admins that is hidden from normal view
	Note        string      `json:"note"`
	Origin      Origin      `json:"origin"`
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

func (b Ban) Expired() bool {
	return time.Now().After(b.ValidUntil)
}

type BansQueryFilter struct {
	httphelper.TargetIDField

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
	Reasons       []reason.Reason
	Personaname   string
	CIDR          string
	CIDROnly      bool
	IncludeGroups bool
	LatestOnly    bool
	ValidUntil    time.Time
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
	repo          Repository
	persons       person.Provider
	reports       Reports
	notif         notification.Notifier
	logChannelID  string
	kickChannelID string
	owner         steamid.SteamID
	servers       *servers.Servers
}

func New(repository Repository, person person.Provider, logChannelID string, kickChannelID string,
	owner steamid.SteamID, reports Reports, notif notification.Notifier, servers *servers.Servers,
) Bans {
	return Bans{
		repo:          repository,
		persons:       person,
		reports:       reports,
		notif:         notif,
		logChannelID:  logChannelID,
		owner:         owner,
		kickChannelID: kickChannelID,
		servers:       servers,
	}
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
			slog.Error("Failed to get existing ban", slog.String("error", errExisting.Error()))

			return errExisting
		}

		oldState = existing.AppealState
	}

	if err := s.repo.Save(ctx, ban); err != nil {
		return err
	}

	if oldState != ban.AppealState {
		s.notif.Send(notification.NewSiteGroup(
			[]permission.Privilege{permission.Moderator, permission.Admin},
			notification.Info,
			fmt.Sprintf("Ban appeal state changed: %s -> %s", oldState, ban.AppealState),
			link.Path(ban)))

		s.notif.Send(notification.NewSiteUser(
			[]steamid.SteamID{ban.TargetID},
			notification.Info,
			fmt.Sprintf("Your mute/ban appeal status has changed: %s -> %s", oldState, ban.AppealState),
			link.Path(ban)))
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

	author, errAuthor := s.persons.GetOrCreatePersonBySteamID(ctx, opts.SourceID)
	if errAuthor != nil {
		return newBan, errAuthor
	}

	target, errTarget := s.persons.GetOrCreatePersonBySteamID(ctx, opts.TargetID)
	if errTarget != nil {
		return newBan, errTarget
	}

	if errSave := s.repo.Save(ctx, &newBan); errSave != nil {
		return newBan, errors.Join(errSave, ErrSaveBan)
	}

	// Close the report if the ban was attached to one
	if newBan.ReportID > 0 {
		if _, errSaveReport := s.reports.SetReportStatus(ctx, newBan.ReportID, author, ClosedWithAction); errSaveReport != nil {
			return newBan, errors.Join(errSaveReport, ErrReportStateUpdate)
		}
	}

	s.sendBanNotification(ctx, newBan, author, target)

	return s.QueryOne(ctx, QueryOpts{BanID: newBan.BanID, EvadeOk: true})
}

func (s Bans) sendBanNotification(ctx context.Context, newBan Ban, author person.Core, target person.Core) {
	expIn := Permanent
	expAt := Permanent

	if newBan.ValidUntil.Year()-time.Now().Year() < 5 {
		expIn = datetime.FmtDuration(newBan.ValidUntil)
		expAt = datetime.FmtTimeShort(newBan.ValidUntil)
	}

	go s.notif.Send(notification.NewDiscord(s.logChannelID, createBanResponse(newBan, author, target)))
	go s.notif.Send(notification.NewSiteUserWithAuthor(
		[]permission.Privilege{permission.Moderator, permission.Admin},
		notification.Info,
		fmt.Sprintf("User banned (steam): %s Duration: %s Author: %s",
			newBan.Name, expIn, author.GetName()),
		link.Path(newBan),
		author,
	))
	go s.notif.Send(notification.NewSiteUser(
		[]steamid.SteamID{newBan.TargetID},
		notification.Warn,
		fmt.Sprintf("You have been %s, Reason: %s, Duration: %s, Ends: %s", newBan.BanType, newBan.Reason.String(), expIn, expAt),
		link.Path(newBan),
	))

	if s.servers == nil {
		return
	}
	result, found := s.servers.FindPlayer(servers.FindOpts{SteamID: newBan.TargetID})
	if !found {
		return
	}

	switch newBan.BanType {
	case bantype.Banned:
		if errKick := result.Server.Kick(ctx, newBan.TargetID, newBan.Reason.String()); errKick != nil && !errors.Is(errKick, servers.ErrPlayerNotFound) {
			slog.Error("Failed to kick player", slog.String("error", errKick.Error()),
				slog.Int64("sid64", newBan.TargetID.Int64()))
		}
		s.notif.Send(notification.NewDiscord(s.kickChannelID, discord.NewMessage(
			discord.Heading("User Kicked [%s]", result.Server.ShortName),
			discord.BodyColouredText(discord.ColourInfo, result.Player.Name))))
	case bantype.NoComm:
		if errSilence := result.Server.Silence(ctx, newBan.TargetID, newBan.Reason.String()); errSilence != nil && !errors.Is(errSilence, servers.ErrPlayerNotFound) {
			slog.Error("Failed to silence player", slog.String("error", errSilence.Error()),
				slog.Int64("sid64", newBan.TargetID.Int64()))
		}
		s.notif.Send(notification.NewDiscord(s.kickChannelID, discord.NewMessage(
			discord.Heading("User Muted [%s]", result.Server.ShortName),
			discord.BodyColouredText(discord.ColourInfo, result.Player.Name))))
	default:
		slog.Error("Unknown ban type")
	}
}

// Unban will set the Current ban to now, making it expired.
// Returns true, nil if the ban exists, and was successfully banned.
// Returns false, nil if the ban does not exist.
func (s Bans) Unban(ctx context.Context, targetSID steamid.SteamID, reason string, author person.Info) (bool, error) {
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

	player, err := s.persons.GetOrCreatePersonBySteamID(ctx, targetSID)
	if err != nil {
		return false, errors.Join(err, ErrFetchPerson)
	}

	s.notif.Send(notification.NewDiscord(s.logChannelID, unbanMessage(player, reason)))
	s.notif.Send(notification.NewSiteGroupNotificationWithAuthor(
		[]permission.Privilege{permission.Moderator, permission.Admin},
		notification.Info,
		fmt.Sprintf("A user has been unbanned: %s, Reason: %s", player.GetName(), reason),
		link.Path(player),
		author,
	))
	s.notif.Send(notification.NewSiteUser(
		[]steamid.SteamID{player.SteamID},
		notification.Info,
		"You have been unmuted/unbanned",
		link.Path(player),
	))

	return true, nil
}

func (s Bans) Delete(ctx context.Context, ban *Ban, hardDelete bool) error {
	return s.repo.Delete(ctx, ban, hardDelete)
}

func (s Bans) Expired(ctx context.Context) ([]Ban, error) {
	return s.repo.Query(ctx, QueryOpts{
		ValidUntil: time.Now(),
	})
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

	if existing.BanType == bantype.NoComm {
		// Currently we do not ban for mute evasion.
		// TODO make this configurable
		return false, errMatch
	}

	dur, errDuration := duration.Parse("P10Y")
	if errDuration != nil {
		return false, errors.Join(errDuration, ErrInvalidBanDuration)
	}

	existing.Note += " Previous expiry: " + existing.ValidUntil.Format(time.DateTime)
	existing.ValidUntil = time.Now().Add(dur.ToTimeDuration())

	if errSave := s.Save(ctx, &existing); errSave != nil {
		slog.Error("Could not update previous ban.", slog.String("error", errSave.Error()))

		return false, errSave
	}

	req := Opts{
		SourceID: s.owner,
		TargetID: steamID,
		Origin:   System,
		Duration: dur,
		BanType:  bantype.Banned,
		Reason:   reason.Evading,
		Note:     fmt.Sprintf("Connecting from same IP as banned player.\n\nEvasion of: [#%d](%s)", existing.BanID, link.Path(existing)),
	}

	_, errSave := s.Create(ctx, req)
	if errSave != nil {
		if errors.Is(errSave, database.ErrDuplicate) {
			// Already banned
			return true, nil
		}

		slog.Error("Could not save evade ban", slog.String("error", errSave.Error()))

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
			return errors.Join(errReq, httphelper.ErrRequestCreate)
		}

		resp, errResp := client.Do(req)
		if errResp != nil {
			return errors.Join(errResp, httphelper.ErrRequestPerform)
		}

		var list SteamGroupInfo

		decoder := xml.NewDecoder(resp.Body)
		if err := decoder.Decode(&list); err != nil {
			_ = resp.Body.Close()

			return errors.Join(err, httphelper.ErrRequestDecode)
		}

		_ = resp.Body.Close()

		groupID := steamid.New(list.GroupID64)
		if !groupID.Valid() {
			return steamid.ErrInvalidSID
		}

		for _, member := range list.Members.SteamID64 {
			steamID := steamid.New(member)
			if !steamID.Valid() {
				continue
			}

			// Statisfy FK
			_, errCreate := s.persons.GetOrCreatePersonBySteamID(ctx, steamID)
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
