package forum

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type forumHandler struct {
	Forums
}

func NewForumHandler(engine *gin.Engine, forums Forums, authenticator httphelper.Authenticator) {
	handler := &forumHandler{Forums: forums}

	engine.GET("/api/forum/active_users", handler.onAPIActiveUsers())

	// opt
	optGrp := engine.Group("/")
	{
		opt := optGrp.Use(authenticator.Middleware(permission.Guest))
		opt.GET("/api/forum/overview", handler.overview())
		opt.GET("/api/forum/messages/recent", handler.recentMessages())
		opt.POST("/api/forum/threads", handler.threads())
		opt.GET("/api/forum/thread/:forum_thread_id", handler.thread())
		opt.GET("/api/forum/forum/:forum_id", handler.forum())
		opt.POST("/api/forum/messages", handler.messages())
	}

	// auth
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(authenticator.Middleware(permission.User))
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
		mod := modGrp.Use(authenticator.Middleware(permission.Moderator))
		// TODO delete category handler
		// TODO delete forum handler
		mod.POST("/api/forum/category", handler.onAPICreateForumCategory())
		mod.GET("/api/forum/category/:forum_category_id", handler.onAPIForumCategory())
		mod.POST("/api/forum/category/:forum_category_id", handler.onAPIUpdateForumCategory())
		mod.POST("/api/forum/forum", handler.onAPICreateForum())
		mod.POST("/api/forum/forum/:forum_id", handler.onAPIUpdateForumForum())
	}
}

type CategoryRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Ordering    int    `json:"ordering"`
}

func (f *forumHandler) recentMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, _ := session.CurrentUserProfile(ctx)
		messages, errThreads := f.RecentActivity(ctx, 5, user.Permissions())
		if errThreads != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errThreads, httphelper.ErrInternal),
				"Could not load recent forum activity"))

			return
		}

		if messages == nil {
			messages = []Message{}
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

		category := Category{
			Title:       stringutil.SanitizeUGC(req.Title),
			Description: stringutil.SanitizeUGC(req.Description),
			Ordering:    req.Ordering,
			CreatedOn:   time.Now(),
			UpdatedOn:   time.Now(),
		}

		if errSave := f.CategorySave(ctx, &category); errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal),
				"Failed to create forum category."))

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

		var category Category

		if errGet := f.Category(ctx, forumCategoryID, &category); errGet != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errGet, httphelper.ErrInternal),
				"Could not load forum category with id: %d", forumCategoryID))

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

		var category Category
		if errGet := f.Category(ctx, categoryID, &category); errGet != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errGet, httphelper.ErrInternal),
				"Failed to load existing category"))

			return
		}

		var req CategoryRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		category.Title = stringutil.SanitizeUGC(req.Title)
		category.Description = stringutil.SanitizeUGC(req.Description)
		category.Ordering = req.Ordering

		if errSave := f.CategorySave(ctx, &category); errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal),
				"Failed to save forum category."))

			return
		}

		ctx.JSON(http.StatusOK, category)
	}
}

type CreateForumRequest struct {
	ForumCategoryID int                  `json:"forum_category_id"`
	PermissionLevel permission.Privilege `json:"permission_level"`
	Title           string               `json:"title"`
	Description     string               `json:"description"`
	Ordering        int                  `json:"ordering"`
}

func (f *forumHandler) onAPICreateForum() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req CreateForumRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		forum := Forum{
			ForumCategoryID: req.ForumCategoryID,
			Title:           stringutil.SanitizeUGC(req.Title),
			Description:     stringutil.SanitizeUGC(req.Description),
			Ordering:        req.Ordering,
			PermissionLevel: req.PermissionLevel,
			CreatedOn:       time.Now(),
			UpdatedOn:       time.Now(),
		}

		if errSave := f.ForumSave(ctx, &forum); errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal),
				"Failed to save new forum"))

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

		var forum Forum
		if errGet := f.Forum(ctx, forumID, &forum); errGet != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errGet, httphelper.ErrInternal)))

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

		if errSave := f.ForumSave(ctx, &forum); errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal),
				"Could not save changed to forum"))

			return
		}

		ctx.JSON(http.StatusOK, forum)
	}
}

type CreateThreadRequest struct {
	Title  string `json:"title"`
	BodyMD string `json:"body_md"`
	Sticky bool   `json:"sticky"`
	Locked bool   `json:"locked"`
}

