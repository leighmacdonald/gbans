package patreon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/austinbspencer/patreon-go-wrapper"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type patreonUsecase struct {
	repository   domain.PatreonRepository
	manager      *Manager
	stateTracker *loginStateTracker
	cu           domain.ConfigUsecase
}

func NewPatreonUsecase(repository domain.PatreonRepository, configUsecase domain.ConfigUsecase) domain.PatreonUsecase {
	return &patreonUsecase{
		repository:   repository,
		cu:           configUsecase,
		manager:      NewPatreonManager(configUsecase),
		stateTracker: newLoginStateTracker(),
	}
}

func (p patreonUsecase) Forget(ctx context.Context, steamID steamid.SteamID) error {
	return p.repository.DeleteTokens(ctx, steamID)
}

func (p patreonUsecase) Start(ctx context.Context) {
	go p.manager.Start(ctx)

	p.checkAuths(ctx)

	ticker := time.NewTicker(time.Hour)

	for {
		select {
		case <-ticker.C:
			p.checkAuths(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (p patreonUsecase) checkAuths(ctx context.Context) {
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

func (p patreonUsecase) refreshToken(ctx context.Context, auth domain.PatreonCredential) error {
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

	httpClient := util.NewHTTPClient()

	resp, errResp := httpClient.Do(req)
	if errResp != nil {
		return errors.Join(errResp, domain.ErrRequestPerform)
	}

	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			slog.Error("Failed to close response body", log.ErrAttr(errClose))
		}
	}()

	var creds domain.PatreonCredential

	decoder := json.NewDecoder(resp.Body)
	if errDec := decoder.Decode(&creds); err != nil {
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

func (p patreonUsecase) CreateOAuthRedirect(steamID steamid.SteamID) string {
	conf := p.cu.Config()
	state := p.stateTracker.create(steamID)

	authorizeURL := fmt.Sprintf("https://www.patreon.com/oauth2/authorize?client_id=%s"+
		"&allow_signup=false"+
		"&response_type=code"+
		"&redirect_uri=%s"+
		"&state=%s",
		conf.Patreon.ClientID,
		conf.ExtURLRaw("/patreon/oauth"),
		state)

	return authorizeURL
}

func (p patreonUsecase) Campaign() patreon.Campaign {
	return p.manager.Campaigns()
}

func (p patreonUsecase) OnOauthLogin(ctx context.Context, state string, code string) error {
	steamID, valid := p.stateTracker.get(state)
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

	httpClient := util.NewHTTPClient()

	resp, errResp := httpClient.Do(req)
	if errResp != nil {
		return errors.Join(errResp, domain.ErrRequestPerform)
	}

	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			slog.Error("Failed to close response body", log.ErrAttr(errClose))
		}
	}()

	var creds domain.PatreonCredential

	decoder := json.NewDecoder(resp.Body)
	if errDec := decoder.Decode(&creds); err != nil {
		slog.Error("Failed to decode access token", log.ErrAttr(errDec))

		return errors.Join(errDec, domain.ErrRequestDecode)
	}

	now := time.Now()
	creds.CreatedOn = now
	creds.SteamID = steamID
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
