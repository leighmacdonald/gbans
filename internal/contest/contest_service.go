package contest

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
	"golang.org/x/exp/slices"
)

type contestHandler struct {
	contests domain.ContestUsecase
	config   domain.ConfigUsecase
	assets   domain.AssetUsecase
}

func NewContestHandler(engine *gin.Engine, contests domain.ContestUsecase,
	config domain.ConfigUsecase, assets domain.AssetUsecase, auth domain.AuthUsecase,
) {
	handler := &contestHandler{
		contests: contests,
		config:   config,
		assets:   assets,
	}

	// opt
	optGrp := engine.Group("/")
	{
		opt := optGrp.Use(auth.AuthMiddleware(domain.PGuest))
		opt.GET("/api/contests", handler.onAPIGetContests())
		opt.GET("/api/contests/:contest_id", handler.onAPIGetContest())
		opt.GET("/api/contests/:contest_id/entries", handler.onAPIGetContestEntries())
	}

	// auth
	authGrp := engine.Group("/")
	{
		authed := authGrp.Use(auth.AuthMiddleware(domain.PUser))
		authed.POST("/api/contests/:contest_id/upload", handler.onAPISaveContestEntryMedia())
		authed.GET("/api/contests/:contest_id/vote/:contest_entry_id/:direction", handler.onAPISaveContestEntryVote())
		authed.POST("/api/contests/:contest_id/submit", handler.onAPISaveContestEntrySubmit())
		authed.DELETE("/api/contest_entry/:contest_entry_id", handler.onAPIDeleteContestEntry())
	}

	// mods
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(auth.AuthMiddleware(domain.PModerator))
		mod.POST("/api/contests", handler.onAPIPostContest())
		mod.DELETE("/api/contests/:contest_id", handler.onAPIDeleteContest())
		mod.PUT("/api/contests/:contest_id", handler.onAPIUpdateContest())
	}
}

func (c *contestHandler) contestFromCtx(ctx *gin.Context) (domain.Contest, bool) {
	contestID, idErr := httphelper.GetUUIDParam(ctx, "contest_id")
	if idErr != nil {
		httphelper.HandleErrBadRequest(ctx)

		return domain.Contest{}, false
	}

	var contest domain.Contest
	if errContests := c.contests.ContestByID(ctx, contestID, &contest); errContests != nil {
		httphelper.HandleErrInternal(ctx)

		return domain.Contest{}, false
	}

	if !contest.Public && httphelper.CurrentUserProfile(ctx).PermissionLevel < domain.PModerator {
		httphelper.ResponseAPIErr(ctx, http.StatusForbidden, domain.ErrNotFound)

		return domain.Contest{}, false
	}

	return contest, true
}

func (c *contestHandler) onAPIGetContests() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		contests, errContests := c.contests.Contests(ctx, httphelper.CurrentUserProfile(ctx))

		if errContests != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to fetch contests", log.ErrAttr(errContests))

			return
		}

		if contests == nil {
			contests = []domain.Contest{}
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

		entries, errEntries := c.contests.ContestEntries(ctx, contest.ContestID)
		if errEntries != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to fetch contest entries", log.ErrAttr(errEntries))

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

		contest, errSave := c.contests.ContestSave(ctx, newContest)
		if errSave != nil {
			httphelper.HandleErrs(ctx, errSave)

			return
		}

		ctx.JSON(http.StatusOK, contest)
	}
}

