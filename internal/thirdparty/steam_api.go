package thirdparty

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
)

const steamQueryMaxResults = 100

func FetchFriends(ctx context.Context, sid64 steamid.SID64) (steamid.Collection, error) {
	friends, errFriends := steamweb.GetFriendList(ctx, sid64)
	if errFriends != nil {
		return nil, errors.Wrap(errFriends, "Failed to fetch friends list")
	}

	var collection steamid.Collection

	for _, friend := range friends {
		collection = append(collection, friend.SteamID)
	}

	return collection, nil
}

const chunkSize = 100

func FetchSummaries(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerSummary, error) {
	var (
		waitGroup = &sync.WaitGroup{}
		results   []steamweb.PlayerSummary
		resultsMu = &sync.RWMutex{}
		hasErr    = int32(0)
	)

	for index := 0; index < len(steamIDs); index += chunkSize {
		waitGroup.Add(1)

		func() {
			defer waitGroup.Done()

			var (
				total           = uint64(len(steamIDs) - index)
				maxResultsCount = golib.UMin64(steamQueryMaxResults, total)
				ids             = steamIDs[index : index+int(maxResultsCount)]
			)

			summaries, errSummaries := steamweb.PlayerSummaries(ctx, ids)
			if errSummaries != nil {
				atomic.AddInt32(&hasErr, 1)
			}

			resultsMu.Lock()
			results = append(results, summaries...)
			resultsMu.Unlock()
		}()
	}

	if hasErr > 0 {
		return nil, errors.New("Failed to fetch all friends")
	}

	return results, nil
}

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
		return nil, errors.New("Failed to fetch all friends")
	}

	return results, nil
}
