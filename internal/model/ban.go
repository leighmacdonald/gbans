package model

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"net"
	"time"
)

// Target defines who the request is being made against
type Target string

func (t Target) SID64() (steamid.SID64, error) {
	// TODO pass ctx, or remove resolve?
	resolveCtx, cancelResolve := context.WithTimeout(context.Background(), time.Second*5)
	defer cancelResolve()
	sid64, errResolveSID := steamid.ResolveSID64(resolveCtx, string(t))
	if errResolveSID != nil {
		return 0, consts.ErrInvalidSID
	}
	if !sid64.Valid() {
		return 0, consts.ErrInvalidSID
	}
	return sid64, nil
}

// Duration defines the length of time the action should be valid for
// A duration of 0 will be interpreted as permanent and set to 10 years in the future
type Duration string

func (value Duration) Value() (time.Duration, error) {
	duration, errDuration := config.ParseDuration(string(value))
	if errDuration != nil {
		return 0, consts.ErrInvalidDuration
	}
	if duration < 0 {
		return 0, consts.ErrInvalidDuration
	}
	if duration == 0 {
		duration = time.Hour * 24 * 365 * 10
	}
	return duration, nil
}

// BanType defines the state of the ban for a user, 0 being no ban
type BanType int

const (
	// Unknown means the ban state could not be determined, failing-open to allowing players
	// to connect.
	Unknown BanType = -1
	// OK Ban state is clean
	OK BanType = 0
	// NoComm means the player cannot communicate while playing voice + chat
	NoComm BanType = 1
	// Banned means the player cannot join the server at all
	Banned BanType = 2
)

// Origin defines the origin of the ban or action
type Origin int

const (
	// System is an automatic ban triggered by the service
	System Origin = 0
	// Bot is a ban using the discord bot interface
	Bot Origin = 1
	// Web is a ban using the web-ui
	Web Origin = 2
	// InGame is a ban using the sourcemod plugin
	InGame Origin = 3
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

// Reason defined a set of predefined ban reasons
// TODO make this fully dynamic?
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
	Language         Reason = 9
	Profile          Reason = 10
	ItemDescriptions Reason = 11
	BotHost          Reason = 12
)

var reasonStr = map[Reason]string{
	Custom:           "Custom",
	External:         "3rd party",
	Cheating:         "Cheating",
	Racism:           "Racism",
	Harassment:       "Person Harassment",
	Exploiting:       "Exploiting",
	WarningsExceeded: "Warnings Exceeding",
	Spam:             "Spam",
	Language:         "Language",
	Profile:          "Profile",
	ItemDescriptions: "Item Name or Descriptions",
	BotHost:          "BotHost",
}

func (r Reason) String() string {
	return reasonStr[r]
}

type BanASN struct {
	BanASNId   int64
	ASNum      int64
	Origin     Origin
	AuthorID   steamid.SID64
	TargetID   steamid.SID64
	Reason     Reason
	ReasonText string
	ValidUntil time.Time
	CreatedOn  time.Time
	UpdatedOn  time.Time
}

func NewBanASN(asn int64, authorId steamid.SID64, reason Reason, duration time.Duration) BanASN {
	if duration.Seconds() == 0 {
		// 100 Years
		duration = time.Hour * 8760 * 100
	}
	return BanASN{
		ASNum:      asn,
		Origin:     System,
		AuthorID:   authorId,
		TargetID:   0,
		Reason:     reason,
		ValidUntil: config.Now().Add(duration),
		CreatedOn:  config.Now(),
		UpdatedOn:  config.Now(),
	}
}

type BanNet struct {
	NetID      int64         `db:"net_id"`
	SteamID    steamid.SID64 `db:"steam_id"`
	AuthorID   steamid.SID64 `db:"author_id"`
	CIDR       *net.IPNet    `db:"cidr"`
	Source     Origin        `db:"source"`
	Reason     Reason        `db:"reason"`
	ReasonText string        `db:"reason_text"`
	CreatedOn  time.Time     `db:"created_on" json:"created_on"`
	UpdatedOn  time.Time     `db:"updated_on" json:"updated_on"`
	ValidUntil time.Time     `db:"valid_until"`
}

