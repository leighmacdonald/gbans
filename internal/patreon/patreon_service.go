package patreon

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"go.uber.org/zap"
)

type PatreonHandler struct {
	pu  domain.PatreonUsecase
	log *zap.Logger
}

func NewPatreonHandler(log *zap.Logger, engine *gin.Engine, pu domain.PatreonUsecase) {
	handler := PatreonHandler{
		pu:  pu,
		log: log.Named("patreon"),
	}

	engine.GET("/api/patreon/campaigns", handler.onAPIGetPatreonCampaigns())

	// mod
	engine.GET("/api/patreon/pledges", handler.onAPIGetPatreonPledges())
}

func (h PatreonHandler) onAPIGetPatreonCampaigns() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tiers, errTiers := h.pu.Tiers()
		if errTiers != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		ctx.JSON(http.StatusOK, tiers)
	}
}

func (h PatreonHandler) onAPIGetPatreonPledges() gin.HandlerFunc {
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
		pledges, _, errPledges := h.pu.Pledges()
		if errPledges != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		ctx.JSON(http.StatusOK, pledges)
	}
}
