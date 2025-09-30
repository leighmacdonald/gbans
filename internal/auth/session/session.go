package session

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain/person"
)

const ctxKeyUserProfile = "user_profile"

var ErrNotLoggedIn = errors.New("not logged in")

func CurrentUserProfile(ctx *gin.Context) (person.Core, error) {
	maybePerson, found := ctx.Get(ctxKeyUserProfile)
	if !found {
		return person.Core{}, ErrNotLoggedIn
	}

	profile, ok := maybePerson.(person.Core)
	if !ok {
		return person.Core{}, ErrNotLoggedIn
	}

	return profile, nil
}
