package relay

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/relay/pb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io"
	"net"
	"time"
)

type Server struct {
	pb.UnimplementedAgentServer
	ctx  context.Context
	addr string
}

func NewServer(ctx context.Context, addr string) Server {
	return Server{
		ctx:  ctx,
		addr: addr,
	}
}

func (s *Server) FetchLog(stream pb.Agent_SendLogServer) error {
	var recvTotal int64
	startTime := time.Now()
	for {
		entry, err := stream.Recv()
		if err == io.EOF {
			endTime := time.Now()
			return stream.SendAndClose(&pb.SendLogSummary{
				MessageCount: recvTotal,
				ElapsedTime:  int32(endTime.Sub(startTime).Seconds()),
			})
		}
		if err != nil {
			log.Errorf("Failed to send log entry: %v", err)
		}
		recvTotal++
		log.Debugln(entry.Message)
	}
}

func (s *Server) Start() error {
	grpcSrv := grpc.NewServer()
	pb.RegisterAgentServer(grpcSrv, s)
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return errors.Wrap(err, "Failed to create listener")
	}
	log.WithFields(log.Fields{"addr": s.addr}).Infof("RPC Listener started")
	return grpcSrv.Serve(listener)
}