type ThreadWithMessage struct {
	Thread
	Message Message `json:"message"`
}

func (f *forumHandler) onAPIThreadCreate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, _ := session.CurrentUserProfile(ctx)

		f.Touch(user)

		forumID, idFound := httphelper.GetIntParam(ctx, "forum_id")
		if !idFound {
			return
		}

		var req CreateThreadRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if len(req.BodyMD) <= 1 {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrBadRequest,
				"Body must be >1 characters"))

			return
		}

		if len(req.Title) <= 4 {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrBadRequest,
				"Title must be >4 characters"))

			return
		}

		var forum Forum
		if errForum := f.Forum(ctx, forumID, &forum); errForum != nil {
			switch {
			case errors.Is(errForum, database.ErrNoResult):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errForum, httphelper.ErrInternal),
					"The forum_id provided does not exist: %d", forumID))
			default:
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errForum, httphelper.ErrInternal),
					"Failed to fetch forum with forum_id: %d", forumID))
			}

			return
		}

		thread := forum.NewThread(req.Title, user.GetSteamID())
		thread.Sticky = req.Sticky
		thread.Locked = req.Locked

		if errSaveThread := f.ThreadSave(ctx, &thread); errSaveThread != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errSaveThread, httphelper.ErrInternal),
				"Failed to save new thread."))

			return
		}

		message := thread.NewMessage(user.GetSteamID(), req.BodyMD)
		if errSaveMessage := f.MessageSave(ctx, &message); errSaveMessage != nil {
			// Drop created thread.
			// TODO transaction
			if errRollback := f.ThreadDelete(ctx, thread.ForumThreadID); errRollback != nil {
				slog.Error("Failed to rollback new thread", log.ErrAttr(errRollback))
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSaveMessage, httphelper.ErrInternal)))

			slog.Error("Failed to save new forum message", log.ErrAttr(errSaveMessage))

			return
		}

		if errIncr := f.ForumIncrMessageCount(ctx, forum.ForumID, true); errIncr != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errIncr, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusCreated, ThreadWithMessage{
			Thread:  thread,
			Message: message,
		})
	}
}

type ThreadUpdate struct {
	Title  string `json:"title"`
	Sticky bool   `json:"sticky"`
	Locked bool   `json:"locked"`
}

func (f *forumHandler) onAPIThreadUpdate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser, _ := session.CurrentUserProfile(ctx)

		forumThreadID, idFound := httphelper.GetInt64Param(ctx, "forum_thread_id")
		if !idFound {
			return
		}

		var req ThreadUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		req.Title = stringutil.SanitizeUGC(req.Title)

		if len(req.Title) < 2 {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrBadRequest,
				"Title must be at least 2 characters long."))

			return
		}

		var thread Thread
		if errGet := f.Thread(ctx, forumThreadID, &thread); errGet != nil {
			if errors.Is(errGet, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusNotFound, httphelper.ErrNotFound,
					"Forum thread does not exist: %d", forumThreadID))
			} else {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errGet, httphelper.ErrInternal),
					"Failed to load existing forum thread."))
			}

			return
		}

		if thread.SourceID != currentUser.GetSteamID() && !currentUser.HasPermission(permission.Moderator) {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusForbidden, httphelper.ErrPermissionDenied,
				"You do not have access to edit this."))

			return
		}

		thread.Title = req.Title
		thread.Sticky = req.Sticky
		thread.Locked = req.Locked

		if errDelete := f.ThreadSave(ctx, &thread); errDelete != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errDelete, httphelper.ErrInternal),
				"Failed to save thread."))

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

		var thread Thread
		if errGet := f.Thread(ctx, forumThreadID, &thread); errGet != nil {
			switch {
			case errors.Is(errGet, database.ErrNoResult):
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, httphelper.ErrNotFound))
			default:
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errGet, httphelper.ErrInternal)))
			}

			return
		}

		if errDelete := f.ThreadDelete(ctx, thread.ForumThreadID); errDelete != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errDelete, httphelper.ErrInternal),
				"Could not delete forum thread: forum_thread_id=%d", forumThreadID))

			return
		}

		var forum Forum
		if errForum := f.Forum(ctx, thread.ForumID, &forum); errForum != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errForum, httphelper.ErrInternal)))

			return
		}

		forum.CountThreads--

		if errSave := f.ForumSave(ctx, &forum); errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal)))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

type MessageUpdate struct {
	BodyMD string `json:"body_md"`
}

