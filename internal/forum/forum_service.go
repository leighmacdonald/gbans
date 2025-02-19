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
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type forumHandler struct {
	forums domain.ForumUsecase
}

func NewHandler(engine *gin.Engine, forums domain.ForumUsecase, auth domain.AuthUsecase) {
	handler := &forumHandler{
		forums: forums,
	}

	engine.GET("/api/forum/active_users", handler.onAPIActiveUsers())

	// opt
	optGrp := engine.Group("/")
	{
		opt := optGrp.Use(auth.Middleware(domain.PGuest))
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
		authed := authedGrp.Use(auth.Middleware(domain.PUser))
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
		mod := modGrp.Use(auth.Middleware(domain.PModerator))
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
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errThreads))

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
			Title:       stringutil.SanitizeUGC(req.Title),
			Description: stringutil.SanitizeUGC(req.Description),
			Ordering:    req.Ordering,
			TimeStamped: domain.NewTimeStamped(),
		}

		if errSave := f.forums.ForumCategorySave(ctx, &category); errSave != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSave))

			return
		}

		ctx.JSON(http.StatusCreated, category)
	}
}

func (f *forumHandler) onAPIForumCategory() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		forumCategoryID, idFound := httphelper.GetIntParam(ctx, "forum_category_id")
		if !idFound {
			return
		}

		var category domain.ForumCategory

		if errGet := f.forums.ForumCategory(ctx, forumCategoryID, &category); errGet != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errGet))

			return
		}

		ctx.JSON(http.StatusOK, category)
	}
}

func (f *forumHandler) onAPIUpdateForumCategory() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		categoryID, idFound := httphelper.GetIntParam(ctx, "forum_category_id")
		if !idFound {
			return
		}

		var category domain.ForumCategory
		if errGet := f.forums.ForumCategory(ctx, categoryID, &category); errGet != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errGet))

			return
		}

		var req CategoryRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		category.Title = stringutil.SanitizeUGC(req.Title)
		category.Description = stringutil.SanitizeUGC(req.Description)
		category.Ordering = req.Ordering

		if errSave := f.forums.ForumCategorySave(ctx, &category); errSave != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSave))

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
			Title:           stringutil.SanitizeUGC(req.Title),
			Description:     stringutil.SanitizeUGC(req.Description),
			Ordering:        req.Ordering,
			PermissionLevel: req.PermissionLevel,
			TimeStamped:     domain.NewTimeStamped(),
		}

		if errSave := f.forums.ForumSave(ctx, &forum); errSave != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSave))

			return
		}

		ctx.JSON(http.StatusCreated, forum)
	}
}

