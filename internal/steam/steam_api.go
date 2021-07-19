package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steam-webapi"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const SteamQueryMaxResults = 100

var errTooManySteamIds = errors.Errorf("Max %d ids per steam api request", SteamQueryMaxResults)

type getFriendListResponse struct {
	FriendsList struct {
		Friends []struct {
			Steamid      string `json:"steamid"`
			Relationship string `json:"relationship"`
			FriendSince  int    `json:"friend_since"`
		} `json:"friends"`
	} `json:"friendslist"`
}

func FetchFriends(sid64 steamid.SID64) ([]steamid.SID64, error) {
	const baseURL = "https://api.steampowered.com/ISteamUser" +
		"/GetFriendList/v0001/?key=%s&steamid=%d&relationship=all&format=json"
	u := fmt.Sprintf(baseURL, config.General.SteamKey, sid64)
	req, err := http.NewRequestWithContext(context.Background(), "GET", u, nil)
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

func FetchSummaries(steamIDs steamid.Collection) ([]steam_webapi.PlayerSummary, error) {
	const chunkSize = 100
	wg := &sync.WaitGroup{}
	var (
		results   []steam_webapi.PlayerSummary
		resultsMu = &sync.RWMutex{}
	)
	hasErr := int32(0)
	for i := 0; i < len(steamIDs); i += chunkSize {
		wg.Add(1)
		func() {
			defer wg.Done()
			t := uint64(len(steamIDs) - i)
			m := golib.UMin64(SteamQueryMaxResults, t)
			ids := steamIDs[i : i+int(m)]
			summaries, err := steam_webapi.PlayerSummaries(ids)
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

func FetchPlayerBans(ctx context.Context, steamIDs []steamid.SID64) ([]*VACState, error) {
	const chunkSize = 100
	wg := &sync.WaitGroup{}
	var (
		results   []*VACState
		resultsMu = &sync.RWMutex{}
	)
	hasErr := int32(0)
	for i := 0; i < len(steamIDs); i += chunkSize {
		wg.Add(1)
		func() {
			defer wg.Done()
			t := uint64(len(steamIDs) - i)
			m := golib.UMin64(SteamQueryMaxResults, t)
			ids := steamIDs[i : i+int(m)]
			summaries, err := QueryVacStatus(ctx, ids)
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

type VACState struct {
	SteamID          steamid.SID64 `json:"SteamId"`
	CommunityBanned  bool          `json:"CommunityBanned"`
	VACBanned        bool          `json:"VACBanned"`
	VACBans          int           `json:"NumberOfVACBans"`
	GameBans         int           `json:"NumberOfGameBans"`
	EconomyBan       string        `json:"EconomyBan"`
	DaysSinceLastBan int           `json:"DaysSinceLastBan"`
}

func QueryVacStatus(ctx context.Context, steamIds []steamid.SID64) ([]*VACState, error) {
	type container struct {
		Players []*VACState `json:"players"`
	}
	const q = "https://api.steampowered.com/ISteamUser/GetPlayerBans/v1"
	if len(steamIds) > SteamQueryMaxResults {
		return nil, errTooManySteamIds
	}
	c := util.NewHTTPClient()
	req, errReq := http.NewRequestWithContext(ctx, "GET", q, nil)
	if errReq != nil {
		return nil, errReq
	}
	var strIds []string
	for _, sid := range steamIds {
		strIds = append(strIds, sid.String())
	}
	qu := req.URL.Query()
	qu.Set("steamids", strings.Join(strIds, ","))
	qu.Set("key", steamid.GetKey())
	req.URL.RawQuery = qu.Encode()
	r, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	b, errB := ioutil.ReadAll(r.Body)
	if errB != nil {
		return nil, errB
	}
	defer func() {
		if errResp := r.Body.Close(); errResp != nil {
			log.Warnf("Failed to close response body: %v", errResp)
		}
	}()
	var p container
	if err2 := json.Unmarshal(b, &p); err2 != nil {
		return nil, err2
	}
	return p.Players, nil
}