func (c *contestHandler) onAPIDeleteContest() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		contestID, idErr := httphelper.GetUUIDParam(ctx, "contest_id")
		if idErr != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get valid contest_id")

			return
		}

		var contest domain.Contest

		if errContest := c.contests.ContestByID(ctx, contestID, &contest); errContest != nil {
			if errors.Is(errContest, domain.ErrNoResult) {
				httphelper.ResponseAPIErr(ctx, http.StatusNotFound, domain.ErrUnknownID)

				return
			}

			httphelper.HandleErrBadRequest(ctx)
			slog.Error("Error getting contest for deletion", log.ErrAttr(errContest))

			return
		}

		if errDelete := c.contests.ContestDelete(ctx, contest.ContestID); errDelete != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Error deleting contest", log.ErrAttr(errDelete))

			return
		}

		ctx.Status(http.StatusAccepted)
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

		contest, errSave := c.contests.ContestSave(ctx, req)
		if errSave != nil {
			httphelper.HandleErrs(ctx, errSave)
			slog.Error("Error updating contest", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusAccepted, contest)
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

		mediaFile, errOpen := req.File.Open()
		if errOpen != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to open form file", log.ErrAttr(errOpen))

			return
		}

		if contest.MediaTypes != "" {
			mimeType, errMimeType := mimetype.DetectReader(mediaFile)
			if errMimeType != nil {
				httphelper.HandleErrInternal(ctx)
				slog.Error("Failed to detect mime type", log.ErrAttr(errMimeType))

				return
			}

			if !slices.Contains(strings.Split(strings.ToLower(contest.MediaTypes), ","), strings.ToLower(mimeType.String())) {
				httphelper.ResponseAPIErr(ctx, http.StatusBadRequest, domain.ErrMimeTypeNotAllowed)
				slog.Warn("User tried to upload file with disallowed mime type", slog.String("mime", strings.ToLower(mimeType.String())))

				return
			}
		}

		authorID := httphelper.CurrentUserProfile(ctx).SteamID

		asset, errCreate := c.assets.Create(ctx, authorID, "media", req.Name, mediaFile)
		if errHandle := httphelper.HandleErrsReturn(ctx, errCreate); errHandle != nil {
			slog.Error("Failed to save user contest media", log.ErrAttr(errHandle))

			return
		}

		ctx.JSON(http.StatusCreated, asset)
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
			httphelper.HandleErrs(ctx, contestIDErr)
			slog.Warn("Got invalid contest id")

			return
		}

		contestEntryID, errContestEntryID := httphelper.GetUUIDParam(ctx, "contest_entry_id")
		if errContestEntryID != nil {
			httphelper.HandleErrNotFound(ctx)
			slog.Error("Unknown contest entry id")

			return
		}

		direction := strings.ToLower(ctx.Param("direction"))
		if direction != "up" && direction != "down" {
			ctx.JSON(http.StatusBadRequest, domain.ErrBadRequest)
			slog.Error("Invalid vote direction option")

			return
		}

		if errVote := c.contests.ContestEntryVote(ctx, contestID, contestEntryID, httphelper.CurrentUserProfile(ctx), direction == "up"); errVote != nil {
			if errors.Is(errVote, domain.ErrVoteDeleted) {
				ctx.JSON(http.StatusOK, voteResult{""})

				return
			}

			httphelper.HandleErrInternal(ctx)

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

		existingEntries, errEntries := c.contests.ContestEntries(ctx, contest.ContestID)
		if errEntries != nil && !errors.Is(errEntries, domain.ErrNoResult) {
			httphelper.ResponseAPIErr(ctx, http.StatusInternalServerError, domain.ErrContestLoadEntries)

			return
		}

		own := 0

		for _, entry := range existingEntries {
			if entry.SteamID == user.SteamID {
				own++
			}

			if own >= contest.MaxSubmissions {
				httphelper.ResponseAPIErr(ctx, http.StatusForbidden, domain.ErrContestMaxEntries)

				return
			}
		}

		steamID := httphelper.CurrentUserProfile(ctx).SteamID

		asset, _, errAsset := c.assets.Get(ctx, req.AssetID)
		if errAsset != nil {
			httphelper.ResponseAPIErr(ctx, http.StatusInternalServerError, domain.ErrEntryCreate)

			return
		}

		if asset.AuthorID != steamID {
			httphelper.ResponseAPIErr(ctx, http.StatusForbidden, domain.ErrPermissionDenied)

			return
		}

		entry, errEntry := contest.NewEntry(steamID, req.AssetID, req.Description)
		if errEntry != nil {
			httphelper.ResponseAPIErr(ctx, http.StatusInternalServerError, domain.ErrEntryCreate)

			return
		}

		if errSave := c.contests.ContestEntrySave(ctx, entry); errSave != nil {
			httphelper.ResponseAPIErr(ctx, http.StatusInternalServerError, domain.ErrEntrySave)

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
			httphelper.ResponseAPIErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		var entry domain.ContestEntry

		if errContest := c.contests.ContestEntry(ctx, contestEntryID, &entry); errContest != nil {
			if errors.Is(errContest, domain.ErrNoResult) {
				httphelper.ResponseAPIErr(ctx, http.StatusNotFound, domain.ErrUnknownID)

				return
			}

			httphelper.ResponseAPIErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			slog.Error("Error getting contest entry for deletion", log.ErrAttr(errContest))

			return
		}

		// Only >=moderators or the entry author are allowed to delete entries.
		if !(user.PermissionLevel >= domain.PModerator || user.SteamID == entry.SteamID) {
			httphelper.ResponseAPIErr(ctx, http.StatusForbidden, domain.ErrPermissionDenied)

			return
		}

		var contest domain.Contest

		if errContest := c.contests.ContestByID(ctx, entry.ContestID, &contest); errContest != nil {
			if errors.Is(errContest, domain.ErrNoResult) {
				httphelper.ResponseAPIErr(ctx, http.StatusNotFound, domain.ErrUnknownID)

				return
			}

			httphelper.ResponseAPIErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			slog.Error("Error getting contest", log.ErrAttr(errContest))

			return
		}

		// Only allow mods to delete entries from expired contests.
		if user.SteamID == entry.SteamID && time.Since(contest.DateEnd) > 0 {
			httphelper.ResponseAPIErr(ctx, http.StatusForbidden, domain.ErrPermissionDenied)

			slog.Error("User tried to delete entry from expired contest")

			return
		}

		if errDelete := c.contests.ContestEntryDelete(ctx, entry.ContestEntryID); errDelete != nil {
			httphelper.ResponseAPIErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
