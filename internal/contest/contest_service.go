package contest

import (
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
	"golang.org/x/exp/slices"
)

type contestHandler struct {
	contestUsecase domain.ContestUsecase
	configUsecase  domain.ConfigUsecase
	mediaUsecase   domain.MediaUsecase
}

func NewContestHandler(engine *gin.Engine, cu domain.ContestUsecase,
	configUsecase domain.ConfigUsecase, mediaUsecase domain.MediaUsecase, ath domain.AuthUsecase,
) {
	handler := &contestHandler{
		contestUsecase: cu,
		configUsecase:  configUsecase,
		mediaUsecase:   mediaUsecase,
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

func (c *contestHandler) contestFromCtx(ctx *gin.Context) (domain.Contest, bool) {
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

func (c *contestHandler) onAPIGetContests() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		contests, errContests := c.contestUsecase.Contests(ctx, httphelper.CurrentUserProfile(ctx))

		if errContests != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, contests)
	}
}

func (c *contestHandler) onAPIGetContest() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		contest, success := c.contestFromCtx(ctx)
		if !success {
			return
		}

		ctx.JSON(http.StatusOK, contest)
	}
}

func (c *contestHandler) onAPIGetContestEntries() gin.HandlerFunc {
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

func (c *contestHandler) onAPIPostContest() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newContest, _ := domain.NewContest("", "", time.Now(), time.Now(), false)
		if !httphelper.Bind(ctx, &newContest) {
			return
		}

		contest, errSave := c.contestUsecase.ContestSave(ctx, newContest)
		if errSave != nil {
			httphelper.ErrorHandled(ctx, errSave)

			return
		}

		ctx.JSON(http.StatusOK, contest)
	}
}

func (c *contestHandler) onAPIDeleteContest() gin.HandlerFunc {
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

			slog.Error("Error getting contest for deletion", log.ErrAttr(errContest))

			return
		}

		if errDelete := c.contestUsecase.ContestDelete(ctx, contest.ContestID); errDelete != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			slog.Error("Error deleting contest", log.ErrAttr(errDelete))

			return
		}

		ctx.Status(http.StatusAccepted)

		slog.Info("Contest deleted",
			slog.String("contest_id", contestID.String()),
			slog.String("title", contest.Title))
	}
}

func (c *contestHandler) onAPIUpdateContest() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if _, success := c.contestFromCtx(ctx); !success {
			return
		}

		var req domain.Contest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		contest, errSave := c.contestUsecase.ContestSave(ctx, req)
		if errSave != nil {
			httphelper.ErrorHandled(ctx, errSave)

			slog.Error("Error updating contest", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusAccepted, contest)

		slog.Info("Contest updated",
			slog.String("contest_id", req.ContestID.String()),
			slog.String("title", req.Title))
	}
}

func (c *contestHandler) onAPISaveContestEntryMedia() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		contest, success := c.contestFromCtx(ctx)
		if !success {
			return
		}

		var req domain.UserUploadedFile
		if !httphelper.Bind(ctx, &req) {
			return
		}

		content, decodeErr := base64.StdEncoding.DecodeString(req.Content)
		if decodeErr != nil {
			ctx.JSON(http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		media, errCreate := c.mediaUsecase.Create(ctx, httphelper.CurrentUserProfile(ctx).SteamID,
			req.Name, req.Mime, content, strings.Split(contest.MediaTypes, ","))
		if errHandle := httphelper.ErrorHandledWithReturn(ctx, errCreate); errHandle != nil {
			slog.Error("Failed to save user contest media", log.ErrAttr(errHandle))

			return
		}

		// Don't bother to resend entire body
		media.Contents = nil

		ctx.JSON(http.StatusCreated, media)
	}
}

func (c *contestHandler) getContestID(ctx *gin.Context) (uuid.UUID, error) {
	return httphelper.GetUUIDParam(ctx, "contest_id")
}

func (c *contestHandler) onAPISaveContestEntryVote() gin.HandlerFunc {
	type voteResult struct {
		CurrentVote string `json:"current_vote"`
	}

	return func(ctx *gin.Context) {
		contestID, contestIDErr := c.getContestID(ctx)
		if contestIDErr != nil {
			httphelper.ErrorHandled(ctx, contestIDErr)

			return
		}

		contestEntryID, errContestEntryID := httphelper.GetUUIDParam(ctx, "contest_entry_id")
		if errContestEntryID != nil {
			ctx.JSON(http.StatusNotFound, domain.ErrNotFound)
			slog.Error("Invalid contest entry id option")

			return
		}

		direction := strings.ToLower(ctx.Param("direction"))
		if direction != "up" && direction != "down" {
			ctx.JSON(http.StatusBadRequest, domain.ErrBadRequest)
			slog.Error("Invalid vote direction option")

			return
		}

		if errVote := c.contestUsecase.ContestEntryVote(ctx, contestID, contestEntryID, httphelper.CurrentUserProfile(ctx), direction == "up"); errVote != nil {
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

func (c *contestHandler) onAPISaveContestEntrySubmit() gin.HandlerFunc {
	type entryReq struct {
		Description string    `json:"description"`
		AssetID     uuid.UUID `json:"asset_id"`
	}

	return func(ctx *gin.Context) {
		user := httphelper.CurrentUserProfile(ctx)
		contest, success := c.contestFromCtx(ctx)

		if !success {
			return
		}

		var req entryReq
		if !httphelper.Bind(ctx, &req) {
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

		slog.Info("New contest entry submitted", slog.String("contest_id", contest.ContestID.String()))
	}
}

func (c *contestHandler) onAPIDeleteContestEntry() gin.HandlerFunc {
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

			slog.Error("Error getting contest entry for deletion", log.ErrAttr(errContest))

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

			slog.Error("Error getting contest", log.ErrAttr(errContest))

			return
		}

		// Only allow mods to delete entries from expired contests.
		if user.SteamID == entry.SteamID && time.Since(contest.DateEnd) > 0 {
			httphelper.ResponseErr(ctx, http.StatusForbidden, domain.ErrPermissionDenied)

			slog.Error("User tried to delete entry from expired contest")

			return
		}

		if errDelete := c.contestUsecase.ContestEntryDelete(ctx, entry.ContestEntryID); errDelete != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			slog.Error("Error deleting contest entry", log.ErrAttr(errDelete))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})

		slog.Info("Contest deleted",
			slog.String("contest_id", entry.ContestID.String()),
			slog.String("contest_entry_id", entry.ContestEntryID.String()),
			slog.String("title", contest.Title))
	}
}
