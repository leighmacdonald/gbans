package model

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/avatar"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type SteamMember interface {
	IsMember(steamID steamid.SID64) (int64, bool)
}

type PersonInfo interface {
	GetDiscordID() string
	GetName() string
	GetAvatar() avatar.AvatarLinks
	GetSteamID() steamid.SID64
	Path() string // config.LinkablePath
}

type SteamIDProvider interface {
	SID64(ctx context.Context) (steamid.SID64, error)
}

// StringSID defines a user provided steam id in an unknown format.
type StringSID string

func (t StringSID) SID64(ctx context.Context) (steamid.SID64, error) {
	resolveCtx, cancelResolve := context.WithTimeout(ctx, time.Second*5)
	defer cancelResolve()

	// TODO cache this as it can be a huge hot path
	sid64, errResolveSID := steamid.ResolveSID64(resolveCtx, string(t))
	if errResolveSID != nil {
		return "", errs.ErrInvalidSID
	}

	if !sid64.Valid() {
		return "", errs.ErrInvalidSID
	}

	return sid64, nil
}

type SimplePerson struct {
	Personaname     string    `json:"personaname"`
	Avatarhash      string    `json:"avatarhash"`
	PermissionLevel Privilege `json:"permission_level"`
}

// UserProfile is the model used in the webui representing the logged-in user.
type UserProfile struct {
	SteamID         steamid.SID64 `json:"steam_id"`
	CreatedOn       time.Time     `json:"created_on"`
	UpdatedOn       time.Time     `json:"updated_on"`
	PermissionLevel Privilege     `json:"permission_level"`
	DiscordID       string        `json:"discord_id"`
	Name            string        `json:"name"`
	Avatarhash      string        `json:"avatarhash"`
	BanID           int64         `json:"ban_id"`
	Muted           bool          `json:"muted"`
}

func (p UserProfile) GetDiscordID() string {
	return p.DiscordID
}

func (p UserProfile) GetName() string {
	return p.Name
}

func (p UserProfile) GetAvatar() avatar.AvatarLinks {
	return avatar.NewAvatarLinks(p.Avatarhash)
}

func (p UserProfile) GetSteamID() steamid.SID64 {
	return p.SteamID
}

func (p UserProfile) Path() string {
	return fmt.Sprintf("/profile/%d", p.SteamID.Int64())
}

// NewUserProfile allocates a new default person instance.
func NewUserProfile(sid64 steamid.SID64) UserProfile {
	t0 := time.Now()

	return UserProfile{
		SteamID:         sid64,
		CreatedOn:       t0,
		UpdatedOn:       t0,
		PermissionLevel: PUser,
		Name:            "Guest",
	}
}

type Person struct {
	// TODO merge use of steamid & steam_id
	SteamID          steamid.SID64         `db:"steam_id" json:"steam_id"`
	CreatedOn        time.Time             `json:"created_on"`
	UpdatedOn        time.Time             `json:"updated_on"`
	PermissionLevel  Privilege             `json:"permission_level"`
	Muted            bool                  `json:"muted"`
	IsNew            bool                  `json:"-"`
	DiscordID        string                `json:"discord_id"`
	IPAddr           net.IP                `json:"-"` // TODO Allow json for admins endpoints
	CommunityBanned  bool                  `json:"community_banned"`
	VACBans          int                   `json:"vac_bans"`
	GameBans         int                   `json:"game_bans"`
	EconomyBan       steamweb.EconBanState `json:"economy_ban"`
	DaysSinceLastBan int                   `json:"days_since_last_ban"`
	UpdatedOnSteam   time.Time             `json:"updated_on_steam"`
	*steamweb.PlayerSummary
}

func (p Person) Expired() bool {
	return p.IsNew || time.Since(p.UpdatedOnSteam) > time.Hour*24*30
}

func (p Person) GetDiscordID() string {
	return p.DiscordID
}

func (p Person) GetName() string {
	return p.PersonaName
}

func (p Person) GetAvatar() avatar.AvatarLinks {
	return avatar.NewAvatarLinks(p.AvatarHash)
}

func (p Person) GetSteamID() steamid.SID64 {
	return p.SteamID
}

func (p Person) Path() string {
	return fmt.Sprintf("/profile/%d", p.SteamID.Int64())
}

// LoggedIn checks for a valid steamID.
func (p Person) LoggedIn() bool {
	return p.SteamID.Valid() && p.SteamID.Int64() > 0
}

// AsTarget checks for a valid steamID.
func (p Person) AsTarget() StringSID {
	return StringSID(p.SteamID.String())
}

