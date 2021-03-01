package service

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/model"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func onAPIGetFilteredWords() gin.HandlerFunc {
	type resp struct {
		Count int      `json:"count"`
		Words []string `json:"words"`
	}
	return func(c *gin.Context) {
		words, err := GetFilteredWords()
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
		stats, err := GetStats()
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
	type req struct {
		SortDesc bool   `json:"sort_desc"`
		Offset   uint64 `json:"offset"`
		Limit    uint64 `json:"limit"`
		OrderBy  string `json:"order_by"`
		Query    string `json:"query"`
	}
	type resp struct {
		Total int                  `json:"total"`
		Bans  []model.BannedPerson `json:"bans"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.BindJSON(&r); err != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		o := newSearchQueryOpts(r.Query)
		o.Limit = r.Limit
		if o.Limit > 100 {
			o.Limit = 100
		} else if o.Limit <= 0 {
			o.Limit = 100
		}
		o.Offset = r.Offset
		switch o.OrderDesc {
		case true:
			o.OrderDesc = true
		case false:
			fallthrough
		default:
			o.OrderDesc = false
		}
		o.OrderBy = r.OrderBy

		bans, err := GetBans(o)
		if err != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			log.Errorf("Failed to fetch bans")
			return
		}
		t, err := GetBansTotal(o)
		if err != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			log.Errorf("Failed to fetch ban total")
			return
		}
		responseOK(c, http.StatusOK, resp{
			Total: t,
			Bans:  bans,
		})
	}
}
