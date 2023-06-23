package thirdparty

import (
	"context"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"sync"
	"sync/atomic"
)

const steamQueryMaxResults = 100

type getFriendListResponse struct {
	FriendsList struct {
		Friends []struct {
			Steamid      string `json:"steamid"`
			Relationship string `json:"relationship"`
			FriendSince  int    `json:"friend_since"`
		} `json:"friends"`
	} `json:"friendslist"`
}

func FetchFriends(ctx context.Context, sid64 steamid.SID64) (steamid.Collection, error) {
	friends, errFriends := steamweb.GetFriendList(ctx, sid64)
	if errFriends != nil {
		return nil, errors.Wrap(errFriends, "Failed to fetch friends list")
	}
	var fl steamid.Collection
	for _, friend := range friends {
		fl = append(fl, friend.SteamID)
	}
	return fl, nil
}

const chunkSize = 100

func FetchSummaries(ctx context.Context, steamIDs steamid.Collection) ([]steamweb.PlayerSummary, error) {
	waitGroup := &sync.WaitGroup{}
	var (
		results   []steamweb.PlayerSummary
		resultsMu = &sync.RWMutex{}
	)
	hasErr := int32(0)
	for i := 0; i < len(steamIDs); i += chunkSize {
		waitGroup.Add(1)
		func() {
			defer waitGroup.Done()
			total := uint64(len(steamIDs) - i)
			maxResultsCount := golib.UMin64(steamQueryMaxResults, total)
			ids := steamIDs[i : i+int(maxResultsCount)]
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
	waitGroup := &sync.WaitGroup{}
	var (
		results   []steamweb.PlayerBanState
		resultsMu = &sync.RWMutex{}
	)
	hasErr := int32(0)
	for i := 0; i < len(steamIDs); i += chunkSize {
		waitGroup.Add(1)
		func() {
			defer waitGroup.Done()
			total := uint64(len(steamIDs) - i)
			maxResults := golib.UMin64(steamQueryMaxResults, total)
			ids := steamIDs[i : i+int(maxResults)]

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
