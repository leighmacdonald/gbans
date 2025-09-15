package query

import (
	"slices"
	"strings"

	sq "github.com/Masterminds/squirrel"
)

const MaxResultsDefault = 100

// Filter provides a structure for common query parameters.
type Filter struct {
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
func (qf Filter) ApplySafeOrder(builder sq.SelectBuilder, validColumns map[string][]string, fallback string) sq.SelectBuilder {
	if qf.OrderBy == "" {
		qf.OrderBy = fallback
	}

	qf.OrderBy = strings.ToLower(qf.OrderBy)

	isValid := false

	for prefix, columns := range validColumns {
		if slices.Contains(columns, qf.OrderBy) {
			qf.OrderBy = prefix + qf.OrderBy
			isValid = true
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

func (qf Filter) ApplyLimitOffsetDefault(builder sq.SelectBuilder) sq.SelectBuilder {
	return qf.ApplyLimitOffset(builder, MaxResultsDefault)
}

func (qf Filter) ApplyLimitOffset(builder sq.SelectBuilder, maxLimit uint64) sq.SelectBuilder {
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
