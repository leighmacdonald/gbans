package domain

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/leighmacdonald/gbans/internal/avatar"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type SourceTarget struct {
	SourcePersonaname string `json:"source_personaname"`
	SourceAvatarhash  string `json:"source_avatarhash"`
	TargetPersonaname string `json:"target_personaname"`
	TargetAvatarhash  string `json:"target_avatarhash"`
}

type BannedCIDRPerson struct {
	BanCIDR
	SourceTarget
}

type BannedSteamPerson struct {
	BanSteam
	SourceTarget
}

func (b BannedSteamPerson) GetName() string {
	// TODO implement me
	panic("implement me")
}

func (b BannedSteamPerson) GetAvatar() avatar.AvatarLinks {
	// TODO implement me
	panic("implement me")
}

func (b BannedSteamPerson) GetSteamID() steamid.SID64 {
	// TODO implement me
	panic("implement me")
}

type BannedGroupPerson struct {
	BanGroup
	SourceTarget
}

type BannedASNPerson struct {
	BanASN
	SourceTarget
}

func NewBannedPerson() BannedSteamPerson {
	banTime := time.Now()

	return BannedSteamPerson{
		BanSteam: BanSteam{
			BanBase: BanBase{
				CreatedOn: banTime,
				UpdatedOn: banTime,
			},
		},
		SourceTarget: SourceTarget{},
	}
}

