package rpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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

// TokenDuration is the default validity period for user authentication tokens.
// Tokens expire after approximately one month.
const (
	TokenDuration         = time.Hour * 24 * 31
	FingerprintCookieName = "fingerprint"
)

var (
	ErrCreateToken = errors.New("failed to create new Access token")
	ErrSignToken   = errors.New("failed create signed string")
)

// UserClaimProvider defines the required fields to extract user claims for JWT creation.
type UserClaimProvider interface {
	GetAvatar() person.Avatar
	GetSteamID() steamid.SteamID
	GetPrivilege() permission.Privilege
	GetName() string
}

// UserRouteAuthFn is a function type that determines if a user has permission to access a given RPC procedure.
type UserRouteAuthFn = func(ctx context.Context, req *http.Request, user UserInfo) bool

// ServerRouteAuthFn is a function type that determines if a server has permission to access a given RPC procedure.
type ServerRouteAuthFn = func(ctx context.Context, req *http.Request, server ServerInfo) bool

// WithServer returns a ServerRouteAuthFn that requires a valid server ID (> 0) to authorize the request.
func WithServer() ServerRouteAuthFn {
	return func(ctx context.Context, req *http.Request, server ServerInfo) bool {
		return server.ServerID > 0
	}
}

// WithMinPermissions returns a UserRouteAuthFn that checks if the user has at least the specified privilege level.
func WithMinPermissions(permission permission.Privilege) UserRouteAuthFn {
	return func(ctx context.Context, req *http.Request, user UserInfo) bool {
		return user.HasPermission(permission)
	}
}

type userClaims struct {
	jwt.RegisteredClaims

	// user context to prevent side-jacking
	// https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html#token-sidejacking
	Fingerprint string               `json:"fingerprint"`
	Privilege   permission.Privilege `json:"privilege"`
	SteamID     string               `json:"steamID"`
	AvatarHash  person.Avatar        `json:"avatarHash"`
	Name        string               `json:"name"`
}

type serverClaims struct {
	// ID = server id
	// Subject = Name
	jwt.RegisteredClaims
}

// Middleware implements connectrpc.com/authn.Authenticator for user and server authentication.
// It uses JWT tokens to authorize RPC procedures and maintains allowlists mapping procedures
// to their required authentication checks. Register user routes with UserRoute() and server
// routes with ServerRoute(). The Authenticate method is called automatically by the authn middleware.
type Middleware struct {
	sync.RWMutex

	siteName        string
	cookie          string
	userAllowList   map[string]UserRouteAuthFn
	serverAllowList map[string]ServerRouteAuthFn
}

// NewMiddleware creates a new authentication middleware for the given site name and cookie secret.
// The cookie secret is used as the HMAC key for signing and verifying JWT tokens.
func NewMiddleware(siteName string, cookie string) *Middleware {
	return &Middleware{
		RWMutex:         sync.RWMutex{},
		siteName:        siteName,
		cookie:          cookie,
		userAllowList:   map[string]UserRouteAuthFn{},
		serverAllowList: map[string]ServerRouteAuthFn{},
	}
}

// UserRoute registers an authentication check for a user-facing RPC procedure.
// The authFunc is called during Authenticate to determine if the requesting user
// has permission to access the procedure. Thread-safe.
func (m *Middleware) UserRoute(procedure string, authFunc UserRouteAuthFn) {
	m.Lock()
	m.userAllowList[procedure] = authFunc
	m.Unlock()
}

// ServerRoute registers an authentication check for a server-facing RPC procedure.
// The authFunc is called during Authenticate to determine if the requesting server
// has permission to access the procedure. Thread-safe.
func (m *Middleware) ServerRoute(procedure string, authFunc ServerRouteAuthFn) {
	m.Lock()
	m.serverAllowList[procedure] = authFunc
	m.Unlock()
}

func (m *Middleware) findProcedure(url *url.URL) (string, bool, bool) {
	procedure, found := authn.InferProcedure(url)
	if !found {
		return "", false, false
	}

	return procedure, strings.Contains(procedure, "Plugin"), true
}

// Authenticate extracts and validates authentication information from an incoming RPC request.
// It determines whether the target procedure is server-facing or user-facing and delegates
// to the appropriate auth handler. Returns nil for procedures not registered in the allowlists,
// allowing unauthenticated access. Implements authn.Authenticator.
func (m *Middleware) Authenticate(ctx context.Context, req *http.Request) (any, error) {
	procedure, isServer, found := m.findProcedure(req.URL)
	if !found {
		return nil, nil
	}

	if isServer {
		return m.authServer(ctx, req, procedure)
	}

	return m.authUser(ctx, req, procedure)
}

