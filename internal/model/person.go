package model

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"net"
	"time"
)

const refreshTokenLen = 80

type ServerPermission struct {
	SteamId         steamid.SID `json:"steam_id"`
	PermissionLevel Privilege   `json:"permission_level"`
	Flags           string      `json:"flags"`
}

type Person struct {
	// TODO merge use of steamid & steam_id
	SteamID          steamid.SID64 `db:"steam_id" json:"steam_id,string"`
	CreatedOn        time.Time     `json:"created_on"`
	UpdatedOn        time.Time     `json:"updated_on"`
	PermissionLevel  Privilege     `json:"permission_level"`
	Muted            bool          `json:"muted"`
	IsNew            bool          `json:"-"`
	DiscordID        string        `json:"discord_id"`
	IPAddr           net.IP        `json:"-"` // TODO Allow json for admins endpoints
	CommunityBanned  bool          `json:"community_banned"`
	VACBans          int           `json:"vac_bans"`
	GameBans         int           `json:"game_bans"`
	EconomyBan       string        `json:"economy_ban"`
	DaysSinceLastBan int           `json:"days_since_last_ban"`
	UpdatedOnSteam   time.Time     `json:"updated_on_steam"`
	*steamweb.PlayerSummary
}

func (p *Person) ToURL() string {
	return config.ExtURL("/profile/%d", p.SteamID.Int64())
}

// LoggedIn checks for a valid steamID
func (p *Person) LoggedIn() bool {
	return p.SteamID.Valid() && p.SteamID.Int64() > 0
}

// AsTarget checks for a valid steamID
func (p *Person) AsTarget() StringSID {
	return StringSID(p.SteamID.String())
}

type SimplePerson struct {
	SteamId     steamid.SID64 `json:"steam_id"`
	PersonaName string        `json:"persona_name"`
	Avatar      string        `json:"avatar"`
	AvatarFull  string        `json:"avatar_full"`
}

type AppealOverview struct {
	BanSteam

	SourceSteamId     steamid.SID64 `json:"source_steam_id"`
	SourcePersonaName string        `json:"source_persona_name"`
	SourceAvatar      string        `json:"source_avatar"`
	SourceAvatarFull  string        `json:"source_avatar_full"`

	TargetSteamId     steamid.SID64 `json:"target_steam_id"`
	TargetPersonaName string        `json:"target_persona_name"`
	TargetAvatar      string        `json:"target_avatar"`
	TargetAvatarFull  string        `json:"target_avatar_full"`
}

// NewPerson allocates a new default person instance
func NewPerson(sid64 steamid.SID64) Person {
	t0 := config.Now()
	return Person{
		SteamID:          sid64,
		CreatedOn:        t0,
		UpdatedOn:        t0,
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
		UpdatedOnSteam:   t0,
		PlayerSummary: &steamweb.PlayerSummary{
			Steamid: sid64.String(),
		},
	}
}

type People []Person

func (p People) AsMap() map[steamid.SID64]Person {
	m := map[steamid.SID64]Person{}
	for _, person := range p {
		m[person.SteamID] = person
	}
	return m
}

type PersonChat struct {
	PersonChatId int64
	SteamId      steamid.SID64
	ServerId     int
	TeamChat     bool
	Message      string
	CreatedOn    time.Time
}

// PersonIPRecord holds a composite result of the more relevant ip2location results
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

// UserProfile is the model used in the webui representing the logged-in user.
type UserProfile struct {
	SteamID         steamid.SID64      `db:"steam_id" json:"steam_id,string"`
	CreatedOn       time.Time          `json:"created_on"`
	UpdatedOn       time.Time          `json:"updated_on"`
	PermissionLevel Privilege          `json:"permission_level"`
	DiscordID       string             `json:"discord_id"`
	Name            string             `json:"name"`
	Avatar          string             `json:"avatar"`
	AvatarFull      string             `json:"avatarfull"`
	BanID           int64              `json:"ban_id"`
	Muted           bool               `json:"muted"`
	Notifications   []UserNotification `json:"notifications"`
}

func (p UserProfile) ToURL() string {
	return config.ExtURL("/profile/%d", p.SteamID.Int64())
}

// NewUserProfile allocates a new default person instance
func NewUserProfile(sid64 steamid.SID64) UserProfile {
	t0 := config.Now()
	return UserProfile{
		SteamID:         sid64,
		CreatedOn:       t0,
		UpdatedOn:       t0,
		PermissionLevel: PUser,
		Name:            "Guest",
	}
}

type PersonAuth struct {
	PersonAuthId int64         `json:"person_auth_id"`
	SteamId      steamid.SID64 `json:"steam_id"`
	IpAddr       net.IP        `json:"ip_addr"`
	RefreshToken string        `json:"refresh_token"`
	CreatedOn    time.Time     `json:"created_on"`
}

func NewPersonAuth(sid64 steamid.SID64, addr net.IP) PersonAuth {
	return PersonAuth{
		PersonAuthId: 0,
		SteamId:      sid64,
		IpAddr:       addr,
		RefreshToken: golib.RandomString(refreshTokenLen),
		CreatedOn:    config.Now(),
	}
}
