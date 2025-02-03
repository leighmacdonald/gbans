package playerqueue

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

func NewPlayerqueueUsecase(repo domain.PlayerqueueRepository) domain.PlayerqueueUsecase {
	return &playerqueueUsecase{repo: repo}
}

type playerqueueUsecase struct {
	repo domain.PlayerqueueRepository
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
