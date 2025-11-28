package person

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/sync/errgroup"
)

const (
	steamQueryMaxResults = 100
	chunkSize            = 100
)

var ErrSteamUpdate = errors.New("failed to update data from steam")

// FetchPlayerBans batch fetches player bans in groups of 100.
// TODO remove and just call client direct? Not sure if the batching is really required.
func FetchPlayerBans(ctx context.Context, tfAPI thirdparty.APIProvider, steamIDs []steamid.SteamID) ([]thirdparty.SteamBan, error) {
	var (
		waitGroup = &sync.WaitGroup{}
		results   []thirdparty.SteamBan
		resultsMu = &sync.RWMutex{}
		hasErr    = int32(0)
	)

	for index := 0; index < len(steamIDs); index += chunkSize {
		waitGroup.Go(func() {
			var (
				total      = uint64(len(steamIDs) - index) //nolint:gosec
				maxResults = min(steamQueryMaxResults, total)
				ids        = steamIDs[index : index+int(maxResults)] // nolint:gosec
			)

			bans, errGetPlayerBans := tfAPI.SteamBans(ctx, ids)
			if errGetPlayerBans != nil {
				atomic.AddInt32(&hasErr, 1)

				return
			}

			resultsMu.Lock()
			results = append(results, bans...)
			resultsMu.Unlock()
		})
	}

	resultsMu.RLock()
	defer resultsMu.RUnlock()

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

func UpdatePlayerSummary(ctx context.Context, personUpdate *Person, tfAPI thirdparty.APIProvider) error {
	errGroup, errCtx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		summaries, errSummaries := tfAPI.Summaries(errCtx, steamid.Collection{personUpdate.SteamID})
		if errSummaries != nil {
			return errors.Join(errSummaries, ErrPlayerAPIFailed)
		}

		if len(summaries) == 0 {
			return ErrInvalidResult
		}
		personUpdate.AvatarHash = summaries[0].AvatarHash
		personUpdate.CommentPermission = summaries[0].CommentPermission
		personUpdate.LastLogoff = summaries[0].LastLogoff
		personUpdate.LocCityID = summaries[0].LocCityId
		personUpdate.LocCountryCode = summaries[0].LocCountryCode
		personUpdate.LocStateCode = summaries[0].LocStateCode
		personUpdate.PersonaName = summaries[0].PersonaName
		personUpdate.PersonaState = summaries[0].PersonaState
		personUpdate.PersonaStateFlags = summaries[0].PersonaStateFlags
		personUpdate.PrimaryClanID = summaries[0].PrimaryClanId
		personUpdate.ProfileState = summaries[0].ProfileState
		personUpdate.ProfileURL = summaries[0].ProfileUrl
		personUpdate.RealName = summaries[0].RealName
		personUpdate.TimeCreated = summaries[0].TimeCreated
		personUpdate.VisibilityState = summaries[0].VisibilityState

		return nil
	})

	errGroup.Go(func() error {
		vac, errBans := FetchPlayerBans(errCtx, tfAPI, steamid.Collection{personUpdate.SteamID})
		if errBans != nil || len(vac) != 1 {
			return errBans
		}

		personUpdate.CommunityBanned = vac[0].CommunityBanned
		personUpdate.VACBans = int(vac[0].NumberOfVacBans)
		personUpdate.GameBans = int(vac[0].NumberOfGameBans)
		personUpdate.EconomyBan = EconBanState(vac[0].EconomyBan)
		personUpdate.CommunityBanned = vac[0].CommunityBanned
		personUpdate.DaysSinceLastBan = int(vac[0].DaysSinceLastBan)
		personUpdate.UpdatedOnSteam = time.Now()

		return nil
	})

	if err := errGroup.Wait(); err != nil {
		return errors.Join(err, ErrSteamUpdate)
	}

	return nil
}
