package model

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
)

type SteamIDProvider interface {
	SID64() (steamid.SID64, error)
}

// StringSID defines a user provided steam id in an unknown format
type StringSID string

func (t StringSID) SID64() (steamid.SID64, error) {
	// TODO pass ctx, or remove resolve?
	resolveCtx, cancelResolve := context.WithTimeout(context.Background(), time.Second*5)
	defer cancelResolve()
	// TODO cache this as it can be a huge hot path
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
// A Duration of 0 will be interpreted as permanent and set to 10 years in the future
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
	Unknown BanType = iota - 1
	// OK Ban state is clean
	OK
	// NoComm means the player cannot communicate while playing voice + chat
	NoComm
	// Banned means the player cannot join the server at all
	Banned
)

// Origin defines the origin of the ban or action
type Origin int

const (
	// System is an automatic ban triggered by the service
	System Origin = iota
	// Bot is a ban using the discord bot interface
	Bot
	// Web is a ban using the web-ui
	Web
	// InGame is a ban using the sourcemod plugin
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

// Reason defined a set of predefined ban reasons
// TODO make this fully dynamic?
type Reason int

const (
	Custom Reason = iota + 1
	External
	Cheating
	Racism
	Harassment
	Exploiting
	WarningsExceeded
	Spam
	Language
	Profile
	ItemDescriptions
	BotHost
)

var reasonStr = map[Reason]string{
	Custom:           "Custom",
	External:         "3rd party",
	Cheating:         "Cheating",
	Racism:           "Racism",
	Harassment:       "Person Harassment",
	Exploiting:       "Exploiting",
	WarningsExceeded: "Warnings Exceeded",
	Spam:             "Spam",
	Language:         "Language",
	Profile:          "Profile",
	ItemDescriptions: "Item Name or Descriptions",
	BotHost:          "BotHost",
}

func (r Reason) String() string {
	return reasonStr[r]
}

// BaseBanOpts defines common ban options that apply to all types to varying degrees
// It should not be instantiated directly, but instead use one of the composites that build
// upon it
type BaseBanOpts struct {
	TargetId    steamid.SID64 `json:"target_id"`
	SourceId    steamid.SID64 `json:"source_id"`
	Duration    time.Duration `json:"duration"`
	BanType     BanType       `json:"ban_type"`
	Reason      Reason        `json:"reason"`
	ReasonText  string        `json:"reason_text"`
	Origin      Origin        `json:"origin"`
	ModNote     string        `json:"mod_note"`
	IsEnabled   bool          `json:"is_enabled"`
	Deleted     bool          `json:"deleted"`
	AppealState AppealState   `json:"appeal_state"`
}

type BanSteamOpts struct {
	BaseBanOpts `json:"base_ban_opts"`
	BanId       int64 `json:"ban_id"`
	ReportId    int64 `json:"report_id"`
}

type BanSteamGroupOpts struct {
	BaseBanOpts
	GroupId   steamid.GID
	GroupName string
}

type BanASNOpts struct {
	BaseBanOpts
	ASNum int64
}

type BanCIDROpts struct {
	BaseBanOpts
	CIDR *net.IPNet
}

// BanGroup represents a steam group whose members are banned from connecting
type BanGroup struct {
	BanBase
	BanGroupId int64       `json:"ban_group_id"`
	GroupId    steamid.GID `json:"group_id,string"`
	GroupName  string      `json:"group_name"`
}

func (banGroup *BanGroup) Apply(opts BanSteamGroupOpts) error {
	banGroup.ApplyBaseOpts(opts.BaseBanOpts)
	banGroup.GroupName = opts.GroupName
	banGroup.GroupId = opts.GroupId
	return nil
}

type BanASN struct {
	BanBase
	BanASNId int64 `json:"ban_asn_id"`
	ASNum    int64 `json:"as_num"`
}

func (banASN *BanASN) Apply(opts BanASNOpts) error {
	banASN.ApplyBaseOpts(opts.BaseBanOpts)
	banASN.ASNum = opts.ASNum
	return nil
}

type BanCIDR struct {
	BanBase
	NetID int64      `json:"net_id"`
	CIDR  *net.IPNet `json:"cidr"`
}

func (banCIDR *BanCIDR) Apply(opts BanCIDROpts) error {
	banCIDR.ApplyBaseOpts(opts.BaseBanOpts)
	banCIDR.CIDR = opts.CIDR
	return nil
}

func (banCIDR *BanCIDR) String() string {
	return fmt.Sprintf("Net: %s Origin: %s Reason: %s", banCIDR.CIDR, banCIDR.Origin, banCIDR.Reason)
}

type AppealState int

const (
	Open AppealState = iota
	Denied
	Accepted
	Reduced
	NoAppeal
)

// BanBase provides a common struct shared between all ban types, it should not be used
// directly
type BanBase struct {
	// SteamID is the steamID of the banned person
	TargetId steamid.SID64 `json:"target_id,string"`
	SourceId steamid.SID64 `json:"source_id,string"`
	// Reason defines the overall ban classification
	BanType BanType `json:"ban_type"`
	// Reason defines the overall ban classification
	Reason Reason `json:"reason"`
	// ReasonText is returned to the client when kicked trying to join the server
	ReasonText      string `json:"reason_text"`
	UnbanReasonText string `json:"unban_reason_text"`
	// Note is a supplementary note added by admins that is hidden from normal view
	Note   string `json:"note"`
	Origin Origin `json:"origin"`

	AppealState AppealState `json:"appeal_state"`

	// Deleted is used for soft-deletes
	Deleted   bool `json:"deleted"`
	IsEnabled bool `json:"is_enabled"`
	// ValidUntil is when the ban will be no longer valid. 0 denotes forever
	ValidUntil time.Time `json:"valid_until" `
	CreatedOn  time.Time `json:"created_on"`
	UpdatedOn  time.Time `json:"updated_on"`
}

func (banBase *BanBase) ApplyBaseOpts(opts BaseBanOpts) {
	banTime := config.Now()
	banBase.BanType = opts.BanType
	banBase.SourceId = opts.SourceId
	banBase.TargetId = opts.TargetId
	banBase.Reason = opts.Reason
	banBase.ReasonText = opts.ReasonText
	banBase.Note = opts.ModNote
	banBase.Origin = opts.Origin
	banBase.Deleted = opts.Deleted
	banBase.IsEnabled = opts.IsEnabled
	banBase.AppealState = opts.AppealState
	banBase.CreatedOn = banTime
	banBase.UpdatedOn = banTime
	banBase.ValidUntil = banTime.Add(opts.Duration)
}

type BanSteam struct {
	BanBase
	BanID    int64 `db:"ban_id" json:"ban_id"`
	ReportId int64 `json:"report_id"`
}

func (banSteam *BanSteam) Apply(opts BanSteamOpts) {
	banSteam.ApplyBaseOpts(opts.BaseBanOpts)
	banSteam.ReportId = opts.ReportId
}

func (banSteam BanSteam) ToURL() string {
	return config.ExtURL("/ban/%d", banSteam.BanID)
}

func (banSteam *BanSteam) String() string {
	return fmt.Sprintf("SID: %d Origin: %s Reason: %s Type: %v", banSteam.TargetId.Int64(), banSteam.Origin, banSteam.ReasonText, banSteam.BanType)
}

type BannedPerson struct {
	Ban    BanSteam `json:"ban"`
	Person Person   `json:"person"`
}

func NewBannedPerson() BannedPerson {
	banTime := config.Now()
	return BannedPerson{
		Ban: BanSteam{
			BanBase: BanBase{
				CreatedOn: banTime,
				UpdatedOn: banTime,
			},
		},
		Person: Person{
			CreatedOn:     banTime,
			UpdatedOn:     banTime,
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
