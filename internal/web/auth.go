package web

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/action"
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

func authMiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		p := model.NewPerson(0)
		ah := c.GetHeader("Authorization")
		tp := strings.SplitN(ah, " ", 2)
		if ah != "" && len(tp) == 2 && tp[0] == "Bearer" {
			token := tp[1]
			if config.General.Mode == "test" && token == testToken {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel()
				loggedInPerson, err2 := store.GetOrCreatePersonBySteamID(ctx, config.General.Owner)
				if err2 != nil {
					log.Errorf("Failed to load persons session user: %v", err2)
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
				p = loggedInPerson
			} else {
				claims := &authClaims{}
				tkn, errC := jwt.ParseWithClaims(token, claims, getTokenKey)
				if errC != nil {
					if errC == jwt.ErrSignatureInvalid {
						c.AbortWithStatus(http.StatusForbidden)
						return
					}
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
				if !tkn.Valid {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
				if !steamid.SID64(claims.SteamID).Valid() {
					c.AbortWithStatus(http.StatusForbidden)
					log.Warnf("Invalid steamID")
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel()
				loggedInPerson, err := store.GetPersonBySteamID(ctx, steamid.SID64(claims.SteamID))
				if err != nil {
					log.Errorf("Failed to load persons session user: %v", err)
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
				p = loggedInPerson
			}
		}
		c.Set("person", p)
		c.Next()
	}
}

func onGetLogout() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO Logout key / mark as invalid manually
		log.WithField("fn", "onGetLogout").Warnf("Unimplemented")
		c.Redirect(http.StatusTemporaryRedirect, "/")
	}
}

func onOpenIDCallback() gin.HandlerFunc {
	oidRx := regexp.MustCompile(`^https://steamcommunity\.com/openid/id/(\d+)$`)
	return func(c *gin.Context) {
		ref, found := c.GetQuery("return_url")
		if !found {
			ref = "/"
		}
		fullURL := config.HTTP.Domain + c.Request.URL.String()
		id, err := openid.Verify(fullURL, discoveryCache, nonceStore)
		if err != nil {
			log.Printf("Error verifying: %q\n", err)
			c.Redirect(302, ref)
			return
		}
		m := oidRx.FindStringSubmatch(id)
		if m == nil || len(m) != 2 {
			c.Redirect(302, ref)
			return
		}
		sid, err := steamid.SID64FromString(m[1])
		if err != nil {
			log.Errorf("Received invalid steamid: %v", err)
			c.Redirect(302, ref)
			return
		}
		act := action.NewGetOrCreatePersonByID(sid.String(), c.Request.RemoteAddr)
		res := <-act.Enqueue().Done()
		if res.Err != nil {
			log.Errorf("Failed to fetch user profile: %v", res.Err)
			c.Redirect(302, ref)
			return
		}
		p, ok := res.Value.(*model.Person)
		if !ok {
			log.Errorf("Failed to cast user profile")
			c.Redirect(302, ref)
			return
		}
		u, errParse := url.Parse("/login/success")
		if errParse != nil {
			c.Redirect(302, ref)
			return
		}
		t, errJWT := NewJWT(sid)
		if errJWT != nil {
			log.Errorf("Failed to create new JWT: %v", errJWT)
			c.Redirect(302, ref)
			return
		}
		v := u.Query()
		v.Set("token", t)
		v.Set("permission_level", fmt.Sprintf("%d", p.PermissionLevel))
		v.Set("next_url", ref)
		u.RawQuery = v.Encode()
		c.Redirect(302, u.String())
	}
}

func onLoginSuccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Data(200, gin.MIMEHTML, []byte(baseLayout))
	}
}

func getTokenKey(_ *jwt.Token) (interface{}, error) {
	return []byte(config.HTTP.CookieKey), nil
}

func onTokenRefresh() gin.HandlerFunc {
	return func(c *gin.Context) {
		ah := c.GetHeader("Authorization")
		tp := strings.SplitN(ah, " ", 2)
		var token string
		if ah != "" && len(tp) == 2 && tp[0] == "Bearer" {
			token = tp[1]
		}
		if token == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		claims := &authClaims{}
		tkn, err := jwt.ParseWithClaims(token, claims, getTokenKey)
		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if !tkn.Valid {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if time.Until(time.Unix(claims.ExpiresAt, 0)) > 30*time.Second {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Now, create a new token for the current use, with a renewed expiration time
		expirationTime := config.Now().Add(24 * time.Hour)
		claims.ExpiresAt = expirationTime.Unix()
		outToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err2 := outToken.SignedString(config.HTTP.CookieKey)
		if err2 != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": tokenString})
	}
}

type authClaims struct {
	SteamID int64 `json:"steam_id"`
	Exp     int64 `json:"exp"`
	jwt.StandardClaims
}

func NewJWT(steamID steamid.SID64) (string, error) {
	claims := &authClaims{
		SteamID:        steamID.Int64(),
		Exp:            config.Now().Add(time.Hour * 24).Unix(),
		StandardClaims: jwt.StandardClaims{},
	}
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := at.SignedString([]byte(config.HTTP.CookieKey))
	if err != nil {
		return "", err
	}
	return token, nil
}

func authMiddleware(level model.Privilege) gin.HandlerFunc {
	type header struct {
		Authorization string `header:"Authorization"`
	}
	return func(c *gin.Context) {
		hdr := header{}
		if err := c.ShouldBindHeader(&hdr); err != nil {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		pcs := strings.Split(hdr.Authorization, " ")
		if len(pcs) != 2 && level > model.PGuest {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		if level > model.PGuest {
			sid, err := sid64FromJWTToken(pcs[1])
			if err != nil {
				log.Errorf("Failed to load persons session user: %v", err)
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			cx, cancel := context.WithTimeout(context.Background(), time.Second*6)
			defer cancel()
			loggedInPerson, err := store.GetPersonBySteamID(cx, sid)
			if err != nil {
				log.Errorf("Failed to load persons session user: %v", err)
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			if level > loggedInPerson.PermissionLevel {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Set("person", loggedInPerson)
		}
		c.Next()
	}
}

func sid64FromJWTToken(token string) (steamid.SID64, error) {
	claims := &authClaims{}
	tkn, errC := jwt.ParseWithClaims(token, claims, getTokenKey)
	if errC != nil {
		if errC == jwt.ErrSignatureInvalid {
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
