package client

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/gbans/internal/web"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	sendQueueSize = 100
)

var (
	errConnClosed = errors.New("connection is closed")
)

var StateName = map[web.State]string{
	web.Closed:                 "closed",
	web.AwaitingAuthentication: "await_auth",
	web.Authenticated:          "authed",
	web.Opened:                 "opened",
	web.Closing:                "closing",
}

// Client represents a generic websocket based api client
type Client struct {
	conn          *websocket.Conn
	authenticated bool
	state         web.State
	SendQ         chan []byte
	RecvQ         chan web.SocketPayload
	address       string
	auth          web.SocketAuthReq
	ctx           context.Context
	sendCount     int64
	recvCount     int64
	sendErrCount  int64
	recvErrCount  int64
	// Connect takes care of opening a connection and sending the authentication
	// This function must be idempotent
	Connect func()
}

func (c *Client) Log() *log.Entry {
	addr := ""
	if c.state != web.Closed && c.conn != nil {
		addr = c.conn.RemoteAddr().String()
	}
	return log.WithFields(log.Fields{"state": fmt.Sprintf("%s", StateName[c.state]), "addr": addr})
}

func (c *Client) State() web.State {
	return c.state
}

func (c *Client) Close() error {
	if c.conn != nil && c.state != web.Closed {
		return c.conn.Close()
	}
	return nil
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
			c.Log().Errorf("error closing ws conn: %v", e)
		}
		c.conn = nil
	}

	c.Log().Debugf("Dialing host: %s", c.address)
	conn, _, err := websocket.DefaultDialer.DialContext(c.ctx, c.address, http.Header{})
	if err != nil {
		return errors.Wrapf(err, "Failed to dial server")
	}
	c.state = web.Opened
	c.Log().Debugf("Sending client auth")
	p, errEnc := web.EncodeWSPayload(web.AuthType, c.auth)
	if errEnc != nil {
		return errors.Wrapf(errEnc, "Failed to encode ws payload: %v", err)
	}
	if errW := conn.WriteMessage(websocket.TextMessage, p); errW != nil {
		c.recvErrCount++
		return errors.Wrapf(errW, "Failed to write payload")
	}
	c.state = web.AwaitingAuthentication
	c.conn = conn
	return nil
}

func (c *Client) Enqueue(payload []byte) error {
	if !c.isOpen() {
		c.Log().Errorf("Enqueue on closed session")
		return errConnClosed
	}
	if len(c.SendQ) >= sendQueueSize {
		c.Log().Warnf("message dropped (Enqueue queue full)")
		return nil
	}
	c.SendQ <- payload
	return nil
}
func (c *Client) stats() {
	statTicker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-statTicker.C:
			c.Log().Infof("[ConnStats] Sent: %d Recv: %d ErrSend: %d ErrRecv: %d",
				atomic.SwapInt64(&c.sendCount, 0),
				atomic.SwapInt64(&c.recvCount, 0),
				atomic.SwapInt64(&c.recvErrCount, 0),
				atomic.SwapInt64(&c.sendErrCount, 0))
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Client) ReadJSON(v interface{}) error {
	return c.conn.ReadJSON(v)
}

func (c *Client) reader() {
	running := true
	go func() {
		<-c.ctx.Done()
		running = false
	}()
	for running {
		if c.state == web.Closed {
			time.Sleep(time.Millisecond * 100)
			continue
		}
		var p web.SocketPayload
		if err := c.conn.ReadJSON(&p); err != nil {
			c.Log().Errorf("failed to read json payload")
			c.state = web.Closed
			c.recvErrCount++
			continue
		}
		c.recvCount++
	}
}

func (c *Client) writer() {
	for {
		select {
		case p := <-c.SendQ:
			if err := c.conn.WriteMessage(websocket.TextMessage, p); err != nil {
				c.Log().Errorf("failed to write json payload")
				c.recvErrCount++
				continue
			}
			c.sendCount++
		case <-c.ctx.Done():
			c.Log().Debugf("ws writer stopped")
			return
		}
	}
}

func New(ctx context.Context, host string, serverName string, token string) (*Client, error) {
	if host == "" {
		return nil, errors.New("Invalid host")
	}
	if token == "" {
		return nil, errors.New("Empty password invalid")
	}
	c := &Client{
		state:   web.Closed,
		conn:    nil,
		SendQ:   make(chan []byte, sendQueueSize),
		RecvQ:   make(chan web.SocketPayload),
		ctx:     ctx,
		address: host + "/ws",
		auth:    web.SocketAuthReq{Token: token, IsServer: true, ServerName: serverName},
	}
	c.Connect = func() {
		if c.State() == web.Closed {
			c.Log().Infof("Connecting to %s", host)
			if err := c.connect(); err != nil {
				c.state = web.Closed
				c.Log().Errorf("Failed to connect: %v", err)
				return
			}
			c.state = web.Authenticated
			c.authenticated = true
			c.Log().Infof("Connected successfully")
		}
	}
	go c.writer()
	go c.reader()
	go c.stats()
	return c, nil
}
