package votes

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type voteHandler struct {
	Votes
}

func NewVotesHandler(engine *gin.Engine, authenticator httphelper.Authenticator, votes Votes) {
	handler := voteHandler{votes}

	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(authenticator.Middleware(permission.Moderator))
		mod.POST("/api/votes", handler.onVotes())
	}
}

func (h voteHandler) onVotes() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req, ok := httphelper.BindJSON[Query](ctx)
		if !ok {
			return
		}

		votes, count, errVotes := h.Query(ctx, req)
		if errVotes != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errVotes, httphelper.ErrInternal)))

			return
		}

		if votes == nil {
			votes = []Result{}
		}

		ctx.JSON(http.StatusOK, httphelper.LazyResult[Result]{
			Count: count,
			Data:  votes,
		})
	}
}
