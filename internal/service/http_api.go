package service

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/pkg/errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	log "github.com/sirupsen/logrus"
)

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

func onPostPingMod() gin.HandlerFunc {
	type pingReq struct {
		ServerName string        `json:"server_name"`
		Name       string        `json:"name"`
		SteamID    steamid.SID64 `json:"steam_id"`
		Reason     string        `json:"reason"`
		Client     int           `json:"client"`
	}
	return func(c *gin.Context) {
		var req pingReq
		if err := c.BindJSON(&req); err != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		for _, c := range config.Discord.ModChannels {
			sendMessage(newMessage(c, fmt.Sprintf("<@&%d> %s", config.Discord.ModRoleID, req.Reason)))
		}
		responseOK(c, http.StatusOK, gin.H{
			"client":  req.Client,
			"message": "Moderators have been notified",
		})
	}
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

func onSAPIPostServerAuth() gin.HandlerFunc {
	type authReq struct {
		ServerName string `json:"server_name"`
		Key        string `json:"key"`
	}
	type authResp struct {
		Status bool   `json:"status"`
		Token  string `json:"token"`
	}
	return func(c *gin.Context) {
		var req authReq
		if err := c.BindJSON(&req); err != nil {
			log.Errorf("Failed to decode auth request: %v", err)
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		srv, err := getServerByName(req.ServerName)
		if err != nil {
			responseErr(c, http.StatusNotFound, nil)
			return
		}
		srv.Token = golib.RandomString(40)
		srv.TokenCreatedOn = config.Now()
		if err := SaveServer(&srv); err != nil {
			log.Errorf("Failed to updated server token: %v", err)
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		responseOK(c, http.StatusOK, authResp{
			Status: true,
			Token:  srv.Token,
		})
	}
}

func onPostServerCheck() gin.HandlerFunc {
	type checkRequest struct {
		ClientID int    `json:"client_id"`
		SteamID  string `json:"steam_id"`
		IP       net.IP `json:"ip"`
	}
	type checkResponse struct {
		ClientID int           `json:"client_id"`
		SteamID  string        `json:"steam_id"`
		BanType  model.BanType `json:"ban_type"`
		Msg      string        `json:"msg"`
	}
	return func(c *gin.Context) {
		var req checkRequest
		if err := c.BindJSON(&req); err != nil {
			responseErr(c, http.StatusInternalServerError, checkResponse{
				BanType: model.Unknown,
				Msg:     "Error determining state",
			})
			return
		}
		resp := checkResponse{
			ClientID: req.ClientID,
			SteamID:  req.SteamID,
			BanType:  model.Unknown,
			Msg:      "",
		}
		// Check IP first
		banNet, err := getBanNet(req.IP)
		if err != nil {
			responseErr(c, http.StatusInternalServerError, checkResponse{
				BanType: model.Unknown,
				Msg:     "Error determining state",
			})
			log.Errorf("Could not get ban net results: %v", err)
			return
		}
		if len(banNet) > 0 {
			resp.BanType = model.Banned
			resp.Msg = fmt.Sprintf("Network banned (C: %d)", len(banNet))
			responseOK(c, http.StatusOK, resp)
			return
		}
		// Check SteamID
		steamID, err := steamid.ResolveSID64(context.Background(), req.SteamID)
		if err != nil || !steamID.Valid() {
			resp.Msg = "Invalid steam id"
			responseErr(c, http.StatusBadRequest, resp)
			return
		}
		ban, err := getBanBySteamID(steamID, false)
		if err != nil {
			if dbErr(err) == errNoResult {
				resp.BanType = model.OK
				responseErr(c, http.StatusOK, resp)
				return
			}
			resp.Msg = "Error determining state"
			responseErr(c, http.StatusInternalServerError, resp)
			return
		}
		resp.BanType = ban.Ban.BanType
		resp.Msg = ban.Ban.ReasonText
		responseOK(c, http.StatusOK, resp)

	}
}

//
//func onAPIPostAppeal() gin.HandlerFunc {
//	type req struct {
//		Email      string `json:"email"`
//		AppealText string `json:"appeal_text"`
//	}
//	return func(c *gin.Context) {
//		var app req
//		if err := c.BindJSON(&app); err != nil {
//			log.Errorf("Received malformed appeal apiBanRequest: %v", err)
//			responseErr(c, http.StatusBadRequest, nil)
//			return
//		}
//		responseOK(c, http.StatusOK, gin.H{})
//	}
//}
//
//func onAPIPostReport() gin.HandlerFunc {
//	return func(c *gin.Context) {
//		responseErr(c, http.StatusInternalServerError, gin.H{})
//	}
//}

func onAPIGetServers() gin.HandlerFunc {
	return func(c *gin.Context) {
		servers, err := getServers()
		if err != nil {
			log.Errorf("Failed to fetch servers: %s", err)
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		responseOK(c, http.StatusOK, servers)
	}
}

func queryFilterFromContext(c *gin.Context) (*queryFilter, error) {
	var qf queryFilter
	if err := c.BindUri(&qf); err != nil {
		return nil, err
	}
	return &qf, nil
}

func onAPIGetPlayers() gin.HandlerFunc {
	return func(c *gin.Context) {
		qf, err := queryFilterFromContext(c)
		if err != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		people, err2 := getPeople(qf)
		if err2 != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		responseOK(c, http.StatusOK, people)
	}
}

func onAPICurrentProfile() gin.HandlerFunc {
	type resp struct {
		Player  *model.Person         `json:"player"`
		Friends []extra.PlayerSummary `json:"friends"`
	}
	return func(c *gin.Context) {
		p := currentPerson(c)
		if !p.SteamID.Valid() {
			responseErr(c, http.StatusForbidden, nil)
			return
		}
		friendIDs, err := fetchFriends(p.SteamID)
		if err != nil {
			responseErr(c, http.StatusServiceUnavailable, "Could not fetch friends")
			return
		}
		friends, err := fetchSummaries(friendIDs)
		if err != nil {
			responseErr(c, http.StatusServiceUnavailable, "Could not fetch summaries")
			return
		}
		var response resp
		response.Player = p
		response.Friends = friends
		responseOK(c, http.StatusOK, response)
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
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		cx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		sid, err := steamid.StringToSID64(r.Query)
		if err != nil {
			sid, err = steamid.ResolveSID64(cx, r.Query)
			if err != nil {
				responseErr(c, http.StatusNotFound, nil)
				return
			}
		}
		person, err := GetOrCreatePersonBySteamID(sid)
		if err != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		sum, err := extra.PlayerSummaries(cx, []steamid.SID64{sid})
		if err != nil || len(sum) != 1 {
			log.Errorf("Failed to get player summary: %v", err)
			responseErr(c, http.StatusInternalServerError, "Could not fetch summary")
			return
		}
		person.PlayerSummary = &sum[0]
		friendIDs, err := fetchFriends(person.SteamID)
		if err != nil {
			responseErr(c, http.StatusServiceUnavailable, "Could not fetch friends")
			return
		}
		friends, err := fetchSummaries(friendIDs)
		if err != nil {
			responseErr(c, http.StatusServiceUnavailable, "Could not fetch summaries")
			return
		}
		var response resp
		response.Player = person
		response.Friends = friends
		responseOK(c, http.StatusOK, response)
	}
}

func onAPIGetFilteredWords() gin.HandlerFunc {
	type resp struct {
		Count int      `json:"count"`
		Words []string `json:"words"`
	}
	return func(c *gin.Context) {
		words, err := getFilteredWords()
		if err != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		responseOK(c, http.StatusOK, resp{
			Count: len(words),
			Words: words,
		})
	}
}

func onAPIGetStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		stats, err := getStats()
		if err != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		serverStateMu.RLock()
		defer serverStateMu.RUnlock()
		for _, server := range serverStates {
			if server.Alive {
				stats.ServersAlive++
			}
		}
		responseOK(c, http.StatusOK, stats)
	}
}

func loadBanMeta(b *model.BannedPerson) {

}

func onAPIGetBanByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		banIDStr := c.Param("ban_id")
		if banIDStr == "" {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		sid, err := strconv.ParseUint(banIDStr, 10, 64)
		if err != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}

		ban, err := getBanByBanID(sid, false)
		if err != nil {
			responseErr(c, http.StatusNotFound, nil)
			log.Errorf("Failed to fetch bans")
			return
		}
		loadBanMeta(ban)
		responseOK(c, http.StatusOK, ban)
	}
}

func onAPIGetBans() gin.HandlerFunc {
	return func(c *gin.Context) {
		o := newQueryFilter("")
		bans, err := getBans(o)
		if err != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			log.Errorf("Failed to fetch bans")
			return
		}
		responseOK(c, http.StatusOK, bans)
	}
}

// LogPayload is the container for log/message payloads
type LogPayload struct {
	ServerName string `json:"server_name"`
	Message    string `json:"message"`
}

func onPostLogAdd() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LogPayload
		if err := c.BindJSON(&req); err != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		logRawQueue <- req
		responseOK(c, http.StatusCreated, nil)
	}
}
