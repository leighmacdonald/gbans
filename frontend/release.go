//go:build release

package frontend

import (
	"embed"
	"errors"
	"io/fs"
	"net/http"
)

// The all: prefix is required: TanStack Router emits lazy route chunks whose
// names start with an underscore (ex. _guest.index-*.js) and the default
// go:embed pattern silently excludes files beginning with "_" or ".".
//
//go:embed all:dist
var embedFS embed.FS

func AddRoutes(mux *http.ServeMux, _ string) error {
	distFS, errFS := fs.Sub(embedFS, "dist")
	if errFS != nil {
		return errors.Join(errFS, ErrContentRoot)
	}

	fsHandler := http.FileServer(http.FS(distFS))
	mux.Handle("GET /", fsHandler)

	indexData, errIndex := embedFS.ReadFile("dist/index.html")
	if errIndex != nil {
		return errors.Join(errIndex, ErrContentRoot)
	}

	for _, rt := range jsRoutes {
		rtCopy := rt
		mux.HandleFunc("GET "+rtCopy, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(indexData)
		})
	}

	return nil
}
