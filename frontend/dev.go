//go:build !release

package frontend

import (
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/leighmacdonald/gbans/internal/fs"
)

func AddRoutes(mux *http.ServeMux, root string) error {
	if root == "" {
		root = "frontend/dist"
	}

	if !fs.Exists(filepath.Join(root, "index.html")) {
		return ErrContentRoot
	}

	fsHandler := http.FileServer(http.Dir(root))
	mux.Handle("GET /", fsHandler)

	for _, rt := range jsRoutes {
		rtCopy := rt
		mux.HandleFunc("GET "+rtCopy, func(w http.ResponseWriter, r *http.Request) {
			indexData, errIndex := os.ReadFile(path.Join(root, "index.html"))
			if errIndex != nil {
				slog.Error("failed to open index.html", slog.String("error", errIndex.Error()))
			}

			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(indexData)
		})
	}

	return nil
}
