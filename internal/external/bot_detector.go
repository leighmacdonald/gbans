package external

import (
	"encoding/json"
	"github.com/leighmacdonald/steamid/v2/steamid"
)

type tf2bdSchema struct {
	Schema   string `json:"$schema"`
	FileInfo struct {
		Authors     []string `json:"authors"`
		Description string   `json:"description"`
		Title       string   `json:"title"`
		UpdateURL   string   `json:"update_url"`
	} `json:"file_info"`
	Players []struct {
		Attributes []string `json:"attributes"`
		LastSeen   struct {
			PlayerName string `json:"player_name,omitempty"`
			Time       int    `json:"time,omitempty"`
		} `json:"last_seen,omitempty"`
		Steamid any      `json:"steamid"`
		Proof   []string `json:"proof,omitempty"`
	} `json:"players"`
}

func parseTF2BD(data []byte) ([]steamid.SID64, error) {
	var bdSchema tf2bdSchema
	if errUnmarshal := json.Unmarshal(data, &bdSchema); errUnmarshal != nil {
		return nil, errUnmarshal
	}
	var steamIds steamid.Collection
	for _, player := range bdSchema.Players {
		var id steamid.SID64
		switch player.Steamid.(type) {
		case string:
			parsedSid64, errParseSid64 := steamid.StringToSID64(player.Steamid.(string))
			if errParseSid64 != nil {
				return nil, errParseSid64
			}
			id = parsedSid64
		case float64:
			id = steamid.SID64(player.Steamid.(float64))
		}
		if !id.Valid() {
			continue
		}
		steamIds = append(steamIds, id)
	}
	return steamIds, nil
}
