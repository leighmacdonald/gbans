package contest

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type ContestUsecase struct {
	repository ContestRepository
}

func NewContestUsecase(repository ContestRepository) ContestUsecase {
	return ContestUsecase{repository: repository}
}

func (c *ContestUsecase) ContestSave(ctx context.Context, contest Contest) (Contest, error) {
	if contest.ContestID.IsNil() {
		newID, errID := uuid.NewV4()
		if errID != nil {
			return contest, errors.Join(errID, domain.ErrUUIDCreate)
		}

		contest.ContestID = newID
	}

	if errSave := c.repository.ContestSave(ctx, &contest); errSave != nil {
		return contest, errSave
	}

	slog.Info("Contest updated",
		slog.String("contest_id", contest.ContestID.String()),
		slog.String("title", contest.Title))

	return contest, nil
}

func (c *ContestUsecase) ContestByID(ctx context.Context, contestID uuid.UUID, contest *Contest) error {
	return c.repository.ContestByID(ctx, contestID, contest)
}

func (c *ContestUsecase) ContestDelete(ctx context.Context, contestID uuid.UUID) error {
	if err := c.repository.ContestDelete(ctx, contestID); err != nil {
		return err
	}

	slog.Info("Contest deleted", slog.String("contest_id", contestID.String()))

	return nil
}

func (c *ContestUsecase) ContestEntryDelete(ctx context.Context, contestEntryID uuid.UUID) error {
	return c.repository.ContestEntryDelete(ctx, contestEntryID)
}

func (c *ContestUsecase) Contests(ctx context.Context, user domain.PersonInfo) ([]Contest, error) {
	return c.repository.Contests(ctx, !user.HasPermission(permission.PModerator))
}

func (c *ContestUsecase) ContestEntry(ctx context.Context, contestID uuid.UUID, entry *ContestEntry) error {
	return c.repository.ContestEntry(ctx, contestID, entry)
}

func (c *ContestUsecase) ContestEntrySave(ctx context.Context, entry ContestEntry) error {
	return c.repository.ContestEntrySave(ctx, entry)
}

func (c *ContestUsecase) ContestEntries(ctx context.Context, contestID uuid.UUID) ([]*ContestEntry, error) {
	return c.repository.ContestEntries(ctx, contestID)
}

func (c *ContestUsecase) ContestEntryVoteGet(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SteamID, record *ContentVoteRecord) error {
	return c.repository.ContestEntryVoteGet(ctx, contestEntryID, steamID, record)
}

func (c *ContestUsecase) ContestEntryVote(ctx context.Context, contestID uuid.UUID, contestEntryID uuid.UUID, user domain.PersonInfo, vote bool) error {
	var contest Contest
	if errContests := c.ContestByID(ctx, contestID, &contest); errContests != nil {
		return errContests
	}

	if !contest.Public && !user.HasPermission(permission.PModerator) {
		return permission.ErrPermissionDenied
	}

	if !contest.Voting || !contest.DownVotes && !vote {
		return httphelper.ErrBadRequest // tODO proper error
	}

	if err := c.repository.ContestEntryVote(ctx, contestEntryID, user.GetSteamID(), vote); err != nil {
		return err
	}

	sid := user.GetSteamID()

	slog.Info("Entry vote registered", slog.String("contest_id", contest.ContestID.String()), slog.Bool("vote", vote), slog.String("steam_id", sid.String()))

	return nil
}

func (c *ContestUsecase) ContestEntryVoteDelete(ctx context.Context, contestEntryVoteID int64) error {
	return c.repository.ContestEntryVoteDelete(ctx, contestEntryVoteID)
}

func (c *ContestUsecase) ContestEntryVoteUpdate(ctx context.Context, contestEntryVoteID int64, newVote bool) error {
	return c.repository.ContestEntryVoteUpdate(ctx, contestEntryVoteID, newVote)
}
