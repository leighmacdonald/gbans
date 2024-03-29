//go:build release

package frontend

import (
	"embed"
	"errors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
)

//go:embed dist/*
var embedFS embed.FS

var ErrEmbedFS = errors.New("failed to load embed.fs path")

func AddRoutes(engine *gin.Engine, _ string, conf domain.Config) {
	engine.Use(embedDist("/", "dist", embedFS))
}

func embedDist(urlPrefix, buildDirectory string, em embed.FS) gin.HandlerFunc {
	dir := static.LocalFile(buildDirectory, true)
	embedDir, _ := fs.Sub(em, buildDirectory)
	fileServer := http.FileServer(http.FS(embedDir))

	if urlPrefix != "" {
		fileServer = http.StripPrefix(urlPrefix, fileServer)
	}

	return func(c *gin.Context) {
		if !dir.Exists(urlPrefix, c.Request.URL.Path) {
			c.Request.URL.Path = "/"
		}

		slog.Info(c.Request.RequestURI)
		if strings.HasSuffix(c.Request.RequestURI, ".js") {
			c.Writer.Header().Set("Content-Type", "application/json")
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}
