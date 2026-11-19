package query_test

import (
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/stretchr/testify/require"
)

func TestApplySafeOrder(t *testing.T) {
	t.Parallel()

	valid := map[string][]string{
		"m.":  {"duration_ms", "start_time"},
		"mp.": {"map_name"},
	}

	builder := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).Select("id").From("match m")

	gotValid, _, errValid := query.Filter{OrderBy: "duration_ms", Desc: true}.
		ApplySafeOrder(builder, valid, "m.start_time").ToSql()
	require.NoError(t, errValid)
	require.Equal(t, "SELECT id FROM match m ORDER BY m.duration_ms DESC", gotValid)

	gotAliased, _, errAliased := query.Filter{OrderBy: "map_name"}.
		ApplySafeOrder(builder, valid, "m.start_time").ToSql()
	require.NoError(t, errAliased)
	require.Equal(t, "SELECT id FROM match m ORDER BY mp.map_name ASC", gotAliased)

	// Unknown columns must never reach ORDER BY; they fall back to a known
	// column instead of producing a SQL error (or an injection vector).
	gotFallback, _, errFallback := query.Filter{OrderBy: "total_duration_ms", Desc: true}.
		ApplySafeOrder(builder, valid, "m.start_time").ToSql()
	require.NoError(t, errFallback)
	require.Equal(t, "SELECT id FROM match m ORDER BY m.start_time DESC", gotFallback)
}
