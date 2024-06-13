package forum

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type forumHandler struct {
	forums domain.ForumUsecase
}

func NewForumHandler(engine *gin.Engine, forums domain.ForumUsecase, auth domain.AuthUsecase) {
	handler := &forumHandler{
		forums: forums,
	}

	engine.GET("/api/forum/active_users", handler.onAPIActiveUsers())

	// opt
	optGrp := engine.Group("/")
	{
		opt := optGrp.Use(auth.AuthMiddleware(domain.PGuest))
		opt.GET("/api/forum/overview", handler.onAPIForumOverview())
		opt.GET("/api/forum/messages/recent", handler.onAPIForumMessagesRecent())
		opt.POST("/api/forum/threads", handler.onAPIForumThreads())
		opt.GET("/api/forum/thread/:forum_thread_id", handler.onAPIForumThread())
		opt.GET("/api/forum/forum/:forum_id", handler.onAPIForum())
		opt.POST("/api/forum/messages", handler.onAPIForumMessages())
	}

	// auth
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(auth.AuthMiddleware(domain.PUser))
		authed.POST("/api/forum/forum/:forum_id/thread", handler.onAPIThreadCreate())
		authed.POST("/api/forum/thread/:forum_thread_id/message", handler.onAPIThreadCreateReply())
		authed.POST("/api/forum/message/:forum_message_id", handler.onAPIThreadMessageUpdate())
		authed.DELETE("/api/forum/thread/:forum_thread_id", handler.onAPIThreadDelete())
		authed.DELETE("/api/forum/message/:forum_message_id", handler.onAPIMessageDelete())
		authed.POST("/api/forum/thread/:forum_thread_id", handler.onAPIThreadUpdate())
	}
	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(auth.AuthMiddleware(domain.PModerator))
		mod.POST("/api/forum/category", handler.onAPICreateForumCategory())
		mod.GET("/api/forum/category/:forum_category_id", handler.onAPIForumCategory())
		mod.POST("/api/forum/category/:forum_category_id", handler.onAPIUpdateForumCategory())
		mod.POST("/api/forum/forum", handler.onAPICreateForumForum())
		mod.POST("/api/forum/forum/:forum_id", handler.onAPIUpdateForumForum())
	}
}

type CategoryRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Ordering    int    `json:"ordering"`
}

func (f *forumHandler) onAPIForumMessagesRecent() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := httphelper.CurrentUserProfile(ctx)

		messages, errThreads := f.forums.ForumRecentActivity(ctx, 5, user.PermissionLevel)
		if errThreads != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Could not load thread messages", log.ErrAttr(errThreads))

			return
		}

		if messages == nil {
			messages = []domain.ForumMessage{}
		}

		ctx.JSON(http.StatusOK, messages)
	}
}

func (f *forumHandler) onAPICreateForumCategory() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req CategoryRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		category := domain.ForumCategory{
			Title:       util.SanitizeUGC(req.Title),
			Description: util.SanitizeUGC(req.Description),
			Ordering:    req.Ordering,
			TimeStamped: domain.NewTimeStamped(),
		}

		if errSave := f.forums.ForumCategorySave(ctx, &category); errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Error creating new forum category", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, category)
	}
}

func (f *forumHandler) onAPIForumCategory() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		forumCategoryID, errCategoryID := httphelper.GetIntParam(ctx, "forum_category_id")
		if errCategoryID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get forum_category_id", log.ErrAttr(errCategoryID))

			return
		}

		var category domain.ForumCategory

		if errGet := f.forums.ForumCategory(ctx, forumCategoryID, &category); errGet != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Error fetching forum category", log.ErrAttr(errGet))

			return
		}

		ctx.JSON(http.StatusOK, category)
	}
}

func (f *forumHandler) onAPIUpdateForumCategory() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		categoryID, errCategoryID := httphelper.GetIntParam(ctx, "forum_category_id")
		if errCategoryID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get forum_category_id", log.ErrAttr(errCategoryID))

			return
		}

		var category domain.ForumCategory
		if errGet := f.forums.ForumCategory(ctx, categoryID, &category); errGet != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get forum category", log.ErrAttr(errGet))

			return
		}

		var req CategoryRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		category.Title = util.SanitizeUGC(req.Title)
		category.Description = util.SanitizeUGC(req.Description)
		category.Ordering = req.Ordering

		if errSave := f.forums.ForumCategorySave(ctx, &category); errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Error creating new forum category", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, category)
	}
}

type CreateForumRequest struct {
	ForumCategoryID int              `json:"forum_category_id"`
	PermissionLevel domain.Privilege `json:"permission_level"`
	CategoryRequest
}

