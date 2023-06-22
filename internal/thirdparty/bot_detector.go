package thirdparty

import (
	"encoding/json"

	"github.com/leighmacdonald/steamid/v2/steamid"
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
	var steamIds steamid.Collection
	for _, player := range bdSchema.Players {
		var id steamid.SID64
		switch player.Steamid.(type) {
		case string:
			sidVal, ok := player.Steamid.(string)
			if !ok {
				continue
			}
			parsedSid64, errParseSid64 := steamid.StringToSID64(sidVal)
			if errParseSid64 != nil {
				return nil, errParseSid64
			}
			id = parsedSid64
		case float64:
			sidVal, ok := player.Steamid.(float64)
			if !ok {
				continue
			}
			id = steamid.SID64(sidVal)
		}
		if !id.Valid() {
			continue
		}
		steamIds = append(steamIds, id)
	}
	return steamIds, nil
}
