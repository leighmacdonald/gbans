package service

import (
	"bytes"
	"encoding/base64"
	"errors"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"go.uber.org/zap"
)

type ContestHandler struct {
	contestUsecase domain.ContestUsecase
	configUsecase  domain.ConfigUsecase
	s3usecase      domain.S3Usecase
	assetUsecase   domain.AssetUsecase
	wikiUsecase    domain.WikiUsecase
	log            *zap.Logger
}

func NewContestHandler(logger *zap.Logger, engine *gin.Engine, cu domain.ContestUsecase,
	configUsecase domain.ConfigUsecase, s3usecase domain.S3Usecase, assetUsecase domain.AssetUsecase, wikiUsecase domain.WikiUsecase,
) {
	handler := &ContestHandler{
		contestUsecase: cu,
		configUsecase:  configUsecase,
		s3usecase:      s3usecase,
		assetUsecase:   assetUsecase,
		wikiUsecase:    wikiUsecase,
		log:            logger.Named("contest"),
	}
	// opt
	engine.GET("/api/contests", handler.onAPIGetContests())
	engine.GET("/api/contests/:contest_id", handler.onAPIGetContest())
	engine.GET("/api/contests/:contest_id/entries", handler.onAPIGetContestEntries())

	// auth
	engine.POST("/api/contests/:contest_id/upload", handler.onAPISaveContestEntryMedia())
	engine.GET("/api/contests/:contest_id/vote/:contest_entry_id/:direction", handler.onAPISaveContestEntryVote())
	engine.POST("/api/contests/:contest_id/submit", handler.onAPISaveContestEntrySubmit())
	engine.DELETE("/api/contest_entry/:contest_entry_id", handler.onAPIDeleteContestEntry())

	// mod
	engine.POST("/api/contests", handler.onAPIPostContest())
	engine.DELETE("/api/contests/:contest_id", handler.onAPIDeleteContest())
	engine.PUT("/api/contests/:contest_id", handler.onAPIUpdateContest())
}

func (c *ContestHandler) contestFromCtx(ctx *gin.Context) (domain.Contest, bool) {
	contestID, idErr := http_helper.GetUUIDParam(ctx, "contest_id")
	if idErr != nil {
		http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

		return domain.Contest{}, false
	}

	var contest domain.Contest
	if errContests := c.contestUsecase.ContestByID(ctx, contestID, &contest); errContests != nil {
		http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

		return domain.Contest{}, false
	}

	if !contest.Public && http_helper.CurrentUserProfile(ctx).PermissionLevel < domain.PModerator {
		http_helper.ResponseErr(ctx, http.StatusForbidden, domain.ErrNotFound)

		return domain.Contest{}, false
	}

	return contest, true
}

func (c *ContestHandler) onAPIGetContests() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := http_helper.CurrentUserProfile(ctx)
		publicOnly := user.PermissionLevel < domain.PModerator
		contests, errContests := c.contestUsecase.Contests(ctx, publicOnly)

		if errContests != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(int64(len(contests)), contests))
	}
}

func (c *ContestHandler) onAPIGetContest() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		contest, success := c.contestFromCtx(ctx)
		if !success {
			return
		}

		ctx.JSON(http.StatusOK, contest)
	}
}

func (c *ContestHandler) onAPIGetContestEntries() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		contest, success := c.contestFromCtx(ctx)
		if !success {
			return
		}

		entries, errEntries := c.contestUsecase.ContestEntries(ctx, contest.ContestID)
		if errEntries != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, entries)
	}
}

func (c *ContestHandler) onAPIPostContest() gin.HandlerFunc {
	log := c.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		newContest, _ := domain.NewContest("", "", time.Now(), time.Now(), false)
		if !http_helper.Bind(ctx, log, &newContest) {
			return
		}

		if newContest.ContestID.IsNil() {
			newID, errID := uuid.NewV4()
			if errID != nil {
				http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

				return
			}

			newContest.ContestID = newID
		}

		if errSave := c.contestUsecase.ContestSave(ctx, &newContest); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		ctx.JSON(http.StatusOK, newContest)
	}
}