// NewPerson allocates a new default person instance.
func NewPerson(sid64 steamid.SID64) Person {
	curTime := time.Now()

	return Person{
		SteamID:          sid64,
		CreatedOn:        curTime,
		UpdatedOn:        curTime,
		PermissionLevel:  PUser,
		Muted:            false,
		IsNew:            true,
		DiscordID:        "",
		IPAddr:           nil,
		CommunityBanned:  false,
		VACBans:          0,
		GameBans:         0,
		EconomyBan:       "none",
		DaysSinceLastBan: 0,
		UpdatedOnSteam:   time.Unix(0, 0),
		PlayerSummary: &steamweb.PlayerSummary{
			SteamID: sid64,
		},
	}
}

type People []Person

func (p People) ToSteamIDCollection() steamid.Collection {
	var collection steamid.Collection

	for _, person := range p {
		collection = append(collection, person.SteamID)
	}

	return collection
}

func (p People) AsMap() map[steamid.SID64]Person {
	m := map[steamid.SID64]Person{}
	for _, person := range p {
		m[person.SteamID] = person
	}

	return m
}

type UserNotification struct {
	PersonNotificationID int64                `json:"person_notification_id"`
	SteamID              steamid.SID64        `json:"steam_id"`
	Read                 bool                 `json:"read"`
	Deleted              bool                 `json:"deleted"`
	Severity             NotificationSeverity `json:"severity"`
	Message              string               `json:"message"`
	Link                 string               `json:"link"`
	Count                int                  `json:"count"`
	CreatedOn            time.Time            `json:"created_on"`
}

// PersonIPRecord holds a composite result of the more relevant ip2location results.
type PersonIPRecord struct {
	IPAddr      net.IP
	CreatedOn   time.Time
	CityName    string
	CountryName string
	CountryCode string
	ASName      string
	ASNum       int
	ISP         string
	UsageType   string
	Threat      string
	DomainUsed  string
}

type PersonAuth struct {
	PersonAuthID int64         `json:"person_auth_id"`
	SteamID      steamid.SID64 `json:"steam_id"`
	IPAddr       net.IP        `json:"ip_addr"`
	RefreshToken string        `json:"refresh_token"`
	CreatedOn    time.Time     `json:"created_on"`
}

func NewPersonAuth(sid64 steamid.SID64, addr net.IP, fingerPrint string) PersonAuth {
	return PersonAuth{
		PersonAuthID: 0,
		SteamID:      sid64,
		IPAddr:       addr,
		RefreshToken: fingerPrint,
		CreatedOn:    time.Now(),
	}
}

type PersonConnection struct {
	PersonConnectionID int64         `json:"person_connection_id"`
	IPAddr             net.IP        `json:"ip_addr"`
	SteamID            steamid.SID64 `json:"steam_id"`
	PersonaName        string        `json:"persona_name"`
	ServerID           int           `json:"server_id"`
	CreatedOn          time.Time     `json:"created_on"`
	ServerNameShort    string        `json:"server_name_short"`
	ServerName         string        `json:"server_name"`
}

type PersonConnections []PersonConnection

type PersonMessage struct {
	PersonMessageID int64         `json:"person_message_id"`
	MatchID         uuid.UUID     `json:"match_id"`
	SteamID         steamid.SID64 `json:"steam_id"`
	AvatarHash      string        `json:"avatar_hash"`
	PersonaName     string        `json:"persona_name"`
	ServerName      string        `json:"server_name"`
	ServerID        int           `json:"server_id"`
	Body            string        `json:"body"`
	Team            bool          `json:"team"`
	CreatedOn       time.Time     `json:"created_on"`
	Flagged         bool          `json:"flagged"`
}

type PersonMessages []PersonMessage

type QueryChatHistoryResult struct {
	PersonMessage
	AutoFilterFlagged int64  `json:"auto_filter_flagged"`
	Pattern           string `json:"pattern"`
}

type PersonSettings struct {
	PersonSettingsID     int64         `json:"person_settings_id"`
	SteamID              steamid.SID64 `json:"steam_id"`
	ForumSignature       string        `json:"forum_signature"`
	ForumProfileMessages bool          `json:"forum_profile_messages"`
	StatsHidden          bool          `json:"stats_hidden"`
	CreatedOn            time.Time     `json:"created_on"`
	UpdatedOn            time.Time     `json:"updated_on"`
}

type UserWarning struct {
	WarnReason    Reason    `json:"warn_reason"`
	Message       string    `json:"message"`
	Matched       string    `json:"matched"`
	MatchedFilter *Filter   `json:"matched_filter"`
	CreatedOn     time.Time `json:"created_on"`
	Personaname   string    `json:"personaname"`
	Avatar        string    `json:"avatar"`
	ServerName    string    `json:"server_name"`
	ServerID      int       `json:"server_id"`
	SteamID       string    `json:"steam_id"`
	CurrentTotal  int       `json:"current_total"`
}

type NewUserWarning struct {
	UserMessage PersonMessage
	UserWarning
}

type Warnings interface {
	State() map[string][]UserWarning
}
