package httphelper_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/internal/opengraph"
	"github.com/stretchr/testify/require"
)

func TestCreateRouterOpenGraphInjection(t *testing.T) {
	t.Parallel()

	openGraph := opengraph.NewRepository(opengraph.Options{
		SiteName:        "gbans router test",
		SiteDescription: "router description",
		BaseURL:         "http://example.com",
	})
	openGraph.Register("/ban/{ban_id}", func(_ context.Context, _ *http.Request, params map[string]string) (opengraph.Tags, error) {
		return opengraph.Tags{Title: "Ban #" + params["ban_id"], Description: "Ban details"}, nil
	})

	mux, handler, errCreate := httphelper.CreateRouter(httphelper.RouterOpts{
		LogLevel:   log.Error,
		OpenGraph:  openGraph,
		StaticPath: t.TempDir(),
	})
	require.NoError(t, errCreate)

	htmlBody := "<html><head><title>base</title></head><body>index</body></html>"
	mux.HandleFunc("GET /ban/{ban_id}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(htmlBody))
	})

	// Crawler user agents get the tags injected into the response.
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, newBotTestRequest(t))
	require.Equal(t, http.StatusOK, res.Code)

	body := res.Body.String()
	require.Contains(t, body, `property="og:title" content="Ban #123 - gbans router test"`)
	require.Contains(t, body, "index</body>")

	// Normal user agents get the body untouched.
	res2 := httptest.NewRecorder()
	handler.ServeHTTP(res2, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ban/123", nil))

	// The buffered crawl body still contains the original content.
	require.Equal(t, http.StatusOK, res2.Code)
	require.Equal(t, htmlBody, res2.Body.String())
}

func TestCreateRouterWithoutOpenGraph(t *testing.T) {
	t.Parallel()

	_, handler, errCreate := httphelper.CreateRouter(httphelper.RouterOpts{LogLevel: log.Error}) //nolint:exhaustruct
	require.NoError(t, errCreate)

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, newBotTestRequest(t))
	require.Equal(t, http.StatusNotFound, res.Code)
	require.NotContains(t, res.Body.String(), "og:")
}

func newBotTestRequest(t *testing.T) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ban/123", nil)
	req.Header.Set("User-Agent", "Googlebot/2.1")

	return req
}
