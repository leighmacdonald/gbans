package web

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/srcdsup/srcdsup"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"github.com/ryanuber/go-glob"
	"go.uber.org/zap"
	"math"
	"net"
	"net/http"
	"regexp"
	"sort"
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
	Error   string `json:"error,omitempty"`
	Result  any    `json:"result"`
}

func responseErrUser(ctx *gin.Context, status int, data any, userMsg string, args ...any) {
	ctx.JSON(status, apiResponse{
		Status: false,
		Error:  fmt.Sprintf(userMsg, args...),
		Result: data,
	})
}

func responseErr(ctx *gin.Context, status int, data any) {
	ctx.JSON(status, apiResponse{Status: false, Result: data})
}

func responseOKUser(ctx *gin.Context, status int, data any, userMsg string, args ...any) {
	ctx.JSON(status, apiResponse{
		Status:  true,
		Message: fmt.Sprintf(userMsg, args...),
		Result:  data,
	})
}

func responseOK(ctx *gin.Context, status int, data any) {
	ctx.JSON(status, apiResponse{Status: true, Result: data})
}

func onAPIPostLog() gin.HandlerFunc {
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
		var server store.Server
		if errServer := store.GetServerByName(ctx, upload.ServerName, &server); errServer != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		rawLogs, errDecode := base64.StdEncoding.DecodeString(upload.Body)
		if errDecode != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		logLines := strings.Split(string(rawLogs), "\n")
		logger.Debug("Uploaded log file", zap.Int("lines", len(logLines)))
		responseOKUser(ctx, http.StatusCreated, nil, "Log uploaded")
		// Send the log to the logReader() for actual processing
		// TODO deal with this potential block
		panic("xx")
		//app.LogFileChan <- &model.LogFilePayload{
		//	Server: server,
		//	Lines:  logLines,
		//	Map:    upload.MapName,
		//}
	}
}

func onAPIPostDemo() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var upload srcdsup.ServerLogUpload
		if errBind := ctx.BindJSON(&upload); errBind != nil {
			logger.Error("Failed to parse demo payload", zap.Error(errBind))
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		if upload.ServerName == "" || upload.Body == "" {
			logger.Error("Missing demo params",
				zap.String("server_name", util.SanitizeLog(upload.ServerName)),
				zap.String("map_name", util.SanitizeLog(upload.MapName)),
				zap.Int("body_len", len(upload.Body)))
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var server store.Server
		if errGetServer := store.GetServerByName(ctx, upload.ServerName, &server); errGetServer != nil {
			logger.Error("Server not found", zap.String("server", util.SanitizeLog(upload.ServerName)))
			responseErrUser(ctx, http.StatusNotFound, nil, "Server not found: %v", upload.ServerName)
			return
		}
		rawDemo, errDecode := base64.StdEncoding.DecodeString(upload.Body)
		if errDecode != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}

		// Convert string based sid to int64
		// NOTE Should probably be sent as a string but sourcemod BigNum is ???
		intStats := map[steamid.SID64]srcdsup.PlayerStats{}
		for steamId, v := range upload.Scores {
			sid64, errSid := steamid.SID64FromString(steamId)
			if errSid != nil {
				logger.Error("Failed to parse score steam id", zap.Error(errSid))
				continue
			}
			intStats[sid64] = v
		}
		newDemo := store.DemoFile{
			ServerID:  server.ServerID,
			Title:     upload.DemoName,
			Data:      rawDemo,
			Size:      int64(len(rawDemo)),
			CreatedOn: config.Now(),
			MapName:   upload.MapName,
			Stats:     intStats,
		}
		if errSave := store.SaveDemo(ctx, &newDemo); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to save demo", zap.Error(errSave))
			return
		}
		responseOK(ctx, http.StatusCreated, gin.H{"demo_id": newDemo.DemoID})
	}
}

func onAPIGetDemoDownload() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		demoId, errId := getInt64Param(ctx, "demo_id")
		if errId != nil || demoId <= 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			logger.Error("Invalid demo id requested", zap.Error(errId))
			return
		}
		var demo store.DemoFile
		if errGet := store.GetDemoById(ctx, demoId, &demo); errGet != nil {
			if errors.Is(errGet, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Error fetching demo", zap.Error(errGet))
			return
		}
		ctx.Header("Content-Description", "File Transfer")
		ctx.Header("Content-Transfer-Encoding", "binary")
		ctx.Header("Content-Disposition", "attachment; filename="+demo.Title)
		ctx.Data(http.StatusOK, "application/octet-stream", demo.Data)
		demo.Downloads++
		if errSave := store.SaveDemo(ctx, &demo); errSave != nil {
			logger.Error("Failed to increment download count for demo", zap.Error(errSave))
		}
	}
}

func onAPIGetDemoDownloadByName() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		demoName := ctx.Param("demo_name")
		if demoName == "" {
			responseErr(ctx, http.StatusBadRequest, nil)
			logger.Error("Invalid demo name requested", zap.String("demo_name", ""))
			return
		}
		var demo store.DemoFile
		if errGet := store.GetDemoByName(ctx, demoName, &demo); errGet != nil {
			if errors.Is(errGet, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Error fetching demo", zap.Error(errGet))
			return
		}
		ctx.Header("Content-Description", "File Transfer")
		ctx.Header("Content-Transfer-Encoding", "binary")
		ctx.Header("Content-Disposition", "attachment; filename="+demo.Title)
		ctx.Data(http.StatusOK, "application/octet-stream", demo.Data)
		demo.Downloads++
		if errSave := store.SaveDemo(ctx, &demo); errSave != nil {
			logger.Error("Failed to increment download count for demo", zap.Error(errSave))
		}
	}
}

func onAPIPostDemosQuery() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var opts store.GetDemosOptions
		if errBind := ctx.BindJSON(&opts); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			logger.Error("Malformed demo query request", zap.Error(errBind))
			return
		}
		demos, errDemos := store.GetDemos(ctx, opts)
		if errDemos != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			logger.Error("Failed to query demos", zap.Error(errDemos))
			return
		}
		responseOK(ctx, http.StatusCreated, demos)
	}
}

func onAPIGetServerAdmins() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		perms, err := store.GetServerPermissions(ctx)
		if err != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, perms)
	}
}

func onAPIPostPingMod() gin.HandlerFunc {
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
		players, found := state.Find(state.FindOpts{SteamID: req.SteamID})
		if !found {
			logger.Error("Failed to find player on /mod call")
			responseErr(ctx, http.StatusFailedDependency, nil)
			return
		}
		//name := req.SteamID.String()
		//if playerInfo.InGame {
		//	name = fmt.Sprintf("%s (%s)", name, playerInfo.Player.Name)
		//}
		var roleStrings []string
		for _, roleID := range config.Discord.ModRoleIDs {
			roleStrings = append(roleStrings, fmt.Sprintf("<@&%s>", roleID))
		}
		embed := discord.RespOk(nil, "New User Report")
		embed.Description = fmt.Sprintf("%s | %s", req.Reason, strings.Join(roleStrings, " "))
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Reporter",
			Value:  players[0].Player.Name,
			Inline: true,
		})

		if req.SteamID.String() != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "ReporterSID",
				Value:  players[0].Player.SID.String(),
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
			discord.SendPayload(discord.Payload{ChannelId: chanId, Embed: embed})
		}
		responseOK(ctx, http.StatusOK, gin.H{
			"client":  req.Client,
			"message": "Moderators have been notified",
		})
	}
}

func onAPIPostBanState() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportId, errId := getInt64Param(ctx, "report_id")
		if errId != nil || reportId <= 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var report store.Report
		if errReport := store.GetReport(ctx, reportId, &report); errReport != nil {
			if errors.Is(errReport, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		discord.SendPayload(discord.Payload{ChannelId: "", Embed: nil})
	}
}

type apiUnbanRequest struct {
	UnbanReasonText string `json:"unban_reason_text"`
}

func onAPIPostSetBanAppealStatus() gin.HandlerFunc {
	type setStatusReq struct {
		AppealState store.AppealState `json:"appeal_state"`
	}
	return func(ctx *gin.Context) {
		banId, banIdErr := getInt64Param(ctx, "ban_id")
		if banIdErr != nil {
			responseErr(ctx, http.StatusBadRequest, "Invalid ban_id format")
			return
		}
		var req setStatusReq
		if errBind := ctx.BindJSON(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, "Invalid request")
			return
		}
		bp := store.NewBannedPerson()
		if banErr := store.GetBanByBanID(ctx, banId, &bp, false); banErr != nil {
			responseErr(ctx, http.StatusInternalServerError, "Failed to query")
			return
		}
		if bp.Ban.AppealState == req.AppealState {
			responseErr(ctx, http.StatusConflict, "State must be different than previous")
			return
		}
		original := bp.Ban.AppealState
		bp.Ban.AppealState = req.AppealState
		if errSave := store.SaveBan(ctx, &bp.Ban); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, "Failed to save appeal state changes")
			return
		}
		responseOK(ctx, http.StatusAccepted, nil)
		logger.Info("Updated ban appeal state",
			zap.Int64("ban_id", banId),
			zap.Int("from_state", int(original)),
			zap.Int("to_state", int(req.AppealState)))
	}
}

