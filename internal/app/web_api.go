package app

import (
	"context"
	"encoding/base64"
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
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/srcdsup/srcdsup"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

// apiResponse represents the common high level response of all api responses. All child data is
// returned by the Data field.
type apiResponse struct {
	// Status is a simple truthy status of the response. See response codes for more specific
	// error handling scenarios
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func responseErrUser(ctx *gin.Context, status int, data any, userMsg string, args ...any) {
	ctx.JSON(status, apiResponse{
		Status:  false,
		Message: fmt.Sprintf(userMsg, args...),
		Data:    data,
	})
}

func responseErr(ctx *gin.Context, status int, data any) {
	ctx.JSON(status, apiResponse{Status: false, Data: data})
}

func responseOKUser(ctx *gin.Context, status int, data any, userMsg string, args ...any) {
	ctx.JSON(status, apiResponse{
		Status:  true,
		Message: fmt.Sprintf(userMsg, args...),
		Data:    data,
	})
}

func responseOK(ctx *gin.Context, status int, data any) {
	ctx.JSON(status, apiResponse{Status: true, Data: data})
}

func resolveUrl(path string, args ...any) string {
	base := config.General.ExternalUrl
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return base + fmt.Sprintf(path, args...)
}

func (web *web) onPostLog(db store.Store, logFileC chan *LogFilePayload) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var upload srcdsup.ServerLogUpload
		if errBind := ctx.BindJSON(&upload); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		if upload.ServerName == "" || upload.Body == "" {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var server model.Server
		if errServer := db.GetServerByName(ctx, upload.ServerName, &server); errServer != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		rawLogs, errDecode := base64.StdEncoding.DecodeString(upload.Body)
		if errDecode != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		logLines := strings.Split(string(rawLogs), "\n")
		log.WithFields(log.Fields{"count": len(logLines)}).Debugf("Uploaded log file")
		responseOKUser(ctx, http.StatusCreated, nil, "Log uploaded")
		// Send the log to the logReader() for actual processing
		logFileC <- &LogFilePayload{
			Server: server,
			Lines:  logLines,
			Map:    upload.MapName,
		}
		log.Tracef("File upload complete")
	}
}

func (web *web) onPostDemo(database store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var upload srcdsup.ServerLogUpload
		if errBind := ctx.BindJSON(&upload); errBind != nil {
			log.Debugf("Failed to parse demo payload: %v", errBind)
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		if upload.ServerName == "" || upload.Body == "" || upload.MapName == "" {
			log.WithFields(log.Fields{
				"server_name": util.SanitizeLog(upload.ServerName),
				"map_name":    util.SanitizeLog(upload.MapName),
				"body_len":    len(upload.Body),
			}).Debug("Missing demo params")
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var server model.Server
		if errGetServer := database.GetServerByName(ctx, upload.ServerName, &server); errGetServer != nil {
			log.WithFields(log.Fields{"server": util.SanitizeLog(upload.ServerName)}).Errorf("Server not found")
			responseErrUser(ctx, http.StatusNotFound, nil, "Server not found: %v", upload.ServerName)
			return
		}
		rawDemo, errDecode := base64.StdEncoding.DecodeString(upload.Body)
		if errDecode != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		dateStr := config.Now().Format("2006-01-02_15-04-05")
		name := fmt.Sprintf("demo_%s_%s_%s.dem", server.ServerNameShort, dateStr, upload.MapName)
		outDir := path.Join(config.General.DemoRootPath, server.ServerNameShort, config.Now().Format("2006-01-02"))
		if errMkDir := os.MkdirAll(outDir, 0775); errMkDir != nil {
			log.Errorf("Failed to create demo dir: %v", errMkDir)
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		outPath := path.Join(outDir, name)
		if errWrite := os.WriteFile(outPath, rawDemo, 0775); errWrite != nil {
			log.Errorf("Failed to write demo: %v", errWrite)
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusCreated, gin.H{"path": outPath})
	}
}

func (web *web) onPostPingMod(database store.Store) gin.HandlerFunc {
	type pingReq struct {
		ServerName string        `json:"server_name"`
		Name       string        `json:"name"`
		SteamID    steamid.SID64 `json:"steam_id"`
		Reason     string        `json:"reason"`
		Client     int           `json:"client"`
	}
	return func(ctx *gin.Context) {
		var req pingReq
		if errBind := ctx.BindJSON(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var playerInfo model.PlayerInfo
		errFind := Find(ctx, database, model.Target(req.SteamID.String()), "", &playerInfo)
		if errFind != nil {
			log.Error("Failed to find player on /mod call")
		}
		//name := req.SteamID.String()
		//if playerInfo.InGame {
		//	name = fmt.Sprintf("%s (%s)", name, playerInfo.Player.Name)
		//}
		var roleStrings []string
		for _, roleID := range config.Discord.ModRoleIDs {
			roleStrings = append(roleStrings, fmt.Sprintf("<@&%s>", roleID))
		}
		embed := respOk(nil, "New User Report")
		embed.Description = fmt.Sprintf("%s | %s", req.Reason, strings.Join(roleStrings, " "))
		if playerInfo.Player.Name != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Reporter",
				Value:  playerInfo.Player.Name,
				Inline: true,
			})
		}
		if req.SteamID.String() != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "ReporterSID",
				Value:  req.SteamID.String(),
				Inline: true,
			})
		}
		if req.ServerName != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Server",
				Value:  req.ServerName,
				Inline: true,
			})
		}
		for _, chanId := range config.Discord.ModChannels {
			select {
			case web.botSendMessageChan <- discordPayload{channelId: chanId, embed: embed}:
			default:
				log.Warnf("Cannot send discord payload, channel full")
				responseErr(ctx, http.StatusInternalServerError, nil)
				return
			}
		}
		responseOK(ctx, http.StatusOK, gin.H{
			"client":  req.Client,
			"message": "Moderators have been notified",
		})
	}
}
func (web *web) onAPIPostBanState(database store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportId, errId := getInt64Param(ctx, "report_id")
		if errId != nil || reportId <= 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var report model.Report
		if errReport := database.GetReport(ctx, reportId, &report); errReport != nil {
			if errors.Is(errReport, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		web.botSendMessageChan <- discordPayload{channelId: "", embed: nil}
	}
}

func (web *web) onAPIPostBanDelete(database store.Store) gin.HandlerFunc {
	type apiUnbanRequest struct {
		UnbanReasonText string `json:"unban_reason_text"`
	}
	return func(ctx *gin.Context) {
		banId, banIdErr := getInt64Param(ctx, "ban_id")
		if banIdErr != nil {
			responseErr(ctx, http.StatusBadRequest, "Invalid ban_id format")
			return
		}
		var req apiUnbanRequest
		if errBind := ctx.BindJSON(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, "Invalid request")
			return
		}
		bp := model.NewBannedPerson()
		if banErr := database.GetBanByBanID(ctx, banId, false, &bp); banErr != nil {
			responseErr(ctx, http.StatusInternalServerError, "Failed to query")
			return
		}
		changed, errSave := Unban(ctx, database, bp.Person.SteamID, req.UnbanReasonText)
		if errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, "Failed to unban")
			return
		}
		if !changed {
			responseErr(ctx, http.StatusConflict, "Failed to save")
			return
		}
		responseOK(ctx, http.StatusAccepted, nil)
	}
}

