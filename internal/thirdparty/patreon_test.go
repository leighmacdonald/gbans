package thirdparty

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewPatreonClient(t *testing.T) {
	c, errNew := NewPatreonClient(context.TODO(), config.Patreon.CreatorAccessToken)
	require.NoError(t, errNew)
	campaign, campaignErr := c.FetchCampaign()
	require.NoError(t, campaignErr)

	pledges, errPledges := c.FetchPledges(campaign.Data[0].ID)
	require.NoError(t, errPledges)
	require.Len(t, pledges, 1)
}
