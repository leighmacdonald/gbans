package domain

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type SteamMember interface {
	IsMember(steamID steamid.SteamID) (int64, bool)
}

type PersonUsecase interface {
	DropPerson(ctx context.Context, transaction pgx.Tx, steamID steamid.SteamID) error
	SavePerson(ctx context.Context, transaction pgx.Tx, person *Person) error
	QueryProfile(ctx context.Context, query string) (ProfileResponse, error)
	GetPersonBySteamID(ctx context.Context, transaction pgx.Tx, sid64 steamid.SteamID) (Person, error)
	GetPeopleBySteamID(ctx context.Context, transaction pgx.Tx, steamIDs steamid.Collection) (People, error)
	GetSteamsAtAddress(ctx context.Context, addr net.IP) (steamid.Collection, error)
	GetPeople(ctx context.Context, transaction pgx.Tx, filter PlayerQuery) (People, int64, error)
	GetOrCreatePersonBySteamID(ctx context.Context, transaction pgx.Tx, sid64 steamid.SteamID) (Person, error)
	GetPersonByDiscordID(ctx context.Context, discordID string) (Person, error)
	GetExpiredProfiles(ctx context.Context, limit uint64) ([]Person, error)
	GetPersonMessageByID(ctx context.Context, personMessageID int64) (PersonMessage, error)
	GetSteamIDsAbove(ctx context.Context, privilege Privilege) (steamid.Collection, error)
	GetSteamIDsByGroups(ctx context.Context, privileges []Privilege) (steamid.Collection, error)
	GetPersonSettings(ctx context.Context, steamID steamid.SteamID) (PersonSettings, error)
	SavePersonSettings(ctx context.Context, user PersonInfo, req PersonSettingsUpdate) (PersonSettings, error)
	SetSteam(ctx context.Context, transaction pgx.Tx, sid64 steamid.SteamID, discordID string) error
	SetPermissionLevel(ctx context.Context, transaction pgx.Tx, steamID steamid.SteamID, level Privilege) error
	UpdateProfiles(ctx context.Context, transaction pgx.Tx, people People) (int, error)
}

type PersonRepository interface {
	DropPerson(ctx context.Context, transaction pgx.Tx, steamID steamid.SteamID) error
	SavePerson(ctx context.Context, transaction pgx.Tx, person *Person) error
	GetPersonBySteamID(ctx context.Context, transaction pgx.Tx, sid64 steamid.SteamID) (Person, error)
	GetPeopleBySteamID(ctx context.Context, transaction pgx.Tx, steamIDs steamid.Collection) (People, error)
	GetSteamsAtAddress(ctx context.Context, addr net.IP) (steamid.Collection, error)
	GetSteamIDsByGroups(ctx context.Context, privileges []Privilege) (steamid.Collection, error)
	GetPeople(ctx context.Context, transaction pgx.Tx, filter PlayerQuery) (People, int64, error)
	GetPersonByDiscordID(ctx context.Context, discordID string) (Person, error)
	GetExpiredProfiles(ctx context.Context, transaction pgx.Tx, limit uint64) ([]Person, error)
	GetPersonMessageByID(ctx context.Context, personMessageID int64) (PersonMessage, error)
	GetSteamIDsAbove(ctx context.Context, privilege Privilege) (steamid.Collection, error)
	GetPersonSettings(ctx context.Context, steamID steamid.SteamID) (PersonSettings, error)
	SavePersonSettings(ctx context.Context, settings *PersonSettings) error
}

type RequestPermissionLevelUpdate struct {
	PermissionLevel Privilege `json:"permission_level"`
}

type PersonInfo interface {
	GetDiscordID() string
	GetName() string
	GetAvatar() AvatarLinks
	GetSteamID() steamid.SteamID
	Path() string // config.LinkablePath
	HasPermission(permission Privilege) bool
}

type SimplePerson struct {
	Personaname     string    `json:"personaname"`
	Avatarhash      string    `json:"avatarhash"`
	PermissionLevel Privilege `json:"permission_level"`
}

// UserProfile is the model used in the webui representing the logged-in user.
type UserProfile struct {
	SteamID         steamid.SteamID `json:"steam_id"`
	CreatedOn       time.Time       `json:"created_on"`
	UpdatedOn       time.Time       `json:"updated_on"`
	PermissionLevel Privilege       `json:"permission_level"`
	DiscordID       string          `json:"discord_id"`
	PatreonID       string          `json:"patreon_id"`
	Name            string          `json:"name"`
	Avatarhash      string          `json:"avatarhash"`
	BanID           int64           `json:"ban_id"`
	Muted           bool            `json:"muted"`
}

func (p UserProfile) HasPermission(privilege Privilege) bool {
	return p.PermissionLevel >= privilege
}

func (p UserProfile) GetDiscordID() string {
	return p.DiscordID
}

func (p UserProfile) GetName() string {
	if p.Name == "" {
		return p.SteamID.String()
	}

	return p.Name
}

func (p UserProfile) GetAvatar() AvatarLinks {
	return NewAvatarLinks(p.Avatarhash)
}

func (p UserProfile) GetSteamID() steamid.SteamID {
	return p.SteamID
}

func (p UserProfile) Path() string {
	return fmt.Sprintf("/profile/%d", p.SteamID.Int64())
}

