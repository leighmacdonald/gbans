package steam

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/sync/errgroup"
)

const (
	steamQueryMaxResults = 100
	chunkSize            = 100
)

var ErrSteamUpdate = errors.New("failed to update data from steam")

// TODO remove and just call client direct? Not sure if the batching is really required.
func FetchPlayerBans(ctx context.Context, tfAPI *thirdparty.TFAPI, steamIDs []steamid.SteamID) ([]thirdparty.SteamBan, error) {
	var (
		waitGroup = &sync.WaitGroup{}
		results   []thirdparty.SteamBan
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

			bans, errGetPlayerBans := tfAPI.Client.SteamBansWithResponse(ctx, &thirdparty.SteamBansParams{
				Steamids: strings.Join(steamid.Collection(ids).ToStringSlice(), ","),
			})
			if errGetPlayerBans != nil || bans.JSON200 == nil {
				atomic.AddInt32(&hasErr, 1)

				return
			}

			resultsMu.Lock()
			results = append(results, *bans.JSON200...)
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

func UpdatePlayerSummary(ctx context.Context, person *domain.Person, tfAPI *thirdparty.TFAPI) error {
	errGroup, errCtx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		summaries, errSummaries := tfAPI.Summaries(errCtx, steamid.Collection{person.SteamID})
		if errSummaries != nil {
			return errors.Join(errSummaries, ErrPlayerAPIFailed)
		}

		if len(summaries) == 0 {
			return ErrInvalidResult
		}
		person.AvatarHash = summaries[0].AvatarHash
		person.CommentPermission = summaries[0].CommentPermission
		person.LastLogoff = summaries[0].LastLogoff
		person.LocCityID = summaries[0].LocCityId
		person.LocCountryCode = summaries[0].LocCountryCode
		person.LocStateCode = summaries[0].LocStateCode
		person.PersonaName = summaries[0].PersonaName
		person.PersonaState = summaries[0].PersonaState
		person.PersonaStateFlags = summaries[0].PersonaStateFlags
		person.PrimaryClanID = summaries[0].PrimaryClanId
		person.ProfileState = summaries[0].ProfileState
		person.ProfileURL = summaries[0].ProfileUrl
		person.RealName = summaries[0].RealName
		person.TimeCreated = summaries[0].TimeCreated
		person.VisibilityState = summaries[0].VisibilityState

		return nil
	})

	errGroup.Go(func() error {
		vac, errBans := FetchPlayerBans(errCtx, tfAPI, steamid.Collection{person.SteamID})
		if errBans != nil || len(vac) != 1 {
			return errBans
		}

		person.CommunityBanned = vac[0].CommunityBanned
		person.VACBans = int(vac[0].NumberOfVacBans)
		person.GameBans = int(vac[0].NumberOfGameBans)
		person.EconomyBan = domain.EconBanState(vac[0].EconomyBan)
		person.CommunityBanned = vac[0].CommunityBanned
		person.DaysSinceLastBan = int(vac[0].DaysSinceLastBan)
		person.UpdatedOnSteam = time.Now()

		return nil
	})

	if err := errGroup.Wait(); err != nil {
		return errors.Join(err, ErrSteamUpdate)
	}

	return nil
}
