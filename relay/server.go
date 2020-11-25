package relay

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

const MaxBufferSize = 1024
const Timeout = time.Second * 5
const ListenPort = 7777

func StartServer(ctx context.Context, address string) {
	pc, err := net.ListenPacket("udp", address)
	if err != nil {
		return
	}
	defer func() {
		if err := pc.Close(); err != nil {
			log.Errorf("Failed to close client conn: %v", err)
		}
	}()
	doneChan := make(chan error, 1)
	buffer := make([]byte, MaxBufferSize)
	go func() {
		for {
			n, addr, err := pc.ReadFrom(buffer)
			if err != nil {
				doneChan <- err
				return
			}

			fmt.Printf("packet-received: bytes=%d from=%s\n",
				n, addr.String())
			var p Payload
			if err := Decode(buffer[:n], &p); err != nil {
				log.Errorf("failed to decode payload: %v", err)
				continue
			}
			//_ = SendPayload(p)
			panic("send payload not implemented")
		}
	}()

	select {
	case <-ctx.Done():
		fmt.Println("cancelled")
		err = ctx.Err()
	case err = <-doneChan:
	}

	return
}
