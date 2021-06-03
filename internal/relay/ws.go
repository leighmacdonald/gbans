package relay

import (
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/gbans/internal/web"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
)

var (
	errConnClosed = errors.New("connection is closed")

	sendQueueSize = 100
)

type Client struct {
	conn          *websocket.Conn
	authenticated bool
	SendQ         chan []byte
	RecvQ         chan web.WebSocketPayload
	address       string
	ctx           context.Context
	password      string
	serverName    string
}

func (c *Client) isOpen() bool {
	return c.conn != nil
}

// connect takes care of initiating the websocket connection with the backend server.
// It will immediately attempt to authenticate itself and bail if a success response is not received
// This 2 way communication is done in lock-step unlike the rest of the communication that occurs asynchronously
// as its a requirement for all subsequent requests and we cannot pass auth headers with websockets.
func (c *Client) connect() error {
	if c.conn != nil {
		if e := c.conn.Close(); e != nil {
			log.Errorf("error closing ws conn: %v", e)
		}
		c.conn = nil
	}
	conn, _, err := websocket.DefaultDialer.DialContext(c.ctx, c.address+"/ws", http.Header{})
	if err != nil {
		return errors.Wrapf(err, "Failed to dial server")
	}
	p, errEnc := web.EncodeWSPayload(web.AuthType, web.WebSocketAuthReq{Token: c.password, IsServer: true, ServerName: c.serverName})
	if errEnc != nil {
		return errors.Wrapf(errEnc, "Failed to encode ws payload: %v", err)
	}
	if errW := conn.WriteMessage(websocket.TextMessage, p); errW != nil {
		return errors.Wrapf(errW, "Failed to write payload")
	}
	var resp web.WebSocketPayload
	if errResp := conn.ReadJSON(&resp); errResp != nil {
		return errors.Wrapf(errResp, "Failed to read authentication reply: %v", errResp)
	}
	var authResp web.WebSocketAuthResp
	if errAuthResp := json.Unmarshal(resp.Data, &authResp); errAuthResp != nil {
		return errors.Wrapf(errAuthResp, "Failed to read authentication payload: %v", errAuthResp)
	}
	if !authResp.Status {
		return errors.New("Authentication status failed")
	}
	log.Infof("Connected to %s", c.address)
	c.conn = conn
	return nil
}

func (c *Client) enqueue(payload []byte) error {
	if !c.isOpen() {
		return errConnClosed
	}
	if len(c.SendQ) >= sendQueueSize {
		log.Warnf("message dropped (enqueue queue full)")
		return nil
	}
	c.SendQ <- payload
	return nil
}

func (c *Client) writer() {
	for {
		select {
		case p := <-c.SendQ:
			if err := c.conn.WriteMessage(websocket.TextMessage, p); err != nil {
				log.Errorf("failed to write json payload")
			}
		case <-c.ctx.Done():
			log.Debugf("ws writer stopped")
			return
		}
	}
}

func newClient(ctx context.Context, serverName string, host string, password string) (*Client, error) {
	if host == "" {
		return nil, errors.New("Invalid host")
	}
	if password == "" {
		return nil, errors.New("Empty password invalid")
	}
	c := &Client{
		serverName: serverName,
		conn:       nil,
		SendQ:      make(chan []byte, sendQueueSize),
		RecvQ:      make(chan web.WebSocketPayload),
		ctx:        ctx,
		address:    host,
		password:   password,
	}
	go c.writer()
	return c, nil
}
