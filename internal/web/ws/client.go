package ws

import (
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
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
}

func (c *Client) onMessage() {
	for {
		select {
		case <-c.done:
			return
		case req := <-c.recvQueue:
			handler, found := c.handlers[req.Type]
			if !found {
				c.log.Warnf("Unhandled payload type: %v")
			}
			if hErr := handler(req); hErr != nil {
				c.log.Errorf("Error handling message: %v", hErr)
			}
		}
	}
}

func (c *Client) reader(conn *websocket.Conn) {
	defer close(c.done)
	for {
		var payload Payload
		if e := conn.ReadJSON(&payload); e != nil {
			c.log.Errorf("Failed to read json from server: %v", e)
			continue
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

func (c *Client) Connect() error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	conn, _, err := websocket.DefaultDialer.Dial(c.host, nil)
	if err != nil {
		return err
	}
	defer func() {
		if errC := conn.Close(); errC != nil {
			c.log.Errorf("Failed closing websocket connection cleanly: %v", errC)
		}
	}()
	c.log.Debugf("Connected")
	go c.onMessage()
	go c.reader(conn)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			c.log.Debugf("Disconnected")
			return nil
		case <-ticker.C:
			if errSup := c.Send(Payload{Type: Sup, Data: nil}); errSup != nil {
				c.log.Errorf("Failed to send sup: %v", errSup)
				continue
			}
			c.log.Debugf("Send sup")
		case p := <-c.sendQueue:
			if errWrite := conn.WriteJSON(p); errWrite != nil {
				c.log.Errorf("Failed to write json to ws")
			}
		}
	}
}
