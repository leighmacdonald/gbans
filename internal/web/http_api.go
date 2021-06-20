package web

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/action"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/steam"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/extra"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strconv"
	"time"
)

type APIResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func responseErr(c *gin.Context, status int, data interface{}) {
	c.JSON(status, APIResponse{
		Status: false,
		Data:   data,
	})
}

func responseOK(c *gin.Context, status int, data interface{}) {
	c.JSON(status, APIResponse{
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
		act := action.NewFind(action.Web, req.SteamID.String())
		res := <-act.Enqueue().Done()
		pi, ok := res.Value.(model.PlayerInfo)
		if !ok {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		name := req.SteamID.String()
		if pi.InGame {
			name += fmt.Sprintf(" (%s)", pi.Player.Name)
		}
		for _, chanId := range config.Discord.ModChannels {
			m := fmt.Sprintf("<@&%s> [%s] (%s): %s", config.Discord.ModRoleID, req.ServerName, name, req.Reason)
			err := discord.Send(chanId, m, false)
			if err != nil {
				responseErr(c, http.StatusInternalServerError, nil)
				return
			}
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
		//		duration, err := config.ParseDuration(r.Duration)
		//		if err != nil {
		//			responseErr(c, http.StatusNotAcceptable, `Invalid duration. Examples: "300m", "1.5h" or "2h45m".
		//Valid time units are "s", "ws", "h".`)
		//			return
		//		}
		var (
			ban    *model.Ban
			banNet *model.BanNet
			e      error
		)
		if r.Network != "" {
			_, _, e = net.ParseCIDR(r.Network)
			if e != nil {
				responseErr(c, http.StatusBadRequest, "Invalid network cidr definition")
				return
			}
		}
		if !r.SteamID.Valid() {
			responseErr(c, http.StatusBadRequest, "Invalid steamid")
			return
		}
		var act action.Action
		if r.Network != "" {
			act = action.NewBanNet(action.Web, r.SteamID.String(), currentPerson(c).SteamID.String(), r.ReasonText, r.Duration, r.Network)
		} else {
			act = action.NewBan(action.Web, r.SteamID.String(), currentPerson(c).SteamID.String(), r.ReasonText, r.Duration)
		}
		res := <-act.Enqueue().Done()
		if res.Err != nil {
			if errors.Is(res.Err, store.ErrDuplicate) {
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		srv, err := store.GetServerByName(ctx, req.ServerName)
		if err != nil {
			responseErr(c, http.StatusNotFound, nil)
			return
		}
		if srv.Password != req.Key {
			responseErr(c, http.StatusForbidden, nil)
			log.Warnf("Invalid server key used: %s", req.ServerName)
			return
		}
		srv.Token = golib.RandomString(40)
		srv.TokenCreatedOn = config.Now()
		if err2 := store.SaveServer(ctx, &srv); err2 != nil {
			log.Errorf("Failed to updated server token: %v", err2)
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		responseOK(c, http.StatusOK, authResp{
			Status: true,
			Token:  srv.Token,
		})
	}
}

type CheckRequest struct {
	ClientID int    `json:"client_id"`
	SteamID  string `json:"steam_id"`
	IP       net.IP `json:"ip"`
}

func onPostServerCheck() gin.HandlerFunc {
	type checkResponse struct {
		ClientID int           `json:"client_id"`
		SteamID  string        `json:"steam_id"`
		BanType  model.BanType `json:"ban_type"`
		Msg      string        `json:"msg"`
	}
	return func(c *gin.Context) {
		var req CheckRequest
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		// Check IP first
		banNet, err := store.GetBanNet(ctx, req.IP)
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
		p := action.NewGetOrCreatePersonByID(steamID.String(), req.IP.String())
		p.EnqueueIgnore()
		ban, errB := store.GetBanBySteamID(ctx, steamID, false)
		if errB != nil {
			if errB == store.ErrNoResult {
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		servers, err := store.GetServers(ctx)
		if err != nil {
			log.Errorf("Failed to fetch servers: %s", err)
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		responseOK(c, http.StatusOK, servers)
	}
}

func queryFilterFromContext(c *gin.Context) (*store.QueryFilter, error) {
	var qf store.QueryFilter
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		people, err2 := store.GetPeople(ctx, qf)
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
		friendIDs, err := steam.FetchFriends(p.SteamID)
		if err != nil {
			responseErr(c, http.StatusServiceUnavailable, "Could not fetch friends")
			return
		}
		friends, err := steam.FetchSummaries(friendIDs)
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
		person, err2 := store.GetOrCreatePersonBySteamID(cx, sid)
		if err2 != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		sum, err3 := extra.PlayerSummaries(cx, []steamid.SID64{sid})
		if err3 != nil || len(sum) != 1 {
			log.Errorf("Failed to get player summary: %v", err3)
			responseErr(c, http.StatusInternalServerError, "Could not fetch summary")
			return
		}
		person.PlayerSummary = &sum[0]
		friendIDs, err4 := steam.FetchFriends(person.SteamID)
		if err4 != nil {
			responseErr(c, http.StatusServiceUnavailable, "Could not fetch friends")
			return
		}
		friends, err5 := steam.FetchSummaries(friendIDs)
		if err5 != nil {
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
		cx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		words, err := store.GetFilters(cx)
		if err != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		var w []string
		for _, f := range words {
			w = append(w, f.Word.String())
		}
		responseOK(c, http.StatusOK, resp{Count: len(words), Words: w})
	}
}

func onAPIGetStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		cx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		stats, err := store.GetStats(cx)
		if err != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		stats.ServersAlive = state.ServersAlive()
		responseOK(c, http.StatusOK, stats)
	}
}

func loadBanMeta(_ *model.BannedPerson) {

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
		cx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		ban, err := store.GetBanByBanID(cx, sid, false)
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
		o := store.NewQueryFilter("")
		cx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		bans, err := store.GetBans(cx, o)
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

func onPostLogAdd(logMsgChan chan LogPayload) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req []LogPayload
		if err := c.BindJSON(&req); err != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		responseOK(c, http.StatusCreated, nil)
		for _, r := range req {
			logMsgChan <- r
		}
	}
}

func onAPIPostServer() gin.HandlerFunc {
	return func(c *gin.Context) {
		responseOK(c, http.StatusOK, gin.H{})
	}
}
