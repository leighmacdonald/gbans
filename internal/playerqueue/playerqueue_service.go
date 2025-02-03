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

	modGroup := engine.Group("/")
	{
		mod := modGroup.Use(auth.MiddlewareWS(domain.PUser))
		mod.GET("/ws", handler.start(origins))
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
