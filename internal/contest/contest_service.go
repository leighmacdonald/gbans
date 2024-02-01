package contest

import (
	"encoding/base64"
	"errors"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

type ContestHandler struct {
	contestUsecase domain.ContestUsecase
	configUsecase  domain.ConfigUsecase
	mediaUsecase   domain.MediaUsecase
	log            *zap.Logger
}

func NewContestHandler(logger *zap.Logger, engine *gin.Engine, cu domain.ContestUsecase,
	configUsecase domain.ConfigUsecase, mediaUsecase domain.MediaUsecase, ath domain.AuthUsecase,
) {
	handler := &ContestHandler{
		contestUsecase: cu,
		configUsecase:  configUsecase,
		mediaUsecase:   mediaUsecase,
		log:            logger.Named("contest"),
	}

	// opt
	optGrp := engine.Group("/")
	{
		opt := optGrp.Use(ath.AuthMiddleware(domain.PGuest))
		opt.GET("/api/contests", handler.onAPIGetContests())
		opt.GET("/api/contests/:contest_id", handler.onAPIGetContest())
		opt.GET("/api/contests/:contest_id/entries", handler.onAPIGetContestEntries())
	}

	// auth
	authGrp := engine.Group("/")
	{
		authed := authGrp.Use(ath.AuthMiddleware(domain.PUser))
		authed.POST("/api/contests/:contest_id/upload", handler.onAPISaveContestEntryMedia())
		authed.GET("/api/contests/:contest_id/vote/:contest_entry_id/:direction", handler.onAPISaveContestEntryVote())
		authed.POST("/api/contests/:contest_id/submit", handler.onAPISaveContestEntrySubmit())
		authed.DELETE("/api/contest_entry/:contest_entry_id", handler.onAPIDeleteContestEntry())
	}

	// mods
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.AuthMiddleware(domain.PModerator))
		mod.POST("/api/contests", handler.onAPIPostContest())
		mod.DELETE("/api/contests/:contest_id", handler.onAPIDeleteContest())
		mod.PUT("/api/contests/:contest_id", handler.onAPIUpdateContest())
	}
}

func (c *ContestHandler) contestFromCtx(ctx *gin.Context) (domain.Contest, bool) {
	contestID, idErr := httphelper.GetUUIDParam(ctx, "contest_id")
	if idErr != nil {
		httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

		return domain.Contest{}, false
	}

	var contest domain.Contest
	if errContests := c.contestUsecase.ContestByID(ctx, contestID, &contest); errContests != nil {
		httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

		return domain.Contest{}, false
	}

	if !contest.Public && httphelper.CurrentUserProfile(ctx).PermissionLevel < domain.PModerator {
		httphelper.ResponseErr(ctx, http.StatusForbidden, domain.ErrNotFound)

		return domain.Contest{}, false
	}

	return contest, true
}

func (c *ContestHandler) onAPIGetContests() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := httphelper.CurrentUserProfile(ctx)
		publicOnly := user.PermissionLevel < domain.PModerator
		contests, errContests := c.contestUsecase.Contests(ctx, publicOnly)

		if errContests != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, entries)
	}
}

func (c *ContestHandler) onAPIPostContest() gin.HandlerFunc {
	log := c.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		newContest, _ := domain.NewContest("", "", time.Now(), time.Now(), false)
		if !httphelper.Bind(ctx, log, &newContest) {
			return
		}

		if newContest.ContestID.IsNil() {
			newID, errID := uuid.NewV4()
			if errID != nil {
				httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

				return
			}

			newContest.ContestID = newID
		}

		if errSave := c.contestUsecase.ContestSave(ctx, &newContest); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		ctx.JSON(http.StatusOK, newContest)
	}
}

