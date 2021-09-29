package external

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestFetchCompHist(t *testing.T) {
	const validSid = 76561197970669109
	var hist CompHist
	c, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	require.NoError(t, FetchCompHist(c, validSid, &hist))
	require.Equal(t, hist.RGLDiv, "Invite")
	require.Greater(t, hist.LogsCount, 100)
}

func TestGetRGLProfile(t *testing.T) {
	const invalidSID = 76561198084134021
	const validSID = 76561197970669109
	var (
		invalidProfile RGLProfile
		validProfile   RGLProfile
	)
	c, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	require.NoError(t, GetRGLProfile(c, validSID, &validProfile))
	require.Equal(t, "Invite", validProfile.Division)
	//require.Equal(t, "froyotech", validProfile.Team)
	require.ErrorIs(t, GetRGLProfile(c, invalidSID, &invalidProfile), ErrNoProfile)
}
