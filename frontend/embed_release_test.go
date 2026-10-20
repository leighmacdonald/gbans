//go:build release

package frontend

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestReleaseEmbedChunks verifies that lazy route chunks (files whose names
// begin with an underscore, e.g. _guest.index-*.js) are embedded and served by
// the release build. go:embed's default pattern excludes such files.
func TestReleaseEmbedChunks(t *testing.T) {
	t.Parallel()

	entries, errRead := os.ReadDir("dist/assets")
	if errRead != nil {
		t.Fatalf("read dist/assets: %v", errRead)
	}

	var chunk string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "_") {
			chunk = entry.Name()

			break
		}
	}

	if chunk == "" {
		t.Fatal("no underscore-prefixed chunk found in dist/assets")
	}

	mux := http.NewServeMux()
	if errAdd := AddRoutes(mux, ""); errAdd != nil {
		t.Fatalf("AddRoutes: %v", errAdd)
	}

	for _, path := range []string{"/assets/" + chunk, "/", "/profile/76561197970669190"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", path, rec.Code)
		}
	}

	t.Logf("verified embedded chunk %s", chunk)
}
