package playerqueue

import (
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type byePayload struct {
	Message string `json:"message"`
}

type emptyPayload struct{}

type JoinPayload struct {
	Servers []int `json:"servers"`
}

type LeavePayload struct {
	Servers []int `json:"servers"`
}

type MessageCreatePayload struct {
	BodyMD string `json:"body_md"`
}

type member struct {
	Name    string `json:"name"`
	SteamID string `json:"steam_id"`
	Hash    string `json:"hash"`
}

type ClientQueueState struct {
	SteamID steamid.SteamID `json:"steam_id"`
}

type queueState struct {
	ServerID int                `json:"server_id"`
	Members  []ClientQueueState `json:"members"`
}

type clientStatePayload struct {
	UpdateUsers   bool         `json:"update_users"`
	UpdateServers bool         `json:"update_servers"`
	Servers       []queueState `json:"servers"`
	Users         []member     `json:"users"`
}

type server struct {
	Name           string `json:"name"`
	ShortName      string `json:"short_name"`
	CC             string `json:"cc"`
	ConnectURL     string `json:"connect_url"`
	ConnectCommand string `json:"connect_command"`
}

type gameStartPayload struct {
	Users  []member `json:"users"`
	Server server   `json:"server"`
}

type purgePayload struct {
	MessageIDs []int64 `json:"message_ids"` //nolint:tagliatelle
}
