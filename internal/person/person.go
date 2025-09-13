package person

import (
	"fmt"
	"net/netip"
	"time"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/playerqueue"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type SteamMember interface {
	IsMember(steamID steamid.SteamID) (int64, bool)
}

type PlayerQuery struct {
	domain.QueryFilter
	domain.TargetIDField
	Personaname string `json:"personaname"`
	IP          string `json:"ip"`
	StaffOnly   bool   `json:"staff_only"`
}

type RequestPermissionLevelUpdate struct {
	PermissionLevel permission.Privilege `json:"permission_level"`
}

// UserProfile is the model used in the webui representing the logged-in user.
type UserProfile struct {
	SteamID               steamid.SteamID        `json:"steam_id"`
	CreatedOn             time.Time              `json:"created_on"`
	UpdatedOn             time.Time              `json:"updated_on"`
	PermissionLevel       permission.Privilege   `json:"permission_level"`
	DiscordID             string                 `json:"discord_id"`
	PatreonID             string                 `json:"patreon_id"`
	Name                  string                 `json:"name"`
	Avatarhash            string                 `json:"avatarhash"`
	BanID                 int64                  `json:"ban_id"`
	Muted                 bool                   `json:"muted"`
	PlayerqueueChatStatus playerqueue.ChatStatus `json:"playerqueue_chat_status"`
}

// EconBanState  holds the users current economy ban status.
type EconBanState string

// EconBanState values
//
//goland:noinspection ALL
const (
	EconBanNone      EconBanState = "none"
	EconBanProbation EconBanState = "probation"
	EconBanBanned    EconBanState = "banned"
)

type Person struct {
	// TODO merge use of steamid & steam_id
	SteamID               steamid.SteamID        `json:"steam_id"`
	CreatedOn             time.Time              `json:"created_on"`
	UpdatedOn             time.Time              `json:"updated_on"`
	PermissionLevel       permission.Privilege   `json:"permission_level"`
	Muted                 bool                   `json:"muted"`
	IsNew                 bool                   `json:"-"`
	DiscordID             string                 `json:"discord_id"`
	PatreonID             string                 `json:"patreon_id"`
	IPAddr                netip.Addr             `json:"-"` // TODO Allow json for admins endpoints
	CommunityBanned       bool                   `json:"community_banned"`
	VACBans               int                    `json:"vac_bans"`
	GameBans              int                    `json:"game_bans"`
	EconomyBan            EconBanState           `json:"economy_ban"`
	DaysSinceLastBan      int                    `json:"days_since_last_ban"`
	UpdatedOnSteam        time.Time              `json:"updated_on_steam"`
	PlayerqueueChatStatus playerqueue.ChatStatus `json:"playerqueue_chat_status"`
	PlayerqueueChatReason string                 `json:"playerqueue_chat_reason"`
	AvatarHash            string                 `json:"avatar_hash"`
	CommentPermission     int64                  `json:"comment_permission"`
	LastLogoff            int64                  `json:"last_logoff"`
	LocCityID             int64                  `json:"loc_city_id"`
	LocCountryCode        string                 `json:"loc_country_code"`
	LocStateCode          string                 `json:"loc_state_code"`
	PersonaName           string                 `json:"persona_name"`
	PersonaState          int64                  `json:"persona_state"`
	PersonaStateFlags     int64                  `json:"persona_state_flags"`
	PrimaryClanID         string                 `json:"primary_clan_id"`
	ProfileState          int64                  `json:"profile_state"`
	ProfileURL            string                 `json:"profile_url"`
	RealName              string                 `json:"real_name"`
	TimeCreated           int64                  `json:"time_created"`
	VisibilityState       int64                  `json:"visibility_state"`
}

func (p Person) Avatar() string {
	return p.GetAvatar().Small()
}

func (p Person) AvatarMedium() string {
	return p.GetAvatar().Medium()
}

func (p Person) AvatarFull() string {
	return p.GetAvatar().Full()
}

func (p Person) Profile() string {
	return "https://steamcommunity.com/profiles/" + p.SteamID.String()
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

func (p Person) Permissions() permission.Privilege {
	return p.PermissionLevel
}

func (p Person) HasPermission(privilege permission.Privilege) bool {
	return p.PermissionLevel >= privilege
}

func (p Person) GetAvatar() domain.Avatar {
	return domain.NewAvatar(p.AvatarHash)
}

func (p Person) GetSteamID() steamid.SteamID {
	return p.SteamID
}

func (p Person) Path() string {
	return fmt.Sprintf("/profile/%d", p.SteamID.Int64())
}

// LoggedIn checks for a valid steamID.
func (p Person) LoggedIn() bool {
	return p.SteamID.Valid() && p.SteamID.Int64() > 0
}

type ProfileResponse struct {
	Player   *Person                  `json:"player"`
	Friends  []thirdparty.SteamFriend `json:"friends"`
	Settings PersonSettings           `json:"settings"`
}

// NewPerson allocates a new default person instance.
func NewPerson(sid64 steamid.SteamID) Person {
	curTime := time.Now()

	return Person{
		SteamID:               sid64,
		CreatedOn:             curTime,
		UpdatedOn:             curTime,
		PermissionLevel:       permission.PUser,
		Muted:                 false,
		IsNew:                 true,
		DiscordID:             "",
		CommunityBanned:       false,
		VACBans:               0,
		GameBans:              0,
		EconomyBan:            "none",
		DaysSinceLastBan:      0,
		UpdatedOnSteam:        time.Unix(0, 0),
		PlayerqueueChatStatus: "readwrite",
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
