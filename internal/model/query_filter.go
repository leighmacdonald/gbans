package model

import (
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

const MaxResultsDefault = 100

// QueryFilter provides a structure for common query parameters.
type QueryFilter struct {
	Offset  uint64 `json:"offset,omitempty" uri:"offset" binding:"gte=0"`
	Limit   uint64 `json:"limit,omitempty" uri:"limit" binding:"gte=0,lte=1000"`
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
	SteamID steamid.SID64 `json:"steam_id"`
}

type ChatHistoryQueryFilter struct {
	QueryFilter
	Personaname   string     `json:"personaname,omitempty"`
	SourceID      StringSID  `json:"source_id,omitempty"`
	ServerID      int        `json:"server_id,omitempty"`
	DateStart     *time.Time `json:"date_start,omitempty"`
	DateEnd       *time.Time `json:"date_end,omitempty"`
	Unrestricted  bool       `json:"-"`
	DontCalcTotal bool       `json:"-"`
	FlaggedOnly   bool       `json:"flagged_only"`
}

type ConnectionHistoryQueryFilter struct {
	QueryFilter
	IP       string    `json:"ip"`
	SourceID StringSID `json:"source_id"`
}

type PlayerQuery struct {
	QueryFilter
	SteamID     StringSID `json:"steam_id"`
	Personaname string    `json:"personaname"`
	IP          string    `json:"ip"`
}

type DemoFilter struct {
	QueryFilter
	SteamID   StringSID `json:"steam_id"`
	ServerIds []int     `json:"server_ids"`
	MapName   string    `json:"map_name"`
}

type FiltersQueryFilter struct {
	QueryFilter
}

type ThreadMessagesQueryFilter struct {
	QueryFilter
	ForumThreadID int64 `json:"forum_thread_id"`
}

type ThreadQueryFilter struct {
	QueryFilter
	ForumID int `json:"forum_id"`
}

type MatchesQueryOpts struct {
	QueryFilter
	SteamID   steamid.SID64 `json:"steam_id"`
	ServerID  int           `json:"server_id"`
	Map       string        `json:"map"`
	TimeStart *time.Time    `json:"time_start,omitempty"`
	TimeEnd   *time.Time    `json:"time_end,omitempty"`
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
	// also banned while the primary targets ban has not Expired.
	IncludeFriendsOnly bool        `json:"include_friends_only"`
	AppealState        AppealState `json:"appeal_state"`
}

type ReportQueryFilter struct {
	QueryFilter
	ReportStatus ReportStatus `json:"report_status"`
	SourceID     StringSID    `json:"source_id"`
	TargetID     StringSID    `json:"target_id"`
}
type AppealQueryFilter struct {
	QueryFilter
	AppealState AppealState `json:"appeal_state"`
	SourceID    StringSID   `json:"source_id"`
	TargetID    StringSID   `json:"target_id"`
}
