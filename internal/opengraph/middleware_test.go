package opengraph_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/leighmacdonald/gbans/internal/opengraph"
	"github.com/stretchr/testify/require"
)

// nolint:gochecknoglobals
var errForced = errors.New("forced tags failure")

func newTestRepository() *opengraph.Repository {
	return opengraph.NewRepository(opengraph.Options{
		SiteName:        "gbans test",
		SiteDescription: "test description",
		BaseURL:         "http://example.com",
		Image:           "https://example.com/banner.webp",
	})
}

func newBotRequest(testCtx context.Context, path string) *http.Request {
	req := httptest.NewRequestWithContext(testCtx, http.MethodGet, path, nil)
	req.Header.Set("User-Agent", "Discordbot/2.0")

	return req
}

func htmlHandler(body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(body))
	})
}

func TestCrawlerUserAgents(t *testing.T) {
	t.Parallel()

	repo := newTestRepository()
	handler := htmlHandler("<html><head></head><body>content</body></html>")

	tests := map[string]bool{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/100.0": false,
		"":               false,
		"Twitterbot/1.0": true,
		"Mozilla/5.0 (compatible; Discordbot/2.0; +https://discordapp.com)":         true,
		"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)":  true,
		"facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)": true,
		"Slackbot-LinkExpanding 1.0":              true,
		"TelegramBot (like TwitterBot)":           true,
		"Google-Apps-Chrome-OS x GoldfishCracker": false,
		"JetBrains-Toolbox-App":                   false,
	}

	for userAgent, expectInjection := range tests {
		t.Run(userAgent, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/anywhere", nil)
			req.Header.Set("User-Agent", userAgent)

			res := httptest.NewRecorder()
			repo.Middleware(handler).ServeHTTP(res, req)

			if expectInjection {
				require.Contains(t, res.Body.String(), "og:description")
			} else {
				require.Equal(t, "<html><head></head><body>content</body></html>", res.Body.String())
			}
		})
	}
}

func TestMiddlewareInjection(t *testing.T) {
	t.Parallel()

	repo := newTestRepository()
	repo.Register("/ban/{ban_id}", func(_ context.Context, _ *http.Request, params map[string]string) (opengraph.Tags, error) {
		return opengraph.Tags{Title: "Ban #" + params["ban_id"], Description: "Ban details"}, nil
	})

	handler := repo.Middleware(htmlHandler("<html><head><title>base</title></head><body>content</body></html>"))

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, newBotRequest(t.Context(), "/ban/100"))

	require.Equal(t, http.StatusOK, res.Code)

	body := res.Body.String()
	require.Contains(t, body, `property="og:title" content="Ban #100 - gbans test"`)
	require.Contains(t, body, `property="og:site_name" content="gbans test"`)
	require.Contains(t, body, `content="http://example.com/ban/100"`)
	require.Contains(t, body, `content="https://example.com/banner.webp"`)
	require.Contains(t, body, `name="twitter:card" content="summary_large_image"`)
	require.Contains(t, body, `property="og:description" content="Ban details"`)
	require.Contains(t, body, "content</body>")
}

func TestMiddlewareUserPassthrough(t *testing.T) {
	t.Parallel()

	repo := newTestRepository()
	body := "<html><head><title>base</title></head><body>unused</body></html>"
	handler := repo.Middleware(htmlHandler(body))

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/ban/100", nil))

	require.Equal(t, body, res.Body.String())
}

func TestMiddlewareHeadRequestPassthrough(t *testing.T) {
	t.Parallel()

	repo := newTestRepository()
	handler := htmlHandler("<html><head></head><body>unused</body></html>")

	res := httptest.NewRecorder()
	repo.Middleware(handler).ServeHTTP(
		res,
		func() *http.Request {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodHead, "/ban/100", nil)
			req.Header.Set("User-Agent", "Slackbot-LinkExpanding")

			return req
		}())

	// The body is written for HEAD requests and discarded by the net/http
	// server itself, but the tags are still resolved and injected.
	require.Contains(t, res.Body.String(), "og:title")
}

