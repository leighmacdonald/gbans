package session

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
)

const ctxKeyUserProfile = "user_profile"

var ErrNotLoggedIn = errors.New("not logged in")

func CurrentUserProfile(ctx *gin.Context) (domain.PersonCore, error) {
	maybePerson, found := ctx.Get(ctxKeyUserProfile)
	if !found {
		return domain.PersonCore{}, ErrNotLoggedIn
	}

	profile, ok := maybePerson.(domain.PersonCore)
	if !ok {
		return domain.PersonCore{}, ErrNotLoggedIn
	}

	return profile, nil
}
