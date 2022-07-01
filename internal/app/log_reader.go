package app

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type srcdsPacket byte

const (
	// Normal log messages (deprecated)
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
	ctx     context.Context
	udpAddr *net.UDPAddr

	database  store.Store
	secretMap map[int]string
	dnsMap    map[string]string
	frequency time.Duration
}

func newRemoteSrcdsLogSource(ctx context.Context, listenAddr string, database store.Store) (*remoteSrcdsLogSource, error) {
	udpAddr, errResolveUDP := net.ResolveUDPAddr("udp4", listenAddr)
	if errResolveUDP != nil {
		return nil, errors.Wrapf(errResolveUDP, "Failed to resolve UDP address")
	}
	return &remoteSrcdsLogSource{
		RWMutex:   &sync.RWMutex{},
		ctx:       ctx,
		udpAddr:   udpAddr,
		database:  database,
		secretMap: map[int]string{},
		dnsMap:    map[string]string{},
		frequency: time.Minute * 5,
	}, nil
}

// Updates DNS -> IP mappings
//func (remoteSrc *remoteSrcdsLogSource) updateDNS() {
//	newServers := map[string]string{}
//	serversCtx, cancelServers := context.WithTimeout(remoteSrc.ctx, time.Second*5)
//	defer cancelServers()
//	servers, errServers := remoteSrc.database.GetServers(serversCtx, true)
//	if errServers != nil {
//		log.Errorf("Failed to load servers to update DNS: %v", errServers)
//		return
//	}
//	for _, server := range servers {
//		ipAddr, errLookup := net.LookupIP(server.Address)
//		if errLookup != nil || len(ipAddr) == 0 {
//			log.Errorf("Failed to lookup dns for host: %v", errLookup)
//			continue
//		}
//		newServers[fmt.Sprintf("%s:%d", ipAddr[0], server.Port)] = server.ServerNameShort
//	}
//	remoteSrc.Lock()
//	defer remoteSrc.Unlock()
//	remoteSrc.dnsMap = newServers
//	log.Debugf("Updated DNS mappings")
//}

func (remoteSrc *remoteSrcdsLogSource) updateSecrets() {
	newServers := map[int]string{}
	serversCtx, cancelServers := context.WithTimeout(remoteSrc.ctx, time.Second*5)
	defer cancelServers()
	servers, errServers := remoteSrc.database.GetServers(serversCtx, false)
	if errServers != nil {
		log.Errorf("Failed to load servers to update DNS: %v", errServers)
		return
	}
	for _, server := range servers {
		newServers[server.LogSecret] = server.ServerNameShort
	}
	remoteSrc.Lock()
	defer remoteSrc.Unlock()
	remoteSrc.secretMap = newServers
	log.Debugf("Updated secret mappings")
}

func (remoteSrc *remoteSrcdsLogSource) addLogAddress(addr string) {
	serversCtx, cancelServers := context.WithTimeout(remoteSrc.ctx, time.Second*10)
	defer cancelServers()
	servers, errServers := remoteSrc.database.GetServers(serversCtx, false)
	if errServers != nil {
		log.Errorf("Failed to load servers to add log addr: %v", errServers)
		return
	}
	queryCtx, cancelQuery := context.WithTimeout(remoteSrc.ctx, time.Second*20)
	defer cancelQuery()
	query.RCON(queryCtx, servers, fmt.Sprintf("logaddress_add %s", addr))
	log.WithField("addr", addr).Infof("Added log address")
}

func (remoteSrc *remoteSrcdsLogSource) removeLogAddress(addr string) {
	serversCtx, cancelServers := context.WithTimeout(remoteSrc.ctx, time.Second*10)
	defer cancelServers()
	servers, errServers := remoteSrc.database.GetServers(serversCtx, false)
	if errServers != nil {
		log.Errorf("Failed to load servers to del log addr: %v", errServers)
		return
	}
	queryCtx, cancelQuery := context.WithTimeout(remoteSrc.ctx, time.Second*20)
	defer cancelQuery()
	query.RCON(queryCtx, servers, fmt.Sprintf("logaddress_del %s", addr))
	log.Debugf("Removed log address")
}

