package service

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/gbans/store"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func onAPIGetFilteredWords() gin.HandlerFunc {
	type resp struct {
		Count int      `json:"count"`
		Words []string `json:"words"`
	}
	return func(c *gin.Context) {
		words, err := store.GetFilteredWords()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{})
			return
		}
		c.JSON(http.StatusOK, resp{
			Count: len(words),
			Words: words,
		})
	}

}

func onAPIGetStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		stats, err := store.GetStats()
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		serverStateMu.RLock()
		defer serverStateMu.RUnlock()
		for _, server := range serverState {
			if server.Alive {
				stats.ServersAlive++
			}
		}
		c.JSON(http.StatusOK, stats)
	}

}

func onAPIGetBans() gin.HandlerFunc {
	type resp struct {
		Total int                  `json:"total"`
		Bans  []model.BannedPerson `json:"bans"`
	}
	return func(c *gin.Context) {
		o := store.NewSearchQueryOpts(c.GetString("q"))
		o.Limit = queryInt(c, "limit")
		if o.Limit > 100 {
			o.Limit = 100
		} else if o.Limit <= 0 {
			o.Limit = 100
		}
		o.Offset = queryInt(c, "offset")
		switch c.Query("desc") {
		case "false":
			o.OrderDesc = false
		case "true":
			fallthrough
		default:
			o.OrderDesc = true
		}
		switch c.DefaultQuery("order_by", "created_on") {
		case "created_on":
			fallthrough
		default:
			o.OrderBy = "created_on"
		}
		log.Println(o)
		bans, err := store.GetBans(o)
		if err != nil {
			log.Errorf("Failed to fetch bans")
			c.JSON(http.StatusInternalServerError, M{})
			return
		}
		total := store.GetBansTotal()
		c.JSON(200, resp{
			Total: total,
			Bans:  bans,
		})
	}
}
