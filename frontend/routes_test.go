//nolint:testpackage
package frontend

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestAddRoutesDeepLinks ensures the SPA routes are served for hard refreshed
// deep links, which requires the patterns to be valid serve mux wildcards and
// conflict free at registration time.
func TestAddRoutesDeepLinks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if errWrite := os.WriteFile(filepath.Join(root, "index.html"), []byte("<html><head></head><body>spa</body></html>"), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}

	mux := http.NewServeMux()
	if errAdd := AddRoutes(mux, root); errAdd != nil {
		t.Fatal(errAdd)
	}

	tests := []string{
		"/", "/profile/76561197970669190", "/ban/4", "/report/1", "/wiki/page/sub",
		"/match/5a3c5854-8165-404f-b3fa-cb7bbdc73504", "/matches", "/servers", "/admin/network/ipInfo",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			res := httptest.NewRecorder()
			mux.ServeHTTP(res, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))

			if res.Code != http.StatusOK {
				t.Fatalf("expected 200 for %s, got %d", path, res.Code)
			}

			if body := res.Body.String(); body != "<html><head></head><body>spa</body></html>" {
				t.Fatalf("expected spa index body, got %q", body)
			}
		})
	}
}
