package model

import (
	"fmt"
	"github.com/leighmacdonald/steamid/steamid"
	"github.com/pkg/errors"
)

var (
	ErrNoResult  = errors.New("No results found")
	ErrDuplicate = errors.New("Duplicate entity")
	ErrRCON      = errors.New("RCON error")
)

type BanType int

const (
	Unknown BanType = -1
	OK      BanType = 0
	NoComm  BanType = 1
	Banned  BanType = 2
)

type Reason int

const (
	Custom     Reason = 1
	External   Reason = 2
	Cheating   Reason = 3
	Racism     Reason = 4
	Harassment Reason = 5
	Exploiting Reason = 6
)

var reasonStr = map[Reason]string{
	Custom:     "",
	External:   "3rd party",
	Cheating:   "Cheating",
	Racism:     "Racism",
	Harassment: "Player Harassment",
	Exploiting: "Exploiting",
}

func ReasonString(reason Reason) string {
	return reasonStr[reason]
}

type Ban struct {
	BanID int64 `db:"ban_id" json:"ban_id"`
	// SteamID is the steamID of the banned person
	SteamID steamid.SID64 `db:"steam_id" json:"steam_id"`
	// AuthorID is the steamID of the person making the ban
	AuthorID steamid.SID64 `db:"author_id" json:"author_id"`
	// Reason defines the overall ban classification
	BanType BanType `db:"ban_type" json:"ban_type"`
	// Reason defines the overall ban classification
	Reason Reason `db:"reason" json:"reason"`
	// IP is the banned client ip, currently only ipv4 as that is all that srcds supports i believe
	IP string `db:"ip" json:"ip"`
	// ReasonText is returned to the client when kicked trying to join the server
	ReasonText string `db:"reason_text" json:"reason_text"`
	// Note is a supplementary note added by admins that is hidden from normal view
	Note string `db:"note" json:"note"`
	// Until is when the ban will be no longer valid. 0 denotes forever
	Until     int64 `json:"until" db:"until"`
	CreatedOn int64 `db:"created_on" json:"created_on"`
	UpdatedOn int64 `db:"updated_on" json:"updated_on"`
}

type Server struct {
	// Auto generated id
	ServerID int64 `db:"server_id"`
	// ServerName is a short reference name for the server eg: us-1
	ServerName string `db:"server_name"`
	// Token is the current valid authentication token that the server uses to make authenticated requests
	Token string `db:"token"`
	// Address is the ip of the server
	Address string `db:"address"`
	// Port is the port of the server
	Port int `db:"port"`
	// RCON is the RCON password for the server
	RCON string `db:"rcon"`
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
