package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type getFriendListResponse struct {
	FriendsList struct {
		Friends []struct {
			Steamid      string `json:"steamid"`
			Relationship string `json:"relationship"`
			FriendSince  int    `json:"friend_since"`
		} `json:"friends"`
	} `json:"friendslist"`
}

func fetchFriends(sid64 steamid.SID64) ([]steamid.SID64, error) {
	const baseURL = "https://api.steampowered.com/ISteamUser" +
		"/GetFriendList/v0001/?key=%s&steamid=%d&relationship=all&format=json"
	u := fmt.Sprintf(baseURL, config.General.SteamKey, sid64)
	req, err := http.NewRequestWithContext(gCtx, "GET", u, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create new request")
	}
	c := &http.Client{Timeout: time.Second * 5}
	resp, err := c.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch friends list")
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read response body")
	}
	var flr getFriendListResponse
	if err := json.Unmarshal(b, &flr); err != nil {
		return nil, errors.Wrap(err, "Failed to decode response body")
	}
	var fl []steamid.SID64
	for _, friend := range flr.FriendsList.Friends {
		sid, err2 := steamid.SID64FromString(friend.Steamid)
		if err2 == nil {
			fl = append(fl, sid)
		}
	}
	return fl, nil
}

func fetchSummaries(steamIDs []steamid.SID64) ([]extra.PlayerSummary, error) {
	const chunkSize = 100
	wg := &sync.WaitGroup{}
	c, cancel := context.WithTimeout(gCtx, time.Second*10)
	defer cancel()
	var (
		results   []extra.PlayerSummary
		resultsMu = &sync.RWMutex{}
	)
	hasErr := int32(0)
	for i := 0; i < len(steamIDs); i += chunkSize {
		wg.Add(1)
		func() {
			defer wg.Done()
			t := uint64(len(steamIDs) - i)
			m := golib.UMin64(100, t)
			ids := steamIDs[i : i+int(m)]
			summaries, err := extra.PlayerSummaries(c, ids)
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

// getOrCreateProfileBySteamID functions the same as GetOrCreatePersonBySteamID except
// that it will also query the steam webapi to fetch and load the extra player summary info
func getOrCreateProfileBySteamID(sid steamid.SID64, ipAddr string) (*model.Person, error) {
	sum, err := extra.PlayerSummaries(context.Background(), []steamid.SID64{sid})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get player summary: %v", err)
	}
	p, err := GetOrCreatePersonBySteamID(sid)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get person: %v", err)
	}
	s := sum[0]
	p.SteamID = sid
	if ipAddr != "" {
		p.IPAddr = ipAddr
	}
	p.PlayerSummary = &s
	if errSave := SavePerson(p); errSave != nil {
		return nil, errors.Wrapf(errSave, "Failed to save person")
	}
	return p, nil
}
