package votes

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type voteHandler struct {
	votes domain.VoteUsecase
}

func NewHandler(engine *gin.Engine, votes domain.VoteUsecase, auth domain.AuthUsecase) {
	handler := voteHandler{votes: votes}

	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(auth.Middleware(domain.PModerator))
		mod.POST("/api/votes", handler.onVotes())
	}
}

func (h voteHandler) onVotes() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.VoteQueryFilter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		votes, count, errVotes := h.votes.Query(ctx, req)
		if errVotes != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errVotes, domain.ErrInternal)))

			return
		}

		if votes == nil {
			votes = []domain.VoteResult{}
		}

		ctx.JSON(http.StatusOK, httphelper.LazyResult{
			Count: count,
			Data:  votes,
		})
	}
}
