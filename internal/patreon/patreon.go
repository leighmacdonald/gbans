package patreon

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/austinbspencer/patreon-go-wrapper"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/oauth2"
)

type PatreonCredential struct {
	SteamID      steamid.SteamID `json:"steam_id"`
	PatreonID    string          `json:"patreon_id"`
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token"`
	ExpiresIn    int             `json:"expires_in"`
	Scope        string          `json:"scope"`
	TokenType    string          `json:"token_type"`
	Version      string          `json:"version"`
	CreatedOn    time.Time       `json:"created_on"`
	UpdatedOn    time.Time       `json:"updated_on"`
}

type Manager struct {
	// patreonClient    *patreon.Client
	patreonMu        *sync.RWMutex
	patreonCampaigns patreon.Campaign
	configUsecase    *config.ConfigUsecase
}

func NewPatreonManager(configUsecase *config.ConfigUsecase) *Manager {
	return &Manager{
		patreonMu:     &sync.RWMutex{},
		configUsecase: configUsecase,
	}
}

// start https://www.patreon.com/portal/registration/register-clients
func (p *Manager) createClient(ctx context.Context, accessToken string, refreshToken string) *patreon.Client {
	config := p.configUsecase.Config()

	oAuthConfig := oauth2.Config{
		ClientID:     config.Patreon.ClientID,
		ClientSecret: config.Patreon.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  patreon.AuthorizationURL,
			TokenURL: patreon.AccessTokenURL,
		},
		Scopes: []string{"campaigns", "identity", "memberships"},
	}

	tok := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		// Must be non-nil, otherwise token will not be expired
		Expiry: time.Now().AddDate(1, 0, 0),
	}

	tc := oAuthConfig.Client(ctx, tok)

	return patreon.NewClient(tc)
}

func (p *Manager) loadUser(client *patreon.Client) (*patreon.UserResponse, error) {
	fieldOpts := patreon.WithFields("user", patreon.UserFields...)
	campOpts := patreon.WithFields("campaign", patreon.CampaignFields...)
	includeOpts := patreon.WithIncludes("campaign")

	user, err := client.FetchIdentity(fieldOpts, campOpts, includeOpts)
	if err != nil {
		return nil, errors.Join(err, domain.ErrQueryPatreon)
	}

	return user, nil
}

func (p *Manager) sync(ctx context.Context) {
	conf := p.configUsecase.Config()
	if !conf.Patreon.Enabled {
		return
	}

	client := p.createClient(ctx, conf.Patreon.CreatorAccessToken, conf.Patreon.CreatorRefreshToken)

	user, errUser := p.loadUser(client)
	if errUser != nil {
		slog.Error("Failed to load patreon user", log.ErrAttr(errUser))

		return
	}

	for _, item := range user.Included.Items {
		res, ok := item.(*patreon.Campaign)
		if !ok {
			slog.Error("Got malformed campaign")

			continue
		}

		p.patreonMu.Lock()
		p.patreonCampaigns = *res
		p.patreonMu.Unlock()

		slog.Debug("Patreon campaign updated")

		break
	}
}

func (p *Manager) Campaigns() patreon.Campaign {
	p.patreonMu.RLock()
	defer p.patreonMu.RUnlock()

	return p.patreonCampaigns
}
