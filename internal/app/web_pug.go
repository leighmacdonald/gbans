package app

import (
	"github.com/leighmacdonald/gbans/pkg/fp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
	"sync"
)

type pugLobby struct {
	*sync.RWMutex
	Leader   *wsClient
	LobbyId  string          `json:"lobby_id"`
	Clients  wsClients       `json:"clients"`
	Messages []wsUserMessage `json:"messages"`
}

func newPugLobby(creator *wsClient, id string) *pugLobby {
	return &pugLobby{
		Leader:   creator,
		RWMutex:  &sync.RWMutex{},
		LobbyId:  id,
		Clients:  wsClients{creator},
		Messages: []wsUserMessage{},
	}
}

func (lobby *pugLobby) clientCount() int {
	lobby.RLock()
	defer lobby.RUnlock()
	return len(lobby.Clients)
}

func (lobby *pugLobby) id() string {
	lobby.RLock()
	defer lobby.RUnlock()
	return lobby.LobbyId
}

func (lobby *pugLobby) join(client *wsClient) error {
	lobby.Lock()
	defer lobby.Unlock()
	if slices.Contains(lobby.Clients, client) {
		return ErrDuplicateClient
	}
	lobby.Clients = append(lobby.Clients, client)
	client.lobbies = append(client.lobbies, lobby)
	log.WithFields(log.Fields{
		"clients": len(lobby.Clients),
		"leader":  len(lobby.Clients) == 1,
		"lobby":   lobby.LobbyId,
	}).Infof("User joined lobby")
	if len(lobby.Clients) == 1 {
		return lobby.promote(client)
	}
	return nil
}

func (lobby *pugLobby) promote(client *wsClient) error {
	lobby.Leader = client
	return nil
}

func (lobby *pugLobby) leave(client *wsClient) error {
	lobby.Lock()
	defer lobby.Unlock()
	if !slices.Contains(lobby.Clients, client) {
		return ErrUnknownClient
	}
	if len(lobby.Clients) == 1 {
		return ErrEmptyLobby
	}
	lobby.Clients = fp.Remove(lobby.Clients, client)
	client.removeLobby(lobby)
	//if client.Leader {
	//	client.Leader = false
	//	return lobby.promote(lobby.Clients[0])
	//}
	return nil
}

func (lobby *pugLobby) broadcast(response wsBaseResponse) error {
	for _, client := range lobby.Clients {
		client.send <- response
	}
	return nil
}

func (lobby *pugLobby) sendUserMessage(msg wsUserMessage) {
	lobby.Lock()
	defer lobby.Unlock()
	lobby.Messages = append(lobby.Messages, msg)
}
