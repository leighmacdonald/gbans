package logparse

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type srcdsPacket byte

const (
	// Normal log messages (unsupported).
	s2aLogString srcdsPacket = 0x52
	// Sent when using sv_logsecret.
	s2aLogString2 srcdsPacket = 0x53
)

type LogEventHandler func(EventType, ServerEvent)

// UDPLogListener handles reading inbound srcds log packets.
type UDPLogListener struct {
	*sync.RWMutex

	logger        *zap.Logger
	udpAddr       *net.UDPAddr
	secretMap     map[int]ServerIDMap // index = logsecret key
	logAddrString string
	onEvent       func(EventType, ServerEvent)
}

func NewUDPLogListener(logger *zap.Logger, logAddr string, onEvent LogEventHandler) (*UDPLogListener, error) {
	udpAddr, errResolveUDP := net.ResolveUDPAddr("udp4", logAddr)
	if errResolveUDP != nil {
		return nil, errors.Wrapf(errResolveUDP, "Failed to resolve UDP address")
	}

	return &UDPLogListener{
		RWMutex:       &sync.RWMutex{},
		onEvent:       onEvent,
		logger:        logger.Named("srcdsLog"),
		udpAddr:       udpAddr,
		secretMap:     map[int]ServerIDMap{},
		logAddrString: logAddr,
	}, nil
}

func (remoteSrc *UDPLogListener) SetSecrets(secrets map[int]ServerIDMap) {
	remoteSrc.Lock()
	defer remoteSrc.Unlock()

	remoteSrc.secretMap = secrets
	remoteSrc.logger.Debug("Updated server id map")
}

type ServerIDMap struct {
	ServerID   int
	ServerName string
}

// Start initiates the udp network log read loop. DNS names are used to
// map the server logs to the internal known server id. The DNS is updated
// every 60 minutes so that it remains up to date.
func (remoteSrc *UDPLogListener) Start(ctx context.Context) {
	type newMsg struct {
		source int64
		body   string
	}

	connection, errListenUDP := net.ListenUDP("udp4", remoteSrc.udpAddr)
	if errListenUDP != nil {
		remoteSrc.logger.Error("Failed to start log listener", zap.Error(errListenUDP))

		return
	}

	defer func() {
		if errConnClose := connection.Close(); errConnClose != nil {
			remoteSrc.logger.Error("Failed to close connection cleanly", zap.Error(errConnClose))
		}
	}()

	remoteSrc.logger.Info("Starting log reader", zap.String("listen_addr", fmt.Sprintf("%s/udp", remoteSrc.udpAddr.String())))

	var (
		running        = atomic.NewBool(true)
		count          = uint64(0)
		insecureCount  = uint64(0)
		errCount       = uint64(0)
		msgIngressChan = make(chan newMsg)
	)

	go func() {
		startTime := time.Now()

		for running.Load() {
			buffer := make([]byte, 1024)

			readLen, _, errReadUDP := connection.ReadFromUDP(buffer)
			if errReadUDP != nil {
				remoteSrc.logger.Warn("UDP log read error", zap.Error(errReadUDP))

				continue
			}

			switch srcdsPacket(buffer[4]) {
			case s2aLogString:
				if insecureCount%10000 == 0 {
					remoteSrc.logger.Error("Using unsupported log packet type 0x52",
						zap.Int64("count", int64(insecureCount+1)))
				}
				insecureCount++
				errCount++
			case s2aLogString2:
				line := string(buffer)

				idx := strings.Index(line, "L ")
				if idx == -1 {
					remoteSrc.logger.Warn("Received malformed log message: Failed to find marker")
					errCount++

					continue
				}

				secret, errConv := strconv.ParseInt(line[5:idx], 10, 32)
				if errConv != nil {
					remoteSrc.logger.Error("Received malformed log message: Failed to parse secret",
						zap.Error(errConv))
					errCount++

					continue
				}

				msgIngressChan <- newMsg{source: secret, body: line[idx : readLen-2]}
				count++

				if count%10000 == 0 {
					rate := float64(count) / time.Since(startTime).Seconds()

					remoteSrc.logger.Debug("UDP SRCDS Logger Packets",
						zap.Uint64("count", count),
						zap.Float64("messages/sec", rate),
						zap.Uint64("errors", errCount))

					startTime = time.Now()
				}
			}
		}
	}()

	parser := NewLogParser()

	rejects := map[int]time.Time{}

	for {
		select {
		case <-ctx.Done():
			running.Store(false)
		case logPayload := <-msgIngressChan:
			remoteSrc.RLock()
			server, found := remoteSrc.secretMap[int(logPayload.source)]
			remoteSrc.RUnlock()

			if !found {
				lastTime, ok := rejects[int(logPayload.source)]
				if !ok || time.Since(lastTime) > time.Minute*5 {
					remoteSrc.logger.Error("Rejecting unknown secret log author")

					rejects[int(logPayload.source)] = time.Now()
				}

				continue
			}

			event, errLogServerEvent := logToServerEvent(parser, server.ServerID, server.ServerName, logPayload.body)
			if errLogServerEvent != nil {
				remoteSrc.logger.Error("Failed to create serverEvent",
					zap.String("body", logPayload.body),
					zap.Error(errLogServerEvent))

				continue
			}

			remoteSrc.onEvent(event.EventType, event)
		}
	}
}

// ServerEvent is a flat struct encapsulating a parsed log event.
type ServerEvent struct {
	ServerID   int
	ServerName string
	*Results
}

func logToServerEvent(parser *LogParser, serverID int, serverName string, msg string) (ServerEvent, error) {
	event := ServerEvent{
		ServerID:   serverID,
		ServerName: serverName,
	}
	parseResult, errParse := parser.Parse(msg)

	if errParse != nil {
		return event, errors.Wrapf(errParse, "Failed to parse log message")
	}

	event.Results = parseResult

	return event, nil
}
