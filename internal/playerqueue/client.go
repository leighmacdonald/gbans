package playerqueue

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
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
	ErrClosedConnection  = errors.New("closed connection")
)

func newClient(steamID steamid.SteamID, name string, avatarHash string, conn *websocket.Conn) Client {
	client := &client{
		steamID:      steamID,
		name:         name,
		avatarhash:   avatarHash,
		conn:         conn,
		responseChan: make(chan Response),
		rl:           ratelimit.New(1, ratelimit.Per(5*time.Second)),
	}

	return client
}

type client struct {
	steamID      steamid.SteamID
	name         string
	avatarhash   string
	conn         *websocket.Conn
	responseChan chan Response
	chatStatus   ChatStatus
	rl           ratelimit.Limiter
}

func (c *client) Limit() {
	c.rl.Take()
}

func (c *client) HasMessageAccess() bool {
	return c.chatStatus != Noaccess
}

func (c *client) Send(response Response) {
	c.responseChan <- response
}

func (c *client) SteamID() steamid.SteamID {
	return c.steamID
}

func (c *client) Name() string {
	return c.name
}

func (c *client) Avatarhash() string {
	return c.avatarhash
}

func (c *client) Next(request *Request) error {
	c.rl.Take()
	if err := c.conn.ReadJSON(request); err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			return ErrClosedConnection
		}

		return errors.Join(err, ErrReadRequest)
	}

	return nil
}

func (c *client) ID() string {
	return c.conn.RemoteAddr().String()
}

func (c *client) Start(ctx context.Context) {
	// TODO refactor this so there is none of this logic under domain.
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

func (c *client) Close() {
	slog.Debug("Closing client connection", slog.String("addr", c.conn.RemoteAddr().String()))
	if errClose := c.conn.Close(); errClose != nil {
		slog.Warn("Error closing client connection", log.ErrAttr(errClose))
	}
}
