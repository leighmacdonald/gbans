package mm

import (
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"time"
)

var (
	ErrPlayerExists  = errors.New("Duplicate player")
	ErrPlayerMissing = errors.New("Player does not exist")
)

type GameType int

const (
	Highlander GameType = iota
	Sixes
)

type Player struct {
	SteamId        steamid.SID64
	ClassSelection []logparse.PlayerClass
}

type Players []*Player

type Game struct {
	CreatedOn  time.Time
	GameType   GameType
	MaxPlayers int
	Players    Players
}

func (game *Game) Join(player *Player) error {
	if fp.Contains(game.Players, player) {
		return ErrPlayerExists
	}
	game.Players = append(game.Players, player)
	return nil
}

func (game *Game) Leave(player *Player) error {
	if !fp.Contains(game.Players, player) {
		return ErrPlayerMissing
	}
	game.Players = fp.Remove(game.Players, player)
	return nil
}

func (game *Game) Validate() []error {
	// Validate the current game settings
	return nil
}

func (game *Game) ReadyUp() []error {
	// Make sure everyone is ready before starting match
	return nil
}

func NewPlayer(preferredClass []logparse.PlayerClass, steamId steamid.SID64) *Player {
	return &Player{
		ClassSelection: preferredClass,
		SteamId:        steamId,
	}
}

func NewGame(gameType GameType) Game {
	maxPlayers := 12
	if gameType == Highlander {
		maxPlayers = 18
	}
	return Game{
		GameType:   gameType,
		MaxPlayers: maxPlayers,
		Players:    Players{},
		CreatedOn:  config.Now(),
	}
}