func (web *web) onAPIPostBanCreate(database store.Store) gin.HandlerFunc {
	type apiBanRequest struct {
		SteamID    steamid.SID64 `json:"steam_id"`
		Duration   string        `json:"duration"`
		BanType    model.BanType `json:"ban_type"`
		Reason     model.Reason  `json:"reason"`
		ReasonText string        `json:"reason_text"`
		Note       string        `json:"note"`
		Network    string        `json:"network"`
		ReportId   int           `json:"report_id"`
	}
	return func(ctx *gin.Context) {
		var banRequest apiBanRequest
		if errBind := ctx.BindJSON(&banRequest); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, "Failed to perform ban")
			return
		}
		if banRequest.Network != "" {
			_, _, errParseCIDR := net.ParseCIDR(banRequest.Network)
			if errParseCIDR != nil {
				responseErr(ctx, http.StatusBadRequest, "Invalid network cidr definition")
				return
			}
		}
		if !banRequest.SteamID.Valid() {
			responseErr(ctx, http.StatusBadRequest, "Invalid steamid")
			return
		}
		newBanOpts := banOpts{
			target:     model.Target(banRequest.SteamID.String()),
			author:     model.Target(currentUserProfile(ctx).SteamID.String()),
			duration:   model.Duration(banRequest.Duration),
			banType:    banRequest.BanType,
			reason:     banRequest.Reason,
			reasonText: banRequest.ReasonText,
			modNote:    banRequest.Note,
			origin:     model.Web,
			reportId:   banRequest.ReportId,
		}
		if banRequest.Network != "" {
			banNetOpts := banNetworkOpts{
				banOpts: newBanOpts,
				cidr:    banRequest.Network,
			}
			var banNet model.BanNet
			if errBanNetwork := BanNetwork(ctx, database, banNetOpts, &banNet); errBanNetwork != nil {
				if errors.Is(errBanNetwork, store.ErrDuplicate) {
					responseErr(ctx, http.StatusConflict, "Duplicate ban")
					return
				}
				responseErr(ctx, http.StatusBadRequest, "Failed to perform ban")
				return
			}
			responseOK(ctx, http.StatusCreated, banNet)
		} else {
			var ban model.Ban
			if errBan := Ban(ctx, database, newBanOpts, &ban, web.botSendMessageChan); errBan != nil {
				if errors.Is(errBan, store.ErrDuplicate) {
					responseErr(ctx, http.StatusConflict, "Duplicate ban")
					return
				}
				responseErr(ctx, http.StatusBadRequest, "Failed to perform ban")
				return
			}
			responseOK(ctx, http.StatusCreated, ban)
		}
	}
}

