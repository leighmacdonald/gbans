package anticheat

import (
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/logparse"
)

type AnticheatEntry struct {
	logparse.StacEntry
	Personaname string `json:"personaname"`
	AvatarHash  string `json:"avatar_hash"`
	Triggered   int    `json:"triggered"`
}

type AnticheatQuery struct {
	domain.QueryFilter
	Name      string             `json:"name" schema:"name"`
	SteamID   string             `json:"steam_id" schema:"steam_id"`
	ServerID  int                `json:"server_id" schema:"server_id"`
	Summary   string             `json:"summary" schema:"summary"`
	Detection logparse.Detection `json:"detection" schema:"detection"`
}