func onAPIPostBanDelete() gin.HandlerFunc {
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
		bp := store.NewBannedPerson()
		if banErr := store.GetBanByBanID(ctx, banId, &bp, false); banErr != nil {
			responseErr(ctx, http.StatusInternalServerError, "Failed to query")
			return
		}
		changed, errSave := app.Unban(ctx, bp.Person.SteamID, req.UnbanReasonText)
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

func onAPIPostBansGroupCreate() gin.HandlerFunc {
	type apiBanRequest struct {
		TargetId   store.StringSID `json:"target_id"`
		GroupId    steamid.GID     `json:"group_id,string"`
		BanType    store.BanType   `json:"ban_type"`
		Duration   string          `json:"duration"`
		Note       string          `json:"note"`
		Reason     store.Reason    `json:"reason"`
		ReasonText string          `json:"reason_text"`
	}
	return func(ctx *gin.Context) {
		var banRequest apiBanRequest
		if errBind := ctx.BindJSON(&banRequest); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, "Failed to perform ban")
			return
		}
		var banSteamGroup store.BanGroup
		sid := currentUserProfile(ctx).SteamID
		if errBanSteamGroup := store.NewBanSteamGroup(
			store.StringSID(sid.String()),
			banRequest.TargetId,
			store.Duration(banRequest.Duration),
			banRequest.Reason,
			banRequest.ReasonText,
			"",
			store.Web,
			banRequest.GroupId,
			"",
			banRequest.BanType,
			&banSteamGroup,
		); errBanSteamGroup != nil {
			responseErr(ctx, http.StatusBadRequest, "Failed to parse options")
			return
		}
		if errBan := app.BanSteamGroup(ctx, &banSteamGroup); errBan != nil {
			if errors.Is(errBan, store.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, "Duplicate steam group ban")
				return
			}
			responseErr(ctx, http.StatusBadRequest, "Failed to perform steam group ban")
			return
		}
		responseOK(ctx, http.StatusCreated, banSteamGroup)
	}
}

func onAPIPostBansASNCreate() gin.HandlerFunc {
	type apiBanRequest struct {
		TargetId   store.StringSID `json:"target_id"`
		BanType    store.BanType   `json:"ban_type"`
		Duration   string          `json:"duration"`
		Note       string          `json:"note"`
		Reason     store.Reason    `json:"reason"`
		ReasonText string          `json:"reason_text"`
		ASNum      int64           `json:"as_num"`
	}
	return func(ctx *gin.Context) {
		var banRequest apiBanRequest
		if errBind := ctx.BindJSON(&banRequest); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, "Failed to perform asn ban")
			return
		}
		var banASN store.BanASN
		sid := currentUserProfile(ctx).SteamID
		if errBanSteamGroup := store.NewBanASN(
			store.StringSID(sid.String()),
			banRequest.TargetId,
			store.Duration(banRequest.Duration),
			banRequest.Reason,
			banRequest.ReasonText,
			banRequest.Note,
			store.Web,
			banRequest.ASNum,
			banRequest.BanType,
			&banASN,
		); errBanSteamGroup != nil {
			responseErr(ctx, http.StatusBadRequest, "Failed to parse options")
			return
		}
		if errBan := app.BanASN(ctx, &banASN); errBan != nil {
			if errors.Is(errBan, store.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, "Duplicate asn ban")
				return
			}
			responseErr(ctx, http.StatusBadRequest, "Failed to perform asn ban")
			return
		}
		responseOK(ctx, http.StatusCreated, banASN)
	}
}

func onAPIPostBansCIDRCreate() gin.HandlerFunc {
	type apiBanRequest struct {
		TargetId   store.StringSID `json:"target_id"`
		BanType    store.BanType   `json:"ban_type"`
		Duration   string          `json:"duration"`
		Note       string          `json:"note"`
		Reason     store.Reason    `json:"reason"`
		ReasonText string          `json:"reason_text"`
		CIDR       string          `json:"cidr"`
	}
	return func(ctx *gin.Context) {
		var banRequest apiBanRequest
		if errBind := ctx.BindJSON(&banRequest); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, "Failed to perform ban")
			return
		}
		var banCIDR store.BanCIDR
		sid := currentUserProfile(ctx).SteamID
		if errBanCIDR := store.NewBanCIDR(
			store.StringSID(sid.String()),
			banRequest.TargetId,
			store.Duration(banRequest.Duration),
			banRequest.Reason,
			banRequest.ReasonText,
			banRequest.Note,
			store.Web,
			banRequest.CIDR,
			banRequest.BanType,
			&banCIDR,
		); errBanCIDR != nil {
			responseErr(ctx, http.StatusBadRequest, "Failed to parse options")
			return
		}
		if errBan := app.BanCIDR(ctx, &banCIDR); errBan != nil {
			if errors.Is(errBan, store.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, "Duplicate cidr ban")
				return
			}
			responseErr(ctx, http.StatusBadRequest, "Failed to perform cidr ban")
			return
		}
		responseOK(ctx, http.StatusCreated, banCIDR)
	}
}

func onAPIPostBanSteamCreate() gin.HandlerFunc {
	type apiBanRequest struct {
		SourceId   store.StringSID `json:"source_id"`
		TargetId   store.StringSID `json:"target_id"`
		Duration   string          `json:"duration"`
		BanType    store.BanType   `json:"ban_type"`
		Reason     store.Reason    `json:"reason"`
		ReasonText string          `json:"reason_text"`
		Note       string          `json:"note"`
		ReportId   int64           `json:"report_id"`
		DemoName   string          `json:"demo_name"`
		DemoTick   int             `json:"demo_tick"`
	}
	return func(ctx *gin.Context) {
		var banRequest apiBanRequest
		if errBind := ctx.BindJSON(&banRequest); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, "Failed to perform ban")
			return
		}
		origin := store.Web
		sid := currentUserProfile(ctx).SteamID
		sourceId := store.StringSID(sid.String())
		// srcds sourced bans provide a source_id to id the admin
		if banRequest.SourceId != "" {
			sourceId = banRequest.SourceId
			origin = store.InGame
		}
		var banSteam store.BanSteam
		if errBanSteam := store.NewBanSteam(
			sourceId,
			banRequest.TargetId,
			store.Duration(banRequest.Duration),
			banRequest.Reason,
			banRequest.ReasonText,
			banRequest.Note,
			origin,
			banRequest.ReportId,
			banRequest.BanType,
			&banSteam,
		); errBanSteam != nil {
			responseErr(ctx, http.StatusBadRequest, "Failed to parse options")
			return
		}
		if errBan := app.BanSteam(ctx, &banSteam); errBan != nil {
			logger.Error("Failed to ban steam profile",
				zap.Error(errBan), zap.Int64("target_id", banSteam.TargetId.Int64()))
			if errors.Is(errBan, store.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, "Duplicate ban")
				return
			}
			responseErr(ctx, http.StatusBadRequest, "Failed to perform ban")
			return
		}
		responseOK(ctx, http.StatusCreated, banSteam)
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
	return func(ctx *gin.Context) {
		var request authReq
		if errBind := ctx.BindJSON(&request); errBind != nil {
			logger.Error("Failed to decode auth request", zap.Error(errBind))
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		var server store.Server
		errGetServer := store.GetServerByName(ctx, request.ServerName, &server)
		if errGetServer != nil {
			logger.Error("Failed to find server auth by name",
				zap.String("name", request.ServerName), zap.Error(errGetServer))
			responseErr(ctx, http.StatusNotFound, nil)
			return
		}
		if server.Password != request.Key {
			responseErr(ctx, http.StatusForbidden, nil)
			logger.Error("Invalid server key used",
				zap.String("server", util.SanitizeLog(request.ServerName)))
			return
		}
		accessToken, errToken := newServerJWT(server.ServerID)
		if errToken != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to create new server access token", zap.Error(errToken))
			return
		}
		server.TokenCreatedOn = config.Now()
		if errSaveServer := store.SaveServer(ctx, &server); errSaveServer != nil {
			logger.Error("Failed to updated server token", zap.Error(errSaveServer))
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, authResp{Status: true, Token: accessToken})
		logger.Info("Server authenticated successfully", zap.String("server", server.ServerNameShort))
	}
}

