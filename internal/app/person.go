package app

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type PersonStore interface {
	GetOrCreatePersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *model.Person) error
	SavePerson(ctx context.Context, person *model.Person) error
}
