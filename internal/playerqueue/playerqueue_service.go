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
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type serverQueueHandler struct {
	queue domain.PlayerqueueUsecase
}

func NewPlayerqueueHandler(engine *gin.Engine, auth domain.AuthUsecase, config domain.ConfigUsecase,
	playerQueue domain.PlayerqueueUsecase,
) {
	conf := config.Config()
	var origins []string
	if conf.General.Mode == domain.ReleaseMode {
		origins = []string{conf.ExternalURL}
	}

	handler := &serverQueueHandler{
		queue: playerQueue,
	}

	authedGroup := engine.Group("/api/playerqueue")
	{
		mod := authedGroup.Use(auth.Middleware(domain.PModerator))
		mod.PUT("/status/:steam_id", handler.status())
		mod.DELETE("/messages/:message_id/:count", handler.purge())
	}

	authedGroupWS := engine.Group("/")
	{
		mod := authedGroupWS.Use(auth.MiddlewareWS(domain.PUser))
		mod.GET("/ws", handler.start(origins))
	}
}

func (h *serverQueueHandler) status() gin.HandlerFunc {
	type request struct {
		Reason     string            `json:"reason"`
		ChatStatus domain.ChatStatus `json:"chat_status"`
	}

	return func(ctx *gin.Context) {
		currentUser := httphelper.CurrentUserProfile(ctx)

		var req request
		if !httphelper.Bind(ctx, &req) {
			return
		}

		steamID, idFound := httphelper.GetSID64Param(ctx, "steam_id")
		if !idFound {
			return
		}

		if err := h.queue.SetChatStatus(ctx, currentUser.SteamID, steamID, req.ChatStatus, req.Reason); err != nil {
			if errors.Is(err, domain.ErrPermissionDenied) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusForbidden, domain.ErrPermissionDenied))

				return
			}

			if errors.Is(err, domain.ErrDuplicate) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusConflict, domain.ErrDuplicate,
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
		currentUser := httphelper.CurrentUserProfile(ctx)
		// Create ws connection
		wsConn, errConn := newClientConn(ctx, validOrigins)
		if errConn != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, errors.Join(errConn, domain.ErrBadRequest),
				"Cannot open ws connection"))

			return
		}

		// Connect to the coordinator with our connection
		client := h.queue.Connect(ctx, currentUser, wsConn)
		defer h.queue.Disconnect(client)

		for {
			request, err := h.handleWSMessage(client)
			if err != nil {
				switch {
				case errors.Is(err, context.Canceled):
					return
				case errors.Is(err, ErrQueueIO):
					slog.Debug("Client connection error", slog.String("client", client.ID()), log.ErrAttr(ErrQueueIO))

					return
				default:
					slog.Error("Error trying to handle websocket message", log.ErrAttr(err))

					return
				}
			}

			if errHandler := h.handleRequest(ctx, client, request, currentUser); errHandler != nil {
				slog.Error("Error trying to handle websocket request", log.ErrAttr(errHandler))

				continue
			}
		}
	}
}

func (h *serverQueueHandler) handleWSMessage(client domain.QueueClient) (domain.Request, error) {
	var payloadInbound domain.Request
	if errRead := client.Next(&payloadInbound); errRead != nil {
		return payloadInbound, errors.Join(errRead, ErrQueueIO)
	}

	return payloadInbound, nil
}

func (h *serverQueueHandler) handleRequest(ctx context.Context, client domain.QueueClient, payloadInbound domain.Request, user domain.UserProfile) error {
	var err error
	switch payloadInbound.Op {
	case domain.JoinQueue:
		var p JoinPayload
		if errUnmarshal := json.Unmarshal(payloadInbound.Payload, &p); errUnmarshal != nil {
			return errors.Join(errUnmarshal, ErrQueueParseMessage)
		}
		err = h.queue.JoinLobbies(client, p.Servers)

	case domain.LeaveQueue:
		var p LeavePayload
		if errUnmarshal := json.Unmarshal(payloadInbound.Payload, &p); errUnmarshal != nil {
			return errors.Join(errUnmarshal, ErrQueueParseMessage)
		}

		err = h.queue.LeaveLobbies(client, p.Servers)
	case domain.Message:
		client.Limit()
		var p MessageCreatePayload
		if errUnmarshal := json.Unmarshal(payloadInbound.Payload, &p); errUnmarshal != nil {
			return errors.Join(errUnmarshal, ErrQueueParseMessage)
		}

		err = h.queue.AddMessage(ctx, p.BodyMD, user)

	default:
		return ErrUnexpectedMessage
	}

	return err
}

func (h *serverQueueHandler) purge() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := httphelper.CurrentUserProfile(ctx)

		messageID, idFound := httphelper.GetInt64Param(ctx, "message_id")
		if !idFound {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, domain.ErrNotFound))

			return
		}

		count, countFound := httphelper.GetIntParam(ctx, "count")
		if !countFound {
			return
		}
		if count <= 0 {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusBadRequest, domain.ErrBadRequest))

			return
		}

		errPurge := h.queue.Purge(ctx, user.SteamID, messageID, count)
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
