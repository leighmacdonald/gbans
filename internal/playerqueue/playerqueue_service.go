package playerqueue

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

type op int

const (
	// Ping is how you both join the swarm, and stay in it.
	Ping op = iota
	Pong
	JoinQueue
	LeaveQueue
	MessageSend
	MessageRecv
	StateUpdate
	StartGame
	Purge
)

type ServerQueuePayloadInbound struct {
	Op      op              `json:"op"`
	Payload json.RawMessage `json:"payload"`
}

type Msg struct {
	Op      op  `json:"op"`
	Payload any `json:"payload"`
}

type serverQueueHandler struct {
	queue       *Queue
	playerQueue domain.PlayerqueueUsecase
}

func NewServerQueueHandler(engine *gin.Engine, auth domain.AuthUsecase, config domain.ConfigUsecase,
	servers domain.ServersUsecase, state domain.StateUsecase, playerQueue domain.PlayerqueueUsecase, chatLogs []domain.Message,
) {
	conf := config.Config()
	var origins []string
	if conf.General.Mode == domain.ReleaseMode {
		origins = []string{conf.ExternalURL}
	}

	handler := &serverQueueHandler{
		queue:       NewServerQueue(100, 1, servers, state, chatLogs),
		playerQueue: playerQueue,
	}

	authedGroup := engine.Group("/api/playerqueue")
	{
		mod := authedGroup.Use(auth.Middleware(domain.PModerator))
		mod.PUT("/status/:steam_id", handler.status())
		mod.DELETE("/message/:message_id", handler.messageDelete())
		mod.DELETE("/purge/:steam_id/:count", handler.purge())
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
		user := httphelper.CurrentUserProfile(ctx)

		var req request
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if err := h.playerQueue.SetChatStatus(ctx, user.SteamID, req.ChatStatus); err != nil {
			httphelper.HandleErrInternal(ctx)

			return
		}

		slog.Info("Set chat status", slog.String("steam_id", user.SteamID.String()), slog.String("status", string(req.ChatStatus)))

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h *serverQueueHandler) start(validOrigins []string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser := httphelper.CurrentUserProfile(ctx)
		wsConn, errConn := newClientConn(ctx, validOrigins)
		if errConn != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Error("Failed to create client connection", log.ErrAttr(errConn))

			return
		}

		client := h.queue.Connect(ctx, currentUser, wsConn)
		defer h.queue.Disconnect(client)

		slog.Debug("Client joined queue swarm", slog.String("client", client.conn.RemoteAddr().String()))

		for {
			select {
			case <-ctx.Done():
				slog.Debug("Closing client connection", slog.String("client", client.conn.RemoteAddr().String()))

				return
			default:
				if err := h.handleMessage(ctx, client, currentUser); err != nil {
					if errors.Is(err, ErrQueueIO) {
						return
					}
					slog.Error("Failed to handle message", log.ErrAttr(err))
				}
			}
		}
	}
}

func (h *serverQueueHandler) handleMessage(ctx context.Context, client *Client, user domain.UserProfile) error {
	var payloadInbound ServerQueuePayloadInbound
	if errRead := client.conn.ReadJSON(&payloadInbound); errRead != nil {
		return errors.Join(errRead, ErrQueueIO)
	}

	var err error
	switch payloadInbound.Op {
	case Ping:
		h.queue.Ping(client)
	case JoinQueue:
		var p joinPayload
		if errUnmarshal := json.Unmarshal(payloadInbound.Payload, &p); errUnmarshal != nil {
			return errors.Join(errUnmarshal, ErrQueueParseMessage)
		}
		err = h.queue.JoinQueue(ctx, client, p.Servers)

	case LeaveQueue:
		var p leavePayload
		if errUnmarshal := json.Unmarshal(payloadInbound.Payload, &p); errUnmarshal != nil {
			return errors.Join(errUnmarshal, ErrQueueParseMessage)
		}

		err = h.queue.LeaveQueue(client, p.Servers)
	case MessageSend:
		var p domain.Message
		if errUnmarshal := json.Unmarshal(payloadInbound.Payload, &p); errUnmarshal != nil {
			return errors.Join(errUnmarshal, ErrQueueParseMessage)
		}

		msg, errMessage := h.queue.Message(p, user)
		if errMessage != nil {
			err = errMessage

			break
		}

		if _, errAdd := h.playerQueue.Add(ctx, msg); errAdd != nil {
			slog.Error("Failed to add playerqueue message", log.ErrAttr(errAdd))
		}

	default:
		return ErrUnexpectedMessage
	}

	return err
}

func (h *serverQueueHandler) messageDelete() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		messageID, errID := httphelper.GetUUIDParam(ctx, "message_id")
		if errID != nil {
			httphelper.HandleErrBadRequest(ctx)

			return
		}

		if errDelete := h.playerQueue.Delete(ctx, messageID); errDelete != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to delete message", log.ErrAttr(errDelete))

			return
		}

		h.queue.purgeMessages(messageID)

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (h *serverQueueHandler) purge() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, err := httphelper.GetSID64Param(ctx, "steam_id")
		if err != nil {
			httphelper.HandleErrBadRequest(ctx)

			return
		}

		count, errCount := httphelper.GetIntParam(ctx, "count")
		if errCount != nil || count <= 0 {
			httphelper.HandleErrBadRequest(ctx)

			return
		}

		var messageIDs []uuid.UUID
		for _, msg := range h.queue.findMessages(steamID, count) {
			messageIDs = append(messageIDs, msg.MessageID)
		}

		if len(messageIDs) == 0 {
			ctx.JSON(http.StatusOK, gin.H{})
		}

		h.queue.purgeMessages(messageIDs...)

		if errDelete := h.playerQueue.Delete(ctx, messageIDs...); errDelete != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to delete message", log.ErrAttr(errDelete))

			return
		}
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
