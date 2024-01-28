package service

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func onAPIGetPatreonCampaigns() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tiers, errTiers := env.Patreon().Tiers()
		if errTiers != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		ctx.JSON(http.StatusOK, tiers)
	}
}

func onAPIGetPatreonPledges() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Only leak specific details
		// type basicPledge struct {
		//	Name      string
		//	Amount    int
		//	CreatedAt time.Time
		// }
		// var basic []basicPledge
		// for _, p := range pledges {
		//	t0 := config.Now()
		//	if p.Attributes.CreatedAt.Valid {
		//		t0 = p.Attributes.CreatedAt.Time.UTC()
		//	}
		//	basic = append(basic, basicPledge{
		//		Name:      users[p.Relationships.Patron.Data.ID].Attributes.FullName,
		//		Amount:    p.Attributes.AmountCents,
		//		CreatedAt: t0,
		//	})
		// }
		pledges, _, errPledges := env.Patreon().Pledges()
		if errPledges != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		ctx.JSON(http.StatusOK, pledges)
	}
}
