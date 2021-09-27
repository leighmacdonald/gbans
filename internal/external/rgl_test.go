package external

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetRGLProfile(t *testing.T) {
	const invalidSID = 76561198084134021
	const validSID = 76561197970669109
	var (
		invalidProfile RGLProfile
		validProfile   RGLProfile
	)
	require.ErrorIs(t, GetRGLProfile(invalidSID, &invalidProfile), ErrNoProfile)
	require.NoError(t, GetRGLProfile(validSID, &validProfile))
	require.Equal(t, "Invite", validProfile.Division)
	require.Equal(t, "froyotech", validProfile.Team)
}
