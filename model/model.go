package model

import "github.com/leighmacdonald/steamid/steamid"

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
	BanID      int64         `db:"ban_id" json:"ban_id"`
	SteamID    steamid.SID64 `db:"steam_id" json:"steam_id"`
	AuthorID   steamid.SID64 `db:"author_id" json:"author_id"`
	Reason     Reason        `db:"reason" json:"reason"`
	IP         string        `db:"ip" json:"ip"`
	ReasonText string        `db:"reason_text" json:"reason_text"`
	Note       string        `db:"note" json:"note"`
	CreatedOn  int64         `db:"created_on" json:"created_on"`
	UpdatedOn  int64         `db:"updated_on" json:"updated_on"`
}

type Server struct {
	ServerID       int64  `db:"server_id"`
	ServerName     string `db:"server_name"`
	Token          string `db:"token"`
	Address        string `db:"address"`
	Port           int    `db:"port"`
	RCON           string `db:"rcon"`
	Password       string `db:"password"`
	TokenCreatedOn int64  `db:"token_created_on"`
	CreatedOn      int64  `db:"created_on"`
	UpdatedOn      int64  `db:"updated_on"`
}
