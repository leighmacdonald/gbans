package store

import (
	"context"
	"fmt"
	"net"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type SteamIDProvider interface {
	SID64(ctx context.Context) (steamid.SID64, error)
}

// StringSID defines a user provided steam id in an unknown format.
type StringSID string

func (t StringSID) SID64(ctx context.Context) (steamid.SID64, error) {
	resolveCtx, cancelResolve := context.WithTimeout(ctx, time.Second*5)
	defer cancelResolve()

	// TODO cache this as it can be a huge hot path
	sid64, errResolveSID := steamid.ResolveSID64(resolveCtx, string(t))
	if errResolveSID != nil {
		return "", consts.ErrInvalidSID
	}

	if !sid64.Valid() {
		return "", consts.ErrInvalidSID
	}

	return sid64, nil
}

// BanType defines the state of the ban for a user, 0 being no ban.
type BanType int

const (
	// Unknown means the ban state could not be determined, failing-open to allowing players
	// to connect.
	Unknown BanType = iota - 1
	// OK Ban state is clean.
	OK
	// NoComm means the player cannot communicate while playing voice + chat.
	NoComm
	// Banned means the player cannot join the server at all.
	Banned
)

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
	}[r]
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

func newBaseBanOpts(ctx context.Context, source SteamIDProvider, target StringSID, duration time.Duration,
	reason Reason, reasonText string, modNote string, origin Origin,
	banType BanType, opts *BaseBanOpts,
) error {
	sourceSid, errSource := source.SID64(ctx)
	if errSource != nil {
		return errors.Wrapf(errSource, "Failed to parse source id")
	}

	targetSid := steamid.New(0)

	if string(target) != "0" {
		newTargetSid, errTargetSid := target.SID64(ctx)
		if errTargetSid != nil {
			return errors.New("Invalid target id")
		}

		targetSid = newTargetSid
	}

	if !(banType == Banned || banType == NoComm) {
		return errors.New("New ban must be ban or nocomm")
	}

	if duration <= 0 {
		return errors.New("Insufficient duration")
	}

	if reason == Custom && reasonText == "" {
		return errors.New("Custom reason cannot be empty")
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
		return errors.New("Invalid report ID")
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
		return errors.New("Invalid asn")
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
		return errors.Wrap(errParse, "Failed to parse cidr address")
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
	BanID           int64 `db:"ban_id" json:"ban_id"`
	ReportID        int64 `json:"report_id"`
	IncludeFriends  bool  `json:"include_friends"`
	CommunityBanned bool  `json:"community_banned"`
	VacBans         int   `json:"vac_bans"`
	GameBans        int   `json:"game_bans"`
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

func (db *Store) DropBan(ctx context.Context, ban *BanSteam, hardDelete bool) error {
	if hardDelete {
		if errExec := db.Exec(ctx, `DELETE FROM ban WHERE ban_id = $1`, ban.BanID); errExec != nil {
			return Err(errExec)
		}

		ban.BanID = 0

		return nil
	} else {
		ban.Deleted = true

		return db.updateBan(ctx, ban)
	}
}

func (db *Store) getBanByColumn(ctx context.Context, column string, identifier any, person *BannedSteamPerson, deletedOk bool) error {
	whereClauses := sq.And{
		sq.Eq{fmt.Sprintf("b.%s", column): identifier}, // valid columns are immutable
	}

	if !deletedOk {
		whereClauses = append(whereClauses, sq.Eq{"b.deleted": false})
	} else {
		whereClauses = append(whereClauses, sq.Gt{"b.valid_until": time.Now()})
	}

	query := db.sb.Select(
		"b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason",
		"b.reason_text", "b.note", "b.origin", "b.valid_until", "b.created_on", "b.updated_on", "b.include_friends",
		"b.deleted", "case WHEN b.report_id is null THEN 0 ELSE b.report_id END",
		"b.unban_reason_text", "b.is_enabled", "b.appeal_state",
		"s.personaname as source_personaname", "s.avatarhash",
		"t.personaname as target_personaname", "t.avatarhash", "t.community_banned", "t.vac_bans", "t.game_bans",
	).
		From("ban b").
		LeftJoin("person s on s.steam_id = b.source_id").
		LeftJoin("person t on t.steam_id = b.target_id").
		Where(whereClauses).
		OrderBy("b.created_on DESC").
		Limit(1)

	row, errQuery := db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errQuery
	}

	var (
		sourceID int64
		targetID int64
	)

	if errScan := row.
		Scan(&person.BanID, &targetID, &sourceID, &person.BanType, &person.Reason,
			&person.ReasonText, &person.Note, &person.Origin, &person.ValidUntil, &person.CreatedOn,
			&person.UpdatedOn, &person.IncludeFriends, &person.Deleted, &person.ReportID, &person.UnbanReasonText,
			&person.IsEnabled, &person.AppealState,
			&person.SourceTarget.SourcePersonaname, &person.SourceTarget.SourceAvatarhash,
			&person.SourceTarget.TargetPersonaname, &person.SourceTarget.TargetAvatarhash,
			&person.CommunityBanned, &person.VacBans, &person.GameBans,
		); errScan != nil {
		return Err(errScan)
	}

	person.SourceID = steamid.New(sourceID)
	person.TargetID = steamid.New(targetID)

	return nil
}

func (db *Store) GetBanBySteamID(ctx context.Context, sid64 steamid.SID64, bannedPerson *BannedSteamPerson, deletedOk bool) error {
	return db.getBanByColumn(ctx, "target_id", sid64, bannedPerson, deletedOk)
}

func (db *Store) GetBanByBanID(ctx context.Context, banID int64, bannedPerson *BannedSteamPerson, deletedOk bool) error {
	return db.getBanByColumn(ctx, "ban_id", banID, bannedPerson, deletedOk)
}

// SaveBan will insert or update the ban record
// New records will have the Ban.BanID set automatically.
func (db *Store) SaveBan(ctx context.Context, ban *BanSteam) error {
	// Ensure the foreign keys are satisfied
	targetPerson := NewPerson(ban.TargetID)
	if errGetPerson := db.GetOrCreatePersonBySteamID(ctx, ban.TargetID, &targetPerson); errGetPerson != nil {
		return errors.Wrapf(errGetPerson, "Failed to get targetPerson for ban")
	}

	authorPerson := NewPerson(ban.SourceID)
	if errGetAuthor := db.GetOrCreatePersonBySteamID(ctx, ban.SourceID, &authorPerson); errGetAuthor != nil {
		return errors.Wrapf(errGetAuthor, "Failed to get author for ban")
	}

	ban.UpdatedOn = time.Now()
	if ban.BanID > 0 {
		return db.updateBan(ctx, ban)
	}

	ban.CreatedOn = ban.UpdatedOn

	existing := NewBannedPerson()

	errGetBan := db.GetBanBySteamID(ctx, ban.TargetID, &existing, false)
	if errGetBan != nil {
		if !errors.Is(errGetBan, ErrNoResult) {
			return errors.Wrapf(errGetBan, "Failed to check existing ban state")
		}
	} else {
		if ban.BanType <= existing.BanType {
			return ErrDuplicate
		}
	}

	return db.insertBan(ctx, ban)
}

func (db *Store) insertBan(ctx context.Context, ban *BanSteam) error {
	const query = `
		INSERT INTO ban (target_id, source_id, ban_type, reason, reason_text, note, valid_until, 
		                 created_on, updated_on, origin, report_id, appeal_state, include_friends)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, case WHEN $11 = 0 THEN null ELSE $11 END, $12, $13)
		RETURNING ban_id`

	errQuery := db.
		QueryRow(ctx, query, ban.TargetID.Int64(), ban.SourceID.Int64(), ban.BanType, ban.Reason, ban.ReasonText,
			ban.Note, ban.ValidUntil, ban.CreatedOn, ban.UpdatedOn, ban.Origin, ban.ReportID, ban.AppealState, ban.IncludeFriends).
		Scan(&ban.BanID)

	if errQuery != nil {
		return Err(errQuery)
	}

	return nil
}

func (db *Store) updateBan(ctx context.Context, ban *BanSteam) error {
	query := db.sb.Update("ban").
		Set("source_id", ban.SourceID.Int64()).
		Set("reason", ban.Reason).
		Set("reason_text", ban.ReasonText).
		Set("note", ban.Note).
		Set("valid_until", ban.ValidUntil).
		Set("updated_on", ban.UpdatedOn).
		Set("origin", ban.Origin).
		Set("ban_type", ban.BanType).
		Set("deleted", ban.Deleted).
		Set("report_id", ban.ReportID).
		Set("unban_reason_text", ban.UnbanReasonText).
		Set("is_enabled", ban.IsEnabled).
		Set("target_id", ban.TargetID.Int64()).
		Set("appeal_state", ban.AppealState).
		Set("include_friends", ban.IncludeFriends).
		Where(sq.Eq{"ban_id": ban.BanID})

	return Err(db.ExecUpdateBuilder(ctx, query))
}

func (db *Store) GetExpiredBans(ctx context.Context) ([]BanSteam, error) {
	query := db.sb.
		Select("ban_id", "target_id", "source_id", "ban_type", "reason", "reason_text", "note",
			"valid_until", "origin", "created_on", "updated_on", "deleted", "case WHEN report_id is null THEN 0 ELSE report_id END",
			"unban_reason_text", "is_enabled", "appeal_state", "include_friends").
		From("ban").
		Where(sq.And{sq.Lt{"valid_until": time.Now()}, sq.Eq{"deleted": false}})

	rows, errQuery := db.QueryBuilder(ctx, query)
	if errQuery != nil {
		return nil, errQuery
	}

	defer rows.Close()

	bans := []BanSteam{}

	for rows.Next() {
		var (
			ban      BanSteam
			sourceID int64
			targetID int64
		)

		if errScan := rows.Scan(&ban.BanID, &targetID, &sourceID, &ban.BanType, &ban.Reason, &ban.ReasonText, &ban.Note,
			&ban.ValidUntil, &ban.Origin, &ban.CreatedOn, &ban.UpdatedOn, &ban.Deleted, &ban.ReportID, &ban.UnbanReasonText,
			&ban.IsEnabled, &ban.AppealState, &ban.IncludeFriends); errScan != nil {
			return nil, errors.Wrap(errScan, "Failed to load ban")
		}

		ban.SourceID = steamid.New(sourceID)
		ban.TargetID = steamid.New(targetID)

		bans = append(bans, ban)
	}

	return bans, nil
}

type AppealQueryFilter struct {
	QueryFilter
	AppealState AppealState `json:"appeal_state"`
	SourceID    StringSID   `json:"source_id"`
	TargetID    StringSID   `json:"target_id"`
}

func (db *Store) getActiveAppealsIDs(ctx context.Context) ([]int64, error) {
	query := db.sb.
		Select("a.ban_id").
		From("ban b").
		LeftJoin("ban_appeal a USING (ban_id)").
		Where(sq.And{sq.Eq{"b.deleted": false}, sq.Eq{"a.deleted": false}}).
		GroupBy("a.ban_id")

	idRows, errIds := db.QueryBuilder(ctx, query)
	if errIds != nil {
		return nil, Err(errIds)
	}

	defer idRows.Close()

	var banIds []int64

	for idRows.Next() {
		var validID int64

		if errScan := idRows.Scan(&validID); errScan != nil {
			return nil, Err(errScan)
		}

		banIds = append(banIds, validID)
	}

	return banIds, nil
}

func (db *Store) GetAppealsByActivity(ctx context.Context, opts AppealQueryFilter) ([]AppealOverview, int64, error) {
	banIds, errBanIds := db.getActiveAppealsIDs(ctx)
	if errBanIds != nil {
		return nil, 0, errBanIds
	}

	var constraints sq.And

	if !opts.Deleted {
		constraints = append(constraints, sq.Eq{"b.deleted": opts.Deleted})
	}

	if opts.AppealState > AnyState {
		constraints = append(constraints, sq.Eq{"b.appeal_state": opts.AppealState})
	}

	if opts.SourceID != "" {
		authorID, errAuthorID := opts.SourceID.SID64(ctx)
		if errAuthorID != nil {
			return nil, 0, errAuthorID
		}

		constraints = append(constraints, sq.Eq{"b.source_id": authorID.Int64()})
	}

	if opts.TargetID != "" {
		targetID, errTargetID := opts.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errTargetID
		}

		constraints = append(constraints, sq.Eq{"b.target_id": targetID.Int64()})
	}

	constraints = append(constraints, sq.Eq{"b.ban_id": banIds})

	counterQuery := db.sb.
		Select("COUNT(b.ban_id)").
		From("ban b").
		Where(constraints)

	count, errCount := db.GetCount(ctx, counterQuery)
	if errCount != nil {
		return nil, 0, Err(errCount)
	}

	builder := db.sb.
		Select("b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason", "b.reason_text",
			"b.note", "b.valid_until", "b.origin", "b.created_on", "b.updated_on", "b.deleted",
			"CASE WHEN b.report_id IS NULL THEN 0 ELSE report_id END",
			"b.unban_reason_text", "b.is_enabled", "b.appeal_state",
			"source.steam_id as source_steam_id", "source.personaname as source_personaname",
			"source.avatarhash as source_avatar",
			"target.steam_id as target_steam_id", "target.personaname as target_personaname",
			"target.avatarhash as target_avatar").
		From("ban b").
		Where(constraints).
		LeftJoin("person source on source.steam_id = b.source_id").
		LeftJoin("person target on target.steam_id = b.target_id")

	builder = opts.QueryFilter.applySafeOrder(builder, map[string][]string{
		"b.": {
			"ban_id", "target_id", "source_id", "ban_type", "reason", "valid_until", "origin", "created_on",
			"updated_on", "deleted", "is_enabled", "appeal_state",
		},
	}, "updated_on")

	builder = opts.QueryFilter.applyLimitOffsetDefault(builder)

	rows, errQuery := db.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, 0, Err(errQuery)
	}

	defer rows.Close()

	overviews := []AppealOverview{}

	for rows.Next() {
		var (
			overview      AppealOverview
			sourceID      int64
			SourceSteamID int64
			targetID      int64
			TargetSteamID int64
		)

		if errScan := rows.Scan(
			&overview.BanID, &targetID, &sourceID, &overview.BanType,
			&overview.Reason, &overview.ReasonText, &overview.Note, &overview.ValidUntil,
			&overview.Origin, &overview.CreatedOn, &overview.UpdatedOn, &overview.Deleted,
			&overview.ReportID, &overview.UnbanReasonText, &overview.IsEnabled, &overview.AppealState,
			&SourceSteamID, &overview.SourcePersonaname, &overview.SourceAvatarhash,
			&TargetSteamID, &overview.TargetPersonaname, &overview.TargetAvatarhash,
		); errScan != nil {
			return nil, 0, errors.Wrap(errScan, "Failed to scan appeal overview")
		}

		overview.SourceID = steamid.New(SourceSteamID)
		overview.TargetID = steamid.New(TargetSteamID)

		overviews = append(overviews, overview)
	}

	return overviews, count, nil
}

