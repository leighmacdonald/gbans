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
	port      int
	sink      chan web.LogPayload
	db        store.Store
	serverMap map[string]string
}

func NewRemoteSrcdsLogSource(listenPort int, db store.Store, sink chan web.LogPayload) (*RemoteSrcdsLogSource, error) {
	udpAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to resolve UDP address")
	}
	return &RemoteSrcdsLogSource{
		RWMutex:   &sync.RWMutex{},
		udpAddr:   udpAddr,
		port:      listenPort,
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

func (srv *RemoteSrcdsLogSource) Start() {
	type newMsg struct {
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

	inChan := make(chan newMsg)
	srv.updateDNS()
	go func() {
		buffer := make([]byte, 4096)
		for {
			n, src, readErr := connection.ReadFromUDP(buffer)
			if readErr != nil {
				log.Warnf("UDP log read error: %v", readErr)
				continue
			}
			inChan <- newMsg{
				source: fmt.Sprintf("%s:%d", src.IP, src.Port),
				body:   string(buffer[4 : n-1]),
			}
		}
	}()
	t := time.NewTicker(time.Minute * 60)
	for {
		select {
		case <-t.C:
			srv.updateDNS()
		case logPayload := <-inChan:
			serverName, found := srv.serverMap[logPayload.source]
			if !found {
				log.Warnf("Rejecting unknown log source: %s [%s]", logPayload.source, logPayload.body)

			}
			switch logPayload.body[0] {
			case 'R':
				srv.sink <- web.LogPayload{
					ServerName: serverName,
					Message:    logPayload.body[1:], // strip "R" log prefix added to remote logs
				}
			case 'S':
				log.Debugf("[SRCLOG] (sec) %s", logPayload.body)
			default:
				log.Debugf("[SRCLOG] (unhandled type) %s", logPayload.body)
			}
		}
	}
}