func (c *ContestHandler) onAPIDeleteContest() gin.HandlerFunc {
	log := c.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		contestID, idErr := http_helper.GetUUIDParam(ctx, "contest_id")
		if idErr != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		var contest domain.Contest

		if errContest := c.contestUsecase.ContestByID(ctx, contestID, &contest); errContest != nil {
			if errors.Is(errContest, domain.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrUnknownID)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			log.Error("Error getting contest for deletion", zap.Error(errContest))

			return
		}

		if errDelete := c.contestUsecase.ContestDelete(ctx, contest.ContestID); errDelete != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			log.Error("Error deleting contest", zap.Error(errDelete))

			return
		}

		ctx.Status(http.StatusAccepted)

		log.Info("Contest deleted",
			zap.String("contest_id", contestID.String()),
			zap.String("title", contest.Title))
	}
}

func (c *ContestHandler) onAPIUpdateContest() gin.HandlerFunc {
	log := c.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		if _, success := c.contestFromCtx(ctx); !success {
			return
		}

		var contest domain.Contest
		if !http_helper.Bind(ctx, log, &contest) {
			return
		}

		if errSave := c.contestUsecase.ContestSave(ctx, &contest); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			log.Error("Error updating contest", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusAccepted, contest)

		log.Info("Contest updated",
			zap.String("contest_id", contest.ContestID.String()),
			zap.String("title", contest.Title))
	}
}

func (c *ContestHandler) onAPISaveContestEntryMedia() gin.HandlerFunc {
	log := c.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		contest, success := c.contestFromCtx(ctx)
		if !success {
			return
		}

		var req domain.UserUploadedFile
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		content, decodeErr := base64.StdEncoding.DecodeString(req.Content)
		if decodeErr != nil {
			ctx.JSON(http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		media, errMedia := domain.NewMedia(http_helper.CurrentUserProfile(ctx).SteamID, req.Name, req.Mime, content)
		if errMedia != nil {
			ctx.JSON(http.StatusBadRequest, domain.ErrBadRequest)
			log.Error("Invalid media uploaded", zap.Error(errMedia))
		}

		conf := c.configUsecase.Config()

		asset, errAsset := domain.NewAsset(content, conf.S3.BucketMedia, "")
		if errAsset != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrAssetCreateFailed)

			return
		}

		if errPut := c.s3usecase.Put(ctx, conf.S3.BucketMedia, asset.Name, bytes.NewReader(content), asset.Size, asset.MimeType); errPut != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrAssetPut)

			log.Error("Failed to save user contest entry media to s3 backend", zap.Error(errPut))

			return
		}

		if errSaveAsset := c.assetUsecase.SaveAsset(ctx, &asset); errSaveAsset != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrAssetSave)

			log.Error("Failed to save user asset to s3 backend", zap.Error(errSaveAsset))
		}

		media.Asset = asset

		media.Contents = nil

		if !contest.MimeTypeAcceptable(media.MimeType) {
			http_helper.ResponseErr(ctx, http.StatusUnsupportedMediaType, domain.ErrInvalidFormat)
			log.Error("User tried uploading file with forbidden mimetype",
				zap.String("mime", media.MimeType), zap.String("name", media.Name))

			return
		}

		if errSave := c.wikiUsecase.SaveMedia(ctx, &media); errSave != nil {
			log.Error("Failed to save user contest media", zap.Error(errSave))

			if errors.Is(errs.DBErr(errSave), domain.ErrDuplicate) {
				http_helper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicateMediaName)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrSaveMedia)

			return
		}

		ctx.JSON(http.StatusCreated, media)
	}
}

