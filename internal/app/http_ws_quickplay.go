package app

//
// import (
//	"github.com/leighmacdonald/gbans/pkg/fp"
//	"golang.org/x/exp/slices"
//	"sync"
// )
//
// type qpLobby struct {
//	*sync.RWMutex
//	LobbyID  string          `json:"lobby_id"`
//	Clients  wsClients       `json:"clients"`
//	Messages []wsUserMessage `json:"messages"`
//	Leader   *wsClient       `json:"leader"`
// }
//
// func newQPLobby(lobbyId string, creator *wsClient) *qpLobby {
//	return &qpLobby{
//		Leader:   creator,
//		RWMutex:  &sync.RWMutex{},
//		LobbyID:  lobbyId,
//		Clients:  wsClients{creator},
//		Messages: []wsUserMessage{},
//	}
// }
//
// func (lobby *qpLobby) clientCount() int {
//	lobby.RLock()
//	defer lobby.RUnlock()
//	return len(lobby.Clients)
// }
//
// func (lobby *qpLobby) id() string {
//	lobby.RLock()
//	defer lobby.RUnlock()
//	return lobby.LobbyID
// }
//
// func (lobby *qpLobby) join(client *wsClient) error {
//	lobby.Lock()
//	defer lobby.Unlock()
//	if slices.Contains(lobby.Clients, client) {
//		return ErrDuplicateClient
//	}
//	lobby.Clients = append(lobby.Clients, client)
//	// TODO ensure uniq
//	client.lobbies = append(client.lobbies, lobby)
//	lobby.logger.Info("User joined lobby")
//	if len(lobby.Clients) == 1 {
//		return lobby.promote(client)
//	}
//	return nil
// }
//
// func (lobby *qpLobby) leave(client *wsClient) error {
//	lobby.Lock()
//	defer lobby.Unlock()
//	if !slices.Contains(lobby.Clients, client) {
//		return ErrUnknownClient
//	}
//	if len(lobby.Clients) == 1 {
//		return ErrEmptyLobby
//	}
//	lobby.Clients = fp.Remove(lobby.Clients, client)
//	client.removeLobby(lobby)
//
//	//if client.Leader {
//	//	client.Leader = false
//	//	return lobby.promote(lobby.Clients[0])
//	//}
//	return nil
// }
//
// func (lobby *qpLobby) promote(client *wsClient) error {
//	lobby.Leader = client
//	return nil
// }
//
// func (lobby *qpLobby) sendUserMessage(msg wsUserMessage) {
//	lobby.Lock()
//	defer lobby.Unlock()
//	lobby.Messages = append(lobby.Messages, msg)
// }
//
// func (lobby *qpLobby) broadcast(response wsRequest) error {
//	for _, client := range lobby.Clients {
//		client.send <- response
//	}
//	return nil
// }
//
// func sendJoinLobbySuccess(client *wsClient, lobby LobbyService) {
//	client.send <- wsRequest{
//		wsMsgTypeJoinLobbySuccess,
//		wsMsgJoinedLobbySuccess{
//			LobbyID: lobby.id(),
//		},
//	}
// }