func (web *web) onSAPIPostServerAuth(database store.Store) gin.HandlerFunc {
	type authReq struct {
		ServerName string `json:"server_name"`
		Key        string `json:"key"`
	}
	type authResp struct {
		Status bool   `json:"status"`
		Token  string `json:"token"`
	}
	return func(ctx *gin.Context) {
		var request authReq
		if errBind := ctx.BindJSON(&request); errBind != nil {
			log.Errorf("Failed to decode auth request: %v", errBind)
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		var server model.Server
		errGetServer := database.GetServerByName(ctx, request.ServerName, &server)
		if errGetServer != nil {
			responseErr(ctx, http.StatusNotFound, nil)
			return
		}
		if server.Password != request.Key {
			responseErr(ctx, http.StatusForbidden, nil)
			log.Warnf("Invalid server key used: %s", util.SanitizeLog(request.ServerName))
			return
		}
		server.Token = golib.RandomString(40)
		server.TokenCreatedOn = config.Now()
		if errSaveServer := database.SaveServer(ctx, &server); errSaveServer != nil {
			log.Errorf("Failed to updated server token: %v", errSaveServer)
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, authResp{Status: true, Token: server.Token})
	}
}

func (web *web) onPostServerCheck(database store.Store) gin.HandlerFunc {
	type checkRequest struct {
		ClientID int         `json:"client_id"`
		SteamID  steamid.SID `json:"steam_id"`
		IP       net.IP      `json:"ip"`
		Name     string      `json:"name,omitempty"`
	}
	type checkResponse struct {
		ClientID int           `json:"client_id"`
		SteamID  steamid.SID   `json:"steam_id"`
		BanType  model.BanType `json:"ban_type"`
		Msg      string        `json:"msg"`
	}
	return func(ctx *gin.Context) {
		var request checkRequest
		if errBind := ctx.BindJSON(&request); errBind != nil {
			responseErr(ctx, http.StatusInternalServerError, checkResponse{
				BanType: model.Unknown,
				Msg:     "Error determining state",
			})
			return
		}
		resp := checkResponse{
			ClientID: request.ClientID,
			SteamID:  request.SteamID,
			BanType:  model.Unknown,
			Msg:      "",
		}
		responseCtx, cancelResponse := context.WithTimeout(ctx, time.Second*15)
		defer cancelResponse()
		// Check SteamID
		steamID := steamid.SIDToSID64(request.SteamID)
		if !steamID.Valid() {
			resp.Msg = "Invalid steam id"
			responseErr(ctx, http.StatusBadRequest, resp)
			return
		}
		var person model.Person
		if errPerson := getOrCreateProfileBySteamID(responseCtx, database, steamID, request.IP.String(), &person); errPerson != nil {
			responseErr(ctx, http.StatusInternalServerError, checkResponse{
				BanType: model.Unknown,
				Msg:     "Error updating profile state",
			})
			return
		}
		if errAddHist := database.AddConnectionHistory(ctx, &model.PersonConnection{
			IPAddr:      request.IP,
			SteamId:     steamid.SIDToSID64(request.SteamID),
			PersonaName: request.Name,
			CreatedOn:   config.Now(),
			IPInfo:      model.PersonIPRecord{},
		}); errAddHist != nil {
			log.Errorf("Failed to add conn history: %v", errAddHist)
		}
		// Check IP first
		banNet, errGetBanNet := database.GetBanNet(responseCtx, request.IP)
		if errGetBanNet != nil {
			responseErr(ctx, http.StatusInternalServerError, checkResponse{
				BanType: model.Unknown,
				Msg:     "Error determining state",
			})
			log.Errorf("Could not get bannedPerson net results: %v", errGetBanNet)
			return
		}
		if len(banNet) > 0 {
			resp.BanType = model.Banned
			resp.Msg = fmt.Sprintf("Network banned (C: %d)", len(banNet))
			responseOK(ctx, http.StatusOK, resp)
			log.WithFields(log.Fields{"type": "cidr", "reason": banNet[0].Reason,
				"steam_id": steamid.SIDToSID64(request.SteamID)}).Infof("Player dropped")
			return
		}
		var asnRecord ip2location.ASNRecord
		errASN := database.GetASNRecordByIP(responseCtx, request.IP, &asnRecord)
		if errASN == nil {
			var asnBan model.BanASN
			if errASNBan := database.GetBanASN(responseCtx, int64(asnRecord.ASNum), &asnBan); errASNBan != nil {
				if !errors.Is(errASNBan, store.ErrNoResult) {
					log.Errorf("Failed to fetch asn bannedPerson: %v", errASNBan)
				}
			} else {
				resp.BanType = model.Banned
				resp.Msg = asnBan.Reason.String()
				responseOK(ctx, http.StatusOK, resp)
				log.WithFields(log.Fields{"type": "asn", "reason": asnBan.Reason}).Infof("Player dropped")
				return
			}
		}
		bannedPerson := model.NewBannedPerson()
		if errGetBan := database.GetBanBySteamID(responseCtx, steamID, false, &bannedPerson); errGetBan != nil {
			if errGetBan == store.ErrNoResult {
				resp.BanType = model.OK
				responseErr(ctx, http.StatusOK, resp)
				return
			}
			resp.Msg = "Error determining state"
			responseErr(ctx, http.StatusInternalServerError, resp)
			return
		}
		resp.BanType = bannedPerson.Ban.BanType
		reason := ""
		if bannedPerson.Ban.Reason == model.Custom && bannedPerson.Ban.ReasonText != "" {
			reason = bannedPerson.Ban.ReasonText
		} else if bannedPerson.Ban.Reason == model.Custom && bannedPerson.Ban.ReasonText == "" {
			reason = "Banned"
		} else {
			reason = bannedPerson.Ban.Reason.String()
		}
		resp.Msg = reason
		responseOK(ctx, http.StatusOK, resp)
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
//		if errBind := c.BindJSON(&app); errBind != nil {
//			log.Errorf("Received malformed appeal apiBanRequest: %v", errBind)
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
//func (w *web) onAPIGetAnsibleHosts(database store.Store) gin.HandlerFunc {
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
//		servers, errGetServers := database.GetServers(ctx, true)
//		if errGetServers != nil {
//			log.Errorf("Failed to fetch servers: %s", errGetServers)
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
func (web *web) onAPIGetPrometheusHosts(database store.Store) gin.HandlerFunc {
	type promStaticConfig struct {
		Targets []string          `json:"targets"`
		Labels  map[string]string `json:"labels"`
	}
	type portMap struct {
		Type string
		Port int
	}
	return func(ctx *gin.Context) {
		var staticConfigs []promStaticConfig
		servers, errGetServers := database.GetServers(ctx, true)
		if errGetServers != nil {
			log.Errorf("Failed to fetch servers: %s", errGetServers)
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		for _, nodePortConfig := range []portMap{{"node", 9100}} {
			staticConfig := promStaticConfig{Targets: nil, Labels: map[string]string{}}
			staticConfig.Labels["__meta_prometheus_job"] = nodePortConfig.Type
			for _, server := range servers {
				host := fmt.Sprintf("%s:%d", server.Address, nodePortConfig.Port)
				found := false
				for _, hostName := range staticConfig.Targets {
					if hostName == host {
						found = true
						break
					}
				}
				if !found {
					staticConfig.Targets = append(staticConfig.Targets, host)
				}
			}
			staticConfigs = append(staticConfigs, staticConfig)
		}
		// Don't wrap in our custom response format
		ctx.JSON(200, staticConfigs)
	}
}

// onAPIGetServers returns the current known cached server state
func (web *web) onAPIGetServers() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		time.Sleep(5 * time.Second)
		responseOK(ctx, http.StatusOK, ServerState())
	}
}

func (web *web) queryFilterFromContext(ctx *gin.Context) (*store.QueryFilter, error) {
	var queryFilter store.QueryFilter
	if errBind := ctx.BindUri(&queryFilter); errBind != nil {
		return nil, errBind
	}
	return &queryFilter, nil
}

func (web *web) onAPIGetPlayers(database store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		queryFilter, errFilterFromContext := web.queryFilterFromContext(c)
		if errFilterFromContext != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		people, errGetPeople := database.GetPeople(c, queryFilter)
		if errGetPeople != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		responseOK(c, http.StatusOK, people)
	}
}

func (web *web) onAPIGetResolveProfile(database store.PersonStore) gin.HandlerFunc {
	type queryParam struct {
		Query string `json:"query"`
	}
	return func(ctx *gin.Context) {
		var param queryParam
		if errBind := ctx.BindJSON(&param); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		id, errResolve := steamid.ResolveSID64(ctx, param.Query)
		if errResolve != nil {
			responseErr(ctx, http.StatusOK, nil)
			return
		}
		var person model.Person
		if errPerson := getOrCreateProfileBySteamID(ctx, database, id, "", &person); errPerson != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, person)
	}
}