func (f *forumHandler) onAPIUpdateForumForum() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		forumID, idFound := httphelper.GetIntParam(ctx, "forum_id")
		if !idFound {
			return
		}

		var forum domain.Forum
		if errGet := f.forums.Forum(ctx, forumID, &forum); errGet != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errGet))

			return
		}

		var req CreateForumRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		forum.ForumCategoryID = req.ForumCategoryID
		forum.Title = stringutil.SanitizeUGC(req.Title)
		forum.Description = stringutil.SanitizeUGC(req.Description)
		forum.Ordering = req.Ordering
		forum.PermissionLevel = req.PermissionLevel

		if errSave := f.forums.ForumSave(ctx, &forum); errSave != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSave))

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

		forumID, idFound := httphelper.GetIntParam(ctx, "forum_id")
		if !idFound {
			return
		}

		var req CreateThreadRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if len(req.BodyMD) <= 1 {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusBadRequest, fmt.Errorf("body: %w", domain.ErrTooShort)))

			return
		}

		if len(req.Title) <= 4 {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusBadRequest, fmt.Errorf("title: %w", domain.ErrTooShort)))

			return
		}

		var forum domain.Forum
		if errForum := f.forums.Forum(ctx, forumID, &forum); errForum != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errForum))

			return
		}

		thread := forum.NewThread(req.Title, user.SteamID)
		thread.Sticky = req.Sticky
		thread.Locked = req.Locked

		if errSaveThread := f.forums.ForumThreadSave(ctx, &thread); errSaveThread != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSaveThread))

			return
		}

		message := thread.NewMessage(user.SteamID, req.BodyMD)

		if errSaveMessage := f.forums.ForumMessageSave(ctx, &message); errSaveMessage != nil {
			// Drop created thread.
			// TODO transaction
			if errRollback := f.forums.ForumThreadDelete(ctx, thread.ForumThreadID); errRollback != nil {
				slog.Error("Failed to rollback new thread", log.ErrAttr(errRollback))
			}

			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSaveMessage))

			slog.Error("Failed to save new forum message", log.ErrAttr(errSaveMessage))

			return
		}

		if errIncr := f.forums.ForumIncrMessageCount(ctx, forum.ForumID, true); errIncr != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errIncr))

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

		forumThreadID, idFound := httphelper.GetInt64Param(ctx, "forum_thread_id")
		if !idFound {
			return
		}

		var req threadUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		req.Title = stringutil.SanitizeUGC(req.Title)

		if len(req.Title) < 2 {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusBadRequest, domain.ErrBadRequest))

			return
		}

		var thread domain.ForumThread
		if errGet := f.forums.ForumThread(ctx, forumThreadID, &thread); errGet != nil {
			if errors.Is(errGet, domain.ErrNoResult) {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusNotFound, domain.ErrNotFound))
			} else {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errGet))
			}

			return
		}

		if thread.SourceID != currentUser.SteamID && !(currentUser.PermissionLevel >= domain.PModerator) {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusForbidden, domain.ErrPermissionDenied))

			return
		}

		thread.Title = req.Title
		thread.Sticky = req.Sticky
		thread.Locked = req.Locked

		if errDelete := f.forums.ForumThreadSave(ctx, &thread); errDelete != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errDelete))

			return
		}

		ctx.JSON(http.StatusOK, thread)
	}
}

func (f *forumHandler) onAPIThreadDelete() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		forumThreadID, idFound := httphelper.GetInt64Param(ctx, "forum_thread_id")
		if !idFound {
			return
		}

		var thread domain.ForumThread
		if errGet := f.forums.ForumThread(ctx, forumThreadID, &thread); errGet != nil {
			if errors.Is(errGet, domain.ErrNoResult) {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusNotFound, domain.ErrNotFound))
			} else {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errGet))
			}

			return
		}

		if errDelete := f.forums.ForumThreadDelete(ctx, thread.ForumThreadID); errDelete != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errDelete))

			return
		}

		var forum domain.Forum
		if errForum := f.forums.Forum(ctx, thread.ForumID, &forum); errForum != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errForum))

			return
		}

		forum.CountThreads--

		if errSave := f.forums.ForumSave(ctx, &forum); errSave != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSave))

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
		if !errForumMessageID {
			return
		}

		var req MessageUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		var message domain.ForumMessage
		if errMessage := f.forums.ForumMessage(ctx, forumMessageID, &message); errMessage != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errMessage))

			return
		}

		if message.SourceID != currentUser.SteamID && !(currentUser.PermissionLevel >= domain.PModerator) {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusForbidden, domain.ErrPermissionDenied))

			return
		}

		req.BodyMD = stringutil.SanitizeUGC(req.BodyMD)

		if len(req.BodyMD) < 10 {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusBadRequest, domain.ErrTooShort))

			return
		}

		message.BodyMD = req.BodyMD

		if errSave := f.forums.ForumMessageSave(ctx, &message); errSave != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSave))

			return
		}

		ctx.JSON(http.StatusOK, message)
	}
}

