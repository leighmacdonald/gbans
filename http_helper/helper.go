package http_helper

import "github.com/gin-gonic/gin"

type ApiError struct {
	Message string `json:"message"`
}

func ResponseErr(ctx *gin.Context, statusCode int, err error) {
	userErr := "API Error"
	if err != nil {
		userErr = err.Error()
	}

	ctx.JSON(statusCode, ApiError{Message: userErr})
}
