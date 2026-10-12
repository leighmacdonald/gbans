package servers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestConnectRedirect(t *testing.T) {
	serversCase, _ := servers.New(servers.NewRepository(fixture.Database), nil, "")

	mux := http.NewServeMux()
	servers.RegisterConnectHandler(mux, serversCase)

	newServer := servers.NewServer(stringutil.SecureRandomString(10), "127.0.0.1", 27015)
	saved, errSave := serversCase.Save(t.Context(), newServer)
	require.NoError(t, errSave)
	t.Cleanup(func() {
		require.NoError(t, serversCase.Delete(context.Background(), saved.ServerID))
	})

	t.Run("redirects to steam link", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/connect/"+strconv.Itoa(int(saved.ServerID)), nil))

		resp := rec.Result()
		defer resp.Body.Close()
		require.Equal(t, http.StatusFound, resp.StatusCode)
		require.Equal(t, "steam://run/440//+connect 127.0.0.1:27015", resp.Header.Get("Location"))
	})

	t.Run("invalid ids 404", func(t *testing.T) {
		for _, target := range []string{"/connect/0", "/connect/-5", "/connect/nope", "/connect/2147483647"} {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
			require.Equal(t, http.StatusNotFound, rec.Result().StatusCode, target)
		}
	})
}
