package mm

import (
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMatch(t *testing.T) {
	game := NewGame(Sixes)
	require.Equal(t, game.MaxPlayers, 12)
	p1 := NewPlayer([]logparse.PlayerClass{logparse.Demo}, steamid.RandSID64())
	p2 := NewPlayer([]logparse.PlayerClass{logparse.Demo}, steamid.RandSID64())
	require.NoError(t, game.Join(p1), "Failed to join game")
	require.NoError(t, game.Join(p2), "Failed to join game")
	require.NoError(t, game.Leave(p1), "Failed to leave game")
	require.Equal(t, 1, len(game.Players))
}