func (m *Middleware) authServer(ctx context.Context, req *http.Request, procedure string) (ServerInfo, error) {
	var info ServerInfo

	authFn, found := m.serverAllowList[procedure]
	if !found {
		return info, nil
	}

	claims, errToken := m.serverClaimsFromRequest(req)
	if errToken != nil {
		return info, errToken
	}

	serverId, err := strconv.ParseInt(claims.ID, 10, 32)
	if err != nil {
		return info, errors.Join(err, ErrBadRequest)
	}

	info.ServerID = int32(serverId)
	info.ServerName = claims.Subject

	if !authFn(ctx, req, info) {
		return info, authn.Errorf("unauthorized")
	}

	return info, nil
}

func (m *Middleware) authUser(ctx context.Context, req *http.Request, procedure string) (UserInfo, error) {
	m.RLock()
	defer m.RUnlock()

	var info UserInfo

	authFn, found := m.userAllowList[procedure]
	if !found {
		return info, nil
	}

	claims, errToken := m.userClaimsFromRequest(req)
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

func (m *Middleware) serverClaimsFromRequest(req *http.Request) (*serverClaims, error) {
	token, ok := authn.BearerToken(req)
	if !ok {
		// TODO Make sure procedure is allowed
		return nil, authn.Errorf("invalid authorization")
	}

	claims := serverClaims{}
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

	return &claims, nil
}

func (m *Middleware) userClaimsFromRequest(req *http.Request) (*userClaims, error) {
	_, errFP := m.fingerprintFromRequest(req)
	if errFP != nil {
		return nil, errFP
	}

	token, ok := authn.BearerToken(req)
	if !ok {
		// TODO Make sure procedure is allowed
		return nil, authn.Errorf("invalid authorization")
	}

	claims := userClaims{}
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

	// if claims.Fingerprint != m.fingerprintHash(fingerprint) {
	// 	slog.Error("Invalid cookie fingerprint, token rejected")

	// 	return nil, authn.Errorf("invalid token")
	// }

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

	return strings.TrimPrefix(fingerprint, "fingerprint="), nil
}

func (m *Middleware) makeGetTokenKey() func(_ *jwt.Token) (any, error) {
	return func(_ *jwt.Token) (any, error) {
		return []byte(m.cookie), nil
	}
}

// NewServerTokenGenerator returns a function that generates JWT tokens for server-to-server authentication.
// The returned function takes a server ID and name, and produces a signed token valid for 7 days.
func NewServerTokenGenerator(siteName string, cookie []byte) func(serverID int32, serverName string) (string, error) {
	return func(serverID int32, serverName string) (string, error) {
		nowTime := time.Now()
		claims := serverClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				ID:        fmt.Sprintf("%d", serverID),
				Issuer:    siteName,
				Subject:   serverName,
				ExpiresAt: jwt.NewNumericDate(nowTime.AddDate(0, 0, 7)),
				IssuedAt:  jwt.NewNumericDate(nowTime),
				NotBefore: jwt.NewNumericDate(nowTime),
			},
		}

		tokenWithClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signedToken, errSigned := tokenWithClaims.SignedString(cookie)

		if errSigned != nil {
			return "", errors.Join(errSigned, ErrSignToken)
		}

		return signedToken, nil
	}
}

func (m *Middleware) newUserToken(user UserClaimProvider, fingerPrint string, validDuration time.Duration) (string, error) {
	nowTime := time.Now()
	sid := user.GetSteamID()
	claims := userClaims{
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

// MakeUserToken generates a new JWT access token and fingerprint cookie value for the given user.
// The returned fingerprint should be set as an HTTP-only cookie to help prevent token sidejacking.
// The access token is valid for TokenDuration (~31 days).
// Returns (accessToken, fingerprint, error).
func (m *Middleware) MakeUserToken(user person.BaseUser) (string, string, error) {
	fingerprint := stringutil.SecureRandomString(40)
	accessToken, errAccess := m.newUserToken(user, fingerprint, TokenDuration)
	if errAccess != nil {
		return "", "", errors.Join(errAccess, ErrCreateToken)
	}

	// FIXME save auth for revocation
	// ipAddr := net.ParseIP(ctx.ClientIP())
	// if ipAddr == nil {
	// 	return UserTokens{}, ErrClientIP
	// }
	//
	// personAuth := NewPersonAuth(sid, ipAddr, accessToken)
	//
	// if saveErr := u.auth.SavePersonAuth(ctx, &personAuth); saveErr != nil {
	// 	return UserTokens{}, errors.Join(saveErr, ErrSaveToken)
	// }

	return accessToken, fingerprint, nil
}

// MakeServerToken generates a new JWT access token and fingerprint cookie value for the given user.
// The returned fingerprint should be set as an HTTP-only cookie to help prevent token sidejacking.
// Returns (accessToken, fingerprint, error).
func (m *Middleware) MakeServerToken(user person.BaseUser) (string, string, error) {
	fingerprint := stringutil.SecureRandomString(40)
	accessToken, errAccess := m.newUserToken(user, fingerprint, TokenDuration)
	if errAccess != nil {
		return "", "", errors.Join(errAccess, ErrCreateToken)
	}

	return accessToken, fingerprint, nil
}
