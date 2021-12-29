package app

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/web"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

type RemoteSrcdsLogSource struct {
	*sync.RWMutex
	udpAddr   *net.UDPAddr
	sink      chan web.LogPayload
	db        store.Store
	serverMap map[string]string
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
		serverMap: map[string]string{},
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
	srv.serverMap = newServers
	log.Debugf("Updated DNS mappings")
}

// Start initiates the udp network log read loop. DNS names are used to
// map the server logs to the internal known server id. The DNS is updated
// every 60 minutes so that it remains up to date.
func (srv *RemoteSrcdsLogSource) Start() {
	type newMsg struct {
		secure bool
		source string
		body   string
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
	srv.updateDNS()
	go func() {
		for {
			buffer := make([]byte, 1024)
			n, src, readErr := connection.ReadFromUDP(buffer)
			if readErr != nil {
				log.Warnf("UDP log read error: %v", readErr)
				continue
			}
			inChan <- newMsg{
				secure: buffer[4] == 'S',
				source: fmt.Sprintf("%s:%d", src.IP, src.Port),
				body:   string(buffer[5 : n-1]),
			}
		}
	}()
	ticker := time.NewTicker(time.Minute * 60)
	for {
		select {
		case <-ticker.C:
			srv.updateDNS()
		case logPayload := <-inChan:
			serverName, found := srv.serverMap[logPayload.source]
			if !found {
				log.Warnf("Rejecting unknown log source: %s [%s]", logPayload.source, logPayload.body)
				continue
			}
			if !logPayload.secure {
				select {
				case srv.sink <- web.LogPayload{ServerName: serverName, Message: logPayload.body}:
				default:
					log.WithFields(log.Fields{"size": len(srv.sink)}).Warnf("Log sink full")
				}
			} else {

			}
			log.WithFields(log.Fields{"id": msgId, "server": serverName, "sec": logPayload.secure, "body": logPayload.body}).
				Tracef("Srcds remote log")
			msgId++
		}
	}
}
