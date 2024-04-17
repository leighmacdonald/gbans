package votes

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
)

func NewVoteHandler(engine *gin.Engine, voteUsecase domain.VoteUsecase, authUsecase domain.AuthUsecase) {
	handler := voteHandler{voteUsecase: voteUsecase}

	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(authUsecase.AuthMiddleware(domain.PModerator))
		mod.POST("/api/votes", handler.onVotes())
	}
}

type voteHandler struct {
	voteUsecase domain.VoteUsecase
}

func (h voteHandler) onVotes() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.VoteQueryFilter
		if !httphelper.Bind(ctx, &req) {
			return
		}

		votes, count, errVotes := h.voteUsecase.Query(ctx, req)
		if errVotes != nil {
			slog.Error("Failed to query vote history", log.ErrAttr(errVotes))
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		if votes == nil {
			votes = []domain.VoteResult{}
		}

		ctx.JSON(http.StatusOK, domain.LazyResult{
			Count: count,
			Data:  votes,
		})
	}
}
