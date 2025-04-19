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

	"github.com/leighmacdonald/gbans/pkg/log"
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

	udpAddr   *net.UDPAddr
	secretMap map[int]ServerIDMap // index = logsecret key
	serverMap map[netip.Addr]bool // index = server ip address
	onEvent   func(EventType, ServerEvent)
}

var (
	ErrResolve    = errors.New("failed to resolve UDP address")
	ErrRateLimit  = errors.New("rate limited")
	ErrUnknownIP  = errors.New("unknown source ip")
	ErrSecretAuth = errors.New("failed secret auth")
)

func NewUDPLogListener(logAddr string, onEvent LogEventHandler) (*UDPLogListener, error) {
	udpAddr, errResolveUDP := net.ResolveUDPAddr("udp4", logAddr)
	if errResolveUDP != nil {
		return nil, errors.Join(errResolveUDP, ErrResolve)
	}

	return &UDPLogListener{
		RWMutex:   &sync.RWMutex{},
		onEvent:   onEvent,
		udpAddr:   udpAddr,
		secretMap: map[int]ServerIDMap{},
		serverMap: map[netip.Addr]bool{},
	}, nil
}

func (remoteSrc *UDPLogListener) SetSecrets(secrets map[int]ServerIDMap) {
	remoteSrc.Lock()
	defer remoteSrc.Unlock()

	remoteSrc.secretMap = secrets
}

func (remoteSrc *UDPLogListener) SetServers(servers map[netip.Addr]bool) {
	remoteSrc.Lock()
	defer remoteSrc.Unlock()

	remoteSrc.serverMap = servers
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
		slog.Error("Failed to start log listener", log.ErrAttr(errListenUDP))

		return
	}

	defer func() {
		if errConnClose := connection.Close(); errConnClose != nil {
			slog.Error("Failed to close connection cleanly", log.ErrAttr(errConnClose))
		}
	}()

	slog.Info("Starting log reader",
		slog.String("listen_addr", remoteSrc.udpAddr.String()+"/udp"))

	var (
		count          = uint64(0)
		insecureCount  = uint64(0)
		errCount       = uint64(0)
		rejectsIP      = map[string]time.Time{} // IP -> last reject time
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
					slog.Warn("UDP log read error", log.ErrAttr(errReadUDP))

					continue
				}

				// IP Check: Ensure the packet originates from a known server IP
				knownIP := false
				if addr, addrOk := netip.AddrFromSlice(remoteAddr.IP); addrOk {
					remoteSrc.RLock()
					_, knownIP = remoteSrc.serverMap[addr]
					remoteSrc.RUnlock()
				}

				if !knownIP {
					lastTime, rejected := rejectsIP[remoteAddr.IP.String()]
					if !rejected || time.Since(lastTime) > time.Minute*5 {
						slog.Warn("Rejecting UDP packet from unknown source IP",
							slog.String("ip", remoteAddr.IP.String()),
							log.ErrAttr(ErrUnknownIP))
						rejectsIP[remoteAddr.IP.String()] = time.Now()
					}

					continue // Discard packet
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
							log.ErrAttr(errConv))

						errCount++

						continue
					}

					msgIngressChan <- newMsg{source: secret, body: line[idx : readLen-2]}

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
	rejects := map[int]time.Time{}

	for {
		select {
		case <-ctx.Done():
			return
		case logPayload := <-msgIngressChan:
			remoteSrc.RLock()
			server, found := remoteSrc.secretMap[int(logPayload.source)]
			remoteSrc.RUnlock()

			if !found {
				lastTime, ok := rejects[int(logPayload.source)]
				if !ok || time.Since(lastTime) > time.Minute*5 {
					slog.Warn("Rejecting unknown secret log author")

					rejects[int(logPayload.source)] = time.Now()
				}

				continue
			}

			event, errLogServerEvent := logToServerEvent(parser, server.ServerID, server.ServerName, logPayload.body)
			if errLogServerEvent != nil {
				slog.Error("Failed to create serverEvent",
					slog.String("body", logPayload.body),
					log.ErrAttr(errLogServerEvent))

				continue
			}

			if event.EventType == Say || event.EventType == SayTeam {
				slog.Info("Got chat message", slog.String("body", logPayload.body), slog.String("server", server.ServerName))
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

var ErrLogParse = errors.New("failed to parse log message")

func logToServerEvent(parser *LogParser, serverID int, serverName string, msg string) (ServerEvent, error) {
	event := ServerEvent{
		ServerID:   serverID,
		ServerName: serverName,
	}

	parseResult, errParse := parser.Parse(msg)

	if errParse != nil {
		return event, errors.Join(errParse, ErrLogParse)
	}

	event.Results = parseResult

	return event, nil
}