type BansQueryFilter struct {
	QueryFilter
	SourceID      StringSID `json:"source_id,omitempty"`
	TargetID      StringSID `json:"target_id,omitempty"`
	Reason        Reason    `json:"reason,omitempty"`
	PermanentOnly bool      `json:"permanent_only,omitempty"`
}

type CIDRBansQueryFilter struct {
	BansQueryFilter
	IP string `json:"ip,omitempty"`
}

type ASNBansQueryFilter struct {
	BansQueryFilter
	ASNum int64 `json:"as_num,omitempty"`
}

type GroupBansQueryFilter struct {
	BansQueryFilter
	GroupID string `json:"group_id"`
}

type SteamBansQueryFilter struct {
	BansQueryFilter
	// IncludeFriendsOnly Return results that have "deep" bans where players friends list is
	// also banned while the primary targets ban has not expired.
	IncludeFriendsOnly bool
}

// GetBansSteam returns all bans that fit the filter criteria passed in.
func (db *Store) GetBansSteam(ctx context.Context, filter SteamBansQueryFilter) ([]BannedSteamPerson, int64, error) {
	builder := db.sb.Select("b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason",
		"b.reason_text", "b.note", "b.origin", "b.valid_until", "b.created_on", "b.updated_on", "b.include_friends",
		"b.deleted", "case WHEN b.report_id is null THEN 0 ELSE b.report_id END",
		"b.unban_reason_text", "b.is_enabled", "b.appeal_state",
		"s.personaname as source_personaname", "s.avatarhash",
		"t.personaname as target_personaname", "t.avatarhash", "t.community_banned", "t.vac_bans", "t.game_bans").
		From("ban b").
		JoinClause("LEFT JOIN person s on s.steam_id = b.source_id").
		JoinClause("LEFT JOIN person t on t.steam_id = b.target_id")

	var ands sq.And

	if !filter.Deleted {
		ands = append(ands, sq.Eq{"b.deleted": false})
	}

	if filter.Reason > 0 {
		ands = append(ands, sq.Eq{"b.reason": filter.Reason})
	}

	if filter.PermanentOnly {
		ands = append(ands, sq.Gt{"b.valid_until": time.Now()})
	}

	if filter.TargetID != "" {
		targetID, errTargetID := filter.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errTargetID
		}

		ands = append(ands, sq.Eq{"b.target_id": targetID.Int64()})
	}

	if filter.SourceID != "" {
		sourceID, errSourceID := filter.SourceID.SID64(ctx)
		if errSourceID != nil {
			return nil, 0, errSourceID
		}

		ands = append(ands, sq.Eq{"b.source_id": sourceID.Int64()})
	}

	if filter.IncludeFriendsOnly {
		ands = append(ands, sq.Eq{"b.include_friends": true})
	}

	if len(ands) > 0 {
		builder = builder.Where(ands)
	}

	builder = filter.QueryFilter.applySafeOrder(builder, map[string][]string{
		"b.": {
			"ban_id", "target_id", "source_id", "ban_type", "reason",
			"origin", "valid_until", "created_on", "updated_on", "include_friends",
			"deleted", "report_id", "is_enabled", "appeal_state",
		},
		"s.": {"source_personaname"},
		"t.": {"target_personaname", "community_banned", "vac_bans", "game_bans"},
	}, "ban_id")

	builder = filter.QueryFilter.applyLimitOffsetDefault(builder)

	count, errCount := db.GetCount(ctx, db.sb.
		Select("COUNT(b.ban_id)").
		From("ban b").
		Where(ands))
	if errCount != nil {
		return nil, 0, Err(errCount)
	}

	if count == 0 {
		return []BannedSteamPerson{}, 0, nil
	}

	var bans []BannedSteamPerson

	rows, errQuery := db.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, 0, Err(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			person   = NewBannedPerson()
			sourceID int64
			targetID int64
		)

		if errScan := rows.
			Scan(&person.BanID, &targetID, &sourceID, &person.BanType, &person.Reason,
				&person.ReasonText, &person.Note, &person.Origin, &person.ValidUntil, &person.CreatedOn,
				&person.UpdatedOn, &person.IncludeFriends, &person.Deleted, &person.ReportID, &person.UnbanReasonText,
				&person.IsEnabled, &person.AppealState,
				&person.SourceTarget.SourcePersonaname, &person.SourceTarget.SourceAvatarhash,
				&person.SourceTarget.TargetPersonaname, &person.SourceTarget.TargetAvatarhash,
				&person.CommunityBanned, &person.VacBans, &person.GameBans); errScan != nil {
			return nil, 0, Err(errScan)
		}

		person.TargetID = steamid.New(targetID)
		person.SourceID = steamid.New(sourceID)

		bans = append(bans, person)
	}

	if bans == nil {
		bans = []BannedSteamPerson{}
	}

	return bans, count, nil
}

