package servers_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestPurgeLogsKeepsLatestEntriesPerServer(t *testing.T) {
	repo := servers.NewRepository(fixture.Database)
	svc, errSvc := servers.New(repo, nil, "")
	require.NoError(t, errSvc)

	ctx := t.Context()

	bigServer, errSave := svc.Save(ctx, servers.NewServer(stringutil.SecureRandomString(10), "10.0.0.1", 27015))
	require.NoError(t, errSave)

	smallServer, errSave := svc.Save(ctx, servers.NewServer(stringutil.SecureRandomString(10), "10.0.0.2", 27015))
	require.NoError(t, errSave)

	// Other tests in this package assume the fixture starts with no servers.
	// t.Context() is already canceled when cleanup functions run, so use a fresh one.
	t.Cleanup(func() {
		cleanCtx := context.Background()
		require.NoError(t, fixture.Database.Exec(cleanCtx,
			`DELETE FROM server_logs WHERE server_id = $1 OR server_id = $2`, bigServer.ServerID, smallServer.ServerID))
		require.NoError(t, fixture.Database.Exec(cleanCtx,
			`DELETE FROM server WHERE server_id = $1 OR server_id = $2`, bigServer.ServerID, smallServer.ServerID))
	})

	const keep = 10000

	base := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

	events := make([]logparse.ServerEvent, 0, keep+5)
	for i := range keep + 5 {
		events = append(events, logparse.ServerEvent{
			ServerID:   bigServer.ServerID,
			ServerName: bigServer.ShortName,
			Raw:        fmt.Sprintf("line %05d", i),
			CreatedOn:  base.Add(time.Duration(i) * time.Second),
		})
	}
	require.NoError(t, repo.InsertLogs(ctx, events))

	smallEvents := []logparse.ServerEvent{
		{ServerID: smallServer.ServerID, ServerName: smallServer.ShortName, Raw: "a", CreatedOn: base},
		{ServerID: smallServer.ServerID, ServerName: smallServer.ShortName, Raw: "b", CreatedOn: base.Add(time.Second)},
		{ServerID: smallServer.ServerID, ServerName: smallServer.ShortName, Raw: "c", CreatedOn: base.Add(2 * time.Second)},
	}
	require.NoError(t, repo.InsertLogs(ctx, smallEvents))

	require.NoError(t, repo.PurgeLogs(ctx))

	var remaining int64
	require.NoError(t, fixture.Database.QueryRow(ctx,
		`SELECT count(*) FROM server_logs WHERE server_id = $1`, bigServer.ServerID).Scan(&remaining))
	require.EqualValues(t, keep, remaining, "purge must keep exactly the latest 10000 lines")

	// Exactly the 10000 most recent lines must survive: "line 00005" through "line 10004".
	var oldestLine, newestLine string
	require.NoError(t, fixture.Database.QueryRow(ctx,
		`SELECT min(body), max(body) FROM server_logs WHERE server_id = $1`, bigServer.ServerID).Scan(&oldestLine, &newestLine))
	require.Equal(t, "line 00005", oldestLine, "oldest kept line must be the first line outside the keep window")
	require.Equal(t, "line 10004", newestLine, "newest kept line must be the most recent line")

	// Servers with fewer lines than the keep window are left untouched.
	var smallRemaining int64
	require.NoError(t, fixture.Database.QueryRow(ctx,
		`SELECT count(*) FROM server_logs WHERE server_id = $1`, smallServer.ServerID).Scan(&smallRemaining))
	require.EqualValues(t, len(smallEvents), smallRemaining)
}
