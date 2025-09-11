package ban

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrInvalidBanType     = errors.New("invalid ban type")
	ErrInvalidBanDuration = errors.New("invalid ban duration")
	ErrInvalidBanReason   = errors.New("custom reason cannot be empty")
	ErrInvalidASN         = errors.New("invalid asn, out of range")
	ErrInvalidCIDR        = errors.New("failed to parse CIDR address")
	ErrInvalidReportID    = errors.New("invalid report id")
)

type RequestUnban struct {
	UnbanReasonText string `json:"unban_reason_text"`
}

type BanAppealMessage struct {
	BanID        int64           `json:"ban_id"`
	BanMessageID int64           `json:"ban_message_id"`
	AuthorID     steamid.SteamID `json:"author_id"`
	MessageMD    string          `json:"message_md"`
	Deleted      bool            `json:"deleted"`
	CreatedOn    time.Time       `json:"created_on"`
	UpdatedOn    time.Time       `json:"updated_on"`
	person.SimplePerson
}

func NewBanAppealMessage(banID int64, authorID steamid.SteamID, message string) BanAppealMessage {
	return BanAppealMessage{
		BanID:     banID,
		AuthorID:  authorID,
		MessageMD: message,
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	}
}

type AppealOverview struct {
	Ban

	SourcePersonaname string `json:"source_personaname"`
	SourceAvatarhash  string `json:"source_avatarhash"`
	TargetPersonaname string `json:"target_personaname"`
	TargetAvatarhash  string `json:"target_avatarhash"`
}

type QueryOpts struct {
	SourceID   steamid.SteamID
	TargetID   steamid.SteamID
	GroupsOnly bool
	BanID      int64
	Deleted    bool
	EvadeOk    bool
}

type BannedPerson struct {
	Ban
	domain.SourceTarget
}

// BanType defines the state of the ban for a user, 0 being no ban.
type BanType int

const (
	// Unknown means the ban state could not be determined, failing-open to allowing players
	// to connect.
	Unknown BanType = iota - 1
	// OK Ban state is clean.
	OK //nolint:varnamelen
	// NoComm means the player cannot communicate while playing voice + chat.
	NoComm
	// Banned means the player cannot join the server at all.
	Banned
	// Network is used when a client connected from a banned CIDR block.
	Network
)

func (bt BanType) String() string {
	switch bt {
	case Network:
		return "network"
	case Unknown:
		return "unknown"
	case NoComm:
		return "mute/gag"
	case Banned:
		return "banned"
	case OK:
		fallthrough
	default:
		return ""
	}
}

// Reason defined a set of predefined ban reasons.
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
	Evading
	Username
)

func (r Reason) String() string {
	return map[Reason]string{
		Custom:           "Custom",
		External:         "3rd party",
		Cheating:         "Cheating",
		Racism:           "Racism",
		Harassment:       "Personal Harassment",
		Exploiting:       "Exploiting",
		WarningsExceeded: "Warnings Exceeded",
		Spam:             "Spam",
		Language:         "Language",
		Profile:          "Profile",
		ItemDescriptions: "Item Name or Descriptions",
		BotHost:          "BotHost",
		Evading:          "Evading",
		Username:         "Inappropriate Username",
	}[r]
}

// Origin defines the origin of the ban or action.
type Origin int

