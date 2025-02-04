package playerqueue

import (
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type pingPayload struct {
	CreatedOn time.Time `json:"created_on"`
}

type pongPayload = pingPayload

type joinPayload struct {
	Servers []int `json:"servers"`
}

type leavePayload struct {
	Servers []int `json:"servers"`
}

type member struct {
	Name    string `json:"name"`
	SteamID string `json:"steam_id"`
	Hash    string `json:"hash"`
}

type ClientQueueState struct {
	SteamID steamid.SteamID `json:"steam_id"`
}

type ServerQueueState struct {
	ServerID int                `json:"server_id"`
	Members  []ClientQueueState `json:"members"`
}

type clientStatePayload struct {
	UpdateUsers   bool               `json:"update_users"`
	UpdateServers bool               `json:"update_servers"`
	Servers       []ServerQueueState `json:"servers"`
	Users         []member           `json:"users"`
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
	MessageIDs []uuid.UUID `json:"message_ids"`
}