func (db *Store) GetBansOlderThan(ctx context.Context, filter QueryFilter, since time.Time) ([]BanSteam, error) {
	query := db.sb.
		Select("b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason",
			"b.reason_text", "b.note", "b.origin", "b.valid_until", "b.created_on", "b.updated_on", "b.deleted",
			"case WHEN b.report_id is null THEN 0 ELSE b.report_id END", "b.unban_reason_text", "b.is_enabled",
			"b.appeal_state", "b.include_friends").
		From("ban b").
		Where(sq.And{sq.Lt{"updated_on": since}, sq.Eq{"deleted": false}})

	rows, errQuery := db.QueryBuilder(ctx, filter.applyLimitOffsetDefault(query))
	if errQuery != nil {
		return nil, errQuery
	}

	defer rows.Close()

	bans := []BanSteam{}

	for rows.Next() {
		var (
			ban      BanSteam
			sourceID int64
			targetID int64
		)

		if errQuery = rows.Scan(&ban.BanID, &targetID, &sourceID, &ban.BanType, &ban.Reason, &ban.ReasonText, &ban.Note,
			&ban.Origin, &ban.ValidUntil, &ban.CreatedOn, &ban.UpdatedOn, &ban.Deleted, &ban.ReportID, &ban.UnbanReasonText,
			&ban.IsEnabled, &ban.AppealState, &ban.AppealState); errQuery != nil {
			return nil, errors.Wrap(errQuery, "Failed to scan ban")
		}

		ban.SourceID = steamid.New(sourceID)
		ban.TargetID = steamid.New(targetID)

		bans = append(bans, ban)
	}

	return bans, nil
}

