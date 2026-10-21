package frontend_test

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// TestIndexHTMLAssetPaths ensures the built index references assets from the
// site root. The SPA is served from deep links like /login/success, where a
// relative asset path would resolve under /login/ and return 404, leaving the
// page blank.
func TestIndexHTMLAssetPaths(t *testing.T) {
	t.Parallel()

	indexData, errRead := os.ReadFile("dist/index.html")
	if errors.Is(errRead, os.ErrNotExist) {
		t.Skip("frontend dist not built")
	}

	if errRead != nil {
		t.Fatal(errRead)
	}

	body := string(indexData)
	for _, relativePath := range []string{`src="./`, `href="./assets/`} {
		if strings.Contains(body, relativePath) {
			t.Fatalf("index.html contains relative asset path %q", relativePath)
		}
	}
}