func (f *forumHandler) onAPICreateForumForum() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req CreateForumRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		forum := domain.Forum{
			ForumCategoryID: req.ForumCategoryID,
			Title:           util.SanitizeUGC(req.Title),
			Description:     util.SanitizeUGC(req.Description),
			Ordering:        req.Ordering,
			PermissionLevel: req.PermissionLevel,
			TimeStamped:     domain.NewTimeStamped(),
		}

		if errSave := f.forums.ForumSave(ctx, &forum); errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Error creating new forum", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, forum)
	}
}

func (f *forumHandler) onAPIUpdateForumForum() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		forumID, errForumID := httphelper.GetIntParam(ctx, "forum_id")
		if errForumID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Got invalid forum_id", log.ErrAttr(errForumID))

			return
		}

		var forum domain.Forum
		if errGet := f.forums.Forum(ctx, forumID, &forum); errGet != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to update forum", log.ErrAttr(errGet), slog.Int("forum_id", forum.ForumID))

			return
		}

		var req CreateForumRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		forum.ForumCategoryID = req.ForumCategoryID
		forum.Title = util.SanitizeUGC(req.Title)
		forum.Description = util.SanitizeUGC(req.Description)
		forum.Ordering = req.Ordering
		forum.PermissionLevel = req.PermissionLevel

		if errSave := f.forums.ForumSave(ctx, &forum); errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Error updating forum", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, forum)
	}
}

func (f *forumHandler) onAPIThreadCreate() gin.HandlerFunc {
	type CreateThreadRequest struct {
		Title  string `json:"title"`
		BodyMD string `json:"body_md"`
		Sticky bool   `json:"sticky"`
		Locked bool   `json:"locked"`
	}

	type ThreadWithMessage struct {
		domain.ForumThread
		Message domain.ForumMessage `json:"message"`
	}

	return func(ctx *gin.Context) {
		user := httphelper.CurrentUserProfile(ctx)

		f.forums.Touch(user)

		forumID, errForumID := httphelper.GetIntParam(ctx, "forum_id")
		if errForumID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get forum_id", log.ErrAttr(errForumID))

			return
		}

		var req CreateThreadRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if len(req.BodyMD) <= 1 {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, fmt.Errorf("body: %w", domain.ErrTooShort))

			return
		}

		if len(req.Title) <= 4 {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, fmt.Errorf("title: %w", domain.ErrTooShort))

			return
		}

		var forum domain.Forum
		if errForum := f.forums.Forum(ctx, forumID, &forum); errForum != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get forum", log.ErrAttr(errForum))

			return
		}

		thread := forum.NewThread(req.Title, user.SteamID)
		thread.Sticky = req.Sticky
		thread.Locked = req.Locked

		if errSaveThread := f.forums.ForumThreadSave(ctx, &thread); errSaveThread != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to save new thread", log.ErrAttr(errSaveThread))

			return
		}

		message := thread.NewMessage(user.SteamID, req.BodyMD)

		if errSaveMessage := f.forums.ForumMessageSave(ctx, &message); errSaveMessage != nil {
			// Drop created thread.
			// TODO transaction
			if errRollback := f.forums.ForumThreadDelete(ctx, thread.ForumThreadID); errRollback != nil {
				slog.Error("Failed to rollback new thread", log.ErrAttr(errRollback))
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			slog.Error("Failed to save new forum message", log.ErrAttr(errSaveMessage))

			return
		}

		if errIncr := f.forums.ForumIncrMessageCount(ctx, forum.ForumID, true); errIncr != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			slog.Error("Failed to increment message count", log.ErrAttr(errIncr))

			return
		}

		ctx.JSON(http.StatusCreated, ThreadWithMessage{
			ForumThread: thread,
			Message:     message,
		})
	}
}

