package discord

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type discordOAuthUsecase struct {
	configUsecase domain.ConfigUsecase
	stateTracker  *util.LoginStateTracker
	repository    domain.DiscordOAuthRepository
}

func NewDiscordOAuthUsecase(repository domain.DiscordOAuthRepository, configUsecase domain.ConfigUsecase) domain.DiscordOAuthUsecase {
	return &discordOAuthUsecase{
		repository:    repository,
		configUsecase: configUsecase,
		stateTracker:  util.NewLoginStateTracker(),
	}
}

func (d discordOAuthUsecase) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)

	d.RefreshOldTokens(ctx)

	for {
		select {
		case <-ticker.C:
			d.RefreshOldTokens(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (d discordOAuthUsecase) GetUserDetail(ctx context.Context, steamID steamid.SteamID) (domain.DiscordUserDetail, error) {
	return d.repository.GetUserDetail(ctx, steamID)
}

func (d discordOAuthUsecase) RefreshOldTokens(ctx context.Context) {
	entries, errOld := d.repository.OldAuths(ctx)
	if errOld != nil {
		if errors.Is(errOld, domain.ErrNoResult) {
			return
		}

		slog.Error("Failed to fetch old discord auth tokens", log.ErrAttr(errOld))

		return
	}

	for _, old := range entries {
		newCreds, errRefresh := d.fetchRefresh(ctx, old)
		if errRefresh != nil {
			slog.Error("Failed to refresh token", log.ErrAttr(errRefresh))

			continue
		}

		if err := d.repository.SaveTokens(ctx, newCreds); err != nil {
			slog.Error("Failed to save refresh tokens", log.ErrAttr(err))
		}

		slog.Debug("Updated discord tokens", slog.String("steam_id", newCreds.SteamID.String()))
	}
}

func (d discordOAuthUsecase) fetchRefresh(ctx context.Context, credentials domain.DiscordCredential) (domain.DiscordCredential, error) {
	conf := d.configUsecase.Config()

	form := url.Values{}
	form.Set("client_id", conf.Discord.AppID)
	form.Set("client_secret", conf.Discord.AppSecret)
	form.Set("refresh_token", credentials.RefreshToken)
	form.Set("grant_type", "refresh_token")

	req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, "https://discord.com/api/oauth2/token",
		strings.NewReader(form.Encode()))

	if errReq != nil {
		return domain.DiscordCredential{}, errors.Join(errReq, domain.ErrRequestCreate)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, errResp := util.NewHTTPClient().Do(req)
	if errResp != nil {
		return domain.DiscordCredential{}, errors.Join(errResp, domain.ErrRequestPerform)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var atr domain.DiscordCredential
	if errJSON := json.NewDecoder(resp.Body).Decode(&atr); errJSON != nil {
		return domain.DiscordCredential{}, errors.Join(errJSON, domain.ErrRequestDecode)
	}

	if atr.AccessToken == "" {
		return domain.DiscordCredential{}, domain.ErrEmptyToken
	}

	credentials.RefreshToken = atr.RefreshToken
	credentials.AccessToken = atr.AccessToken
	credentials.Scope = atr.Scope
	credentials.ExpiresIn = atr.ExpiresIn
	credentials.TokenType = atr.TokenType
	credentials.UpdatedOn = time.Now()

	return credentials, nil
}

// Logout will delete users details and their associated token via cascading. A logout api request
// is also sent to discord.
func (d discordOAuthUsecase) Logout(ctx context.Context, steamID steamid.SteamID) error {
	userDetail, errDetails := d.repository.GetUserDetail(ctx, steamID)
	if errDetails != nil {
		return errDetails
	}

	token, errToken := d.repository.GetTokens(ctx, steamID)
	if errToken != nil && !errors.Is(errToken, domain.ErrNotFound) {
		return errToken
	}

	if err := d.repository.DeleteUserDetail(ctx, userDetail.SteamID); err != nil {
		return err
	}

	if token.AccessToken == "" {
		// We don't have a token for some reason, don't make request.
		return nil
	}

	conf := d.configUsecase.Config()

	form := url.Values{}
	form.Set("client_id", conf.Discord.AppID)
	form.Set("client_secret", conf.Discord.AppSecret)
	form.Set("token", token.AccessToken)
	form.Set("token_type_hint", "access_token")

	req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, "https://discord.com/api/oauth2/token/revoke", strings.NewReader(form.Encode()))
	if errReq != nil {
		return errors.Join(errReq, domain.ErrRequestCreate)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, errResp := util.NewHTTPClient().Do(req)
	if errResp != nil {
		return errors.Join(errResp, domain.ErrRequestPerform)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	return nil
}

func (d discordOAuthUsecase) CreateStatefulLoginURL(steamID steamid.SteamID) (string, error) {
	config := d.configUsecase.Config()

	inviteLink, errParse := url.Parse("https://discord.com/oauth2/authorize")
	if errParse != nil {
		return "", errors.Join(errParse, domain.ErrValidateURL)
	}

	values := inviteLink.Query()
	values.Set("client_id", config.Discord.AppID)
	values.Set("scope", "identify")
	values.Set("state", d.stateTracker.Create(steamID))
	values.Set("redirect_uri", config.ExtURLRaw("/discord/oauth"))
	values.Set("response_type", "code")

	inviteLink.RawQuery = values.Encode()

	return inviteLink.String(), nil
}

func (d discordOAuthUsecase) HandleOAuthCode(ctx context.Context, code string, state string) error {
	client := util.NewHTTPClient()

	steamID, found := d.stateTracker.Get(state)
	if !found {
		return domain.ErrNotFound
	}

	token, errToken := d.fetchToken(ctx, client, code)
	if errToken != nil {
		return errToken
	}

	discordUser, errID := d.fetchDiscordUser(ctx, client, token.AccessToken, steamID)
	if errID != nil {
		return errID
	}

	if discordUser.ID == "" {
		return errToken
	}

	// user details saved first to satisfy foreign key
	if err := d.repository.SaveUserDetail(ctx, discordUser); err != nil {
		return err
	}

	token.DiscordID = discordUser.ID
	token.SteamID = steamID

	if err := d.repository.SaveTokens(ctx, token); err != nil {
		return err
	}

	slog.Info("Discord account linked successfully",
		slog.String("discord_id", discordUser.ID),
		slog.String("sid64", steamID.String()))

	return nil
}

func (d discordOAuthUsecase) fetchDiscordUser(ctx context.Context, client *http.Client, accessToken string, steamID steamid.SteamID) (domain.DiscordUserDetail, error) {
	var details domain.DiscordUserDetail

	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, "https://discord.com/api/users/@me", nil)
	if errReq != nil {
		return details, errors.Join(errReq, domain.ErrRequestCreate)
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)
	resp, errResp := client.Do(req)

	if errResp != nil {
		return details, errors.Join(errResp, domain.ErrRequestPerform)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if errJSON := json.NewDecoder(resp.Body).Decode(&details); errJSON != nil {
		return details, errors.Join(errJSON, domain.ErrRequestDecode)
	}

	details.SteamID = steamID

	return details, nil
}

func (d discordOAuthUsecase) fetchToken(ctx context.Context, client *http.Client, code string) (domain.DiscordCredential, error) {
	conf := d.configUsecase.Config()

	form := url.Values{}
	form.Set("client_id", conf.Discord.AppID)
	form.Set("client_secret", conf.Discord.AppSecret)
	form.Set("redirect_uri", conf.ExtURLRaw("/discord/oauth"))
	form.Set("code", code)
	form.Set("grant_type", "authorization_code")
	// form.Set("state", state.String())
	form.Set("scope", "identify")
	req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, "https://discord.com/api/oauth2/token", strings.NewReader(form.Encode()))

	if errReq != nil {
		return domain.DiscordCredential{}, errors.Join(errReq, domain.ErrRequestCreate)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, errResp := client.Do(req)
	if errResp != nil {
		return domain.DiscordCredential{}, errors.Join(errResp, domain.ErrRequestPerform)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var atr domain.DiscordCredential
	if errJSON := json.NewDecoder(resp.Body).Decode(&atr); errJSON != nil {
		return domain.DiscordCredential{}, errors.Join(errJSON, domain.ErrRequestDecode)
	}

	if atr.AccessToken == "" {
		return domain.DiscordCredential{}, domain.ErrEmptyToken
	}

	return atr, nil
}