func onAPIPostServerCheck() gin.HandlerFunc {
	type checkRequest struct {
		ClientID int         `json:"client_id"`
		SteamID  steamid.SID `json:"steam_id"`
		IP       net.IP      `json:"ip"`
		Name     string      `json:"name,omitempty"`
	}
	type checkResponse struct {
		ClientID        int             `json:"client_id"`
		SteamID         steamid.SID     `json:"steam_id"`
		BanType         store.BanType   `json:"ban_type"`
		PermissionLevel store.Privilege `json:"permission_level"`
		Msg             string          `json:"msg"`
	}
	return func(ctx *gin.Context) {
		var request checkRequest
		if errBind := ctx.BindJSON(&request); errBind != nil {
			responseErr(ctx, http.StatusInternalServerError, checkResponse{
				BanType: store.Unknown,
				Msg:     "Error determining state",
			})
			return
		}
		resp := checkResponse{
			ClientID: request.ClientID,
			SteamID:  request.SteamID,
			BanType:  store.Unknown,
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

		if app.IsSteamGroupBanned(steamID) {
			resp.BanType = store.Banned
			resp.Msg = "Group Banned"
			responseErr(ctx, http.StatusOK, resp)
			logger.Info("Player dropped", zap.String("drop_type", "group"),
				zap.Int64("sid64", steamID.Int64()))
			return
		}

		var person store.Person
		if errPerson := app.PersonBySID(responseCtx, steamID, &person); errPerson != nil {
			responseErr(ctx, http.StatusInternalServerError, checkResponse{
				BanType: store.Unknown,
				Msg:     "Error updating profile state",
			})
			return
		}
		resp.PermissionLevel = person.PermissionLevel
		if errAddHist := store.AddConnectionHistory(ctx, &store.PersonConnection{
			IPAddr:      request.IP,
			SteamId:     steamid.SIDToSID64(request.SteamID),
			PersonaName: request.Name,
			CreatedOn:   config.Now(),
			IPInfo:      store.PersonIPRecord{},
		}); errAddHist != nil {
			logger.Error("Failed to add conn history", zap.Error(errAddHist))
		}
		// Check IP first
		banNet, errGetBanNet := store.GetBanNetByAddress(responseCtx, request.IP)
		if errGetBanNet != nil {
			responseErr(ctx, http.StatusInternalServerError, checkResponse{
				BanType: store.Unknown,
				Msg:     "Error determining state",
			})
			logger.Error("Could not get bannedPerson net results", zap.Error(errGetBanNet))
			return
		}
		if len(banNet) > 0 {
			resp.BanType = store.Banned
			resp.Msg = fmt.Sprintf("Network banned (C: %d)", len(banNet))
			responseOK(ctx, http.StatusOK, resp)
			logger.Info("Player dropped", zap.String("drop_type", "cidr"),
				zap.Int64("sid64", steamID.Int64()))
			return
		}
		var asnRecord ip2location.ASNRecord
		errASN := store.GetASNRecordByIP(responseCtx, request.IP, &asnRecord)
		if errASN == nil {
			var asnBan store.BanASN
			if errASNBan := store.GetBanASN(responseCtx, int64(asnRecord.ASNum), &asnBan); errASNBan != nil {
				if !errors.Is(errASNBan, store.ErrNoResult) {
					logger.Error("Failed to fetch asn bannedPerson", zap.Error(errASNBan))
				}
			} else {
				resp.BanType = store.Banned
				resp.Msg = asnBan.Reason.String()
				responseOK(ctx, http.StatusOK, resp)
				logger.Info("Player dropped", zap.String("drop_type", "asn"),
					zap.Int64("sid64", steamID.Int64()))
				return
			}
		}
		bannedPerson := store.NewBannedPerson()
		if errGetBan := store.GetBanBySteamID(responseCtx, steamID, &bannedPerson, false); errGetBan != nil {
			if errGetBan == store.ErrNoResult {
				// No ban, exit early
				resp.BanType = store.OK
				responseOK(ctx, http.StatusOK, resp)
				return
			}
			resp.Msg = "Error determining state"
			responseErr(ctx, http.StatusInternalServerError, resp)
			return
		}
		resp.BanType = bannedPerson.Ban.BanType
		reason := ""
		if bannedPerson.Ban.Reason == store.Custom && bannedPerson.Ban.ReasonText != "" {
			reason = bannedPerson.Ban.ReasonText
		} else if bannedPerson.Ban.Reason == store.Custom && bannedPerson.Ban.ReasonText == "" {
			reason = "Banned"
		} else {
			reason = bannedPerson.Ban.Reason.String()
		}
		resp.Msg = fmt.Sprintf("Banned\nReason: %s\nAppeal: %s\nRemaining: %s", reason, bannedPerson.Ban.ToURL(),
			bannedPerson.Ban.ValidUntil.Sub(config.Now()).Round(time.Minute).String())
		responseOK(ctx, http.StatusOK, resp)
		if resp.BanType == store.NoComm {
			logger.Info("Player muted", zap.Int64("sid64", steamID.Int64()))
		} else if resp.BanType == store.Banned {
			logger.Info("Player dropped", zap.String("drop_type", "steam"),
				zap.Int64("sid64", steamID.Int64()))
		}
	}
}

//
//func (w *web) onAPIGetAnsibleHosts() gin.HandlerFunc {
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
//			web.logger.Error("Failed to fetch servers: %s", errGetServers)
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
func onAPIGetPrometheusHosts() gin.HandlerFunc {
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
		servers, errGetServers := store.GetServers(ctx, true)
		if errGetServers != nil {
			logger.Error("Failed to fetch servers", zap.Error(errGetServers))
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

func getDefaultFloat64(s string, def float64) float64 {
	if s != "" {
		l, errLat := strconv.ParseFloat(s, 64)
		if errLat != nil {
			return def
		}
		return l
	}
	return def
}

// onAPIGetServerStates returns the current known cached server state
func onAPIGetServerStates() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		lat := getDefaultFloat64(ctx.GetHeader("cf-iplatitude"), 41.7774)
		lon := getDefaultFloat64(ctx.GetHeader("cf-iplongitude"), 87.6160)
		//region := ctx.GetHeader("cf-region-code")
		ns := state.State()
		var ss []BaseServer
		for _, srv := range ns {
			if !srv.Enabled {
				continue
			}
			ss = append(ss, BaseServer{
				Host:       srv.Host,
				Port:       srv.Port,
				Name:       srv.Name,
				NameShort:  srv.NameShort,
				Region:     srv.Region,
				Players:    srv.PlayerCount,
				MaxPlayers: srv.MaxPlayers,
				Bots:       srv.Bots,
				Map:        srv.Map,
				GameTypes:  []string{},
				Latitude:   lat,
				Longitude:  lon,
				Distance:   distance(srv.Latitude, srv.Longitude, lat, lon),
			})
		}
		sort.SliceStable(ss, func(i, j int) bool {
			return ss[i].Distance < ss[j].Distance
		})
		responseOK(ctx, http.StatusOK, ss)
	}
}

func queryFilterFromContext(ctx *gin.Context) (store.QueryFilter, error) {
	var queryFilter store.QueryFilter
	if errBind := ctx.BindUri(&queryFilter); errBind != nil {
		return queryFilter, errBind
	}
	return queryFilter, nil
}

func onAPIGetPlayers() gin.HandlerFunc {
	return func(c *gin.Context) {
		queryFilter, errFilterFromContext := queryFilterFromContext(c)
		if errFilterFromContext != nil {
			responseErr(c, http.StatusBadRequest, nil)
			return
		}
		people, errGetPeople := store.GetPeople(c, queryFilter)
		if errGetPeople != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			return
		}
		responseOK(c, http.StatusOK, people)
	}
}

func onAPIGetResolveProfile() gin.HandlerFunc {
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
		var person store.Person
		if errPerson := app.PersonBySID(ctx, id, &person); errPerson != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, person)
	}
}

func onAPICurrentProfileNotifications() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userProfile := currentUserProfile(ctx)
		notifications, errNot := store.GetPersonNotifications(ctx, userProfile.SteamID)
		if errNot != nil {
			if errors.Is(errNot, store.ErrNoResult) {
				responseOK(ctx, http.StatusOK, []store.UserNotification{})
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, notifications)
	}
}

func onAPICurrentProfile() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userProfile := currentUserProfile(ctx)
		if !userProfile.SteamID.Valid() {
			logger.Error("Failed to load user profile",
				zap.Int64("sid64", userProfile.SteamID.Int64()),
				zap.String("name", userProfile.Name),
				zap.String("permission_level", userProfile.PermissionLevel.String()))
			responseErr(ctx, http.StatusForbidden, nil)
			return
		}
		responseOK(ctx, http.StatusOK, userProfile)
	}
}

func onAPIExportBansValveSteamId() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		bans, errBans := store.GetBansSteam(ctx, store.BansQueryFilter{
			PermanentOnly: true,
		})
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		var entries []string
		for _, ban := range bans {
			if ban.Ban.Deleted ||
				!ban.Ban.IsEnabled {
				continue
			}
			entries = append(entries, fmt.Sprintf("banid 0 %s", steamid.SID64ToSID(ban.Person.SteamID)))
		}
		ctx.Data(http.StatusOK, "text/plain", []byte(strings.Join(entries, "\n")))
	}
}
func onAPIExportBansValveIP() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		bans, errBans := store.GetBansNet(ctx)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		var entries []string
		for _, ban := range bans {
			if ban.Deleted ||
				!ban.IsEnabled {
				continue
			}
			entries = append(entries, fmt.Sprintf("addip 0 %s", ban.CIDR.IP.String()))
		}
		ctx.Data(http.StatusOK, "text/plain", []byte(strings.Join(entries, "\n")))
	}
}

