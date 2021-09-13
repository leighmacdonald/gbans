package relay

import (
	"bytes"
	"context"
	"github.com/leighmacdonald/gbans/internal/relay/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
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
	pb.AgentClient
	ctx         context.Context
	dialOpts    []grpc.DialOption
	conn        *grpc.ClientConn
	messageChan chan pb.LogEntry
	opts        Opts
	l           log.Logger
}

func NewAgent(ctx context.Context, o Opts) (*Agent, error) {
	var dialOpts []grpc.DialOption
	//if *tls {
	//	if *caFile == "" {
	//		*caFile = data.Path("x509/ca_cert.pem")
	//	}
	//	creds, err := credentials.NewClientTLSFromFile(*caFile, *serverHostOverride)
	//	if err != nil {
	//		log.Fatalf("Failed to create TLS credentials %v", err)
	//	}
	//	dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	//} else {
	//
	//}
	dialOpts = append(dialOpts, grpc.WithInsecure())
	//dialOpts = append(dialOpts, grpc.WithBlock())
	dialOpts = append(dialOpts, grpc.WithUserAgent("gbans"))
	dialOpts = append(dialOpts, grpc.WithReturnConnectionError())

	return &Agent{ctx: ctx, dialOpts: dialOpts, opts: o, messageChan: make(chan pb.LogEntry)}, nil
}

func (a *Agent) Stop() {
	if a.conn != nil {
		if err := a.conn.Close(); err != nil {
			a.l.Errorf("Error shutting down grpc conn: %v", err)
		}
	}
}

func (a *Agent) Start() error {
	a.l.WithFields(log.Fields{"addr": a.opts.ServerAddress}).Debugf("Connecting")
	conn, err := grpc.Dial(a.opts.ServerAddress, a.dialOpts...)
	if err != nil {
		return err
	}
	a.l.WithFields(log.Fields{"addr": a.opts.ServerAddress}).Debugf("Connected")
	a.conn = conn
	defer a.Stop()

	client := pb.NewAgentClient(conn)

	go func() {
		if errL := a.listener(client); errL != nil {
			log.Errorf("Error on rcon listener: %v", errL)
		}
	}()
	<-a.ctx.Done()
	return nil
}

var header = []byte{255, 255, 255, 255}

const (
	mbNoPassword = 0x52
	mbPassword   = 0x53
)

// listener receives srcds log broadcasts
// mp_logdetail 3
// sv_logsecret xxx
// logaddress_add 192.168.0.101:7777
// log on
func (a *Agent) listener(client pb.AgentClient) error {
	l, err := net.ListenPacket("udp", a.opts.LogListenAddress)
	if err != nil {
		return err
	}
	a.l.WithFields(log.Fields{"addr": a.opts.LogListenAddress}).Debugf("Listening")
	defer l.Close()
	doneChan := make(chan error, 1)
	buffer := make([]byte, 1024)
	go func() {
		os, errSL := client.SendLog(a.ctx)
		if errSL != nil {
			doneChan <- errSL
			return
		}
		for {
			n, cAddr, errR := l.ReadFrom(buffer)
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
			if buffer[4] == mbPassword {
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
			log.Debugln(msg)
			if errSend := os.Send(&pb.LogEntry{Server: "xxx", Message: msg}); errSend != nil {
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
