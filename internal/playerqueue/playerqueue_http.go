package playerqueue

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type serverQueueHandler struct {
	*Playerqueue
}

func NewPlayerqueueHandler(engine *gin.Engine, auth httphelper.Authenticator, playerQueue *Playerqueue) {
	var origins []string
	// if conf.General.Mode == config.ReleaseMode {
	// 	origins = []string{conf.ExternalURL}
	// }

	handler := &serverQueueHandler{Playerqueue: playerQueue}

	authedGroup := engine.Group("/api/playerqueue")
	{
		mod := authedGroup.Use(auth.Middleware(permission.Moderator))
		mod.PUT("/status/:steam_id", handler.status())
		mod.DELETE("/messages/:message_id/:count", handler.purge())
	}

	authedGroupWS := engine.Group("/")
	{
		mod := authedGroupWS.Use(auth.MiddlewareWS(permission.User))
		mod.GET("/ws", handler.start(origins))
	}
}

func (h *serverQueueHandler) status() gin.HandlerFunc {
	type request struct {
		Reason     string     `json:"reason"`
		ChatStatus ChatStatus `json:"chat_status"`
	}

	return func(ctx *gin.Context) {
		currentUser, _ := session.CurrentUserProfile(ctx)
		req, ok := httphelper.BindJSON[request](ctx)
		if !ok {
			return
		}

		steamID, idFound := httphelper.GetSID64Param(ctx, "steam_id")
		if !idFound {
			return
		}

		if err := h.SetChatStatus(ctx, currentUser.GetSteamID(), steamID, req.ChatStatus, req.Reason); err != nil {
			if errors.Is(err, httphelper.ErrPermissionDenied) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusForbidden, httphelper.ErrPermissionDenied))

				return
			}

			if errors.Is(err, database.ErrDuplicate) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusConflict, database.ErrDuplicate,
					"Status must be different"))

				return
			}

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h *serverQueueHandler) start(validOrigins []string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser, _ := session.CurrentUserProfile(ctx)
		// Create ws connection
		wsConn, errConn := newClientConn(ctx, validOrigins)
		if errConn != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, errors.Join(errConn, httphelper.ErrBadRequest),
				"Cannot open ws connection"))

			return
		}

		// Connect to the coordinator with our connection
		clientConn := h.Connect(ctx, currentUser, wsConn)
		defer h.Disconnect(clientConn)

		for {
			request, err := h.handleWSMessage(clientConn)
			if err != nil {
				switch {
				case errors.Is(err, context.Canceled):
					return
				case errors.Is(err, ErrQueueIO):
					slog.Debug("Client connection error", slog.String("client", clientConn.ID()), slog.String("error", ErrQueueIO.Error()))

					return
				default:
					slog.Error("Error trying to handle websocket message", slog.String("error", err.Error()))

					return
				}
			}

			if errHandler := h.handleRequest(ctx, clientConn, request, currentUser); errHandler != nil {
				slog.Error("Error trying to handle websocket request", slog.String("error", errHandler.Error()))

				continue
			}
		}
	}
}

func (h *serverQueueHandler) handleWSMessage(client Client) (Request, error) {
	var payloadInbound Request
	if errRead := client.Next(&payloadInbound); errRead != nil {
		return payloadInbound, errors.Join(errRead, ErrQueueIO)
	}

	return payloadInbound, nil
}

func (h *serverQueueHandler) handleRequest(ctx context.Context, client Client, payloadInbound Request, user person.Info) error {
	var err error
	switch payloadInbound.Op {
	case JoinQueue:
		var p JoinPayload
		if errUnmarshal := json.Unmarshal(payloadInbound.Payload, &p); errUnmarshal != nil {
			return errors.Join(errUnmarshal, ErrQueueParseMessage)
		}
		err = h.JoinLobbies(client, p.Servers)

	case LeaveQueue:
		var p LeavePayload
		if errUnmarshal := json.Unmarshal(payloadInbound.Payload, &p); errUnmarshal != nil {
			return errors.Join(errUnmarshal, ErrQueueParseMessage)
		}

		err = h.LeaveLobbies(client, p.Servers)
	case Message:
		client.Limit()
		var p MessageCreatePayload
		if errUnmarshal := json.Unmarshal(payloadInbound.Payload, &p); errUnmarshal != nil {
			return errors.Join(errUnmarshal, ErrQueueParseMessage)
		}

		err = h.AddMessage(ctx, p.BodyMD, user)

	default:
		return ErrUnexpectedMessage
	}

	return err
}

func (h *serverQueueHandler) purge() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, _ := session.CurrentUserProfile(ctx)

		messageID, idFound := httphelper.GetInt64Param(ctx, "message_id")
		if !idFound {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, httphelper.ErrNotFound))

			return
		}

		count, countFound := httphelper.GetIntParam(ctx, "count")
		if !countFound {
			return
		}
		if count <= 0 {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusBadRequest, httphelper.ErrBadRequest))

			return
		}

		errPurge := h.Purge(ctx, user.GetSteamID(), messageID, count)
		if errPurge != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errPurge))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

var errUpgrader = errors.New("failed to upgrade websocket connection")

func newClientConn(ctx *gin.Context, validOrigin []string) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(req *http.Request) bool {
			if len(validOrigin) == 0 {
				return true
			}

			origin := req.Header.Get("Origin")
			for _, v := range validOrigin {
				if strings.HasPrefix(origin, v) {
					return true
				}
			}

			return false
		},
	}

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return nil, errors.Join(err, errUpgrader)
	}

	return conn, nil
}