func (db *Store) SaveBanMessage(ctx context.Context, message *UserMessage) error {
	var err error
	if message.MessageID > 0 {
		err = db.updateBanMessage(ctx, message)
	} else {
		err = db.insertBanMessage(ctx, message)
	}

	bannedPerson := NewBannedPerson()
	if errBan := db.GetBanByBanID(ctx, message.ParentID, &bannedPerson, true); errBan != nil {
		return ErrNoResult
	}

	bannedPerson.UpdatedOn = time.Now()

	if errUpdate := db.updateBan(ctx, &bannedPerson.BanSteam); errUpdate != nil {
		return errUpdate
	}

	return err
}

func (db *Store) updateBanMessage(ctx context.Context, message *UserMessage) error {
	message.UpdatedOn = time.Now()

	query := db.sb.Update("ban_appeal").
		Set("deleted", message.Deleted).
		Set("author_id", message.AuthorID.Int64()).
		Set("updated_on", message.UpdatedOn).
		Set("message_md", message.Contents).
		Where(sq.Eq{"ban_message_id": message.MessageID})

	if errQuery := db.ExecUpdateBuilder(ctx, query); errQuery != nil {
		return Err(errQuery)
	}

	db.log.Info("Ban appeal message updated",
		zap.Int64("ban_id", message.ParentID),
		zap.Int64("message_id", message.MessageID),
		zap.Int64("author_id", message.AuthorID.Int64()))

	return nil
}

