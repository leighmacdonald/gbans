package forum

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type forumUsecase struct {
	repo    domain.ForumRepository
	tracker *Tracker
	discord domain.DiscordUsecase
}

func NewForumUsecase(repository domain.ForumRepository, discord domain.DiscordUsecase) domain.ForumUsecase {
	return &forumUsecase{repo: repository, discord: discord, tracker: NewTracker()}
}

func (f forumUsecase) Start(ctx context.Context) {
	f.tracker.Start(ctx)
}

func (f forumUsecase) Touch(up domain.UserProfile) {
	f.tracker.Touch(up)
}

func (f forumUsecase) Current() []domain.ForumActivity {
	return f.tracker.Current()
}

func (f forumUsecase) ForumCategories(ctx context.Context) ([]domain.ForumCategory, error) {
	return f.repo.ForumCategories(ctx)
}

func (f forumUsecase) ForumCategorySave(ctx context.Context, category *domain.ForumCategory) error {
	if err := f.repo.ForumCategorySave(ctx, category); err != nil {
		return err
	}

	f.discord.SendPayload(domain.ChannelForumLog, discord.ForumCategorySave(*category))

	return nil
}

func (f forumUsecase) ForumCategory(ctx context.Context, categoryID int, category *domain.ForumCategory) error {
	return f.repo.ForumCategory(ctx, categoryID, category)
}

func (f forumUsecase) ForumCategoryDelete(ctx context.Context, category domain.ForumCategory) error {
	if err := f.repo.ForumCategoryDelete(ctx, category.ForumCategoryID); err != nil {
		return err
	}

	f.discord.SendPayload(domain.ChannelForumLog, discord.ForumCategoryDelete(category))

	return nil
}

func (f forumUsecase) Forums(ctx context.Context) ([]domain.Forum, error) {
	return f.repo.Forums(ctx)
}

func (f forumUsecase) ForumSave(ctx context.Context, forum *domain.Forum) error {
	if err := f.repo.ForumSave(ctx, forum); err != nil {
		return err
	}

	f.discord.SendPayload(domain.ChannelForumLog, discord.ForumSaved(*forum))

	return nil
}

func (f forumUsecase) Forum(ctx context.Context, forumID int, forum *domain.Forum) error {
	return f.repo.Forum(ctx, forumID, forum)
}

func (f forumUsecase) ForumDelete(ctx context.Context, forumID int) error {
	return f.repo.ForumDelete(ctx, forumID)
}

func (f forumUsecase) ForumThreadSave(ctx context.Context, thread *domain.ForumThread) error {
	return f.repo.ForumThreadSave(ctx, thread)
}

func (f forumUsecase) ForumThread(ctx context.Context, forumThreadID int64, thread *domain.ForumThread) error {
	return f.repo.ForumThread(ctx, forumThreadID, thread)
}

func (f forumUsecase) ForumThreadIncrView(ctx context.Context, forumThreadID int64) error {
	return f.repo.ForumThreadIncrView(ctx, forumThreadID)
}

func (f forumUsecase) ForumThreadDelete(ctx context.Context, forumThreadID int64) error {
	return f.repo.ForumThreadDelete(ctx, forumThreadID)
}

func (f forumUsecase) ForumThreads(ctx context.Context, filter domain.ThreadQueryFilter) ([]domain.ThreadWithSource, error) {
	return f.repo.ForumThreads(ctx, filter)
}

func (f forumUsecase) ForumIncrMessageCount(ctx context.Context, forumID int, incr bool) error {
	return f.repo.ForumIncrMessageCount(ctx, forumID, incr)
}

func (f forumUsecase) ForumMessageSave(ctx context.Context, message *domain.ForumMessage) error {
	if err := f.repo.ForumMessageSave(ctx, message); err != nil {
		return err
	}

	f.discord.SendPayload(domain.ChannelForumLog, discord.ForumMessageSaved(*message))

	return nil
}

func (f forumUsecase) ForumRecentActivity(ctx context.Context, limit uint64, permissionLevel domain.Privilege) ([]domain.ForumMessage, error) {
	return f.repo.ForumRecentActivity(ctx, limit, permissionLevel)
}

func (f forumUsecase) ForumMessage(ctx context.Context, messageID int64, forumMessage *domain.ForumMessage) error {
	return f.repo.ForumMessage(ctx, messageID, forumMessage)
}

func (f forumUsecase) ForumMessages(ctx context.Context, filters domain.ThreadMessagesQuery) ([]domain.ForumMessage, error) {
	return f.repo.ForumMessages(ctx, filters)
}

func (f forumUsecase) ForumMessageDelete(ctx context.Context, messageID int64) error {
	return f.repo.ForumMessageDelete(ctx, messageID)
}

func (f forumUsecase) ForumMessageVoteApply(ctx context.Context, messageVote *domain.ForumMessageVote) error {
	return f.repo.ForumMessageVoteApply(ctx, messageVote)
}

func (f forumUsecase) ForumMessageVoteByID(ctx context.Context, messageVoteID int64, messageVote *domain.ForumMessageVote) error {
	return f.repo.ForumMessageVoteByID(ctx, messageVoteID, messageVote)
}
