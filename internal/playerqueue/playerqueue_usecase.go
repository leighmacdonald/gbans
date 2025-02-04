package playerqueue

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

func NewPlayerqueueUsecase(repo domain.PlayerqueueRepository) domain.PlayerqueueUsecase {
	return &playerqueueUsecase{repo: repo}
}

type playerqueueUsecase struct {
	repo    domain.PlayerqueueRepository
	persons domain.PersonUsecase
}

func (p playerqueueUsecase) Delete(ctx context.Context, messageID ...uuid.UUID) error {
	if len(messageID) == 0 {
		return nil
	}

	return p.repo.Delete(ctx, messageID...)
}

func (p playerqueueUsecase) SetChatStatus(ctx context.Context, steamID steamid.SteamID, status domain.ChatStatus) error {
	if !steamID.Valid() {
		return domain.ErrInvalidSID
	}

	person, errPerson := p.persons.GetOrCreatePersonBySteamID(ctx, nil, steamID)
	if errPerson != nil {
		return errPerson
	}

	person.PlayerqueueChatStatus = status

	if errSave := p.persons.SavePerson(ctx, nil, &person); errSave != nil {
		return errSave
	}

	return nil
}

func (p playerqueueUsecase) Add(ctx context.Context, message domain.Message) (domain.Message, error) {
	if len(message.BodyMD) == 0 {
		return domain.Message{}, domain.ErrInvalidParameter
	}

	sid := steamid.New(message.SteamID)

	if !sid.Valid() {
		return domain.Message{}, domain.ErrInvalidSID
	}

	return p.repo.Save(ctx, message)
}

func (p playerqueueUsecase) Recent(ctx context.Context, limit uint64) ([]domain.Message, error) {
	if limit == 0 {
		limit = 50
	}

	return p.repo.Query(ctx, domain.PlayerqueueQueryOpts{
		QueryFilter: domain.QueryFilter{
			Limit:   limit,
			Desc:    true,
			OrderBy: "message_id",
			Deleted: false,
		},
	})
}