func onAPIExportSourcemodSimpleAdmins() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		privilegedIds, errPrivilegedIds := store.GetSteamIdsAbove(ctx, store.PReserved)
		if errPrivilegedIds != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		players, errPlayers := store.GetPeopleBySteamID(ctx, privilegedIds)
		if errPlayers != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		sort.Slice(players, func(i, j int) bool {
			return players[i].PermissionLevel > players[j].PermissionLevel
		})
		bld := strings.Builder{}
		for _, player := range players {
			perms := ""
			switch player.PermissionLevel {
			case store.PAdmin:
				perms = "z"
			case store.PModerator:
				perms = "abcdefgjk"
			case store.PEditor:
				perms = "ak"
			case store.PReserved:
				perms = "a"
			}
			bld.WriteString(fmt.Sprintf("\"%s\" \"%s\"\n", steamid.SID64ToSID3(player.SteamID), perms))
		}
		ctx.String(http.StatusOK, bld.String())
	}
}

func onAPIExportBansTF2BD() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// TODO limit / make specialized query since this returns all results
		bans, errBans := store.GetBansSteam(ctx, store.BansQueryFilter{
			QueryFilter: store.QueryFilter{},
			SteamId:     0,
		})
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		var filtered []store.BannedPerson
		for _, ban := range bans {
			if ban.Ban.Reason != store.Cheating ||
				ban.Ban.Deleted ||
				!ban.Ban.IsEnabled {
				continue
			}
			filtered = append(filtered, ban)
		}
		out := thirdparty.TF2BDSchema{
			Schema: "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/schemas/v3/playerlist.schema.json",
			FileInfo: thirdparty.FileInfo{
				Authors:     []string{config.General.SiteName},
				Description: "Players permanently banned for cheating",
				Title:       fmt.Sprintf("%s Cheater List", config.General.SiteName),
				UpdateURL:   config.ExtURL("/export/bans/tf2bd"),
			},
			Players: []thirdparty.Players{},
		}
		for _, ban := range filtered {
			out.Players = append(out.Players, thirdparty.Players{
				Attributes: []string{"cheater"},
				Steamid:    ban.Ban.TargetId.Int64(),
				LastSeen: thirdparty.LastSeen{
					PlayerName: ban.Person.PersonaName,
					Time:       int(ban.Ban.UpdatedOn.Unix()),
				},
			})
		}
		ctx.JSON(http.StatusOK, out)
	}
}

func onAPIProfile() gin.HandlerFunc {
	type req struct {
		Query string `form:"query"`
	}
	type resp struct {
		Player  *store.Person            `json:"player"`
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
		person := store.NewPerson(sid)
		if errGetProfile := app.PersonBySID(requestCtx, sid, &person); errGetProfile != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		friendIDs, errFetchFriends := thirdparty.FetchFriends(requestCtx, person.SteamID)
		if errFetchFriends != nil {
			responseErr(ctx, http.StatusServiceUnavailable, "Could not fetch friends")
			return
		}
		// TODO add ctx to steamweb lib
		friends, errFetchSummaries := thirdparty.FetchSummaries(friendIDs)
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

func onAPIGetWordFilters() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		words, errGetFilters := store.GetFilters(ctx)
		if errGetFilters != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, words)
	}
}

func onAPIPostWordMatch() gin.HandlerFunc {
	type matchRequest struct {
		Query string
	}
	return func(ctx *gin.Context) {
		var req matchRequest
		if errBind := ctx.BindJSON(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			logger.Error("Failed to parse request", zap.Error(errBind))
			return
		}
		words, errGetFilters := store.GetFilters(ctx)
		if errGetFilters != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		var matches []store.Filter
		for _, filter := range words {
			if filter.Match(req.Query) {
				matches = append(matches, filter)
			}
		}
		responseOK(ctx, http.StatusOK, matches)
	}
}

func onAPIDeleteWordFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		wordId, wordIdErr := getInt64Param(ctx, "word_id")
		if wordIdErr != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var filter store.Filter
		if errGet := store.GetFilterByID(ctx, wordId, &filter); errGet != nil {
			if errors.Is(errGet, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		if errDrop := store.DropFilter(ctx, &filter); errDrop != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, nil)
	}
}

func onAPIPostWordFilter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var filter store.Filter
		if errBind := ctx.BindJSON(&filter); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			logger.Error("Failed to parse request", zap.Error(errBind))
			return
		}
		if filter.Pattern == "" {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		if filter.IsRegex {
			_, compErr := regexp.Compile(filter.Pattern)
			if compErr != nil {
				responseErr(ctx, http.StatusBadRequest, nil)
				return
			}
		}
		now := config.Now()
		if filter.FilterID > 0 {
			var existingFilter store.Filter
			if errGet := store.GetFilterByID(ctx, filter.FilterID, &existingFilter); errGet != nil {
				if errors.Is(errGet, store.ErrNoResult) {
					responseErr(ctx, http.StatusNotFound, nil)
					return
				}
				responseErr(ctx, http.StatusInternalServerError, nil)
				return
			}
			existingFilter.UpdatedOn = now
			existingFilter.Pattern = filter.Pattern
			existingFilter.IsRegex = filter.IsRegex
			existingFilter.IsEnabled = filter.IsEnabled
			if errSave := app.FilterAdd(ctx, &existingFilter); errSave != nil {
				responseErr(ctx, http.StatusInternalServerError, nil)
				return
			}
			filter = existingFilter
		} else {
			profile := currentUserProfile(ctx)
			newFilter := store.Filter{
				AuthorId:  profile.SteamID,
				Pattern:   filter.Pattern,
				CreatedOn: now,
				UpdatedOn: now,
				IsRegex:   filter.IsRegex,
				IsEnabled: filter.IsEnabled,
			}
			if errSave := app.FilterAdd(ctx, &newFilter); errSave != nil {
				responseErr(ctx, http.StatusInternalServerError, nil)
				return
			}
			filter = newFilter
		}
		responseOK(ctx, http.StatusOK, filter)
	}
}

func onAPIGetCompHist() gin.HandlerFunc {
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
		var hist thirdparty.CompHist
		if errFetch := thirdparty.FetchCompHist(ctx, sid, &hist); errFetch != nil {
			responseErr(ctx, http.StatusInternalServerError, "query failed")
			return
		}
		responseOK(ctx, http.StatusOK, hist)
	}
}

func onAPIGetStats() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var stats store.Stats
		if errGetStats := store.GetStats(ctx, &stats); errGetStats != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		stats.ServersAlive = 1
		responseOK(ctx, http.StatusOK, stats)
	}
}

func loadBanMeta(_ *store.BannedPerson) {

}

func onAPIGetBanByID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		curUser := currentUserProfile(ctx)
		banId, errId := getInt64Param(ctx, "ban_id")
		if errId != nil || banId == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		deletedOk := false
		fullValue, fullOk := ctx.GetQuery("deleted")
		if fullOk {
			deleted, deletedOkErr := strconv.ParseBool(fullValue)
			if deletedOkErr != nil {
				logger.Error("Failed to parse ban full query value", zap.Error(deletedOkErr))
			} else {
				deletedOk = deleted
			}
		}

		bannedPerson := store.NewBannedPerson()
		if errGetBan := store.GetBanByBanID(ctx, banId, &bannedPerson, deletedOk); errGetBan != nil {
			responseErr(ctx, http.StatusNotFound, nil)
			logger.Error("Failed to fetch bans", zap.Error(errGetBan))
			return
		}

		if !checkPrivilege(ctx, curUser, steamid.Collection{bannedPerson.Person.SteamID}, store.PModerator) {
			return
		}
		loadBanMeta(&bannedPerson)
		responseOK(ctx, http.StatusOK, bannedPerson)
	}
}

func onAPIGetAppeals() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var queryFilter store.QueryFilter
		if errBind := ctx.BindJSON(&queryFilter); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		bans, errBans := store.GetAppealsByActivity(ctx, queryFilter)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to fetch bans", zap.Error(errBans))
			return
		}
		responseOK(ctx, http.StatusOK, bans)
	}
}

func onAPIGetBansSteam() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var queryFilter store.BansQueryFilter
		if errBind := ctx.BindJSON(&queryFilter); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		bans, errBans := store.GetBansSteam(ctx, queryFilter)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to fetch bans", zap.Error(errBans))
			return
		}
		responseOK(ctx, http.StatusOK, bans)
	}
}

func onAPIGetBansCIDR() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var queryFilter store.BansQueryFilter
		if errBind := ctx.BindJSON(&queryFilter); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		// TODO filters
		bans, errBans := store.GetBansNet(ctx)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to fetch bans", zap.Error(errBans))
			return
		}
		responseOK(ctx, http.StatusOK, bans)
	}
}

func onAPIDeleteBansCIDR() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		netId, netIdErr := getInt64Param(ctx, "net_id")
		if netIdErr != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var req apiUnbanRequest
		if errBind := ctx.BindJSON(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, "Invalid request")
			return
		}
		var banCidr store.BanCIDR
		if errFetch := store.GetBanNetById(ctx, netId, &banCidr); errFetch != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		banCidr.UnbanReasonText = req.UnbanReasonText
		banCidr.Deleted = true
		if errSave := store.SaveBanNet(ctx, &banCidr); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to delete cidr ban", zap.Error(errSave))
			return
		}
		banCidr.NetID = 0
		responseOK(ctx, http.StatusOK, banCidr)
	}
}

func onAPIGetBansGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var queryFilter store.BansQueryFilter
		if errBind := ctx.BindJSON(&queryFilter); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		// TODO filters
		banGroups, errBans := store.GetBanGroups(ctx)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to fetch banGroups", zap.Error(errBans))
			return
		}
		responseOK(ctx, http.StatusOK, banGroups)
	}
}

func onAPIDeleteBansGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupId, groupIdErr := getInt64Param(ctx, "ban_group_id")
		if groupIdErr != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var req apiUnbanRequest
		if errBind := ctx.BindJSON(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, "Invalid request")
			return
		}
		var banGroup store.BanGroup
		if errFetch := store.GetBanGroupById(ctx, groupId, &banGroup); errFetch != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		banGroup.UnbanReasonText = req.UnbanReasonText
		banGroup.Deleted = true
		if errSave := store.SaveBanGroup(ctx, &banGroup); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to delete asn ban", zap.Error(errSave))
			return
		}
		banGroup.BanGroupId = 0
		responseOK(ctx, http.StatusOK, banGroup)
	}
}

func onAPIGetBansASN() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var queryFilter store.BansQueryFilter
		if errBind := ctx.BindJSON(&queryFilter); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		// TODO filters
		banASN, errBans := store.GetBansASN(ctx)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to fetch banASN", zap.Error(errBans))
			return
		}
		responseOK(ctx, http.StatusOK, banASN)
	}
}

func onAPIDeleteBansASN() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		asnId, asnIdErr := getInt64Param(ctx, "asn_id")
		if asnIdErr != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var req apiUnbanRequest
		if errBind := ctx.BindJSON(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, "Invalid request")
			return
		}
		var banAsn store.BanASN
		if errFetch := store.GetBanASN(ctx, asnId, &banAsn); errFetch != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		banAsn.UnbanReasonText = req.UnbanReasonText
		banAsn.Deleted = true
		if errSave := store.SaveBanASN(ctx, &banAsn); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to delete asn ban", zap.Error(errSave))
			return
		}
		banAsn.BanASNId = 0
		responseOK(ctx, http.StatusOK, banAsn)
	}
}

func onAPIGetServers() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		servers, errServers := store.GetServers(ctx, true)
		if errServers != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, servers)
	}
}

type serverUpdateRequest struct {
	Name          string  `json:"server_name"`
	NameShort     string  `json:"server_name_short"`
	Host          string  `json:"host"`
	Port          int     `json:"port"`
	ReservedSlots int     `json:"reserved_slots"`
	RCON          string  `json:"rcon"`
	Lat           float32 `json:"lat"`
	Lon           float32 `json:"lon"`
	CC            string  `json:"cc"`
	DefaultMap    string  `json:"default_map"`
	Region        string  `json:"region"`
	IsEnabled     bool    `json:"is_enabled"`
}

func onAPIPostServerUpdate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		serverId, idErr := getIntParam(ctx, "server_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var server store.Server
		if errServer := store.GetServer(ctx, serverId, &server); errServer != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}

		var serverReq serverUpdateRequest
		if errBind := ctx.BindJSON(&serverReq); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			logger.Error("Failed to parse request to update server", zap.Error(errBind))
			return
		}
		server.ServerNameShort = serverReq.NameShort
		server.ServerNameLong = serverReq.Name
		server.Address = serverReq.Host
		server.Port = serverReq.Port
		server.ReservedSlots = serverReq.ReservedSlots
		server.RCON = serverReq.RCON
		server.Latitude = serverReq.Lat
		server.Longitude = serverReq.Lon
		server.CC = serverReq.CC
		server.Region = serverReq.Region
		server.IsEnabled = serverReq.IsEnabled

		if errSave := store.SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to update server", zap.Error(errSave))
			return
		}
		responseOK(ctx, http.StatusOK, server)
		logger.Info("Server config updated",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ServerNameShort))
	}
}

func onAPIPostServerDelete() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		serverId, idErr := getIntParam(ctx, "server_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var server store.Server
		if errServer := store.GetServer(ctx, serverId, &server); errServer != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		server.Deleted = true
		if errSave := store.SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to delete server", zap.Error(errSave))
			return
		}
		responseOK(ctx, http.StatusOK, server)
		logger.Info("Server config deleted",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ServerNameShort))
	}
}

func onAPIPostServer() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var serverReq serverUpdateRequest
		if errBind := ctx.BindJSON(&serverReq); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			logger.Error("Failed to parse request for new server", zap.Error(errBind))
			return
		}
		server := store.NewServer(serverReq.NameShort, serverReq.Host, serverReq.Port)
		server.ServerNameLong = serverReq.Name
		server.ReservedSlots = serverReq.ReservedSlots
		server.RCON = serverReq.RCON
		server.Latitude = serverReq.Lat
		server.Longitude = serverReq.Lon
		server.CC = serverReq.CC
		server.Region = serverReq.Region
		server.IsEnabled = serverReq.IsEnabled
		if errSave := store.SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to save new server", zap.Error(errSave))
			return
		}
		responseOK(ctx, http.StatusOK, server)
		logger.Info("Server config created",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ServerNameShort))

	}
}

func onAPIPostReportCreate() gin.HandlerFunc {
	type createReport struct {
		SourceId    store.StringSID `json:"source_id"`
		TargetId    store.StringSID `json:"target_id"`
		Description string          `json:"description"`
		Reason      store.Reason    `json:"reason"`
		ReasonText  string          `json:"reason_text"`
		DemoName    string          `json:"demo_name"`
		DemoTick    int             `json:"demo_tick"`
	}
	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)
		var newReport createReport
		if errBind := ctx.BindJSON(&newReport); errBind != nil {
			responseErrUser(ctx, http.StatusBadRequest, nil, "Invalid request")
			logger.Error("Failed to bind report", zap.Error(errBind))
			return
		}
		// Server initiated requests will have a sourceId set by the server
		// Web based reports the source should not be set, the reporter will be taken from the
		// current session information instead
		if newReport.SourceId == "" {
			newReport.SourceId = store.StringSID(currentUser.SteamID.String())
		}
		sourceId, errSourceId := newReport.SourceId.SID64()
		if errSourceId != nil {
			responseErrUser(ctx, http.StatusBadRequest, nil, "Failed to resolve steam id")
			logger.Error("Invalid steam_id", zap.Error(errSourceId))
			return
		}
		targetId, errTargetId := newReport.TargetId.SID64()
		if errTargetId != nil {
			responseErrUser(ctx, http.StatusBadRequest, nil, "Failed to resolve steam id")
			logger.Error("Invalid target_id", zap.Error(errTargetId))
			return
		}
		if sourceId == targetId {
			responseErrUser(ctx, http.StatusForbidden, nil, "Cannot report yourself")
			return
		}
		var personSource store.Person
		if errCreatePerson := app.PersonBySID(ctx, sourceId, &personSource); errCreatePerson != nil {
			responseErrUser(ctx, http.StatusInternalServerError, nil, "Internal error")
			logger.Error("Could not load player profile", zap.Error(errCreatePerson))
			return
		}
		var personTarget store.Person
		if errCreatePerson := app.PersonBySID(ctx, targetId, &personTarget); errCreatePerson != nil {
			responseErrUser(ctx, http.StatusInternalServerError, nil, "Internal error")
			logger.Error("Could not load player profile", zap.Error(errCreatePerson))
			return
		}

		// Ensure the user doesn't already have an open report against the user
		var existing store.Report
		if errReports := store.GetReportBySteamId(ctx, currentUser.SteamID, targetId, &existing); errReports != nil {
			if !errors.Is(errReports, store.ErrNoResult) {
				logger.Error("Failed to query reports by steam id", zap.Error(errReports))
				responseErr(ctx, http.StatusInternalServerError, nil)
				return
			}
		}
		if existing.ReportId > 0 {
			responseErrUser(ctx, http.StatusConflict, nil,
				"Must resolve existing report for user before creating another")
			return
		}

		// TODO encapsulate all operations in single tx
		report := store.NewReport()
		report.SourceId = sourceId
		report.ReportStatus = store.Opened
		report.Description = newReport.Description
		report.TargetId = targetId
		report.Reason = newReport.Reason
		report.ReasonText = newReport.ReasonText
		parts := strings.Split(newReport.DemoName, "/")
		report.DemoName = parts[len(parts)-1]
		report.DemoTick = newReport.DemoTick
		if errReportSave := store.SaveReport(ctx, &report); errReportSave != nil {
			responseErrUser(ctx, http.StatusInternalServerError, nil, "Failed to save report")
			logger.Error("Failed to save report", zap.Error(errReportSave))
			return
		}
		responseOK(ctx, http.StatusCreated, report)

		embed := discord.RespOk(nil, "New user report created")
		embed.Description = report.Description
		embed.URL = report.ToURL()
		discord.AddAuthorProfile(embed,
			currentUser.SteamID, currentUser.Name, currentUser.ToURL())
		name := personSource.PersonaName
		if name == "" {
			name = report.TargetId.String()
		}
		discord.AddField(embed, "Subject", name)
		discord.AddField(embed, "Reason", report.Reason.String())
		if report.ReasonText != "" {
			discord.AddField(embed, "Custom Reason", report.ReasonText)
		}
		if report.DemoName != "" {
			discord.AddField(embed, "Demo", config.ExtURL("/demos/name/%s", report.DemoName))
			discord.AddField(embed, "Demo Tick", fmt.Sprintf("%d", report.DemoTick))
		}
		discord.AddFieldsSteamID(embed, report.TargetId)
		discord.AddLink(embed, report)
		discord.SendPayload(discord.Payload{
			ChannelId: config.Discord.ReportLogChannelId,
			Embed:     embed,
		})
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

