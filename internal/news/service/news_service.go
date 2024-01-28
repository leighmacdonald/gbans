package service

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"net/http"
	"runtime"
)

func onAPIGetNewsLatest() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := env.Store().GetNewsLatest(ctx, 50, false)
		if errGetNewsLatest != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newsLatest)
	}
}

func onAPIPostNewsCreate() gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.NewsEntry
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		if errSave := env.Store().SaveNewsArticle(ctx, &req); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, req)

		conf := env.Config()

		go env.SendPayload(conf.Discord.LogChannelID, discord.NewNewsMessage(req.BodyMD, req.Title))
	}
}

func onAPIPostNewsUpdate() gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		newsID, errID := getIntParam(ctx, "news_id")
		if errID != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter

			return
		}

		var entry domain.NewsEntry
		if errGet := env.Store().GetNewsByID(ctx, newsID, &entry); errGet != nil {
			if errors.Is(errs.DBErr(errGet), errs.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		if !http_helper.Bind(ctx, log, &entry) {
			return
		}

		if errSave := env.Store().SaveNewsArticle(ctx, &entry); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusAccepted, entry)

		conf := env.Config()
		env.SendPayload(conf.Discord.LogChannelID, discord.EditNewsMessages(entry.Title, entry.BodyMD))
	}
}

func onAPIGetNewsAll() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := env.Store().GetNewsLatest(ctx, 100, true)
		if errGetNewsLatest != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newsLatest)
	}
}
