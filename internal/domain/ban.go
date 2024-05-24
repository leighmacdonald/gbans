package domain

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

type BanSteamRepository interface {
	Save(ctx context.Context, ban *BanSteam) error
	GetBySteamID(ctx context.Context, sid64 steamid.SteamID, deletedOk bool) (BannedSteamPerson, error)
	GetByBanID(ctx context.Context, banID int64, deletedOk bool) (BannedSteamPerson, error)
	GetByLastIP(ctx context.Context, lastIP netip.Addr, deletedOk bool) (BannedSteamPerson, error)
	Delete(ctx context.Context, ban *BanSteam, hardDelete bool) error
	Get(ctx context.Context, filter SteamBansQueryFilter) ([]BannedSteamPerson, error)
	ExpiredBans(ctx context.Context) ([]BanSteam, error)
	GetOlderThan(ctx context.Context, filter QueryFilter, since time.Time) ([]BanSteam, error)
	Stats(ctx context.Context, stats *Stats) error
	TruncateCache(ctx context.Context) error
	InsertCache(ctx context.Context, steamID steamid.SteamID, entries []int64) error
}

type BanSteamUsecase interface {
	IsOnIPWithBan(ctx context.Context, curUser PersonInfo, steamID steamid.SteamID, address netip.Addr) (bool, error)
	GetBySteamID(ctx context.Context, sid64 steamid.SteamID, deletedOk bool) (BannedSteamPerson, error)
	GetByBanID(ctx context.Context, banID int64, deletedOk bool) (BannedSteamPerson, error)
	GetByLastIP(ctx context.Context, lastIP netip.Addr, deletedOk bool) (BannedSteamPerson, error)
	Save(ctx context.Context, ban *BanSteam) error
	Ban(ctx context.Context, curUser PersonInfo, banSteam *BanSteam) error
	Unban(ctx context.Context, targetSID steamid.SteamID, reason string) (bool, error)
	Delete(ctx context.Context, ban *BanSteam, hardDelete bool) error
	Get(ctx context.Context, filter SteamBansQueryFilter) ([]BannedSteamPerson, error)
	Expired(ctx context.Context) ([]BanSteam, error)
	GetOlderThan(ctx context.Context, filter QueryFilter, since time.Time) ([]BanSteam, error)
	Stats(ctx context.Context, stats *Stats) error
	UpdateCache(ctx context.Context) error
}

type BanGroupRepository interface {
	Save(ctx context.Context, banGroup *BanGroup) error
	Ban(ctx context.Context, banGroup *BanGroup) error
	GetByGID(ctx context.Context, groupID steamid.SteamID, banGroup *BanGroup) error
	GetByID(ctx context.Context, banGroupID int64, banGroup *BanGroup) error
	Get(ctx context.Context, filter GroupBansQueryFilter) ([]BannedGroupPerson, error)
	GetMembersList(ctx context.Context, parentID int64, list *MembersList) error
	SaveMembersList(ctx context.Context, list *MembersList) error
	Delete(ctx context.Context, banGroup *BanGroup) error
	TruncateCache(ctx context.Context) error
	InsertCache(ctx context.Context, groupID steamid.SteamID, entries []int64) error
}

type BanGroupUsecase interface {
	Save(ctx context.Context, banGroup *BanGroup) error
	Ban(ctx context.Context, banGroup *BanGroup) error
	GetByGID(ctx context.Context, groupID steamid.SteamID, banGroup *BanGroup) error
	GetByID(ctx context.Context, banGroupID int64, banGroup *BanGroup) error
	Get(ctx context.Context, filter GroupBansQueryFilter) ([]BannedGroupPerson, error)
	GetMembersList(ctx context.Context, parentID int64, list *MembersList) error
	SaveMembersList(ctx context.Context, list *MembersList) error
	Delete(ctx context.Context, banGroup *BanGroup) error
	UpdateCache(ctx context.Context) error
}

type BanNetRepository interface {
	GetByAddress(ctx context.Context, ipAddr netip.Addr) ([]BanCIDR, error)
	GetByID(ctx context.Context, netID int64, banNet *BanCIDR) error
	Get(ctx context.Context, filter CIDRBansQueryFilter) ([]BannedCIDRPerson, error)
	Save(ctx context.Context, banNet *BanCIDR) error
	Delete(ctx context.Context, banNet *BanCIDR) error
	Expired(ctx context.Context) ([]BanCIDR, error)
}

type BanNetUsecase interface {
	Ban(ctx context.Context, banNet *BanCIDR) error
	GetByAddress(ctx context.Context, ipAddr netip.Addr) ([]BanCIDR, error)
	GetByID(ctx context.Context, netID int64, banNet *BanCIDR) error
	Get(ctx context.Context, filter CIDRBansQueryFilter) ([]BannedCIDRPerson, error)
	Save(ctx context.Context, banNet *BanCIDR) error
	Delete(ctx context.Context, banNet *BanCIDR) error
	Expired(ctx context.Context) ([]BanCIDR, error)
}