func (f *forumHandler) onAPIThreadUpdate() gin.HandlerFunc {
	type threadUpdate struct {
		Title  string `json:"title"`
		Sticky bool   `json:"sticky"`
		Locked bool   `json:"locked"`
	}

	return func(ctx *gin.Context) {
		currentUser := httphelper.CurrentUserProfile(ctx)

		forumThreadID, errForumTheadID := httphelper.GetInt64Param(ctx, "forum_thread_id")
		if errForumTheadID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get forum_thread_id", log.ErrAttr(errForumTheadID))

			return
		}

		var req threadUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		req.Title = util.SanitizeUGC(req.Title)

		if len(req.Title) < 2 {
			httphelper.HandleErrBadRequest(ctx)

			return
		}

		var thread domain.ForumThread
		if errGet := f.forums.ForumThread(ctx, forumThreadID, &thread); errGet != nil {
			if errors.Is(errGet, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)
				slog.Warn("Requested unknown forum thread", log.ErrAttr(errGet))
			} else {
				httphelper.HandleErrInternal(ctx)
				slog.Error("Failed to get forum thread", log.ErrAttr(errGet))
			}

			return
		}

		if thread.SourceID != currentUser.SteamID && !(currentUser.PermissionLevel >= domain.PModerator) {
			httphelper.HandleErrPermissionDenied(ctx)

			return
		}

		thread.Title = req.Title
		thread.Sticky = req.Sticky
		thread.Locked = req.Locked

		if errDelete := f.forums.ForumThreadSave(ctx, &thread); errDelete != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to update thread", log.ErrAttr(errDelete))

			return
		}

		ctx.JSON(http.StatusOK, thread)
	}
}

func (f *forumHandler) onAPIThreadDelete() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		forumThreadID, errForumTheadID := httphelper.GetInt64Param(ctx, "forum_thread_id")
		if errForumTheadID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get forum_thread_id", log.ErrAttr(errForumTheadID))

			return
		}

		var thread domain.ForumThread
		if errGet := f.forums.ForumThread(ctx, forumThreadID, &thread); errGet != nil {
			if errors.Is(errGet, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)
				slog.Warn("Failed to get thread", slog.Int64("forum_thread_id", forumThreadID))
			} else {
				httphelper.HandleErrInternal(ctx)
				slog.Error("Failed to get thread", slog.Int64("forum_thread_id", forumThreadID))
			}

			return
		}

		if errDelete := f.forums.ForumThreadDelete(ctx, thread.ForumThreadID); errDelete != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to delete thread", log.ErrAttr(errDelete))

			return
		}

		var forum domain.Forum
		if errForum := f.forums.Forum(ctx, thread.ForumID, &forum); errForum != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to load forum", log.ErrAttr(errForum))

			return
		}

		forum.CountThreads--

		if errSave := f.forums.ForumSave(ctx, &forum); errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to save thread count", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (f *forumHandler) onAPIThreadMessageUpdate() gin.HandlerFunc {
	type MessageUpdate struct {
		BodyMD string `json:"body_md"`
	}

	return func(ctx *gin.Context) {
		currentUser := httphelper.CurrentUserProfile(ctx)

		f.forums.Touch(currentUser)

		forumMessageID, errForumMessageID := httphelper.GetInt64Param(ctx, "forum_message_id")
		if errForumMessageID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)
			slog.Warn("Failed to get forum_message_id", log.ErrAttr(errForumMessageID))

			return
		}

		var req MessageUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		var message domain.ForumMessage
		if errMessage := f.forums.ForumMessage(ctx, forumMessageID, &message); errMessage != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to load forum message", log.ErrAttr(errMessage))

			return
		}

		if message.SourceID != currentUser.SteamID && !(currentUser.PermissionLevel >= domain.PModerator) {
			httphelper.HandleErrPermissionDenied(ctx)

			return
		}

		req.BodyMD = util.SanitizeUGC(req.BodyMD)

		if len(req.BodyMD) < 10 {
			httphelper.HandleErrBadRequest(ctx)

			return
		}

		message.BodyMD = req.BodyMD

		if errSave := f.forums.ForumMessageSave(ctx, &message); errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to update forum message", slog.Int64("forum_message_id", message.ForumMessageID))

			return
		}

		ctx.JSON(http.StatusOK, message)
	}
}

