package patreon

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"gopkg.in/mxpv/patreon-go.v1"
)

type Manager struct {
	patreonClient    *patreon.Client
	patreonMu        *sync.RWMutex
	patreonCampaigns []patreon.Campaign
	patreonPledges   []patreon.Pledge
}

func NewPatreonManager() *Manager {
	return &Manager{
		patreonMu: &sync.RWMutex{},
	}
}

// // start https://www.patreon.com/portal/registration/register-clients
// func (p *PatreonManager) start(ctx context.Context) (*patreon.Client, error) {
//	log := p.log.Named("patreonClient")
//	cat, crt, errAuth := p.db.GetPatreonAuth(ctx)
//
//	if errAuth != nil || cat == "" || crt == "" {
//		// Attempt to use Config file values as the initial source if we have nothing saved.
//		// These are only used once as they are dynamically updated and stored
//		// in the database for subsequent retrievals
//		cat = p.conf.Patreon.CreatorAccessToken
//		crt = p.conf.Patreon.CreatorRefreshToken
//	}
//
//	oAuthConfig := oauth2.Config{
//		ClientID:     p.conf.Patreon.ClientID,
//		ClientSecret: p.conf.Patreon.ClientSecret,
//		Endpoint: oauth2.Endpoint{
//			AuthURL:  patreon.AuthorizationURL,
//			TokenURL: patreon.AccessTokenURL,
//		},
//		Scopes: []string{"users", "Pledges-to-me", "campaigns", "my-campaign"},
//	}
//
//	tok := &oauth2.Token{
//		AccessToken:  cat,
//		RefreshToken: crt,
//		// Must be non-nil, otherwise token will not be expired
//		Expiry: time.Now().Add(1 * time.Hour),
//	}
//
//	tc := oAuthConfig.Client(ctx, tok)
//
//	p.patreonClient = patreon.NewClient(tc)
//	if errUpdate := updateToken(ctx, p.db, oAuthConfig, tok); errUpdate != nil {
//		return nil, errUpdate
//	}
//	// litmus test
//	_, errFetchTest := p.patreonClient.FetchUser()
//	if errFetchTest != nil {
//		return nil, errors.Wrap(errFetchTest, "Failed to fetch patreon user")
//	}
//
//	go func() {
//		t0 := time.NewTicker(time.Minute * 60)
//
//		for {
//			select {
//			case <-t0.C:
//				if errUpdate := updateToken(ctx, p.db, oAuthConfig, tok); errUpdate != nil {
//					log.Error("Failed to update patreon token", log.ErrAttr(errUpdate))
//				}
//			case <-ctx.Done():
//				return
//			}
//		}
//	}()
//
//	return p.patreonClient, nil
// }
//
// unc updateToken(ctx context.Context, database patreonStore, oAuthConfig oauth2.Config, tok *oauth2.Token) error {
//	tokSrc := oAuthConfig.TokenSource(ctx, tok)
//
//	newToken, errToken := tokSrc.Token()
//	if errToken != nil {
//		return errors.Wrap(errToken, "Failed to get oath token")
//	}
//
//	if saveTokenErr := database.SetPatreonAuth(ctx, newToken.AccessToken, newToken.RefreshToken); saveTokenErr != nil {
//		return errors.Wrap(errToken, "Failed to save new oath token")
//	}
//
//	*tok = *newToken
//
//	return nil
// }

func (p *Manager) Tiers() ([]patreon.Campaign, error) {
	campaigns, errCampaigns := p.patreonClient.FetchCampaign()
	if errCampaigns != nil {
		return nil, errors.Join(errCampaigns, domain.ErrPatreonFetchCampaign)
	}

	return campaigns.Data, nil
}

func (p *Manager) Pledges() ([]patreon.Pledge, map[string]*patreon.User, error) {
	campaignResponse, err := p.patreonClient.FetchCampaign()
	if err != nil {
		return nil, nil, errors.Join(err, domain.ErrPatreonFetchCampaign)
	}

	if len(campaignResponse.Data) == 0 {
		return nil, nil, domain.ErrPatreonInvalidCampaign
	}

	var (
		campaignID = campaignResponse.Data[0].ID
		cursor     = ""
		page       = 1
		out        []patreon.Pledge
		users      = make(map[string]*patreon.User) // Get all the users in an easy-to-lookup way
	)

	for {
		pledgesResponse, errFetch := p.patreonClient.FetchPledges(campaignID,
			patreon.WithPageSize(25),
			patreon.WithCursor(cursor))
		if errFetch != nil {
			return nil, nil, errors.Join(errFetch, domain.ErrPatreonFetchPledges)
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

func (p *Manager) Start(ctx context.Context) {
	var (
		updateTimer = time.NewTicker(time.Hour * 1)
		updateChan  = make(chan any)
	)

	if p.patreonClient == nil {
		return
	}

	go func() {
		updateChan <- true
	}()

	for {
		select {
		case <-updateTimer.C:
			updateChan <- true
		case <-updateChan:
			newCampaigns, errCampaigns := p.Tiers()
			if errCampaigns != nil {
				slog.Error("Failed to refresh campaigns", log.ErrAttr(errCampaigns))

				return
			}

			newPledges, _, errPledges := p.Pledges()
			if errPledges != nil {
				slog.Error("Failed to refresh Pledges", log.ErrAttr(errPledges))

				return
			}

			p.patreonMu.Lock()
			p.patreonCampaigns = newCampaigns
			p.patreonPledges = newPledges
			// patreonUsers = newUsers
			p.patreonMu.Unlock()

			cents := 0
			totalCents := 0

			for _, pledge := range newPledges {
				cents += pledge.Attributes.AmountCents

				if pledge.Attributes.TotalHistoricalAmountCents != nil {
					totalCents += *pledge.Attributes.TotalHistoricalAmountCents
				}
			}

			slog.Info("Patreon Updated", slog.Int("campaign_count", len(newCampaigns)),
				slog.Int("current_cents", cents), slog.Int("total_cents", totalCents))
		case <-ctx.Done():
			return
		}
	}
}
