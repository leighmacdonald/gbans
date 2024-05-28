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
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/util"
)

func AddRoutes(engine *gin.Engine, root string, conf domain.Config) error {
	if root == "" {
		root = "frontend/dist"
	}

	if !util.Exists(filepath.Join(root, "index.html")) {
		return ErrContentRoot
	}

	engine.Use(static.Serve("/", static.LocalFile(root, false)))

	for _, rt := range jsRoutes {
		engine.GET(rt, func(ctx *gin.Context) {
			indexData, errIndex := os.ReadFile(path.Join(root, "index.html"))
			if errIndex != nil {
				slog.Error("failed to open index.html", log.ErrAttr(errIndex))
			}

			if conf.Sentry.SentryDSNWeb != "" {
				ctx.Header("Document-Policy", "js-profiling")
			}

			ctx.Data(http.StatusOK, "text/html", indexData)
		})
	}

	return nil
}
