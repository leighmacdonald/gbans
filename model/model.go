package model

import (
	"fmt"
	"github.com/leighmacdonald/gbans/config"
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
	Spam             Reason = 8
)

var reasonStr = map[Reason]string{
	Custom:           "",
	External:         "3rd party",
	Cheating:         "Cheating",
	Racism:           "Racism",
	Harassment:       "Person Harassment",
	Exploiting:       "Exploiting",
	WarningsExceeded: "Warnings Exceeding",
	Spam:             "Spam",
}

func ReasonString(reason Reason) string {
	return reasonStr[reason]
}

type BanNet struct {
	NetID      int64         `db:"net_id"`
	SteamID    steamid.SID64 `db:"steam_id"`
	AuthorID   steamid.SID64 `db:"author_id"`
	CIDR       *net.IPNet    `db:"cidr"`
	Source     BanSource     `source:"source"`
	Reason     string        `db:"reason"`
	CreatedOn  time.Time     `db:"created_on" json:"created_on"`
	UpdatedOn  time.Time     `db:"updated_on" json:"updated_on"`
	ValidUntil time.Time     `db:"valid_until"`
}

func NewBan(steamID steamid.SID64, authorID steamid.SID64, reason string, duration time.Duration, source BanSource) (Ban, error) {
	if duration.Seconds() == 0 {
		// 100 Years
		duration = time.Hour * 8760 * 100
	}
	return Ban{
		SteamID:    steamID,
		AuthorID:   authorID,
		BanType:    Banned,
		Reason:     0,
		ReasonText: reason,
		Note:       "",
		Source:     source,
		ValidUntil: config.Now().Add(duration),
		CreatedOn:  config.Now(),
		UpdatedOn:  config.Now(),
	}, nil
}

func NewBanNet(cidr string, reason string, duration time.Duration, source BanSource) (BanNet, error) {
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		return BanNet{}, err
	}
	if duration.Seconds() == 0 {
		// 100 Years
		duration = time.Hour * 8760 * 100
	}
	return BanNet{
		CIDR:       n,
		Source:     source,
		Reason:     reason,
		CreatedOn:  config.Now(),
		UpdatedOn:  config.Now(),
		ValidUntil: config.Now().Add(duration),
	}, nil
}

func (b BanNet) String() string {
	return fmt.Sprintf("Net: %s Source: %s Reason: %s", b.CIDR, b.Source, b.Reason)
}

type Ban struct {
	BanID uint64 `db:"ban_id" json:"ban_id"`
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
	// ValidUntil is when the ban will be no longer valid. 0 denotes forever
	ValidUntil time.Time `json:"valid_until" db:"valid_until"`
	CreatedOn  time.Time `db:"created_on" json:"created_on"`
	UpdatedOn  time.Time `db:"updated_on" json:"updated_on"`
}

func (b Ban) String() string {
	return fmt.Sprintf("SID: %d Source: %s Reason: %s Type: %v",
		b.SteamID.Int64(), b.Source, b.ReasonText, b.BanType)
}

type BannedPerson struct {
	BanID    uint64        `db:"ban_id" json:"ban_id"`
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
	// ValidUntil is when the ban will be no longer valid. 0 denotes forever
	ValidUntil   time.Time `json:"valid_until" db:"valid_until"`
	CreatedOn    time.Time `db:"created_on" json:"created_on"`
	UpdatedOn    time.Time `db:"updated_on" json:"updated_on"`
	PersonaName  string    `json:"personaname" db:"personaname"`
	ProfileURL   string    `json:"profileurl" db:"profileurl"`
	Avatar       string    `json:"avatar" db:"avatar"`
	AvatarMedium string    `json:"avatarmedium" db:"avatarmedium"`
	AvatarFull   string    `json:"avatarfull" db:"avatarfull"`
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
	TokenCreatedOn time.Time `db:"token_created_on"`
	CreatedOn      time.Time `db:"created_on"`
	UpdatedOn      time.Time `db:"updated_on"`
}

func (s Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

func (s Server) Slots(statusSlots int) int {
	return statusSlots - s.ReservedSlots
}

type Person struct {
	SteamID   steamid.SID64 `db:"steam_id" json:"steam_id"`
	Name      string        `db:"name" json:"name"`
	IPAddr    string        `db:"ip_addr" json:"ip_addr"`
	CreatedOn time.Time     `db:"created_on" json:"created_on"`
	UpdatedOn time.Time     `db:"updated_on" json:"updated_on"`
	IsNew     bool          `db:"-" json:"-"`
	*extra.PlayerSummary
}

// EqualID is used for html templates which assume int and not int64 types
func (p *Person) LoggedIn() bool {
	return p.SteamID.Valid() && p.SteamID.Int64() > 0
}

func NewPerson(sid64 steamid.SID64) *Person {
	return &Person{
		SteamID:       sid64,
		IsNew:         true,
		CreatedOn:     config.Now(),
		UpdatedOn:     config.Now(),
		PlayerSummary: &extra.PlayerSummary{},
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
	BanID       uint64      `db:"ban_id" json:"ban_id"`
	AppealText  string      `db:"appeal_text" json:"appeal_text"`
	AppealState AppealState `db:"appeal_state" json:"appeal_state"`
	Email       string      `db:"email" json:"email"`
	CreatedOn   time.Time   `db:"created_on" json:"created_on"`
	UpdatedOn   time.Time   `db:"updated_on" json:"updated_on"`
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