func NewBan(steamID steamid.SID64, authorID steamid.SID64, duration time.Duration) Ban {
	if duration.Seconds() == 0 {
		// 100 Years
		duration = time.Hour * 8760 * 100
	}
	return Ban{
		SteamID:    steamID,
		AuthorID:   authorID,
		BanType:    Banned,
		Reason:     Custom,
		ReasonText: "Unspecified",
		Note:       "",
		Source:     System,
		ValidUntil: config.Now().Add(duration),
		CreatedOn:  config.Now(),
		UpdatedOn:  config.Now(),
	}
}

func NewBanNet(cidr string, reason Reason, duration time.Duration, source Origin) (BanNet, error) {
	_, network, errParseCIDR := net.ParseCIDR(cidr)
	if errParseCIDR != nil {
		return BanNet{}, errParseCIDR
	}
	if duration.Seconds() == 0 {
		// 100 Years
		duration = time.Hour * 8760 * 100
	}
	return BanNet{
		CIDR:       network,
		Source:     source,
		Reason:     reason,
		CreatedOn:  config.Now(),
		UpdatedOn:  config.Now(),
		ValidUntil: config.Now().Add(duration),
	}, nil
}

func (b BanNet) String() string {
	return fmt.Sprintf("Net: %s Origin: %s Reason: %s", b.CIDR, b.Source, b.Reason)
}

type Ban struct {
	BanID int64 `db:"ban_id" json:"ban_id"`
	// SteamID is the steamID of the banned person
	SteamID  steamid.SID64 `db:"steam_id" json:"steam_id,string"`
	AuthorID steamid.SID64 `db:"author_id" json:"author_id,string"`
	// Reason defines the overall ban classification
	BanType BanType `db:"ban_type" json:"ban_type"`
	// Reason defines the overall ban classification
	Reason Reason `db:"reason" json:"reason"`
	// ReasonText is returned to the client when kicked trying to join the server
	ReasonText      string `db:"reason_text" json:"reason_text"`
	UnbanReasonText string `db:"unban_reason_text" json:"unban_reason_text"`
	// Note is a supplementary note added by admins that is hidden from normal view
	Note     string `db:"note" json:"note"`
	Source   Origin `json:"ban_source" db:"ban_source"`
	ReportId int    `json:"report_id"`
	// Deleted is used for soft-deletes
	Deleted bool `json:"deleted" db:"deleted"`
	// ValidUntil is when the ban will be no longer valid. 0 denotes forever
	ValidUntil time.Time `json:"valid_until" db:"valid_until"`
	CreatedOn  time.Time `db:"created_on" json:"created_on"`
	UpdatedOn  time.Time `db:"updated_on" json:"updated_on"`
}

func (b Ban) String() string {
	return fmt.Sprintf("SID: %d Origin: %s Reason: %s Type: %v", b.SteamID.Int64(), b.Source, b.ReasonText, b.BanType)
}

type BannedPerson struct {
	Ban    Ban    `json:"ban"`
	Person Person `json:"person"`
}

func NewBannedPerson() BannedPerson {
	return BannedPerson{
		Ban: Ban{
			CreatedOn: config.Now(),
			UpdatedOn: config.Now(),
		},
		Person: Person{
			CreatedOn:     config.Now(),
			UpdatedOn:     config.Now(),
			PlayerSummary: &steamweb.PlayerSummary{},
		},
	}
}

type UserMessage struct {
	ParentId  int64         `json:"parent_id"`
	MessageId int64         `json:"message_id"`
	AuthorId  steamid.SID64 `json:"author_id,string"`
	Message   string        `json:"contents"`
	Deleted   bool          `json:"deleted"`
	CreatedOn time.Time     `json:"created_on"`
	UpdatedOn time.Time     `json:"updated_on"`
}

func NewUserMessage(parentId int64, authorId steamid.SID64, message string) UserMessage {
	return UserMessage{
		ParentId:  parentId,
		AuthorId:  authorId,
		Message:   message,
		CreatedOn: config.Now(),
		UpdatedOn: config.Now(),
	}
}