func (f *forumHandler) onAPIThreadMessageUpdate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser, _ := session.CurrentUserProfile(ctx)

		f.Touch(currentUser)

		forumMessageID, errForumMessageID := httphelper.GetInt64Param(ctx, "forum_message_id")
		if !errForumMessageID {
			return
		}

		var req MessageUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		var message Message
		if errMessage := f.Message(ctx, forumMessageID, &message); errMessage != nil {
			switch {
			case errors.Is(errMessage, database.ErrNoResult):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusNotFound, errors.Join(errMessage, httphelper.ErrNotFound),
					"Message not found, cannot update."))
			default:
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errMessage, httphelper.ErrInternal)))
			}

			return
		}

		if message.SourceID != currentUser.GetSteamID() && !currentUser.HasPermission(permission.Moderator) {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusForbidden, httphelper.ErrPermissionDenied,
				"You do not have permission to edit this message."))

			return
		}

		req.BodyMD = stringutil.SanitizeUGC(req.BodyMD)

		if len(req.BodyMD) < 10 {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrTooShort,
				"Body must be at least 10 characters."))

			return
		}

		message.BodyMD = req.BodyMD

		if errSave := f.MessageSave(ctx, &message); errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal),
				"Could not save forum message"))

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

		var message Message
		if err := f.Message(ctx, forumMessageID, &message); err != nil {
			if errors.Is(err, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, httphelper.ErrNotFound, "Forum message does not exist"))
			} else {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))
			}

			return
		}

		var thread Thread
		if err := f.Thread(ctx, message.ForumThreadID, &thread); err != nil {
			if errors.Is(err, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, httphelper.ErrNotFound))
			} else {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))
			}

			return
		}

		if thread.Locked {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusForbidden, ErrThreadLocked))

			return
		}

		messages, errMessage := f.Messages(ctx, ThreadMessagesQuery{ForumThreadID: message.ForumThreadID})
		if errMessage != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errMessage, httphelper.ErrInternal)))

			return
		}

		isThreadParent := messages[0].ForumMessageID == message.ForumMessageID

		if isThreadParent {
			if err := f.ThreadDelete(ctx, message.ForumThreadID); err != nil {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

				return
			}

			// Delete the thread if it's the first message
			var forum Forum
			if errForum := f.Forum(ctx, thread.ForumID, &forum); errForum != nil {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errForum, httphelper.ErrInternal)))

				return
			}

			forum.CountThreads--

			if errSave := f.ForumSave(ctx, &forum); errSave != nil {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal)))

				return
			}

			slog.Error("Thread deleted due to parent deletion", slog.Int64("forum_thread_id", thread.ForumThreadID))
		} else {
			if errDelete := f.MessageDelete(ctx, message.ForumMessageID); errDelete != nil {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errDelete, httphelper.ErrInternal)))

				return
			}
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

type ThreadReply struct {
	BodyMD string `json:"body_md"`
}

func (f *forumHandler) onAPIThreadCreateReply() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser, _ := session.CurrentUserProfile(ctx)

		f.Touch(currentUser)

		forumThreadID, idFound := httphelper.GetInt64Param(ctx, "forum_thread_id")
		if !idFound {
			return
		}

		var thread Thread
		if errThread := f.Thread(ctx, forumThreadID, &thread); errThread != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errThread, httphelper.ErrInternal)))

			return
		}

		if thread.Locked && !currentUser.HasPermission(permission.Editor) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusForbidden, ErrThreadLocked))

			return
		}

		var req ThreadReply
		if !httphelper.Bind(ctx, &req) {
			return
		}

		req.BodyMD = stringutil.SanitizeUGC(req.BodyMD)

		if len(req.BodyMD) < 3 {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrBadRequest,
				"Body must be at least 3 characters."))

			return
		}

		newMessage := thread.NewMessage(currentUser.GetSteamID(), req.BodyMD)
		if errSave := f.MessageSave(ctx, &newMessage); errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal)))

			return
		}

		var message Message
		if errFetch := f.Message(ctx, newMessage.ForumMessageID, &message); errFetch != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errFetch, httphelper.ErrInternal)))

			return
		}

		if errIncr := f.ForumIncrMessageCount(ctx, thread.ForumID, true); errIncr != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errIncr, httphelper.ErrInternal)))

			return
		}

		newMessage.Personaname = currentUser.GetName()
		newMessage.Avatarhash = currentUser.GetAvatar().Hash()
		newMessage.PermissionLevel = currentUser.Permissions()
		newMessage.Online = true

		ctx.JSON(http.StatusCreated, newMessage)
	}
}

