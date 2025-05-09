package srcds

import (
	"context"
	"errors"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
)

func NewSpeedrunUsecase(repo domain.SpeedrunRepository) domain.SpeedrunUsecase {
	return &speedrunUsecase{repo: repo}
}

type speedrunUsecase struct {
	repo domain.SpeedrunRepository
}

func (u *speedrunUsecase) Recent(ctx context.Context, limit int) ([]domain.SpeedrunMapOverview, error) {
	if limit <= 0 || limit > 100 {
		return nil, domain.ErrValueOutOfRange
	}

	return u.repo.Recent(ctx, limit)
}

func (u *speedrunUsecase) TopNOverall(ctx context.Context, count int) (map[string][]domain.Speedrun, error) {
	if count <= 0 || count > 1000 {
		return nil, domain.ErrValueOutOfRange
	}

	return u.repo.TopNOverall(ctx, count)
}

func (u *speedrunUsecase) ByID(ctx context.Context, speedrunID int) (domain.Speedrun, error) {
	if speedrunID <= 0 {
		return domain.Speedrun{}, domain.ErrValueOutOfRange
	}

	return u.repo.ByID(ctx, speedrunID)
}

func (u *speedrunUsecase) ByMap(ctx context.Context, mapName string) ([]domain.SpeedrunMapOverview, error) {
	if mapName == "" {
		return []domain.SpeedrunMapOverview{}, domain.ErrValueOutOfRange
	}

	return u.repo.ByMap(ctx, mapName)
}

func (u *speedrunUsecase) Save(ctx context.Context, details domain.Speedrun) (domain.Speedrun, error) {
	if len(details.PointCaptures) == 0 {
		return details, domain.ErrInsufficientDetails
	}

	var validPlayers []domain.SpeedrunParticipant //nolint:prealloc
	for _, player := range details.Players {
		if int64(details.Duration)/4 > int64(player.Duration) {
			continue
		}

		validPlayers = append(validPlayers, player)
	}

	details.Players = validPlayers

	if err := u.repo.Save(ctx, &details); err != nil {
		return domain.Speedrun{}, err
	}

	return details, nil
}

func (u *speedrunUsecase) Query(_ context.Context, _ domain.SpeedrunQuery) ([]domain.Speedrun, error) {
	return nil, nil
}

func (u *speedrunUsecase) RoundStart() (uuid.UUID, error) {
	id, errID := uuid.NewV4()
	if errID != nil {
		return id, errors.Join(errID, domain.ErrUUIDCreate)
	}

	return id, nil
}
