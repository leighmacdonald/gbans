package server_queue

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
	"log/slog"
	"net/http"
)

type serverQueueHandler struct {
	queue *ServerQueue
}

func NewServerQueueHandler(engine *gin.Engine, auth domain.AuthUsecase) {
	handler := &serverQueueHandler{
		queue: NewServerQueue(),
	}

	modGroup := engine.Group("/")
	{
		mod := modGroup.Use(auth.AuthMiddlewareWS(domain.PUser))
		mod.GET("/ws", handler.start())
	}
}

func newClientConn(ctx *gin.Context) (*websocket.Conn, error) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to upgrade websocket connection"))
	}

	return conn, nil
}

func (h *serverQueueHandler) start() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser := httphelper.CurrentUserProfile(ctx)

		wsConn, errConn := newClientConn(ctx)
		if errConn != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Error("Failed to create client connection", log.ErrAttr(errConn))
			return
		}

		client := h.queue.ConnectClient(ctx, currentUser, wsConn)
		defer h.queue.DisconnectClient(client)

		slog.Debug("Client joined queue swarm", slog.String("client", client.conn.RemoteAddr().String()))

		for {
			select {
			case <-ctx.Done():
				slog.Debug("Closing client connection", slog.String("client", client.conn.RemoteAddr().String()))

				return
			default:
				if err := h.handleMessage(client, currentUser); err != nil {
					if errors.Is(err, ErrQueueIO) {
						return
					}
					slog.Error("Failed to handle message", log.ErrAttr(err))
				}
			}

		}
	}
}

func (h *serverQueueHandler) handleMessage(client *ClientConn, user domain.UserProfile) error {
	var payloadInbound domain.ServerQueuePayloadInbound
	if errRead := client.conn.ReadJSON(&payloadInbound); errRead != nil {
		return errors.Join(errRead, ErrQueueIO)
	}

	switch payloadInbound.Op {
	case domain.Ping:
		h.queue.Ping(client)
	case domain.Join:
		var p domain.JoinPayload
		if errUnmarshal := json.Unmarshal(payloadInbound.Payload, &p); errUnmarshal != nil {
			return errors.Join(errUnmarshal, ErrQueueParseMessage)
		}
		h.queue.ConnectClient(p, user)
	case domain.Leave:
		var p domain.LeavePayload
		if errUnmarshal := json.Unmarshal(payloadInbound.Payload, &p); errUnmarshal != nil {
			return errors.Join(errUnmarshal, ErrQueueParseMessage)
		}
		h.queue.DisconnectClient(p, user)
	case domain.MessageSend:
		var p domain.ServerQueueMessage
		if errUnmarshal := json.Unmarshal(payloadInbound.Payload, &p); errUnmarshal != nil {
			return errors.Join(errUnmarshal, ErrQueueParseMessage)
		}
		h.queue.Message(p, user)

	default:
		return ErrUnexpectedMessage
	}

	return nil
}
