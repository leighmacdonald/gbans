package app

import (
	"encoding/json"
	"sync"
)

type qpBaseResponse struct {
	MsgType qpMsgType `json:"msg_type"`
	Payload any       `json:"payload"`
}

type qpBasePayload struct {
	MsgType qpMsgType       `json:"msg_type"`
	Payload json.RawMessage `json:"payload"`
}

type qpConnectionManager struct {
	*sync.RWMutex
	connections qpClients
	lobbies     map[string]*qpLobby
}

type qpJoinLobbyRequestPayload struct {
	LobbyId string `json:"lobby_id"`
}

type qpUserMessage struct {
	SteamId   string `json:"steam_id"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type qpMsgJoinedLobbySuccess struct {
	Lobby *qpLobby `json:"lobby"`
}

type qpMsgErrorResponse struct {
	Error string `json:"error"`
}
