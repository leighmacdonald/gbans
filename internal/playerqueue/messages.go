package playerqueue

import (
	"time"

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

type ServerQueueState struct {
	ServerID int               `json:"server_id"`
	Members  []steamid.SteamID `json:"members"`
}

type clientStatePayload struct {
	UpdateUsers   bool               `json:"update_users"`
	UpdateServers bool               `json:"update_servers"`
	Servers       []ServerQueueState `json:"servers"`
	Users         []member           `json:"users"`
}
