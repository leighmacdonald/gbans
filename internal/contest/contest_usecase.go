package contest

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type contestUsecase struct {
	contestRepo domain.ContestRepository
}

func NewContestUsecase(contestRepository domain.ContestRepository) domain.ContestUsecase {
	return &contestUsecase{contestRepo: contestRepository}
}

func (c *contestUsecase) ContestSave(ctx context.Context, contest *domain.Contest) error {
	return c.contestRepo.ContestSave(ctx, contest)
}

func (c *contestUsecase) ContestByID(ctx context.Context, contestID uuid.UUID, contest *domain.Contest) error {
	return c.contestRepo.ContestByID(ctx, contestID, contest)
}

func (c *contestUsecase) ContestDelete(ctx context.Context, contestID uuid.UUID) error {
	return c.contestRepo.ContestDelete(ctx, contestID)
}

func (c *contestUsecase) ContestEntryDelete(ctx context.Context, contestEntryID uuid.UUID) error {
	return c.contestRepo.ContestEntryDelete(ctx, contestEntryID)
}

func (c *contestUsecase) Contests(ctx context.Context, publicOnly bool) ([]domain.Contest, error) {
	return c.contestRepo.Contests(ctx, publicOnly)
}

func (c *contestUsecase) ContestEntry(ctx context.Context, contestID uuid.UUID, entry *domain.ContestEntry) error {
	return c.contestRepo.ContestEntry(ctx, contestID, entry)
}

func (c *contestUsecase) ContestEntrySave(ctx context.Context, entry domain.ContestEntry) error {
	return c.contestRepo.ContestEntrySave(ctx, entry)
}

func (c *contestUsecase) ContestEntries(ctx context.Context, contestID uuid.UUID) ([]*domain.ContestEntry, error) {
	return c.contestRepo.ContestEntries(ctx, contestID)
}

func (c *contestUsecase) ContestEntryVoteGet(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SID64, record *domain.ContentVoteRecord) error {
	return c.contestRepo.ContestEntryVoteGet(ctx, contestEntryID, steamID, record)
}

func (c *contestUsecase) ContestEntryVote(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SID64, vote bool) error {
	return c.contestRepo.ContestEntryVote(ctx, contestEntryID, steamID, vote)
}

func (c *contestUsecase) ContestEntryVoteDelete(ctx context.Context, contestEntryVoteID int64) error {
	return c.contestRepo.ContestEntryVoteDelete(ctx, contestEntryVoteID)
}

func (c *contestUsecase) ContestEntryVoteUpdate(ctx context.Context, contestEntryVoteID int64, newVote bool) error {
	return c.contestRepo.ContestEntryVoteUpdate(ctx, contestEntryVoteID, newVote)
}
