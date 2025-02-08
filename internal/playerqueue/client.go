package playerqueue

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"go.uber.org/ratelimit"
)

var (
	ErrUnexpectedMessage = errors.New("unexpected message")
	ErrQueueIO           = errors.New("failed to read / write from connection")
	ErrQueueParseMessage = errors.New("failed to parse message")
	ErrBadInput          = errors.New("bad user input")
	ErrFindLobby         = errors.New("failed to find a Lobby")
	ErrHostname          = errors.New("failed to resolve hostname")
	ErrReadRequest       = errors.New("failed to read/decode request")
)

func newClient(steamID steamid.SteamID, name string, avatarHash string, conn *websocket.Conn) domain.QueueClient {
	client := &Client{
		steamID:      steamID,
		name:         name,
		avatarhash:   avatarHash,
		conn:         conn,
		responseChan: make(chan domain.Response, 2),
		lastPing:     time.Time{},
		rl:           ratelimit.New(1, ratelimit.Per(5*time.Second)),
	}

	return client
}

type Client struct {
	steamID      steamid.SteamID
	name         string
	avatarhash   string
	conn         *websocket.Conn
	responseChan chan domain.Response
	lastPing     time.Time
	chatStatus   domain.ChatStatus
	rl           ratelimit.Limiter
}

func (c *Client) Limit() {
	c.rl.Take()
}

func (c *Client) HasMessageAccess() bool {
	return c.chatStatus != domain.Noaccess
}

func (c *Client) Send(response domain.Response) {
	c.responseChan <- response
}

func (c *Client) SteamID() steamid.SteamID {
	return c.steamID
}

func (c *Client) Name() string {
	return c.name
}

func (c *Client) Avatarhash() string {
	return c.avatarhash
}

func (c *Client) Next(request *domain.Request) error {
	if err := c.conn.ReadJSON(request); err != nil {
		return errors.Join(err, ErrReadRequest)
	}

	return nil
}

func (c *Client) Ping() {
	c.Send(domain.Response{
		Op:      domain.Pong,
		Payload: pongPayload{CreatedOn: time.Now()},
	})
}

func (c *Client) ID() string {
	return c.conn.RemoteAddr().String()
}

func (c *Client) IsTimedOut() bool {
	return time.Since(c.lastPing) > time.Minute
}

func (c *Client) Start(ctx context.Context) {
	// TODO refactor this so there is nonoe of this logic under domain.
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-c.responseChan:
			if errWrite := c.conn.WriteJSON(msg); errWrite != nil {
				slog.Error("Failed to send message to client", log.ErrAttr(errWrite))
			}
		}
	}
}

func (c *Client) Close() {
	slog.Debug("Closing client connection", slog.String("addr", c.conn.RemoteAddr().String()))
	if errClose := c.conn.Close(); errClose != nil {
		slog.Warn("Error closing client connection", log.ErrAttr(errClose))
	}
}
