package auth

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	TokenDuration         = time.Hour * 24 * 31
	FingerprintCookieName = "fingerprint"
)

type PersonAuth struct {
	PersonAuthID int64           `json:"person_auth_id"`
	SteamID      steamid.SteamID `json:"steam_id"`
	IPAddr       net.IP          `json:"ip_addr"`
	AccessToken  string          `json:"access_token"`
	CreatedOn    time.Time       `json:"created_on"`
}

type ServerAuthClaims struct {
	jwt.RegisteredClaims

	ServerID int `json:"server_id"`
}

const CtxKeyUserProfile = "user_profile"

type Authentication struct {
	auth      Repository
	persons   *person.Persons
	bans      ban.Bans
	servers   *servers.Servers
	sentryDSN string
	siteName  string
	cookieKey string
}

func NewAuthentication(repository Repository, siteName string, cookieKey string, persons *person.Persons,
	bans ban.Bans, servers *servers.Servers, sentryDSN string,
) *Authentication {
	return &Authentication{
		auth:      repository,
		persons:   persons,
		bans:      bans,
		servers:   servers,
		sentryDSN: sentryDSN,
		siteName:  siteName,
		cookieKey: cookieKey,
	}
}

func (u *Authentication) DeletePersonAuth(ctx context.Context, authID int64) error {
	return u.auth.DeletePersonAuth(ctx, authID)
}

func (u *Authentication) GetPersonAuthByRefreshToken(ctx context.Context, token string, auth *PersonAuth) error {
	return u.auth.GetPersonAuthByFingerprint(ctx, token, auth)
}

func (u *Authentication) loginSID(ctx *gin.Context, level permission.Privilege, steamID steamid.SteamID) {
	loggedInPerson, errGetPerson := u.persons.BySteamID(ctx, steamID)
	if errGetPerson != nil {
		slog.Error("Failed to load person during auth", slog.String("error", errGetPerson.Error()))
		ctx.AbortWithStatus(http.StatusForbidden)

		return
	}
	if u.sentryDSN != "" {
		sentry.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetUser(sentry.User{
				ID:        loggedInPerson.SteamID.String(),
				IPAddress: ctx.ClientIP(),
				Username:  loggedInPerson.PersonaName,
			})
		})
	}
	if level > loggedInPerson.PermissionLevel {
		ctx.AbortWithStatus(http.StatusForbidden)

		return
	}

	bannedPerson, errBan := u.bans.QueryOne(ctx, ban.QueryOpts{TargetID: steamID, EvadeOk: true})
	if errBan != nil && !errors.Is(errBan, ban.ErrBanDoesNotExist) {
		slog.Error("Failed to fetch authed user ban", slog.String("error", errBan.Error()))
	}

	profile := personDomain.Core{
		SteamID:         loggedInPerson.SteamID,
		PermissionLevel: loggedInPerson.PermissionLevel,
		DiscordID:       loggedInPerson.DiscordID,
		PatreonID:       loggedInPerson.PatreonID,
		Name:            loggedInPerson.PersonaName,
		Avatarhash:      loggedInPerson.AvatarHash,
		BanID:           bannedPerson.BanID,
	}

	ctx.Set(CtxKeyUserProfile, profile)

	if u.sentryDSN != "" {
		if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetUser(sentry.User{
					ID:        steamID.String(),
					IPAddress: ctx.ClientIP(),
					Username:  loggedInPerson.PersonaName,
				})
			})
		}
	}
}
