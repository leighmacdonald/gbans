package app

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/yohcop/openid-go"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// noOpDiscoveryCache implements the DiscoveryCache interface and doesn't cache anything.
type noOpDiscoveryCache struct{}

// Put is a no op.
func (n *noOpDiscoveryCache) Put(_ string, _ openid.DiscoveredInfo) {}

// Get always returns nil.
func (n *noOpDiscoveryCache) Get(_ string) openid.DiscoveredInfo {
	return nil
}

var nonceStore = openid.NewSimpleNonceStore()
var discoveryCache = &noOpDiscoveryCache{}

const testToken = "test-token"

func (web *web) authServerMiddleWare(database store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		tp := strings.SplitN(authHeader, " ", 2)
		if authHeader != "" && len(tp) == 2 && tp[0] == "Bearer" {
			token := tp[1]
			claims := &serverAuthClaims{}
			parsedToken, errParseClaims := jwt.ParseWithClaims(token, claims, getTokenKey)
			if errParseClaims != nil {
				if errParseClaims == jwt.ErrSignatureInvalid {
					log.Error("jwt signature invalid!")
					ctx.AbortWithStatus(http.StatusUnauthorized)
					return
				}
				ctx.AbortWithStatus(http.StatusUnauthorized)
				//log.WithFields(log.Fields{"claim": token}).Errorf("Failed to parse jwt claims: %s", errParseClaims)
				return
			}
			if !parsedToken.Valid {
				ctx.AbortWithStatus(http.StatusUnauthorized)
				log.Error("Invalid jwt token parsed")
				return
			}
			if claims.ServerId <= 0 {
				ctx.AbortWithStatus(http.StatusUnauthorized)
				log.Errorf("Invalid jwt claim ServerId!")
				return
			}
			var server model.Server
			if errGetPerson := database.GetServer(ctx, claims.ServerId, &server); errGetPerson != nil || server.ServerID <= 0 {
				log.Errorf("Failed to load persons session user: %v", errGetPerson)
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}
		}
		ctx.Next()
	}
}

func (web *web) onGetLogout() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// TODO Logout key / mark as invalid manually
		log.WithField("fn", "onGetLogout").Warnf("Unimplemented")
		ctx.Redirect(http.StatusTemporaryRedirect, "/")
	}
}

func (web *web) onOpenIDCallback(database store.Store) gin.HandlerFunc {
	oidRx := regexp.MustCompile(`^https://steamcommunity\.com/openid/id/(\d+)$`)
	return func(ctx *gin.Context) {
		referralUrl, found := ctx.GetQuery("return_url")
		if !found {
			referralUrl = "/"
		}
		var idStr string
		fullURL := config.General.ExternalUrl + ctx.Request.URL.String()
		if config.Debug.SkipOpenIDValidation {
			// Pull the sid out of the query without doing a signature check
			values, errParse := url.Parse(fullURL)
			if errParse != nil {
				log.Errorf("Failed to parse url: %query", errParse)
				ctx.Redirect(302, referralUrl)
				return
			}
			idStr = values.Query().Get("openid.identity")
		} else {
			id, errVerify := openid.Verify(fullURL, discoveryCache, nonceStore)
			if errVerify != nil {
				log.Errorf("Error verifying openid auth response: %v", errVerify)
				ctx.Redirect(302, referralUrl)
				return
			}
			idStr = id
		}
		match := oidRx.FindStringSubmatch(idStr)
		if match == nil || len(match) != 2 {
			ctx.Redirect(302, referralUrl)
			return
		}
		sid, errDecodeSid := steamid.SID64FromString(match[1])
		if errDecodeSid != nil {
			log.Errorf("Received invalid steamid: %query", errDecodeSid)
			ctx.Redirect(302, referralUrl)
			return
		}
		person := model.NewPerson(sid)
		if errGetProfile := getOrCreateProfileBySteamID(ctx, database, sid, "", &person); errGetProfile != nil {
			log.Errorf("Failed to fetch user profile: %query", errGetProfile)
			ctx.Redirect(302, referralUrl)
			return
		}
		accessToken, errJWT := newUserJWT(sid)
		if errJWT != nil {
			log.Errorf("Failed to create new access token: %q", errJWT)
			ctx.Redirect(302, referralUrl)
			return
		}
		parsedUrl, errParse := url.Parse("/login/success")
		if errParse != nil {
			ctx.Redirect(302, referralUrl)
			return
		}
		ipAddr := net.ParseIP(ctx.ClientIP())
		refreshToken := model.NewPersonAuth(sid, ipAddr)
		if errAuth := database.GetPersonAuth(ctx, sid, ipAddr, &refreshToken); errAuth != nil {
			if !errors.Is(errAuth, store.ErrNoResult) {
				log.Errorf("Failed to fetch refresh token: %q", errAuth)
				ctx.Redirect(302, referralUrl)
				return
			}
			if createErr := database.SavePersonAuth(ctx, &refreshToken); createErr != nil {
				log.Errorf("Failed to create new refresh token: %q", errAuth)
				ctx.Redirect(302, referralUrl)
				return
			}
			return
		}
		query := parsedUrl.Query()
		query.Set("refresh", refreshToken.RefreshToken)
		query.Set("token", accessToken)
		//query.Set("permission_level", fmt.Sprintf("%d", person.PermissionLevel))
		query.Set("next_url", referralUrl)
		parsedUrl.RawQuery = query.Encode()
		ctx.Redirect(302, parsedUrl.String())
		log.WithFields(log.Fields{
			"sid":              sid,
			"status":           "success",
			"permission_level": person.PermissionLevel,
		}).Infof("User login")
	}
}

