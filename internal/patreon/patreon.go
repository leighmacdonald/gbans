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

	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/json"
	"github.com/leighmacdonald/gbans/internal/oauth"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	authURL  = "https://www.patreon.com/oauth2/authorize"
	tokenURL = "https://www.patreon.com/api/oauth2/token" //nolint:gosec
)

var ErrQueryPatreon = errors.New("failed to query patreon")

type Config struct {
	sync.RWMutex

	Enabled             bool
	IntegrationsEnabled bool
	ClientID            string
	ClientSecret        string
	CreatorAccessToken  string
	CreatorRefreshToken string
}

type Credential struct {
	SteamID      steamid.SteamID
	PatreonID    string
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	Scope        string
	TokenType    string
	Version      string
	CreatedOn    time.Time
	UpdatedOn    time.Time
}

type Manager struct {
	*Config

	patreonMu        *sync.RWMutex
	patreonCampaigns Campaign
}

func NewPatreonManager(config *Config) Manager {
	return Manager{
		Config:    config,
		patreonMu: &sync.RWMutex{},
	}
}

func (p *Manager) sync(ctx context.Context) {
	if !p.Enabled {
		return
	}

	if p.CreatorAccessToken == "" {
		slog.Debug("Skipping patreon sync, no creator access token configured")

		return
	}

	campaigns, errCampaigns := newClient(p.CreatorAccessToken).fetchCampaigns(ctx)
	if errCampaigns != nil {
		slog.Error("Failed to load patreon campaigns", slog.String("error", errCampaigns.Error()))

		return
	}

	if len(campaigns) == 0 {
		slog.Error("Failed to load patreon campaigns, empty result")

		return
	}

	p.patreonMu.Lock()
	p.patreonCampaigns = campaigns[0]
	p.patreonMu.Unlock()

	slog.Debug("Patreon campaign updated")
}

func (p *Manager) Campaigns() Campaign {
	p.patreonMu.RLock()
	defer p.patreonMu.RUnlock()

	return p.patreonCampaigns
}

type Patreon struct {
	*Config

	repository   Repository
	manager      Manager
	stateTracker *oauth.LoginStateTracker
}

func NewPatreon(repository Repository, config *Config) Patreon {
	return Patreon{
		Config:       config,
		repository:   repository,
		manager:      NewPatreonManager(config),
		stateTracker: oauth.NewLoginStateTracker(),
	}
}

func (p *Patreon) Forget(ctx context.Context, steamID steamid.SteamID) error {
	return p.repository.DeleteTokens(ctx, steamID)
}

func (p *Patreon) Sync(ctx context.Context) {
	p.manager.sync(ctx)
	p.checkAuths(ctx)
}

func (p *Patreon) checkAuths(ctx context.Context) {
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

func (p *Patreon) refreshToken(ctx context.Context, auth Credential) error {
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

	patreonID, errUser := newClient(creds.AccessToken).fetchIdentityUserID(ctx)
	if errUser != nil {
		return errUser
	}
	creds.PatreonID = patreonID

	if errSave := p.repository.SaveTokens(ctx, creds); errSave != nil {
		return errSave
	}

	return nil
}

func (p *Patreon) CreateOAuthRedirect(steamID steamid.SteamID) string {
	state := p.stateTracker.Create(steamID)

	authenticationURL, _ := url.Parse(authURL)
	values := authenticationURL.Query()
	values.Set("client_id", p.ClientID)
	values.Set("allow_signup", "false")
	values.Set("response_type", "code")
	values.Set("redirect_uri", link.Raw("/patreon/oauth"))
	values.Set("state", state)
	values.Set("scope", "identity campaigns")

	authenticationURL.RawQuery = values.Encode()

	return authenticationURL.String()
}

func (p *Patreon) Campaign() Campaign {
	return p.manager.Campaigns()
}

func (p *Patreon) OnOauthLogin(ctx context.Context, state string, code string) error {
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

	patreonID, errUser := newClient(creds.AccessToken).fetchIdentityUserID(ctx)
	if errUser != nil {
		return errUser
	}
	creds.PatreonID = patreonID

	if errSave := p.repository.SaveTokens(ctx, creds); errSave != nil {
		return errSave
	}

	return nil
}
