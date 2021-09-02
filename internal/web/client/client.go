package client

import (
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/gbans/internal/web"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	sendQueueSize = 100
)

var (
	ErrClosed = errors.New("Conn closed")
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
	*sync.RWMutex
	conn           *websocket.Conn
	state          int32
	SendQ          chan []byte
	RecvQ          chan web.SocketPayload
	address        string
	auth           web.SocketAuthReq
	ctx            context.Context
	sendBytesCount int64
	recvBytesCount int64
	sendCount      int64
	recvCount      int64
	sendErrCount   int64
	recvErrCount   int64
	// Connect takes care of opening a connection and sending the authentication
	// This function must be idempotent
	Connect func()
}

func (c *Client) Log() *log.Entry {
	addr := ""
	if c.State() != web.Closed && c.isOpen() {
		addr = c.conn.RemoteAddr().String()
	}
	return log.WithFields(log.Fields{"state": StateName[c.State()], "addr": addr})
}

func (c *Client) State() web.State {
	return web.State(atomic.LoadInt32(&c.state))
}

func (c *Client) Close() error {
	if c.isOpen() && web.State(c.state) != web.Closed {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) isOpen() bool {
	c.RLock()
	defer c.RUnlock()
	return c.conn != nil
}

// connect takes care of initiating the websocket connection with the backend server.
// It will immediately attempt to authenticate itself and bail if a success response is not received
// This 2 way communication is done in lock-step unlike the rest of the communication that occurs asynchronously
// as its a requirement for all subsequent requests and we cannot pass auth headers with websockets.
func (c *Client) connect() error {
	if c.isOpen() {
		log.Infof("Reconnecting to server")
		if e := c.conn.Close(); e != nil {
			c.Log().Errorf("error closing ws conn: %v", e)
		}
	}
	c.Log().Debugf("Dialing host: %s", c.address)
	conn, _, err := websocket.DefaultDialer.DialContext(c.ctx, c.address, http.Header{})
	if err != nil {
		return errors.Wrapf(err, "Failed to dial server")
	}
	atomic.SwapInt32(&c.state, int32(web.Opened))
	c.Lock()
	c.conn = conn
	c.Unlock()
	return nil
}

func (c *Client) authenticate() error {
	atomic.SwapInt32(&c.state, int32(web.AwaitingAuthentication))
	c.Log().Debugf("Sending client auth")
	p, errEnc := web.EncodeWSPayload(web.AuthType, c.auth)
	if errEnc != nil {
		return errors.Wrapf(errEnc, "Failed to encode ws payload: %v", errEnc)
	}
	return c.WriteJSON(p)
}

func (c *Client) stats() {
	statTicker := time.NewTicker(time.Second * 5)
	defer c.Log().Debugf("Stats closed")
	for {
		select {
		case <-statTicker.C:
			c.Log().Infof("[ConnStats] Sent: %d Recv: %d SentB: %d RecvB: %d ErrSend: %d ErrRecv: %d",
				atomic.SwapInt64(&c.sendCount, 0),
				atomic.SwapInt64(&c.recvCount, 0),
				atomic.LoadInt64(&c.sendBytesCount),
				atomic.LoadInt64(&c.recvBytesCount),
				atomic.SwapInt64(&c.recvErrCount, 0),
				atomic.SwapInt64(&c.sendErrCount, 0))
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Client) ReadJSON(v interface{}) error {
	if !c.isOpen() {
		return ErrClosed
	}
	c.Lock()
	defer c.Unlock()
	return c.conn.ReadJSON(v)
}

func (c *Client) WriteJSON(v []byte) error {
	if !c.isOpen() {
		return ErrClosed
	}
	c.Lock()
	defer c.Unlock()
	return c.conn.WriteMessage(websocket.TextMessage, v)
}

func (c *Client) reader() {
	running := true
	go func() {
		<-c.ctx.Done()
		running = false
	}()
	authFailShown := false
	for running {
		if !c.isOpen() || c.State() == web.Closed {
			time.Sleep(time.Millisecond * 10)
			continue
		}
		var p web.SocketPayload
		if err := c.ReadJSON(&p); err != nil {
			c.Log().Errorf("failed to read json payload")
			atomic.SwapInt32(&c.state, int32(web.Closed))
			atomic.AddInt64(&c.recvErrCount, 1)
			continue
		}
		log.Debugln(p)
		switch p.PayloadType {
		case web.ErrType:
			var wsErr web.WSErrRes
			if ej := json.Unmarshal(p.Data, &wsErr); ej != nil {
				c.Log().Errorf("Failed to unmarshal err response: %v", ej)
				continue
			}
			c.Log().Errorf("wserr: %s", wsErr.Error)
		case web.AuthFailType:
			var wsErr web.WSErrRes
			if ej := json.Unmarshal(p.Data, &wsErr); ej != nil {
				c.Log().Errorf("Failed to unmarshal auth fail response: %v", ej)
				continue
			}
			atomic.SwapInt32(&c.state, int32(web.AwaitingAuthentication))
			if !authFailShown {
				c.Log().Errorf("Auth failed: %s", wsErr.Error)
				authFailShown = true
			}
		case web.AuthOKType:
			atomic.SwapInt32(&c.state, int32(web.Authenticated))
			authFailShown = false
		}
		atomic.AddInt64(&c.recvBytesCount, int64(len(p.Data)+1)) // approx
		atomic.AddInt64(&c.recvCount, 1)
	}
	c.Log().Debugf("Reader closed")
}

//func (c *Client) writer() {
//	defer c.Log().Debugf("Writer closed")
//	for {
//		select {
//		case p := <-c.SendQ:
//			if err := c.WriteJSON(p); err != nil {
//				c.Log().Errorf("failed to write json payload")
//				atomic.AddInt64(&c.recvErrCount, 1)
//				continue
//			}
//			atomic.AddInt64(&c.sendBytesCount, int64(len(p)))
//			atomic.AddInt64(&c.sendCount, 1)
//		case <-c.ctx.Done():
//			return
//		}
//	}
//
//}

func (c *Client) SetState(s web.State) {
	atomic.SwapInt32(&c.state, int32(s))
}

func New(ctx context.Context, host string, serverName string, token string) (*Client, error) {
	if host == "" {
		return nil, errors.New("Invalid host")
	}
	if token == "" {
		return nil, errors.New("Empty password invalid")
	}
	c := &Client{
		RWMutex: &sync.RWMutex{},
		state:   int32(web.Closed),
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
				c.SetState(web.Closed)
				c.Log().Errorf("Failed to connect: %v", err)
				return
			}
			c.Log().Infof("Connected successfully")
		}
		if c.State() == web.Opened {
			if errA := c.authenticate(); errA != nil {
				c.Log().Errorf("Error sending auth: %v", errA)
			}
		}
	}
	go c.reader()
	go c.stats()
	return c, nil
}
