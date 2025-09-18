package discordoauth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/oauth"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type DiscordCredential struct {
	SteamID      steamid.SteamID `json:"steam_id"`
	DiscordID    string          `json:"discord_id"`
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token"`
	ExpiresIn    int             `json:"expires_in"`
	Scope        string          `json:"scope"`
	TokenType    string          `json:"token_type"`
	CreatedOn    time.Time       `json:"created_on"`
	UpdatedOn    time.Time       `json:"updated_on"`
}

type DiscordUserDetail struct {
	SteamID          steamid.SteamID `json:"steam_id"`
	ID               string          `json:"id"`
	Username         string          `json:"username"`
	Avatar           string          `json:"avatar"`
	AvatarDecoration any             `json:"avatar_decoration"`
	Discriminator    string          `json:"discriminator"`
	PublicFlags      int             `json:"public_flags"`
	Flags            int             `json:"flags"`
	Banner           any             `json:"banner"`
	BannerColor      any             `json:"banner_color"`
	AccentColor      any             `json:"accent_color"`
	Locale           string          `json:"locale"`
	MfaEnabled       bool            `json:"mfa_enabled"`
	PremiumType      int             `json:"premium_type"`
	CreatedOn        time.Time       `json:"created_on"`
	UpdatedOn        time.Time       `json:"updated_on"`
}

type DiscordOAuth struct {
	config     *config.Configuration
	state      *oauth.LoginStateTracker
	repository DiscordOAuthRepository
}

func NewDiscordOAuth(repository DiscordOAuthRepository, config *config.Configuration) DiscordOAuth {
	return DiscordOAuth{
		repository: repository,
		config:     config,
		state:      oauth.NewLoginStateTracker(),
	}
}

func (d DiscordOAuth) GetUserDetail(ctx context.Context, steamID steamid.SteamID) (DiscordUserDetail, error) {
	return d.repository.GetUserDetail(ctx, steamID)
}

func (d DiscordOAuth) RefreshTokens(ctx context.Context) error {
	entries, errOld := d.repository.OldAuths(ctx)
	if errOld != nil {
		if errors.Is(errOld, database.ErrNoResult) {
			return nil
		}

		slog.Error("Failed to fetch old discord auth tokens", log.ErrAttr(errOld))

		return errOld
	}

	for _, old := range entries {
		newCreds, errRefresh := d.fetchRefresh(ctx, old)
		if errRefresh != nil {
			// slog.Error("Failed to refresh token", log.ErrAttr(errRefresh))
			continue
		}

		if err := d.repository.SaveTokens(ctx, newCreds); err != nil {
			slog.Error("Failed to save refresh tokens", log.ErrAttr(err))

			return err
		}

		slog.Debug("Updated discord tokens", slog.String("steam_id", newCreds.SteamID.String()))
	}

	return nil
}

func (d DiscordOAuth) fetchRefresh(ctx context.Context, credentials DiscordCredential) (DiscordCredential, error) {
	conf := d.config.Config()

	form := url.Values{}
	form.Set("client_id", conf.Discord.AppID)
	form.Set("client_secret", conf.Discord.AppSecret)
	form.Set("refresh_token", credentials.RefreshToken)
	form.Set("grant_type", "refresh_token")

	req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, "https://discord.com/api/oauth2/token",
		strings.NewReader(form.Encode()))

	if errReq != nil {
		return DiscordCredential{}, errors.Join(errReq, domain.ErrRequestCreate)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, errResp := httphelper.NewHTTPClient().Do(req)
	if errResp != nil {
		return DiscordCredential{}, errors.Join(errResp, domain.ErrRequestPerform)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var atr DiscordCredential
	if errJSON := json.NewDecoder(resp.Body).Decode(&atr); errJSON != nil {
		return DiscordCredential{}, errors.Join(errJSON, domain.ErrRequestDecode)
	}

	if atr.AccessToken == "" {
		return DiscordCredential{}, domain.ErrEmptyToken
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
func (d DiscordOAuth) Logout(ctx context.Context, steamID steamid.SteamID) error {
	userDetail, errDetails := d.repository.GetUserDetail(ctx, steamID)
	if errDetails != nil {
		return errDetails
	}

	token, errToken := d.repository.GetTokens(ctx, steamID)
	if errToken != nil && !errors.Is(errToken, httphelper.ErrNotFound) {
		return errToken
	}

	if err := d.repository.DeleteUserDetail(ctx, userDetail.SteamID); err != nil {
		return err
	}

	if token.AccessToken == "" {
		// We don't have a token for some reason, don't make request.
		return nil
	}

	conf := d.config.Config()

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

	resp, errResp := httphelper.NewHTTPClient().Do(req)
	if errResp != nil {
		return errors.Join(errResp, domain.ErrRequestPerform)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	return nil
}

func (d DiscordOAuth) CreateStatefulLoginURL(steamID steamid.SteamID) (string, error) {
	config := d.config.Config()

	inviteLink, errParse := url.Parse("https://discord.com/oauth2/authorize")
	if errParse != nil {
		return "", errors.Join(errParse, domain.ErrValidateURL)
	}

	values := inviteLink.Query()
	values.Set("client_id", config.Discord.AppID)
	values.Set("scope", "identify")
	values.Set("state", d.state.Create(steamID))
	values.Set("redirect_uri", config.ExtURLRaw("/discord/oauth"))
	values.Set("response_type", "code")

	inviteLink.RawQuery = values.Encode()

	return inviteLink.String(), nil
}

func (d DiscordOAuth) HandleOAuthCode(ctx context.Context, code string, state string) error {
	client := httphelper.NewHTTPClient()

	steamID, found := d.state.Get(state)
	if !found {
		return httphelper.ErrNotFound
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

func (d DiscordOAuth) fetchDiscordUser(ctx context.Context, client *http.Client, accessToken string, steamID steamid.SteamID) (DiscordUserDetail, error) {
	var details DiscordUserDetail

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

func (d DiscordOAuth) fetchToken(ctx context.Context, client *http.Client, code string) (DiscordCredential, error) {
	conf := d.config.Config()

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
		return DiscordCredential{}, errors.Join(errReq, domain.ErrRequestCreate)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, errResp := client.Do(req)
	if errResp != nil {
		return DiscordCredential{}, errors.Join(errResp, domain.ErrRequestPerform)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var atr DiscordCredential
	if errJSON := json.NewDecoder(resp.Body).Decode(&atr); errJSON != nil {
		return DiscordCredential{}, errors.Join(errJSON, domain.ErrRequestDecode)
	}

	if atr.AccessToken == "" {
		return DiscordCredential{}, domain.ErrEmptyToken
	}

	return atr, nil
}
