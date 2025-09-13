package forum

import (
	"context"
	"log/slog"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/notification"
)

type ForumUsecase struct {
	repo          *ForumRepository
	tracker       *Tracker
	notifications *notification.NotificationUsecase
}

func NewForumUsecase(repository *ForumRepository, notifications *notification.NotificationUsecase) *ForumUsecase {
	return &ForumUsecase{repo: repository, notifications: notifications, tracker: NewTracker()}
}

func (f ForumUsecase) Start(ctx context.Context) {
	f.tracker.Start(ctx)
}

func (f ForumUsecase) Touch(up domain.PersonInfo) {
	f.tracker.Touch(up)
}

func (f ForumUsecase) Current() []ForumActivity {
	return f.tracker.Current()
}

func (f ForumUsecase) ForumCategories(ctx context.Context) ([]ForumCategory, error) {
	return f.repo.ForumCategories(ctx)
}

func (f ForumUsecase) ForumCategorySave(ctx context.Context, category *ForumCategory) error {
	isNew := category.ForumCategoryID == 0

	if err := f.repo.ForumCategorySave(ctx, category); err != nil {
		return err
	}

	if isNew {
		slog.Info("New forum category created", slog.String("title", category.Title))
	} else {
		slog.Info("Forum category updated", slog.String("title", category.Title))
	}

	// f.notifications.Enqueue(ctx, notification.NewDiscordNotification(discord.ChannelForumLog, ForumCategorySave(*category)))

	return nil
}

func (f ForumUsecase) ForumCategory(ctx context.Context, categoryID int, category *ForumCategory) error {
	return f.repo.ForumCategory(ctx, categoryID, category)
}

func (f ForumUsecase) ForumCategoryDelete(ctx context.Context, category ForumCategory) error {
	if err := f.repo.ForumCategoryDelete(ctx, category.ForumCategoryID); err != nil {
		return err
	}

	// f.notifications.Enqueue(ctx, notification.NewDiscordNotification(discord.ChannelForumLog, ForumCategoryDelete(category)))
	slog.Info("Forum category deleted", slog.String("category", category.Title), slog.Int("forum_category_id", category.ForumCategoryID))

	return nil
}

func (f ForumUsecase) Forums(ctx context.Context) ([]Forum, error) {
	return f.repo.Forums(ctx)
}

func (f ForumUsecase) ForumSave(ctx context.Context, forum *Forum) error {
	isNew := forum.ForumID == 0

	if err := f.repo.ForumSave(ctx, forum); err != nil {
		return err
	}

	// f.notifications.Enqueue(ctx, notification.NewDiscordNotification(discord.ChannelForumLog, ForumSaved(*forum)))

	if isNew {
		slog.Info("New forum created", slog.String("title", forum.Title))
	} else {
		slog.Info("Forum updated", slog.String("title", forum.Title), slog.Int("forum_id", forum.ForumID))
	}

	return nil
}

func (f ForumUsecase) Forum(ctx context.Context, forumID int, forum *Forum) error {
	return f.repo.Forum(ctx, forumID, forum)
}

func (f ForumUsecase) ForumDelete(ctx context.Context, forumID int) error {
	if err := f.repo.ForumDelete(ctx, forumID); err != nil {
		return err
	}

	slog.Info("Forum deleted successfully", slog.Int("forum_id", forumID))

	return nil
}

func (f ForumUsecase) ForumThreadSave(ctx context.Context, thread *ForumThread) error {
	isNew := thread.ForumThreadID == 0

	if err := f.repo.ForumThreadSave(ctx, thread); err != nil {
		return err
	}

	if isNew {
		slog.Info("Thread created", slog.String("title", thread.Title), slog.Int64("thread_id", thread.ForumThreadID))
	} else {
		slog.Info("Forum thread updates", slog.String("title", thread.Title), slog.Int64("thread_id", thread.ForumThreadID))
	}

	return nil
}

func (f ForumUsecase) ForumThread(ctx context.Context, forumThreadID int64, thread *ForumThread) error {
	return f.repo.ForumThread(ctx, forumThreadID, thread)
}

func (f ForumUsecase) ForumThreadIncrView(ctx context.Context, forumThreadID int64) error {
	return f.repo.ForumThreadIncrView(ctx, forumThreadID)
}

func (f ForumUsecase) ForumThreadDelete(ctx context.Context, forumThreadID int64) error {
	if err := f.repo.ForumThreadDelete(ctx, forumThreadID); err != nil {
		return err
	}

	slog.Info("Forum thread deleted", slog.Int64("forum_thread_id", forumThreadID))

	return nil
}

func (f ForumUsecase) ForumThreads(ctx context.Context, filter ThreadQueryFilter) ([]ThreadWithSource, error) {
	return f.repo.ForumThreads(ctx, filter)
}

func (f ForumUsecase) ForumIncrMessageCount(ctx context.Context, forumID int, incr bool) error {
	return f.repo.ForumIncrMessageCount(ctx, forumID, incr)
}

func (f ForumUsecase) ForumMessageSave(ctx context.Context, fMessage *ForumMessage) error {
	isNew := fMessage.ForumMessageID == 0

	if err := f.repo.ForumMessageSave(ctx, fMessage); err != nil {
		return err
	}

	// f.notifications.Enqueue(ctx, notification.NewDiscordNotification(discord.ChannelForumLog, ForumMessageSaved(*fMessage)))

	if isNew {
		slog.Info("Created new forum message", slog.Int64("forum_thread_id", fMessage.ForumThreadID))
	} else {
		slog.Info("Forum message edited", slog.Int64("forum_thread_id", fMessage.ForumThreadID))
	}

	return nil
}

func (f ForumUsecase) ForumRecentActivity(ctx context.Context, limit uint64, permissionLevel permission.Privilege) ([]ForumMessage, error) {
	return f.repo.ForumRecentActivity(ctx, limit, permissionLevel)
}

func (f ForumUsecase) ForumMessage(ctx context.Context, messageID int64, forumMessage *ForumMessage) error {
	return f.repo.ForumMessage(ctx, messageID, forumMessage)
}

func (f ForumUsecase) ForumMessages(ctx context.Context, filters ThreadMessagesQuery) ([]ForumMessage, error) {
	return f.repo.ForumMessages(ctx, filters)
}

func (f ForumUsecase) ForumMessageDelete(ctx context.Context, messageID int64) error {
	if err := f.repo.ForumMessageDelete(ctx, messageID); err != nil {
		return err
	}

	slog.Info("Forum message deleted", slog.Int64("message_id", messageID))

	return nil
}

func (f ForumUsecase) ForumMessageVoteApply(ctx context.Context, messageVote *ForumMessageVote) error {
	return f.repo.ForumMessageVoteApply(ctx, messageVote)
}

func (f ForumUsecase) ForumMessageVoteByID(ctx context.Context, messageVoteID int64, messageVote *ForumMessageVote) error {
	return f.repo.ForumMessageVoteByID(ctx, messageVoteID, messageVote)
}
