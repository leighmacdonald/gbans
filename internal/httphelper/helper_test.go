package httphelper_test

import (
	"crypto/tls"
	"net/http/httptest"
	"testing"

	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/stretchr/testify/require"
)

func TestRequestBaseURL(t *testing.T) {
	t.Run("direct http", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://localhost:6007/auth/callback", nil)
		require.Equal(t, "http://localhost:6007", httphelper.RequestBaseURL(req))
	})

	t.Run("direct https", func(t *testing.T) {
		req := httptest.NewRequest("GET", "https://gbans.roto.lol/", nil)
		req.TLS = &tls.ConnectionState{}
		require.Equal(t, "https://gbans.roto.lol", httphelper.RequestBaseURL(req))
	})

	t.Run("forwarded proto wins", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://127.0.0.1:6006/", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Forwarded-Host", "gbans.roto.lol")
		require.Equal(t, "https://gbans.roto.lol", httphelper.RequestBaseURL(req))
	})

	t.Run("cf-visitor scheme for tunnels", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://127.0.0.1:6006/", nil)
		req.Host = "gbans.roto.lol"
		req.Header.Set("CF-Visitor", `{"scheme":"https"}`)
		require.Equal(t, "https://gbans.roto.lol", httphelper.RequestBaseURL(req))
	})

	t.Run("malformed cf-visitor ignored", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://127.0.0.1:6006/", nil)
		req.Header.Set("CF-Visitor", "not-json")
		require.Equal(t, "http://127.0.0.1:6006", httphelper.RequestBaseURL(req))
	})
}

func TestClientIP(t *testing.T) {
	t.Run("cf-connecting-ip wins", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://localhost:6007/", nil)
		req.Header.Set("CF-Connecting-IP", "1.2.3.4")
		req.Header.Set("X-Forwarded-For", "5.6.7.8")
		addr, ok := httphelper.ClientIP(req)
		require.True(t, ok)
		require.Equal(t, "1.2.3.4", addr.String())
	})

	t.Run("forwarded chain uses leftmost", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://localhost:6007/", nil)
		req.Header.Set("X-Forwarded-For", "9.9.9.9, 10.0.0.1, 172.16.0.1")
		addr, ok := httphelper.ClientIP(req)
		require.True(t, ok)
		require.Equal(t, "9.9.9.9", addr.String())
	})

	t.Run("real ip fallback", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://localhost:6007/", nil)
		req.Header.Set("X-Real-IP", "8.8.8.8")
		addr, ok := httphelper.ClientIP(req)
		require.True(t, ok)
		require.Equal(t, "8.8.8.8", addr.String())
	})

	t.Run("remote addr fallback", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://localhost:6007/", nil)
		req.RemoteAddr = "203.0.113.7:4567"
		addr, ok := httphelper.ClientIP(req)
		require.True(t, ok)
		require.Equal(t, "203.0.113.7", addr.String())
	})

	t.Run("garbage headers fall through", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://localhost:6007/", nil)
		req.Header.Set("CF-Connecting-IP", "not-an-ip")
		req.RemoteAddr = "203.0.113.9:4567"
		addr, ok := httphelper.ClientIP(req)
		require.True(t, ok)
		require.Equal(t, "203.0.113.9", addr.String())
	})

	t.Run("nothing usable", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://localhost:6007/", nil)
		req.RemoteAddr = "garbage"
		_, ok := httphelper.ClientIP(req)
		require.False(t, ok)
	})
}
