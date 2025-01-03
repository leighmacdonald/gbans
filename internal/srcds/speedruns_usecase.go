package srcds

import (
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

func (u *speedrunUsecase) RoundStart() (uuid.UUID, error) {
	id, errID := uuid.NewV4()
	if errID != nil {
		return id, errID
	}

	return id, nil
}

func (u *speedrunUsecase) RoundFinish(details domain.SpeedrunDetails) error {
	if len(details.Rounds) == 0 {
		return domain.ErrInsufficientDetails
	}

	var validPlayers []domain.SpeedrunPlayer
	for _, player := range details.Players {
		if details.Time/2 > player.Time {
			continue
		}

		validPlayers = append(validPlayers, player)
	}

	details.Players = validPlayers

	return u.repo.Save(details)
}

func (u *speedrunUsecase) Query(query domain.SpeedrunQuery) ([]domain.SpeedrunDetails, error) {
	return nil, errors.New("error")
}
