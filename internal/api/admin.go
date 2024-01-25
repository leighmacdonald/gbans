package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func onAPIGetServerAdmins(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		perms, err := env.Store().GetServerPermissions(ctx)
		if err != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, perms)
	}
}