func (web *web) onAPICurrentProfile() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userProfile := currentUserProfile(ctx)
		if !userProfile.SteamID.Valid() {
			responseErr(ctx, http.StatusForbidden, nil)
			return
		}
		responseOK(ctx, http.StatusOK, userProfile)
	}
}

func (web *web) onAPIProfile(database store.Store) gin.HandlerFunc {
	type req struct {
		Query string `form:"query"`
	}
	type resp struct {
		Player  *model.Person            `json:"player"`
		Friends []steamweb.PlayerSummary `json:"friends"`
	}
	return func(ctx *gin.Context) {
		requestCtx, cancelRequest := context.WithTimeout(ctx, time.Second*15)
		defer cancelRequest()
		var request req
		if errBind := ctx.Bind(&request); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		sid, errResolveSID64 := steamid.ResolveSID64(requestCtx, request.Query)
		if errResolveSID64 != nil {
			responseErr(ctx, http.StatusNotFound, nil)
			return
		}
		person := model.NewPerson(sid)
		if errGetProfile := getOrCreateProfileBySteamID(requestCtx, database, sid, "", &person); errGetProfile != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		friendIDs, errFetchFriends := steam.FetchFriends(requestCtx, person.SteamID)
		if errFetchFriends != nil {
			responseErr(ctx, http.StatusServiceUnavailable, "Could not fetch friends")
			return
		}
		// TODO add ctx to steamweb lib
		friends, errFetchSummaries := steam.FetchSummaries(friendIDs)
		if errFetchSummaries != nil {
			responseErr(ctx, http.StatusServiceUnavailable, "Could not fetch summaries")
			return
		}
		var response resp
		response.Player = &person
		response.Friends = friends
		responseOK(ctx, http.StatusOK, response)
	}
}

func (web *web) onAPIGetFilteredWords(database store.Store) gin.HandlerFunc {
	type resp struct {
		Count int      `json:"count"`
		Words []string `json:"words"`
	}
	return func(ctx *gin.Context) {
		words, errGetFilters := database.GetFilters(ctx)
		if errGetFilters != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		var fWords []string
		for _, word := range words {
			for _, pattern := range word.Patterns {
				fWords = append(fWords, pattern)
			}
		}
		responseOK(ctx, http.StatusOK, resp{Count: len(fWords), Words: fWords})
	}
}

func (web *web) onAPIGetCompHist() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sidStr := ctx.DefaultQuery("sid", "")
		if sidStr == "" {
			responseErr(ctx, http.StatusBadRequest, "missing sid")
			return
		}
		sid, errStringToSID64 := steamid.StringToSID64(sidStr)
		if errStringToSID64 != nil || !sid.Valid() {
			responseErr(ctx, http.StatusBadRequest, "invalid sid")
			return
		}
		var hist external.CompHist
		if errFetch := external.FetchCompHist(ctx, sid, &hist); errFetch != nil {
			responseErr(ctx, http.StatusInternalServerError, "query failed")
			return
		}
		responseOK(ctx, http.StatusOK, hist)
	}
}

func (web *web) onAPIGetStats(database store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var stats model.Stats
		if errGetStats := database.GetStats(ctx, &stats); errGetStats != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		stats.ServersAlive = 1
		responseOK(ctx, http.StatusOK, stats)
	}
}

func loadBanMeta(_ *model.BannedPerson) {

}

func (web *web) onAPIGetBanByID(database store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banId, errId := getInt64Param(ctx, "ban_id")
		if errId != nil || banId == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		bannedPerson := model.NewBannedPerson()
		if errGetBan := database.GetBanByBanID(ctx, banId, false, &bannedPerson); errGetBan != nil {
			responseErr(ctx, http.StatusNotFound, nil)
			log.Errorf("Failed to fetch bans: %v", errGetBan)
			return
		}
		loadBanMeta(&bannedPerson)
		responseOK(ctx, http.StatusOK, bannedPerson)
	}
}

func (web *web) onAPIGetBans(database store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var queryFilter store.BansQueryFilter
		if errBind := ctx.BindJSON(&queryFilter); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		bans, errBans := database.GetBans(ctx, &queryFilter)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Errorf("Failed to fetch bans: %v", errBans)
			return
		}
		responseOK(ctx, http.StatusOK, bans)
	}
}

func (web *web) onAPIPostServer(database store.ServerStore) gin.HandlerFunc {
	type newServerReq struct {
		NameShort     string  `json:"name_short"`
		Host          string  `json:"host"`
		Port          int     `json:"port"`
		ReservedSlots int     `json:"reserved_slots"`
		RCON          string  `json:"rcon"`
		Lat           float64 `json:"lat"`
		Lon           float64 `json:"lon"`
		CC            string  `json:"cc"`
		DefaultMap    string  `json:"default_map"`
		Region        string  `json:"region"`
	}

	return func(ctx *gin.Context) {
		var serverReq newServerReq
		if errBind := ctx.BindJSON(&serverReq); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Errorf("Failed to parse request for new server: %v", errBind)
			return
		}
		server := model.NewServer(serverReq.NameShort, serverReq.Host, serverReq.Port)
		server.RCON = serverReq.RCON
		server.ReservedSlots = serverReq.ReservedSlots
		server.DefaultMap = serverReq.DefaultMap
		server.Location.Latitude = serverReq.Lat
		server.Location.Longitude = serverReq.Lon
		server.CC = serverReq.CC
		server.Region = serverReq.Region
		if errSave := database.SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Errorf("Failed to save new server: %v", errSave)
			return
		}
		responseOK(ctx, http.StatusOK, server)
	}
}

