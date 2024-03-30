//go:build release

package frontend

import (
	"embed"
	"errors"
	"net/http"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
)

//go:embed dist/*
var embedFS embed.FS

func AddRoutes(engine *gin.Engine, _ string, conf domain.Config) error {
	engine.Use(static.Serve("/", static.EmbedFolder(embedFS, "dist")))

	engine.NoRoute(func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/")
	})

	indexData, errIndex := embedFS.ReadFile("dist/index.html")
	if errIndex != nil {
		return errors.Join(errIndex, ErrContentRoot)
	}

	for _, rt := range jsRoutes {
		engine.GET(rt, func(ctx *gin.Context) {
			if conf.Log.SentryDSNWeb != "" {
				ctx.Header("Document-Policy", "js-profiling")
			}

			ctx.Data(http.StatusOK, "text/html", indexData)
		})
	}

	return nil
}
