package thirdparty

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

type testDb struct {
	at string
	rt string
}

func (db *testDb) SetPatreonAuth(ctx context.Context, accessToken string, refreshToken string) error {
	db.at = accessToken
	db.rt = refreshToken
	return nil
}
func (db *testDb) GetPatreonAuth(ctx context.Context) (string, string, error) {
	return db.at, db.rt, nil
}

func TestNewPatreonClient(t *testing.T) {
	c, errNew := NewPatreonClient(context.TODO(), &testDb{})
	require.NoError(t, errNew)
	campaign, campaignErr := c.FetchCampaign()
	require.NoError(t, campaignErr)

	pledges, errPledges := c.FetchPledges(campaign.Data[0].ID)
	require.NoError(t, errPledges)
	require.Len(t, pledges, 1)
}