func (web *web) onAPIPostAppeal(_ store.Store) gin.HandlerFunc {
	type newAppeal struct {
		BanId  int    `json:"ban_id"`
		Reason string `json:"reason"`
	}
	return func(ctx *gin.Context) {
		var appeal newAppeal
		if errBind := ctx.BindJSON(&appeal); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		responseErr(ctx, http.StatusServiceUnavailable, nil)
	}
}

func (web *web) onAPIPostReportCreate(database store.Store) gin.HandlerFunc {
	type createReport struct {
		SteamId     string       `json:"steam_id"`
		Description string       `json:"description"`
		Reason      model.Reason `json:"reason"`
		ReasonText  string       `json:"reason_text"`
	}
	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)
		var newReport createReport
		if errBind := ctx.BindJSON(&newReport); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Errorf("Failed to bind report: %v", errBind)
			return
		}
		sid, errSid := steamid.ResolveSID64(ctx, newReport.SteamId)
		if errSid != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Errorf("Invaid steam_id: %v", errSid)
			return
		}
		var person model.Person
		if errCreatePerson := getOrCreateProfileBySteamID(ctx, database, sid, "", &person); errCreatePerson != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Errorf("Could not load player profile: %v", errCreatePerson)
			return
		}
		// TODO encapsulate all operations in single tx
		report := model.NewReport()
		report.AuthorId = currentUser.SteamID
		report.ReportStatus = model.Opened
		report.Description = newReport.Description
		report.ReportedId = sid
		report.Reason = newReport.Reason
		report.ReasonText = newReport.ReasonText
		if errReportSave := database.SaveReport(ctx, &report); errReportSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Errorf("Failed to save report: %v", errReportSave)
			return
		}
		responseOK(ctx, http.StatusCreated, report)

		embed := respOk(nil, "New user report created")
		embed.Description = report.Description
		embed.URL = resolveUrl("/report/%d", report.ReportId)
		embed.Author = &discordgo.MessageEmbedAuthor{
			URL:     resolveUrl("/profile/%d", report.AuthorId.Int64()),
			Name:    currentUser.Name,
			IconURL: currentUser.Avatar,
		}
		name := person.PersonaName
		if name == "" {
			name = report.ReportedId.String()
		}
		addField(embed, "Subject", name)
		addField(embed, "Reason", report.Reason.String())
		if report.ReasonText != "" {
			addField(embed, "Custom Reason", report.ReasonText)
		}
		addFieldsSteamID(embed, report.ReportedId)
		web.botSendMessageChan <- discordPayload{
			channelId: config.Discord.ModLogChannelId,
			embed:     embed}
	}
}

func getSID64Param(c *gin.Context, key string) (steamid.SID64, error) {
	i, errGetParam := getInt64Param(c, key)
	if errGetParam != nil {
		return 0, errGetParam
	}
	sid := steamid.SID64(i)
	if !sid.Valid() {
		return 0, consts.ErrInvalidSID
	}
	return sid, nil
}

func getInt64Param(ctx *gin.Context, key string) (int64, error) {
	valueStr := ctx.Param(key)
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

func getIntParam(ctx *gin.Context, key string) (int, error) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		return 0, errors.Errorf("Failed to get %s", key)
	}
	return util.StringToInt(valueStr), nil
}

func (web *web) onAPIPostReportMessage(database store.ReportStore) gin.HandlerFunc {
	type req struct {
		Message string `json:"message"`
	}
	return func(ctx *gin.Context) {
		reportId, errId := getInt64Param(ctx, "report_id")
		if errId != nil || reportId == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var request req
		if errBind := ctx.BindJSON(&request); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		if request.Message == "" {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var report model.Report
		if errReport := database.GetReport(ctx, reportId, &report); errReport != nil {
			if store.Err(errReport) == store.ErrNoResult {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Errorf("Failed to load report: %v", errReport)
			return
		}
		person := currentUserProfile(ctx)
		msg := model.NewUserMessage(reportId, person.SteamID, request.Message)
		if errSave := database.SaveReportMessage(ctx, &msg); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Errorf("Failed to save report message: %v", errSave)
			return
		}
		responseOK(ctx, http.StatusCreated, msg)

		embed := &discordgo.MessageEmbed{
			Title:       "New report message posted",
			Description: msg.Message,
		}
		addField(embed, "Author", report.AuthorId.String())
		web.botSendMessageChan <- discordPayload{
			channelId: config.Discord.ModLogChannelId,
			embed:     embed}
	}
}

func (web *web) onAPIEditReportMessage(database store.ReportStore) gin.HandlerFunc {
	type editMessage struct {
		Message string `json:"body_md"`
	}
	return func(ctx *gin.Context) {
		reportMessageId, errId := getInt64Param(ctx, "report_message_id")
		if errId != nil || reportMessageId == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var existing model.UserMessage
		if errExist := database.GetReportMessageById(ctx, reportMessageId, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		curUser := currentUserProfile(ctx)
		if !isAllowed(curUser, existing.AuthorId, model.PModerator) {
			responseErr(ctx, http.StatusUnauthorized, nil)
			return
		}
		var message editMessage
		if errBind := ctx.BindJSON(&message); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		if message.Message == "" {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		if message.Message == existing.Message {
			responseErr(ctx, http.StatusConflict, nil)
			return
		}
		embed := &discordgo.MessageEmbed{
			Title:       "New report message edited",
			Description: util.DiffString(existing.Message, message.Message),
		}
		existing.Message = message.Message
		if errSave := database.SaveReportMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Errorf("Failed to save report message: %v", errSave)
			return
		}
		responseOK(ctx, http.StatusCreated, message)

		addField(embed, "Author", curUser.SteamID.String())
		web.botSendMessageChan <- discordPayload{
			channelId: config.Discord.ModLogChannelId,
			embed:     embed}
	}
}

func (web *web) onAPIDeleteReportMessage(database store.ReportStore) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportMessageId, errId := getInt64Param(ctx, "report_message_id")
		if errId != nil || reportMessageId == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var existing model.UserMessage
		if errExist := database.GetReportMessageById(ctx, reportMessageId, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		curUser := currentUserProfile(ctx)
		if !isAllowed(curUser, existing.AuthorId, model.PModerator) {
			responseErr(ctx, http.StatusUnauthorized, nil)
			return
		}
		existing.Deleted = true
		if errSave := database.SaveReportMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Errorf("Failed to save report message: %v", errSave)
			return
		}
		responseOK(ctx, http.StatusNoContent, nil)

		embed := &discordgo.MessageEmbed{
			Title:       "User report message deleted",
			Description: existing.Message,
		}
		addField(embed, "Author", curUser.SteamID.String())
		web.botSendMessageChan <- discordPayload{
			channelId: config.Discord.ModLogChannelId,
			embed:     embed}
	}
}

