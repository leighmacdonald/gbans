package service

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/yohcop/openid-go"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
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
		if ah != "" && len(tp) == 2 && tp[0] != "Bearer" {
			token := tp[1]
			steamID, err2 := tokenMetadata(token)
			if err2 != nil {
				c.AbortWithStatus(http.StatusForbidden)
				log.Warnf("Invalid JWT received: %s", err2.Error())
				return
			}
			if !steamID.Valid() {
				c.AbortWithStatus(http.StatusForbidden)
				log.Warnf("Invalid steamID")
				return
			}
			loggedInPerson, err := getPersonBySteamID(steamID)
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
		// TODO Logout key
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

func newJWT(steamID steamid.SID64) (string, error) {
	atClaims := jwt.MapClaims{}
	atClaims["steam_id"] = steamID.Int64()
	atClaims["exp"] = time.Now().Add(time.Hour * 24).Unix()
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	token, err := at.SignedString([]byte(os.Getenv(config.HTTP.CookieKey)))
	if err != nil {
		return "", err
	}
	return token, nil
}

func verifyJWT(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		//Make sure that the token method conform to "SigningMethodHMAC"
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.HTTP.CookieKey), nil
	})
	if err != nil {
		return nil, err
	}
	if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
		return nil, err
	}
	if token.Valid {
		return token, nil
	} else if ve, ok := err.(*jwt.ValidationError); ok {
		if ve.Errors&jwt.ValidationErrorMalformed != 0 {
			return nil, errors.Wrap(ve, "Not a valid token")
		} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
			// Token is either expired or not active yet
			return nil, errors.Wrap(ve, "Token expired")
		} else {
			return nil, errors.Wrap(ve, "Unknown error handling token")
		}
	} else {
		return nil, errors.Wrap(ve, "Invalid token")
	}
}

func tokenMetadata(tokenString string) (steamid.SID64, error) {
	token, err := verifyJWT(tokenString)
	if err != nil {
		return 0, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		steamIdStr, ok := claims["steam_id"].(string)
		if !ok {
			return 0, err
		}
		steamId, err := steamid.SID64FromString(steamIdStr)
		if err != nil {
			return 0, err
		}
		expInt, err2 := strconv.ParseInt(fmt.Sprintf("%d", claims["exp"]), 10, 64)
		if err2 != nil {
			return 0, err2
		}
		expTIme := time.Unix(expInt, 0)
		if config.Now().After(expTIme) {
			return 0, errDuplicate
		}
		return steamId, nil
	}
	return 0, err
}
