package agent

import (
	"bytes"
	"context"
	"github.com/leighmacdonald/gbans/internal/web"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"strings"
)

type Instance struct {
	Name string
	// sv_logsecret to identify the server sending the entry
	Secret []byte
}

type Opts struct {
	ServerAddress    string
	LogListenAddress string
	Instances        []Instance
}

type Agent struct {
	ctx         context.Context
	messageChan chan string
	opts        Opts
	client      *web.RPCClient
	l           log.Logger
}

func NewAgent(ctx context.Context, o Opts) (*Agent, error) {
	c, err := web.NewRPCClient(o.ServerAddress)
	if err != nil {
		return nil, err
	}
	return &Agent{ctx: ctx, opts: o, messageChan: make(chan string), client: c}, nil
}

var header = []byte{255, 255, 255, 255}

const (
	mbNoSecret = 0x52
	mbSecret   = 0x53
)

func (a *Agent) connect() error {
	go a.client.Start()
	return nil
}

func (a *Agent) Start() error {
	// TODO auto retry/error handling
	if err3 := a.connect(); err3 != nil {
		log.Errorf("Agent returned error: %v", err3)
	}
	go a.LogListener()
	return nil
}

// LogListener receives srcds log broadcasts
// mp_logdetail 3
// sv_logsecret xxx
// logaddress_add 192.168.0.101:7777
// log on
func (a *Agent) LogListener() error {
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
	go func() {
		var (
			n     int
			cAddr net.Addr
			errR  error
		)
		for {
			n, cAddr, errR = l.ReadFrom(buffer)
			if errR != nil {
				doneChan <- errR
				return
			}
			log.WithFields(log.Fields{"addr": cAddr.String(), "size": n}).Debugf("Got log message")
			if n < 16 {
				a.l.Warnf("Recieved payload too small")
				continue
			}
			if bytes.Compare(buffer[0:4], header) != 0 {
				a.l.Warnf("Got invalid header")
				continue
			}
			var msg string
			idx := bytes.Index(buffer, []byte("L "))
			if buffer[4] == mbSecret {
				// has password
				if idx >= 0 {
					pw := buffer[5:idx]
					log.Debugln(string(pw))
				}
				msg = string(buffer[idx : n-2])
			} else {
				msg = string(buffer[idx : n-2])
			}
			msg = strings.TrimRight(msg, "\r\n")
			if errSend := a.client.Send(web.SrvLogRaw, msg); errSend != nil {
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