func (c *ContestHandler) onAPIDeleteContest() gin.HandlerFunc {
	log := c.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		contestID, idErr := httphelper.GetUUIDParam(ctx, "contest_id")
		if idErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		var contest domain.Contest

		if errContest := c.contestUsecase.ContestByID(ctx, contestID, &contest); errContest != nil {
			if errors.Is(errContest, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrUnknownID)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			log.Error("Error getting contest for deletion", zap.Error(errContest))

			return
		}

		if errDelete := c.contestUsecase.ContestDelete(ctx, contest.ContestID); errDelete != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
		if !httphelper.Bind(ctx, log, &contest) {
			return
		}

		if errSave := c.contestUsecase.ContestSave(ctx, &contest); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		content, decodeErr := base64.StdEncoding.DecodeString(req.Content)
		if decodeErr != nil {
			ctx.JSON(http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		media, errCreate := c.mediaUsecase.Create(ctx, httphelper.CurrentUserProfile(ctx).SteamID,
			req.Name, req.Mime, content, strings.Split(contest.MediaTypes, ","))
		if errHandle := httphelper.ErrorHandled(ctx, errCreate); errHandle != nil {
			log.Error("Failed to save user contest media", zap.Error(errHandle))

			return
		}

		// Don't bother to resend entire body
		media.Contents = nil

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

		contestEntryID, errContestEntryID := httphelper.GetUUIDParam(ctx, "contest_entry_id")
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

		currentUser := httphelper.CurrentUserProfile(ctx)

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
		user := httphelper.CurrentUserProfile(ctx)
		contest, success := c.contestFromCtx(ctx)

		if !success {
			return
		}

		var req entryReq
		if !httphelper.Bind(ctx, log, &req) {
			return
		}

		if contest.MediaTypes != "" {
			// TODO delete assets? reformat this?
			var media domain.Media
			if errMedia := c.mediaUsecase.GetMediaByAssetID(ctx, req.AssetID, &media); errMedia != nil {
				httphelper.ResponseErr(ctx, http.StatusFailedDependency, domain.ErrFetchMedia)

				return
			}

			if !slices.Contains(strings.Split(contest.MediaTypes, ","), strings.ToLower(media.MimeType)) {
				httphelper.ResponseErr(ctx, http.StatusFailedDependency, domain.ErrInvalidFormat)

				return
			}
		}

		existingEntries, errEntries := c.contestUsecase.ContestEntries(ctx, contest.ContestID)
		if errEntries != nil && !errors.Is(errEntries, domain.ErrNoResult) {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrContestLoadEntries)

			return
		}

		own := 0

		for _, entry := range existingEntries {
			if entry.SteamID == user.SteamID {
				own++
			}

			if own >= contest.MaxSubmissions {
				httphelper.ResponseErr(ctx, http.StatusForbidden, domain.ErrContestMaxEntries)

				return
			}
		}

		steamID := httphelper.CurrentUserProfile(ctx).SteamID

		entry, errEntry := contest.NewEntry(steamID, req.AssetID, req.Description)
		if errEntry != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrEntryCreate)

			return
		}

		if errSave := c.contestUsecase.ContestEntrySave(ctx, entry); errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrEntrySave)

			return
		}

		ctx.JSON(http.StatusCreated, entry)

		log.Info("New contest entry submitted", zap.String("contest_id", contest.ContestID.String()))
	}
}

func (c *ContestHandler) onAPIDeleteContestEntry() gin.HandlerFunc {
	log := c.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		user := httphelper.CurrentUserProfile(ctx)

		contestEntryID, idErr := httphelper.GetUUIDParam(ctx, "contest_entry_id")
		if idErr != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		var entry domain.ContestEntry

		if errContest := c.contestUsecase.ContestEntry(ctx, contestEntryID, &entry); errContest != nil {
			if errors.Is(errContest, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrUnknownID)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			log.Error("Error getting contest entry for deletion", zap.Error(errContest))

			return
		}

		// Only >=moderators or the entry author are allowed to delete entries.
		if !(user.PermissionLevel >= domain.PModerator || user.SteamID == entry.SteamID) {
			httphelper.ResponseErr(ctx, http.StatusForbidden, domain.ErrPermissionDenied)

			return
		}

		var contest domain.Contest

		if errContest := c.contestUsecase.ContestByID(ctx, entry.ContestID, &contest); errContest != nil {
			if errors.Is(errContest, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrUnknownID)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			log.Error("Error getting contest", zap.Error(errContest))

			return
		}

		// Only allow mods to delete entries from expired contests.
		if user.SteamID == entry.SteamID && time.Since(contest.DateEnd) > 0 {
			httphelper.ResponseErr(ctx, http.StatusForbidden, domain.ErrPermissionDenied)

			log.Error("User tried to delete entry from expired contest")

			return
		}

		if errDelete := c.contestUsecase.ContestEntryDelete(ctx, entry.ContestEntryID); errDelete != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
