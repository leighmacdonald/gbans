package servers_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestServers(t *testing.T) {
	testDB := tests.NewFixture()
	defer testDB.Close()

	serversCase := servers.NewServers(servers.NewRepository(testDB.Database))

	// no results yet
	noServers, errServers := serversCase.Servers(t.Context(), servers.Query{})
	require.NoError(t, errServers)
	require.Equal(t, []servers.Server{}, noServers)

	// Add a server
	newServer := servers.NewServer(stringutil.SecureRandomString(10), stringutil.SecureRandomString(10)+".com", 27015)
	saved, errSave := serversCase.Save(t.Context(), newServer)
	require.NoError(t, errSave)
	require.True(t, saved.ServerID > 0)
	require.Equal(t, newServer.ShortName, saved.ShortName)
	require.Equal(t, newServer.Address, saved.Address)
	require.Equal(t, newServer.Port, saved.Port)

	// Add a second server
	otherServer, errSave := serversCase.Save(t.Context(), servers.NewServer(stringutil.SecureRandomString(10), stringutil.SecureRandomString(10)+".com", 27015))
	require.NoError(t, errSave)

	// Query them all
	serversAll, errServers := serversCase.Servers(t.Context(), servers.Query{})
	require.Len(t, serversAll, 2)

	// Delete one
	require.NoError(t, serversCase.Delete(t.Context(), otherServer.ServerID))

	_, deletedErr := serversCase.Server(t.Context(), otherServer.ServerID)
	require.ErrorIs(t, servers.ErrNotFound, deletedErr)

	// Query all
	serversDeleted, errServers := serversCase.Servers(t.Context(), servers.Query{})
	require.Len(t, serversDeleted, 1)

	// Query all including soft-deleted
	serversAllDeleted, errServers := serversCase.Servers(t.Context(), servers.Query{IncludeDeleted: true})
	require.Len(t, serversAllDeleted, 2)

	byPass, _ := serversCase.GetByPassword(t.Context(), saved.Password)
	require.EqualValues(t, saved, byPass)

	byName, _ := serversCase.GetByName(t.Context(), saved.ShortName)
	require.EqualValues(t, saved, byName)
}
