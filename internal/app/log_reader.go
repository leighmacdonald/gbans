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
	"github.com/leighmacdonald/steamid/v2/steamid"
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
	ctx       context.Context
	logger    *zap.Logger
	udpAddr   *net.UDPAddr
	database  store.Store
	secretMap map[int]string
	frequency time.Duration
}

func newRemoteSrcdsLogSource(ctx context.Context, logger *zap.Logger, listenAddr string, database store.Store) (*remoteSrcdsLogSource, error) {
	udpAddr, errResolveUDP := net.ResolveUDPAddr("udp4", listenAddr)
	if errResolveUDP != nil {
		return nil, errors.Wrapf(errResolveUDP, "Failed to resolve UDP address")
	}
	return &remoteSrcdsLogSource{
		RWMutex:   &sync.RWMutex{},
		ctx:       ctx,
		logger:    logger,
		udpAddr:   udpAddr,
		database:  database,
		secretMap: map[int]string{},
		frequency: time.Minute * 5,
	}, nil
}

func (remoteSrc *remoteSrcdsLogSource) updateSecrets() {
	newServers := map[int]string{}
	serversCtx, cancelServers := context.WithTimeout(remoteSrc.ctx, time.Second*5)
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

func (remoteSrc *remoteSrcdsLogSource) addLogAddress(addr string) {
	serversCtx, cancelServers := context.WithTimeout(remoteSrc.ctx, time.Second*10)
	defer cancelServers()
	servers, errServers := remoteSrc.database.GetServers(serversCtx, false)
	if errServers != nil {
		remoteSrc.logger.Error("Failed to load servers to add log addr", zap.Error(errServers))
		return
	}
	queryCtx, cancelQuery := context.WithTimeout(remoteSrc.ctx, time.Second*20)
	defer cancelQuery()
	query.RCON(queryCtx, remoteSrc.logger, servers, fmt.Sprintf("logaddress_add %s", addr))
	remoteSrc.logger.Info("Added udp log address", zap.String("addr", addr))
}

func (remoteSrc *remoteSrcdsLogSource) removeLogAddress(addr string) {
	serversCtx, cancelServers := context.WithTimeout(remoteSrc.ctx, time.Second*10)
	defer cancelServers()
	servers, errServers := remoteSrc.database.GetServers(serversCtx, false)
	if errServers != nil {
		remoteSrc.logger.Error("Failed to load servers to del log addr", zap.Error(errServers))
		return
	}
	queryCtx, cancelQuery := context.WithTimeout(remoteSrc.ctx, time.Second*20)
	defer cancelQuery()
	query.RCON(queryCtx, remoteSrc.logger, servers, fmt.Sprintf("logaddress_del %s", addr))
	remoteSrc.logger.Debug("Removed log address", zap.String("addr", addr))
}

// start initiates the udp network log read loop. DNS names are used to
// map the server logs to the internal known server id. The DNS is updated
// every 60 minutes so that it remains up to date.
func (remoteSrc *remoteSrcdsLogSource) start(database store.Store) {
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
	remoteSrc.updateSecrets()
	if config.Debug.AddRCONLogAddress != "" {
		remoteSrc.addLogAddress(config.Debug.AddRCONLogAddress)
		defer remoteSrc.removeLogAddress(config.Debug.AddRCONLogAddress)
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
	pc := newPlayerCache(remoteSrc.logger)
	ticker := time.NewTicker(remoteSrc.frequency)
	//errCount := 0
	for {
		select {
		case <-remoteSrc.ctx.Done():
			running = false
		case <-ticker.C:
			remoteSrc.updateSecrets()
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
			if errServer := database.GetServerByName(remoteSrc.ctx, serverName, &server); errServer != nil {
				remoteSrc.logger.Debug("Failed to get server by name", zap.Error(errServer))
				continue
			}
			var serverEvent model.ServerEvent
			if errLogServerEvent := logToServerEvent(remoteSrc.ctx, remoteSrc.logger, server, logPayload.body,
				database, pc, &serverEvent); errLogServerEvent != nil {
				remoteSrc.logger.Debug("Failed to create ServerEvent", zap.Error(errLogServerEvent))
				continue
			}

			event.Emit(serverEvent)
		}
	}
}

func logToServerEvent(ctx context.Context, logger *zap.Logger, server model.Server, msg string, db store.Store,
	playerStateCache *playerCache, event *model.ServerEvent) error {
	var resultToSource = func(sid string, results logparse.Results, nameKey string, player *model.Person) error {
		if sid == "BOT" {
			player.SteamID = logparse.BotSid
			name, ok := results.Values[nameKey]
			if !ok {
				return errors.New("Failed to parse bot name")
			}
			player.PersonaName = name.(string)
			return nil
		} else {
			return db.GetOrCreatePersonBySteamID(ctx, steamid.SID3ToSID64(steamid.SID3(sid)), player)
		}
	}

	parseResult := logparse.Parse(msg)
	event.Server = server
	event.EventType = parseResult.MsgType

	playerSource := model.NewPerson(0)
	sid1, sid1Found := parseResult.Values["sid"]
	if sid1Found {
		if sourceErr := resultToSource(sid1.(string), parseResult, "name", &playerSource); sourceErr != nil {
			return sourceErr
		}
		event.Source = playerSource
	}
	playerTarget := model.NewPerson(0)
	sid2, sid2Found := parseResult.Values["sid2"]
	if sid2Found {
		if sourceErr := resultToSource(sid2.(string), parseResult, "name2", &playerTarget); sourceErr != nil {
			return sourceErr
		}
		event.Target = playerTarget
	}
	aposValue, aposFound := parseResult.Values["attacker_position"]
	if aposFound {
		var attackerPosition logparse.Pos
		if errParsePOS := logparse.ParsePOS(aposValue.(string), &attackerPosition); errParsePOS != nil {
			logger.Warn("Failed to parse attacker position", zap.Error(errParsePOS))
		}
		event.AttackerPOS = attackerPosition
		delete(parseResult.Values, "attacker_position")
	}
	vposValue, vposFound := parseResult.Values["victim_position"]
	if vposFound {
		var victimPosition logparse.Pos
		if errParsePOS := logparse.ParsePOS(vposValue.(string), &victimPosition); errParsePOS != nil {
			logger.Warn("Failed to parse victim position", zap.Error(errParsePOS))
		}
		event.VictimPOS = victimPosition
		delete(parseResult.Values, "victim_position")
	}
	asValue, asFound := parseResult.Values["assister_position"]
	if asFound {
		var assisterPosition logparse.Pos
		if errParsePOS := logparse.ParsePOS(asValue.(string), &assisterPosition); errParsePOS != nil {
			logger.Warn("Failed to parse assister position", zap.Error(errParsePOS))
		}
		event.AssisterPOS = assisterPosition
		delete(parseResult.Values, "assister_position")
	}

	critType, critTypeFound := parseResult.Values["crit"]
	if critTypeFound {
		event.Crit = critType.(logparse.CritType)
		delete(parseResult.Values, "crit")
	}

	weapon := logparse.UnknownWeapon
	weaponValue, weaponFound := parseResult.Values["weapon"]
	if weaponFound {
		weapon = logparse.ParseWeapon(weaponValue.(string))
	}
	event.Weapon = weapon

	var class logparse.PlayerClass
	classValue, classFound := parseResult.Values["class"]
	if classFound {
		if !logparse.ParsePlayerClass(classValue.(string), &class) {
			class = logparse.Spectator
		}
		delete(parseResult.Values, "class")
	} else if event.Source.SteamID != 0 {
		class = playerStateCache.getClass(event.Source.SteamID)
	}
	event.PlayerClass = class

	var damage int64
	dmgValue, dmgFound := parseResult.Values["damage"]
	if dmgFound {
		parsedDamage, errParseDamage := strconv.ParseInt(dmgValue.(string), 10, 32)
		if errParseDamage != nil {
			logger.Warn("failed to parse damage value", zap.Error(errParseDamage))
		}
		damage = parsedDamage
		delete(parseResult.Values, "damage")
	}
	event.Damage = damage

	var realDamage int64
	realDmgValue, realDmgFound := parseResult.Values["realdamage"]
	if realDmgFound {
		parsedRealDamage, errParseRealDamage := strconv.ParseInt(realDmgValue.(string), 10, 32)
		if errParseRealDamage != nil {
			logger.Warn("failed to parse damage value", zap.Error(errParseRealDamage))
		}
		realDamage = parsedRealDamage
		delete(parseResult.Values, "realdamage")
	}
	event.RealDamage = realDamage

	var item logparse.PickupItem
	itemValue, itemFound := parseResult.Values["item"]
	if itemFound {
		if !logparse.ParsePickupItem(itemValue.(string), &item) {
			item = 0
		}
	}
	event.Item = item
	var team logparse.Team
	teamValue, teamFound := parseResult.Values["team"]
	if teamFound {
		if !logparse.ParseTeam(teamValue.(string), &team) {
			team = 0
		}
	} else {
		if event.Source.SteamID.Valid() {
			team = playerStateCache.getTeam(event.Source.SteamID)
		}
	}
	event.Team = team

	healingValue, healingFound := parseResult.Values["healing"]
	if healingFound {
		healingP, errParseHealing := strconv.ParseInt(healingValue.(string), 10, 32)
		if errParseHealing != nil {
			logger.Warn("failed to parse healing value", zap.Error(errParseHealing))
		}
		event.Healing = healingP
	}

	createdOnValue, createdOnFound := parseResult.Values["created_on"]
	if !createdOnFound {
		event.CreatedOn = config.Now()
	} else {
		event.CreatedOn = createdOnValue.(time.Time)
	}
	// Remove keys that get mapped to actual schema columns
	for _, key := range []string{
		"created_on", "item", "weapon", "healing",
		"name", "pid", "sid", "team",
		"name2", "pid2", "sid2", "team2"} {
		delete(parseResult.Values, key)
	}
	event.MetaData = parseResult.Values
	switch parseResult.MsgType {
	case logparse.SpawnedAs:
		playerStateCache.setClass(event.Source.SteamID, event.PlayerClass)
	case logparse.JoinedTeam:
		playerStateCache.setTeam(event.Source.SteamID, event.Team)
	}
	return nil
}
