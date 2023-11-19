//go:build !release

package assets

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

func StaticRoutes(engine *gin.Engine, testing bool) error {
	absStaticPath, errStaticPath := filepath.Abs("./internal/assets/dist")
	if errStaticPath != nil {
		return errors.Wrap(errStaticPath, "Failed to setup static paths")
	}

	engine.StaticFS("/dist", http.Dir(absStaticPath))

	if !testing {
		engine.LoadHTMLFiles(filepath.Join(absStaticPath, "index.html"))
	}

	return nil
}
