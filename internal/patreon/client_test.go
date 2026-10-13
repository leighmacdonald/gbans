//nolint:testpackage // Must be internal to cover the unexported APIv2 client.
package patreon

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockAPIv2(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return newClientForURL(server.URL, "test-token")
}

func TestFetchCampaigns(t *testing.T) {
	client := newMockAPIv2(t, func(res http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/api/oauth2/v2/campaigns", req.URL.Path)
		assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
		assert.Contains(t, req.URL.Query().Get("fields[campaign]"), "creation_name")

		res.Header().Set("Content-Type", "application/json")
		_, _ = res.Write([]byte(`{"data": [{
			"id": "1234560",
			"type": "campaign",
			"attributes": {
				"creation_name": "Test Creations",
				"image_small_url": "https://example.com/small.png",
				"image_url": "https://example.com/banner.png",
				"summary": null,
				"thanks_msg": "Thanks!",
				"patron_count": 42,
				"url": "https://www.patreon.com/test",
				"vanity": "test",
				"is_monthly": true,
				"created_at": "2018-05-04T23:34:08+00:00",
				"published_at": null
			}
		}]}`))
	})

	campaigns, err := client.fetchCampaigns(t.Context())
	require.NoError(t, err)
	require.Len(t, campaigns, 1)

	campaign := campaigns[0]
	require.Equal(t, "1234560", campaign.ID)
	require.Equal(t, "Test Creations", campaign.CreationName)
	require.Equal(t, "https://example.com/small.png", campaign.ImageSmallURL)
	require.Equal(t, "https://example.com/banner.png", campaign.ImageURL)
	require.Empty(t, campaign.Summary)
	require.Equal(t, "Thanks!", campaign.ThanksMsg)
	require.Equal(t, int32(42), campaign.PatronCount)
	require.Equal(t, "https://www.patreon.com/test", campaign.URL)
	require.Equal(t, "test", campaign.Vanity)
	require.True(t, campaign.IsMonthly)
	require.Equal(t, 2018, campaign.CreatedAt.Year())
	require.True(t, campaign.PublishedAt.IsZero())
}

func TestFetchCampaignsError(t *testing.T) {
	client := newMockAPIv2(t, func(res http.ResponseWriter, _ *http.Request) {
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusUnauthorized)
		_, _ = res.Write([]byte(`{"errors": [{"title": "Unauthorized", "detail": "Invalid token", "status": "401"}]}`))
	})

	_, err := client.fetchCampaigns(t.Context())
	require.Error(t, err)
	require.ErrorIs(t, err, ErrQueryPatreon)
	require.Contains(t, err.Error(), "Invalid token")
}

func TestFetchIdentityUserID(t *testing.T) {
	client := newMockAPIv2(t, func(res http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/api/oauth2/v2/identity", req.URL.Path)
		assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))

		res.Header().Set("Content-Type", "application/json")
		_, _ = res.Write([]byte(`{"data": {"id": "98765", "type": "user"}}`))
	})

	userID, err := client.fetchIdentityUserID(t.Context())
	require.NoError(t, err)
	require.Equal(t, "98765", userID)
}

func TestFetchIdentityUserIDEmpty(t *testing.T) {
	client := newMockAPIv2(t, func(res http.ResponseWriter, _ *http.Request) {
		res.Header().Set("Content-Type", "application/json")
		_, _ = res.Write([]byte(`{"data": {"id": "", "type": "user"}}`))
	})

	_, err := client.fetchIdentityUserID(t.Context())
	require.ErrorIs(t, err, ErrQueryPatreon)
}
