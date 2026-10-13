package person

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/gbans/internal/thirdparty"
	steamv1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/steam/v1"
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
func FetchPlayerBans(ctx context.Context, tfAPI thirdparty.APIProvider, steamIDs []steamid.SteamID) ([]*steamv1.SteamBan, error) {
	var (
		waitGroup = &sync.WaitGroup{}
		results   []*steamv1.SteamBan
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

	waitGroup.Wait()

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
		personUpdate.AvatarHash = summaries[0].GetAvatarHash()
		personUpdate.CommentPermission = summaries[0].GetCommentPermission()
		personUpdate.LastLogoff = new(time.Unix(int64(summaries[0].GetLastLogoff()), 0))
		personUpdate.LocCityID = summaries[0].GetLocCityId()
		personUpdate.LocCountryCode = summaries[0].GetLocCountryCode()
		personUpdate.LocStateCode = summaries[0].GetLocStateCode()
		personUpdate.PersonaName = summaries[0].GetPersonaName()
		personUpdate.PersonaState = summaries[0].GetPersonaState()
		personUpdate.PersonaStateFlags = summaries[0].GetPersonaStateFlags()
		personUpdate.PrimaryClanID = summaries[0].GetPrimaryClanId()
		personUpdate.ProfileState = summaries[0].GetProfileState()
		personUpdate.ProfileURL = summaries[0].GetProfileUrl()
		personUpdate.RealName = summaries[0].GetRealName()
		personUpdate.TimeCreated = int64(summaries[0].GetTimeCreated())
		personUpdate.VisibilityState = summaries[0].GetVisibilityState()

		return nil
	})

	errGroup.Go(func() error {
		vac, errBans := FetchPlayerBans(errCtx, tfAPI, steamid.Collection{personUpdate.SteamID})
		if errBans != nil || len(vac) != 1 {
			return errBans
		}

		personUpdate.CommunityBanned = vac[0].GetCommunityBanned()
		personUpdate.VACBans = vac[0].GetNumberOfVacBans()
		personUpdate.GameBans = vac[0].GetNumberOfGameBans()
		personUpdate.EconomyBan = EconBanState(vac[0].GetEconomyBan())
		personUpdate.CommunityBanned = vac[0].GetCommunityBanned()
		personUpdate.DaysSinceLastBan = vac[0].GetDaysSinceLastBan()
		personUpdate.UpdatedOnSteam = time.Now()

		return nil
	})

	if err := errGroup.Wait(); err != nil {
		return errors.Join(err, ErrSteamUpdate)
	}

	return nil
}
