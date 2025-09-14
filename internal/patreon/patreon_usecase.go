package patreon

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/austinbspencer/patreon-go-wrapper"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/queue"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/oauth"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/riverqueue/river"
)

type PatreonUsecase struct {
	repository   PatreonRepository
	manager      *Manager
	stateTracker *oauth.LoginStateTracker
	cu           *config.ConfigUsecase
}

func NewPatreonUsecase(repository PatreonRepository, configUsecase *config.ConfigUsecase) PatreonUsecase {
	return PatreonUsecase{
		repository:   repository,
		cu:           configUsecase,
		manager:      NewPatreonManager(configUsecase),
		stateTracker: oauth.NewLoginStateTracker(),
	}
}

func (p PatreonUsecase) Forget(ctx context.Context, steamID steamid.SteamID) error {
	return p.repository.DeleteTokens(ctx, steamID)
}

func (p PatreonUsecase) Sync(ctx context.Context) {
	p.manager.sync(ctx)
	p.checkAuths(ctx)
}

func (p PatreonUsecase) checkAuths(ctx context.Context) {
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

func (p PatreonUsecase) refreshToken(ctx context.Context, auth PatreonCredential) error {
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

	httpClient := httphelper.NewHTTPClient()

	resp, errResp := httpClient.Do(req)
	if errResp != nil {
		return errors.Join(errResp, domain.ErrRequestPerform)
	}

	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			slog.Error("Failed to close response body", log.ErrAttr(errClose))
		}
	}()

	var creds PatreonCredential

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

func (p PatreonUsecase) CreateOAuthRedirect(steamID steamid.SteamID) string {
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

func (p PatreonUsecase) Campaign() patreon.Campaign {
	return p.manager.Campaigns()
}

func (p PatreonUsecase) OnOauthLogin(ctx context.Context, state string, code string) error {
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

	httpClient := httphelper.NewHTTPClient()

	resp, errResp := httpClient.Do(req)
	if errResp != nil {
		return errors.Join(errResp, domain.ErrRequestPerform)
	}

	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			slog.Error("Failed to close response body", log.ErrAttr(errClose))
		}
	}()

	var creds PatreonCredential

	decoder := json.NewDecoder(resp.Body)
	if errDec := decoder.Decode(&creds); err != nil {
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

type AuthUpdateArgs struct{}

func (args AuthUpdateArgs) Kind() string {
	return "patreon_auth"
}

func (args AuthUpdateArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: string(queue.Default), UniqueOpts: river.UniqueOpts{ByPeriod: time.Hour}}
}

func NewSyncWorker(patreon PatreonUsecase) *SyncWorker {
	return &SyncWorker{patreon: patreon}
}

type SyncWorker struct {
	river.WorkerDefaults[AuthUpdateArgs]
	patreon PatreonUsecase
}

func (worker *SyncWorker) Work(ctx context.Context, _ *river.Job[AuthUpdateArgs]) error {
	worker.patreon.Sync(ctx)

	return nil
}
