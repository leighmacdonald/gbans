package app

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/web"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RemoteSrcdsLogSource struct {
	*sync.RWMutex
	udpAddr   *net.UDPAddr
	sink      chan web.LogPayload
	db        store.Store
	secretMap map[int64]string
	dnsMap    map[string]string
}

func NewRemoteSrcdsLogSource(listenAddr string, db store.Store, sink chan web.LogPayload) (*RemoteSrcdsLogSource, error) {
	udpAddr, err := net.ResolveUDPAddr("udp4", listenAddr)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to resolve UDP address")
	}
	return &RemoteSrcdsLogSource{
		RWMutex:   &sync.RWMutex{},
		udpAddr:   udpAddr,
		db:        db,
		sink:      sink,
		secretMap: map[int64]string{},
		dnsMap:    map[string]string{},
	}, nil
}

// Updates DNS -> IP mappings
func (srv *RemoteSrcdsLogSource) updateDNS() {
	newServers := map[string]string{}
	servers, errServers := srv.db.GetServers(context.Background(), true)
	if errServers != nil {
		log.Errorf("Failed to load servers to update DNS: %v", errServers)
		return
	}
	for _, server := range servers {
		ipAddr, errLookup := net.LookupIP(server.Address)
		if errLookup != nil || len(ipAddr) == 0 {
			log.Errorf("Failed to lookup dns for host: %v", errLookup)
			continue
		}
		newServers[fmt.Sprintf("%s:%d", ipAddr[0], server.Port)] = server.ServerName
	}
	srv.Lock()
	defer srv.Unlock()
	srv.dnsMap = newServers
	log.Debugf("Updated DNS mappings")
}

func (srv *RemoteSrcdsLogSource) updateSecrets() {
	newServers := map[int64]string{}
	servers, errServers := srv.db.GetServers(context.Background(), true)
	if errServers != nil {
		log.Errorf("Failed to load servers to update DNS: %v", errServers)
		return
	}
	for _, server := range servers {
		newId := rand.Int63()
		ipAddr, errLookup := net.LookupIP(server.Address)
		if errLookup != nil || len(ipAddr) == 0 {
			log.Errorf("Failed to lookup dns for host: %v", errLookup)
			continue
		}
		newServers[newId] = server.ServerName
		go func(s model.Server, i int64) {
			for _, cmd := range []string{
				fmt.Sprintf("sv_logsecret %d", i),
				fmt.Sprintf("logaddress_add %s", config.Log.SrcdsLogExternalHost),
			} {
				_, errRcon := query.ExecRCON(s, cmd)
				if errRcon != nil {
					log.Errorf("Failed to run srcds log command: %s [%s]", cmd, errRcon)
					break
				}
			}

		}(server, newId)
	}
	srv.Lock()
	defer srv.Unlock()
	srv.secretMap = newServers
	log.Debugf("Updated secret mappings")
}

// Start initiates the udp network log read loop. DNS names are used to
// map the server logs to the internal known server id. The DNS is updated
// every 60 minutes so that it remains up to date.
func (srv *RemoteSrcdsLogSource) Start() {
	type newMsg struct {
		secure    bool
		source    int64
		sourceDNS string
		body      string
	}
	connection, err := net.ListenUDP("udp4", srv.udpAddr)
	if err != nil {
		log.Errorf("Failed to start log listener: %v", err)
		return
	}
	defer func() {
		if errConnClose := connection.Close(); errConnClose != nil {
			log.Errorf("Failed to close connection cleanly: %v", errConnClose)
		}
	}()
	msgId := 0
	inChan := make(chan newMsg)
	srv.updateSecrets()
	srv.updateDNS()
	go func() {
		for {
			buffer := make([]byte, 1024)
			n, src, readErr := connection.ReadFromUDP(buffer)
			if readErr != nil {
				log.Warnf("UDP log read error: %v", readErr)
				continue
			}
			switch buffer[4] {
			case 0x52:
				inChan <- newMsg{
					secure:    false,
					sourceDNS: fmt.Sprintf("%s:%d", src.IP, src.Port),
					body:      string(buffer[5 : n-2]),
				}
			case 0x53:
				line := string(buffer)
				idx := strings.Index(line, "L ")
				if idx == -1 {
					continue
				}
				secret, errConv := strconv.ParseInt(line[5:idx], 10, 64)
				if errConv != nil {
					continue
				}
				inChan <- newMsg{
					secure: true,
					source: secret,
					body:   line[idx : n-2],
				}
			}
		}
	}()
	ticker := time.NewTicker(time.Minute * 60)
	for {
		select {
		case <-ticker.C:
			srv.updateSecrets()
			srv.updateDNS()
		case logPayload := <-inChan:
			payload := web.LogPayload{Message: logPayload.body}
			if logPayload.secure {
				srv.RLock()
				serverName, found := srv.secretMap[logPayload.source]
				srv.RUnlock()
				if !found {
					log.Warnf("Rejecting unknown secret log source: %s [%s]", logPayload.sourceDNS, logPayload.body)
					continue
				}
				payload.ServerName = serverName
			} else {
				srv.RLock()
				serverName, found := srv.dnsMap[logPayload.sourceDNS]
				srv.RUnlock()
				if !found {
					log.Warnf("Rejecting unknown dns log source: %d [%s]", logPayload.source, logPayload.body)
					continue
				}
				payload.ServerName = serverName
			}
			select {
			case srv.sink <- payload:
			default:
				log.WithFields(log.Fields{"size": len(srv.sink)}).Warnf("Log sink full")
			}
			log.WithFields(log.Fields{"id": msgId, "server": payload.ServerName, "sec": logPayload.secure, "body": logPayload.body}).
				Tracef("Srcds remote log")
			msgId++
		}
	}
}
