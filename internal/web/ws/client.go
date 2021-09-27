package ws

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"math/rand"
	"sync"
	"time"
)

func NewClient(host string, handlers Handlers) (*Client, error) {
	c := &Client{
		host:      host,
		handlers:  handlers,
		log:       log.WithFields(log.Fields{"host": host}),
		done:      make(chan struct{}),
		recvQueue: make(chan Payload, 100),
		sendQueue: make(chan Payload, 100),
		connMu:    &sync.RWMutex{},
	}
	return c, nil
}

// Client provides a generic websocket client for agent connections
type Client struct {
	host      string
	sendQueue chan Payload
	recvQueue chan Payload
	log       *log.Entry
	handlers  Handlers
	done      chan struct{}
	conn      *websocket.Conn
	connMu    *sync.RWMutex
	bytesSent int64
	bytesRecv int64
}

func (c *Client) onMessage() {
	for {
		select {
		case <-c.done:
			return
		case req := <-c.recvQueue:
			handler, found := c.handlers[req.Type]
			if !found {
				c.log.Warnf("Unhandled payload type: %v", req.Type)
				continue
			}
			if hErr := handler(req); hErr != nil {
				c.log.Errorf("Error handling message: %v", hErr)
			}
		}
	}
}

func (c *Client) reader() {
	for {
		mt, r, err := c.conn.NextReader()
		if err != nil {
			c.connMu.Lock()
			c.conn = nil
			c.connMu.Unlock()
			c.log.Errorf("Reader error: %v", err)
			return
		}
		if mt == websocket.BinaryMessage {
			log.Debugf("Got non text message")
			continue
		}
		var payload Payload
		err = json.NewDecoder(r).Decode(&payload)
		if err == io.EOF {
			// One value is expected in the message.
			err = io.ErrUnexpectedEOF
		}
		if err != nil {
			c.log.Errorf("Error decoding json from server: %v", err)
			return
		}
		c.recvQueue <- payload
	}
}

func (c *Client) Send(payload Payload) error {
	select {
	case c.sendQueue <- payload:
		return nil
	default:
		return ErrQueueFull
	}
}

func (c *Client) connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.host, nil)
	if err != nil {
		return err
	}
	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()
	return nil
}

func (c *Client) connLoop() error {
	c.done = make(chan struct{})
	defer close(c.done)
	if errConn := c.connect(); errConn != nil {
		return errors.Wrapf(errConn, "Failed to connect to ws api")
	}
	defer func() {
		if errC := c.conn.Close(); errC != nil {
			c.log.Errorf("Failed closing websocket connection cleanly: %v", errC)
		}
	}()
	c.log.Debugf("Connected")
	go c.onMessage()
	go c.reader()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			c.log.Debugf("Disconnected")
			return nil
		case <-ticker.C:
			b, err := json.Marshal(Ping{Nonce: rand.Int63()})
			if err != nil {
				return err
			}
			if errSup := c.Send(Payload{Type: Sup, Data: b}); errSup != nil && !errors.Is(errSup, ErrQueueFull) {
				return errors.Wrapf(errSup, "Failed to send sup response")
			}
			c.log.Debugf("Send sup")
		case p := <-c.sendQueue:
			if c.conn != nil {
				if errWrite := c.conn.WriteJSON(p); errWrite != nil {
					return errors.Wrapf(errWrite, "Failed to write json to ws")
				}
			}
		}
	}
}

func (c *Client) Start() error {
	for {
		log.Debugf("Initiating ws connection")
		if err := c.connLoop(); err != nil {
			log.Errorf("Conn error (reconnecting): %v", err)
			time.Sleep(time.Second * 10)
		}
	}
}
