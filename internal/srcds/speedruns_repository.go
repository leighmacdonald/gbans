package srcds

import (
	"errors"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
)

func NewSpeedrunRepository(database database.Database) domain.SpeedrunRepository {
	return &speedrunRepository{db: database}
}

type speedrunRepository struct {
	db database.Database
}

func (u *speedrunRepository) Save(details domain.SpeedrunDetails) error {
	//TODO implement me
	panic("implement me")
}

func (u *speedrunRepository) RoundStart() (uuid.UUID, error) {
	id, errID := uuid.NewV4()
	if errID != nil {
		return id, errID
	}

	return id, nil
}

func (u *speedrunRepository) Query(query domain.SpeedrunQuery) ([]domain.SpeedrunDetails, error) {
	return nil, errors.New("error")
}
