package model

import (
	"fmt"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"net"
	"time"
)

var (
	ErrRCON = errors.New("RCON error")
)

type BanType int

const (
	Unknown BanType = -1
	OK      BanType = 0
	NoComm  BanType = 1
	Banned  BanType = 2
)

type BanSource int

const (
	System BanSource = 0
	Bot    BanSource = 1
	Web    BanSource = 2
	InGame BanSource = 3
)

func (s BanSource) String() string {
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

type Reason int

const (
	Custom           Reason = 1
	External         Reason = 2
	Cheating         Reason = 3
	Racism           Reason = 4
	Harassment       Reason = 5
	Exploiting       Reason = 6
	WarningsExceeded Reason = 7
)

var reasonStr = map[Reason]string{
	Custom:     "",
	External:   "3rd party",
	Cheating:   "Cheating",
	Racism:     "Racism",
	Harassment: "Person Harassment",
	Exploiting: "Exploiting",
}

func ReasonString(reason Reason) string {
	return reasonStr[reason]
}

type BanNet struct {
	NetID     int64     `db:"net_id"`
	CIDR      string    `db:"cidr"`
	Source    BanSource `source:"source"`
	Reason    string    `db:"reason"`
	CreatedOn int64     `db:"created_on" json:"created_on"`
	UpdatedOn int64     `db:"updated_on" json:"updated_on"`
	Until     int64     `db:"until"`
}

func NewBanNet(cidr string, reason string, duration time.Duration, source BanSource) (BanNet, error) {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return BanNet{}, err
	}
	if duration.Seconds() == 0 {
		// 100 Years
		duration = time.Hour * 8760 * 100
	}
	return BanNet{
		CIDR:      cidr,
		Source:    source,
		Reason:    reason,
		CreatedOn: time.Now().Unix(),
		UpdatedOn: time.Now().Unix(),
		Until:     time.Now().Add(duration).Unix(),
	}, nil
}

func (b BanNet) String() string {
	return fmt.Sprintf("Net: %s Source: %s Reason: %s", b.CIDR, b.Source, b.Reason)
}

type Ban struct {
	BanID int64 `db:"ban_id" json:"ban_id"`
	// SteamID is the steamID of the banned person
	SteamID  steamid.SID64 `db:"steam_id" json:"steam_id"`
	AuthorID steamid.SID64 `db:"author_id" json:"author_id"`
	// Reason defines the overall ban classification
	BanType BanType `db:"ban_type" json:"ban_type"`
	// Reason defines the overall ban classification
	Reason Reason `db:"reason" json:"reason"`
	// ReasonText is returned to the client when kicked trying to join the server
	ReasonText string `db:"reason_text" json:"reason_text"`
	// Note is a supplementary note added by admins that is hidden from normal view
	Note   string    `db:"note" json:"note"`
	Source BanSource `json:"ban_source" db:"ban_source"`
	// Until is when the ban will be no longer valid. 0 denotes forever
	Until     int64 `json:"until" db:"until"`
	CreatedOn int64 `db:"created_on" json:"created_on"`
	UpdatedOn int64 `db:"updated_on" json:"updated_on"`
}

func (b Ban) String() string {
	return fmt.Sprintf("SID: %d Source: %s Reason: %s Type: %v",
		b.SteamID.Int64(), b.Source, b.ReasonText, b.BanType)
}

type BannedPerson struct {
	BanID    int64         `db:"ban_id" json:"ban_id"`
	SteamID  steamid.SID64 `db:"steam_id" json:"steam_id"`
	AuthorID steamid.SID64 `db:"author_id" json:"author_id"`
	// Reason defines the overall ban classification
	BanType BanType `db:"ban_type" json:"ban_type"`
	// Reason defines the overall ban classification
	Reason Reason `db:"reason" json:"reason"`
	// ReasonText is returned to the client when kicked trying to join the server
	ReasonText string `db:"reason_text" json:"reason_text"`
	// Note is a supplementary note added by admins that is hidden from normal view
	Note   string    `db:"note" json:"note"`
	Source BanSource `json:"ban_source" db:"ban_source"`
	// Until is when the ban will be no longer valid. 0 denotes forever
	Until        int64  `json:"until" db:"until"`
	CreatedOn    int64  `db:"created_on" json:"created_on"`
	UpdatedOn    int64  `db:"updated_on" json:"updated_on"`
	PersonaName  string `json:"personaname" db:"personaname"`
	ProfileURL   string `json:"profileurl" db:"profileurl"`
	Avatar       string `json:"avatar" db:"avatar"`
	AvatarMedium string `json:"avatarmedium" db:"avatarmedium"`
}

type Server struct {
	// Auto generated id
	ServerID int64 `db:"server_id"`
	// ServerName is a short reference name for the server eg: us-1
	ServerName string `db:"short_name"`
	// Token is the current valid authentication token that the server uses to make authenticated requests
	Token string `db:"token"`
	// Address is the ip of the server
	Address string `db:"address"`
	// Port is the port of the server
	Port int `db:"port"`
	// RCON is the RCON password for the server
	RCON          string `db:"rcon"`
	ReservedSlots int    `db:"reserved_slots" json:"reserved_slots"`
	// Password is what the server uses to generate a token to make authenticated calls
	Password string `db:"password"`
	// TokenCreatedOn is set when changing the token
	TokenCreatedOn int64 `db:"token_created_on"`
	CreatedOn      int64 `db:"created_on"`
	UpdatedOn      int64 `db:"updated_on"`
}

func (s Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

func (s Server) Slots(statusSlots int) int {
	return statusSlots - s.ReservedSlots
}

type Person struct {
	SteamID   steamid.SID64 `db:"steam_id"`
	Name      string        `db:"name"`
	IPAddr    string        `db:"ip_addr"`
	CreatedOn int64         `db:"created_on"`
	UpdatedOn int64         `db:"updated_on"`
	extra.PlayerSummary
}

// EqualID is used for html templates which assume int and not int64 types
func (p Person) LoggedIn() bool {
	return p.SteamID.Valid() && p.SteamID.Int64() > 0
}

func NewPerson() Person {
	return Person{
		CreatedOn: time.Now().Unix(),
		UpdatedOn: time.Now().Unix(),
	}
}

type AppealState int

const (
	ASNew     AppealState = 0
	ASDenied  AppealState = 1
	ASGranted AppealState = 2
)

type Appeal struct {
	AppealID    int         `db:"appeal_id" json:"appeal_id"`
	BanID       int         `db:"ban_id" json:"ban_id"`
	AppealText  string      `db:"appeal_text" json:"appeal_text"`
	AppealState AppealState `db:"appeal_state" json:"appeal_state"`
	Email       string      `db:"email" json:"email"`
	CreatedOn   int64       `db:"created_on" json:"created_on"`
	UpdatedOn   int64       `db:"updated_on" json:"updated_on"`
}

type Stats struct {
	BansTotal     int `json:"bans_total"`
	BansDay       int `json:"bans_day"`
	BansWeek      int `json:"bans_week"`
	BansMonth     int `json:"bans_month"`
	BansCIDRTotal int `json:"bans_cidr_total"`
	AppealsOpen   int `json:"appeals_open"`
	AppealsClosed int `json:"appeals_closed"`
	FilteredWords int `json:"filtered_words"`
	ServersAlive  int `json:"servers_alive"`
	ServersTotal  int `json:"servers_total"`
}
