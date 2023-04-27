package query

import (
	"context"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"time"
)

// ResolveSID is just a simple helper for calling steamid.ResolveSID64 with a timeout
func ResolveSID(ctx context.Context, sidStr string) (steamid.SID64, error) {
	localCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	return steamid.ResolveSID64(localCtx, sidStr)
}