func (f *forumHandler) onAPIMessageDelete() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		forumMessageID, errForumMessageID := httphelper.GetInt64Param(ctx, "forum_message_id")
		if errForumMessageID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get forum_message_id", log.ErrAttr(errForumMessageID))

			return
		}

		var message domain.ForumMessage
		if err := f.forums.ForumMessage(ctx, forumMessageID, &message); err != nil {
			if errors.Is(err, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)
				slog.Warn("Cannot get message, does not exist", slog.Int64("forum_message_id", forumMessageID))
			} else {
				httphelper.HandleErrInternal(ctx)
				slog.Warn("Error trying to get message", slog.Int64("forum_message_id", forumMessageID))
			}

			return
		}

		var thread domain.ForumThread
		if err := f.forums.ForumThread(ctx, message.ForumThreadID, &thread); err != nil {
			if errors.Is(err, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)
				slog.Warn("Failed to load forum thread. Does not exist.", slog.Int64("forum_thread_id", message.ForumThreadID))
			} else {
				httphelper.HandleErrInternal(ctx)
				slog.Warn("Failed to load forum thread", slog.Int64("forum_thread_id", message.ForumThreadID), log.ErrAttr(err))
			}

			return
		}

		if thread.Locked {
			httphelper.ResponseErr(ctx, http.StatusForbidden, domain.ErrThreadLocked)

			return
		}

		messages, errMessage := f.forums.ForumMessages(ctx, domain.ThreadMessagesQuery{ForumThreadID: message.ForumThreadID})
		if errMessage != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to fetch forum messages", log.ErrAttr(errMessage))

			return
		}

		isThreadParent := messages[0].ForumMessageID == message.ForumMessageID

		if isThreadParent {
			if err := f.forums.ForumThreadDelete(ctx, message.ForumThreadID); err != nil {
				httphelper.HandleErrInternal(ctx)
				slog.Error("Failed to delete forum thread", log.ErrAttr(err), slog.Int64("forum_thread_id", message.ForumThreadID))

				return
			}

			// Delete the thread if it's the first message
			var forum domain.Forum
			if errForum := f.forums.Forum(ctx, thread.ForumID, &forum); errForum != nil {
				httphelper.HandleErrInternal(ctx)
				slog.Error("Failed to load forum", log.ErrAttr(errForum), slog.Int("forum_id", thread.ForumID))

				return
			}

			forum.CountThreads--

			if errSave := f.forums.ForumSave(ctx, &forum); errSave != nil {
				httphelper.HandleErrInternal(ctx)
				slog.Error("Failed to save thread count", log.ErrAttr(errSave))

				return
			}

			slog.Error("Thread deleted due to parent deletion", slog.Int64("forum_thread_id", thread.ForumThreadID))
		} else {
			if errDelete := f.forums.ForumMessageDelete(ctx, message.ForumMessageID); errDelete != nil {
				httphelper.HandleErrInternal(ctx)
				slog.Error("Failed to delete message", log.ErrAttr(errDelete))

				return
			}
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (f *forumHandler) onAPIThreadCreateReply() gin.HandlerFunc {
	type ThreadReply struct {
		BodyMD string `json:"body_md"`
	}

	return func(ctx *gin.Context) {
		currentUser := httphelper.CurrentUserProfile(ctx)

		f.forums.Touch(currentUser)

		forumThreadID, errForumID := httphelper.GetInt64Param(ctx, "forum_thread_id")
		if errForumID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get forum_thread_id", log.ErrAttr(errForumID))

			return
		}

		var thread domain.ForumThread
		if errThread := f.forums.ForumThread(ctx, forumThreadID, &thread); errThread != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to load forum thread", log.ErrAttr(errThread), slog.Int64("forum_thread_id", forumThreadID))

			return
		}

		if thread.Locked && currentUser.PermissionLevel < domain.PEditor {
			httphelper.ResponseErr(ctx, http.StatusForbidden, domain.ErrThreadLocked)

			return
		}

		var req ThreadReply
		if !httphelper.Bind(ctx, &req) {
			return
		}

		req.BodyMD = util.SanitizeUGC(req.BodyMD)

		if len(req.BodyMD) < 3 {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, fmt.Errorf("body: %w", domain.ErrTooShort))

			return
		}

		newMessage := thread.NewMessage(currentUser.SteamID, req.BodyMD)
		if errSave := f.forums.ForumMessageSave(ctx, &newMessage); errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to same forum message", log.ErrAttr(errSave))

			return
		}

		var message domain.ForumMessage
		if errFetch := f.forums.ForumMessage(ctx, newMessage.ForumMessageID, &message); errFetch != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to load forum message", log.ErrAttr(errFetch))

			return
		}

		if errIncr := f.forums.ForumIncrMessageCount(ctx, thread.ForumID, true); errIncr != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to increment message count", log.ErrAttr(errIncr))
		}

		newMessage.Personaname = currentUser.Name
		newMessage.Avatarhash = currentUser.Avatarhash
		newMessage.PermissionLevel = currentUser.PermissionLevel
		newMessage.Online = true

		ctx.JSON(http.StatusCreated, newMessage)
	}
}

func (f *forumHandler) onAPIForumOverview() gin.HandlerFunc {
	type Overview struct {
		Categories []domain.ForumCategory `json:"categories"`
	}

	return func(ctx *gin.Context) {
		currentUser := httphelper.CurrentUserProfile(ctx)

		f.forums.Touch(currentUser)

		categories, errCats := f.forums.ForumCategories(ctx)
		if errCats != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Could not load categories", log.ErrAttr(errCats))

			return
		}

		currentForums, errForums := f.forums.Forums(ctx)
		if errForums != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Could not load forums", log.ErrAttr(errForums))

			return
		}

		for index := range categories {
			for _, forum := range currentForums {
				if currentUser.PermissionLevel < forum.PermissionLevel {
					continue
				}

				if categories[index].ForumCategoryID == forum.ForumCategoryID {
					categories[index].Forums = append(categories[index].Forums, forum)
				}
			}

			if categories[index].Forums == nil {
				categories[index].Forums = []domain.Forum{}
			}
		}

		ctx.JSON(http.StatusOK, Overview{Categories: categories})
	}
}

