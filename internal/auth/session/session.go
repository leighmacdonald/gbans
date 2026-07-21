package session

import (
	"context"
	"errors"

	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

var ErrNotLoggedIn = errors.New("not logged in")

func CurrentUserProfile(ctx context.Context) (person.Core, error) {
	maybePerson := ctx.Value(httphelper.CtxKeyUserProfile)
	if maybePerson == nil {
		return person.Core{}, ErrNotLoggedIn
	}

	profile, ok := maybePerson.(person.Core)
	if !ok {
		return person.Core{}, ErrNotLoggedIn
	}

	return profile, nil
}
