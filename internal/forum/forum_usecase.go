package forum

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/domain"
)

type forumUsecase struct {
	forumRepo domain.ForumRepository
}

func NewForumUsecase(fr domain.ForumRepository) domain.ForumUsecase {
	return &forumUsecase{forumRepo: fr}
}

func (f forumUsecase) ForumCategories(ctx context.Context) ([]domain.ForumCategory, error) {
	return f.forumRepo.ForumCategories(ctx)
}

func (f forumUsecase) ForumCategorySave(ctx context.Context, category *domain.ForumCategory) error {
	return f.forumRepo.ForumCategorySave(ctx, category)
}

func (f forumUsecase) ForumCategory(ctx context.Context, categoryID int, category *domain.ForumCategory) error {
	return f.forumRepo.ForumCategory(ctx, categoryID, category)
}

func (f forumUsecase) ForumCategoryDelete(ctx context.Context, categoryID int) error {
	return f.forumRepo.ForumCategoryDelete(ctx, categoryID)
}

func (f forumUsecase) Forums(ctx context.Context) ([]domain.Forum, error) {
	return f.forumRepo.Forums(ctx)
}

func (f forumUsecase) ForumSave(ctx context.Context, forum *domain.Forum) error {
	return f.forumRepo.ForumSave(ctx, forum)
}

func (f forumUsecase) Forum(ctx context.Context, forumID int, forum *domain.Forum) error {
	return f.forumRepo.Forum(ctx, forumID, forum)
}

func (f forumUsecase) ForumDelete(ctx context.Context, forumID int) error {
	return f.forumRepo.ForumDelete(ctx, forumID)
}

func (f forumUsecase) ForumThreadSave(ctx context.Context, thread *domain.ForumThread) error {
	return f.forumRepo.ForumThreadSave(ctx, thread)
}

func (f forumUsecase) ForumThread(ctx context.Context, forumThreadID int64, thread *domain.ForumThread) error {
	return f.forumRepo.ForumThread(ctx, forumThreadID, thread)
}

func (f forumUsecase) ForumThreadIncrView(ctx context.Context, forumThreadID int64) error {
	return f.forumRepo.ForumThreadIncrView(ctx, forumThreadID)
}

func (f forumUsecase) ForumThreadDelete(ctx context.Context, forumThreadID int64) error {
	return f.forumRepo.ForumThreadDelete(ctx, forumThreadID)
}

func (f forumUsecase) ForumThreads(ctx context.Context, filter domain.ThreadQueryFilter) ([]domain.ThreadWithSource, int64, error) {
	return f.forumRepo.ForumThreads(ctx, filter)
}

func (f forumUsecase) ForumIncrMessageCount(ctx context.Context, forumID int, incr bool) error {
	return f.forumRepo.ForumIncrMessageCount(ctx, forumID, incr)
}

func (f forumUsecase) ForumMessageSave(ctx context.Context, message *domain.ForumMessage) error {
	return f.ForumMessageSave(ctx, message)
}

func (f forumUsecase) ForumRecentActivity(ctx context.Context, limit uint64, permissionLevel domain.Privilege) ([]domain.ForumMessage, error) {
	return f.forumRepo.ForumRecentActivity(ctx, limit, permissionLevel)
}

func (f forumUsecase) ForumMessage(ctx context.Context, messageID int64, forumMessage *domain.ForumMessage) error {
	return f.forumRepo.ForumMessage(ctx, messageID, forumMessage)
}

func (f forumUsecase) ForumMessages(ctx context.Context, filters domain.ThreadMessagesQueryFilter) ([]domain.ForumMessage, int64, error) {
	return f.forumRepo.ForumMessages(ctx, filters)
}

func (f forumUsecase) ForumMessageDelete(ctx context.Context, messageID int64) error {
	return f.forumRepo.ForumMessageDelete(ctx, messageID)
}

func (f forumUsecase) ForumMessageVoteApply(ctx context.Context, messageVote *domain.ForumMessageVote) error {
	return f.forumRepo.ForumMessageVoteApply(ctx, messageVote)
}

func (f forumUsecase) ForumMessageVoteByID(ctx context.Context, messageVoteID int64, messageVote *domain.ForumMessageVote) error {
	return f.forumRepo.ForumMessageVoteByID(ctx, messageVoteID, messageVote)
}