func onAPIPostReportMessage() gin.HandlerFunc {
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
		var report store.Report
		if errReport := store.GetReport(ctx, reportId, &report); errReport != nil {
			if store.Err(errReport) == store.ErrNoResult {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusBadRequest, nil)
			logger.Error("Failed to load report", zap.Error(errReport))
			return
		}
		person := currentUserProfile(ctx)
		msg := store.NewUserMessage(reportId, person.SteamID, request.Message)
		if errSave := store.SaveReportMessage(ctx, &msg); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to save report message", zap.Error(errSave))
			return
		}
		responseOK(ctx, http.StatusCreated, msg)

		embed := &discordgo.MessageEmbed{
			Title:       "New report message posted",
			Description: msg.Message,
		}
		discord.AddField(embed, "Author", report.SourceId.String())
		discord.AddLink(embed, report)
		discord.SendPayload(discord.Payload{
			ChannelId: config.Discord.ReportLogChannelId,
			Embed:     embed})
	}
}

func onAPIEditReportMessage() gin.HandlerFunc {
	type editMessage struct {
		Message string `json:"body_md"`
	}
	return func(ctx *gin.Context) {
		reportMessageId, errId := getInt64Param(ctx, "report_message_id")
		if errId != nil || reportMessageId == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}

		var existing store.UserMessage
		if errExist := store.GetReportMessageById(ctx, reportMessageId, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		curUser := currentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorId}, store.PModerator) {
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
		existing.Message = message.Message
		if errSave := store.SaveReportMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to save report message", zap.Error(errSave))
			return
		}
		responseOK(ctx, http.StatusCreated, message)

		embed := &discordgo.MessageEmbed{
			Title:       "New report message edited",
			Description: message.Message,
		}
		discord.AddField(embed, "Old Message", existing.Message)
		discord.AddField(embed, "Report Link", config.ExtURL("/report/%d", existing.ParentId))
		discord.AddField(embed, "Author", curUser.SteamID.String())
		embed.Image = &discordgo.MessageEmbedImage{URL: curUser.AvatarFull}
		discord.SendPayload(discord.Payload{
			ChannelId: config.Discord.ModLogChannelId,
			Embed:     embed})
	}
}

func onAPIDeleteReportMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportMessageId, errId := getInt64Param(ctx, "report_message_id")
		if errId != nil || reportMessageId == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var existing store.UserMessage
		if errExist := store.GetReportMessageById(ctx, reportMessageId, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		curUser := currentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorId}, store.PModerator) {
			return
		}
		existing.Deleted = true
		if errSave := store.SaveReportMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to save report message", zap.Error(errSave))
			return
		}
		responseOK(ctx, http.StatusNoContent, nil)

		embed := &discordgo.MessageEmbed{
			Title:       "User report message deleted",
			Description: existing.Message,
		}
		discord.AddField(embed, "Author", curUser.SteamID.String())
		discord.SendPayload(discord.Payload{
			ChannelId: config.Discord.ModLogChannelId,
			Embed:     embed})

	}
}

type AuthorMessage struct {
	Author  store.Person      `json:"author"`
	Message store.UserMessage `json:"message"`
}

func onAPISetReportStatus() gin.HandlerFunc {
	type stateUpdateReq struct {
		Status store.ReportStatus `json:"status"`
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
		var report store.Report
		if errGet := store.GetReport(c, reportId, &report); errGet != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			logger.Error("Failed to get report to set state", zap.Error(errGet))
			return
		}
		if report.ReportStatus == newStatus.Status {
			responseOK(c, http.StatusConflict, nil)
			return
		}
		original := report.ReportStatus
		report.ReportStatus = newStatus.Status
		if errSave := store.SaveReport(c, &report); errSave != nil {
			responseErr(c, http.StatusInternalServerError, nil)
			logger.Error("Failed to save report state", zap.Error(errSave))
			return
		}
		responseOK(c, http.StatusAccepted, nil)
		logger.Info("Report status changed",
			zap.Int64("report_id", report.ReportId),
			zap.String("from_status", original.String()),
			zap.String("to_status", report.ReportStatus.String()))
		panic("xxx")
		//discord.SendDiscord(model.NotificationPayload{
		//	Sids:     steamid.Collection{report.SourceId},
		//	Severity: store.SeverityInfo,
		//	Message:  "Report status updated",
		//	Link:     report.ToURL(),
		//})
	}
}

func onAPIGetReportMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportId, errParam := getInt64Param(ctx, "report_id")
		if errParam != nil {
			responseErr(ctx, http.StatusNotFound, nil)
			return
		}
		var report store.Report
		if errGetReport := store.GetReport(ctx, reportId, &report); errGetReport != nil {
			responseErr(ctx, http.StatusNotFound, nil)
			return
		}

		if !checkPrivilege(ctx, currentUserProfile(ctx), steamid.Collection{report.SourceId, report.TargetId}, store.PModerator) {
			return
		}

		reportMessages, errGetReportMessages := store.GetReportMessages(ctx, reportId)
		if errGetReportMessages != nil {
			responseErr(ctx, http.StatusNotFound, nil)
			return
		}
		var ids steamid.Collection
		for _, msg := range reportMessages {
			ids = append(ids, msg.AuthorId)
		}
		authors, authorsErr := store.GetPeopleBySteamID(ctx, ids)
		if authorsErr != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
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
		responseOK(ctx, http.StatusOK, authorMessages)
	}
}

type reportWithAuthor struct {
	Author  store.Person `json:"author"`
	Subject store.Person `json:"subject"`
	Report  store.Report `json:"report"`
}

func onAPIGetReports() gin.HandlerFunc {
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
		reports, errReports := store.GetReports(ctx, opts)
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
			authorIds = append(authorIds, report.SourceId)
		}
		authors, errAuthors := store.GetPeopleBySteamID(ctx, fp.Uniq[steamid.SID64](authorIds))
		if errAuthors != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		authorMap := authors.AsMap()

		var subjectIds steamid.Collection
		for _, report := range reports {
			subjectIds = append(subjectIds, report.TargetId)
		}
		subjects, errSubjects := store.GetPeopleBySteamID(ctx, fp.Uniq[steamid.SID64](subjectIds))
		if errSubjects != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		subjectMap := subjects.AsMap()

		for _, report := range reports {
			userReports = append(userReports, reportWithAuthor{
				Author:  authorMap[report.SourceId],
				Report:  report,
				Subject: subjectMap[report.TargetId],
			})
		}
		sort.SliceStable(userReports, func(i, j int) bool {
			return userReports[i].Report.ReportId > userReports[j].Report.ReportId
		})
		responseOK(ctx, http.StatusOK, userReports)
	}
}

func onAPIGetReport() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportId, errParam := getInt64Param(ctx, "report_id")
		if errParam != nil {
			responseErr(ctx, http.StatusNotFound, nil)
			return
		}
		var report reportWithAuthor
		if errReport := store.GetReport(ctx, reportId, &report.Report); errReport != nil {
			if store.Err(errReport) == store.ErrNoResult {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusBadRequest, nil)
			logger.Error("Failed to load report", zap.Error(errReport))
			return
		}

		if !checkPrivilege(ctx, currentUserProfile(ctx), steamid.Collection{report.Report.SourceId}, store.PModerator) {
			responseErr(ctx, http.StatusUnauthorized, nil)
			return
		}

		if errAuthor := app.PersonBySID(ctx, report.Report.SourceId, &report.Author); errAuthor != nil {
			if store.Err(errAuthor) == store.ErrNoResult {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusBadRequest, nil)
			logger.Error("Failed to load report author", zap.Error(errAuthor))
			return
		}
		if errSubject := app.PersonBySID(ctx, report.Report.TargetId, &report.Subject); errSubject != nil {
			if store.Err(errSubject) == store.ErrNoResult {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusBadRequest, nil)
			logger.Error("Failed to load report subject", zap.Error(errSubject))
			return
		}

		responseOK(ctx, http.StatusOK, report)
	}
}

func onAPIGetNewsLatest() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := store.GetNewsLatest(ctx, 50, false)
		if errGetNewsLatest != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, newsLatest)
	}
}

func onAPIGetNewsAll() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := store.GetNewsLatest(ctx, 100, true)
		if errGetNewsLatest != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, newsLatest)
	}
}

