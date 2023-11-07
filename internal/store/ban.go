package store

import (
	"context"
	"fmt"
	"net"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
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

type BannedPerson struct {
	Ban    BanSteam `json:"ban"`
	Person Person   `json:"person"`
}

func NewBannedPerson() BannedPerson {
	banTime := time.Now()

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
	BanGroupID int64       `json:"ban_group_id"`
	GroupID    steamid.GID `json:"group_id"`
	GroupName  string      `json:"group_name"`
}

func (banGroup *BanGroup) Apply(opts BanSteamGroupOpts) error {
	banGroup.ApplyBaseOpts(opts.BaseBanOpts)
	banGroup.GroupName = opts.GroupName
	banGroup.GroupID = opts.GroupID

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

type BanSteam struct {
	BanBase
	BanID          int64 `db:"ban_id" json:"ban_id"`
	ReportID       int64 `json:"report_id"`
	IncludeFriends bool  `json:"include_friends"`
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

func (db *Store) getBanByColumn(ctx context.Context, column string, identifier any, person *BannedPerson, deletedOk bool) error {
	whereClauses := sq.And{
		sq.Eq{fmt.Sprintf("b.%s", column): identifier},
	}

	if !deletedOk {
		whereClauses = append(whereClauses, sq.Eq{"b.deleted": false})
	} else {
		whereClauses = append(whereClauses, sq.Gt{"b.valid_until": time.Now()})
	}

	builder := db.sb.Select(
		"b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason",
		"b.reason_text", "b.note", "b.origin", "b.valid_until", "b.created_on", "b.updated_on", "b.include_friends",
		"p.steam_id as sid2",
		"p.created_on as created_on2", "p.updated_on as updated_on2", "p.communityvisibilitystate",
		"p.profilestate", "p.personaname", "p.profileurl", "p.avatar", "p.avatarmedium", "p.avatarfull",
		"p.avatarhash", "p.personastate", "p.realname", "p.timecreated", "p.loccountrycode", "p.locstatecode",
		"p.loccityid", "p.permission_level", "p.discord_id", "p.community_banned", "p.vac_bans", "p.game_bans",
		"p.economy_ban", "p.days_since_last_ban", "b.deleted", "case WHEN b.report_id is null THEN 0 ELSE b.report_id END",
		"b.unban_reason_text", "b.is_enabled", "b.appeal_state").
		From("ban b").
		JoinClause("LEFT OUTER JOIN person p on p.steam_id = b.target_id").
		Where(whereClauses).
		GroupBy("b.ban_id, p.steam_id").
		OrderBy("b.created_on DESC").
		Limit(1)

	query, args, queryErr := builder.ToSql()
	if queryErr != nil {
		return errors.Wrap(queryErr, "Failed to create query")
	}

	var (
		sid      int64
		sourceID int64
		targetID int64
	)

	if errQuery := db.conn.
		QueryRow(ctx, query, args...).
		Scan(&person.Ban.BanID, &targetID, &sourceID, &person.Ban.BanType, &person.Ban.Reason,
			&person.Ban.ReasonText, &person.Ban.Note, &person.Ban.Origin, &person.Ban.ValidUntil, &person.Ban.CreatedOn,
			&person.Ban.UpdatedOn, &person.Ban.IncludeFriends, &sid, &person.Person.CreatedOn, &person.Person.UpdatedOn,
			&person.Person.CommunityVisibilityState, &person.Person.ProfileState, &person.Person.PersonaName,
			&person.Person.ProfileURL, &person.Person.Avatar, &person.Person.AvatarMedium, &person.Person.AvatarFull,
			&person.Person.AvatarHash, &person.Person.PersonaState, &person.Person.RealName, &person.Person.TimeCreated,
			&person.Person.LocCountryCode, &person.Person.LocStateCode, &person.Person.LocCityID,
			&person.Person.PermissionLevel, &person.Person.DiscordID, &person.Person.CommunityBanned,
			&person.Person.VACBans, &person.Person.GameBans, &person.Person.EconomyBan, &person.Person.DaysSinceLastBan,
			&person.Ban.Deleted, &person.Ban.ReportID, &person.Ban.UnbanReasonText, &person.Ban.IsEnabled,
			&person.Ban.AppealState); errQuery != nil {
		return Err(errQuery)
	}

	person.Person.SteamID = steamid.New(sid)
	person.Ban.SourceID = steamid.New(sourceID)
	person.Ban.TargetID = steamid.New(targetID)

	return nil
}

func (db *Store) GetBanBySteamID(ctx context.Context, sid64 steamid.SID64, bannedPerson *BannedPerson, deletedOk bool) error {
	return db.getBanByColumn(ctx, "target_id", sid64, bannedPerson, deletedOk)
}

func (db *Store) GetBanByBanID(ctx context.Context, banID int64, bannedPerson *BannedPerson, deletedOk bool) error {
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
		if ban.BanType <= existing.Ban.BanType {
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
	const query = `
		UPDATE ban
		SET source_id = $2, reason = $3, reason_text = $4, note = $5, valid_until = $6, updated_on = $7, 
			origin = $8, ban_type = $9, deleted = $10, report_id = case WHEN $11 = 0 THEN null ELSE $11 END, 
			unban_reason_text = $12, is_enabled = $13, target_id = $14, appeal_state = $15, include_friends = $16
		WHERE ban_id = $1`

	if errExec := db.
		Exec(ctx, query, ban.BanID, ban.SourceID.Int64(), ban.Reason, ban.ReasonText, ban.Note, ban.ValidUntil,
			ban.UpdatedOn, ban.Origin, ban.BanType, ban.Deleted, ban.ReportID, ban.UnbanReasonText, ban.IsEnabled,
			ban.TargetID.Int64(), ban.AppealState, ban.IncludeFriends); errExec != nil {
		return Err(errExec)
	}

	return nil
}

func (db *Store) GetExpiredBans(ctx context.Context) ([]BanSteam, error) {
	const query = `
		SELECT ban_id, target_id, source_id, ban_type, reason, reason_text, note, valid_until, origin, 
		       created_on, updated_on, deleted, case WHEN report_id is null THEN 0 ELSE report_id END, 
		       unban_reason_text, is_enabled, appeal_state, include_friends
		FROM ban
       	WHERE valid_until < $1 AND deleted = false`

	rows, errQuery := db.Query(ctx, query, time.Now())
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
	AuthorQueryFilter
	AppealState AppealState `json:"appeal_state"`
	TargetID    StringSID   `json:"target_id"`
}

func (db *Store) getActiveAppealsIDs(ctx context.Context) ([]int64, error) {
	const appealedBansQuery = `
		SELECT a.ban_id 
		FROM ban b 
			LEFT JOIN ban_appeal a USING (ban_id) 
		WHERE b.deleted = false AND a.deleted = false 
		GROUP BY a.ban_id`

	idRows, errIds := db.Query(ctx, appealedBansQuery)
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

	ands := sq.And{
		sq.Eq{"b.deleted": opts.Deleted},
	}

	if opts.AppealState > AnyState {
		ands = append(ands, sq.Eq{"b.appeal_state": opts.AppealState})
	}

	if opts.AuthorID != "" {
		authorID, errAuthorID := opts.AuthorID.SID64(ctx)
		if errAuthorID != nil {
			return nil, 0, errAuthorID
		}

		ands = append(ands, sq.Eq{"b.source_id": authorID.Int64()})
	}

	if opts.TargetID != "" {
		targetID, errTargetID := opts.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errTargetID
		}

		ands = append(ands, sq.Eq{"b.target_id": targetID.Int64()})
	}

	ands = append(ands, sq.Eq{"b.ban_id": banIds})

	counterQuery, counterArgs, errCounter := db.sb.
		Select("COUNT(b.ban_id)").
		From("ban b").
		Where(ands).
		ToSql()
	if errCounter != nil {
		return nil, 0, Err(errCounter)
	}

	count, errCount := db.GetCount(ctx, counterQuery, counterArgs...)
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
		Where(ands).
		LeftJoin("person source on source.steam_id = b.source_id").
		LeftJoin("person target on target.steam_id = b.target_id")

	order := "desc"
	if !opts.Desc {
		order = "asc"
	}

	var limit uint64

	if opts.Limit <= 0 || opts.Limit > 100 {
		limit = 50
	} else {
		limit = opts.Limit
	}

	offset := uint64(0)
	if opts.Offset > 0 {
		offset = opts.Offset
	}

	builder = builder.Limit(limit).Offset(offset)

	orderBy := "b.updated_on"
	if opts.OrderBy != "" {
		orderBy = fmt.Sprintf("b.%s", opts.OrderBy)
	}

	builder = builder.OrderBy(fmt.Sprintf("%s %s", orderBy, order))

	query, args, errBuilder := builder.ToSql()
	if errBuilder != nil {
		return nil, 0, Err(errBuilder)
	}

	overviews := []AppealOverview{}

	rows, errQuery := db.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, 0, Err(errQuery)
	}

	defer rows.Close()

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
			&SourceSteamID, &overview.SourcePersonaName, &overview.SourceAvatar,
			&TargetSteamID, &overview.TargetPersonaName, &overview.TargetAvatar,
		); errScan != nil {
			return nil, 0, errors.Wrap(errScan, "Failed to scan appeal overview")
		}

		overview.SourceSteamID = steamid.New(SourceSteamID)
		overview.TargetSteamID = steamid.New(TargetSteamID)
		overview.TargetID = steamid.New(targetID)
		overview.SourceID = steamid.New(sourceID)

		overviews = append(overviews, overview)
	}

	return overviews, count, nil
}

