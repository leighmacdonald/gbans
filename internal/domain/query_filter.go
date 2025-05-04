package domain

import (
	"context"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const MaxResultsDefault = 100

// QueryFilter provides a structure for common query parameters.
type QueryFilter struct {
	Offset  uint64 `json:"offset,omitempty" schema:"offset" binding:"gte=0"`
	Limit   uint64 `json:"limit,omitempty" schema:"limit" binding:"gte=0,lte=10000"`
	Desc    bool   `json:"desc,omitempty" schema:"desc"`
	Query   string `json:"query,omitempty" schema:"query"`
	OrderBy string `json:"order_by,omitempty" schema:"order_by"`
	Deleted bool   `json:"deleted,omitempty" schema:"deleted"`
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
		builder = builder.OrderBy(qf.OrderBy + " DESC")
	} else {
		builder = builder.OrderBy(qf.OrderBy + " ASC")
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
	CIDR    string `json:"cidr,omitempty"`
	ASN     int    `json:"asn,omitempty"`
	Sid64   int64  `json:"sid64,omitempty"`
	Network string `json:"network,omitempty"`
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
	ServerIDs []int  `json:"server_ids"` //nolint:tagliatelle
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
	SourceSteamID(ctx context.Context) (steamid.SteamID, bool)
}

type TargetIDProvider interface {
	TargetSteamID(ctx context.Context) (steamid.SteamID, bool)
}

type BansQueryFilter struct {
	Deleted bool `json:"deleted"`
}

type CIDRBansQueryFilter struct {
	Deleted bool `json:"deleted"`
}

type ASNBansQueryFilter struct {
	Deleted bool `json:"deleted"`
}

type GroupBansQueryFilter struct {
	Deleted bool `json:"deleted"`
}

type SteamBansQueryFilter struct {
	TargetIDField
	Deleted bool `schema:"deleted"`
}

type ReportQueryFilter struct {
	SourceIDField
	Deleted bool `json:"deleted"`
}

type AppealQueryFilter struct {
	Deleted bool `json:"deleted"`
}

type SteamIDField struct {
	SteamIDValue string `json:"steam_id"  url:"steam_id"` //nolint:tagliatelle
}

func (f SteamIDField) SteamID(ctx context.Context) (steamid.SteamID, bool) {
	if f.SteamIDValue == "" {
		return steamid.SteamID{}, false
	}

	sid, err := steamid.Resolve(ctx, f.SteamIDValue)
	if err != nil {
		return sid, false
	}

	return sid, sid.Valid()
}

type SourceIDField struct {
	SourceID string `json:"source_id"  url:"source_id"`
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
	TargetID string `json:"target_id" url:"target_id"`
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
	GroupID string `json:"group_id"  url:"group_id"`
}

func (f TargetGIDField) TargetGroupID(ctx context.Context) (steamid.SteamID, bool) {
	sid, err := steamid.Resolve(ctx, f.GroupID)
	if err != nil {
		return sid, false
	}

	return sid, sid.Valid()
}