// NewUserProfile allocates a new default person instance.
func NewUserProfile(sid64 steamid.SteamID) UserProfile {
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
	SteamID          steamid.SteamID       `db:"steam_id" json:"steam_id"`
	CreatedOn        time.Time             `json:"created_on"`
	UpdatedOn        time.Time             `json:"updated_on"`
	PermissionLevel  Privilege             `json:"permission_level"`
	Muted            bool                  `json:"muted"`
	IsNew            bool                  `json:"-"`
	DiscordID        string                `json:"discord_id"`
	PatreonID        string                `json:"patreon_id"`
	IPAddr           netip.Addr            `json:"-"` // TODO Allow json for admins endpoints
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

func (p Person) HasPermission(privilege Privilege) bool {
	return p.PermissionLevel >= privilege
}

func (p Person) GetAvatar() AvatarLinks {
	return NewAvatarLinks(p.AvatarHash)
}

func (p Person) GetSteamID() steamid.SteamID {
	return p.SteamID
}

func (p Person) Path() string {
	return fmt.Sprintf("/profile/%d", p.SteamID.Int64())
}

func (p Person) ToUserProfile() UserProfile {
	return UserProfile{
		SteamID:         p.SteamID,
		CreatedOn:       p.CreatedOn,
		UpdatedOn:       p.UpdatedOn,
		PermissionLevel: p.PermissionLevel,
		DiscordID:       p.DiscordID,
		PatreonID:       p.PatreonID,
		Name:            p.PersonaName,
		Avatarhash:      p.AvatarHash,
		BanID:           0,
		Muted:           p.Muted,
	}
}

// LoggedIn checks for a valid steamID.
func (p Person) LoggedIn() bool {
	return p.SteamID.Valid() && p.SteamID.Int64() > 0
}

type ProfileResponse struct {
	Player   *Person           `json:"player"`
	Friends  []steamweb.Friend `json:"friends"`
	Settings PersonSettings    `json:"settings"`
}

// NewPerson allocates a new default person instance.
func NewPerson(sid64 steamid.SteamID) Person {
	curTime := time.Now()

	return Person{
		SteamID:          sid64,
		CreatedOn:        curTime,
		UpdatedOn:        curTime,
		PermissionLevel:  PUser,
		Muted:            false,
		IsNew:            true,
		DiscordID:        "",
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

func (p People) AsMap() map[steamid.SteamID]Person {
	m := map[steamid.SteamID]Person{}
	for _, person := range p {
		m[person.SteamID] = person
	}

	return m
}

type UserNotification struct {
	PersonNotificationID int64                `json:"person_notification_id"`
	SteamID              steamid.SteamID      `json:"steam_id"`
	Read                 bool                 `json:"read"`
	Deleted              bool                 `json:"deleted"`
	Severity             NotificationSeverity `json:"severity"`
	Message              string               `json:"message"`
	Link                 string               `json:"link"`
	Count                int                  `json:"count"`
	Author               *UserProfile         `json:"author"`
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
	PersonAuthID int64           `json:"person_auth_id"`
	SteamID      steamid.SteamID `json:"steam_id"`
	IPAddr       net.IP          `json:"ip_addr"`
	AccessToken  string          `json:"access_token"`
	CreatedOn    time.Time       `json:"created_on"`
}

func NewPersonAuth(sid64 steamid.SteamID, addr net.IP, accessToken string) PersonAuth {
	return PersonAuth{
		PersonAuthID: 0,
		SteamID:      sid64,
		IPAddr:       addr,
		AccessToken:  accessToken,
		CreatedOn:    time.Now(),
	}
}

type PersonConnection struct {
	PersonConnectionID int64           `json:"person_connection_id"`
	IPAddr             netip.Addr      `json:"ip_addr"`
	SteamID            steamid.SteamID `json:"steam_id"`
	PersonaName        string          `json:"persona_name"`
	ServerID           int             `json:"server_id"`
	CreatedOn          time.Time       `json:"created_on"`
	ServerNameShort    string          `json:"server_name_short"`
	ServerName         string          `json:"server_name"`
}

type PersonConnections []PersonConnection

type PersonMessage struct {
	PersonMessageID   int64           `json:"person_message_id"`
	MatchID           uuid.UUID       `json:"match_id"`
	SteamID           steamid.SteamID `json:"steam_id"`
	AvatarHash        string          `json:"avatar_hash"`
	PersonaName       string          `json:"persona_name"`
	ServerName        string          `json:"server_name"`
	ServerID          int             `json:"server_id"`
	Body              string          `json:"body"`
	Team              bool            `json:"team"`
	CreatedOn         time.Time       `json:"created_on"`
	AutoFilterFlagged int64           `json:"auto_filter_flagged"`
}

type PersonMessages []PersonMessage

type QueryChatHistoryResult struct {
	PersonMessage
	Pattern string `json:"pattern"`
}

type PersonSettings struct {
	PersonSettingsID     int64           `json:"person_settings_id"`
	SteamID              steamid.SteamID `json:"steam_id"`
	ForumSignature       string          `json:"forum_signature"`
	ForumProfileMessages bool            `json:"forum_profile_messages"`
	StatsHidden          bool            `json:"stats_hidden"`

	// This key will be absent to indicate that this feature
	// is disabled (and UI should not be shown to the user).
	CenterProjectiles *bool `json:"center_projectiles,omitempty"`

	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

type PersonSettingsUpdate struct {
	ForumSignature       string `json:"forum_signature"`
	ForumProfileMessages bool   `json:"forum_profile_messages"`
	StatsHidden          bool   `json:"stats_hidden"`
	CenterProjectiles    *bool  `json:"center_projectiles,omitempty"`
}

type UserWarning struct {
	WarnReason    Reason    `json:"warn_reason"`
	Message       string    `json:"message"`
	Matched       string    `json:"matched"`
	MatchedFilter Filter    `json:"matched_filter"`
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
	PlayerID    int
	UserWarning
}

type Warnings interface {
	State() map[string][]UserWarning
}
