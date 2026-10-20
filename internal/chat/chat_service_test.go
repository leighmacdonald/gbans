//nolint:testpackage // exercises the unexported toMessage conversion directly.
package chat

import (
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToMessageMatchID(t *testing.T) {
	t.Parallel()

	t.Run("nil match id", func(t *testing.T) {
		t.Parallel()

		msg := toMessage(&QueryChatHistoryResult{})

		require.NotNil(t, msg)
		assert.Nil(t, msg.MatchId)
	})

	t.Run("match id", func(t *testing.T) {
		t.Parallel()

		matchID, errID := uuid.NewV4()
		require.NoError(t, errID)

		result := &QueryChatHistoryResult{}
		result.MatchID = &matchID

		msg := toMessage(result)

		require.NotNil(t, msg.MatchId)
		assert.Equal(t, matchID.String(), msg.GetMatchId())
	})
}