func (c *ContestHandler) onAPISaveContestEntryVote() gin.HandlerFunc {
	log := c.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type voteResult struct {
		CurrentVote string `json:"current_vote"`
	}

	return func(ctx *gin.Context) {
		contest, success := c.contestFromCtx(ctx)
		if !success {
			return
		}

		contestEntryID, errContestEntryID := http_helper.GetUUIDParam(ctx, "contest_entry_id")
		if errContestEntryID != nil {
			ctx.JSON(http.StatusNotFound, domain.ErrNotFound)
			log.Error("Invalid contest entry id option")

			return
		}

		direction := strings.ToLower(ctx.Param("direction"))
		if direction != "up" && direction != "down" {
			ctx.JSON(http.StatusBadRequest, domain.ErrBadRequest)
			log.Error("Invalid vote direction option")

			return
		}

		if !contest.Voting || !contest.DownVotes && direction != "down" {
			ctx.JSON(http.StatusBadRequest, domain.ErrBadRequest)
			log.Error("Voting not enabled")

			return
		}

		currentUser := http_helper.CurrentUserProfile(ctx)

		if errVote := c.contestUsecase.ContestEntryVote(ctx, contestEntryID, currentUser.SteamID, direction == "up"); errVote != nil {
			if errors.Is(errVote, domain.ErrVoteDeleted) {
				ctx.JSON(http.StatusOK, voteResult{""})

				return
			}

			ctx.JSON(http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, voteResult{direction})
	}
}

func (c *ContestHandler) onAPISaveContestEntrySubmit() gin.HandlerFunc {
	type entryReq struct {
		Description string    `json:"description"`
		AssetID     uuid.UUID `json:"asset_id"`
	}

	log := c.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		user := http_helper.CurrentUserProfile(ctx)
		contest, success := c.contestFromCtx(ctx)

		if !success {
			return
		}

		var req entryReq
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		if contest.MediaTypes != "" {
			var media domain.Media
			if errMedia := c.wikiUsecase.GetMediaByAssetID(ctx, req.AssetID, &media); errMedia != nil {
				http_helper.ResponseErr(ctx, http.StatusFailedDependency, domain.ErrFetchMedia)

				return
			}

			if !contest.MimeTypeAcceptable(media.MimeType) {
				http_helper.ResponseErr(ctx, http.StatusFailedDependency, domain.ErrInvalidFormat)

				return
			}
		}

		existingEntries, errEntries := c.contestUsecase.ContestEntries(ctx, contest.ContestID)
		if errEntries != nil && !errors.Is(errEntries, domain.ErrNoResult) {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrContestLoadEntries)

			return
		}

		own := 0

		for _, entry := range existingEntries {
			if entry.SteamID == user.SteamID {
				own++
			}

			if own >= contest.MaxSubmissions {
				http_helper.ResponseErr(ctx, http.StatusForbidden, domain.ErrContestMaxEntries)

				return
			}
		}

		steamID := http_helper.CurrentUserProfile(ctx).SteamID

		entry, errEntry := contest.NewEntry(steamID, req.AssetID, req.Description)
		if errEntry != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrEntryCreate)

			return
		}

		if errSave := c.contestUsecase.ContestEntrySave(ctx, entry); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrEntrySave)

			return
		}

		ctx.JSON(http.StatusCreated, entry)

		log.Info("New contest entry submitted", zap.String("contest_id", contest.ContestID.String()))
	}
}

func (c *ContestHandler) onAPIDeleteContestEntry() gin.HandlerFunc {
	log := c.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		user := http_helper.CurrentUserProfile(ctx)

		contestEntryID, idErr := http_helper.GetUUIDParam(ctx, "contest_entry_id")
		if idErr != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		var entry domain.ContestEntry

		if errContest := c.contestUsecase.ContestEntry(ctx, contestEntryID, &entry); errContest != nil {
			if errors.Is(errContest, domain.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrUnknownID)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			log.Error("Error getting contest entry for deletion", zap.Error(errContest))

			return
		}

		// Only >=moderators or the entry author are allowed to delete entries.
		if !(user.PermissionLevel >= domain.PModerator || user.SteamID == entry.SteamID) {
			http_helper.ResponseErr(ctx, http.StatusForbidden, domain.ErrPermissionDenied)

			return
		}

		var contest domain.Contest

		if errContest := c.contestUsecase.ContestByID(ctx, entry.ContestID, &contest); errContest != nil {
			if errors.Is(errContest, domain.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrUnknownID)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			log.Error("Error getting contest", zap.Error(errContest))

			return
		}

		// Only allow mods to delete entries from expired contests.
		if user.SteamID == entry.SteamID && time.Since(contest.DateEnd) > 0 {
			http_helper.ResponseErr(ctx, http.StatusForbidden, domain.ErrPermissionDenied)

			log.Error("User tried to delete entry from expired contest")

			return
		}

		if errDelete := c.contestUsecase.ContestEntryDelete(ctx, entry.ContestEntryID); errDelete != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			log.Error("Error deleting contest entry", zap.Error(errDelete))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})

		log.Info("Contest deleted",
			zap.String("contest_id", entry.ContestID.String()),
			zap.String("contest_entry_id", entry.ContestEntryID.String()),
			zap.String("title", contest.Title))
	}
}
