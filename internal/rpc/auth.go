package rpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"connectrpc.com/authn"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

const (
	TokenDuration         = time.Hour * 24 * 31
	FingerprintCookieName = "fingerprint"
)

var (
	ErrCreateToken = errors.New("failed to create new Access token")
	ErrSignToken   = errors.New("failed create signed string")
)

type UserClaimProvider interface {
	GetAvatar() person.Avatar
	GetSteamID() steamid.SteamID
	GetPrivilege() permission.Privilege
	GetName() string
}

type RouteAuthFn = func(ctx context.Context, req *http.Request, user UserInfo) bool

func WithMinPermissions(permission permission.Privilege) RouteAuthFn {
	return func(ctx context.Context, req *http.Request, user UserInfo) bool {
		return user.HasPermission(permission)
	}
}

type Claims struct {
	jwt.RegisteredClaims

	// user context to prevent side-jacking
	// https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html#token-sidejacking
	Fingerprint string               `json:"fingerprint"`
	Privilege   permission.Privilege `json:"privilege"`
	SteamID     string               `json:"steamID"`
	AvatarHash  person.Avatar        `json:"avatarHash"`
	Name        string               `json:"name"`
}

type Middleware struct {
	sync.RWMutex

	siteName  string
	cookie    string
	allowList map[string]RouteAuthFn
}

func NewMiddleware(siteName string, cookie string) *Middleware {
	return &Middleware{
		RWMutex:   sync.RWMutex{},
		siteName:  siteName,
		cookie:    cookie,
		allowList: map[string]RouteAuthFn{},
	}
}

func (m *Middleware) AuthedRoute(procedure string, authFunc RouteAuthFn) {
	m.Lock()
	m.allowList[procedure] = authFunc
	m.Unlock()
}

func (m *Middleware) findProcedure(url *url.URL) (RouteAuthFn, bool) {
	procedure, found := authn.InferProcedure(url)
	if !found {
		return nil, false
	}

	m.RLock()
	defer m.RUnlock()

	authFn, ok := m.allowList[procedure]
	if !ok {
		return nil, false
	}

	return authFn, true
}

func (m *Middleware) Authenticate(ctx context.Context, req *http.Request) (any, error) {
	authFn, required := m.findProcedure(req.URL)
	if !required {
		return nil, nil
	}

	var info UserInfo

	claims, errToken := m.claimsFromRequest(req)
	if errToken != nil {
		return info, errToken
	}

	sid := steamid.New(claims.Subject)
	if !sid.Valid() {
		return info, authn.Errorf("invalid authorization")
	}

	info.SteamID = sid
	info.Privilege = claims.Privilege
	info.AvatarHash = claims.AvatarHash
	info.Name = claims.Name

	if !authFn(ctx, req, info) {
		return info, authn.Errorf("unauthorized")
	}

	return info, nil
}

func (m *Middleware) claimsFromRequest(req *http.Request) (*Claims, error) {
	fingerprint, errFP := m.fingerprintFromRequest(req)
	if errFP != nil {
		return nil, errFP
	}

	token, ok := authn.BearerToken(req)
	if !ok {
		// TODO Make sure procedure is allowed
		return nil, authn.Errorf("invalid authorization")
	}

	claims := Claims{}
	tkn, errParseClaims := jwt.ParseWithClaims(token, &claims, m.makeGetTokenKey())
	if errParseClaims != nil {
		if errors.Is(errParseClaims, jwt.ErrSignatureInvalid) {
			return nil, authn.Errorf("invalid authorization")
		}

		if errors.Is(errParseClaims, jwt.ErrTokenExpired) {
			return nil, authn.Errorf("expired authorization")
		}

		return nil, authn.Errorf("invalid authorization")
	}

	if !tkn.Valid {
		return nil, authn.Errorf("invalid token")
	}

	if claims.Fingerprint != m.fingerprintHash(fingerprint) {
		slog.Error("Invalid cookie fingerprint, token rejected")

		return nil, authn.Errorf("invalid token")
	}

	return &claims, nil
}

func (m *Middleware) fingerprintHash(fingerprint string) string {
	hasher := sha256.New()

	return hex.EncodeToString(hasher.Sum([]byte(fingerprint)))
}

func (m *Middleware) fingerprintFromRequest(req *http.Request) (string, error) {
	fp, errFP := req.Cookie(FingerprintCookieName)
	if errFP != nil {
		return "", authn.Errorf("invalid fingerprint cookie")
	}

	fingerprint := fp.String()
	if fingerprint == "" {
		return "", authn.Errorf("empty fingerprint")
	}

	if strings.HasPrefix(fingerprint, "fingerprint=") {
		fingerprint = strings.TrimPrefix(fingerprint, "fingerprint=")
	}

	return fingerprint, nil
}

func (m *Middleware) makeGetTokenKey() func(_ *jwt.Token) (any, error) {
	return func(_ *jwt.Token) (any, error) {
		return []byte(m.cookie), nil
	}
}

func (m *Middleware) newUserToken(user UserClaimProvider, fingerPrint string, validDuration time.Duration) (string, error) {
	nowTime := time.Now()
	sid := user.GetSteamID()
	claims := Claims{
		Fingerprint: m.fingerprintHash(fingerPrint),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.siteName,
			Subject:   sid.String(),
			ExpiresAt: jwt.NewNumericDate(nowTime.Add(validDuration)),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			NotBefore: jwt.NewNumericDate(nowTime),
		},
		SteamID:    sid.String(),
		Privilege:  user.GetPrivilege(),
		AvatarHash: user.GetAvatar(),
	}
	tokenWithClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, errSigned := tokenWithClaims.SignedString([]byte(m.cookie))

	if errSigned != nil {
		return "", errors.Join(errSigned, ErrSignToken)
	}

	return signedToken, nil
}

// MakeToken generates new jwt auth tokens
// fingerprint is a random string used to prevent side-jacking.
func (m *Middleware) MakeToken(user person.BaseUser) (string, string, error) {
	fingerprint := stringutil.SecureRandomString(40)
	accessToken, errAccess := m.newUserToken(user, fingerprint, TokenDuration)
	if errAccess != nil {
		return "", "", errors.Join(errAccess, ErrCreateToken)
	}

	// FIXME save auth for revocation
	//ipAddr := net.ParseIP(ctx.ClientIP())
	//if ipAddr == nil {
	//	return UserTokens{}, ErrClientIP
	//}
	//
	//personAuth := NewPersonAuth(sid, ipAddr, accessToken)
	//
	//if saveErr := u.auth.SavePersonAuth(ctx, &personAuth); saveErr != nil {
	//	return UserTokens{}, errors.Join(saveErr, ErrSaveToken)
	//}

	return accessToken, fingerprint, nil
}