func (db *Store) insertBanMessage(ctx context.Context, message *UserMessage) error {
	const query = `
	INSERT INTO ban_appeal (
		ban_id, author_id, message_md, deleted, created_on, updated_on
	)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING ban_message_id
	`

	if errQuery := db.QueryRow(ctx, query,
		message.ParentID,
		message.AuthorID.Int64(),
		message.Contents,
		message.Deleted,
		message.CreatedOn,
		message.UpdatedOn,
	).Scan(&message.MessageID); errQuery != nil {
		return Err(errQuery)
	}

	db.log.Info("Ban appeal message created",
		zap.Int64("ban_id", message.ParentID),
		zap.Int64("message_id", message.MessageID),
		zap.Int64("author_id", message.AuthorID.Int64()))

	return nil
}

func (db *Store) GetBanMessages(ctx context.Context, banID int64) ([]UserMessage, error) {
	query := db.sb.
		Select("ban_message_id", "ban_id", "author_id", "message_md", "deleted", "created_on", "updated_on").
		From("ban_appeal").
		Where(sq.And{sq.Eq{"deleted": false}, sq.Eq{"ban_id": banID}}).
		OrderBy("created_on")

	rows, errQuery := db.QueryBuilder(ctx, query)
	if errQuery != nil {
		if errors.Is(Err(errQuery), ErrNoResult) {
			return nil, nil
		}
	}

	defer rows.Close()

	messages := []UserMessage{}

	for rows.Next() {
		var (
			msg      UserMessage
			authorID int64
		)

		if errScan := rows.Scan(
			&msg.MessageID,
			&msg.ParentID,
			&authorID,
			&msg.Contents,
			&msg.Deleted,
			&msg.CreatedOn,
			&msg.UpdatedOn,
		); errScan != nil {
			return nil, Err(errQuery)
		}

		msg.AuthorID = steamid.New(authorID)

		messages = append(messages, msg)
	}

	return messages, nil
}