func TestMiddlewareNonHTMLPassthrough(t *testing.T) {
	t.Parallel()

	repo := newTestRepository()
	jsonHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1}`))
	})

	res := httptest.NewRecorder()
	repo.Middleware(jsonHandler).ServeHTTP(res, newBotRequest(t.Context(), "/api/1"))

	require.Equal(t, `{"id":1}`, res.Body.String())
}

func TestMiddlewareErrorStatusPassthrough(t *testing.T) {
	t.Parallel()

	repo := newTestRepository()
	errorHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		http.Error(w, "oops", http.StatusBadRequest)
	})

	res := httptest.NewRecorder()
	repo.Middleware(errorHandler).ServeHTTP(res, newBotRequest(t.Context(), "/error"))

	require.Equal(t, http.StatusBadRequest, res.Code)
	require.NotContains(t, res.Body.String(), "og:")
}

func TestMiddlewareErrorRouteFallback(t *testing.T) {
	t.Parallel()

	repo := newTestRepository()
	failingFunc := func(context.Context, *http.Request, map[string]string) (opengraph.Tags, error) {
		return opengraph.Tags{}, errForced
	}
	repo.Register("/fails", failingFunc)

	res := httptest.NewRecorder()
	repo.Middleware(htmlHandler("<html><head></head><body></body></html>")).ServeHTTP(res, newBotRequest(t.Context(), "/fails"))

	require.Equal(t, http.StatusOK, res.Code)

	body := res.Body.String()
	require.Contains(t, body, `property="og:title" content="gbans test"`)
	require.Contains(t, body, `property="og:description" content="test description"`)
}

func TestMiddlewareInvalidPatternsDiscarded(t *testing.T) {
	t.Parallel()

	repo := newTestRepository()
	for pattern, tagsFunc := range map[string]opengraph.TagsFunc{
		"":             nil,
		"/unset":       nil,
		"/bad/":        goodTagsFunc,
		"/bad/{a...}/": goodTagsFunc,
		"/x/{a...}y":   goodTagsFunc,
	} {
		repo.Register(pattern, tagsFunc)
	}

	res := httptest.NewRecorder()
	repo.Middleware(htmlHandler("<html><head></head><body></body></html>")).ServeHTTP(
		res, newBotRequest(t.Context(), "/bad/ok"))

	// All registrations were discarded, the site defaults are used.
	require.Contains(t, res.Body.String(), `property="og:title" content="gbans test"`)
}

func goodTagsFunc(context.Context, *http.Request, map[string]string) (opengraph.Tags, error) {
	return opengraph.Tags{Title: "custom"}, nil
}

func TestMiddlewareLongestMatchWins(t *testing.T) {
	t.Parallel()

	repo := newTestRepository()
	repo.Register("/forums/{forum_id}", func(context.Context, *http.Request, map[string]string) (opengraph.Tags, error) {
		return opengraph.Tags{Title: "forum"}, nil
	})
	repo.Register("/forums/thread/{forum_thread_id}", func(context.Context, *http.Request, map[string]string) (opengraph.Tags, error) {
		return opengraph.Tags{Title: "thread"}, nil
	})

	res := httptest.NewRecorder()
	repo.Middleware(htmlHandler("<html><head></head><body></body></html>")).ServeHTTP(
		res, newBotRequest(t.Context(), "/forums/thread/5"))

	require.Contains(t, res.Body.String(), `property="og:title" content="thread - gbans test"`)
}

func TestMiddlewareCatchAllMatch(t *testing.T) {
	t.Parallel()

	repo := newTestRepository()
	repo.Register("/wiki/{slug...}", func(_ context.Context, _ *http.Request, params map[string]string) (opengraph.Tags, error) {
		return opengraph.Tags{Title: "Wiki " + params["slug"], Description: "slug match"}, nil
	})

	res := httptest.NewRecorder()
	repo.Middleware(htmlHandler("<html><head></head><body></body></html>")).ServeHTTP(
		res, newBotRequest(t.Context(), "/wiki/a/b/page"))

	require.Contains(t, res.Body.String(), `property="og:title" content="Wiki a/b/page - gbans test"`)
}

func TestMiddlewareLargeBodyPassthrough(t *testing.T) {
	t.Parallel()

	repo := newTestRepository()

	// The body exceeds the buffer cap and is passed through unmodified.
	bigSuffix := strings.Repeat("x", 1024*1024+1)
	res := httptest.NewRecorder()
	repo.Middleware(htmlHandler("<html><head></head><body>"+bigSuffix+"</body></html>")).ServeHTTP(
		res, newBotRequest(t.Context(), "/large"))

	require.Contains(t, res.Body.String(), bigSuffix)
	require.NotContains(t, res.Body.String(), "og:")
}

func TestMiddlewareFlushingPassthrough(t *testing.T) {
	t.Parallel()

	repo := newTestRepository()
	flushHandler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/html")

		_, _ = writer.Write([]byte("<html><head></head><body>part1"))

		if flusher, ok := writer.(http.Flusher); ok {
			flusher.Flush()
		}

		_, _ = writer.Write([]byte("part2"))
	})

	res := httptest.NewRecorder()
	repo.Middleware(flushHandler).ServeHTTP(res, newBotRequest(t.Context(), "/stream"))

	// The streamed body is passed through verbatim, including writes made
	// after the flush.
	require.Equal(t, "<html><head></head><body>part1part2", res.Body.String())
	require.NotContains(t, res.Body.String(), "og:")
}

func TestMiddlewareTrailingSlashMatchesIndexRoute(t *testing.T) {
	t.Parallel()

	repo := newTestRepository()
	repo.Register("/forums", func(context.Context, *http.Request, map[string]string) (opengraph.Tags, error) {
		return opengraph.Tags{Title: "forums"}, nil
	})

	res := httptest.NewRecorder()
	repo.Middleware(htmlHandler("<html><head></head><body></body></html>")).ServeHTTP(
		res, newBotRequest(t.Context(), "/forums/"))

	require.Contains(t, res.Body.String(), `property="og:title" content="forums - gbans test"`)
}

func TestMiddlewareDuplicateWriteHeaderIgnored(t *testing.T) {
	t.Parallel()

	repo := newTestRepository()
	handler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/html")
		writer.WriteHeader(http.StatusOK)
		writer.WriteHeader(http.StatusInternalServerError)

		_, _ = writer.Write([]byte("<html><head></head><body></body></html>"))
	})

	res := httptest.NewRecorder()
	repo.Middleware(handler).ServeHTTP(res, newBotRequest(t.Context(), "/dup"))

	require.Equal(t, http.StatusOK, res.Code)
	require.Contains(t, res.Body.String(), "og:title")
}

func TestMiddlewareUnmatchedRouteDefaults(t *testing.T) {
	t.Parallel()

	repo := newTestRepository()

	res := httptest.NewRecorder()
	repo.Middleware(htmlHandler("<html><head></head><body></body></html>")).ServeHTTP(
		res, newBotRequest(t.Context(), "/anything/else"))

	require.Contains(t, res.Body.String(), `property="og:description" content="test description"`)
}
