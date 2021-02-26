package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

func onIndex() gin.HandlerFunc {
	return func(c *gin.Context) {
		render(c, "home", defaultArgs(c))
	}
}

func onGetServers() gin.HandlerFunc {
	return func(c *gin.Context) {
		serverStateMu.RLock()
		state := serverState
		serverStateMu.RUnlock()
		a := defaultArgs(c)
		a.V["servers"] = state
		render(c, "servers", a)
	}
}

func onGetBans() gin.HandlerFunc {
	return func(c *gin.Context) {
		render(c, "bans", defaultArgs(c))
	}
}

func onGetBanPlayer() gin.HandlerFunc {
	return func(c *gin.Context) {
		render(c, "ban_player", defaultArgs(c))
	}
}

func onGetAppeal() gin.HandlerFunc {
	return func(c *gin.Context) {
		usr := currentPerson(c)
		ban, err := GetBan(usr.SteamID)
		if err != nil {
			if errors.Is(err, errNoResult) {
				flash(c, lError, "No Ban Found", "Please login with the account in question")
				c.Redirect(http.StatusTemporaryRedirect, c.Request.Referer())
				return
			} else {
				log.Errorf("Failed to lookup ban: %v", err)
				c.String(http.StatusInternalServerError, "oops")
				return
			}
		}
		args := defaultArgs(c)
		args.V["ban"] = ban
		render(c, "appeal", args)
	}
}

func onAPIGetServers() gin.HandlerFunc {
	return func(c *gin.Context) {
		servers, err := getServers()
		if err != nil {
			log.Errorf("Failed to fetch servers: %s", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, servers)
	}
}

func onAPIPostAppeal() gin.HandlerFunc {
	type req struct {
		Email      string `json:"email"`
		AppealText string `json:"appeal_text"`
	}
	return func(c *gin.Context) {
		var app req
		if err := c.BindJSON(&app); err != nil {
			log.Errorf("Received malformed appeal req: %v", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		c.JSON(http.StatusOK, gin.H{})
	}
}

func onAdminFilteredWords() gin.HandlerFunc {
	return func(c *gin.Context) {
		words, err := GetFilteredWords()
		if err != nil {
			log.Errorf("Failed to load filtered word sets from db: %v", err)
			c.Redirect(http.StatusTemporaryRedirect, c.Request.Referer())
			return
		}
		args := defaultArgs(c)
		args.V["words"] = words
		render(c, "admin_filtered_words", args)
	}
}

func onGetProfileSettings() gin.HandlerFunc {
	return func(c *gin.Context) {
		render(c, "profile_settings", defaultArgs(c))
	}
}

func onGetAdminImport() gin.HandlerFunc {
	return func(c *gin.Context) {
		render(c, "admin_import", defaultArgs(c))
	}
}

func onGetAdminServers() gin.HandlerFunc {
	return func(c *gin.Context) {
		render(c, "admin_servers", defaultArgs(c))
	}
}

func onGetAdminPeople() gin.HandlerFunc {
	return func(c *gin.Context) {
		render(c, "admin_people", defaultArgs(c))
	}
}

func onAPIPostReport() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{})
	}
}

func onAPIProfile() gin.HandlerFunc {
	type req struct {
		Query string `form:"query"`
	}
	type resp struct {
		Player  model.Person          `json:"player"`
		Friends []extra.PlayerSummary `json:"friends"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.Bind(&r); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		cx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		sid, err := steamid.StringToSID64(r.Query)
		if err != nil {
			sid, err = steamid.ResolveSID64(cx, r.Query)
			if err != nil {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}
		}
		person, err := getOrCreatePersonBySteamID(sid)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		sum, err := extra.PlayerSummaries(cx, []steamid.SID64{sid})
		if err != nil || len(sum) != 1 {
			log.Errorf("Failed to get player summary: %v", err)
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		person.PlayerSummary = sum[0]
		friendIDs, err := fetchFriends(person.SteamID)
		if err != nil {
			log.Error("Could not fetch friends")
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		friends, err := fetchSummaries(friendIDs)
		if err != nil {
			log.Error("Could not fetch summaries")
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		var response resp
		response.Player = person
		response.Friends = friends
		c.JSON(http.StatusOK, response)
	}
}

type getFriendListRequest struct {
	SteamIDs []steamid.SID64
}

type getFriendListResponse struct {
	Friendslist struct {
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
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
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
	for _, friend := range flr.Friendslist.Friends {
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
	c, cancel := context.WithTimeout(ctx, time.Second*10)
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
			for _, s := range summaries {
				results = append(results, s)
			}
			resultsMu.Unlock()
		}()
	}
	if hasErr > 0 {
		return nil, errors.New("Failed to fetch all friends")
	}
	return results, nil
}

func onAPIPostBan() gin.HandlerFunc {
	type req struct {
		SteamID    steamid.SID64 `json:"steam_id"`
		AuthorID   steamid.SID64 `json:"author_id"`
		Duration   string        `json:"duration"`
		BanType    model.BanType `json:"ban_type"`
		Reason     model.Reason  `json:"reason"`
		ReasonText string        `json:"reason_text"`
	}

	return func(c *gin.Context) {
		var r req
		if err := c.BindJSON(&r); err != nil {
			c.JSON(http.StatusBadRequest, StatusResponse{
				Success: false,
				Message: "Failed to perform ban",
			})
			return
		}
		duration, err := time.ParseDuration(r.Duration)
		if err != nil {
			c.JSON(http.StatusNotAcceptable, StatusResponse{
				Success: false,
				Message: `Invalid duration. Examples: "300m", "1.5h" or "2h45m". 
Valid time units are "s", "m", "h".`,
			})
		}
		if err := BanPlayer(c, r.SteamID, r.AuthorID, duration, r.Reason, r.ReasonText, model.Web); err != nil {
			c.JSON(http.StatusNotAcceptable, StatusResponse{
				Success: false,
				Message: "Failed to perform ban",
			})
		}
	}
}
