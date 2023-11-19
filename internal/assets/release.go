//go:build release

package assets

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

//go:embed dist/*
var embedFS embed.FS

func StaticRoutes(engine *gin.Engine, _ bool) error {
	subFs, errSubFS := fs.Sub(embedFS, "dist")
	if errSubFS != nil {
		return errors.Wrap(errSubFS, "Could not setup embedfs")
	}

	engine.SetHTMLTemplate(template.
		Must(template.New("").
			Delims("{{", "}}").
			Funcs(engine.FuncMap).
			ParseFS(subFs, "index.html")))
	engine.StaticFS("/dist", http.FS(subFs))

	return nil
}
