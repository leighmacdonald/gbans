package thirdparty

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

const steamQueryMaxResults = 100

const chunkSize = 100

func FetchPlayerBans(ctx context.Context, steamIDs []steamid.SID64) ([]steamweb.PlayerBanState, error) {
	var (
		waitGroup = &sync.WaitGroup{}
		results   []steamweb.PlayerBanState
		resultsMu = &sync.RWMutex{}
		hasErr    = int32(0)
	)

	for index := 0; index < len(steamIDs); index += chunkSize {
		waitGroup.Add(1)

		func() {
			defer waitGroup.Done()

			var (
				total      = uint64(len(steamIDs) - index)
				maxResults = golib.UMin64(steamQueryMaxResults, total)
				ids        = steamIDs[index : index+int(maxResults)]
			)

			bans, errGetPlayerBans := steamweb.GetPlayerBans(ctx, ids)
			if errGetPlayerBans != nil {
				atomic.AddInt32(&hasErr, 1)
			}

			resultsMu.Lock()
			results = append(results, bans...)
			resultsMu.Unlock()
		}()
	}

	if hasErr > 0 {
		return nil, ErrSteamBans
	}

	return results, nil
}

// ResolveSID is just a simple helper for calling steamid.ResolveSID64 with a timeout.
func ResolveSID(ctx context.Context, sidStr string) (steamid.SID64, error) {
	localCtx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	sid64, errString := steamid.StringToSID64(sidStr)
	if errString == nil && sid64.Valid() {
		return sid64, nil
	}

	sid, errResolve := steamid.ResolveSID64(localCtx, sidStr)
	if errResolve != nil {
		return "", errors.Join(errResolve, ErrResolveVanity)
	}

	return sid, nil
}

var (
	ErrPlayerAPIFailed = errors.New("could not update player from steam api")
	ErrResolveVanity   = errors.New("failed to resolve vanity")
	ErrInvalidResult   = errors.New("invalid response received")
	ErrSteamBans       = errors.New("failed to fetch player bans")
)

func UpdatePlayerSummary(ctx context.Context, person *domain.Person) error {
	summaries, errSummaries := steamweb.PlayerSummaries(ctx, steamid.Collection{person.SteamID})
	if errSummaries != nil {
		return errors.Join(errSummaries, ErrPlayerAPIFailed)
	}

	if len(summaries) > 0 {
		s := summaries[0]
		person.PlayerSummary = &s
	} else {
		return ErrInvalidResult
	}

	vac, errBans := FetchPlayerBans(ctx, steamid.Collection{person.SteamID})
	if errBans != nil || len(vac) != 1 {
		return errBans
	} else {
		person.CommunityBanned = vac[0].CommunityBanned
		person.VACBans = vac[0].NumberOfVACBans
		person.GameBans = vac[0].NumberOfGameBans
		person.EconomyBan = steamweb.EconBanNone
		person.CommunityBanned = vac[0].CommunityBanned
		person.DaysSinceLastBan = vac[0].DaysSinceLastBan
	}

	person.UpdatedOnSteam = time.Now()

	return nil
}
