//go:build !release

package frontend

import (
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/pkg/fs"
	"github.com/leighmacdonald/gbans/pkg/log"
)

func AddRoutes(engine *gin.Engine, root string) error {
	if root == "" {
		root = "frontend/dist"
	}

	if !fs.Exists(filepath.Join(root, "index.html")) {
		return ErrContentRoot
	}

	engine.Use(static.Serve("/", static.LocalFile(root, false)))

	for _, rt := range jsRoutes {
		engine.GET(rt, func(ctx *gin.Context) {
			indexData, errIndex := os.ReadFile(path.Join(root, "index.html"))
			if errIndex != nil {
				slog.Error("failed to open index.html", log.ErrAttr(errIndex))
			}

			ctx.Data(http.StatusOK, "text/html", indexData)
		})
	}

	return nil
}
