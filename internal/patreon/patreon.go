package patreon

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/austinbspencer/patreon-go-wrapper"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/json"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/oauth"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/oauth2"
)

type Credential struct {
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
	config           *config.Configuration
}

func NewPatreonManager(config *config.Configuration) Manager {
	return Manager{
		patreonMu: &sync.RWMutex{},
		config:    config,
	}
}

// start https://www.patreon.com/portal/registration/register-clients
func (p *Manager) createClient(ctx context.Context, accessToken string, refreshToken string) *patreon.Client {
	config := p.config.Config()

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
	conf := p.config.Config()
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

type Patreon struct {
	repository   Repository
	manager      Manager
	stateTracker *oauth.LoginStateTracker
	cu           *config.Configuration
}

func NewPatreon(repository Repository, config *config.Configuration) Patreon {
	return Patreon{
		repository:   repository,
		cu:           config,
		manager:      NewPatreonManager(config),
		stateTracker: oauth.NewLoginStateTracker(),
	}
}

func (p Patreon) Forget(ctx context.Context, steamID steamid.SteamID) error {
	return p.repository.DeleteTokens(ctx, steamID)
}

func (p Patreon) Sync(ctx context.Context) {
	p.manager.sync(ctx)
	p.checkAuths(ctx)
}

func (p Patreon) checkAuths(ctx context.Context) {
	oldAuths, errOldAuths := p.repository.OldAuths(ctx)
	if errOldAuths != nil {
		slog.Error("Failed to load old auths", log.ErrAttr(errOldAuths))

		return
	}

	for _, oldAuth := range oldAuths {
		if err := p.refreshToken(ctx, oldAuth); err != nil {
			slog.Error("Failed to refresh users patreon token", log.ErrAttr(err))
		}
	}
}

func (p Patreon) refreshToken(ctx context.Context, auth Credential) error {
	conf := p.cu.Config()

	form := url.Values{}
	form.Add("grant_type", "refresh_token")
	form.Add("client_id", conf.Patreon.ClientID)
	form.Add("client_secret", conf.Patreon.ClientSecret)
	form.Add("refresh_token", auth.RefreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://www.patreon.com/api/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return errors.Join(err, domain.ErrRequestCreate)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	httpClient := httphelper.NewClient()

	resp, errResp := httpClient.Do(req)
	if errResp != nil {
		return errors.Join(errResp, domain.ErrRequestPerform)
	}

	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			slog.Error("Failed to close response body", log.ErrAttr(errClose))
		}
	}()

	creds, errDec := json.Decode[Credential](resp.Body)
	if errDec != nil {
		slog.Error("Failed to decode access token", log.ErrAttr(errDec))

		return errors.Join(errDec, domain.ErrRequestDecode)
	}

	now := time.Now()
	creds.CreatedOn = now
	creds.SteamID = auth.SteamID
	creds.UpdatedOn = now

	client := p.manager.createClient(ctx, creds.AccessToken, creds.RefreshToken)

	user, errUser := p.manager.loadUser(client)
	if errUser != nil {
		return errUser
	}

	creds.PatreonID = user.Data.ID

	if errSave := p.repository.SaveTokens(ctx, creds); errSave != nil {
		return errSave
	}

	return nil
}

func (p Patreon) CreateOAuthRedirect(steamID steamid.SteamID) string {
	conf := p.cu.Config()
	state := p.stateTracker.Create(steamID)

	authURL, _ := url.Parse("https://www.patreon.com/oauth2/authorize")
	values := authURL.Query()
	values.Set("client_id", conf.Patreon.ClientID)
	values.Set("allow_signup", "false")
	values.Set("response_type", "code")
	values.Set("redirect_uri", conf.ExtURLRaw("/patreon/oauth"))
	values.Set("state", state)
	values.Set("scope", "campaigns identity campaigns.members")

	authURL.RawQuery = values.Encode()

	return authURL.String()
}

func (p Patreon) Campaign() patreon.Campaign {
	return p.manager.Campaigns()
}

func (p Patreon) OnOauthLogin(ctx context.Context, state string, code string) error {
	steamID, valid := p.stateTracker.Get(state)
	if !valid {
		return domain.ErrInvalidSID
	}

	conf := p.cu.Config()

	form := url.Values{}
	form.Add("code", code)
	form.Add("grant_type", "authorization_code")
	form.Add("client_id", conf.Patreon.ClientID)
	form.Add("client_secret", conf.Patreon.ClientSecret)
	form.Add("redirect_uri", conf.ExtURLRaw("/patreon/oauth"))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://www.patreon.com/api/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return errors.Join(err, domain.ErrRequestCreate)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	httpClient := httphelper.NewClient()

	resp, errResp := httpClient.Do(req)
	if errResp != nil {
		return errors.Join(errResp, domain.ErrRequestPerform)
	}

	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			slog.Error("Failed to close response body", log.ErrAttr(errClose))
		}
	}()

	creds, errDec := json.Decode[Credential](resp.Body)
	if errDec != nil {
		slog.Error("Failed to decode access token", log.ErrAttr(errDec))

		return errors.Join(errDec, domain.ErrRequestDecode)
	}

	now := time.Now()
	creds.CreatedOn = now
	creds.UpdatedOn = now
	creds.SteamID = steamID

	client := p.manager.createClient(ctx, creds.AccessToken, creds.RefreshToken)

	user, errUser := p.manager.loadUser(client)
	if errUser != nil {
		return errUser
	}

	creds.PatreonID = user.Data.ID

	if errSave := p.repository.SaveTokens(ctx, creds); errSave != nil {
		return errSave
	}

	return nil
}
