package servers

import (
	"context"
	"errors"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
)

func NewSpeedrunUsecase(repo SpeedrunRepository) SpeedrunUsecase {
	return SpeedrunUsecase{repo: repo}
}

type SpeedrunUsecase struct {
	repo SpeedrunRepository
}

func (u *SpeedrunUsecase) Recent(ctx context.Context, limit int) ([]SpeedrunMapOverview, error) {
	if limit <= 0 || limit > 100 {
		return nil, domain.ErrValueOutOfRange
	}

	return u.repo.Recent(ctx, limit)
}

func (u *SpeedrunUsecase) TopNOverall(ctx context.Context, count int) (map[string][]Speedrun, error) {
	if count <= 0 || count > 1000 {
		return nil, domain.ErrValueOutOfRange
	}

	return u.repo.TopNOverall(ctx, count)
}

func (u *SpeedrunUsecase) ByID(ctx context.Context, speedrunID int) (Speedrun, error) {
	if speedrunID <= 0 {
		return Speedrun{}, domain.ErrValueOutOfRange
	}

	return u.repo.ByID(ctx, speedrunID)
}

func (u *SpeedrunUsecase) ByMap(ctx context.Context, mapName string) ([]SpeedrunMapOverview, error) {
	if mapName == "" {
		return []SpeedrunMapOverview{}, domain.ErrValueOutOfRange
	}

	return u.repo.ByMap(ctx, mapName)
}

func (u *SpeedrunUsecase) Save(ctx context.Context, details Speedrun) (Speedrun, error) {
	if len(details.PointCaptures) == 0 {
		return details, ErrInsufficientDetails
	}

	var validPlayers []SpeedrunParticipant //nolint:prealloc
	for _, player := range details.Players {
		if int64(details.Duration)/4 > int64(player.Duration) {
			continue
		}

		validPlayers = append(validPlayers, player)
	}

	details.Players = validPlayers

	if err := u.repo.Save(ctx, &details); err != nil {
		return Speedrun{}, err
	}

	return details, nil
}

func (u *SpeedrunUsecase) Query(_ context.Context, _ SpeedrunQuery) ([]Speedrun, error) {
	return nil, nil
}

func (u *SpeedrunUsecase) RoundStart() (uuid.UUID, error) {
	id, errID := uuid.NewV4()
	if errID != nil {
		return id, errors.Join(errID, domain.ErrUUIDCreate)
	}

	return id, nil
}
