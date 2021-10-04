// Package agent implements a service that can create, delete, manage, etc. local or remote game installations.
// The agent communicates with the main gbans instance via websockets over the `/api/ws` route
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/leighmacdonald/gbans/internal/web/ws"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"strings"
)

// instance represents a unique game instance on a single physical machine.
type instance struct {
	Name string
	// sv_logsecret to identify the server sending the entry, this must be unique
	// for logs to be associated with the correct server instance
	Secret []byte
}

// InstanceCollection represents a collection of instance configurations
type InstanceCollection []instance

// Opts defines what kind of options to use for the agent and connections
type Opts struct {
	ServerAddress    string
	LogListenAddress string
	Instances        InstanceCollection
}

// agent implements the actual agent functionality
type agent struct {
	ctx         context.Context
	messageChan chan string
	opts        Opts
	client      *ws.Client
	l           log.Logger
}

// NewAgent allocates and configures a new agent instance
func NewAgent(ctx context.Context, o Opts) (*agent, error) {
	handlers := ws.Handlers{
		ws.Sup: func(payload ws.Payload) error {
			log.Debugf("Got sup.")
			return nil
		},
	}
	c, err := ws.NewClient(o.ServerAddress, handlers)
	if err != nil {
		return nil, err
	}
	return &agent{ctx: ctx, opts: o, messageChan: make(chan string), client: c}, nil
}

var header = []byte{255, 255, 255, 255}

const (
	mbNoSecret = 0x52
	mbSecret   = 0x53
)

// Start initiates the local UDP based log listener socket as well as starts the main
// agent connection handler loop.
func (a *agent) Start() error {
	go func() {
		if err := a.logListener(); err != nil {
			log.Errorf("Log listener returned err: %v", err)
		}
	}()
	return a.client.Start()
}

// logListener receives srcds log broadcasts
// mp_logdetail 3
// sv_logsecret xxx
// logaddress_add 192.168.0.101:7777
// log on
func (a *agent) logListener() error {
	l, err := net.ListenPacket("udp", a.opts.LogListenAddress)
	if err != nil {
		return err
	}
	defer func() {
		if errC := l.Close(); errC != nil {
			log.Errorf("Failed to close log LogListener: %v", errC)
		}
	}()
	a.l.WithFields(log.Fields{"addr": a.opts.LogListenAddress}).Debugf("Listening")
	doneChan := make(chan error, 1)
	buffer := make([]byte, 1024)
	log.Infof("Listening on: %s", a.opts.LogListenAddress)
	go func() {
		var (
			n     int
			cAddr net.Addr
			errR  error
		)
		for {
			n, cAddr, errR = l.ReadFrom(buffer)
			if errR != nil {
				log.Errorf("Failed to read from udp buff: %v", errR)
				// doneChan <- errR
				continue
			}
			log.WithFields(log.Fields{"addr": cAddr.String(), "size": n}).Debugf("Got log message")
			if n < 16 {
				a.l.Warnf("Recieved payload too small")
				continue
			}
			if !bytes.Equal(buffer[0:4], header) {
				a.l.Warnf("Got invalid header")
				continue
			}
			var msg string
			idx := bytes.Index(buffer, []byte("L "))
			switch buffer[4] {
			case mbSecret:
				// has password
				if idx >= 0 {
					pw := buffer[5:idx]
					log.Debugln(string(pw))
				}
				msg = string(buffer[idx : n-2])
			case mbNoSecret:
				msg = string(buffer[idx : n-2])
			default:
				log.Errorf("Invalid log message type")
				continue
			}
			msg = strings.TrimRight(msg, "\r\n")
			b, errEnv := json.Marshal(ws.LogPayload{
				ServerName: "",
				Message:    msg,
			})
			if errEnv != nil {
				continue
			}
			if errSend := a.client.Send(ws.Payload{
				Type: ws.SrvLogRaw,
				Data: b,
			}); errSend != nil {
				if errors.Is(errSend, ws.ErrQueueFull) {
					log.WithFields(log.Fields{"msg": msg}).Debugf("Msg discarded")
					continue
				}
				if errSend != io.EOF {
					doneChan <- errSend
					return
				}
			}
		}
	}()
	select {
	case <-a.ctx.Done():
		err = a.ctx.Err()
	case errD := <-doneChan:
		log.Errorf("Received error: %v", errD)
	}
	return err
}