// start initiates the udp network log read loop. DNS names are used to
// map the server logs to the internal known server id. The DNS is updated
// every 60 minutes so that it remains up to date.
func (remoteSrc *remoteSrcdsLogSource) start() {
	type newMsg struct {
		secure    bool
		source    int64
		sourceDNS string
		body      string
	}
	connection, errListenUDP := net.ListenUDP("udp4", remoteSrc.udpAddr)
	if errListenUDP != nil {
		log.Errorf("Failed to start log listener: %v", errListenUDP)
		return
	}
	defer func() {
		if errConnClose := connection.Close(); errConnClose != nil {
			log.Errorf("Failed to close connection cleanly: %v", errConnClose)
		}
	}()
	//msgId := 0
	msgIngressChan := make(chan newMsg)
	remoteSrc.updateSecrets()
	if config.Debug.AddRCONLogAddress != "" {
		remoteSrc.addLogAddress(config.Debug.AddRCONLogAddress)
		defer remoteSrc.removeLogAddress(config.Debug.AddRCONLogAddress)
	}
	//remoteSrc.updateDNS()
	go func() {
		for {
			buffer := make([]byte, 1024)
			readLen, sourceAddress, errReadUDP := connection.ReadFromUDP(buffer)
			if errReadUDP != nil {
				log.Warnf("UDP log read error: %v", errReadUDP)
				continue
			}
			switch srcdsPacket(buffer[4]) {
			case s2aLogString:
				msgIngressChan <- newMsg{
					secure:    false,
					sourceDNS: fmt.Sprintf("%s:%d", sourceAddress.IP, sourceAddress.Port),
					body:      string(buffer[5 : readLen-2])}
			case s2aLogString2:
				line := string(buffer)
				idx := strings.Index(line, "L ")
				if idx == -1 {
					log.Warnf("Received malformed log message: Failed to find marker")
					continue
				}
				secret, errConv := strconv.ParseInt(line[5:idx], 10, 32)
				if errConv != nil {
					log.Warnf("Received malformed log message: Failed to parse secret: %v", errConv)
					continue
				}
				msgIngressChan <- newMsg{secure: true, source: secret, body: line[idx : readLen-2]}
			}
		}
	}()
	ticker := time.NewTicker(remoteSrc.frequency)
	//errCount := 0
	for {
		select {
		case <-ticker.C:
			remoteSrc.updateSecrets()
			//remoteSrc.updateDNS()
		case _ = <-msgIngressChan:
			//payload := model.LogPayload{Message: logPayload.body}
			//if logPayload.secure {
			//	remoteSrc.RLock()
			//	serverName, found := remoteSrc.secretMap[int(logPayload.source)]
			//	remoteSrc.RUnlock()
			//	if !found {
			//		log.Tracef("Rejecting unknown secret log author: %s [%s]", logPayload.sourceDNS, logPayload.body)
			//		continue
			//	}
			//	payload.ServerName = serverName
			//} else {
			//	remoteSrc.RLock()
			//	serverName, found := remoteSrc.dnsMap[logPayload.sourceDNS]
			//	remoteSrc.RUnlock()
			//	if !found {
			//		log.Tracef("Rejecting unknown dns log author: %d [%s]", logPayload.source, logPayload.body)
			//		continue
			//	}
			//	payload.ServerName = serverName
			//}
			//select {
			//case remoteSrc.sink <- payload:
			//default:
			//	errCount++
			//	if errCount%1000 == 0 {
			//		log.WithFields(log.Fields{"size": len(remoteSrc.sink)}).Warnf("Log sink full")
			//	}
			//}
			//log.WithFields(log.Fields{"id": msgId, "server": payload.ServerName, "sec": logPayload.secure, "body": logPayload.body}).
			//	Tracef("Srcds remote log")
			//msgId++
		}
	}
}

func logToServerEvent(ctx context.Context, server model.Server, msg string, db store.Store, playerStateCache *playerCache,
	event *model.ServerEvent) error {
	parseResult := logparse.Parse(msg)
	event.Server = server
	event.EventType = parseResult.MsgType
	var playerSource model.Person
	sid1, sid1Found := parseResult.Values["sid"]
	if sid1Found {
		errGetSourcePlayer := db.GetOrCreatePersonBySteamID(ctx, steamid.SID3ToSID64(steamid.SID3(sid1.(string))), &playerSource)
		if errGetSourcePlayer != nil {
			return errors.Wrapf(errGetSourcePlayer, "Failed to get source player [%s]", msg)
		} else {
			event.Source = playerSource
		}
	}

	var playerTarget model.Person
	sid2, sid2Found := parseResult.Values["sid2"]
	if sid2Found {
		if errGetTargetPlayer := db.GetOrCreatePersonBySteamID(ctx, steamid.SID3ToSID64(steamid.SID3(sid2.(string))), &playerTarget); errGetTargetPlayer != nil {
		} else {
			event.Target = playerTarget
		}
	}
	aposValue, aposFound := parseResult.Values["attacker_position"]
	if aposFound {
		var attackerPosition logparse.Pos
		if errParsePOS := logparse.ParsePOS(aposValue.(string), &attackerPosition); errParsePOS != nil {
			log.Warnf("Failed to parse attacker position: %p", errParsePOS)
		}
		event.AttackerPOS = attackerPosition
		delete(parseResult.Values, "attacker_position")
	}
	vposValue, vposFound := parseResult.Values["victim_position"]
	if vposFound {
		var victimPosition logparse.Pos
		if errParsePOS := logparse.ParsePOS(vposValue.(string), &victimPosition); errParsePOS != nil {
			log.Warnf("Failed to parse victim position: %parseResult", errParsePOS)
		}
		event.VictimPOS = victimPosition
		delete(parseResult.Values, "victim_position")
	}
	asValue, asFound := parseResult.Values["assister_position"]
	if asFound {
		var assisterPosition logparse.Pos
		if errParsePOS := logparse.ParsePOS(asValue.(string), &assisterPosition); errParsePOS != nil {
			log.Warnf("Failed to parse assister position: %parseResult", errParsePOS)
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
			log.Warnf("failed to parse damage value: %parseResult", errParseDamage)
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
			log.Warnf("failed to parse damage value: %parseResult", errParseRealDamage)
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
			log.Warnf("failed to parse healing value: %parseResult", errParseHealing)
		}
		event.Healing = healingP
	}

	createdOnValue, createdOnFound := parseResult.Values["created_on"]
	if !createdOnFound {
		// log.Warnf("created_on missing")
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
