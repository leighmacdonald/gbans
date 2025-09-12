package session

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
)

const ctxKeyUserProfile = "user_profile"

var ErrNotLoggedIn = errors.New("not logged in")

func CurrentUserProfile(ctx *gin.Context) (domain.PersonInfo, error) {
	maybePerson, found := ctx.Get(ctxKeyUserProfile)
	if !found {
		return nil, ErrNotLoggedIn
	}

	profile, ok := maybePerson.(domain.PersonInfo)
	if !ok {
		return nil, ErrNotLoggedIn
	}

	return profile, nil
}
