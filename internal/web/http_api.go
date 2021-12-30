package web

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/action"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/external"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/steam"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strconv"
	"strings"
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

type demoPostRequest struct {
	ServerName string `form:"server_name"`
}

func (w *web) onPostDemo(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r demoPostRequest
		if errR := c.Bind(&r); errR != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		f, hdr, err := c.Request.FormFile("file")
		if err != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		var server model.Server
		if errS := db.GetServerByName(c, r.ServerName, &server); errS != nil {
			responseErr(c, http.StatusNotFound, nil)
			return
		}
		var d []byte
		_, errRead := f.Read(d)
		if errRead != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		demo, errDF := model.NewDemoFile(server.ServerID, hdr.Filename, d)
		if errDF != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		if errSave := db.SaveDemo(c, &demo); errSave != nil {
			log.Errorf("Failed to save demo to store: %v", errSave)
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		responseOK(c, http.StatusCreated, demo)
	}
}

func (w *web) onPostPingMod(bot discord.ChatBot) gin.HandlerFunc {
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
		var pi model.PlayerInfo
		err := w.executor.Find(req.SteamID.String(), "", &pi)
		if err != nil {
			log.Error("Failed to find player on /mod call")
		}
		//name := req.SteamID.String()
		//if pi.InGame {
		//	name = fmt.Sprintf("%s (%s)", name, pi.Player.Name)
		//}
		var roleStrings []string
		for _, i := range config.Discord.ModRoleIDs {
			roleStrings = append(roleStrings, fmt.Sprintf("<@&%s>", i))
		}
		e := discord.RespOk(nil, "New User Report")
		e.Description = fmt.Sprintf("%s | %s", req.Reason, strings.Join(roleStrings, " "))
		if pi.Player.Name != "" {
			e.Fields = append(e.Fields, &discordgo.MessageEmbedField{
				Name:   "Reporter",
				Value:  pi.Player.Name,
				Inline: true,
			})
		}
		if req.SteamID.String() != "" {
			e.Fields = append(e.Fields, &discordgo.MessageEmbedField{
				Name:   "ReporterSID",
				Value:  req.SteamID.String(),
				Inline: true,
			})
		}
		if req.ServerName != "" {
			e.Fields = append(e.Fields, &discordgo.MessageEmbedField{
				Name:   "Server",
				Value:  req.ServerName,
				Inline: true,
			})
		}
		for _, chanId := range config.Discord.ModChannels {
			if errSend := bot.SendEmbed(chanId, e); errSend != nil {
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

func (w *web) onAPIPostBanCreate() gin.HandlerFunc {
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
		if r.Network != "" {
			var b model.BanNet
			if bErr := w.executor.BanNetwork(action.NewBanNet(model.Web, r.SteamID.String(),
				currentPerson(c).SteamID.String(), r.ReasonText, r.Duration, r.Network), &b); bErr != nil {
				if errors.Is(bErr, store.ErrDuplicate) {
					responseErr(c, http.StatusConflict, "Duplicate ban")
					return
				}
				responseErr(c, http.StatusBadRequest, "Failed to perform ban")
				return
			}
			responseOK(c, http.StatusCreated, banNet)
		} else {
			var b model.Ban
			if bErr := w.executor.Ban(action.NewBan(model.Web, r.SteamID.String(), currentPerson(c).SteamID.String(),
				r.ReasonText, r.Duration), &b); bErr != nil {
				if errors.Is(bErr, store.ErrDuplicate) {
					responseErr(c, http.StatusConflict, "Duplicate ban")
					return
				}
				responseErr(c, http.StatusBadRequest, "Failed to perform ban")
				return
			}
			responseOK(c, http.StatusCreated, ban)
		}
	}
}

func (w *web) onSAPIPostServerAuth(db store.Store) gin.HandlerFunc {
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
		var srv model.Server
		err := db.GetServerByName(ctx, req.ServerName, &srv)
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
		if err2 := db.SaveServer(ctx, &srv); err2 != nil {
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

func (w *web) onPostServerCheck(db store.Store) gin.HandlerFunc {
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		// Check IP first
		banNet, err := db.GetBanNet(ctx, req.IP)
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
			log.WithFields(log.Fields{"type": "cidr", "reason": banNet[0].Reason}).Infof("Player dropped")
			return
		}
		// Check SteamID
		steamID, errResolve := steamid.ResolveSID64(context.Background(), req.SteamID)
		if errResolve != nil || !steamID.Valid() {
			resp.Msg = "Invalid steam id"
			responseErr(c, http.StatusBadRequest, resp)
			return
		}
		var asnRecord ip2location.ASNRecord
		errASN := db.GetASNRecordByIP(ctx, req.IP, &asnRecord)
		if errASN == nil {
			var asnBan model.BanASN
			if errASNBan := db.GetBanASN(ctx, int64(asnRecord.ASNum), &asnBan); errASNBan != nil {
				if !errors.Is(errASNBan, store.ErrNoResult) {
					log.Errorf("Failed to fetch asn ban: %v", errASNBan)
				}
			} else {
				resp.BanType = model.Banned
				resp.Msg = asnBan.Reason
				responseOK(c, http.StatusOK, resp)
				log.WithFields(log.Fields{"type": "asn", "reason": asnBan.Reason}).Infof("Player dropped")
				return
			}
		}
		ban := model.NewBannedPerson()
		if errB := db.GetBanBySteamID(ctx, steamID, false, &ban); errB != nil {
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

//
//func (w *web) onAPIGetAnsibleHosts(db store.Store) gin.HandlerFunc {
//	type groupConfig struct {
//		Hosts    []string               `json:"hosts"`
//		Vars     map[string]interface{} `json:"vars"`
//		Children []string               `json:"children"`
//	}
//	type ansibleStaticConfig map[string]groupConfig
//
//	return func(c *gin.Context) {
//		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
//		defer cancel()
//		servers, err := db.GetServers(ctx, true)
//		if err != nil {
//			log.Errorf("Failed to fetch servers: %s", err)
//			responseErr(c, http.StatusInternalServerError, nil)
//			return
//		}
//		var hosts []string
//		for _, server := range servers {
//			hosts = append(hosts, server.Address)
//		}
//		hostCfg := ansibleStaticConfig{"all": groupConfig{
//			Hosts:    hosts,
//			Vars:     nil,
//			Children: nil,
//		}}
//		responseOK(c, http.StatusOK, hostCfg)
//	}
//}

// https://prometheus.io/docs/prometheus/latest/configuration/configuration/#http_sd_config
func (w *web) onAPIGetPrometheusHosts(db store.Store) gin.HandlerFunc {
	type promStaticConfig struct {
		Targets []string          `json:"targets"`
		Labels  map[string]string `json:"labels"`
	}
	type portMap struct {
		Type string
		Port int
	}
	return func(c *gin.Context) {
		var staticConfigs []promStaticConfig
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		servers, err := db.GetServers(ctx, true)
		if err != nil {
			log.Errorf("Failed to fetch servers: %s", err)
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		for _, nodePortConfig := range []portMap{{"node", 9100}} {
			ps := promStaticConfig{Targets: nil, Labels: map[string]string{}}
			ps.Labels["__meta_prometheus_job"] = nodePortConfig.Type
			for _, server := range servers {
				host := fmt.Sprintf("%s:%d", server.Address, nodePortConfig.Port)
				found := false
				for _, h := range ps.Targets {
					if h == host {
						found = true
						break
					}
				}
				if !found {
					ps.Targets = append(ps.Targets, host)
				}
			}
			staticConfigs = append(staticConfigs, ps)
		}
		// Don't wrap in our custom response format
		c.JSON(200, staticConfigs)
	}
}

func (w *web) onAPIGetServers(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		servers, err := db.GetServers(ctx, true)
		if err != nil {
			log.Errorf("Failed to fetch servers: %s", err)
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		responseOK(c, http.StatusOK, servers)
	}
}

func (w *web) queryFilterFromContext(c *gin.Context) (*store.QueryFilter, error) {
	var qf store.QueryFilter
	if err := c.BindUri(&qf); err != nil {
		return nil, err
	}
	return &qf, nil
}

func (w *web) onAPIGetPlayers(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		qf, err := w.queryFilterFromContext(c)
		if err != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		people, err2 := db.GetPeople(ctx, qf)
		if err2 != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		responseOK(c, http.StatusOK, people)
	}
}

func (w *web) onAPICurrentProfile() gin.HandlerFunc {
	type resp struct {
		Player  *model.Person            `json:"player"`
		Friends []steamweb.PlayerSummary `json:"friends"`
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
		response.Player = &p
		response.Friends = friends
		responseOK(c, http.StatusOK, response)
	}
}

func (w *web) onAPIProfile(db store.Store) gin.HandlerFunc {
	type req struct {
		Query string `form:"query"`
	}
	type resp struct {
		Player  *model.Person            `json:"player"`
		Friends []steamweb.PlayerSummary `json:"friends"`
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
		person := model.NewPerson(sid)
		if err2 := db.GetOrCreatePersonBySteamID(cx, sid, &person); err2 != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		sum, err3 := steamweb.PlayerSummaries(steamid.Collection{sid})
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
		response.Player = &person
		response.Friends = friends
		responseOK(c, http.StatusOK, response)
	}
}

func (w *web) onAPIGetFilteredWords(db store.Store) gin.HandlerFunc {
	type resp struct {
		Count int      `json:"count"`
		Words []string `json:"words"`
	}
	return func(c *gin.Context) {
		cx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		words, err := db.GetFilters(cx)
		if err != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		var w []string
		for _, f := range words {
			w = append(w, f.Pattern.String())
		}
		responseOK(c, http.StatusOK, resp{Count: len(words), Words: w})
	}
}

func (w *web) onAPIGetCompHist(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		sidStr := c.DefaultQuery("sid", "")
		if sidStr == "" {
			responseErr(c, http.StatusBadRequest, "missing sid")
			return
		}
		sid, err := steamid.StringToSID64(sidStr)
		if err != nil || !sid.Valid() {
			responseErr(c, http.StatusBadRequest, "invalid sid")
			return
		}
		cx, cancel := context.WithTimeout(c, time.Second*10)
		defer cancel()
		var hist external.CompHist
		if err := external.FetchCompHist(cx, sid, &hist); err != nil {
			responseErr(c, http.StatusInternalServerError, "query failed")
			return
		}
		responseOK(c, http.StatusOK, hist)
	}
}

func (w *web) onAPIGetStats(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		cx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		var stats model.Stats
		if err := db.GetStats(cx, &stats); err != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		stats.ServersAlive = 1
		responseOK(c, http.StatusOK, stats)
	}
}

func loadBanMeta(_ *model.BannedPerson) {

}

func (w *web) onAPIGetBanByID(db store.Store) gin.HandlerFunc {
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
		ban := model.NewBannedPerson()
		if errB := db.GetBanByBanID(cx, sid, false, &ban); errB != nil {
			responseErr(c, http.StatusNotFound, nil)
			log.Errorf("Failed to fetch bans: %v", errB)
			return
		}
		loadBanMeta(&ban)
		responseOK(c, http.StatusOK, ban)
	}
}

func (w *web) onAPIGetBans(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		o := store.NewQueryFilter("")
		cx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		bans, err := db.GetBans(cx, o)
		if err != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			log.Errorf("Failed to fetch bans")
			return
		}
		responseOK(c, http.StatusOK, bans)
	}
}

func (w *web) onAPIPostServer() gin.HandlerFunc {
	return func(c *gin.Context) {
		responseOK(c, http.StatusOK, gin.H{})
	}
}

func (w *web) onSup(p Payload) error {

	return nil
}
