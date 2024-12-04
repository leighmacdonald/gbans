package thirdparty

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

const steamQueryMaxResults = 100

const chunkSize = 100

func FetchPlayerBans(ctx context.Context, steamIDs []steamid.SteamID) ([]steamweb.PlayerBanState, error) {
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
				total      = uint64(len(steamIDs) - index) //nolint:gosec
				maxResults = min(steamQueryMaxResults, total)
				ids        = steamIDs[index : index+int(maxResults)] // nolint:gosec
			)

			bans, errGetPlayerBans := steamweb.GetPlayerBans(ctx, httphelper.NewHTTPClient(), ids)
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

var (
	ErrPlayerAPIFailed = errors.New("could not update player from steam api")
	ErrInvalidResult   = errors.New("invalid response received")
	ErrSteamBans       = errors.New("failed to fetch player bans")
)

func UpdatePlayerSummary(ctx context.Context, person *domain.Person) error {
	summaries, errSummaries := steamweb.PlayerSummaries(ctx, httphelper.NewHTTPClient(), steamid.Collection{person.SteamID})
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
	}

	person.CommunityBanned = vac[0].CommunityBanned
	person.VACBans = vac[0].NumberOfVACBans
	person.GameBans = vac[0].NumberOfGameBans
	person.EconomyBan = steamweb.EconBanNone
	person.CommunityBanned = vac[0].CommunityBanned
	person.DaysSinceLastBan = vac[0].DaysSinceLastBan
	person.UpdatedOnSteam = time.Now()

	return nil
}
