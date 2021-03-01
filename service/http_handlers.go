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
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const baseLayout = `<!doctype html>
    <html class="no-js" lang="en">
    <head>
        <meta charset="utf-8"/>
        <meta http-equiv="x-ua-compatible" content="ie=edge">
        <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
		<link rel="apple-touch-icon" sizes="180x180" href="/dist/apple-touch-icon.png">
		<link rel="icon" type="image/png" sizes="32x32" href="/dist/favicon-32x32.png">
		<link rel="icon" type="image/png" sizes="16x16" href="/dist/favicon-16x16.png">
		<link rel="manifest" href="/dist/site.webmanifest">
		<link rel="mask-icon" href="/dist/safari-pinned-tab.svg" color="#5bbad5">
		<meta name="msapplication-TileColor" content="#941739">
		<meta name="theme-color" content="#ffffff">
        <title>gbans</title>
    </head>
    <body>
    <div id="root"></div>
    <script src="/dist/bundle.js"></script>
    </body>
    </html>`

func onIndex() gin.HandlerFunc {
	//goland:noinspection HtmlUnknownTarget

	return func(c *gin.Context) {
		c.Data(200, gin.MIMEHTML, []byte(baseLayout))
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
			log.Errorf("Received malformed appeal apiBanRequest: %v", err)
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		c.JSON(http.StatusOK, gin.H{})
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
		Player  *model.Person         `json:"player"`
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
		person, err := GetOrCreatePersonBySteamID(sid)
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
		person.PlayerSummary = &sum[0]
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

type apiResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func responseErr(c *gin.Context, status int, data interface{}) {
	c.JSON(status, apiResponse{
		Status: false,
		Data:   data,
	})
}

func responseOK(c *gin.Context, status int, data interface{}) {
	c.JSON(status, apiResponse{
		Status: true,
		Data:   data,
	})
}

type apiBanRequest struct {
	SteamID    steamid.SID64 `json:"steam_id"`
	Duration   string        `json:"duration"`
	BanType    model.BanType `json:"ban_type"`
	Reason     model.Reason  `json:"reason"`
	ReasonText string        `json:"reason_text"`
	Network    string        `json:"network"`
}

func onAPIPostBanCreate() gin.HandlerFunc {
	return func(c *gin.Context) {
		var r apiBanRequest
		if err := c.BindJSON(&r); err != nil {
			responseErr(c, http.StatusBadRequest, "Failed to perform ban")
			return
		}
		duration, err := config.ParseDuration(r.Duration)
		if err != nil {
			responseErr(c, http.StatusNotAcceptable, `Invalid duration. Examples: "300m", "1.5h" or "2h45m". 
Valid time units are "s", "m", "h".`)
			return
		}
		var (
			n      *net.IPNet
			ban    *model.Ban
			banNet *model.BanNet
			e      error
		)
		if r.Network != "" {
			_, n, err = net.ParseCIDR(r.Network)
			if err != nil {
				responseErr(c, http.StatusBadRequest, "Invalid network cidr definition")
				return
			}
		}
		if !r.SteamID.Valid() {
			responseErr(c, http.StatusBadRequest, "Invalid steamid")
			return
		}

		if r.Network != "" {
			banNet, e = BanNetwork(c, n, r.SteamID, currentPerson(c).SteamID, duration, r.Reason, r.ReasonText, model.Web)
		} else {
			ban, e = BanPlayer(c, r.SteamID, currentPerson(c).SteamID, duration, r.Reason, r.ReasonText, model.Web)
		}
		if e != nil {
			if errors.Is(e, errDuplicate) {
				responseErr(c, http.StatusConflict, "Duplicate ban")
				return
			}
			responseErr(c, http.StatusInternalServerError, "Failed to perform ban")
			return
		}
		if r.Network != "" {
			responseOK(c, http.StatusCreated, banNet)
		} else {
			responseOK(c, http.StatusCreated, ban)
		}
	}
}
