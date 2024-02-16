//go:build !release

package frontend

import (
	"net/http"
	"os"
	"path"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
)

func AddRoutes(engine *gin.Engine, root string, conf domain.Config) {
	if root == "" {
		root = "frontend/dist"
	}

	engine.Use(static.Serve("/", static.LocalFile(root, false)))

	for _, rt := range jsRoutes {
		engine.GET(rt, func(ctx *gin.Context) {
			indexData, errIndex := os.ReadFile(path.Join(root, "index.html"))
			if errIndex != nil {
				panic("failed to load index.html")
			}

			if conf.Log.SentryDSNWeb != "" {
				ctx.Header("Document-Policy", "js-profiling")
			}

			ctx.Data(http.StatusOK, "text/html", indexData)
		})
	}
}