func (f *forumHandler) onAPIMessageDelete() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		forumMessageID, idFound := httphelper.GetInt64Param(ctx, "forum_message_id")
		if !idFound {
			return
		}

		var message domain.ForumMessage
		if err := f.forums.ForumMessage(ctx, forumMessageID, &message); err != nil {
			if errors.Is(err, domain.ErrNoResult) {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, domain.ErrNotFound))
			} else {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, err))
			}

			return
		}

		var thread domain.ForumThread
		if err := f.forums.ForumThread(ctx, message.ForumThreadID, &thread); err != nil {
			if errors.Is(err, domain.ErrNoResult) {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, domain.ErrNotFound))
			} else {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, err))
			}

			return
		}

		if thread.Locked {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusForbidden, domain.ErrThreadLocked))

			return
		}

		messages, errMessage := f.forums.ForumMessages(ctx, domain.ThreadMessagesQuery{ForumThreadID: message.ForumThreadID})
		if errMessage != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errMessage))

			return
		}

		isThreadParent := messages[0].ForumMessageID == message.ForumMessageID

		if isThreadParent {
			if err := f.forums.ForumThreadDelete(ctx, message.ForumThreadID); err != nil {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, err))

				return
			}

			// Delete the thread if it's the first message
			var forum domain.Forum
			if errForum := f.forums.Forum(ctx, thread.ForumID, &forum); errForum != nil {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errForum))

				return
			}

			forum.CountThreads--

			if errSave := f.forums.ForumSave(ctx, &forum); errSave != nil {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSave))

				return
			}

			slog.Error("Thread deleted due to parent deletion", slog.Int64("forum_thread_id", thread.ForumThreadID))
		} else {
			if errDelete := f.forums.ForumMessageDelete(ctx, message.ForumMessageID); errDelete != nil {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errDelete))

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

		forumThreadID, idFound := httphelper.GetInt64Param(ctx, "forum_thread_id")
		if !idFound {
			return
		}

		var thread domain.ForumThread
		if errThread := f.forums.ForumThread(ctx, forumThreadID, &thread); errThread != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errThread))

			return
		}

		if thread.Locked && currentUser.PermissionLevel < domain.PEditor {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusForbidden, domain.ErrThreadLocked))

			return
		}

		var req ThreadReply
		if !httphelper.Bind(ctx, &req) {
			return
		}

		req.BodyMD = stringutil.SanitizeUGC(req.BodyMD)

		if len(req.BodyMD) < 3 {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusBadRequest, fmt.Errorf("body: %w", domain.ErrTooShort)))

			return
		}

		newMessage := thread.NewMessage(currentUser.SteamID, req.BodyMD)
		if errSave := f.forums.ForumMessageSave(ctx, &newMessage); errSave != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSave))

			return
		}

		var message domain.ForumMessage
		if errFetch := f.forums.ForumMessage(ctx, newMessage.ForumMessageID, &message); errFetch != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errFetch))

			return
		}

		if errIncr := f.forums.ForumIncrMessageCount(ctx, thread.ForumID, true); errIncr != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errIncr))

			return
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
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errCats))

			return
		}

		currentForums, errForums := f.forums.Forums(ctx)
		if errForums != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errForums))

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

		if categories == nil {
			categories = []domain.ForumCategory{}
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
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errThreads))

			return
		}

		var forum domain.Forum
		if err := f.forums.Forum(ctx, tqf.ForumID, &forum); err != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, err))

			return
		}

		if forum.PermissionLevel > currentUser.PermissionLevel {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusForbidden, domain.ErrPermissionDenied))

			return
		}

		ctx.JSON(http.StatusOK, threads)
	}
}

func (f *forumHandler) onAPIForumThread() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser := httphelper.CurrentUserProfile(ctx)

		f.forums.Touch(currentUser)

		forumThreadID, idFound := httphelper.GetInt64Param(ctx, "forum_thread_id")
		if !idFound {
			return
		}

		var thread domain.ForumThread
		if errThreads := f.forums.ForumThread(ctx, forumThreadID, &thread); errThreads != nil {
			if errors.Is(errThreads, domain.ErrNoResult) {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusNotFound, domain.ErrNotFound))
			} else {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errThreads))
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

		forumID, idFound := httphelper.GetIntParam(ctx, "forum_id")
		if !idFound {
			return
		}

		var forum domain.Forum

		if errForum := f.forums.Forum(ctx, forumID, &forum); errForum != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errForum))

			return
		}

		if forum.PermissionLevel > currentUser.PermissionLevel {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusForbidden, domain.ErrPermissionDenied))

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
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errMessages))

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