func (db *Store) GetBanMessageByID(ctx context.Context, banMessageID int, message *UserMessage) error {
	query := db.sb.
		Select("ban_message_id", "ban_id", "author_id", "message_md", "deleted", "created_on", "updated_on").
		From("ban_appeal").
		Where(sq.Eq{"ban_message_id": banMessageID})

	var authorID int64

	row, errQuery := db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errQuery
	}

	if errScan := row.Scan(
		&message.MessageID,
		&message.ParentID,
		&authorID,
		&message.Contents,
		&message.Deleted,
		&message.CreatedOn,
		&message.UpdatedOn,
	); errScan != nil {
		return Err(errScan)
	}

	message.AuthorID = steamid.New(authorID)

	return nil
}

func (db *Store) DropBanMessage(ctx context.Context, message *UserMessage) error {
	query := db.sb.
		Update("ban_appeal").
		Set("deleted", true).
		Where(sq.Eq{"ban_message_id": message.MessageID})

	if errExec := db.ExecUpdateBuilder(ctx, query); errExec != nil {
		return Err(errExec)
	}

	db.log.Info("Appeal message deleted", zap.Int64("ban_message_id", message.MessageID))
	message.Deleted = true

	return nil
}

func (db *Store) GetBanGroup(ctx context.Context, groupID steamid.GID, banGroup *BanGroup) error {
	query := db.sb.
		Select("ban_group_id", "source_id", "target_id", "group_name", "is_enabled", "deleted",
			"note", "unban_reason_text", "origin", "created_on", "updated_on", "valid_until", "appeal_state", "group_id").
		From("ban_group").
		Where(sq.And{sq.Eq{"group_id": groupID.Int64()}, sq.Eq{"deleted": false}})

	var (
		sourceID   int64
		targetID   int64
		newGroupID int64
	)

	row, errQuery := db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errQuery
	}

	if errScan := row.Scan(&banGroup.BanGroupID, &sourceID, &targetID, &banGroup.GroupName, &banGroup.IsEnabled,
		&banGroup.Deleted, &banGroup.Note, &banGroup.UnbanReasonText, &banGroup.Origin, &banGroup.CreatedOn,
		&banGroup.UpdatedOn, &banGroup.ValidUntil, &banGroup.AppealState, &newGroupID); errScan != nil {
		return Err(errScan)
	}

	banGroup.SourceID = steamid.New(sourceID)
	banGroup.TargetID = steamid.New(targetID)
	banGroup.GroupID = steamid.NewGID(newGroupID)

	return nil
}

