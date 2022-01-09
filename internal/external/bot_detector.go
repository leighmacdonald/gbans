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
	var t tf2bdSchema
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	var ids []steamid.SID64
	for _, s := range t.Players {
		var id steamid.SID64
		switch s.Steamid.(type) {
		case string:
			var err error
			id, err = steamid.StringToSID64(s.Steamid.(string))
			if err != nil {
				return nil, err
			}
		case float64:
			id = steamid.SID64(s.Steamid.(float64))
		}
		if !id.Valid() {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}
