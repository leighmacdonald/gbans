package httphelper_test

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

type recordedRequest struct {
	method   string
	route    string
	status   int
	duration time.Duration
}

type recordingMetrics struct {
	requests []recordedRequest
}

func (m *recordingMetrics) ObserveRequest(method string, route string, status int, duration time.Duration) {
	m.requests = append(m.requests, recordedRequest{method, route, status, duration})
}

func TestPrometheusMiddlewareRecordsRequests(t *testing.T) {
	metrics := &recordingMetrics{}
	handler := httphelper.PrometheusMiddleware(metrics)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/thing", nil))

	require.Equal(t, http.StatusCreated, res.Code)
	require.Len(t, metrics.requests, 1)
	require.Equal(t, http.MethodPost, metrics.requests[0].method)
	require.Equal(t, http.StatusCreated, metrics.requests[0].status)
	require.NotEmpty(t, metrics.requests[0].route)
}

func TestNewInternalServer(t *testing.T) {
	get := func(t *testing.T, client *http.Client, url string) *http.Response {
		t.Helper()

		req, errReq := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		require.NoError(t, errReq)

		res, errDo := client.Do(req)
		require.NoError(t, errDo)

		return res
	}

	t.Run("only metrics registered", func(t *testing.T) {
		server := httphelper.NewInternalServer("127.0.0.1:0", true, false, prometheus.NewRegistry())
		testServer := httptest.NewServer(server.Handler)
		defer testServer.Close()

		res := get(t, testServer.Client(), testServer.URL+"/metrics")
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)

		resPprof := get(t, testServer.Client(), testServer.URL+"/debug/pprof/")
		defer resPprof.Body.Close()
		require.Equal(t, http.StatusNotFound, resPprof.StatusCode)
	})

	t.Run("only pprof registered", func(t *testing.T) {
		server := httphelper.NewInternalServer("127.0.0.1:0", false, true, prometheus.NewRegistry())
		testServer := httptest.NewServer(server.Handler)
		defer testServer.Close()

		res := get(t, testServer.Client(), testServer.URL+"/debug/pprof/")
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)

		resMetrics := get(t, testServer.Client(), testServer.URL+"/metrics")
		defer resMetrics.Body.Close()
		require.Equal(t, http.StatusNotFound, resMetrics.StatusCode)
	})
}

func TestRequestBaseURL(t *testing.T) {
	t.Run("direct http", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), "GET", "http://localhost:6007/auth/callback", nil)
		require.Equal(t, "http://localhost:6007", httphelper.RequestBaseURL(req))
	})

	t.Run("direct https", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), "GET", "https://gbans.roto.lol/", nil)
		req.TLS = &tls.ConnectionState{}
		require.Equal(t, "https://gbans.roto.lol", httphelper.RequestBaseURL(req))
	})

	t.Run("forwarded proto wins", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), "GET", "http://127.0.0.1:6006/", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Forwarded-Host", "gbans.roto.lol")
		require.Equal(t, "https://gbans.roto.lol", httphelper.RequestBaseURL(req))
	})

	t.Run("cf-visitor scheme for tunnels", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), "GET", "http://127.0.0.1:6006/", nil)
		req.Host = "gbans.roto.lol"
		req.Header.Set("Cf-Visitor", `{"scheme":"https"}`)
		require.Equal(t, "https://gbans.roto.lol", httphelper.RequestBaseURL(req))
	})

	t.Run("malformed cf-visitor ignored", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), "GET", "http://127.0.0.1:6006/", nil)
		req.Header.Set("Cf-Visitor", "not-json")
		require.Equal(t, "http://127.0.0.1:6006", httphelper.RequestBaseURL(req))
	})
}

func TestClientIP(t *testing.T) {
	t.Run("cf-connecting-ip wins", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), "GET", "http://localhost:6007/", nil)
		req.Header.Set("Cf-Connecting-Ip", "1.2.3.4")
		req.Header.Set("X-Forwarded-For", "5.6.7.8")
		addr, ok := httphelper.ClientIP(req)
		require.True(t, ok)
		require.Equal(t, "1.2.3.4", addr.String())
	})

	t.Run("forwarded chain uses leftmost", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), "GET", "http://localhost:6007/", nil)
		req.Header.Set("X-Forwarded-For", "9.9.9.9, 10.0.0.1, 172.16.0.1")
		addr, ok := httphelper.ClientIP(req)
		require.True(t, ok)
		require.Equal(t, "9.9.9.9", addr.String())
	})

	t.Run("real ip fallback", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), "GET", "http://localhost:6007/", nil)
		req.Header.Set("X-Real-IP", "8.8.8.8")
		addr, ok := httphelper.ClientIP(req)
		require.True(t, ok)
		require.Equal(t, "8.8.8.8", addr.String())
	})

	t.Run("remote addr fallback", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), "GET", "http://localhost:6007/", nil)
		req.RemoteAddr = "203.0.113.7:4567"
		addr, ok := httphelper.ClientIP(req)
		require.True(t, ok)
		require.Equal(t, "203.0.113.7", addr.String())
	})

	t.Run("garbage headers fall through", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), "GET", "http://localhost:6007/", nil)
		req.Header.Set("Cf-Connecting-Ip", "not-an-ip")
		req.RemoteAddr = "203.0.113.9:4567"
		addr, ok := httphelper.ClientIP(req)
		require.True(t, ok)
		require.Equal(t, "203.0.113.9", addr.String())
	})

	t.Run("nothing usable", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), "GET", "http://localhost:6007/", nil)
		req.RemoteAddr = "garbage"
		_, ok := httphelper.ClientIP(req)
		require.False(t, ok)
	})
}