func onAPIPostNewsCreate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var entry store.NewsEntry
		if errBind := ctx.BindJSON(&entry); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		if errSave := store.SaveNewsArticle(ctx, &entry); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusCreated, entry)
		go discord.SendPayload(discord.Payload{
			ChannelId: config.Discord.ModLogChannelId,
			Embed: &discordgo.MessageEmbed{
				Title:       "News Created",
				Description: fmt.Sprintf("News Posted: %s", entry.Title)},
		})
	}
}

func onAPIPostNewsUpdate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsId, errId := getIntParam(ctx, "news_id")
		if errId != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var entry store.NewsEntry
		if errGet := store.GetNewsById(ctx, newsId, &entry); errGet != nil {
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
		if errSave := store.SaveNewsArticle(ctx, &entry); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusAccepted, entry)
		discord.SendPayload(discord.Payload{
			ChannelId: config.Discord.ModLogChannelId,
			Embed: &discordgo.MessageEmbed{
				Title:       "News Updated",
				Description: fmt.Sprintf("News Updated: %s", entry.Title)},
		})

	}
}

func onAPISaveMedia() gin.HandlerFunc {
	type UserUploadedFile struct {
		Content string `json:"content"`
		Name    string `json:"name"`
		Mime    string `json:"mime"`
		Size    int64  `json:"size"`
	}
	return func(ctx *gin.Context) {
		var upload UserUploadedFile
		if errBind := ctx.BindJSON(&upload); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		content, decodeErr := base64.StdEncoding.DecodeString(upload.Content)
		if decodeErr != nil {
			responseErr(ctx, http.StatusUnprocessableEntity, nil)
			return
		}
		media, errMedia := store.NewMedia(currentUserProfile(ctx).SteamID, upload.Name, upload.Mime, content)
		if errMedia != nil {
			responseErrUser(ctx, http.StatusBadRequest, nil, "Invalid media")
			logger.Error("Invalid media uploaded", zap.Error(errMedia))
		}
		if !fp.Contains(store.MediaSafeMimeTypesImages, media.MimeType) {
			responseErrUser(ctx, http.StatusBadRequest, nil, "Invalid image format")
			logger.Error("User tried uploading image with forbidden mimetype",
				zap.String("mime", media.MimeType), zap.String("name", media.Name))
			return
		}
		if errSave := store.SaveMedia(ctx, &media); errSave != nil {
			logger.Error("Failed to save wiki media", zap.Error(errSave))
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

func onAPIGetWikiSlug() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slug := strings.ToLower(ctx.Param("slug"))
		if slug[0] == '/' {
			slug = slug[1:]
		}
		var page wiki.Page
		if errGetWikiSlug := store.GetWikiPageBySlug(ctx, slug, &page); errGetWikiSlug != nil {
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

func onGetMediaById() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mediaId, idErr := getIntParam(ctx, "media_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var media store.Media
		if errMedia := store.GetMediaById(ctx, mediaId, &media); errMedia != nil {
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

func onAPISaveWikiSlug() gin.HandlerFunc {
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
		if errGetWikiSlug := store.GetWikiPageBySlug(ctx, request.Slug, &page); errGetWikiSlug != nil {
			if errors.Is(errGetWikiSlug, store.ErrNoResult) {
				page.CreatedOn = config.Now()
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
		if errSave := store.SaveWikiPage(ctx, &page); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusCreated, page)
	}
}

func onAPIGetMatches() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var opts store.MatchesQueryOpts
		if errBind := ctx.BindJSON(&opts); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		matches, matchesErr := store.Matches(ctx, opts)
		if matchesErr != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, matches)
	}
}

func onAPIGetMatch() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		matchId, errId := getIntParam(ctx, "match_id")
		if errId != nil {
			logger.Error("Invalid match_id value", zap.Error(errId))
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		match, errMatch := store.MatchGetById(ctx, matchId)
		if errMatch != nil {
			if errors.Is(errMatch, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		responseOK(ctx, http.StatusOK, match)
	}
}

func onAPIGetPersonConnections() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamId, errId := getSID64Param(ctx, "steam_id")
		if errId != nil {
			logger.Error("Invalid steam_id value", zap.Error(errId))
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		// TODO paging
		ipHist, errIpHist := store.GetPersonIPHistory(ctx, steamId, 1000)
		if errIpHist != nil && !errors.Is(errIpHist, store.ErrNoResult) {
			logger.Error("Failed to query connection history",
				zap.Error(errIpHist), zap.Int64("sid64", steamId.Int64()))
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		if ipHist == nil {
			ipHist = store.PersonConnections{}
		}
		responseOK(ctx, http.StatusOK, ipHist)
	}
}

func onAPIQueryMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var query store.ChatHistoryQueryFilter
		if !bind(ctx, &query) {
			return
		}
		if query.Limit <= 0 || query.Limit > 1000 {
			query.Limit = 1000
		}
		// TODO paging
		chat, errChat := store.QueryChatHistory(ctx, query)
		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			logger.Error("Failed to query chat history",
				zap.Error(errChat), zap.String("sid", query.SteamId))
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		if chat == nil {
			chat = store.PersonMessages{}
		}
		responseOK(ctx, http.StatusOK, chat)
	}
}

func onAPIGetMessageContext() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		messageId, errId := getInt64Param(ctx, "person_message_id")
		if errId != nil {
			logger.Error("Invalid steam_id value", zap.Error(errId))
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var message store.PersonMessage
		if errMsg := store.GetPersonMessageById(ctx, messageId, &message); errMsg != nil {
			if errors.Is(errMsg, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		after := message.CreatedOn.Add(-time.Hour)
		before := message.CreatedOn.Add(time.Hour)
		// TODO paging
		chat, errChat := store.QueryChatHistory(ctx, store.ChatHistoryQueryFilter{
			ServerId:    message.ServerId,
			SentAfter:   &after,
			SentBefore:  &before,
			QueryFilter: store.QueryFilter{}})
		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			logger.Error("Failed to query chat history",
				zap.Error(errChat), zap.Int64("person_message_id", messageId))
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		if chat == nil {
			chat = store.PersonMessages{}
		}
		responseOK(ctx, http.StatusOK, chat)
	}
}

func onAPIGetPersonMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamId, errId := getSID64Param(ctx, "steam_id")
		if errId != nil {
			logger.Error("Invalid steam_id value", zap.Error(errId))
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		// TODO paging
		chat, errChat := store.QueryChatHistory(ctx, store.ChatHistoryQueryFilter{
			SteamId: steamId.String(),
			QueryFilter: store.QueryFilter{
				Limit: 1000,
			}})
		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			logger.Error("Failed to query chat history",
				zap.Error(errChat), zap.Int64("sid64", steamId.Int64()))
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		if chat == nil {
			chat = store.PersonMessages{}
		}
		responseOK(ctx, http.StatusOK, chat)
	}
}

type AuthorBanMessage struct {
	Author  store.Person      `json:"author"`
	Message store.UserMessage `json:"message"`
}

func onAPIGetBanMessages() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banId, errParam := getInt64Param(ctx, "ban_id")
		if errParam != nil {
			responseErr(ctx, http.StatusNotFound, nil)
			return
		}
		banPerson := store.NewBannedPerson()
		if errGetBan := store.GetBanByBanID(ctx, banId, &banPerson, true); errGetBan != nil {
			responseErr(ctx, http.StatusNotFound, nil)
			return
		}
		if !checkPrivilege(ctx, currentUserProfile(ctx), steamid.Collection{banPerson.Ban.TargetId, banPerson.Ban.SourceId}, store.PModerator) {
			return
		}
		banMessages, errGetBanMessages := store.GetBanMessages(ctx, banId)
		if errGetBanMessages != nil {
			responseErr(ctx, http.StatusNotFound, nil)
			return
		}
		var ids steamid.Collection
		for _, msg := range banMessages {
			ids = append(ids, msg.AuthorId)
		}
		authors, authorsErr := store.GetPeopleBySteamID(ctx, ids)
		if authorsErr != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
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
		responseOK(ctx, http.StatusOK, authorMessages)
	}
}

func onAPIDeleteBanMessage() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banMessageId, errId := getIntParam(ctx, "ban_message_id")
		if errId != nil || banMessageId == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var existing store.UserMessage
		if errExist := store.GetBanMessageById(ctx, banMessageId, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		curUser := currentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorId}, store.PModerator) {
			return
		}
		existing.Deleted = true
		if errSave := store.SaveBanMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to save appeal message", zap.Error(errSave))
			return
		}
		responseOK(ctx, http.StatusNoContent, nil)

		embed := &discordgo.MessageEmbed{
			Title:       "User appeal message deleted",
			Description: existing.Message,
		}
		discord.AddField(embed, "Author", curUser.SteamID.String())
		discord.SendPayload(discord.Payload{
			ChannelId: config.Discord.ReportLogChannelId,
			Embed:     embed})

	}
}

