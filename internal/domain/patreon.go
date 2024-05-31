package domain

import (
	"context"

	"github.com/austinbspencer/patreon-go-wrapper"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type PatreonUsecase interface {
	Start(ctx context.Context)
	Campaign() patreon.Campaign
	OnOauthLogin(ctx context.Context, state string, code string) error
	CreateOAuthRedirect(steamID steamid.SteamID) string
	Forget(ctx context.Context, steamID steamid.SteamID) error
}

type PatreonRepository interface {
	SetPatreonAuth(ctx context.Context, accessToken string, refreshToken string) error
	GetPatreonAuth(ctx context.Context) (string, string, error)
	SaveTokens(ctx context.Context, creds PatreonCredential) error
	GetTokens(ctx context.Context, steamID steamid.SteamID) (PatreonCredential, error)
	DeleteTokens(ctx context.Context, steamID steamid.SteamID) error
}

type PatreonCredential struct {
	SteamID      steamid.SteamID `json:"steam_id"`
	PatreonID    string          `json:"patreon_id"`
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token"`
	ExpiresIn    int             `json:"expires_in"`
	Scope        string          `json:"scope"`
	TokenType    string          `json:"token_type"`
	Version      string          `json:"version"`
}
