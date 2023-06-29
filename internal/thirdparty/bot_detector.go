package thirdparty

import (
	"encoding/json"

	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
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
	Schema   string    `json:"$schema"` //nolint:tagliatelle
	FileInfo FileInfo  `json:"file_info"`
	Players  []Players `json:"players"`
}

func parseTF2BD(data []byte) ([]steamid.SID64, error) {
	var bdSchema TF2BDSchema
	if errUnmarshal := json.Unmarshal(data, &bdSchema); errUnmarshal != nil {
		return nil, errors.Wrap(errUnmarshal, "Failed to unmarshal tf2bd schema")
	}

	var steamIds []steamid.SID64 //nolint:prealloc
	for _, player := range bdSchema.Players {
		steamID := steamid.New(player.Steamid)
		if !steamID.Valid() {
			continue
		}

		steamIds = append(steamIds, steamID)
	}

	return steamIds, nil
}
