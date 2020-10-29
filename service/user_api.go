package service

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/gbans/store"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func onGetBans() gin.HandlerFunc {
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
