package app

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/external"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/steam"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/fp"
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

type apiResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func responseErr(c *gin.Context, status int, data any) {
	c.JSON(status, apiResponse{
		Status: false,
		Data:   data,
	})
}

func responseOK(c *gin.Context, status int, data any) {
	c.JSON(status, apiResponse{
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

func (w *web) onPostPingMod(db store.Store) gin.HandlerFunc {
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
		err := Find(db, model.Target(req.SteamID.String()), "", &pi)
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
		e := respOk(nil, "New User Report")
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
			select {
			case w.botSendMessageChan <- discordPayload{channelId: chanId, message: e}:
			default:
				log.Warnf("Cannot send discord payload, channel full")
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

func (w *web) onAPIPostBanCreate(db store.Store) gin.HandlerFunc {
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
			bn := banNetworkOpts{
				banOpts: banOpts{target: model.Target(r.SteamID.String()),
					author:   model.Target(currentPerson(c).SteamID.String()),
					duration: model.Duration(r.Duration),
					reason:   r.ReasonText,
					origin:   model.Web,
				},
				cidr: r.Network,
			}
			var b model.BanNet
			if bErr := BanNetwork(db, bn, &b); bErr != nil {
				if errors.Is(bErr, store.ErrDuplicate) {
					responseErr(c, http.StatusConflict, "Duplicate ban")
					return
				}
				responseErr(c, http.StatusBadRequest, "Failed to perform ban")
				return
			}
			responseOK(c, http.StatusCreated, banNet)
		} else {
			bo := banOpts{
				target:   model.Target(r.SteamID.String()),
				author:   model.Target(currentPerson(c).SteamID.String()),
				duration: model.Duration(r.Duration),
				reason:   r.ReasonText,
				origin:   model.Web,
			}
			var b model.Ban
			if bErr := Ban(db, bo, &b, w.botSendMessageChan); bErr != nil {
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
//		Vars     map[string]any `json:"vars"`
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
		servers, err := db.GetServers(c, true)
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
	type playerInfo struct {
		SteamID       steamid.SID64 `json:"steam_id"`
		Name          string        `json:"name"`
		UserId        int           `json:"user_id"`
		ConnectedTime int64         `json:"connected_secs"`
	}
	type serverInfo struct {
		ServerID int64 `db:"server_id" json:"server_id"`
		// ServerName is a short reference name for the server eg: us-1
		ServerName     string `json:"server_name"`
		ServerNameLong string `json:"server_name_long"`
		Address        string `json:"address"`
		// Port is the port of the server
		Port              int          `json:"port"`
		PasswordProtected bool         `json:"password_protected"`
		VAC               bool         `json:"vac"`
		Region            string       `json:"region"`
		CC                string       `json:"cc"`
		Latitude          float64      `json:"latitude"`
		Longitude         float64      `json:"longitude"`
		CurrentMap        string       `json:"current_map"`
		Tags              []string     `json:"tags"`
		DefaultMap        string       `json:"default_map"`
		ReservedSlots     int          `json:"reserved_slots"`
		CreatedOn         time.Time    `json:"created_on"`
		UpdatedOn         time.Time    `json:"updated_on"`
		PlayersMax        int          `json:"players_max"`
		Players           []playerInfo `json:"players"`
	}
	return func(c *gin.Context) {
		servers, err := db.GetServers(c, true)
		if err != nil {
			log.Errorf("Failed to fetch servers: %s", err)
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		currentState := ServerState()
		var si []serverInfo
		for _, srv := range servers {
			v := serverInfo{
				ServerID:          srv.ServerID,
				ServerName:        srv.ServerName,
				ServerNameLong:    srv.ServerNameLong,
				Address:           srv.Address,
				Port:              srv.Port,
				PasswordProtected: srv.Password != "",
				Region:            srv.Region,
				CC:                srv.CC,
				Latitude:          srv.Location.Latitude,
				Longitude:         srv.Location.Longitude,
				CurrentMap:        "",
				DefaultMap:        srv.DefaultMap,
				ReservedSlots:     srv.ReservedSlots,
				CreatedOn:         srv.CreatedOn,
				UpdatedOn:         srv.UpdatedOn,
				Players:           nil,
			}
			state, stateFound := currentState[v.ServerName]
			if stateFound {
				v.VAC = state.A2S.VAC
				v.CurrentMap = state.Status.Map
				v.PlayersMax = state.Status.PlayersMax
				v.Tags = state.Status.Tags
				for _, pl := range state.Status.Players {
					v.Players = append(v.Players, playerInfo{
						SteamID:       pl.SID,
						Name:          pl.Name,
						UserId:        pl.UserID,
						ConnectedTime: int64(pl.ConnectedTime.Seconds()),
					})
				}
			}

			si = append(si, v)
		}

		responseOK(c, http.StatusOK, si)
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
		people, err2 := db.GetPeople(c, qf)
		if err2 != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		responseOK(c, http.StatusOK, people)
	}
}

func (w *web) onAPIGetResolveProfile(db store.PersonStore) gin.HandlerFunc {
	type queryParam struct {
		Query string `json:"query"`
	}
	return func(c *gin.Context) {
		var q queryParam
		if errBind := c.BindJSON(&q); errBind != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		id, errResolve := steamid.ResolveSID64(c, q.Query)
		if errResolve != nil {
			responseErr(c, http.StatusOK, nil)
			return
		}
		var p model.Person
		if errPerson := getOrCreateProfileBySteamID(c, db, id, "", &p); errPerson != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		responseOK(c, http.StatusOK, p)
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
		var fWords []string
		for _, f := range words {
			fWords = append(fWords, f.Pattern.String())
		}
		responseOK(c, http.StatusOK, resp{Count: len(fWords), Words: fWords})
	}
}

func (w *web) onAPIGetCompHist() gin.HandlerFunc {
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
		if errFetch := external.FetchCompHist(cx, sid, &hist); errFetch != nil {
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
		banId, err := strconv.ParseUint(banIDStr, 10, 64)
		if err != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		cx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		ban := model.NewBannedPerson()
		if errB := db.GetBanByBanID(cx, banId, false, &ban); errB != nil {
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

func (w *web) onAPIEvents(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var q model.LogQueryOpts
		if errBind := c.BindJSON(&q); errBind != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		events, err := db.FindLogEvents(c, q)
		if err != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		responseOK(c, http.StatusOK, events)
	}
}

func (w *web) onAPIPostAppeal(db store.Store) gin.HandlerFunc {
	type newAppeal struct {
		BanId  int    `json:"ban_id"`
		Reason string `json:"reason"`
	}
	return func(c *gin.Context) {
		var appeal newAppeal
		if err := c.BindJSON(&appeal); err != nil {

		}
	}
}

func (w *web) onAPIGetAppeal(db store.Store) gin.HandlerFunc {
	type Appeal struct {
		Person model.BannedPerson `json:"person"`
		Appeal model.Appeal       `json:"appeal"`
	}
	return func(c *gin.Context) {
		banIdStr := c.Param("ban_id")
		if banIdStr == "" {
			responseErr(c, http.StatusNotFound, nil)
			return
		}
		banId, errBanId := strconv.ParseUint(banIdStr, 10, 64)
		if errBanId != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		var appeal model.Appeal
		if errAppeal := db.GetAppeal(c, banId, &appeal); errAppeal != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		responseOK(c, http.StatusOK, appeal)
	}
}

func (w *web) onAPIPostReportCreate(db store.Store) gin.HandlerFunc {
	type reportMedia struct {
		FileName string `json:"file_name"`
		MimeType string `json:"mime_type"`
		Content  []byte `json:"content"`
		Size     int64  `json:"size"`
	}
	type createReport struct {
		SteamId     string        `json:"steam_id"`
		Title       string        `json:"title"`
		Description string        `json:"description"`
		Media       []reportMedia `json:"media"`
	}
	return func(c *gin.Context) {
		currentUser := currentPerson(c)
		var cr createReport
		if errBind := c.BindJSON(&cr); errBind != nil {
			responseErr(c, http.StatusBadRequest, nil)
			log.Errorf("Failed to bind report: %v", errBind)
			return
		}
		sid, errSid := steamid.ResolveSID64(c, cr.SteamId)
		if errSid != nil {
			responseErr(c, http.StatusBadRequest, nil)
			log.Errorf("Invaid steam_id: %v", errSid)
			return
		}
		var p model.Person
		if errCreatePerson := getOrCreateProfileBySteamID(c, db, sid, "", &p); errCreatePerson != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			log.Errorf("Could not load player profile: %v", errCreatePerson)
			return
		}
		// TODO encapsulate all operations in single tx
		report := model.NewReport()
		report.AuthorId = currentUser.SteamID
		report.ReportStatus = model.Opened
		report.Title = cr.Title
		report.Description = cr.Description
		report.ReportedId = sid
		if errReportSave := db.SaveReport(c, &report); errReportSave != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			log.Errorf("Failed to save report: %v", errReportSave)
			return
		}
		for _, media := range cr.Media {
			rm := model.NewReportMedia(report.ReportId)
			rm.AuthorId = currentUser.SteamID
			rm.Contents = media.Content
			rm.MimeType = media.MimeType
			rm.Size = media.Size
			if errSaveMedia := db.SaveReportMedia(c, report.ReportId, &rm); errSaveMedia != nil {
				responseErr(c, http.StatusInternalServerError, nil)
				log.Errorf("Failed to save report media: %v", errSaveMedia)
				return
			}
		}
		responseOK(c, http.StatusCreated, report)
	}
}

func getSID64Param(c *gin.Context, key string) (steamid.SID64, error) {
	i, err := getInt64Param(c, key)
	if err != nil {
		return 0, err
	}
	sid := steamid.SID64(i)
	if !sid.Valid() {
		return 0, consts.ErrInvalidSID
	}
	return sid, nil
}

func getInt64Param(c *gin.Context, key string) (int64, error) {
	valueStr := c.Param(key)
	if valueStr == "" {
		return 0, errors.Errorf("Failed to get %s", key)
	}
	value, valueErr := strconv.ParseInt(valueStr, 10, 64)
	if valueErr != nil {
		return 0, errors.Errorf("Failed to parse %s: %v", key, valueErr)
	}
	if value <= 0 {
		return 0, errors.Errorf("Invalid %s: %v", key, valueErr)
	}
	return value, nil
}

func (w *web) onAPIGetReportMedia(db store.ReportStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		mediaId, errParam := getInt64Param(c, "report_media_id")
		if errParam != nil {
			responseErr(c, http.StatusNotFound, nil)
			return
		}
		var m model.ReportMedia
		if errMedia := db.GetReportMediaById(c, int(mediaId), &m); errMedia != nil {
			responseErr(c, http.StatusNotFound, nil)
			return
		}
		c.Data(http.StatusOK, m.MimeType, m.Contents)
	}
}

func (w *web) onAPIPostReportMessage(db store.ReportStore) gin.HandlerFunc {
	type req struct {
		Message string `json:"message"`
	}
	return func(c *gin.Context) {
		reportId, errParam := getInt64Param(c, "report_id")
		if errParam != nil {
			responseErr(c, http.StatusNotFound, nil)
			return
		}
		var r req
		if errBind := c.BindJSON(&r); errBind != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		if r.Message == "" {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		var report model.Report
		if errReport := db.GetReport(c, int(reportId), &report); errReport != nil {
			if store.Err(errReport) == store.ErrNoResult {
				responseErr(c, http.StatusNotFound, nil)
				return
			}
			responseErr(c, http.StatusBadRequest, nil)
			log.Errorf("Failed to load report: %v", errReport)
			return
		}
		p := currentPerson(c)
		msg := model.NewReportMessage(int(reportId), p.SteamID, r.Message)
		if errSave := db.SaveReportMessage(c, int(reportId), &msg); errSave != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			log.Errorf("Failed to save report message: %v", errSave)
			return
		}
		responseOK(c, http.StatusCreated, msg)
	}
}

type AuthorReportMessage struct {
	Author  model.Person        `json:"author"`
	Message model.ReportMessage `json:"message"`
}

func (w *web) onAPIGetReportMessages(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		reportId, errParam := getInt64Param(c, "report_id")
		if errParam != nil {
			responseErr(c, http.StatusNotFound, nil)
			return
		}
		msgs, errMsgs := db.GetReportMessages(c, int(reportId))
		if errMsgs != nil {
			responseErr(c, http.StatusNotFound, nil)
			return
		}
		var ids steamid.Collection
		for _, msg := range msgs {
			ids = append(ids, msg.AuthorId)
		}
		authors, authorsErr := db.GetPeopleBySteamID(c, ids)
		if authorsErr != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		authorsMap := authors.AsMap()
		var authorMsgs []AuthorReportMessage
		for _, m := range msgs {
			authorMsgs = append(authorMsgs, AuthorReportMessage{
				Author:  authorsMap[m.AuthorId],
				Message: m,
			})
		}
		responseOK(c, http.StatusOK, authorMsgs)
	}
}

type reportWithAuthor struct {
	Author model.Person `json:"author"`
	Report model.Report `json:"report"`
}

func (w *web) onAPIGetReports(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var opts store.AuthorQueryFilter
		if errBind := c.BindJSON(&opts); errBind != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		var f store.AuthorQueryFilter
		if opts.Limit > 0 && opts.Limit <= 100 {
			f.Limit = opts.Limit
		} else {
			f.Limit = 25
		}
		var userReports []reportWithAuthor
		reports, errReports := db.GetReports(c, f)
		if errReports != nil {
			if store.Err(errReports) == store.ErrNoResult {
				responseOK(c, http.StatusNoContent, nil)
				return
			}
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		var authorIds steamid.Collection
		for _, report := range reports {
			authorIds = append(authorIds, report.ReportedId)
		}
		authors, errAuthors := db.GetPeopleBySteamID(c, fp.Uniq[steamid.SID64](authorIds))
		if errAuthors != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		am := authors.AsMap()
		for _, r := range reports {
			userReports = append(userReports, reportWithAuthor{
				Author: am[r.ReportedId],
				Report: r,
			})
		}

		responseOK(c, http.StatusOK, userReports)
	}
}

func (w *web) onAPIGetReport(db store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		reportId, errParam := getInt64Param(c, "report_id")
		if errParam != nil {
			responseErr(c, http.StatusNotFound, nil)
			return
		}
		var report reportWithAuthor
		if errReport := db.GetReport(c, int(reportId), &report.Report); errReport != nil {
			if store.Err(errReport) == store.ErrNoResult {
				responseErr(c, http.StatusNotFound, nil)
				return
			}
			responseErr(c, http.StatusBadRequest, nil)
			log.Errorf("Failed to load report: %v", errReport)
			return
		}
		if errAuthor := db.GetOrCreatePersonBySteamID(c, report.Report.AuthorId, &report.Author); errAuthor != nil {
			if store.Err(errAuthor) == store.ErrNoResult {
				responseErr(c, http.StatusNotFound, nil)
				return
			}
			responseErr(c, http.StatusBadRequest, nil)
			log.Errorf("Failed to load report author: %v", errAuthor)
			return
		}
		responseOK(c, http.StatusOK, report)
	}
}

func (w *web) onAPILogsQuery(db store.StatStore) gin.HandlerFunc {
	type req struct {
		SteamID string `json:"steam_id"`
		Limit   int    `json:"limit"`
	}
	type logMsg struct {
		CreatedOn time.Time `json:"created_on"`
		Message   string    `json:"message"`
	}
	return func(c *gin.Context) {
		var r req
		if errBind := c.BindJSON(&r); errBind != nil {
			if errBind != nil {
				responseErr(c, http.StatusBadRequest, nil)
				return
			}
		}
		sid, errSid := steamid.StringToSID64(r.SteamID)
		if errSid != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		hist, errHist := db.GetChatHistory(c, sid, 100)
		if errHist != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		var logs []logMsg
		for _, h := range hist {
			logs = append(logs, logMsg{h.CreatedOn, h.Msg})
		}
		responseOK(c, http.StatusOK, logs)
	}
}
