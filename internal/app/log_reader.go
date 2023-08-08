package app

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/logparse"
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

// remoteSrcdsLogSource handles reading inbound srcds log packets, and emitting a web.LogPayload
// that can be further parsed/processed.
//
// On, start and every hour after, a new sv_logsecret value for every instance is randomly generated and
// assigned remotely over rcon. This allows us to associate certain semi secret id's with specific server
// instances.
type remoteSrcdsLogSource struct {
	*sync.RWMutex
	db            *store.Store
	eb            *eventBroadcaster
	logger        *zap.Logger
	udpAddr       *net.UDPAddr
	secretMap     map[int]string
	frequency     time.Duration
	logAddrString string
}

func newRemoteSrcdsLogSource(logger *zap.Logger, database *store.Store, logAddr string, broadcaster *eventBroadcaster) (*remoteSrcdsLogSource, error) {
	udpAddr, errResolveUDP := net.ResolveUDPAddr("udp4", logAddr)
	if errResolveUDP != nil {
		return nil, errors.Wrapf(errResolveUDP, "Failed to resolve UDP address")
	}

	return &remoteSrcdsLogSource{
		RWMutex:       &sync.RWMutex{},
		eb:            broadcaster,
		db:            database,
		logger:        logger.Named("srcdsLog"),
		udpAddr:       udpAddr,
		secretMap:     map[int]string{},
		logAddrString: logAddr,
		frequency:     time.Minute * 5,
	}, nil
}

func (remoteSrc *remoteSrcdsLogSource) updateSecrets(ctx context.Context) {
	newServers := map[int]string{}
	serversCtx, cancelServers := context.WithTimeout(ctx, time.Second*5)

	defer cancelServers()

	servers, errServers := remoteSrc.db.GetServers(serversCtx, true)
	if errServers != nil {
		remoteSrc.logger.Error("Failed to load servers to update DNS", zap.Error(errServers))

		return
	}

	for _, server := range servers {
		newServers[server.LogSecret] = server.ServerName
	}

	remoteSrc.Lock()
	defer remoteSrc.Unlock()

	remoteSrc.secretMap = newServers
	remoteSrc.logger.Debug("Updated secret mappings")
}

// start initiates the udp network log read loop. DNS names are used to
// map the server logs to the internal known server id. The DNS is updated
// every 60 minutes so that it remains up to date.
func (remoteSrc *remoteSrcdsLogSource) start(ctx context.Context) {
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

	remoteSrc.updateSecrets(ctx)

	remoteSrc.logger.Info("Starting log reader", zap.String("listen_addr", fmt.Sprintf("%s/udp", remoteSrc.udpAddr.String())))

	var (
		running        = atomic.NewBool(true)
		count          = uint64(0)
		insecureCount  = uint64(0)
		errCount       = uint64(0)
		msgIngressChan = make(chan newMsg)
	)

	go func() {
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
					remoteSrc.logger.Debug("UDP SRCDS Logger Packets",
						zap.Uint64("count", count), zap.Uint64("errors", errCount))
				}
			}
		}
	}()

	var (
		parser = logparse.New()
		ticker = time.NewTicker(remoteSrc.frequency)
	)

	serverCache := map[string]store.Server{}

	for {
		select {
		case <-ctx.Done():
			running.Store(false)
		case <-ticker.C:
			remoteSrc.updateSecrets(ctx)

			serverCache = map[string]store.Server{}
		case logPayload := <-msgIngressChan:
			var serverName string

			remoteSrc.RLock()
			serverNameValue, found := remoteSrc.secretMap[int(logPayload.source)]
			remoteSrc.RUnlock()

			if !found {
				remoteSrc.logger.Error("Rejecting unknown secret log author")

				continue
			}

			serverName = serverNameValue

			server, serverFound := serverCache[serverName]
			if !serverFound {
				if errServer := remoteSrc.db.GetServerByName(ctx, serverName, &server, true, false); errServer != nil {
					remoteSrc.logger.Debug("Failed to get server by name", zap.Error(errServer))

					continue
				}

				serverCache[serverName] = server
			}

			event, errLogServerEvent := logToServerEvent(parser, server.ServerID, server.ServerName, logPayload.body)
			if errLogServerEvent != nil {
				remoteSrc.logger.Debug("Failed to create serverEvent", zap.Error(errLogServerEvent))

				continue
			}

			remoteSrc.eb.Emit(event)
		}
	}
}

func logToServerEvent(parser *logparse.LogParser, serverID int, serverName string, msg string) (serverEvent, error) {
	event := serverEvent{
		ServerID:   serverID,
		ServerName: serverName,
	}
	// var resultToSource = func(sid string, results logparse.Results, nameKey string, player *model.Person) error {
	//	if sid == "BOT" {
	//		panic("fixme")
	//		//player.SteamID = logparse.BotSid
	//		//name, ok := results.Values[nameKey]
	//		//if !ok {
	//		//	return errors.New("Failed to parse bot name")
	//		//}
	//		//player.PersonaName = name.(string)
	//		return nil
	//	} else {
	//		return db.GetOrCreatePersonBySteamID(ctx, steamid.SID3ToSID64(steamid.SID3(sid)), player)
	//	}
	// }
	parseResult, errParse := parser.Parse(msg)
	if errParse != nil {
		return event, errors.Wrapf(errParse, "Failed to parse log message")
	}

	event.Results = parseResult

	return event, nil
}
