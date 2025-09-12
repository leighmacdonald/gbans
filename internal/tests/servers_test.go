package tests_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestServers(t *testing.T) {
	router := testRouter()
	owner := loginUser(getOwner())
	user := loginUser(getUser())

	var serversSet []servers.Server
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/servers_admin", nil, http.StatusOK, &authTokens{user: owner}, &serversSet)
	require.Len(t, serversSet, 1)

	var safeServers []servers.ServerInfoSafe
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/servers", nil, http.StatusOK, &authTokens{user: user}, &safeServers)
	require.Len(t, safeServers, 1)

	newServer := servers.RequestServerUpdate{
		ServerName:      "test-1 long",
		ServerNameShort: "test-1",
		Host:            "1.2.3.4",
		Port:            27015,
		ReservedSlots:   8,
		Password:        stringutil.SecureRandomString(8),
		RCON:            stringutil.SecureRandomString(8),
		Lat:             10,
		Lon:             10,
		CC:              "us",
		Region:          "na",
		IsEnabled:       true,
		EnableStats:     false,
		LogSecret:       12345678,
	}

	var server servers.Server
	testEndpointWithReceiver(t, router, http.MethodPost, "/api/servers", newServer, http.StatusOK, &authTokens{user: owner}, &server)

	require.Equal(t, newServer.ServerNameShort, server.ShortName)
	require.Equal(t, newServer.ServerName, server.Name)
	require.Equal(t, newServer.Host, server.Address)
	require.Equal(t, newServer.Port, server.Port)
	require.Equal(t, newServer.ReservedSlots, server.ReservedSlots)
	require.Equal(t, newServer.Password, server.Password)
	require.Equal(t, newServer.RCON, server.RCON)
	require.InEpsilon(t, newServer.Lat, server.Latitude, 0.001)
	require.InEpsilon(t, newServer.Lon, server.Longitude, 0.001)
	require.Equal(t, newServer.CC, server.CC)
	require.Equal(t, newServer.Region, server.Region)
	require.Equal(t, newServer.IsEnabled, server.IsEnabled)
	require.Equal(t, newServer.EnableStats, server.EnableStats)
	require.Equal(t, newServer.LogSecret, server.LogSecret)

	testEndpointWithReceiver(t, router, http.MethodGet, "/api/servers_admin", nil, http.StatusOK, &authTokens{user: owner}, &serversSet)
	require.Len(t, serversSet, 2)

	testEndpointWithReceiver(t, router, http.MethodGet, "/api/servers", nil, http.StatusOK, &authTokens{user: user}, &safeServers)
	require.Len(t, safeServers, 2)

	update := servers.RequestServerUpdate{
		ServerName:      "test-2 long",
		ServerNameShort: "test-2",
		Host:            "2.3.4.5",
		Port:            27016,
		ReservedSlots:   5,
		Password:        stringutil.SecureRandomString(8),
		RCON:            stringutil.SecureRandomString(8),
		Lat:             11,
		Lon:             11,
		CC:              "de",
		Region:          "eu",
		IsEnabled:       true,
		EnableStats:     true,
		LogSecret:       23456789,
	}

	var updated servers.Server
	testEndpointWithReceiver(t, router, http.MethodPost, fmt.Sprintf("/api/servers/%d", server.ServerID), update, http.StatusOK, &authTokens{user: owner}, &updated)

	require.Equal(t, update.ServerNameShort, updated.ShortName)
	require.Equal(t, update.ServerName, updated.Name)
	require.Equal(t, update.Host, updated.Address)
	require.Equal(t, update.Port, updated.Port)
	require.Equal(t, update.ReservedSlots, updated.ReservedSlots)
	require.Equal(t, update.Password, updated.Password)
	require.Equal(t, update.RCON, updated.RCON)
	require.InEpsilon(t, update.Lat, updated.Latitude, 0.001)
	require.InEpsilon(t, update.Lon, updated.Longitude, 0.001)
	require.Equal(t, update.CC, updated.CC)
	require.Equal(t, update.Region, updated.Region)
	require.Equal(t, update.IsEnabled, updated.IsEnabled)
	require.Equal(t, update.EnableStats, updated.EnableStats)
	require.Equal(t, update.LogSecret, updated.LogSecret)

	testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/servers/%d", server.ServerID), nil, http.StatusOK, &authTokens{user: owner})
	testEndpoint(t, router, http.MethodDelete, fmt.Sprintf("/api/servers/%d", server.ServerID), nil, http.StatusNotFound, &authTokens{user: owner})
	testEndpoint(t, router, http.MethodDelete, "/api/servers/xx", nil, http.StatusBadRequest, &authTokens{user: owner})
}

func TestServersPermissions(t *testing.T) {
	testPermissions(t, testRouter(), []permTestValues{
		{
			path:   "/api/servers",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: admin,
		},
		{
			path:   "/api/servers/1",
			method: http.MethodPost,
			code:   http.StatusForbidden,
			levels: admin,
		},
		{
			path:   "/api/servers/1",
			method: http.MethodDelete,
			code:   http.StatusForbidden,
			levels: admin,
		},
		{
			path:   "/api/servers_admin",
			method: http.MethodGet,
			code:   http.StatusForbidden,
			levels: admin,
		},
	})
}
