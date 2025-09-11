package auth

import (
	"net"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	AuthTokenDuration     = time.Hour * 24 * 31
	FingerprintCookieName = "fingerprint"
)

type UserTokens struct {
	Access      string `json:"access"`
	Fingerprint string `json:"fingerprint"`
}

type UserAuthClaims struct {
	// user context to prevent side-jacking
	// https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html#token-sidejacking
	Fingerprint string `json:"fingerprint"`
	jwt.RegisteredClaims
}

type ServerAuthClaims struct {
	ServerID int `json:"server_id"`
	// A random string which is used to fingerprint and prevent side-jacking
	jwt.RegisteredClaims
}

type PersonAuth struct {
	PersonAuthID int64           `json:"person_auth_id"`
	SteamID      steamid.SteamID `json:"steam_id"`
	IPAddr       net.IP          `json:"ip_addr"`
	AccessToken  string          `json:"access_token"`
	CreatedOn    time.Time       `json:"created_on"`
}

func NewPersonAuth(sid64 steamid.SteamID, addr net.IP, accessToken string) PersonAuth {
	return PersonAuth{
		PersonAuthID: 0,
		SteamID:      sid64,
		IPAddr:       addr,
		AccessToken:  accessToken,
		CreatedOn:    time.Now(),
	}
}
