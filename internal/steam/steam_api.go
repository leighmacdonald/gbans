// Package steam is used for communicating with the steam api using the steamweb package.
package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
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

func FetchFriends(sid64 steamid.SID64) (steamid.Collection, error) {
	const baseURL = "https://api.steampowered.com/ISteamUser" +
		"/GetFriendList/v0001/?key=%s&steamid=%d&relationship=all&format=json"
	u := fmt.Sprintf(baseURL, config.General.SteamKey, sid64)
	req, errReq := http.NewRequestWithContext(context.Background(), "GET", u, nil)
	if errReq != nil {
		return nil, errors.Wrap(errReq, "Failed to create new request")
	}
	c := &http.Client{Timeout: time.Second * 5}
	resp, errDo := c.Do(req)
	if errDo != nil {
		return nil, errors.Wrap(errDo, "Failed to fetch friends list")
	}
	body, errRead := ioutil.ReadAll(resp.Body)
	if errRead != nil {
		return nil, errors.Wrap(errRead, "Failed to read response body")
	}
	var flr getFriendListResponse
	if errUnmarshal := json.Unmarshal(body, &flr); errUnmarshal != nil {
		return nil, errors.Wrap(errUnmarshal, "Failed to decode response body")
	}
	var fl steamid.Collection
	for _, friend := range flr.FriendsList.Friends {
		sid, errSid := steamid.SID64FromString(friend.Steamid)
		if errSid == nil {
			fl = append(fl, sid)
		}
	}
	return fl, nil
}

const chunkSize = 100

func FetchSummaries(steamIDs steamid.Collection) ([]steamweb.PlayerSummary, error) {
	wg := &sync.WaitGroup{}
	var (
		results   []steamweb.PlayerSummary
		resultsMu = &sync.RWMutex{}
	)
	hasErr := int32(0)
	for i := 0; i < len(steamIDs); i += chunkSize {
		wg.Add(1)
		func() {
			defer wg.Done()
			t := uint64(len(steamIDs) - i)
			m := golib.UMin64(steamQueryMaxResults, t)
			ids := steamIDs[i : i+int(m)]
			summaries, err := steamweb.PlayerSummaries(ids)
			if err != nil {
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

func FetchPlayerBans(steamIDs []steamid.SID64) ([]steamweb.PlayerBanState, error) {
	wg := &sync.WaitGroup{}
	var (
		results   []steamweb.PlayerBanState
		resultsMu = &sync.RWMutex{}
	)
	hasErr := int32(0)
	for i := 0; i < len(steamIDs); i += chunkSize {
		wg.Add(1)
		func() {
			defer wg.Done()
			t := uint64(len(steamIDs) - i)
			m := golib.UMin64(steamQueryMaxResults, t)
			ids := steamIDs[i : i+int(m)]

			bans, err := steamweb.GetPlayerBans(ids)
			if err != nil {
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
