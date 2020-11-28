package service

import (
	"context"
	"fmt"
	"github.com/gin-contrib/sessions"
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
)

// NoOpDiscoveryCache implements the DiscoveryCache interface and doesn't cache anything.
type NoOpDiscoveryCache struct{}

// Put is a no op.
func (n *NoOpDiscoveryCache) Put(id string, info openid.DiscoveredInfo) {}

// Get always returns nil.
func (n *NoOpDiscoveryCache) Get(id string) openid.DiscoveredInfo {
	return nil
}

var nonceStore = openid.NewSimpleNonceStore()
var discoveryCache = &NoOpDiscoveryCache{}

func authMiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		s := sessions.Default(c)
		guest := model.NewPerson()
		var p model.Person
		var err error
		v := s.Get("steam_id")
		if v != nil {
			p, err = GetPersonBySteamID(steamid.SID64(v.(int64)))
			if err != nil {
				log.Errorf("Failed to load persons session user: %v", err)
				p = guest
			}
		} else {
			p = guest
		}
		c.Set("person", p)
		c.Next()
	}
}

func onGetLogin() gin.HandlerFunc {
	const f = "https://steamcommunity.com/openid/login" +
		"?openid.ns=http://specs.openid.net/auth/2.0" +
		"&openid.mode=checkid_setup" +
		"&openid.return_to=%s/auth/callback?return_url=%s" +
		"&openid.realm=%s&openid.ns.sreg=http://openid.net/extensions/sreg/1.1" +
		"&openid.claimed_id=http://specs.openid.net/auth/2.0/identifier_select" +
		"&openid.identity=http://specs.openid.net/auth/2.0/identifier_select"
	return func(c *gin.Context) {
		fullURL, err := url.Parse(c.Request.Referer())
		if err != nil {
			c.Redirect(303, "/")
			return
		}
		ref := fullURL.Path
		u := fmt.Sprintf(f, config.HTTP.Domain, ref, config.HTTP.Domain)
		c.Redirect(303, u)
	}
}

func onGetLogout() gin.HandlerFunc {
	return func(c *gin.Context) {
		ses := sessions.Default(c)
		ses.Delete("steam_id")
		if err := ses.Save(); err != nil {
			log.Errorf("Failed to clear user session: %v", err)
		}
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
		p.PlayerSummary = s
		if err := SavePerson(&p); err != nil {
			log.Errorf("Failed to save person: %v", err)
			c.Redirect(302, ref)
			return
		}
		ses := sessions.Default(c)
		ses.Set("steam_id", sid.Int64())
		if err := ses.Save(); err != nil {
			log.Errorf("Failed to save person to session: %v", err)
			c.Redirect(302, ref)
			return
		}
		c.Redirect(302, ref)
	}
}
