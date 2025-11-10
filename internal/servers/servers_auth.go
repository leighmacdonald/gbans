package servers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

func NewServerAuth(servers Servers, sentryDSN string) *ServerAuth {
	return &ServerAuth{servers: servers, sentryDSN: sentryDSN}
}

type ServerAuth struct {
	servers   Servers
	sentryDSN string
}

func (s ServerAuth) Middleware(ctx *gin.Context) {
	reqAuthHeader := ctx.GetHeader("Authorization")
	if reqAuthHeader == "" {
		ctx.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	if strings.HasPrefix(reqAuthHeader, "Bearer ") {
		parts := strings.Split(reqAuthHeader, " ")
		if len(parts) != 2 {
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		reqAuthHeader = parts[1]
	}

	server, errServer := s.servers.GetByPassword(ctx, reqAuthHeader)
	if errServer != nil {
		slog.Error("Failed to load server during auth", slog.String("error", errServer.Error()), slog.String("token", reqAuthHeader), slog.String("IP", ctx.ClientIP()))
		ctx.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	ctx.Set("server_id", server.ServerID)

	if s.sentryDSN != "" {
		if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetUser(sentry.User{
					ID:        strconv.Itoa(server.ServerID),
					IPAddress: server.Addr(),
					Name:      server.ShortName,
				})
			})
		}
	}

	ctx.Next()
}