func (db *Store) GetBanGroupByID(ctx context.Context, banGroupID int64, banGroup *BanGroup) error {
	query := db.sb.
		Select("ban_group_id", "source_id", "target_id", "group_name", "is_enabled", "deleted",
			"note", "unban_reason_text", "origin", "created_on", "updated_on", "valid_until", "appeal_state", "group_id").
		From("ban_group").
		Where(sq.And{sq.Eq{"ban_group_id": banGroupID}, sq.Eq{"is_enabled": true}, sq.Eq{"deleted": false}})

	var (
		groupID  int64
		sourceID int64
		targetID int64
	)

	row, errQuery := db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errQuery
	}

	if errScan := row.Scan(
		&banGroup.BanGroupID,
		&sourceID,
		&targetID,
		&banGroup.GroupName,
		&banGroup.IsEnabled,
		&banGroup.Deleted,
		&banGroup.Note,
		&banGroup.UnbanReasonText,
		&banGroup.Origin,
		&banGroup.CreatedOn,
		&banGroup.UpdatedOn,
		&banGroup.ValidUntil,
		&banGroup.AppealState,
		&groupID); errScan != nil {
		return Err(errScan)
	}

	banGroup.SourceID = steamid.New(sourceID)
	banGroup.TargetID = steamid.New(targetID)
	banGroup.GroupID = steamid.NewGID(groupID)

	return nil
}

func (db *Store) GetBanGroups(ctx context.Context, filter GroupBansQueryFilter) ([]BannedGroupPerson, int64, error) {
	builder := db.sb.
		Select("b.ban_group_id", "b.source_id", "b.target_id", "b.group_name", "b.is_enabled", "b.deleted",
			"b.note", "b.unban_reason_text", "b.origin", "b.created_on", "b.updated_on", "b.valid_until",
			"b.appeal_state", "b.group_id",
			"s.personaname as source_personaname", "s.avatarhash",
			"t.personaname as target_personaname", "t.avatarhash", "t.community_banned", "t.vac_bans", "t.game_bans").
		From("ban_group b").
		LeftJoin("person s ON s.steam_id = b.source_id").
		LeftJoin("person t ON t.steam_id = b.target_id")

	var constraints sq.And

	if !filter.Deleted {
		constraints = append(constraints, sq.Eq{"b.deleted": false})
	}

	if filter.Reason > 0 {
		constraints = append(constraints, sq.Eq{"b.reason": filter.Reason})
	}

	if filter.PermanentOnly {
		constraints = append(constraints, sq.Gt{"b.valid_until": time.Now()})
	}

	if filter.GroupID != "" {
		gid := steamid.NewGID(filter.GroupID)
		if !gid.Valid() {
			return nil, 0, steamid.ErrInvalidGID
		}

		constraints = append(constraints, sq.Eq{"b.group_id": gid.Int64()})
	}

	if filter.TargetID != "" {
		targetID, errTargetID := filter.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errTargetID
		}

		constraints = append(constraints, sq.Eq{"b.target_id": targetID.Int64()})
	}

	if filter.SourceID != "" {
		sourceID, errSourceID := filter.SourceID.SID64(ctx)
		if errSourceID != nil {
			return nil, 0, errSourceID
		}

		constraints = append(constraints, sq.Eq{"b.source_id": sourceID.Int64()})
	}

	builder = filter.QueryFilter.applySafeOrder(builder, map[string][]string{
		"b.": {
			"ban_group_id", "source_id", "target_id", "group_name", "is_enabled", "deleted",
			"origin", "created_on", "updated_on", "valid_until", "appeal_state", "group_id",
		},
		"s.": {"source_personaname"},
		"t.": {"target_personaname", "community_banned", "vac_bans", "game_bans"},
	}, "ban_group_id")

	builder = filter.applyLimitOffsetDefault(builder).Where(constraints)

	rows, errRows := db.QueryBuilder(ctx, builder)
	if errRows != nil {
		if errors.Is(errRows, ErrNoResult) {
			return []BannedGroupPerson{}, 0, nil
		}

		return nil, 0, Err(errRows)
	}

	defer rows.Close()

	var groups []BannedGroupPerson

	for rows.Next() {
		var (
			group    BannedGroupPerson
			groupID  int64
			sourceID int64
			targetID int64
		)

		if errScan := rows.Scan(
			&group.BanGroupID,
			&sourceID,
			&targetID,
			&group.GroupName,
			&group.IsEnabled,
			&group.Deleted,
			&group.Note,
			&group.UnbanReasonText,
			&group.Origin,
			&group.CreatedOn,
			&group.UpdatedOn,
			&group.ValidUntil,
			&group.AppealState,
			&groupID,
			&group.SourceTarget.SourcePersonaname, &group.SourceTarget.SourceAvatarhash,
			&group.SourceTarget.TargetPersonaname, &group.SourceTarget.TargetAvatarhash,
			&group.CommunityBanned, &group.VacBans, &group.GameBans,
		); errScan != nil {
			return nil, 0, Err(errScan)
		}

		group.SourceID = steamid.New(sourceID)
		group.TargetID = steamid.New(targetID)
		group.GroupID = steamid.NewGID(groupID)

		groups = append(groups, group)
	}

	count, errCount := db.GetCount(ctx, db.sb.
		Select("b.ban_group_id").
		From("ban_group b").
		Where(constraints))
	if errCount != nil {
		if errors.Is(errCount, ErrNoResult) {
			return []BannedGroupPerson{}, 0, nil
		}

		return nil, 0, errCount
	}

	if groups == nil {
		groups = []BannedGroupPerson{}
	}

	return groups, count, nil
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

