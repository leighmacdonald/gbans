package sourcemod_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/leighmacdonald/gbans/internal/sourcemod"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/stretchr/testify/require"
)

func TestSeedQueue_Allowed(t *testing.T) {
	seedQueue := sourcemod.NewSeedQueue()

	synctest.Test(t, func(t *testing.T) {
		// Allow first.
		require.True(t, seedQueue.Allowed(100, tests.UserSID.String()))
		// Block second attempt.
		require.False(t, seedQueue.Allowed(100, tests.UserSID.String()))
		// Allow it to timeout and do it again.
		time.Sleep(time.Minute * 10)
		require.True(t, seedQueue.Allowed(100, tests.UserSID.String()))
		// Try it as another user before timeout.
		require.False(t, seedQueue.Allowed(100, tests.ModSID.String()))
	})
}
