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

type LogEventHandler func(EventType, ServerEvent)

// Listener handles reading inbound srcds log packets.
type Listener struct {
	*sync.RWMutex

	udpAddr   *net.UDPAddr
	secretMap map[int]ServerIDMap // index = logsecret key
	serverMap map[netip.Addr]bool // index = server ip address
	onEvent   func(EventType, ServerEvent)
}

var (
	ErrResolve   = errors.New("failed to resolve UDP address")
	ErrUnknownIP = errors.New("unknown source ip")
)

func NewListener(logAddr string, onEvent LogEventHandler) (*Listener, error) {
	listenAddress, errResolveUDP := net.ResolveUDPAddr("udp4", logAddr)
	if errResolveUDP != nil {
		return nil, errors.Join(errResolveUDP, ErrResolve)
	}

	return &Listener{
		RWMutex:   &sync.RWMutex{},
		onEvent:   onEvent,
		udpAddr:   listenAddress,
		secretMap: map[int]ServerIDMap{},
		serverMap: map[netip.Addr]bool{},
	}, nil
}

func (remoteSrc *Listener) SetSecrets(secrets map[int]ServerIDMap) {
	remoteSrc.Lock()
	defer remoteSrc.Unlock()

	remoteSrc.secretMap = secrets
}

func (remoteSrc *Listener) SetServers(servers map[netip.Addr]bool) {
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
func (remoteSrc *Listener) Start(ctx context.Context) { //nolint:cyclop
	type newMsg struct {
		source int64
		body   string
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
					slog.Warn("UDP log read error", slog.String("string", errReadUDP.Error()))

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
							slog.String("string", ErrUnknownIP.Error()))
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
							slog.String("error", errConv.Error()))

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
					slog.String("error", errLogServerEvent.Error()))

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
	*Results

	ServerID   int
	ServerName string
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