type AuthorMessage struct {
	Author  model.Person      `json:"author"`
	Message model.UserMessage `json:"message"`
}

func (web *web) onAPISetReportStatus(database store.ReportStore) gin.HandlerFunc {
	type stateUpdateReq struct {
		Status model.ReportStatus `json:"status"`
	}
	return func(c *gin.Context) {
		reportId, errParam := getInt64Param(c, "report_id")
		if errParam != nil {
			responseErr(c, http.StatusNotFound, nil)
			return
		}
		var newStatus stateUpdateReq
		if errBind := c.BindJSON(&newStatus); errBind != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		var report model.Report
		if errGet := database.GetReport(c, reportId, &report); errGet != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			log.Errorf("Failed to get report to set state: %v", errGet)
			return
		}
		if report.ReportStatus == newStatus.Status {
			responseOK(c, http.StatusConflict, nil)
			return
		}
		original := report.ReportStatus
		report.ReportStatus = newStatus.Status
		if errSave := database.SaveReport(c, &report); errSave != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			log.Errorf("Failed to save report state: %v", errSave)
			return
		}
		responseOK(c, http.StatusAccepted, nil)
		log.WithFields(log.Fields{
			"report_id": report.ReportId,
			"from":      original,
			"to":        report.ReportStatus,
		}).Infof("Report status changed")
	}
}

func (web *web) onAPIGetReportMessages(database store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		reportId, errParam := getInt64Param(c, "report_id")
		if errParam != nil {
			responseErr(c, http.StatusNotFound, nil)
			return
		}
		reportMessages, errGetReportMessages := database.GetReportMessages(c, reportId)
		if errGetReportMessages != nil {
			responseErr(c, http.StatusNotFound, nil)
			return
		}
		var ids steamid.Collection
		for _, msg := range reportMessages {
			ids = append(ids, msg.AuthorId)
		}
		authors, authorsErr := database.GetPeopleBySteamID(c, ids)
		if authorsErr != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		authorsMap := authors.AsMap()
		var authorMessages []AuthorMessage
		for _, message := range reportMessages {
			authorMessages = append(authorMessages, AuthorMessage{
				Author:  authorsMap[message.AuthorId],
				Message: message,
			})
		}
		responseOK(c, http.StatusOK, authorMessages)
	}
}

type reportWithAuthor struct {
	Author  model.Person `json:"author"`
	Subject model.Person `json:"subject"`
	Report  model.Report `json:"report"`
}

func (web *web) onAPIGetReports(database store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var opts store.AuthorQueryFilter
		if errBind := ctx.BindJSON(&opts); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		if opts.Limit <= 0 && opts.Limit > 100 {
			opts.Limit = 25
		}
		var userReports []reportWithAuthor
		reports, errReports := database.GetReports(ctx, opts)
		if errReports != nil {
			if store.Err(errReports) == store.ErrNoResult {
				responseOK(ctx, http.StatusNoContent, nil)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		var authorIds steamid.Collection
		for _, report := range reports {
			authorIds = append(authorIds, report.AuthorId)
		}
		authors, errAuthors := database.GetPeopleBySteamID(ctx, fp.Uniq[steamid.SID64](authorIds))
		if errAuthors != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		authorMap := authors.AsMap()

		var subjectIds steamid.Collection
		for _, report := range reports {
			subjectIds = append(subjectIds, report.ReportedId)
		}
		subjects, errSubjects := database.GetPeopleBySteamID(ctx, fp.Uniq[steamid.SID64](subjectIds))
		if errSubjects != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		subjectMap := subjects.AsMap()

		for _, report := range reports {
			userReports = append(userReports, reportWithAuthor{
				Author:  authorMap[report.AuthorId],
				Report:  report,
				Subject: subjectMap[report.ReportedId],
			})
		}

		responseOK(ctx, http.StatusOK, userReports)
	}
}

func (web *web) onAPIGetReport(database store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportId, errParam := getInt64Param(ctx, "report_id")
		if errParam != nil {
			responseErr(ctx, http.StatusNotFound, nil)
			return
		}
		var report reportWithAuthor
		if errReport := database.GetReport(ctx, reportId, &report.Report); errReport != nil {
			if store.Err(errReport) == store.ErrNoResult {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Errorf("Failed to load report: %v", errReport)
			return
		}
		if errAuthor := database.GetOrCreatePersonBySteamID(ctx, report.Report.AuthorId, &report.Author); errAuthor != nil {
			if store.Err(errAuthor) == store.ErrNoResult {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Errorf("Failed to load report author: %v", errAuthor)
			return
		}

		if errSubject := database.GetOrCreatePersonBySteamID(ctx, report.Report.ReportedId, &report.Subject); errSubject != nil {
			if store.Err(errSubject) == store.ErrNoResult {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Errorf("Failed to load report subject: %v", errSubject)
			return
		}
		curUser := currentUserProfile(ctx)

		// Only allow report authors and subjects to view reports made, or mods+.
		if curUser.PermissionLevel == model.PUser &&
			!(report.Subject.SteamID == curUser.SteamID ||
				report.Author.SteamID == curUser.SteamID) {
			responseErrUser(ctx, http.StatusUnauthorized, nil, "Permission denied")
			return
		}

		responseOK(ctx, http.StatusOK, report)
	}
}

func (web *web) onAPIGetNewsLatest(database store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := database.GetNewsLatest(ctx, 5, false)
		if errGetNewsLatest != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, newsLatest)
	}
}

func (web *web) onAPIGetNewsAll(database store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := database.GetNewsLatest(ctx, 100, true)
		if errGetNewsLatest != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, newsLatest)
	}
}