const (
	// System is an automatic ban triggered by the service.
	System Origin = iota
	// Bot is a ban using the discord bot interface.
	Bot
	// Web is a ban using the web-ui.
	Web
	// InGame is a ban using the sourcemod plugin.
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

type AppealState int

const (
	AnyState AppealState = iota - 1
	Open
	Denied
	Accepted
	Reduced
	NoAppeal
)

func (as AppealState) String() string {
	switch as {
	case Denied:
		return "Denied"
	case Accepted:
		return "Accepted"
	case Reduced:
		return "Reduced"
	case NoAppeal:
		return "No Appeal"
	case AnyState:
		fallthrough
	case Open:
		fallthrough
	default:
		return "Open"
	}
}

type BanOpts struct {
	SourceID       steamid.SteamID
	TargetID       steamid.SteamID
	Duration       string    `json:"duration"`
	ValidUntil     time.Time `json:"valid_until"`
	BanType        BanType   `json:"ban_type"`
	Reason         Reason    `json:"reason"`
	ReasonText     string    `json:"reason_text"`
	Note           string    `json:"note"`
	ReportID       int64     `json:"report_id"`
	DemoName       string    `json:"demo_name"`
	DemoTick       int       `json:"demo_tick"`
	IncludeFriends bool      `json:"include_friends"`
	EvadeOk        bool      `json:"evade_ok"`
}

// BanBase provides a common struct shared between all ban types, it should not be used
// directly.
type Ban struct {
	// SteamID is the steamID of the banned person
	TargetID steamid.SteamID `json:"target_id"`
	SourceID steamid.SteamID `json:"source_id"`

	BanID           int64 `json:"ban_id"`
	ReportID        int64 `json:"report_id"`
	IncludeFriends  bool  `json:"include_friends"`
	CommunityBanned bool  `json:"community_banned"`

	LastIP  string `json:"last_ip"`
	EvadeOk bool   `json:"evade_ok"`

	// Reason defines the overall ban classification
	BanType BanType `json:"ban_type"`
	// Reason defines the overall ban classification
	Reason Reason `json:"reason"`
	// ReasonText is returned to the client when kicked trying to join the server
	ReasonText      string `json:"reason_text"`
	UnbanReasonText string `json:"unban_reason_text"`
	// Note is a supplementary note added by admins that is hidden from normal view
	Note        string      `json:"note"`
	Origin      Origin      `json:"origin"`
	CIDR        string      `json:"cidr"`
	ASNum       int64       `json:"as_num"`
	Name        string      `json:"name"`
	AppealState AppealState `json:"appeal_state"`

	// Deleted is used for soft-deletes
	Deleted   bool `json:"deleted"`
	IsEnabled bool `json:"is_enabled"`
	// ValidUntil is when the ban will be no longer valid. 0 denotes forever
	ValidUntil time.Time `json:"valid_until" `
	CreatedOn  time.Time `json:"created_on"`
	UpdatedOn  time.Time `json:"updated_on"`
}

// Opts defines common ban options that apply to all types to varying degrees
// It should not be instantiated directly, but instead use one of the composites that build
// upon it.
type Opts struct {
	TargetID    steamid.SteamID `json:"target_id"`
	SourceID    steamid.SteamID `json:"source_id"`
	Duration    time.Duration   `json:"duration"`
	BanType     BanType         `json:"ban_type"`
	Reason      Reason          `json:"reason"`
	ReasonText  string          `json:"reason_text"`
	Origin      Origin          `json:"origin"`
	ModNote     string          `json:"mod_note"`
	IsEnabled   bool            `json:"is_enabled"`
	Deleted     bool            `json:"deleted"`
	AppealState AppealState     `json:"appeal_state"`
	ReportID    int64           `json:"report_id"`
	ASNum       int64           `json:"as_num"`
	CIDR        string          `json:"cidr"`
	EvadeOk     bool            `json:"evade_ok"`
	LastIP      string          `json:"last_ip"`
	Name        string          `json:"name"`
}

func (opts Opts) Validate() error {
	if !opts.SourceID.Valid() || !opts.TargetID.Valid() {
		return domain.ErrInvalidSID
	}

	if opts.BanType != Banned && opts.BanType != NoComm {
		return ErrInvalidBanType
	}

	if opts.Duration <= 0 {
		return ErrInvalidBanDuration
	}

	if opts.Reason == Custom && opts.ReasonText == "" {
		return ErrInvalidBanReason
	}

	if opts.ReportID < 0 {
		return ErrInvalidReportID
	}

	if opts.ASNum > 0 {
		// Valid public ASN ranges
		// https://www.iana.org/assignments/as-numbers/as-numbers.xhtml
		ranges := []struct {
			start int64
			end   int64
		}{
			{1, 23455},
			{23457, 64495},
			{131072, 4199999999},
		}

		inRange := false

		for _, r := range ranges {
			if opts.ASNum >= r.start && opts.ASNum <= r.end {
				inRange = true

				break
			}
		}

		if !inRange {
			return ErrInvalidASN
		}
	}

	_, _, errParse := net.ParseCIDR(opts.CIDR)
	if errParse != nil {
		return errors.Join(errParse, ErrInvalidCIDR)
	}

	return nil
}

//goland:noinspection ALL
func (banSteam Ban) Path() string {
	return fmt.Sprintf("/ban/%d", banSteam.BanID)
}

//goland:noinspection ALL
func (banSteam Ban) String() string {
	return fmt.Sprintf("SID: %d Origin: %s Reason: %s Type: %v", banSteam.TargetID.Int64(), banSteam.Origin, banSteam.ReasonText, banSteam.BanType)
}

func NewBan(opts Opts) (Ban, error) {
	if err := opts.Validate(); err != nil {
		return Ban{}, err
	}

	ban := Ban{
		TargetID:        opts.TargetID,
		SourceID:        opts.SourceID,
		ReportID:        opts.ReportID,
		ASNum:           opts.ASNum,
		BanType:         opts.BanType,
		IncludeFriends:  false,
		CommunityBanned: false,
		LastIP:          opts.LastIP,
		EvadeOk:         opts.EvadeOk,
		Reason:          opts.Reason,
		ReasonText:      opts.ReasonText,
		Note:            opts.ModNote,
		Origin:          opts.Origin,
		Name:            opts.Name,
		CIDR:            opts.CIDR,
		ValidUntil:      time.Now().Add(opts.Duration),
		AppealState:     opts.AppealState,
		Deleted:         opts.Deleted,
		IsEnabled:       opts.IsEnabled,
		CreatedOn:       time.Now(),
		UpdatedOn:       time.Now(),
	}

	return ban, nil
}

type MembersList struct {
	MembersID int64
	ParentID  int64
	Members   steamid.Collection
	CreatedOn time.Time
	UpdatedOn time.Time
}

func NewMembersList(parentID int64, members steamid.Collection) MembersList {
	now := time.Now()

	return MembersList{
		ParentID:  parentID,
		Members:   members,
		CreatedOn: now,
		UpdatedOn: now,
	}
}
