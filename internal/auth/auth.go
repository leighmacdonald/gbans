package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	TokenDuration         = time.Hour * 24 * 31
	FingerprintCookieName = "fingerprint"
)

type PersonAuth struct {
	PersonAuthID int64
	SteamID      steamid.SteamID
	IPAddr       net.IP
	AccessToken  string
	CreatedOn    time.Time
}

type tokenExchangeEntry struct {
	token     string
	expiresAt time.Time
}

type TokenExchange struct {
	mu    sync.Mutex
	codes map[string]tokenExchangeEntry
}

func NewTokenExchange(ctx context.Context) *TokenExchange {
	tx := &TokenExchange{codes: map[string]tokenExchangeEntry{}}
	go tx.cleanup(ctx)

	return tx
}

func (tx *TokenExchange) Create(token string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	code := hex.EncodeToString(b)

	tx.mu.Lock()
	tx.codes[code] = tokenExchangeEntry{token: token, expiresAt: time.Now().Add(time.Minute)}
	tx.mu.Unlock()

	return code
}

func (tx *TokenExchange) Exchange(code string) (string, bool) {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	entry, ok := tx.codes[code]
	if !ok {
		return "", false
	}

	delete(tx.codes, code)

	if time.Now().After(entry.expiresAt) {
		return "", false
	}

	return entry.token, true
}

func (tx *TokenExchange) cleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tx.mu.Lock()
			now := time.Now()
			for code, entry := range tx.codes {
				if now.After(entry.expiresAt) {
					delete(tx.codes, code)
				}
			}
			tx.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

type ServerAuthClaims struct {
	jwt.RegisteredClaims

	ServerID int
}

const CtxKeyUserProfile = httphelper.CtxKeyUserProfile

type Authentication struct {
	auth      Repository
	persons   *person.Persons
	bans      ban.Bans
	servers   *servers.Servers
	sentryDSN string
	siteName  string
	cookieKey string
	exchange  *TokenExchange
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

func (u *Authentication) StartExchange(ctx context.Context) {
	u.exchange = NewTokenExchange(ctx)
}

func (u *Authentication) CreateExchangeCode(token string) string {
	if u.exchange == nil {
		return ""
	}

	return u.exchange.Create(token)
}

func (u *Authentication) ExchangeCode(code string) (string, bool) {
	if u.exchange == nil {
		return "", false
	}

	return u.exchange.Exchange(code)
}

func RegisterExchangeHandler(mux *http.ServeMux, auth *Authentication) {
	mux.HandleFunc("POST /api/auth/exchange", func(res http.ResponseWriter, req *http.Request) {
		var body struct {
			Code string `json:"code"`
		}
		if errDecode := json.NewDecoder(req.Body).Decode(&body); errDecode != nil {
			httphelper.RespondJSON(res, http.StatusBadRequest, map[string]string{"error": "invalid request"})

			return
		}

		token, ok := auth.ExchangeCode(body.Code)
		if !ok {
			httphelper.RespondJSON(res, http.StatusNotFound, map[string]string{"error": "code not found or expired"})

			return
		}

		httphelper.RespondJSON(res, http.StatusOK, map[string]string{"token": token})
	})
}

func NewPersonAuth(steamID steamid.SteamID, ipAddr net.IP, refreshToken string) PersonAuth {
	return PersonAuth{
		SteamID:     steamID,
		IPAddr:      ipAddr,
		AccessToken: refreshToken,
		CreatedOn:   time.Now(),
	}
}

func (u *Authentication) SavePersonAuth(ctx context.Context, auth PersonAuth) error {
	return u.auth.SavePersonAuth(ctx, &auth)
}

func (u *Authentication) DeletePersonAuth(ctx context.Context, authID int64) error {
	return u.auth.DeletePersonAuth(ctx, authID)
}

func (u *Authentication) GetPersonAuthByRefreshToken(ctx context.Context, token string, auth *PersonAuth) error {
	return u.auth.GetPersonAuthByRefreshToken(ctx, token, auth)
}

func (u *Authentication) loginSID(ctx context.Context, res http.ResponseWriter, req *http.Request, level permission.Privilege, steamID steamid.SteamID) {
	loggedInPerson, errGetPerson := u.persons.BySteamID(ctx, steamID)
	if errGetPerson != nil {
		slog.Error("Failed to load person during auth", slog.String("error", errGetPerson.Error()))
		res.WriteHeader(http.StatusForbidden)

		return
	}
	if u.sentryDSN != "" {
		sentry.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetUser(sentry.User{
				ID:        loggedInPerson.SteamID.String(),
				IPAddress: req.RemoteAddr,
				Username:  loggedInPerson.PersonaName,
			})
		})
	}
	if level > loggedInPerson.PermissionLevel {
		res.WriteHeader(http.StatusForbidden)

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

	*req = *req.WithContext(context.WithValue(req.Context(), CtxKeyUserProfile, profile))

	if u.sentryDSN != "" {
		if hub := sentry.GetHubFromContext(ctx); hub != nil {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetUser(sentry.User{
					ID:        steamID.String(),
					IPAddress: req.RemoteAddr,
					Username:  loggedInPerson.PersonaName,
				})
			})
		}
	}
}
