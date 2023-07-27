package app

import (
	"context"
	"time"

	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

// ResolveSID is just a simple helper for calling steamid.ResolveSID64 with a timeout.
func ResolveSID(ctx context.Context, sidStr string) (steamid.SID64, error) {
	localCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	sid, errResolve := steamid.ResolveSID64(localCtx, sidStr)
	if errResolve != nil {
		return "", errors.Wrap(errResolve, "Failed to resolve vanity")
	}

	return sid, nil
}
