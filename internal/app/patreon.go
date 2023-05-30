package app

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"gopkg.in/mxpv/patreon-go.v1"
	"time"
)

type PatreonStore interface {
	SetPatreonAuth(ctx context.Context, accessToken string, refreshToken string) error
	GetPatreonAuth(ctx context.Context) (string, string, error)
}

// NewPatreonClient https://www.patreon.com/portal/registration/register-clients
func NewPatreonClient(ctx context.Context) (*patreon.Client, error) {
	cat, crt, errAuth := store.GetPatreonAuth(ctx)
	if errAuth != nil || cat == "" || crt == "" {
		// Attempt to use config file values as the initial source if we have nothing saved.
		// These are only used once as they are dynamically updated and stored
		// in the database for subsequent retrievals
		cat = config.Patreon.CreatorAccessToken
		crt = config.Patreon.CreatorRefreshToken
	}
	oAuthConfig := oauth2.Config{
		ClientID:     config.Patreon.ClientId,
		ClientSecret: config.Patreon.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  patreon.AuthorizationURL,
			TokenURL: patreon.AccessTokenURL,
		},
		Scopes: []string{"users", "pledges-to-me", "campaigns", "my-campaign"},
	}

	tok := &oauth2.Token{
		AccessToken:  cat,
		RefreshToken: crt,
		// Must be non-nil, otherwise token will not be expired
		Expiry: time.Now().Add(1 * time.Hour),
	}

	tc := oAuthConfig.Client(context.Background(), tok)
	client := patreon.NewClient(tc)

	if errUpdate := updateToken(ctx, oAuthConfig, tok); errUpdate != nil {
		return nil, errUpdate
	}
	// litmus test
	_, errFetchTest := client.FetchUser()
	if errFetchTest != nil {
		return nil, errFetchTest
	}
	go func() {
		t0 := time.NewTicker(time.Minute * 60)
		for {
			select {
			case <-t0.C:
				if errUpdate := updateToken(ctx, oAuthConfig, tok); errUpdate != nil {
					logger.Error("Failed to update patreon token", zap.Error(errUpdate))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return client, nil
}

func updateToken(ctx context.Context, oAuthConfig oauth2.Config, tok *oauth2.Token) error {
	tokSrc := oAuthConfig.TokenSource(ctx, tok)
	newToken, errToken := tokSrc.Token()
	if errToken != nil {
		return errors.Wrap(errToken, "Failed to get oath token")
	}
	if saveTokenErr := store.SetPatreonAuth(ctx, newToken.AccessToken, newToken.RefreshToken); saveTokenErr != nil {
		return errors.Wrap(errToken, "Failed to save new oath token")
	}
	*tok = *newToken
	return nil
}

func PatreonGetTiers(client *patreon.Client) ([]patreon.Campaign, error) {
	campaigns, campaignsErr := client.FetchCampaign()
	if campaignsErr != nil {
		return nil, campaignsErr
	}
	return campaigns.Data, nil
}

func PatreonGetPledges(client *patreon.Client) ([]patreon.Pledge, map[string]*patreon.User, error) {
	campaignResponse, err := client.FetchCampaign()
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to fetch campaign")
	}
	if len(campaignResponse.Data) == 0 {
		return nil, nil, errors.New("No campaign returned")
	}
	campaignId := campaignResponse.Data[0].ID

	cursor := ""
	page := 1
	var out []patreon.Pledge
	// Get all the users in an easy-to-lookup way
	users := make(map[string]*patreon.User)

	for {
		pledgesResponse, errFetch := client.FetchPledges(campaignId,
			patreon.WithPageSize(25),
			patreon.WithCursor(cursor))
		if errFetch != nil {
			return nil, nil, errFetch
		}

		for _, item := range pledgesResponse.Included.Items {
			u, ok := item.(*patreon.User)
			if !ok {
				continue
			}
			users[u.ID] = u
		}
		out = append(out, pledgesResponse.Data...)
		nextLink := pledgesResponse.Links.Next
		if nextLink == "" {
			break
		}
		cursor = nextLink
		page++
	}
	return out, users, nil
}
