//go:build release

package frontend

import (
	"embed"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var embedFS embed.FS

func AddRoutes(engine *gin.Engine, _ string) error {
	engine.Use(static.Serve("/", static.EmbedFolder(embedFS, "dist")))

	engine.NoRoute(func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/")
	})

	indexData, errIndex := embedFS.ReadFile("dist/index.html")
	if errIndex != nil {
		slog.Error("Failed to read index.html")

		return errors.Join(errIndex, ErrContentRoot)
	}

	for _, rt := range jsRoutes {
		engine.GET(rt, func(ctx *gin.Context) {
			ctx.Data(http.StatusOK, "text/html", indexData)
		})
	}

	return nil
}