type BansQueryFilter struct {
	QueryFilter
	SteamID       steamid.SID64 `json:"steam_id,omitempty"`
	Reasons       []Reason
	PermanentOnly bool
	// IncludeFriendsOnly Return results that have "deep" bans where players friends list is
	// also banned while the primary targets ban has not expired.
	IncludeFriendsOnly bool
}

func NewBansQueryFilter(steamID steamid.SID64) BansQueryFilter {
	return BansQueryFilter{
		SteamID:     steamID,
		QueryFilter: NewQueryFilter(""),
	}
}

// GetBansSteam returns all bans that fit the filter criteria passed in.
func (db *Store) GetBansSteam(ctx context.Context, filter BansQueryFilter) ([]BannedPerson, error) {
	builder := db.sb.Select("b.ban_id as ban_id", "b.target_id as target_id", "b.source_id as source_id",
		"b.ban_type as ban_type", "b.reason as reason", "b.reason_text as reason_text",
		"b.note as note", "b.origin as origin", "b.valid_until as valid_until", "b.created_on as created_on",
		"b.updated_on as updated_on", "b.include_friends", "p.steam_id as sid2",
		"p.created_on as created_on2", "p.updated_on as updated_on2", "p.communityvisibilitystate",
		"p.profilestate", "p.personaname as personaname", "p.profileurl", "p.avatar", "p.avatarmedium", "p.avatarfull",
		"p.avatarhash", "p.personastate", "p.realname", "p.timecreated", "p.loccountrycode", "p.locstatecode",
		"p.loccityid", "p.permission_level", "p.discord_id as discord_id", "p.community_banned", "p.vac_bans", "p.game_bans",
		"p.economy_ban", "p.days_since_last_ban", "b.deleted as deleted",
		"case WHEN b.report_id is null THEN 0 ELSE b.report_id END", "b.unban_reason_text", "b.is_enabled",
		"b.appeal_state as appeal_state").
		From("ban b").
		JoinClause("LEFT OUTER JOIN person p on p.steam_id = b.target_id")

	var ands sq.And

	if !filter.Deleted {
		ands = append(ands, sq.Eq{"deleted": false})
	}

	if len(filter.Reasons) > 0 {
		ands = append(ands, sq.Eq{"reason": filter.Reasons})
	}

	if filter.PermanentOnly {
		ands = append(ands, sq.Gt{"valid_until": time.Now()})
	}

	if filter.SteamID.Valid() {
		ands = append(ands, sq.Eq{"b.target_id": filter.SteamID.Int64()})
	}

	if filter.IncludeFriendsOnly {
		ands = append(ands, sq.Eq{"include_friends": true})
	}

	if len(ands) > 0 {
		builder = builder.Where(ands)
	}

	if filter.OrderBy != "" {
		if filter.Desc {
			builder = builder.OrderBy(fmt.Sprintf("b.%s DESC", filter.OrderBy))
		} else {
			builder = builder.OrderBy(fmt.Sprintf("b.%s ASC", filter.OrderBy))
		}
	}

	if filter.Limit > 0 {
		builder = builder.Limit(filter.Limit)
	}

	if filter.Offset > 0 {
		builder = builder.Offset(filter.Offset)
	}

	query, args, errQueryBuilder := builder.ToSql()
	if errQueryBuilder != nil {
		return nil, Err(errQueryBuilder)
	}

	bans := []BannedPerson{}

	rows, errQuery := db.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			bannedPerson = NewBannedPerson()
			sourceID     int64
			targetID     int64
			steamID      int64
		)

		if errScan := rows.Scan(&bannedPerson.Ban.BanID, &targetID, &sourceID,
			&bannedPerson.Ban.BanType, &bannedPerson.Ban.Reason, &bannedPerson.Ban.ReasonText,
			&bannedPerson.Ban.Note, &bannedPerson.Ban.Origin, &bannedPerson.Ban.ValidUntil,
			&bannedPerson.Ban.CreatedOn, &bannedPerson.Ban.UpdatedOn, &bannedPerson.Ban.IncludeFriends,
			&steamID, &bannedPerson.Person.CreatedOn, &bannedPerson.Person.UpdatedOn,
			&bannedPerson.Person.CommunityVisibilityState, &bannedPerson.Person.ProfileState,
			&bannedPerson.Person.PersonaName, &bannedPerson.Person.ProfileURL, &bannedPerson.Person.Avatar,
			&bannedPerson.Person.AvatarMedium, &bannedPerson.Person.AvatarFull, &bannedPerson.Person.AvatarHash,
			&bannedPerson.Person.PersonaState, &bannedPerson.Person.RealName, &bannedPerson.Person.TimeCreated,
			&bannedPerson.Person.LocCountryCode, &bannedPerson.Person.LocStateCode, &bannedPerson.Person.LocCityID,
			&bannedPerson.Person.PermissionLevel, &bannedPerson.Person.DiscordID, &bannedPerson.Person.CommunityBanned,
			&bannedPerson.Person.VACBans, &bannedPerson.Person.GameBans, &bannedPerson.Person.EconomyBan,
			&bannedPerson.Person.DaysSinceLastBan, &bannedPerson.Ban.Deleted, &bannedPerson.Ban.ReportID,
			&bannedPerson.Ban.UnbanReasonText, &bannedPerson.Ban.IsEnabled, &bannedPerson.Ban.AppealState); errScan != nil {
			return nil, Err(errScan)
		}

		bannedPerson.Person.SteamID = steamid.New(steamID)
		bannedPerson.Ban.TargetID = steamid.New(targetID)
		bannedPerson.Ban.SourceID = steamid.New(sourceID)

		bans = append(bans, bannedPerson)
	}

	return bans, nil
}