type Overview struct {
	Categories []Category `json:"categories"`
}

func (f *forumHandler) overview() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser, _ := session.CurrentUserProfile(ctx)

		f.Touch(currentUser)

		categories, errCats := f.Categories(ctx)
		if errCats != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errCats, httphelper.ErrInternal)))

			return
		}

		currentForums, errForums := f.Forums.Forums(ctx)
		if errForums != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errForums, httphelper.ErrInternal)))

			return
		}

		for index := range categories {
			for _, forum := range currentForums {
				if !currentUser.HasPermission(forum.PermissionLevel) {
					continue
				}

				if categories[index].ForumCategoryID == forum.ForumCategoryID {
					categories[index].Forums = append(categories[index].Forums, forum)
				}
			}

			if categories[index].Forums == nil {
				categories[index].Forums = []Forum{}
			}
		}

		if categories == nil {
			categories = []Category{}
		}

		ctx.JSON(http.StatusOK, Overview{Categories: categories})
	}
}

func (f *forumHandler) threads() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser, _ := session.CurrentUserProfile(ctx)

		f.Touch(currentUser)

		var tqf ThreadQueryFilter
		if !httphelper.Bind(ctx, &tqf) {
			return
		}

		threads, errThreads := f.Threads(ctx, tqf)
		if errThreads != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errThreads, httphelper.ErrInternal)))

			return
		}

		var forum Forum
		if err := f.Forum(ctx, tqf.ForumID, &forum); err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal)))

			return
		}

		if !currentUser.HasPermission(forum.PermissionLevel) {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusForbidden, httphelper.ErrPermissionDenied,
				"You do not have permission to access this forum."))

			return
		}

		ctx.JSON(http.StatusOK, threads)
	}
}

func (f *forumHandler) thread() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser, _ := session.CurrentUserProfile(ctx)

		f.Touch(currentUser)

		forumThreadID, idFound := httphelper.GetInt64Param(ctx, "forum_thread_id")
		if !idFound {
			return
		}

		var thread Thread
		if errThreads := f.Thread(ctx, forumThreadID, &thread); errThreads != nil {
			if errors.Is(errThreads, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, httphelper.ErrNotFound))
			} else {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errThreads, httphelper.ErrInternal)))
			}

			return
		}

		ctx.JSON(http.StatusOK, thread)

		if err := f.ThreadIncrView(ctx, forumThreadID); err != nil {
			slog.Error("Failed to increment thread view count", log.ErrAttr(err), slog.Int64("forum_thread_id", forumThreadID))
		}
	}
}

func (f *forumHandler) forum() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser, _ := session.CurrentUserProfile(ctx)

		forumID, idFound := httphelper.GetIntParam(ctx, "forum_id")
		if !idFound {
			return
		}

		var forum Forum

		if errForum := f.Forum(ctx, forumID, &forum); errForum != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errForum, httphelper.ErrInternal)))

			return
		}

		if !currentUser.HasPermission(forum.PermissionLevel) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusForbidden, httphelper.ErrPermissionDenied))

			return
		}

		ctx.JSON(http.StatusOK, forum)
	}
}

func (f *forumHandler) messages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var queryFilter ThreadMessagesQuery
		if !httphelper.Bind(ctx, &queryFilter) {
			return
		}

		messages, errMessages := f.Messages(ctx, queryFilter)
		if errMessages != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errMessages, httphelper.ErrInternal)))

			return
		}

		activeUsers := f.Current()

		for idx := range messages {
			for _, activity := range activeUsers {
				if messages[idx].SourceID == activity.Person.GetSteamID() {
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
		SteamID         steamid.SteamID      `json:"steam_id"`
		Personaname     string               `json:"personaname"`
		PermissionLevel permission.Privilege `json:"permission_level"`
		CreatedOn       time.Time            `json:"created_on"`
	}

	return func(ctx *gin.Context) {
		var results []userActivity

		for _, act := range f.Current() {
			results = append(results, userActivity{
				SteamID:         act.Person.GetSteamID(),
				Personaname:     act.Person.GetName(),
				PermissionLevel: act.Person.Permissions(),
				CreatedOn:       act.LastActivity,
			})
		}

		ctx.JSON(http.StatusOK, results)
	}
}