func (f *forumHandler) onAPIForumThreads() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser := httphelper.CurrentUserProfile(ctx)

		f.forums.Touch(currentUser)

		var tqf domain.ThreadQueryFilter
		if !httphelper.Bind(ctx, &tqf) {
			return
		}

		threads, errThreads := f.forums.ForumThreads(ctx, tqf)
		if errThreads != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Could not load threads", log.ErrAttr(errThreads), slog.Int("forum_id", tqf.ForumID))

			return
		}

		var forum domain.Forum
		if err := f.forums.Forum(ctx, tqf.ForumID, &forum); err != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Could not load forum", log.ErrAttr(errThreads), slog.Int("forum_id", tqf.ForumID))

			return
		}

		if forum.PermissionLevel > currentUser.PermissionLevel {
			httphelper.HandleErrPermissionDenied(ctx)
			slog.Error("User does not have access to forum", slog.Int("forum_id", tqf.ForumID))

			return
		}

		ctx.JSON(http.StatusOK, threads)
	}
}

func (f *forumHandler) onAPIForumThread() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser := httphelper.CurrentUserProfile(ctx)

		f.forums.Touch(currentUser)

		forumThreadID, errID := httphelper.GetInt64Param(ctx, "forum_thread_id")
		if errID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get forum_thread_id", log.ErrAttr(errID))

			return
		}

		var thread domain.ForumThread
		if errThreads := f.forums.ForumThread(ctx, forumThreadID, &thread); errThreads != nil {
			if errors.Is(errThreads, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)
			} else {
				httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
				slog.Error("Could not load threads", log.ErrAttr(errThreads), slog.Int64("forum_thread_id", forumThreadID))
			}

			return
		}

		ctx.JSON(http.StatusOK, thread)

		if err := f.forums.ForumThreadIncrView(ctx, forumThreadID); err != nil {
			slog.Error("Failed to increment thread view count", log.ErrAttr(err), slog.Int64("forum_thread_id", forumThreadID))
		}
	}
}

func (f *forumHandler) onAPIForum() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser := httphelper.CurrentUserProfile(ctx)

		forumID, errForumID := httphelper.GetIntParam(ctx, "forum_id")
		if errForumID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get forum_id", log.ErrAttr(errForumID))

			return
		}

		var forum domain.Forum

		if errForum := f.forums.Forum(ctx, forumID, &forum); errForum != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Could not load forum", log.ErrAttr(errForum), slog.Int("forum_id", forumID))

			return
		}

		if forum.PermissionLevel > currentUser.PermissionLevel {
			httphelper.HandleErrPermissionDenied(ctx)

			return
		}

		ctx.JSON(http.StatusOK, forum)
	}
}

func (f *forumHandler) onAPIForumMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var queryFilter domain.ThreadMessagesQuery
		if !httphelper.Bind(ctx, &queryFilter) {
			return
		}

		messages, errMessages := f.forums.ForumMessages(ctx, queryFilter)
		if errMessages != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Could not load thread messages", log.ErrAttr(errMessages))

			return
		}

		activeUsers := f.forums.Current()

		for idx := range messages {
			for _, activity := range activeUsers {
				if messages[idx].SourceID == activity.Person.SteamID {
					messages[idx].Online = true

					break
				}
			}
		}

		ctx.JSON(http.StatusOK, messages)
	}
}

func (f *forumHandler) onAPIActiveUsers() gin.HandlerFunc {
	type userActivity struct {
		SteamID         steamid.SteamID  `json:"steam_id"`
		Personaname     string           `json:"personaname"`
		PermissionLevel domain.Privilege `json:"permission_level"`
		CreatedOn       time.Time        `json:"created_on"`
	}

	return func(ctx *gin.Context) {
		var results []userActivity

		for _, act := range f.forums.Current() {
			results = append(results, userActivity{
				SteamID:         act.Person.SteamID,
				Personaname:     act.Person.Name,
				PermissionLevel: act.Person.PermissionLevel,
				CreatedOn:       act.LastActivity,
			})
		}

		ctx.JSON(http.StatusOK, results)
	}
}
