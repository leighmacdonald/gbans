package app

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
	log "github.com/sirupsen/logrus"
	"github.com/yohcop/openid-go"
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

func (web *web) authMiddleWare(database store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		person := model.NewPerson(0)
		authHeader := ctx.GetHeader("Authorization")
		tp := strings.SplitN(authHeader, " ", 2)
		if authHeader != "" && len(tp) == 2 && tp[0] == "Bearer" {
			token := tp[1]
			if config.General.Mode == "test" && token == testToken {
				loggedInPerson := model.NewPerson(config.General.Owner)
				if errGetPerson := database.GetOrCreatePersonBySteamID(ctx, config.General.Owner, &loggedInPerson); errGetPerson != nil {
					log.Errorf("Failed to load persons session user: %v", errGetPerson)
					ctx.AbortWithStatus(http.StatusForbidden)
					return
				}
				person = loggedInPerson
			} else {
				claims := &authClaims{}
				parsedToken, errParseClaims := jwt.ParseWithClaims(token, claims, getTokenKey)
				if errParseClaims != nil {
					if errParseClaims == jwt.ErrSignatureInvalid {
						ctx.AbortWithStatus(http.StatusForbidden)
						return
					}
					ctx.AbortWithStatus(http.StatusForbidden)
					return
				}
				if !parsedToken.Valid {
					ctx.AbortWithStatus(http.StatusForbidden)
					return
				}
				if !steamid.SID64(claims.SteamID).Valid() {
					ctx.AbortWithStatus(http.StatusForbidden)
					log.Warnf("Invalid steamID")
					return
				}

				loggedInPerson := model.NewPerson(steamid.SID64(claims.SteamID))
				if errGetPerson := database.GetPersonBySteamID(ctx, steamid.SID64(claims.SteamID), &loggedInPerson); errGetPerson != nil {
					log.Errorf("Failed to load persons session user: %v", errGetPerson)
					ctx.AbortWithStatus(http.StatusForbidden)
					return
				}
				person = loggedInPerson
			}
		}
		ctx.Set("person", person)
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
		fullURL := config.HTTP.Domain + ctx.Request.URL.String()
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
				log.Errorf("Error verifying openid auth response: %query", errVerify)
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
		webToken, errJWT := newJWT(sid)
		if errJWT != nil {
			log.Errorf("Failed to create new JWT: %query", errJWT)
			ctx.Redirect(302, referralUrl)
			return
		}
		parsedUrl, errParse := url.Parse("/login/success")
		if errParse != nil {
			ctx.Redirect(302, referralUrl)
			return
		}
		query := parsedUrl.Query()
		query.Set("token", webToken)
		query.Set("permission_level", fmt.Sprintf("%d", person.PermissionLevel))
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
		claims := &authClaims{}
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
		if time.Until(time.Unix(claims.ExpiresAt, 0)) > 30*time.Second {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		// Now, create a new token for the current user, with a renewed expiration time
		expirationTime := config.Now().Add(24 * time.Hour)
		claims.ExpiresAt = expirationTime.Unix()
		outToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, errSign := outToken.SignedString(config.HTTP.CookieKey)
		if errSign != nil {
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"token": tokenString})
	}
}

type authClaims struct {
	SteamID int64 `json:"steam_id"`
	Exp     int64 `json:"exp"`
	jwt.StandardClaims
}

func newJWT(steamID steamid.SID64) (string, error) {
	claims := &authClaims{
		SteamID:        steamID.Int64(),
		Exp:            config.Now().Add(time.Hour * 24).Unix(),
		StandardClaims: jwt.StandardClaims{},
	}
	tokenWithClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, errSigned := tokenWithClaims.SignedString([]byte(config.HTTP.CookieKey))
	if errSigned != nil {
		return "", errSigned
	}
	return signedToken, nil
}

func authMiddleware(database store.Store, level model.Privilege) gin.HandlerFunc {
	type header struct {
		Authorization string `header:"Authorization"`
	}
	return func(ctx *gin.Context) {
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
		if level >= model.PUser {
			sid, errFromToken := sid64FromJWTToken(pcs[1])
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
			profile := model.UserProfile{
				SteamID:         loggedInPerson.SteamID,
				CreatedOn:       loggedInPerson.CreatedOn,
				UpdatedOn:       loggedInPerson.UpdatedOn,
				PermissionLevel: loggedInPerson.PermissionLevel,
				DiscordID:       loggedInPerson.DiscordID,
				Name:            loggedInPerson.PersonaName,
				Avatar:          loggedInPerson.Avatar,
				AvatarFull:      loggedInPerson.AvatarFull,
			}
			bp := model.NewBannedPerson()
			if banErr := database.GetBanBySteamID(ctx, profile.SteamID, false, &bp); banErr == nil {
				profile.BanID = bp.Ban.BanID
			}
			ctx.Set(ctxKeyUserProfile, profile)
		}
		ctx.Next()
	}
}

func sid64FromJWTToken(token string) (steamid.SID64, error) {
	claims := &authClaims{}
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
