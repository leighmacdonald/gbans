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

func (u *speedrunUsecase) ByID(ctx context.Context, speedrunID int) (domain.Speedrun, error) {
	if speedrunID <= 0 {
		return domain.Speedrun{}, domain.ErrValueOutOfRange
	}

	return u.repo.ByID(ctx, speedrunID)
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
