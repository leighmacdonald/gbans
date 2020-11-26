package service

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/store"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func onIndex() gin.HandlerFunc {
	return func(c *gin.Context) {
		render(c, "home", defaultArgs(c))
	}
}

func onServers() gin.HandlerFunc {
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
		c.JSON(http.StatusNotFound, gin.H{})
	}
}

func onGetMutes() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{})
	}
}

func onGetAppeal() gin.HandlerFunc {
	return func(c *gin.Context) {
		usr := currentPerson(c)
		ban, err := store.GetBan(usr.SteamID)
		if err != nil {
			if errors.Is(err, store.ErrNoResult) {
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

func onPostAppeal() gin.HandlerFunc {
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
		words, err := store.GetFilteredWords()
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
