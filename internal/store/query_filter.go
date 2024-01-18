package store

import (
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
)

// QueryFilter provides a structure for common query parameters.
type QueryFilter struct {
	Offset  uint64 `json:"offset,omitempty" uri:"offset" binding:"gte=0"`
	Limit   uint64 `json:"limit,omitempty" uri:"limit" binding:"gte=0,lte=1000"`
	Desc    bool   `json:"desc,omitempty" uri:"desc"`
	Query   string `json:"query,omitempty" uri:"query"`
	OrderBy string `json:"order_by,omitempty" uri:"order_by"`
	Deleted bool   `json:"deleted,omitempty" uri:"deleted"`
}

// applySafeOrder is used to ensure that a user requested column is valid. This
// is used to prevent potential injection attacks as there is no parameterized
// order by value.
func (qf QueryFilter) applySafeOrder(builder sq.SelectBuilder, validColumns map[string][]string, fallback string) sq.SelectBuilder {
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

func (qf QueryFilter) applyLimitOffsetDefault(builder sq.SelectBuilder) sq.SelectBuilder {
	return qf.applyLimitOffset(builder, maxResultsDefault)
}

func (qf QueryFilter) applyLimitOffset(builder sq.SelectBuilder, maxLimit uint64) sq.SelectBuilder {
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