type BanASNRepository interface {
	Save(ctx context.Context, banASN *BanASN) error
	GetByASN(ctx context.Context, asNum int64, banASN *BanASN) error
	GetByID(ctx context.Context, banID int64, banASN *BanASN) error
	Get(ctx context.Context, filter ASNBansQueryFilter) ([]BannedASNPerson, error)
	Delete(ctx context.Context, banASN *BanASN) error
	Expired(ctx context.Context) ([]BanASN, error)
}

type BanASNUsecase interface {
	Ban(ctx context.Context, banASN *BanASN) error
	GetByASN(ctx context.Context, asNum int64, banASN *BanASN) error
	GetByID(ctx context.Context, banID int64, banASN *BanASN) error
	Get(ctx context.Context, filter ASNBansQueryFilter) ([]BannedASNPerson, error)
	Save(ctx context.Context, banASN *BanASN) error
	Delete(ctx context.Context, banASN *BanASN) error
	Expired(ctx context.Context) ([]BanASN, error)
	Unban(ctx context.Context, asnNum string) (bool, error)
}

type UnbanRequest struct {
	UnbanReasonText string `json:"unban_reason_text"`
}

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

func (b BannedSteamPerson) GetAvatar() AvatarLinks {
	// TODO implement me
	panic("implement me")
}

func (b BannedSteamPerson) GetSteamID() steamid.SteamID {
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
	TargetID steamid.SteamID `json:"target_id"`
	SourceID steamid.SteamID `json:"source_id"`
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
}

type BanSteamOpts struct {
	BaseBanOpts `json:"base_ban_opts"`
	BanID       int64 `json:"ban_id"`
	ReportID    int64 `json:"report_id"`
}

type BanSteamGroupOpts struct {
	BaseBanOpts
	GroupID   steamid.SteamID
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
	BanGroupID      int64           `json:"ban_group_id"`
	GroupID         steamid.SteamID `json:"group_id"`
	GroupName       string          `json:"group_name"`
	CommunityBanned bool            `json:"community_banned"`
	VacBans         int             `json:"vac_bans"`
	GameBans        int             `json:"game_bans"`
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
	EvadeOk         bool   `json:"evade_ok"`
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

func newBaseBanOpts(source steamid.SteamID, target steamid.SteamID, duration time.Duration,
	reason Reason, reasonText string, modNote string, origin Origin,
	banType BanType, opts *BaseBanOpts,
) error {
	if !source.Valid() || !target.Valid() {
		return ErrInvalidSID
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

	opts.TargetID = target
	opts.SourceID = source
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

func NewBanSteam(source steamid.SteamID, target steamid.SteamID, duration time.Duration,
	reason Reason, reasonText string, modNote string, origin Origin, reportID int64, banType BanType,
	includeFriends bool, evadeOk bool, banSteam *BanSteam,
) error {
	var opts BanSteamOpts

	errBaseOpts := newBaseBanOpts(source, target, duration, reason, reasonText, modNote, origin, banType, &opts.BaseBanOpts)
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
	banSteam.EvadeOk = evadeOk

	return nil
}

func NewBanASN(source steamid.SteamID, target steamid.SteamID, duration time.Duration,
	reason Reason, reasonText string, modNote string, origin Origin, asNum int64, banType BanType, banASN *BanASN,
) error {
	var opts BanASNOpts

	errBaseOpts := newBaseBanOpts(source, target, duration, reason, reasonText, modNote, origin, banType, &opts.BaseBanOpts)
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

func NewBanCIDR(source steamid.SteamID, target steamid.SteamID, duration time.Duration,
	reason Reason, reasonText string, modNote string, origin Origin, cidr string,
	banType BanType, banCIDR *BanCIDR,
) error {
	var opts BanCIDROpts
	if errBaseOpts := newBaseBanOpts(source, target, duration, reason, reasonText, modNote, origin,
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

func NewBanSteamGroup(source steamid.SteamID, target steamid.SteamID, duration time.Duration,
	modNote string, origin Origin, groupID steamid.SteamID, groupName string,
	banType BanType, banGroup *BanGroup,
) error {
	var opts BanSteamGroupOpts

	errBaseOpts := newBaseBanOpts(source, target, duration, Custom, "Group Ban", modNote, origin, banType, &opts.BaseBanOpts)
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
	BanID        int64           `json:"ban_id"`
	BanMessageID int64           `json:"ban_message_id"`
	AuthorID     steamid.SteamID `json:"author_id"`
	MessageMD    string          `json:"message_md"`
	Deleted      bool            `json:"deleted"`
	CreatedOn    time.Time       `json:"created_on"`
	UpdatedOn    time.Time       `json:"updated_on"`
	SimplePerson
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
