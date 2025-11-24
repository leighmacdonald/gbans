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
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/json"
	"github.com/leighmacdonald/gbans/internal/oauth"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/oauth2"
)

const (
	authURL  = "https://www.patreon.com/oauth2/authorize"
	tokenURL = "https://www.patreon.com/api/oauth2/token"
)

var ErrQueryPatreon = errors.New("failed to query patreon")

type Config struct {
	Enabled             bool   `json:"enabled"`
	IntegrationsEnabled bool   `json:"integrations_enabled"`
	ClientID            string `json:"client_id"`
	ClientSecret        string `json:"client_secret"`
	CreatorAccessToken  string `json:"creator_access_token"`
	CreatorRefreshToken string `json:"creator_refresh_token"`
}

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
	Config

	// patreonClient    *patreon.Client
	patreonMu        *sync.RWMutex
	patreonCampaigns patreon.Campaign
}

func NewPatreonManager(config Config) Manager {
	return Manager{
		Config:    config,
		patreonMu: &sync.RWMutex{},
	}
}

// start https://www.patreon.com/portal/registration/register-clients
func (p *Manager) createClient(ctx context.Context, accessToken string, refreshToken string) *patreon.Client {
	oAuthConfig := oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
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
		return nil, errors.Join(err, ErrQueryPatreon)
	}

	if user == nil {
		return nil, ErrQueryPatreon
	}

	return user, nil
}

func (p *Manager) sync(ctx context.Context) {
	if !p.Enabled {
		return
	}

	client := p.createClient(ctx, p.CreatorAccessToken, p.CreatorRefreshToken)

	user, errUser := p.loadUser(client)
	if errUser != nil {
		slog.Error("Failed to load patreon user", slog.String("error", errUser.Error()))

		return
	}

	if user == nil {
		slog.Error("Failed to load user, nil result")

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
	Config

	repository   Repository
	manager      Manager
	stateTracker *oauth.LoginStateTracker
}

func NewPatreon(repository Repository, config Config) Patreon {
	return Patreon{
		Config:       config,
		repository:   repository,
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
		slog.Error("Failed to load old auths", slog.String("error", errOldAuths.Error()))

		return
	}

	for _, oldAuth := range oldAuths {
		if err := p.refreshToken(ctx, oldAuth); err != nil {
			slog.Error("Failed to refresh users patreon token", slog.String("error", err.Error()))
		}
	}
}

func (p Patreon) refreshToken(ctx context.Context, auth Credential) error {
	form := url.Values{}
	form.Add("grant_type", "refresh_token")
	form.Add("client_id", p.ClientID)
	form.Add("client_secret", p.ClientSecret)
	form.Add("refresh_token", auth.RefreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return errors.Join(err, httphelper.ErrRequestCreate)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	httpClient := httphelper.NewClient()

	resp, errResp := httpClient.Do(req)
	if errResp != nil {
		return errors.Join(errResp, httphelper.ErrRequestPerform)
	}

	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			slog.Error("Failed to close response body", slog.String("error", errClose.Error()))
		}
	}()

	creds, errDec := json.Decode[Credential](resp.Body)
	if errDec != nil {
		slog.Error("Failed to decode access token", slog.String("error", errDec.Error()))

		return errors.Join(errDec, httphelper.ErrRequestDecode)
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
	if user == nil {
		return ErrQueryPatreon
	}
	creds.PatreonID = user.Data.ID

	if errSave := p.repository.SaveTokens(ctx, creds); errSave != nil {
		return errSave
	}

	return nil
}

func (p Patreon) CreateOAuthRedirect(steamID steamid.SteamID) string {
	state := p.stateTracker.Create(steamID)

	authenticationURL, _ := url.Parse(authURL)
	values := authenticationURL.Query()
	values.Set("client_id", p.ClientID)
	values.Set("allow_signup", "false")
	values.Set("response_type", "code")
	values.Set("redirect_uri", link.Raw("/patreon/oauth"))
	values.Set("state", state)
	values.Set("scope", "campaigns identity campaigns.members")

	authenticationURL.RawQuery = values.Encode()

	return authenticationURL.String()
}

func (p Patreon) Campaign() patreon.Campaign {
	return p.manager.Campaigns()
}

func (p Patreon) OnOauthLogin(ctx context.Context, state string, code string) error {
	steamID, valid := p.stateTracker.Get(state)
	if !valid {
		return steamid.ErrInvalidSID
	}

	form := url.Values{}
	form.Add("code", code)
	form.Add("grant_type", "authorization_code")
	form.Add("client_id", p.ClientID)
	form.Add("client_secret", p.ClientSecret)
	form.Add("redirect_uri", link.Raw("/patreon/oauth"))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return errors.Join(err, httphelper.ErrRequestCreate)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	httpClient := httphelper.NewClient()

	resp, errResp := httpClient.Do(req)
	if errResp != nil {
		return errors.Join(errResp, httphelper.ErrRequestPerform)
	}

	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			slog.Error("Failed to close response body", slog.String("error", errClose.Error()))
		}
	}()

	creds, errDec := json.Decode[Credential](resp.Body)
	if errDec != nil {
		slog.Error("Failed to decode access token", slog.String("error", errDec.Error()))

		return errors.Join(errDec, httphelper.ErrRequestDecode)
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
	if user == nil {
		return ErrQueryPatreon
	}
	creds.PatreonID = user.Data.ID

	if errSave := p.repository.SaveTokens(ctx, creds); errSave != nil {
		return errSave
	}

	return nil
}
