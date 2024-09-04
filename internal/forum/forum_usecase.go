package forum

import (
	"context"
	"log/slog"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type forums struct {
	repo          domain.ForumRepository
	tracker       *Tracker
	notifications domain.NotificationUsecase
}

func NewForumUsecase(repository domain.ForumRepository, notifications domain.NotificationUsecase) domain.ForumUsecase {
	return &forums{repo: repository, notifications: notifications, tracker: NewTracker()}
}

func (f forums) Start(ctx context.Context) {
	f.tracker.Start(ctx)
}

func (f forums) Touch(up domain.UserProfile) {
	f.tracker.Touch(up)
}

func (f forums) Current() []domain.ForumActivity {
	return f.tracker.Current()
}

func (f forums) ForumCategories(ctx context.Context) ([]domain.ForumCategory, error) {
	return f.repo.ForumCategories(ctx)
}

func (f forums) ForumCategorySave(ctx context.Context, category *domain.ForumCategory) error {
	isNew := category.ForumCategoryID == 0

	if err := f.repo.ForumCategorySave(ctx, category); err != nil {
		return err
	}

	if isNew {
		slog.Info("New forum category created", slog.String("title", category.Title))
	} else {
		slog.Info("Forum category updated", slog.String("title", category.Title))
	}

	f.notifications.Enqueue(ctx, domain.NewDiscordNotification(domain.ChannelForumLog, discord.ForumCategorySave(*category)))

	return nil
}

func (f forums) ForumCategory(ctx context.Context, categoryID int, category *domain.ForumCategory) error {
	return f.repo.ForumCategory(ctx, categoryID, category)
}

func (f forums) ForumCategoryDelete(ctx context.Context, category domain.ForumCategory) error {
	if err := f.repo.ForumCategoryDelete(ctx, category.ForumCategoryID); err != nil {
		return err
	}

	f.notifications.Enqueue(ctx, domain.NewDiscordNotification(domain.ChannelForumLog, discord.ForumCategoryDelete(category)))
	slog.Info("Forum category deleted", slog.String("category", category.Title), slog.Int("forum_category_id", category.ForumCategoryID))

	return nil
}

func (f forums) Forums(ctx context.Context) ([]domain.Forum, error) {
	return f.repo.Forums(ctx)
}

func (f forums) ForumSave(ctx context.Context, forum *domain.Forum) error {
	isNew := forum.ForumID == 0

	if err := f.repo.ForumSave(ctx, forum); err != nil {
		return err
	}

	f.notifications.Enqueue(ctx, domain.NewDiscordNotification(domain.ChannelForumLog, discord.ForumSaved(*forum)))

	if isNew {
		slog.Info("New forum created", slog.String("title", forum.Title))
	} else {
		slog.Info("Forum updated", slog.String("title", forum.Title), slog.Int("forum_id", forum.ForumID))
	}

	return nil
}

func (f forums) Forum(ctx context.Context, forumID int, forum *domain.Forum) error {
	return f.repo.Forum(ctx, forumID, forum)
}

func (f forums) ForumDelete(ctx context.Context, forumID int) error {
	if err := f.repo.ForumDelete(ctx, forumID); err != nil {
		return err
	}

	slog.Info("Forum deleted successfully", slog.Int("forum_id", forumID))

	return nil
}

func (f forums) ForumThreadSave(ctx context.Context, thread *domain.ForumThread) error {
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

func (f forums) ForumThread(ctx context.Context, forumThreadID int64, thread *domain.ForumThread) error {
	return f.repo.ForumThread(ctx, forumThreadID, thread)
}

func (f forums) ForumThreadIncrView(ctx context.Context, forumThreadID int64) error {
	return f.repo.ForumThreadIncrView(ctx, forumThreadID)
}

func (f forums) ForumThreadDelete(ctx context.Context, forumThreadID int64) error {
	if err := f.repo.ForumThreadDelete(ctx, forumThreadID); err != nil {
		return err
	}

	slog.Info("Forum thread deleted", slog.Int64("forum_thread_id", forumThreadID))

	return nil
}

func (f forums) ForumThreads(ctx context.Context, filter domain.ThreadQueryFilter) ([]domain.ThreadWithSource, error) {
	return f.repo.ForumThreads(ctx, filter)
}

func (f forums) ForumIncrMessageCount(ctx context.Context, forumID int, incr bool) error {
	return f.repo.ForumIncrMessageCount(ctx, forumID, incr)
}

func (f forums) ForumMessageSave(ctx context.Context, message *domain.ForumMessage) error {
	isNew := message.ForumMessageID == 0

	if err := f.repo.ForumMessageSave(ctx, message); err != nil {
		return err
	}

	f.notifications.Enqueue(ctx, domain.NewDiscordNotification(domain.ChannelForumLog, discord.ForumMessageSaved(*message)))

	if isNew {
		slog.Info("Created new forum message", slog.Int64("forum_thread_id", message.ForumThreadID))
	} else {
		slog.Info("Forum message edited", slog.Int64("forum_thread_id", message.ForumThreadID))
	}

	return nil
}

func (f forums) ForumRecentActivity(ctx context.Context, limit uint64, permissionLevel domain.Privilege) ([]domain.ForumMessage, error) {
	return f.repo.ForumRecentActivity(ctx, limit, permissionLevel)
}

func (f forums) ForumMessage(ctx context.Context, messageID int64, forumMessage *domain.ForumMessage) error {
	return f.repo.ForumMessage(ctx, messageID, forumMessage)
}

func (f forums) ForumMessages(ctx context.Context, filters domain.ThreadMessagesQuery) ([]domain.ForumMessage, error) {
	return f.repo.ForumMessages(ctx, filters)
}

func (f forums) ForumMessageDelete(ctx context.Context, messageID int64) error {
	if err := f.repo.ForumMessageDelete(ctx, messageID); err != nil {
		return err
	}

	slog.Info("Forum message deleted", slog.Int64("message_id", messageID))

	return nil
}

func (f forums) ForumMessageVoteApply(ctx context.Context, messageVote *domain.ForumMessageVote) error {
	return f.repo.ForumMessageVoteApply(ctx, messageVote)
}

func (f forums) ForumMessageVoteByID(ctx context.Context, messageVoteID int64, messageVote *domain.ForumMessageVote) error {
	return f.repo.ForumMessageVoteByID(ctx, messageVoteID, messageVote)
}
