package app

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/event"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type srcdsPacket byte

const (
	// Normal log messages (unsupported)
	s2aLogString srcdsPacket = 0x52
	// Sent when using sv_logsecret
	s2aLogString2 srcdsPacket = 0x53
)

// remoteSrcdsLogSource handles reading inbound srcds log packets, and emitting a web.LogPayload
// that can be further parsed/processed.
//
// On, start and every hour after, a new sv_logsecret value for every instance is randomly generated and
// assigned remotely over rcon. This allows us to associate certain semi secret id's with specific server
// instances
type remoteSrcdsLogSource struct {
	*sync.RWMutex
	logger    *zap.Logger
	udpAddr   *net.UDPAddr
	database  store.Store
	secretMap map[int]string
	frequency time.Duration
}

func newRemoteSrcdsLogSource(logger *zap.Logger, listenAddr string, database store.Store) (*remoteSrcdsLogSource, error) {
	udpAddr, errResolveUDP := net.ResolveUDPAddr("udp4", listenAddr)
	if errResolveUDP != nil {
		return nil, errors.Wrapf(errResolveUDP, "Failed to resolve UDP address")
	}
	return &remoteSrcdsLogSource{
		RWMutex:   &sync.RWMutex{},
		logger:    logger,
		udpAddr:   udpAddr,
		database:  database,
		secretMap: map[int]string{},
		frequency: time.Minute * 5,
	}, nil
}

func (remoteSrc *remoteSrcdsLogSource) updateSecrets(ctx context.Context) {
	newServers := map[int]string{}
	serversCtx, cancelServers := context.WithTimeout(ctx, time.Second*5)
	defer cancelServers()
	servers, errServers := remoteSrc.database.GetServers(serversCtx, false)
	if errServers != nil {
		remoteSrc.logger.Error("Failed to load servers to update DNS", zap.Error(errServers))
		return
	}
	for _, server := range servers {
		newServers[server.LogSecret] = server.ServerNameShort
	}
	remoteSrc.Lock()
	defer remoteSrc.Unlock()
	remoteSrc.secretMap = newServers
	remoteSrc.logger.Debug("Updated secret mappings")
}

func (remoteSrc *remoteSrcdsLogSource) addLogAddress(ctx context.Context, addr string) {
	serversCtx, cancelServers := context.WithTimeout(ctx, time.Second*10)
	defer cancelServers()
	servers, errServers := remoteSrc.database.GetServers(serversCtx, false)
	if errServers != nil {
		remoteSrc.logger.Error("Failed to load servers to add log addr", zap.Error(errServers))
		return
	}
	queryCtx, cancelQuery := context.WithTimeout(ctx, time.Second*20)
	defer cancelQuery()
	query.RCON(queryCtx, remoteSrc.logger, servers, fmt.Sprintf("logaddress_add %s", addr))
	remoteSrc.logger.Info("Added udp log address", zap.String("addr", addr))
}

func (remoteSrc *remoteSrcdsLogSource) removeLogAddress(ctx context.Context, addr string) {
	serversCtx, cancelServers := context.WithTimeout(ctx, time.Second*10)
	defer cancelServers()
	servers, errServers := remoteSrc.database.GetServers(serversCtx, false)
	if errServers != nil {
		remoteSrc.logger.Error("Failed to load servers to del log addr", zap.Error(errServers))
		return
	}
	queryCtx, cancelQuery := context.WithTimeout(ctx, time.Second*20)
	defer cancelQuery()
	query.RCON(queryCtx, remoteSrc.logger, servers, fmt.Sprintf("logaddress_del %s", addr))
	remoteSrc.logger.Debug("Removed log address", zap.String("addr", addr))
}

// start initiates the udp network log read loop. DNS names are used to
// map the server logs to the internal known server id. The DNS is updated
// every 60 minutes so that it remains up to date.
func (remoteSrc *remoteSrcdsLogSource) start(ctx context.Context, database store.Store) {
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
	//msgId := 0
	msgIngressChan := make(chan newMsg)
	remoteSrc.updateSecrets(ctx)
	if config.Debug.AddRCONLogAddress != "" {
		remoteSrc.addLogAddress(ctx, config.Debug.AddRCONLogAddress)
		defer remoteSrc.removeLogAddress(ctx, config.Debug.AddRCONLogAddress)
	}
	running := true
	count := uint64(0)
	insecureCount := uint64(0)
	errCount := uint64(0)
	go func() {
		for running {
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
	//pc := newPlayerCache(remoteSrc.logger)
	ticker := time.NewTicker(remoteSrc.frequency)
	//errCount := 0
	for {
		select {
		case <-ctx.Done():
			running = false
		case <-ticker.C:
			remoteSrc.updateSecrets(ctx)
			//remoteSrc.updateDNS()
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
			var server model.Server
			if errServer := database.GetServerByName(ctx, serverName, &server); errServer != nil {
				remoteSrc.logger.Debug("Failed to get server by name", zap.Error(errServer))
				continue
			}
			var serverEvent model.ServerEvent
			if errLogServerEvent := logToServerEvent(server, logPayload.body, &serverEvent); errLogServerEvent != nil {
				remoteSrc.logger.Debug("Failed to create ServerEvent", zap.Error(errLogServerEvent))
				continue
			}

			event.Emit(serverEvent)
		}
	}
}

func logToServerEvent(server model.Server, msg string, event *model.ServerEvent) error {
	//var resultToSource = func(sid string, results logparse.Results, nameKey string, player *model.Person) error {
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
	//}
	parseResult, errParse := logparse.Parse(msg)
	if errParse != nil {
		return errParse
	}
	event.Server = server
	event.Results = parseResult
	return nil
}
