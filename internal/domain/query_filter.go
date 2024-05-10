package domain

import (
	"context"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const MaxResultsDefault = 100

// QueryFilter provides a structure for common query parameters.
type QueryFilter struct {
	Offset  uint64 `json:"offset,omitempty" uri:"offset" binding:"gte=0"`
	Limit   uint64 `json:"limit,omitempty" uri:"limit" binding:"gte=0,lte=10000"`
	Desc    bool   `json:"desc,omitempty" uri:"desc"`
	Query   string `json:"query,omitempty" uri:"query"`
	OrderBy string `json:"order_by,omitempty" uri:"order_by"`
	Deleted bool   `json:"deleted,omitempty" uri:"deleted"`
}

// ApplySafeOrder is used to ensure that a user requested column is valid. This
// is used to prevent potential injection attacks as there is no parameterized
// order by value.
func (qf QueryFilter) ApplySafeOrder(builder sq.SelectBuilder, validColumns map[string][]string, fallback string) sq.SelectBuilder {
	if qf.OrderBy == "" {
		qf.OrderBy = fallback
	}

	qf.OrderBy = strings.ToLower(qf.OrderBy)

	isValid := false

	for prefix, columns := range validColumns {
		for _, name := range columns {
			if name == qf.OrderBy {
				qf.OrderBy = prefix + qf.OrderBy
				isValid = true

				break
			}
		}

		if isValid {
			break
		}
	}

	if qf.Desc {
		builder = builder.OrderBy(fmt.Sprintf("%s DESC", qf.OrderBy))
	} else {
		builder = builder.OrderBy(fmt.Sprintf("%s ASC", qf.OrderBy))
	}

	return builder
}

func (qf QueryFilter) ApplyLimitOffsetDefault(builder sq.SelectBuilder) sq.SelectBuilder {
	return qf.ApplyLimitOffset(builder, MaxResultsDefault)
}

func (qf QueryFilter) ApplyLimitOffset(builder sq.SelectBuilder, maxLimit uint64) sq.SelectBuilder {
	if qf.Limit > maxLimit {
		qf.Limit = maxLimit
	}

	if qf.Limit > 0 {
		builder = builder.Limit(qf.Limit)
	}

	if qf.Offset > 0 {
		builder = builder.Offset(qf.Offset)
	}

	return builder
}

type NotificationQuery struct {
	QueryFilter
	SteamID string `json:"steam_id"`
}

func (f NotificationQuery) SourceSteamID() (steamid.SteamID, bool) {
	sid := steamid.New(f.SteamID)

	return sid, sid.Valid()
}

type ChatHistoryQueryFilter struct {
	QueryFilter
	SourceIDField
	Personaname   string     `json:"personaname,omitempty"`
	ServerID      int        `json:"server_id,omitempty"`
	DateStart     *time.Time `json:"date_start,omitempty"`
	DateEnd       *time.Time `json:"date_end,omitempty"`
	Unrestricted  bool       `json:"-"`
	DontCalcTotal bool       `json:"-"`
	FlaggedOnly   bool       `json:"flagged_only"`
}

func (f ChatHistoryQueryFilter) SourceSteamID() (steamid.SteamID, bool) {
	sid := steamid.New(f.SourceID)

	return sid, sid.Valid()
}

type ConnectionHistoryQuery struct {
	QueryFilter
	SourceIDField
	CIDR    string `json:"cidr"`
	ASN     int    `json:"asn"`
	Sid64   int64
	Network string
}

type PlayerQuery struct {
	QueryFilter
	TargetIDField
	Personaname string `json:"personaname"`
	IP          string `json:"ip"`
	StaffOnly   bool   `json:"staff_only"`
}

type DemoFilter struct {
	QueryFilter
	SteamID   string `json:"steam_id"`
	ServerIds []int  `json:"server_ids"`
	MapName   string `json:"map_name"`
}

func (f DemoFilter) SourceSteamID() (steamid.SteamID, bool) {
	sid := steamid.New(f.SteamID)

	return sid, sid.Valid()
}

type FiltersQueryFilter struct {
	QueryFilter
}

type ThreadMessagesQuery struct {
	Deleted       bool  `json:"deleted,omitempty" uri:"deleted"`
	ForumThreadID int64 `json:"forum_thread_id"`
}

type ThreadQueryFilter struct {
	ForumID int `json:"forum_id"`
}

type MatchesQueryOpts struct {
	QueryFilter
	SteamID   string     `json:"steam_id"`
	ServerID  int        `json:"server_id"`
	Map       string     `json:"map"`
	TimeStart *time.Time `json:"time_start,omitempty"`
	TimeEnd   *time.Time `json:"time_end,omitempty"`
}

func (mqf MatchesQueryOpts) TargetSteamID() (steamid.SteamID, bool) {
	sid := steamid.New(mqf.SteamID)

	return sid, sid.Valid()
}

type SourceIDProvider interface {
	SourceSteamID(context.Context) (steamid.SteamID, bool)
}

type TargetIDProvider interface {
	TargetSteamID(context.Context) (steamid.SteamID, bool)
}

type BansQueryFilter struct {
	QueryFilter
	SourceIDField
	TargetIDField
	Reason        Reason `json:"reason,omitempty"`
	PermanentOnly bool   `json:"permanent_only,omitempty"`
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
	// also banned while the primary targets ban has not Expired.
	IncludeFriendsOnly bool        `json:"include_friends_only"`
	AppealState        AppealState `json:"appeal_state"`
}

type ReportQueryFilter struct {
	QueryFilter
	SourceIDField
	TargetIDField
	ReportStatus ReportStatus `json:"report_status"`
}

type AppealQueryFilter struct {
	QueryFilter
	SourceIDField
	TargetIDField
	AppealState AppealState `json:"appeal_state"`
}

type TargetID struct {
	SteamID string `json:"steam_id"`
}

func (f TargetID) SteamSteamID(ctx context.Context) (steamid.SteamID, bool) {
	if f.SteamID == "" {
		return steamid.SteamID{}, false
	}

	sid, err := steamid.Resolve(ctx, f.SteamID)
	if err != nil {
		return sid, false
	}

	return sid, sid.Valid()
}

type SourceIDField struct {
	SourceID string `json:"source_id"`
}

func (f SourceIDField) SourceSteamID(ctx context.Context) (steamid.SteamID, bool) {
	if f.SourceID == "" {
		return steamid.SteamID{}, false
	}

	sid, err := steamid.Resolve(ctx, f.SourceID)
	if err != nil {
		return sid, false
	}

	return sid, sid.Valid()
}

type TargetIDField struct {
	TargetID string `json:"target_id"`
}

func (f TargetIDField) TargetSteamID(ctx context.Context) (steamid.SteamID, bool) {
	if f.TargetID == "" {
		return steamid.SteamID{}, false
	}

	sid, err := steamid.Resolve(ctx, f.TargetID)
	if err != nil {
		return sid, false
	}

	return sid, sid.Valid()
}

type TargetGIDField struct {
	GroupID string `json:"group_id"`
}

func (f TargetGIDField) TargetGroupID(ctx context.Context) (steamid.SteamID, bool) {
	sid, err := steamid.Resolve(ctx, f.GroupID)
	if err != nil {
		return sid, false
	}

	return sid, sid.Valid()
}
