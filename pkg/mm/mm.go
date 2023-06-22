package mm

import (
	"strings"

	"github.com/pkg/errors"
)

var (
	ErrPlayerExists  = errors.New("Duplicate player")
	ErrPlayerMissing = errors.New("Player does not exist")
)

type GameType int

const (
	Highlander GameType = iota
	Sixes
	Ultiduo
)

type GameConfig int

const (
	CfgRGL GameConfig = iota
	CfgUGC
	CfgOzFortress
)

func addTeamPrefixes(keys ...string) []string {
	var out []string
	for _, key := range keys {
		for _, t := range []string{"red", "blu"} {
			out = append(out, strings.Join([]string{t, key}, "_"))
		}
	}
	return out
}

var (
	ClassMappingKeysHL      = addTeamPrefixes("scout", "soldier", "pyro", "demoman", "heavyweapons", "engineer", "medic", "sniper", "spy")
	ClassMappingKeysSixes   = addTeamPrefixes("scout_pocket", "scout_flank", "soldier_pocket", "soldier_roamer", "demoman", "medic")
	ClassMappingKeysUltiduo = addTeamPrefixes("soldier", "medic")
)