func onAPIPostBanMessage() gin.HandlerFunc {
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
		bp := store.NewBannedPerson()
		if errReport := store.GetBanByBanID(ctx, banId, &bp, true); errReport != nil {
			if store.Err(errReport) == store.ErrNoResult {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusBadRequest, nil)
			logger.Error("Failed to load ban", zap.Error(errReport))
			return
		}
		userProfile := currentUserProfile(ctx)
		if bp.Ban.AppealState != store.Open && userProfile.PermissionLevel < store.PModerator {
			responseErr(ctx, http.StatusForbidden, nil)
			logger.Warn("User tried to bypass posting restriction",
				zap.Int64("ban_id", bp.Ban.BanID), zap.Int64("steam_id", bp.Person.SteamID.Int64()))
			return
		}
		msg := store.NewUserMessage(banId, userProfile.SteamID, request.Message)
		if errSave := store.SaveBanMessage(ctx, &msg); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to save ban appeal message", zap.Error(errSave))
			return
		}
		responseOK(ctx, http.StatusCreated, msg)

		embed := &discordgo.MessageEmbed{
			Title:       "New ban appeal message posted",
			Description: msg.Message,
			Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: userProfile.AvatarFull},
			Color:       discord.DefaultLevelColors.Info,
			URL:         config.ExtURL("/ban/%d", banId),
		}
		discord.AddAuthorProfile(embed,
			userProfile.SteamID, userProfile.Name, userProfile.ToURL())
		discord.SendPayload(discord.Payload{
			ChannelId: config.Discord.ReportLogChannelId,
			Embed:     embed})

	}
}

func onAPIEditBanMessage() gin.HandlerFunc {
	type editMessage struct {
		Message string `json:"body_md"`
	}
	return func(ctx *gin.Context) {
		reportMessageId, errId := getIntParam(ctx, "ban_message_id")
		if errId != nil || reportMessageId == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		var existing store.UserMessage
		if errExist := store.GetBanMessageById(ctx, reportMessageId, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)
				return
			}
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		curUser := currentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorId}, store.PModerator) {
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
		if errSave := store.SaveBanMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			logger.Error("Failed to save ban appeal message", zap.Error(errSave))
			return
		}
		responseOK(ctx, http.StatusCreated, message)

		discord.AddField(embed, "Author", curUser.SteamID.String())
		discord.SendPayload(discord.Payload{
			ChannelId: config.Discord.ReportLogChannelId,
			Embed:     embed})

	}
}

type BaseServer struct {
	ServerId    int      `json:"server_id"`
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	Name        string   `json:"name"`
	NameShort   string   `json:"name_short"`
	Region      string   `json:"region"`
	CountryCode string   `json:"cc"`
	Players     int      `json:"players"`
	MaxPlayers  int      `json:"max_players"`
	Bots        int      `json:"bots"`
	Map         string   `json:"map"`
	GameTypes   []string `json:"game_types"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	Distance    float64  `json:"distance"`
}

var distance = func(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	radianLat1 := math.Pi * lat1 / 180
	radianLat2 := math.Pi * lat2 / 180
	theta := lng1 - lng2
	radianTheta := math.Pi * theta / 180
	dist := math.Sin(radianLat1)*math.Sin(radianLat2) + math.Cos(radianLat1)*math.Cos(radianLat2)*math.Cos(radianTheta)
	if dist > 1 {
		dist = 1
	}
	dist = math.Acos(dist)
	dist = dist * 180 / math.Pi
	dist = dist * 60 * 1.1515
	dist = dist * 1.609344 // convert to km
	return dist
}

func onAPIPostServerQuery() gin.HandlerFunc {
	type masterQueryRequest struct {
		// ctf,payload,cp,mvm,pd,passtime,mannpower,koth
		GameTypes  []string  `json:"game_types,omitempty"`
		AppId      int64     `json:"app_id,omitempty"`
		Maps       []string  `json:"maps,omitempty"`
		PlayersMin int       `json:"players_min,omitempty"`
		PlayersMax int       `json:"players_max,omitempty"`
		NotFull    bool      `json:"not_full,omitempty"`
		Location   []float64 `json:"location,omitempty"`
		Name       string    `json:"name,omitempty"`
		HasBots    bool      `json:"has_bots,omitempty"`
	}

	var filterGameTypes = func(servers []state.ServerLocation, gameTypes []string) []state.ServerLocation {
		var valid []state.ServerLocation
		for _, server := range servers {
			serverTypes := strings.Split(server.Gametype, ",")
			for _, gt := range gameTypes {
				if fp.Contains(serverTypes, gt) {
					valid = append(valid, server)
					break
				}
			}
		}
		return valid
	}

	var filterMaps = func(servers []state.ServerLocation, mapNames []string) []state.ServerLocation {
		var valid []state.ServerLocation
		for _, server := range servers {
			for _, mapName := range mapNames {
				if glob.Glob(mapName, server.Map) {
					valid = append(valid, server)
					break
				}
			}
		}
		return valid
	}

	var filterPlayersMin = func(servers []state.ServerLocation, minimum int) []state.ServerLocation {
		var valid []state.ServerLocation
		for _, server := range servers {
			if server.Players >= minimum {
				valid = append(valid, server)
				break
			}
		}
		return valid
	}

	var filterPlayersMax = func(servers []state.ServerLocation, maximum int) []state.ServerLocation {
		var valid []state.ServerLocation
		for _, server := range servers {
			if server.Players <= maximum {
				valid = append(valid, server)
				break
			}
		}
		return valid
	}

	return func(ctx *gin.Context) {
		var record ip2location.LocationRecord
		if errLoc := store.GetLocationRecord(ctx, net.ParseIP("68.144.74.48"), &record); errLoc != nil {
			responseErr(ctx, http.StatusForbidden, nil)
			return
		}
		var req masterQueryRequest
		if errBind := ctx.BindJSON(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			return
		}
		filtered := state.MasterServerList()
		if len(req.GameTypes) > 0 {
			filtered = filterGameTypes(filtered, req.GameTypes)
		}
		if len(req.Maps) > 0 {
			filtered = filterMaps(filtered, req.GameTypes)
		}
		if req.PlayersMin > 0 {
			filtered = filterPlayersMin(filtered, req.PlayersMin)
		}
		if req.PlayersMax > 0 {
			filtered = filterPlayersMax(filtered, req.PlayersMax)
		}
		var slim []BaseServer
		for _, server := range filtered {
			dist := distance(server.Latitude, server.Longitude, record.LatLong.Latitude, record.LatLong.Longitude)
			if dist <= 0 || dist > 5000 {
				continue
			}
			slim = append(slim, BaseServer{
				Host: server.Addr,
				Port: server.Gameport,
				Name: server.Name,
				//Region:     server.Region,
				Players:    server.Players,
				MaxPlayers: server.MaxPlayers,
				Bots:       server.Bots,
				Map:        server.Map,
				GameTypes:  strings.Split(server.Gametype, ","),
				Latitude:   server.Latitude,
				Longitude:  server.Longitude,
				Distance:   dist,
			})
		}
		sort.SliceStable(slim, func(i, j int) bool {
			return slim[i].Distance < slim[j].Distance
		})
		responseOK(ctx, http.StatusOK, slim)
	}
}

func onAPIGetTF2Stats() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		source, sourceFound := ctx.GetQuery("source")
		if !sourceFound {
			responseErr(ctx, http.StatusInternalServerError, []state.GlobalTF2StatsSnapshot{})
			return
		}
		durationStr, errDuration := ctx.GetQuery("duration")
		if !errDuration {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		intValue, errParse := strconv.ParseInt(durationStr, 10, 64)
		if errParse != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			return
		}
		duration := store.StatDuration(intValue)
		switch source {
		case "local":
			localStats, errGetStats := store.GetLocalTF2Stats(ctx, duration)
			if errGetStats != nil {
				responseErr(ctx, http.StatusInternalServerError, nil)
				return
			}
			responseOK(ctx, http.StatusOK, fp.Reverse(localStats))
		//case "global":
		//	gStats, errGetStats := web.app.Store().GetGlobalTF2Stats(ctx, duration)
		//	if errGetStats != nil {
		//		responseErr(ctx, http.StatusInternalServerError, nil)
		//		return
		//	}
		//	responseOK(ctx, http.StatusOK, fp.Reverse(gStats))
		default:
			responseErr(ctx, http.StatusBadRequest, nil)
		}
	}
}

func onAPIGetPatreonCampaigns() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		responseOK(ctx, http.StatusOK, app.PatreonCampaigns())
	}
}

func onAPIGetPatreonPledges() gin.HandlerFunc {
	// Only leak specific details
	//type basicPledge struct {
	//	Name      string
	//	Amount    int
	//	CreatedAt time.Time
	//}
	return func(ctx *gin.Context) {

		//var basic []basicPledge
		//for _, p := range pledges {
		//	t0 := config.Now()
		//	if p.Attributes.CreatedAt.Valid {
		//		t0 = p.Attributes.CreatedAt.Time.UTC()
		//	}
		//	basic = append(basic, basicPledge{
		//		Name:      users[p.Relationships.Patron.Data.ID].Attributes.FullName,
		//		Amount:    p.Attributes.AmountCents,
		//		CreatedAt: t0,
		//	})
		//}
		responseOK(ctx, http.StatusOK, app.PatreonPledges())
	}
}
