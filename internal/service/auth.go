package service

import (
	"context"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/steamid/v2/extra"
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

func authMiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		p := model.NewPerson(0)
		ah := c.GetHeader("Authorization")
		tp := strings.SplitN(ah, " ", 2)
		if ah != "" && len(tp) == 2 && tp[0] == "Bearer" {
			token := tp[1]
			claims := &authClaims{}
			tkn, err := jwt.ParseWithClaims(token, claims, getTokenKey)
			if err != nil {
				if err == jwt.ErrSignatureInvalid {
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
			loggedInPerson, err := getPersonBySteamID(steamid.SID64(claims.SteamID))
			if err != nil {
				log.Errorf("Failed to load persons session user: %v", err)
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			p = loggedInPerson
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
		sum, err := extra.PlayerSummaries(context.Background(), []steamid.SID64{sid})
		if err != nil {
			log.Errorf("Failed to get player summary: %v", err)
			c.Redirect(302, ref)
			return
		}
		p, err := GetOrCreatePersonBySteamID(sid)
		if err != nil {
			log.Errorf("Failed to get person: %v", err)
			c.Redirect(302, ref)
			return
		}
		s := sum[0]
		p.SteamID = sid
		p.IPAddr = c.Request.RemoteAddr
		p.PlayerSummary = &s
		if err := SavePerson(p); err != nil {
			log.Errorf("Failed to save person: %v", err)
			c.Redirect(302, ref)
			return
		}
		u, err := url.Parse(routeRaw(string(routeLoginSuccess)))
		if err != nil {
			c.Redirect(302, ref)
			return
		}
		t, err := newJWT(sid)
		if err != nil {
			c.Redirect(302, ref)
			return
		}
		v := u.Query()
		v.Set("token", t)
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
func getTokenKey(token *jwt.Token) (interface{}, error) {
	return []byte(config.HTTP.CookieKey), nil
}
func onTokenRefresh() gin.HandlerFunc {
	return func(c *gin.Context) {
		// (BEGIN) The code uptil this point is the same as the first part of the `Welcome` route
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

		if time.Unix(claims.ExpiresAt, 0).Sub(time.Now()) > 30*time.Second {
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

func newJWT(steamID steamid.SID64) (string, error) {
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
