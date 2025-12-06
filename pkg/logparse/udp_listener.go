package logparse

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"
)

type srcdsPacket byte

const (
	// Normal log messages (unsupported).
	s2aLogString srcdsPacket = 0x52
	// Sent when using sv_logsecret.
	s2aLogString2 srcdsPacket = 0x53
)

// PacketAuthenticator is responsible for validating the incoming packet secret.
type PacketAuthenticator func(secret int64, clientIP net.IP) (int, string, error)

type LogEventHandler func(EventType, ServerEvent)

// Listener handles reading inbound srcds log packets.
type Listener struct {
	*sync.RWMutex

	packetAuth PacketAuthenticator
	udpAddr    *net.UDPAddr
	secretMap  map[int]ServerIDMap // index = logsecret key
	serverMap  map[netip.Addr]bool // index = server ip address
	onEvent    func(EventType, ServerEvent)
}

var ErrResolve = errors.New("failed to resolve UDP address")

func NewListener(logAddr string, onEvent LogEventHandler, authenticator PacketAuthenticator) (*Listener, error) {
	listenAddress, errResolveUDP := net.ResolveUDPAddr("udp4", logAddr)
	if errResolveUDP != nil {
		return nil, errors.Join(errResolveUDP, ErrResolve)
	}

	return &Listener{
		RWMutex:    &sync.RWMutex{},
		onEvent:    onEvent,
		udpAddr:    listenAddress,
		packetAuth: authenticator,
	}, nil
}

type ServerIDMap struct {
	ServerID   int
	ServerName string
}

// Start initiates the udp network log read loop. DNS names are used to
// map the server logs to the internal known server id. The DNS is updated
// every 60 minutes so that it remains up to date.
func (remoteSrc *Listener) Start(ctx context.Context) { //nolint:cyclop
	type newMsg struct {
		body       string
		serverID   int
		serverName string
	}

	connection, errListenUDP := net.ListenUDP("udp4", remoteSrc.udpAddr)
	if errListenUDP != nil {
		slog.Error("Failed to start log listener", slog.String("error", errListenUDP.Error()))

		return
	}

	defer func() {
		if errConnClose := connection.Close(); errConnClose != nil {
			slog.Error("Failed to close connection cleanly", slog.String("error", errConnClose.Error()))
		}
	}()

	slog.Info("Starting log reader",
		slog.String("listen_addr", remoteSrc.udpAddr.String()+"/udp"))

	var (
		count          = uint64(0)
		insecureCount  = uint64(0)
		errCount       = uint64(0)
		msgIngressChan = make(chan newMsg)
	)

	go func() {
		startTime := time.Now()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				buffer := make([]byte, 1024)

				readLen, remoteAddr, errReadUDP := connection.ReadFromUDP(buffer)
				if errReadUDP != nil {
					slog.Warn("UDP log read error", slog.String("string", errReadUDP.Error()))

					continue
				}

				switch srcdsPacket(buffer[4]) {
				case s2aLogString: // Legacy/insecure format (no secret)
					if insecureCount%10000 == 0 {
						slog.Error("Using unsupported log packet type 0x52",
							slog.Int64("count", int64(insecureCount+1))) // nolint:gosec
					}

					insecureCount++
					errCount++

				case s2aLogString2: // Secure format (with secret)
					line := string(buffer)

					idx := strings.Index(line, "L ")
					if idx == -1 {
						slog.Warn("Received malformed log message: Failed to find marker")

						errCount++

						continue
					}

					secret, errConv := strconv.ParseInt(line[5:idx], 10, 32)
					if errConv != nil {
						slog.Error("Received malformed log message: Failed to parse secret",
							slog.String("error", errConv.Error()))

						errCount++

						continue
					}

					// IP Check: Ensure the packet originates from a known server IP
					serverID, serverName, errAuth := remoteSrc.packetAuth(secret, remoteAddr.IP)
					if errAuth != nil {
						continue
					}

					msgIngressChan <- newMsg{body: line[idx : readLen-2], serverID: serverID, serverName: serverName}

					count++

					if count%10000 == 0 {
						rate := float64(count) / time.Since(startTime).Seconds()

						slog.Debug("UDP SRCDS Logger Packets",
							slog.Uint64("count", count),
							slog.Float64("messages/sec", rate),
							slog.Uint64("errors", errCount))

						startTime = time.Now()
					}
				}
			}
		}
	}()

	parser := NewLogParser()

	for {
		select {
		case <-ctx.Done():
			return
		case logPayload := <-msgIngressChan:
			event, errLogServerEvent := logToServerEvent(parser, logPayload.serverID, logPayload.serverName, logPayload.body)
			if errLogServerEvent != nil {
				slog.Error("Failed to create serverEvent",
					slog.String("body", logPayload.body),
					slog.String("error", errLogServerEvent.Error()))

				continue
			}

			if event.EventType == Say || event.EventType == SayTeam {
				slog.Debug("Got chat message", slog.String("body", logPayload.body),
					slog.String("server_name", logPayload.serverName))
			}

			go remoteSrc.onEvent(event.EventType, event)
		}
	}
}

// ServerEvent is a flat struct encapsulating a parsed log event.
type ServerEvent struct {
	Results

	ServerID   int
	ServerName string
}

var ErrLogParse = errors.New("failed to parse log message")

func logToServerEvent(parser *LogParser, serverID int, serverName string, msg string) (ServerEvent, error) {
	event := ServerEvent{ServerID: serverID, ServerName: serverName}

	parseResult, errParse := parser.Parse(msg)
	if errParse != nil {
		return event, errors.Join(errParse, ErrLogParse)
	}

	event.Results = parseResult

	return event, nil
}
