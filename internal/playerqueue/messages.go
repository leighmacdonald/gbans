package playerqueue

import (
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type ByePayload struct {
	Message string `json:"message"`
}

type MessagePayload struct {
	Messages []domain.ChatLog `json:"messages"`
}

type JoinPayload struct {
	Servers []int `json:"servers"`
}

type LeavePayload struct {
	Servers []int `json:"servers"`
}

type MessageCreatePayload struct {
	BodyMD string `json:"body_md"`
}

type Member struct {
	Name    string `json:"name"`
	SteamID string `json:"steam_id"`
	Hash    string `json:"hash"`
}

type ClientQueueState struct {
	SteamID steamid.SteamID `json:"steam_id"`
}

type LobbyState struct {
	ServerID int                `json:"server_id"`
	Members  []ClientQueueState `json:"members"`
}

type ClientStatePayload struct {
	UpdateUsers   bool         `json:"update_users"`
	UpdateServers bool         `json:"update_servers"`
	Lobbies       []LobbyState `json:"lobbies"`
	Users         []Member     `json:"users"`
}

type LobbyServer struct {
	Name           string `json:"name"`
	ShortName      string `json:"short_name"`
	CC             string `json:"cc"`
	ConnectURL     string `json:"connect_url"`
	ConnectCommand string `json:"connect_command"`
}

type GameStartPayload struct {
	Users  []Member    `json:"users"`
	Server LobbyServer `json:"server"`
}

type PurgePayload struct {
	MessageIDs []int64 `json:"message_ids"` //nolint:tagliatelle
}
