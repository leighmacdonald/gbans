package forum

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	v1 "github.com/leighmacdonald/gbans/internal/forum/v1"
	"github.com/leighmacdonald/gbans/internal/forum/v1/forumv1connect"
	personv1 "github.com/leighmacdonald/gbans/internal/person/v1"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	forumv1connect.UnimplementedForumServiceHandler

	forums Forums
}

func NewService(forums Forums, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := forumv1connect.NewForumServiceHandler(Service{forums: forums}, option...)

	authMiddleware.UserRoute(forumv1connect.ForumServiceActiveUsersProcedure, rpc.WithMinPermissions(permission.Guest))
	authMiddleware.UserRoute(forumv1connect.ForumServiceOverviewProcedure, rpc.WithMinPermissions(permission.Guest))
	authMiddleware.UserRoute(forumv1connect.ForumServiceRecentMessagesProcedure, rpc.WithMinPermissions(permission.Guest))
	authMiddleware.UserRoute(forumv1connect.ForumServiceThreadsProcedure, rpc.WithMinPermissions(permission.Guest))
	authMiddleware.UserRoute(forumv1connect.ForumServiceThreadProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(forumv1connect.ForumServiceThreadDeleteProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(forumv1connect.ForumServiceForumProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(forumv1connect.ForumServiceThreadMessagesProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(forumv1connect.ForumServiceThreadCreateProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(forumv1connect.ForumServiceThreadEditProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(forumv1connect.ForumServiceThreadReplyCreateProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(forumv1connect.ForumServiceThreadReplyEditProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(forumv1connect.ForumServiceThreadMessageDeleteProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(forumv1connect.ForumServiceCategoryCreateProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.UserRoute(forumv1connect.ForumServiceCategoryEditProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.UserRoute(forumv1connect.ForumServiceCategoryProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.UserRoute(forumv1connect.ForumServiceForumCreateProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.UserRoute(forumv1connect.ForumServiceForumEditProcedure, rpc.WithMinPermissions(permission.Moderator))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) ActiveUsers(_ context.Context, _ *emptypb.Empty) (*v1.ActiveUsersResponse, error) {
	current := s.forums.Current()
	resp := v1.ActiveUsersResponse{UserActivity: make([]*v1.UserActivity, len(current))}

	for idx, act := range current {
		sid := act.Person.GetSteamID()
		resp.UserActivity[idx] = &v1.UserActivity{
			SteamId:         ptr.To(sid.Int64()),
			PersonaName:     ptr.To(act.Person.GetName()),
			PermissionLevel: ptr.To(personv1.Privilege(act.Person.GetPrivilege())),
			CreatedOn:       timestamppb.New(act.LastActivity),
		}
	}

	return &resp, nil
}

func (s Service) Overview(ctx context.Context, _ *emptypb.Empty) (*v1.OverviewResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	s.forums.Touch(user)

	categories, errCats := s.forums.Categories(ctx)
	if errCats != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	currentForums, errForums := s.forums.Forums(ctx)
	if errForums != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	for index := range categories {
		for _, forum := range currentForums {
			if !user.HasPermission(forum.PermissionLevel) {
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

	resp := v1.OverviewResponse{Categories: make([]*v1.Category, len(categories))}
	for index, cat := range categories {
		resp.Categories[index] = toCategory(cat)
	}

	return &resp, nil
}

func (s Service) RecentMessages(ctx context.Context, _ *emptypb.Empty) (*v1.RecentMessagesResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	messages, errThreads := s.forums.RecentActivity(ctx, 5, user.GetPrivilege())
	if errThreads != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.RecentMessagesResponse{Messages: make([]*v1.Message, len(messages))}
	for idx, msg := range messages {
		resp.Messages[idx] = toMessage(msg)
	}

	return &resp, nil
}

func toMessage(msg Message) *v1.Message {
	return &v1.Message{
		ForumMessageId:  &msg.ForumMessageID,
		ForumThreadId:   &msg.ForumThreadID,
		SourceId:        ptr.To(msg.SourceID.Int64()),
		BodyMd:          &msg.BodyMD,
		Title:           &msg.Title,
		Online:          &msg.Online,
		Signature:       &msg.Signature,
		PersonaName:     &msg.Personaname,
		AvatarHash:      &msg.Avatarhash,
		PermissionLevel: ptr.To(personv1.Privilege(msg.PermissionLevel)),
		CreatedOn:       timestamppb.New(msg.CreatedOn),
		UpdatedOn:       timestamppb.New(msg.UpdatedOn),
	}
}

func (s Service) Threads(ctx context.Context, req *v1.ThreadsRequest) (*v1.ThreadsResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	s.forums.Touch(user)

	threads, errThreads := s.forums.Threads(ctx, ThreadQueryFilter{ForumID: req.GetForumId()})
	if errThreads != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	var forum Forum
	if err := s.forums.Forum(ctx, req.GetForumId(), &forum); err != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	if !user.HasPermission(forum.PermissionLevel) {
		return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
	}

	resp := v1.ThreadsResponse{Threads: make([]*v1.ThreadWithSource, len(threads))}
	for idx, thread := range threads {
		resp.Threads[idx] = fromThreadWithSource(thread)
	}

	return &resp, nil
}

func (s Service) Thread(ctx context.Context, req *v1.ThreadRequest) (*v1.ThreadResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	s.forums.Touch(user)

	var thread Thread
	if errThreads := s.forums.Thread(ctx, req.GetForumThreadId(), &thread); errThreads != nil {
		if errors.Is(errThreads, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	if err := s.forums.ThreadIncrView(ctx, req.GetForumThreadId()); err != nil {
		slog.Error("Failed to increment thread view count", slog.String("error", err.Error()),
			slog.Int("forum_thread_id", int(req.GetForumThreadId())))
	}

	return &v1.ThreadResponse{Thread: fromThread(thread)}, nil
}

func (s Service) ThreadDelete(ctx context.Context, req *v1.ThreadDeleteRequest) (*emptypb.Empty, error) {
	var thread Thread
	if errGet := s.forums.Thread(ctx, req.GetForumThreadId(), &thread); errGet != nil {
		if errors.Is(errGet, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	if errDelete := s.forums.ThreadDelete(ctx, thread.ForumThreadID); errDelete != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	var forum Forum
	if errForum := s.forums.Forum(ctx, thread.ForumID, &forum); errForum != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	forum.CountThreads--

	if errSave := s.forums.ForumSave(ctx, &forum); errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func (s Service) Forum(ctx context.Context, req *v1.ForumRequest) (*v1.ForumResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)

	var forum Forum

	if errForum := s.forums.Forum(ctx, req.GetForumId(), &forum); errForum != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	if !user.HasPermission(forum.PermissionLevel) {
		return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
	}

	return &v1.ForumResponse{Forum: toForum(forum)}, nil
}

func (s Service) ThreadMessages(ctx context.Context, req *v1.ThreadMessagesRequest) (*v1.ThreadMessagesResponse, error) {
	messages, errMessages := s.forums.Messages(ctx, ThreadMessagesQuery{
		Deleted:       req.GetDeleted(),
		ForumThreadID: req.GetForumThreadId(),
	})
	if errMessages != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	activeUsers := s.forums.Current()
	resp := v1.ThreadMessagesResponse{Messages: make([]*v1.Message, len(messages))}

	for idx := range messages {
		for _, activity := range activeUsers {
			if messages[idx].SourceID == activity.Person.GetSteamID() {
				messages[idx].Online = true

				break
			}
		}

		resp.Messages[idx] = toMessage(messages[idx])
	}

	return &resp, nil
}

func (s Service) ThreadCreate(ctx context.Context, req *v1.ThreadCreateRequest) (*v1.ThreadCreateResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	s.forums.Touch(user)

	var forum Forum
	if errForum := s.forums.Forum(ctx, req.GetForumId(), &forum); errForum != nil {
		switch {
		case errors.Is(errForum, database.ErrNoResult):
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		default:
			return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
		}
	}

	thread := forum.NewThread(req.GetTitle(), user.GetSteamID())
	thread.Sticky = req.GetSticky()
	thread.Locked = req.GetLocked()

	if errSaveThread := s.forums.ThreadSave(ctx, &thread); errSaveThread != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	message := thread.NewMessage(user.GetSteamID(), req.GetBodyMd())
	if errSaveMessage := s.forums.MessageSave(ctx, &message); errSaveMessage != nil {
		// Drop created thread.
		// TODO transaction
		if errRollback := s.forums.ThreadDelete(ctx, thread.ForumThreadID); errRollback != nil {
			slog.Error("Failed to rollback new thread", slog.String("error", errRollback.Error()))
		}

		slog.Error("Failed to save new forum message", slog.String("error", errSaveMessage.Error()))

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	if errIncr := s.forums.ForumIncrMessageCount(ctx, forum.ForumID, true); errIncr != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.ThreadCreateResponse{
		Thread:  fromThread(thread),
		Message: toMessage(message),
	}, nil
}

func (s Service) ThreadEdit(ctx context.Context, req *v1.ThreadEditRequest) (*v1.ThreadEditResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	req.Title = ptr.To(stringutil.SanitizeUGC(req.GetTitle()))

	var thread Thread
	if errGet := s.forums.Thread(ctx, req.GetForumThreadId(), &thread); errGet != nil {
		if errors.Is(errGet, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	if thread.SourceID != user.GetSteamID() && !user.HasPermission(permission.Moderator) {
		return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
	}

	thread.Title = req.GetTitle()
	thread.Sticky = req.GetSticky()
	thread.Locked = req.GetLocked()

	if errDelete := s.forums.ThreadSave(ctx, &thread); errDelete != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.ThreadEditResponse{Thread: fromThread(thread)}, nil
}

func (s Service) ThreadReplyCreate(ctx context.Context, req *v1.ThreadReplyCreateRequest) (*v1.ThreadReplyCreateResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	s.forums.Touch(user)

	var thread Thread
	if errThread := s.forums.Thread(ctx, req.GetForumThreadId(), &thread); errThread != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	if thread.Locked && !user.HasPermission(permission.Editor) {
		return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
	}

	body := stringutil.SanitizeUGC(req.GetBodyMd())
	newMessage := thread.NewMessage(user.GetSteamID(), body)
	if errSave := s.forums.MessageSave(ctx, &newMessage); errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	newMessage.Personaname = user.GetName()
	newMessage.Avatarhash = user.GetAvatar().Hash()
	newMessage.PermissionLevel = user.GetPrivilege()
	newMessage.Online = true

	return &v1.ThreadReplyCreateResponse{Message: toMessage(newMessage)}, nil
}

func (s Service) ThreadReplyEdit(ctx context.Context, req *v1.ThreadReplyEditRequest) (*v1.ThreadReplyEditResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	s.forums.Touch(user)

	var message Message
	if errMessage := s.forums.Message(ctx, req.GetForumMessageId(), &message); errMessage != nil {
		if errors.Is(errMessage, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	if message.SourceID != user.GetSteamID() && !user.HasPermission(permission.Moderator) {
		return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
	}

	message.BodyMD = stringutil.SanitizeUGC(req.GetBodyMd())

	if errSave := s.forums.MessageSave(ctx, &message); errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.ThreadReplyEditResponse{Message: toMessage(message)}, nil
}

func (s Service) ThreadMessageDelete(ctx context.Context, req *v1.ThreadMessageDeleteRequest) (*emptypb.Empty, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	if err := s.forums.MessageDelete(ctx, user, req.GetForumMessageId()); err != nil {
		switch {
		case errors.Is(err, database.ErrNoResult):
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		case errors.Is(err, ErrThreadLocked):
			return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
		case errors.Is(err, permission.ErrDenied):
			return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
		default:
			return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
		}
	}

	return &emptypb.Empty{}, nil
}

func (s Service) CategoryCreate(ctx context.Context, req *v1.CategoryCreateRequest) (*v1.CategoryCreateResponse, error) {
	category := Category{
		Title:       stringutil.SanitizeUGC(req.GetTitle()),
		Description: stringutil.SanitizeUGC(req.GetDescription()),
		Ordering:    req.GetOrdering(),
		CreatedOn:   time.Now(),
		UpdatedOn:   time.Now(),
	}

	if errSave := s.forums.CategorySave(ctx, &category); errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.CategoryCreateResponse{Category: toCategory(category)}, nil
}

func (s Service) CategoryEdit(ctx context.Context, req *v1.CategoryEditRequest) (*v1.CategoryEditResponse, error) {
	var category Category
	if errGet := s.forums.Category(ctx, req.GetForumCategoryId(), &category); errGet != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	category.Title = stringutil.SanitizeUGC(req.GetTitle())
	category.Description = stringutil.SanitizeUGC(req.GetDescription())
	category.Ordering = req.GetOrdering()

	if errSave := s.forums.CategorySave(ctx, &category); errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.CategoryEditResponse{Category: toCategory(category)}, nil
}

func (s Service) Category(ctx context.Context, req *v1.CategoryRequest) (*v1.CategoryResponse, error) {
	var category Category
	if errGet := s.forums.Category(ctx, req.GetForumCategoryId(), &category); errGet != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.CategoryResponse{Category: toCategory(category)}, nil
}

func (s Service) ForumCreate(ctx context.Context, req *v1.ForumCreateRequest) (*v1.ForumCreateResponse, error) {
	forum := Forum{
		ForumCategoryID: req.GetForumCategoryId(),
		Title:           stringutil.SanitizeUGC(req.GetTitle()),
		Description:     stringutil.SanitizeUGC(req.GetDescription()),
		Ordering:        req.GetOrdering(),
		PermissionLevel: permission.Privilege(req.GetPermissionLevel()),
		CreatedOn:       time.Now(),
		UpdatedOn:       time.Now(),
	}

	if errSave := s.forums.ForumSave(ctx, &forum); errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.ForumCreateResponse{Forum: toForum(forum)}, nil
}

func (s Service) ForumEdit(ctx context.Context, req *v1.ForumEditRequest) (*v1.ForumEditResponse, error) {
	var forum Forum
	if errGet := s.forums.Forum(ctx, req.GetForumId(), &forum); errGet != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	forum.ForumCategoryID = req.GetForumCategoryId()
	forum.Title = stringutil.SanitizeUGC(req.GetTitle())
	forum.Description = stringutil.SanitizeUGC(req.GetDescription())
	forum.Ordering = req.GetOrdering()
	forum.PermissionLevel = permission.Privilege(req.GetPermissionLevel())

	if errSave := s.forums.ForumSave(ctx, &forum); errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.ForumEditResponse{Forum: toForum(forum)}, nil
}

func fromThread(thread Thread) *v1.Thread {
	return &v1.Thread{
		ForumId:   &thread.ForumID,
		SourceId:  ptr.To(thread.SourceID.Int64()),
		Title:     &thread.Title,
		CreatedOn: timestamppb.New(thread.CreatedOn),
		UpdatedOn: timestamppb.New(thread.UpdatedOn),
	}
}

func fromThreadWithSource(thread ThreadWithSource) *v1.ThreadWithSource {
	return &v1.ThreadWithSource{
		Thread:               fromThread(thread.Thread),
		PersonaName:          &thread.Personaname,
		AvatarHash:           &thread.Avatarhash,
		PermissionLevel:      ptr.To(personv1.Privilege(thread.PermissionLevel)),
		RecentForumMessageId: &thread.RecentForumMessageID,
		RecentCreatedOn:      timestamppb.New(thread.RecentCreatedOn),
		RecentSteamId:        &thread.RecentSteamID,
		RecentPersonaName:    &thread.RecentPersonaname,
		RecentAvatarHash:     &thread.RecentAvatarhash,
	}
}

func toForum(forum Forum) *v1.Forum {
	return &v1.Forum{
		ForumId:             &forum.ForumID,
		ForumCategoryId:     &forum.ForumCategoryID,
		LastThreadId:        &forum.LastThreadID,
		Title:               &forum.Title,
		Description:         &forum.Description,
		Ordering:            &forum.Ordering,
		CountThreads:        &forum.CountThreads,
		CountMessages:       &forum.CountMessages,
		PermissionLevel:     ptr.To(personv1.Privilege(forum.PermissionLevel)),
		RecentForumThreadId: &forum.RecentForumThreadID,
		RecentForumTitle:    &forum.RecentForumTitle,
		RecentSourceId:      &forum.RecentSourceID,
		RecentPersonaName:   &forum.RecentPersonaname,
		RecentAvatarHash:    &forum.RecentAvatarhash,
		RecentCreatedOn:     timestamppb.New(forum.RecentCreatedOn),
		CreatedOn:           timestamppb.New(forum.CreatedOn),
		UpdatedOn:           timestamppb.New(forum.UpdatedOn),
	}
}

func toCategory(cat Category) *v1.Category {
	v1cat := v1.Category{
		ForumCategoryId: &cat.ForumCategoryID,
		Title:           &cat.Title,
		Description:     &cat.Description,
		Ordering:        &cat.Ordering,
		CreatedOn:       timestamppb.New(cat.CreatedOn),
		UpdatedOn:       timestamppb.New(cat.UpdatedOn),
	}

	for index, forum := range cat.Forums {
		v1cat.Forums[index] = toForum(forum)
	}

	return &v1cat
}
