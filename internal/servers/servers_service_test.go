package servers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/authn"
	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/opengraph"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/internal/servers"
	v1 "github.com/leighmacdonald/gbans/internal/servers/v1"
	"github.com/leighmacdonald/gbans/internal/servers/v1/serversv1connect"
	"github.com/leighmacdonald/gbans/pkg/broadcaster"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

type testPermissionResolver struct {
	perms []rolesv1.Permission
}

func (r testPermissionResolver) PermissionsBySteamID(context.Context, steamid.SteamID) []rolesv1.Permission {
	return r.perms
}

// authRoundTripper injects the bearer token and fingerprint cookie into every
// outgoing request, which covers both unary and streaming calls.
type authRoundTripper struct {
	token       string
	fingerprint string
}

func (h authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+h.token)
	req.Header.Set("Cookie", rpc.FingerprintCookieName+"="+h.fingerprint)

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("round trip: %w", err)
	}

	return resp, nil
}

func TestServersServiceStreamLogs(t *testing.T) {
	serverCase, err := servers.New(servers.NewRepository(fixture.Database),
		broadcaster.New[logparse.EventType, logparse.ServerEvent](), "")
	require.NoError(t, err)

	saved, err := serverCase.Save(t.Context(), servers.NewServer("stream-logs", "stream-logs.example.com", 27015))
	require.NoError(t, err)

	// The fixture database is shared by all tests in this package.
	t.Cleanup(func() {
		require.NoError(t, serverCase.Delete(context.Background(), saved.ServerID))
	})

	const history = 3
	for n := 1; n <= history; n++ {
		serverCase.LogBuffer().Push(saved.ServerID, makeEvent(saved.ServerID, n))
	}

	resolver := testPermissionResolver{perms: []rolesv1.Permission{rolesv1.Permission_PERMISSION_SERVER_READ}}
	authMiddleware := rpc.NewMiddleware("test-site", "test-cookie-secret", resolver)
	roleAuth := rpc.NewRoleAuth(resolver)

	svc := servers.NewServersService(serverCase, &network.Networks{},
		opengraph.NewRepository(opengraph.Options{}), roleAuth, authMiddleware)

	apiMux := http.NewServeMux()
	apiMux.Handle(svc.Pattern, svc.Handler)
	testServer := httptest.NewServer(authn.NewMiddleware(authMiddleware.Authenticate).Wrap(apiMux))
	t.Cleanup(testServer.Close)

	user := person.Core{SteamID: steamid.New(76561198000000001), Name: "Tester"}
	token, fingerprint, err := authMiddleware.MakeUserToken(user)
	require.NoError(t, err)

	client := serversv1connect.NewServersServiceClient(
		&http.Client{Transport: authRoundTripper{token: token, fingerprint: fingerprint}}, testServer.URL)

	t.Run("streams history and live lines", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()

		stream, errStream := client.StreamLogs(ctx, &v1.StreamLogsRequest{ServerId: &saved.ServerID})
		require.NoError(t, errStream)

		for n := 1; n <= history; n++ {
			require.True(t, stream.Receive(), "expected line %d", n)
			msg := stream.Msg()
			require.Contains(t, msg.GetBody(), fmt.Sprintf("line %d", n))
			require.Equal(t, saved.ServerID, msg.GetServerId())
			require.Equal(t, "stream-logs", msg.GetServerName())
			require.Equal(t, v1.LogEventType_LOG_EVENT_TYPE_SAY, msg.GetEventType())
		}

		serverCase.LogBuffer().Push(saved.ServerID, makeEvent(saved.ServerID, 4))
		require.True(t, stream.Receive())
		require.Contains(t, stream.Msg().GetBody(), "line 4")
	})

	t.Run("history limit trims the buffer", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()

		limit := int32(2)
		stream, errStream := client.StreamLogs(ctx, &v1.StreamLogsRequest{
			ServerId:     &saved.ServerID,
			HistoryLimit: &limit,
		})
		require.NoError(t, errStream)

		// The buffer now holds 4 lines, only the last 2 are sent as history.
		for _, wantLine := range []string{"line 3", "line 4"} {
			require.True(t, stream.Receive(), "expected %s", wantLine)
			require.Contains(t, stream.Msg().GetBody(), wantLine)
		}
	})

	t.Run("unknown server is not found", func(t *testing.T) {
		missing := int32(999999)
		stream, errStream := client.StreamLogs(t.Context(), &v1.StreamLogsRequest{ServerId: &missing})
		require.NoError(t, errStream)

		require.False(t, stream.Receive())

		var connectErr *connect.Error
		require.ErrorAs(t, stream.Err(), &connectErr)
		require.Equal(t, connect.CodeNotFound, connectErr.Code())
	})

	t.Run("unauthenticated request is rejected", func(t *testing.T) {
		unauthClient := serversv1connect.NewServersServiceClient(http.DefaultClient, testServer.URL)

		stream, errStream := unauthClient.StreamLogs(t.Context(), &v1.StreamLogsRequest{ServerId: &saved.ServerID})
		require.NoError(t, errStream)

		require.False(t, stream.Receive())
		require.Error(t, stream.Err())
	})
}
