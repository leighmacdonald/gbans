package service

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func onIndex() gin.HandlerFunc {
	return func(c *gin.Context) {
		render(c, "home", defaultArgs(c))
	}
}

func onGetServers() gin.HandlerFunc {
	return func(c *gin.Context) {
		serverStateMu.RLock()
		state := serverState
		serverStateMu.RUnlock()
		a := defaultArgs(c)
		a.V["servers"] = state
		render(c, "servers", a)
	}
}

func onGetBans() gin.HandlerFunc {
	return func(c *gin.Context) {
		render(c, "bans", defaultArgs(c))
	}
}

func onGetAppeal() gin.HandlerFunc {
	return func(c *gin.Context) {
		usr := currentPerson(c)
		ban, err := GetBan(usr.SteamID)
		if err != nil {
			if errors.Is(err, ErrNoResult) {
				flash(c, lError, "No Ban Found", "Please login with the account in question")
				c.Redirect(http.StatusTemporaryRedirect, c.Request.Referer())
				return
			} else {
				log.Errorf("Failed to lookup ban: %v", err)
				c.String(http.StatusInternalServerError, "oops")
				return
			}
		}
		args := defaultArgs(c)
		args.V["ban"] = ban
		render(c, "appeal", args)
	}
}

func onAPIGetServers() gin.HandlerFunc {
	return func(c *gin.Context) {
		servers, err := GetServers()
		if err != nil {
			log.Errorf("Failed to fetch servers: %s", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, servers)
	}
}

func onAPIPostAppeal() gin.HandlerFunc {
	type req struct {
		Email      string `json:"email"`
		AppealText string `json:"appeal_text"`
	}
	return func(c *gin.Context) {
		var app req
		if err := c.BindJSON(&app); err != nil {
			log.Errorf("Received malformed appeal req: %v", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		c.JSON(http.StatusOK, gin.H{})
	}
}

func onAdminFilteredWords() gin.HandlerFunc {
	return func(c *gin.Context) {
		words, err := GetFilteredWords()
		if err != nil {
			log.Errorf("Failed to load filtered word sets from db: %v", err)
			c.Redirect(http.StatusTemporaryRedirect, c.Request.Referer())
			return
		}
		args := defaultArgs(c)
		args.V["words"] = words
		render(c, "admin_filtered_words", args)
	}
}

func onGetProfileSettings() gin.HandlerFunc {
	return func(c *gin.Context) {
		render(c, "profile_settings", defaultArgs(c))
	}
}

func onGetAdminImport() gin.HandlerFunc {
	return func(c *gin.Context) {
		render(c, "admin_import", defaultArgs(c))
	}
}

func onGetAdminServers() gin.HandlerFunc {
	return func(c *gin.Context) {
		render(c, "admin_servers", defaultArgs(c))
	}
}

func onGetAdminPeople() gin.HandlerFunc {
	return func(c *gin.Context) {
		render(c, "admin_people", defaultArgs(c))
	}
}

func onAPIPostReport() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{})
	}
}

func onAPIPostBan() gin.HandlerFunc {
	type req struct {
		SteamID    steamid.SID64 `json:"steam_id"`
		AuthorID   steamid.SID64 `json:"author_id"`
		Duration   string        `json:"duration"`
		BanType    model.BanType `json:"ban_type"`
		Reason     model.Reason  `json:"reason"`
		ReasonText string        `json:"reason_text"`
	}

	return func(c *gin.Context) {
		var r req
		if err := c.BindJSON(&r); err != nil {
			c.JSON(http.StatusBadRequest, StatusResponse{
				Success: false,
				Message: "Failed to perform ban",
			})
			return
		}
		duration, err := time.ParseDuration(r.Duration)
		if err != nil {
			c.JSON(http.StatusNotAcceptable, StatusResponse{
				Success: false,
				Message: `Invalid duration. Examples: "300m", "1.5h" or "2h45m". 
Valid time units are "s", "m", "h".`,
			})
		}
		if err := BanPlayer(c, r.SteamID, r.AuthorID, duration, r.Reason, r.ReasonText, model.Web); err != nil {
			c.JSON(http.StatusNotAcceptable, StatusResponse{
				Success: false,
				Message: "Failed to perform ban",
			})
		}
	}
}