// BanBase provides a common struct shared between all ban types, it should not be used
// directly.
type BanBase struct {
	// SteamID is the steamID of the banned person
	TargetID steamid.SID64 `json:"target_id"`
	SourceID steamid.SID64 `json:"source_id"`
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
	banTime := time.Now()
	banBase.BanType = opts.BanType
	banBase.SourceID = opts.SourceID
	banBase.TargetID = opts.TargetID
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

// BaseBanOpts defines common ban options that apply to all types to varying degrees
// It should not be instantiated directly, but instead use one of the composites that build
// upon it.
type BaseBanOpts struct {
	TargetID    steamid.SID64 `json:"target_id"`
	SourceID    steamid.SID64 `json:"source_id"`
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
	BanID       int64 `json:"ban_id"`
	ReportID    int64 `json:"report_id"`
}

type BanSteamGroupOpts struct {
	BaseBanOpts
	GroupID   steamid.GID
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

// BanGroup represents a steam group whose members are banned from connecting.
type BanGroup struct {
	BanBase
	SourceTarget
	BanGroupID      int64       `json:"ban_group_id"`
	GroupID         steamid.GID `json:"group_id"`
	GroupName       string      `json:"group_name"`
	CommunityBanned bool        `json:"community_banned"`
	VacBans         int         `json:"vac_bans"`
	GameBans        int         `json:"game_bans"`
}

func (banGroup *BanGroup) Apply(opts BanSteamGroupOpts) error {
	banGroup.ApplyBaseOpts(opts.BaseBanOpts)
	banGroup.GroupName = opts.GroupName
	banGroup.GroupID = opts.GroupID

	return nil
}

type BanASN struct {
	BanBase
	BanASNId        int64 `json:"ban_asn_id"`
	ASNum           int64 `json:"as_num"`
	CommunityBanned bool  `json:"community_banned"`
	VacBans         int   `json:"vac_bans"`
	GameBans        int   `json:"game_bans"`
}

func (banASN *BanASN) Apply(opts BanASNOpts) error {
	banASN.ApplyBaseOpts(opts.BaseBanOpts)
	banASN.ASNum = opts.ASNum

	return nil
}

type BanCIDR struct {
	BanBase
	NetID           int64  `json:"net_id"`
	CIDR            string `json:"cidr"`
	CommunityBanned bool   `json:"community_banned"`
	VacBans         int    `json:"vac_bans"`
	GameBans        int    `json:"game_bans"`
}

func (banCIDR *BanCIDR) Apply(opts BanCIDROpts) error {
	banCIDR.ApplyBaseOpts(opts.BaseBanOpts)
	banCIDR.CIDR = opts.CIDR.String()

	return nil
}

func (banCIDR *BanCIDR) String() string {
	return fmt.Sprintf("Net: %s Origin: %s Reason: %s", banCIDR.CIDR, banCIDR.Origin, banCIDR.Reason)
}

type BanSteam struct {
	BanBase
	BanID           int64  `db:"ban_id" json:"ban_id"`
	ReportID        int64  `json:"report_id"`
	IncludeFriends  bool   `json:"include_friends"`
	CommunityBanned bool   `json:"community_banned"`
	VacBans         int    `json:"vac_bans"`
	GameBans        int    `json:"game_bans"`
	LastIP          net.IP `json:"last_ip"`
}

//goland:noinspection ALL
func (banSteam *BanSteam) Apply(opts BanSteamOpts) {
	banSteam.ApplyBaseOpts(opts.BaseBanOpts)
	banSteam.ReportID = opts.ReportID
}

//goland:noinspection ALL
func (banSteam BanSteam) Path() string {
	return fmt.Sprintf("/ban/%d", banSteam.BanID)
}

//goland:noinspection ALL
func (banSteam BanSteam) String() string {
	return fmt.Sprintf("SID: %d Origin: %s Reason: %s Type: %v", banSteam.TargetID.Int64(), banSteam.Origin, banSteam.ReasonText, banSteam.BanType)
}

func newBaseBanOpts(ctx context.Context, source SteamIDProvider, target StringSID, duration time.Duration,
	reason Reason, reasonText string, modNote string, origin Origin,
	banType BanType, opts *BaseBanOpts,
) error {
	sourceSid, errSource := source.SID64(ctx)
	if errSource != nil {
		return errors.Join(errSource, errs.ErrSourceID)
	}

	targetSid := steamid.New(0)

	if string(target) != "0" {
		newTargetSid, errTargetSid := target.SID64(ctx)
		if errTargetSid != nil {
			return errs.ErrTargetID
		}

		targetSid = newTargetSid
	}

	if !(banType == Banned || banType == NoComm) {
		return ErrInvalidBanType
	}

	if duration <= 0 {
		return ErrInvalidBanDuration
	}

	if reason == Custom && reasonText == "" {
		return ErrInvalidBanReason
	}

	opts.TargetID = targetSid
	opts.SourceID = sourceSid
	opts.Duration = duration
	opts.ModNote = modNote
	opts.Reason = reason
	opts.ReasonText = reasonText
	opts.Origin = origin
	opts.Deleted = false
	opts.BanType = banType
	opts.IsEnabled = true

	return nil
}

func NewBanSteam(ctx context.Context, source SteamIDProvider, target StringSID, duration time.Duration,
	reason Reason, reasonText string, modNote string, origin Origin, reportID int64, banType BanType,
	includeFriends bool, banSteam *BanSteam,
) error {
	var opts BanSteamOpts

	errBaseOpts := newBaseBanOpts(ctx, source, target, duration, reason, reasonText, modNote, origin, banType, &opts.BaseBanOpts)
	if errBaseOpts != nil {
		return errBaseOpts
	}

	if reportID < 0 {
		return ErrInvalidReportID
	}

	opts.ReportID = reportID
	banSteam.Apply(opts)
	banSteam.ReportID = opts.ReportID
	banSteam.BanID = opts.BanID
	banSteam.IncludeFriends = includeFriends

	return nil
}

func NewBanASN(ctx context.Context, source SteamIDProvider, target StringSID, duration time.Duration,
	reason Reason, reasonText string, modNote string, origin Origin, asNum int64, banType BanType, banASN *BanASN,
) error {
	var opts BanASNOpts

	errBaseOpts := newBaseBanOpts(ctx, source, target, duration, reason, reasonText, modNote, origin, banType, &opts.BaseBanOpts)
	if errBaseOpts != nil {
		return errBaseOpts
	}

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
		if asNum >= r.start && asNum <= r.end {
			inRange = true

			break
		}
	}

	if !inRange {
		return ErrInvalidASN
	}

	opts.ASNum = asNum

	return banASN.Apply(opts)
}

func NewBanCIDR(ctx context.Context, source SteamIDProvider, target StringSID, duration time.Duration,
	reason Reason, reasonText string, modNote string, origin Origin, cidr string,
	banType BanType, banCIDR *BanCIDR,
) error {
	var opts BanCIDROpts
	if errBaseOpts := newBaseBanOpts(ctx, source, target, duration, reason, reasonText, modNote, origin,
		banType, &opts.BaseBanOpts); errBaseOpts != nil {
		return errBaseOpts
	}

	_, parsedNetwork, errParse := net.ParseCIDR(cidr)
	if errParse != nil {
		return errors.Join(errParse, ErrInvalidCIDR)
	}

	opts.CIDR = parsedNetwork

	return banCIDR.Apply(opts)
}

func NewBanSteamGroup(ctx context.Context, source SteamIDProvider, target StringSID, duration time.Duration,
	modNote string, origin Origin, groupID steamid.GID, groupName string,
	banType BanType, banGroup *BanGroup,
) error {
	var opts BanSteamGroupOpts

	errBaseOpts := newBaseBanOpts(ctx, source, target, duration, Custom, "Group Ban", modNote, origin, banType, &opts.BaseBanOpts)
	if errBaseOpts != nil {
		return errBaseOpts
	}

	// TODO validate gid here w/fetch?
	opts.GroupID = groupID
	opts.GroupName = groupName

	return banGroup.Apply(opts)
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

type AppealOverview struct {
	BanSteam

	SourcePersonaname string `json:"source_personaname"`
	SourceAvatarhash  string `json:"source_avatarhash"`
	TargetPersonaname string `json:"target_personaname"`
	TargetAvatarhash  string `json:"target_avatarhash"`
}

type BanAppealMessage struct {
	BanID        int64         `json:"ban_id"`
	BanMessageID int64         `json:"ban_message_id"`
	AuthorID     steamid.SID64 `json:"author_id"`
	MessageMD    string        `json:"message_md"`
	Deleted      bool          `json:"deleted"`
	CreatedOn    time.Time     `json:"created_on"`
	UpdatedOn    time.Time     `json:"updated_on"`
	SimplePerson
}

func NewBanAppealMessage(banID int64, authorID steamid.SID64, message string) BanAppealMessage {
	return BanAppealMessage{
		BanID:     banID,
		AuthorID:  authorID,
		MessageMD: message,
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	}
}
