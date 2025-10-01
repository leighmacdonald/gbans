package servers_test

import (
	"os"
	"testing"

	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

var fixture *tests.Fixture

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	os.Exit(m.Run())
}

func TestServers(t *testing.T) {
	serversCase := servers.NewServers(servers.NewRepository(fixture.Database))

	// no results yet
	noServers, errServers := serversCase.Servers(t.Context(), servers.Query{})
	require.NoError(t, errServers)
	require.Equal(t, []servers.Server{}, noServers)

	// Add a server
	newServer := servers.NewServer(stringutil.SecureRandomString(10), stringutil.SecureRandomString(10)+".com", 27015)
	saved, errSave := serversCase.Save(t.Context(), newServer)
	require.NoError(t, errSave)
	require.Positive(t, saved.ServerID)
	require.Equal(t, newServer.ShortName, saved.ShortName)
	require.Equal(t, newServer.Address, saved.Address)
	require.Equal(t, newServer.Port, saved.Port)

	// Add a second server
	otherServer, errSave := serversCase.Save(t.Context(), servers.NewServer(stringutil.SecureRandomString(10), stringutil.SecureRandomString(10)+".com", 27015))
	require.NoError(t, errSave)

	// Query them all
	serversAll, errServers2 := serversCase.Servers(t.Context(), servers.Query{})
	require.Len(t, serversAll, 2)
	require.NoError(t, errServers2)

	// Delete one
	require.NoError(t, serversCase.Delete(t.Context(), otherServer.ServerID))

	_, deletedErr := serversCase.Server(t.Context(), otherServer.ServerID)
	require.ErrorIs(t, servers.ErrNotFound, deletedErr)

	// Query all
	serversDeleted, errServers3 := serversCase.Servers(t.Context(), servers.Query{})
	require.Len(t, serversDeleted, 1)
	require.NoError(t, errServers3)

	// Query all including soft-deleted
	serversAllDeleted, errServers4 := serversCase.Servers(t.Context(), servers.Query{IncludeDeleted: true})
	require.Len(t, serversAllDeleted, 2)
	require.NoError(t, errServers4)

	byPass, _ := serversCase.GetByPassword(t.Context(), saved.Password)
	require.Equal(t, saved, byPass)

	byName, _ := serversCase.GetByName(t.Context(), saved.ShortName)
	require.Equal(t, saved, byName)
}