func getTokenKey(_ *jwt.Token) (any, error) {
	return []byte(config.HTTP.CookieKey), nil
}

func (web *web) onTokenRefresh() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		tp := strings.SplitN(authHeader, " ", 2)
		var token string
		if authHeader != "" && len(tp) == 2 && tp[0] == "Bearer" {
			token = tp[1]
		}
		if token == "" {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		claims := &personAuthClaims{}
		parsedClaims, errParseClaims := jwt.ParseWithClaims(token, claims, getTokenKey)
		if errParseClaims != nil {
			if errParseClaims == jwt.ErrSignatureInvalid {
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if !parsedClaims.Valid {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// Don't reissue too often
		if time.Since(time.Unix(claims.IssuedAt, 0)) < 30*time.Second {
			ctx.AbortWithStatus(http.StatusTooEarly)
			return
		}
		// Now, create a new token for the current user, with a renewed expiration time
		newToken, errJWT := newUserJWT(steamid.SID64(claims.SteamID))
		if errJWT != nil || newToken == token {
			log.Errorf("Failed to renew JWT: %q", errJWT)
			responseErr(ctx, http.StatusUnauthorized, nil)
			return
		}
		responseOK(ctx, http.StatusOK, userToken{Token: newToken})
	}
}

type userToken struct {
	Token string `json:"token"`
}

type personAuthClaims struct {
	SteamID int64 `json:"steam_id"`
	jwt.StandardClaims
}

type serverAuthClaims struct {
	ServerId int `json:"server_id"`
	jwt.StandardClaims
}

const authTokenLifetimeDuration = time.Hour * 24

func newUserJWT(steamID steamid.SID64) (string, error) {
	t0 := config.Now()
	claims := &personAuthClaims{
		SteamID: steamID.Int64(),
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: t0.Add(authTokenLifetimeDuration).Unix(),
			IssuedAt:  t0.Unix(),
		},
	}
	tokenWithClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, errSigned := tokenWithClaims.SignedString([]byte(config.HTTP.CookieKey))
	if errSigned != nil {
		return "", errors.Wrap(errSigned, "Failed create signed string")
	}
	return signedToken, nil
}

func newServerJWT(serverId int) (string, error) {
	t0 := config.Now()
	claims := &serverAuthClaims{
		ServerId: serverId,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: t0.Add(authTokenLifetimeDuration).Unix(),
			IssuedAt:  t0.Unix(),
		},
	}
	tokenWithClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, errSigned := tokenWithClaims.SignedString([]byte(config.HTTP.CookieKey))
	if errSigned != nil {
		return "", errors.Wrap(errSigned, "Failed create signed string")
	}
	return signedToken, nil
}

func authMiddleware(database store.Store, level model.Privilege) gin.HandlerFunc {
	type header struct {
		Authorization string `header:"Authorization"`
	}
	return func(ctx *gin.Context) {
		var token string
		if ctx.FullPath() == "/ws/quickplay" {
			token = ctx.Query("token")
		} else {
			hdr := header{}
			if errBind := ctx.ShouldBindHeader(&hdr); errBind != nil {
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
			pcs := strings.Split(hdr.Authorization, " ")
			if len(pcs) != 2 && level >= model.PUser {
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
			token = pcs[1]
		}
		if level >= model.PUser {
			sid, errFromToken := sid64FromJWTToken(token)
			if errFromToken != nil {
				log.Errorf("Failed to load persons session user: %v", errFromToken)
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
			loggedInPerson := model.NewPerson(sid)
			if errGetPerson := database.GetPersonBySteamID(ctx, sid, &loggedInPerson); errGetPerson != nil {
				log.Errorf("Failed to load persons session user: %v", errGetPerson)
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
			if level > loggedInPerson.PermissionLevel {
				ctx.AbortWithStatus(http.StatusForbidden)
				return
			}
			bp := model.NewBannedPerson()
			if errBan := database.GetBanBySteamID(ctx, sid, &bp, false); errBan != nil {
				if !errors.Is(errBan, store.ErrNoResult) {
					log.Errorf("Failed to fetch authed user ban: %v", errBan)
				}
			}
			profile := model.UserProfile{
				SteamID:         loggedInPerson.SteamID,
				CreatedOn:       loggedInPerson.CreatedOn,
				UpdatedOn:       loggedInPerson.UpdatedOn,
				PermissionLevel: loggedInPerson.PermissionLevel,
				DiscordID:       loggedInPerson.DiscordID,
				Name:            loggedInPerson.PersonaName,
				Avatar:          loggedInPerson.Avatar,
				AvatarFull:      loggedInPerson.AvatarFull,
				Muted:           loggedInPerson.Muted,
				BanID:           bp.Ban.BanID,
			}
			ctx.Set(ctxKeyUserProfile, profile)
		}
		ctx.Next()
	}
}

func sid64FromJWTToken(token string) (steamid.SID64, error) {
	claims := &personAuthClaims{}
	tkn, errParseClaims := jwt.ParseWithClaims(token, claims, getTokenKey)
	if errParseClaims != nil {
		if errParseClaims == jwt.ErrSignatureInvalid {
			return 0, consts.ErrAuthentication
		}
		return 0, consts.ErrAuthentication
	}
	if !tkn.Valid {
		return 0, consts.ErrAuthentication
	}
	sid := steamid.SID64(claims.SteamID)
	if !sid.Valid() {
		log.Warnf("Invalid steamID")
		return 0, consts.ErrAuthentication
	}
	return sid, nil
}