func (web *web) onAPIPostNewsCreate(database store.NewsStore) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var entry model.NewsEntry
		if errBind := ctx.BindJSON(&entry); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		if errSave := database.SaveNewsArticle(ctx, &entry); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusCreated, entry)

		web.botSendMessageChan <- discordPayload{
			channelId: "882471332254715915",
			embed: &discordgo.MessageEmbed{
				Title:       "News Created",
				Description: fmt.Sprintf("News Posted: %s", entry.Title)},
		}
	}
}

func (web *web) onAPIPostNewsUpdate(database store.NewsStore) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsId, errId := getIntParam(ctx, "news_id")
		if errId != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var entry model.NewsEntry
		if errGet := database.GetNewsById(ctx, newsId, &entry); errGet != nil {
			if errors.Is(store.Err(errGet), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		if errBind := ctx.BindJSON(&entry); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		if errSave := database.SaveNewsArticle(ctx, &entry); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusAccepted, entry)

		web.botSendMessageChan <- discordPayload{
			channelId: "882471332254715915",
			embed: &discordgo.MessageEmbed{
				Title:       "News Updated",
				Description: fmt.Sprintf("News Updated: %s", entry.Title)},
		}
	}
}

func (web *web) onAPISaveMedia(database store.MediaStore) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var upload model.UserUploadedFile
		if errBind := ctx.BindJSON(&upload); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		content, decodeErr := base64.StdEncoding.DecodeString(upload.Content)
		if decodeErr != nil {
			responseErr(ctx, http.StatusUnprocessableEntity, nil)
			return
		}
		if int64(len(content)) != upload.Size {
			responseErr(ctx, http.StatusUnprocessableEntity, nil)
			return
		}
		media := model.Media{
			AuthorId:  currentUserProfile(ctx).SteamID,
			MimeType:  upload.Mime,
			Size:      upload.Size,
			Contents:  content,
			Name:      upload.Name,
			Deleted:   false,
			CreatedOn: config.Now(),
			UpdatedOn: config.Now(),
		}
		if errSave := database.SaveMedia(ctx, &media); errSave != nil {
			log.Errorf("Failed to save wiki media: %v", errSave)
			if errors.Is(store.Err(errSave), store.ErrDuplicate) {
				responseErrUser(ctx, http.StatusConflict, nil, "Duplicate media name")
				return
			}
			responseErrUser(ctx, http.StatusInternalServerError, nil, "Could not same media")
			return
		}
		responseOKUser(ctx, http.StatusAccepted, media, "Media uploaded successfully")
	}
}

func (web *web) onAPIGetWikiSlug(database store.WikiStore) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slug := ctx.Param("slug")
		if slug[0] == '/' {
			slug = slug[1:]
		}
		var page wiki.Page
		if errGetWikiSlug := database.GetWikiPageBySlug(ctx, slug, &page); errGetWikiSlug != nil {
			if errors.Is(errGetWikiSlug, store.ErrNoResult) {
				responseOK(ctx, http.StatusOK, page)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, page)
	}
}
func (web *web) onGetMedia(database store.MediaStore) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		name := ctx.Param("name")
		if name[0] == '/' {
			name = name[1:]
		}
		var media model.Media
		if errMedia := database.GetMediaByName(ctx, name, &media); errMedia != nil {
			if errors.Is(store.Err(errMedia), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
			} else {
				responseErr(ctx, http.StatusInternalServerError, nil)
			}
			return
		}
		ctx.Data(http.StatusOK, media.MimeType, media.Contents)
	}
}

func (web *web) onAPISaveWikiSlug(database store.WikiStore) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var request wiki.Page
		if errBind := ctx.BindJSON(&request); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		if request.Slug == "" || request.BodyMD == "" {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var page wiki.Page
		if errGetWikiSlug := database.GetWikiPageBySlug(ctx, request.Slug, &page); errGetWikiSlug != nil {
			if errors.Is(errGetWikiSlug, store.ErrNoResult) {
				page.CreatedOn = time.Now()
				page.Revision += 1
				page.Slug = request.Slug
			} else {
				responseErr(ctx, http.StatusInternalServerError, nil)
				return
			}
		} else {
			page = page.NewRevision()
		}
		page.BodyMD = request.BodyMD
		if errSave := database.SaveWikiPage(ctx, &page); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusCreated, page)
	}
}

func (web *web) onAPIGetMatches(database store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var opts store.MatchesQueryOpts
		if errBind := ctx.BindJSON(&opts); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		matches, matchesErr := database.Matches(ctx, opts)
		if matchesErr != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, matches)
	}
}