func (db *Store) GetBansOlderThan(ctx context.Context, filter QueryFilter, since time.Time) ([]BanSteam, error) {
	query, args, queryErr := db.sb.
		Select("b.ban_id", "b.target_id", "b.source_id", "b.ban_type", "b.reason",
			"b.reason_text", "b.note", "b.origin", "b.valid_until", "b.created_on", "b.updated_on", "b.deleted",
			"case WHEN b.report_id is null THEN 0 ELSE b.report_id END", "b.unban_reason_text", "b.is_enabled",
			"b.appeal_state", "b.include_friends").
		From("ban b").
		Where(sq.And{sq.Lt{"updated_on": since}, sq.Eq{"deleted": false}}).
		Limit(filter.Limit).
		Offset(filter.Offset).
		ToSql()

	if queryErr != nil {
		return nil, Err(queryErr)
	}

	bans := []BanSteam{}

	rows, errQuery := db.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, errQuery
	}

	defer rows.Close()

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

	bannedPerson.Ban.UpdatedOn = time.Now()

	if errUpdate := db.updateBan(ctx, &bannedPerson.Ban); errUpdate != nil {
		return errUpdate
	}

	return err
}

func (db *Store) updateBanMessage(ctx context.Context, message *UserMessage) error {
	const query = `
	UPDATE ban_appeal
	SET deleted = $2, author_id = $3, updated_on = $4, message_md = $5
	WHERE ban_message_id = $1`

	message.UpdatedOn = time.Now()

	if errQuery := db.Exec(ctx, query,
		message.MessageID,
		message.Deleted,
		message.AuthorID.Int64(),
		message.UpdatedOn,
		message.Contents,
	); errQuery != nil {
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
	const query = `
	SELECT
	ban_message_id, ban_id, author_id, message_md, deleted, created_on, updated_on
	FROM ban_appeal
	WHERE deleted = false AND ban_id = $1
	ORDER BY created_on`

	rows, errQuery := db.Query(ctx, query, banID)
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
	const query = `
	SELECT
		ban_message_id, ban_id, author_id, message_md, deleted, created_on, updated_on
	FROM ban_appeal
	WHERE ban_message_id = $1`

	var authorID int64
	if errQuery := db.QueryRow(ctx, query, banMessageID).Scan(
		&message.MessageID,
		&message.ParentID,
		&authorID,
		&message.Contents,
		&message.Deleted,
		&message.CreatedOn,
		&message.UpdatedOn,
	); errQuery != nil {
		return Err(errQuery)
	}

	message.AuthorID = steamid.New(authorID)

	return nil
}

func (db *Store) DropBanMessage(ctx context.Context, message *UserMessage) error {
	const q = `UPDATE ban_appeal SET deleted = true WHERE ban_message_id = $1`
	if errExec := db.Exec(ctx, q, message.MessageID); errExec != nil {
		return Err(errExec)
	}

	db.log.Info("Appeal message deleted", zap.Int64("ban_message_id", message.MessageID))
	message.Deleted = true

	return nil
}

func (db *Store) GetBanGroup(ctx context.Context, groupID steamid.GID, banGroup *BanGroup) error {
	const query = `
	SELECT ban_group_id, source_id, target_id, group_name, is_enabled, deleted,
	note, unban_reason_text, origin, created_on, updated_on, valid_until, appeal_state, group_id
	FROM ban_group
	WHERE group_id = $1 AND deleted = false`

	var (
		sourceID   int64
		targetID   int64
		newGroupID int64
	)

	errQuery := db.QueryRow(ctx, query, groupID.Int64()).
		Scan(
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
			&newGroupID)
	if errQuery != nil {
		return Err(errQuery)
	}

	banGroup.SourceID = steamid.New(sourceID)
	banGroup.TargetID = steamid.New(targetID)
	banGroup.GroupID = steamid.NewGID(newGroupID)

	return nil
}

func (db *Store) GetBanGroupByID(ctx context.Context, banGroupID int64, banGroup *BanGroup) error {
	const query = `
	SELECT ban_group_id, source_id, target_id, group_name, is_enabled, deleted,
	note, unban_reason_text, origin, created_on, updated_on, valid_until, appeal_state, group_id
	FROM ban_group
	WHERE ban_group_id = $1 AND is_enabled = true AND deleted = false`

	var (
		groupID  int64
		sourceID int64
		targetID int64
	)

	errQuery := db.QueryRow(ctx, query, banGroupID).
		Scan(
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
			&groupID)
	if errQuery != nil {
		return Err(errQuery)
	}

	banGroup.SourceID = steamid.New(sourceID)
	banGroup.TargetID = steamid.New(targetID)
	banGroup.GroupID = steamid.NewGID(groupID)

	return nil
}

func (db *Store) GetBanGroups(ctx context.Context) ([]BanGroup, error) {
	const query = `
	SELECT ban_group_id, source_id, target_id, group_name, is_enabled, deleted,
	note, unban_reason_text, origin, created_on, updated_on, valid_until, appeal_state, group_id
	FROM ban_group
	WHERE deleted = false`

	rows, errRows := db.Query(ctx, query)
	if errRows != nil {
		return nil, Err(errRows)
	}

	defer rows.Close()

	groups := []BanGroup{}

	for rows.Next() {
		var (
			group    BanGroup
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
			&groupID); errScan != nil {
			return nil, Err(errScan)
		}

		group.SourceID = steamid.New(sourceID)
		group.TargetID = steamid.New(targetID)
		group.GroupID = steamid.NewGID(groupID)

		groups = append(groups, group)
	}

	return groups, nil
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
	query, args, errQuery := db.sb.
		Select("members_id", "parent_id", "members", "created_on", "updated_on").
		From("members").
		Where(sq.Eq{"parent_id": parentID}).
		ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}

	return Err(db.
		QueryRow(ctx, query, args...).
		Scan(&list.MembersID, &list.ParentID, &list.Members, &list.CreatedOn, &list.UpdatedOn))
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
	const query = `
	UPDATE ban_group
	SET source_id = $2, target_id = $3, group_name = $4, is_enabled = $5, deleted = $6, note = $7, unban_reason_text = $8,
	origin = $9, updated_on = $10, group_id = $11, valid_until = $12, appeal_state = $13
	WHERE ban_group_id = $1`

	banGroup.UpdatedOn = time.Now()

	return Err(db.Exec(ctx, query, banGroup.BanGroupID, banGroup.SourceID.Int64(), banGroup.TargetID.Int64(),
		banGroup.GroupName, banGroup.IsEnabled, banGroup.Deleted, banGroup.Note, banGroup.UnbanReasonText,
		banGroup.Origin, banGroup.UpdatedOn, banGroup.GroupID.Int64(), banGroup.ValidUntil, banGroup.AppealState))
}

func (db *Store) DropBanGroup(ctx context.Context, banGroup *BanGroup) error {
	banGroup.IsEnabled = false
	banGroup.Deleted = true

	return db.SaveBanGroup(ctx, banGroup)
}
