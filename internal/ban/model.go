package ban

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/pkg/datetime"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrInvalidBanType     = errors.New("invalid ban type")
	ErrInvalidBanDuration = errors.New("invalid ban duration")
	ErrInvalidBanReason   = errors.New("custom reason cannot be empty")
	ErrInvalidASN         = errors.New("invalid asn, out of range")
	ErrInvalidCIDR        = errors.New("failed to parse CIDR address")
	ErrInvalidReportID    = errors.New("invalid report id")
	ErrGetBan             = errors.New("failed to load existing ban")
	ErrSaveBan            = errors.New("failed to save ban")
	ErrReportStateUpdate  = errors.New("failed to update report state")
	ErrFetchPerson        = errors.New("failed to fetch/create person")
)

type NewBanMessage struct {
	Message string `json:"message"`
}

type AppealQueryFilter struct {
	Deleted bool `json:"deleted"`
}

type RequestUnban struct {
	UnbanReasonText string `json:"unban_reason_text"`
}

type BanAppealMessage struct {
	BanID           int64                `json:"ban_id"`
	BanMessageID    int64                `json:"ban_message_id"`
	AuthorID        steamid.SteamID      `json:"author_id"`
	MessageMD       string               `json:"message_md"`
	Deleted         bool                 `json:"deleted"`
	CreatedOn       time.Time            `json:"created_on"`
	UpdatedOn       time.Time            `json:"updated_on"`
	Avatarhash      string               `json:"avatarhash"`
	Personaname     string               `json:"personaname"`
	PermissionLevel permission.Privilege `json:"permission_level"`
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

// BanOpts defines common ban options that apply to all types to varying degrees
// It should not be instantiated directly, but instead use one of the composites that build
// upon it.
type BanOpts struct {
	TargetID       steamid.SteamID `json:"target_id"`
	SourceID       steamid.SteamID `json:"source_id"`
	Duration       time.Duration   `json:"duration"`
	BanType        ban.BanType     `json:"ban_type"`
	Reason         ban.Reason      `json:"reason"`
	ReasonText     string          `json:"reason_text"`
	Origin         ban.Origin      `json:"origin"`
	ModNote        string          `json:"mod_note"`
	IsEnabled      bool            `json:"is_enabled"`
	Deleted        bool            `json:"deleted"`
	AppealState    AppealState     `json:"appeal_state"`
	ReportID       int64           `json:"report_id"`
	ASNum          int64           `json:"as_num"`
	CIDR           string          `json:"cidr"`
	EvadeOk        bool            `json:"evade_ok"`
	LastIP         string          `json:"last_ip"`
	Name           string          `json:"name"`
	DemoName       string          `json:"demo_name"`
	DemoTick       int             `json:"demo_tick"`
	IncludeFriends bool            `json:"include_friends"`
	Note           string          `json:"note"`
}

func (opts *BanOpts) SetDuration(durString string) error {
	duration, errDuration := datetime.CalcDuration(durString, opts.ValidUntil)
	if errDuration != nil {
		return errDuration
	}
	opts.Duration = duration
	return nil
}

func (opts *BanOpts) Validate() error {
	if !opts.SourceID.Valid() || !opts.TargetID.Valid() {
		return domain.ErrInvalidSID
	}

	if opts.BanType != ban.Banned && opts.BanType != ban.NoComm {
		return ErrInvalidBanType
	}

	if opts.Duration <= 0 {
		return ErrInvalidBanDuration
	}

	if opts.Reason == ban.Custom && len(opts.ReasonText) < 3 {
		return fmt.Errorf("%w: Custom reason must be at least 3 characters", ErrBanOptsInvalid)
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

// BanBase provides a common struct shared between all ban types, it should not be used
// directly.
type Ban struct {
	// SteamID is the steamID of the banned person
	TargetID       steamid.SteamID `json:"target_id"`
	SourceID       steamid.SteamID `json:"source_id"`
	BanID          int64           `json:"ban_id"`
	ReportID       int64           `json:"report_id"`
	IncludeFriends bool            `json:"include_friends"`
	LastIP         string          `json:"last_ip"`
	EvadeOk        bool            `json:"evade_ok"`

	// Reason defines the overall ban classification
	BanType ban.BanType `json:"ban_type"`
	// Reason defines the overall ban classification
	Reason ban.Reason `json:"reason"`
	// ReasonText is returned to the client when kicked trying to join the server
	ReasonText      string `json:"reason_text"`
	UnbanReasonText string `json:"unban_reason_text"`
	// Note is a supplementary note added by admins that is hidden from normal view
	Note        string      `json:"note"`
	Origin      ban.Origin  `json:"origin"`
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

type QueryOpts struct {
	SourceID steamid.SteamID
	// TargetID can represent a SteamID or a group ID. They both use steamID formats, just in a different numberspace
	TargetID      steamid.SteamID
	GroupsOnly    bool
	BanID         int64
	Deleted       bool
	EvadeOk       bool
	Personaname   string
	CIDR          string
	IncludeGroups bool
	ASNum         bool
	LatestOnly    bool
}

//goland:noinspection ALL
func (banSteam Ban) Path() string {
	return fmt.Sprintf("/ban/%d", banSteam.BanID)
}

//goland:noinspection ALL
func (banSteam Ban) String() string {
	return fmt.Sprintf("SID: %d Origin: %s Reason: %s Type: %v", banSteam.TargetID.Int64(), banSteam.Origin, banSteam.ReasonText, banSteam.BanType)
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

type Stats struct {
	BansTotal     int `json:"bans_total"`
	BansDay       int `json:"bans_day"`
	BansWeek      int `json:"bans_week"`
	BansMonth     int `json:"bans_month"`
	Bans3Month    int `json:"bans3_month"`
	Bans6Month    int `json:"bans6_month"`
	BansYear      int `json:"bans_year"`
	BansCIDRTotal int `json:"bans_cidr_total"`
	AppealsOpen   int `json:"appeals_open"`
	AppealsClosed int `json:"appeals_closed"`
	FilteredWords int `json:"filtered_words"`
	ServersAlive  int `json:"servers_alive"`
	ServersTotal  int `json:"servers_total"`
}
