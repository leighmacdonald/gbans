package patreon_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/authn"
	"github.com/leighmacdonald/gbans/internal/patreon"
	"github.com/leighmacdonald/gbans/internal/patreon/v1/patreonv1connect"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

type stubResolver struct{}

func (stubResolver) PermissionsBySteamID(_ context.Context, _ steamid.SteamID) []rolesv1.Permission {
	return nil
}

func newTestServer(t *testing.T, patreonSvc patreon.Patreon, authFunc func(context.Context, *http.Request) (any, error)) (patreonv1connect.PatreonServiceClient, func()) {
	t.Helper()

	middleware := rpc.NewMiddleware("gbans-test", "test-cookie-secret", stubResolver{})
	svc := patreon.NewService(patreonSvc, middleware)

	mux := http.NewServeMux()
	mux.Handle(svc.Pattern, svc.Handler)

	server := httptest.NewServer(authn.NewMiddleware(authFunc).Wrap(mux))

	return patreonv1connect.NewPatreonServiceClient(server.Client(), server.URL), server.Close
}

func authedAs(sid steamid.SteamID) func(context.Context, *http.Request) (any, error) {
	return func(_ context.Context, _ *http.Request) (any, error) {
		return rpc.UserInfo{SteamID: sid}, nil
	}
}

func TestPatronCampaignsEmpty(t *testing.T) {
	client, cleanup := newTestServer(t,
		patreon.NewPatreon(patreon.NewRepository(nil), &patreon.Config{}),
		authedAs(steamid.RandSID64()))
	defer cleanup()

	resp, err := client.PatronCampaigns(t.Context(), &emptypb.Empty{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp.GetCampaigns())
}

func TestPatronLogin(t *testing.T) {
	sid := steamid.RandSID64()
	client, cleanup := newTestServer(t,
		patreon.NewPatreon(patreon.NewRepository(nil), &patreon.Config{ClientID: "client-id"}),
		authedAs(sid))
	defer cleanup()

	resp, err := client.PatronLogin(t.Context(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Contains(t, resp.GetUrl(), "https://www.patreon.com/oauth2/authorize")
	require.Contains(t, resp.GetUrl(), "client_id=client-id")
}

func TestPatronLogout(t *testing.T) {
	fixture := tests.NewFixture()
	defer fixture.Close()

	sid := steamid.RandSID64()
	client, cleanup := newTestServer(t,
		patreon.NewPatreon(patreon.NewRepository(fixture.Database), &patreon.Config{}),
		authedAs(sid))
	defer cleanup()

	_, err := client.PatronLogout(t.Context(), &emptypb.Empty{})
	require.NoError(t, err)
}