func (web *web) onAPIGetMatch(database store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		matchId, errId := getIntParam(ctx, "match_id")
		if errId != nil {
			log.Errorf("Invalid match_id value")
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		match, errMatch := database.MatchGetById(ctx, matchId)
		if errMatch != nil {
			if errors.Is(errMatch, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			log.WithFields(log.Fields{"match_id": matchId}).
				Errorf("Failed to load match: %v", errMatch)
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, match)
	}
}

func (web *web) onAPIGetPersonConnections(database store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamId, errId := getSID64Param(ctx, "steam_id")
		if errId != nil {
			log.Errorf("Invalid steam_id value")
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		// TODO paging
		ipHist, errIpHist := database.GetPersonIPHistory(ctx, steamId, 1000)
		if errIpHist != nil && !errors.Is(errIpHist, store.ErrNoResult) {
			log.WithFields(log.Fields{"sid": steamId}).Errorf("Failed to query connection history: %v", errIpHist)
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		if ipHist == nil {
			ipHist = model.PersonConnections{}
		}
		responseOK(ctx, http.StatusOK, ipHist)
	}
}

func (web *web) onAPIGetPersonMessages(database store.Store) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamId, errId := getSID64Param(ctx, "steam_id")
		if errId != nil {
			log.Errorf("Invalid steam_id value")
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		// TODO paging
		chat, errChat := database.GetChatHistory(ctx, steamId, 1000)
		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			log.WithFields(log.Fields{"sid": steamId}).Errorf("Failed to query chat history: %v", errChat)
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		if chat == nil {
			chat = model.PersonMessages{}
		}
		responseOK(ctx, http.StatusOK, chat)
	}
}

type AuthorBanMessage struct {
	Author  model.Person      `json:"author"`
	Message model.UserMessage `json:"message"`
}

func (web *web) onAPIGetBanMessages(database store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		banId, errParam := getInt64Param(c, "ban_id")
		if errParam != nil {
			responseErr(c, http.StatusNotFound, nil)
			return
		}
		banMessages, errGetBanMessages := database.GetBanMessages(c, banId)
		if errGetBanMessages != nil {
			responseErr(c, http.StatusNotFound, nil)
			return
		}
		var ids steamid.Collection
		for _, msg := range banMessages {
			ids = append(ids, msg.AuthorId)
		}
		authors, authorsErr := database.GetPeopleBySteamID(c, ids)
		if authorsErr != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		authorsMap := authors.AsMap()
		var authorMessages []AuthorBanMessage
		for _, message := range banMessages {
			authorMessages = append(authorMessages, AuthorBanMessage{
				Author:  authorsMap[message.AuthorId],
				Message: message,
			})
		}
		responseOK(c, http.StatusOK, authorMessages)
	}
}

func (web *web) onAPIDeleteBanMessage(database store.BanStore) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banMessageId, errId := getIntParam(ctx, "ban_message_id")
		if errId != nil || banMessageId == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var existing model.UserMessage
		if errExist := database.GetBanMessageById(ctx, banMessageId, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		curUser := currentUserProfile(ctx)
		if !isAllowed(curUser, existing.AuthorId, model.PModerator) {
			responseErr(ctx, http.StatusUnauthorized, nil)
			return
		}
		existing.Deleted = true
		if errSave := database.SaveBanMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Errorf("Failed to save appeal message: %v", errSave)
			return
		}
		responseOK(ctx, http.StatusNoContent, nil)

		embed := &discordgo.MessageEmbed{
			Title:       "User appeal message deleted",
			Description: existing.Message,
		}
		addField(embed, "Author", curUser.SteamID.String())
		web.botSendMessageChan <- discordPayload{
			channelId: config.Discord.ModLogChannelId,
			embed:     embed}
	}
}

func (web *web) onAPIPostBanMessage(database store.BanStore) gin.HandlerFunc {
	type req struct {
		Message string `json:"message"`
	}
	return func(ctx *gin.Context) {
		banId, errId := getInt64Param(ctx, "ban_id")
		if errId != nil || banId == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var request req
		if errBind := ctx.BindJSON(&request); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		if request.Message == "" {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		bp := model.NewBannedPerson()
		if errReport := database.GetBanByBanID(ctx, banId, false, &bp); errReport != nil {
			if store.Err(errReport) == store.ErrNoResult {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Errorf("Failed to load ban: %v", errReport)
			return
		}
		person := currentUserProfile(ctx)
		msg := model.NewUserMessage(banId, person.SteamID, request.Message)
		if errSave := database.SaveBanMessage(ctx, &msg); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Errorf("Failed to save ban appeal message: %v", errSave)
			return
		}
		responseOK(ctx, http.StatusCreated, msg)

		embed := &discordgo.MessageEmbed{
			Title:       "New ban appeal message posted",
			Description: msg.Message,
		}
		addField(embed, "Author", msg.AuthorId.String())
		web.botSendMessageChan <- discordPayload{
			channelId: config.Discord.ModLogChannelId,
			embed:     embed}
	}
}

func (web *web) onAPIEditBanMessage(database store.BanStore) gin.HandlerFunc {
	type editMessage struct {
		Message string `json:"body_md"`
	}
	return func(ctx *gin.Context) {
		reportMessageId, errId := getIntParam(ctx, "ban_message_id")
		if errId != nil || reportMessageId == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var existing model.UserMessage
		if errExist := database.GetBanMessageById(ctx, reportMessageId, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		curUser := currentUserProfile(ctx)
		if !isAllowed(curUser, existing.AuthorId, model.PModerator) {
			responseErr(ctx, http.StatusUnauthorized, nil)
			return
		}
		var message editMessage
		if errBind := ctx.BindJSON(&message); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		if message.Message == "" {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		if message.Message == existing.Message {
			responseErr(ctx, http.StatusConflict, nil)
			return
		}
		embed := &discordgo.MessageEmbed{
			Title:       "Ban appeal message edited",
			Description: util.DiffString(existing.Message, message.Message),
		}
		existing.Message = message.Message
		if errSave := database.SaveBanMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Errorf("Failed to save ban appeal message: %v", errSave)
			return
		}
		responseOK(ctx, http.StatusCreated, message)

		addField(embed, "Author", curUser.SteamID.String())
		web.botSendMessageChan <- discordPayload{
			channelId: config.Discord.ModLogChannelId,
			embed:     embed}
	}
}