func (db *Store) GetMembersList(ctx context.Context, parentID int64, list *MembersList) error {
	row, err := db.QueryRowBuilder(ctx, db.sb.
		Select("members_id", "parent_id", "members", "created_on", "updated_on").
		From("members").
		Where(sq.Eq{"parent_id": parentID}))
	if err != nil {
		return err
	}

	return Err(row.Scan(&list.MembersID, &list.ParentID, &list.Members, &list.CreatedOn, &list.UpdatedOn))
}

func (db *Store) SaveMembersList(ctx context.Context, list *MembersList) error {
	if list.MembersID > 0 {
		list.UpdatedOn = time.Now()

		const update = `UPDATE members SET members = $2::jsonb, updated_on = $3 WHERE members_id = $1`

		return Err(db.Exec(ctx, update, list.MembersID, list.Members, list.UpdatedOn))
	} else {
		const insert = `INSERT INTO members (parent_id, members, created_on, updated_on) 
		VALUES ($1, $2::jsonb, $3, $4) RETURNING members_id`

		return Err(db.QueryRow(ctx, insert, list.ParentID, list.Members, list.CreatedOn, list.UpdatedOn).Scan(&list.MembersID))
	}
}

func (db *Store) SaveBanGroup(ctx context.Context, banGroup *BanGroup) error {
	if banGroup.BanGroupID > 0 {
		return db.updateBanGroup(ctx, banGroup)
	}

	return db.insertBanGroup(ctx, banGroup)
}

func (db *Store) insertBanGroup(ctx context.Context, banGroup *BanGroup) error {
	const query = `
	INSERT INTO ban_group (source_id, target_id, group_id, group_name, is_enabled, deleted, note,
	unban_reason_text, origin, created_on, updated_on, valid_until, appeal_state)
	VALUES ($1, $2, $3, $4, $5, $6,$7, $8, $9, $10, $11, $12, $13)
	RETURNING ban_group_id`

	return Err(db.
		QueryRow(ctx, query, banGroup.SourceID.Int64(), banGroup.TargetID.Int64(), banGroup.GroupID.Int64(),
			banGroup.GroupName, banGroup.IsEnabled, banGroup.Deleted, banGroup.Note, banGroup.UnbanReasonText, banGroup.Origin,
			banGroup.CreatedOn, banGroup.UpdatedOn, banGroup.ValidUntil, banGroup.AppealState).
		Scan(&banGroup.BanGroupID))
}

func (db *Store) updateBanGroup(ctx context.Context, banGroup *BanGroup) error {
	banGroup.UpdatedOn = time.Now()

	return Err(db.ExecUpdateBuilder(ctx, db.sb.
		Update("ban_group").
		Set("source_id", banGroup.SourceID.Int64()).
		Set("target_id", banGroup.TargetID.Int64()).
		Set("group_name", banGroup.GroupName).
		Set("is_enabled", banGroup.IsEnabled).
		Set("deleted", banGroup.Deleted).
		Set("note", banGroup.Note).
		Set("unban_reason_text", banGroup.UnbanReasonText).
		Set("origin", banGroup.Origin).
		Set("updated_on", banGroup.UpdatedOn).
		Set("group_id", banGroup.GroupID.Int64()).
		Set("valid_until", banGroup.ValidUntil).
		Set("appeal_state", banGroup.AppealState).
		Where(sq.Eq{"ban_group_id": banGroup.BanGroupID})))
}

func (db *Store) DropBanGroup(ctx context.Context, banGroup *BanGroup) error {
	banGroup.IsEnabled = false
	banGroup.Deleted = true

	return db.SaveBanGroup(ctx, banGroup)
}
