package thirdparty

import (
	"encoding/json"

	"github.com/leighmacdonald/steamid/v3/steamid"
)

type FileInfo struct {
	Authors     []string `json:"authors"`
	Description string   `json:"description"`
	Title       string   `json:"title"`
	UpdateURL   string   `json:"update_url"`
}
type LastSeen struct {
	PlayerName string `json:"player_name,omitempty"`
	Time       int    `json:"time,omitempty"`
}
type Players struct {
	Attributes []string `json:"attributes"`
	LastSeen   LastSeen `json:"last_seen,omitempty"`
	Steamid    any      `json:"steamid"`
	Proof      []string `json:"proof,omitempty"`
}

type TF2BDSchema struct {
	Schema   string    `json:"$schema"`
	FileInfo FileInfo  `json:"file_info"`
	Players  []Players `json:"players"`
}

func parseTF2BD(data []byte) ([]steamid.SID64, error) {
	var bdSchema TF2BDSchema
	if errUnmarshal := json.Unmarshal(data, &bdSchema); errUnmarshal != nil {
		return nil, errUnmarshal
	}

	steamIds := make([]steamid.SID64, len(bdSchema.Players))
	for index, player := range bdSchema.Players {
		steamID := steamid.New(player.Steamid)
		if !steamID.Valid() {
			continue
		}

		steamIds[index] = steamID
	}

	return steamIds, nil
}
