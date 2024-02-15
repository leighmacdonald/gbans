package patreon

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type patreonHandler struct {
	pu domain.PatreonUsecase
}

func NewPatreonHandler(engine *gin.Engine, pu domain.PatreonUsecase, ath domain.AuthUsecase) {
	handler := patreonHandler{pu: pu}

	engine.GET("/api/patreon/campaigns", handler.onAPIGetPatreonCampaigns())

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.AuthMiddleware(domain.PUser))
		mod.GET("/api/patreon/pledges", handler.onAPIGetPatreonPledges())
	}
}

func (h patreonHandler) onAPIGetPatreonCampaigns() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tiers, errTiers := h.pu.Tiers()
		if errTiers != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		ctx.JSON(http.StatusOK, tiers)
	}
}

func (h patreonHandler) onAPIGetPatreonPledges() gin.HandlerFunc {
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
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		ctx.JSON(http.StatusOK, pledges)
	}
}
