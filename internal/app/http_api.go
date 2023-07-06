package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gin-gonic/gin"
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
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"github.com/ryanuber/go-glob"
	"go.uber.org/zap"
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

func onAPIPostLog(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

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
		if errServer := app.db.GetServerByName(ctx, upload.ServerName, &server); errServer != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		rawLogs, errDecode := base64.StdEncoding.DecodeString(upload.Body)
		if errDecode != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		logLines := strings.Split(string(rawLogs), "\n")
		log.Debug("Uploaded log file", zap.Int("lines", len(logLines)))
		responseOKUser(ctx, http.StatusCreated, nil, "Log uploaded")
		// Send the log to the logReader() for actual processing
		// TODO deal with this potential block
		// app.LogFileChan <- &model.LogFilePayload{
		//	Server: server,
		//	Lines:  logLines,
		//	Map:    upload.MapName,
		//}
		panic("xx")
	}
}

func onAPIPostDemo(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var upload srcdsup.ServerLogUpload
		if errBind := ctx.BindJSON(&upload); errBind != nil {
			log.Error("Failed to parse demo payload", zap.Error(errBind))
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		if upload.ServerName == "" || upload.Body == "" {
			log.Error("Missing demo params",
				zap.String("server_name", util.SanitizeLog(upload.ServerName)),
				zap.String("map_name", util.SanitizeLog(upload.MapName)),
				zap.Int("body_len", len(upload.Body)))
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		var server store.Server
		if errGetServer := app.db.GetServerByName(ctx, upload.ServerName, &server); errGetServer != nil {
			log.Error("Server not found", zap.String("server", util.SanitizeLog(upload.ServerName)))
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

		for steamID, PlayerStat := range upload.Scores {
			sid64, errSid := steamid.SID64FromString(steamID)
			if errSid != nil {
				log.Error("Failed to parse score steam id", zap.Error(errSid))

				continue
			}

			intStats[sid64] = PlayerStat
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

		if errSave := app.db.SaveDemo(ctx, &newDemo); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to save demo", zap.Error(errSave))

			return
		}

		responseOK(ctx, http.StatusCreated, gin.H{"demo_id": newDemo.DemoID})
	}
}

func onAPIGetDemoDownload(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		demoID, errID := getInt64Param(ctx, "demo_id")
		if errID != nil || demoID <= 0 {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Invalid demo id requested", zap.Error(errID))

			return
		}

		var demo store.DemoFile
		if errGet := app.db.GetDemoByID(ctx, demoID, &demo); errGet != nil {
			if errors.Is(errGet, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, nil)

			log.Error("Error fetching demo", zap.Error(errGet))

			return
		}

		ctx.Header("Content-Description", "File Transfer")
		ctx.Header("Content-Transfer-Encoding", "binary")
		ctx.Header("Content-Disposition", "attachment; filename="+demo.Title)
		ctx.Data(http.StatusOK, "application/octet-stream", demo.Data)

		demo.Downloads++

		if errSave := app.db.SaveDemo(ctx, &demo); errSave != nil {
			log.Error("Failed to increment download count for demo", zap.Error(errSave))
		}
	}
}

func onAPIGetDemoDownloadByName(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		demoName := ctx.Param("demo_name")
		if demoName == "" {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Invalid demo name requested", zap.String("demo_name", ""))

			return
		}

		var demo store.DemoFile

		if errGet := app.db.GetDemoByName(ctx, demoName, &demo); errGet != nil {
			if errors.Is(errGet, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Error fetching demo", zap.Error(errGet))

			return
		}

		ctx.Header("Content-Description", "File Transfer")
		ctx.Header("Content-Transfer-Encoding", "binary")
		ctx.Header("Content-Disposition", "attachment; filename="+demo.Title)
		ctx.Data(http.StatusOK, "application/octet-stream", demo.Data)

		demo.Downloads++

		if errSave := app.db.SaveDemo(ctx, &demo); errSave != nil {
			log.Error("Failed to increment download count for demo", zap.Error(errSave))
		}
	}
}

func onAPIPostDemosQuery(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var opts store.GetDemosOptions
		if errBind := ctx.BindJSON(&opts); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Malformed demo query request", zap.Error(errBind))

			return
		}

		demos, errDemos := app.db.GetDemos(ctx, opts)
		if errDemos != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to query demos", zap.Error(errDemos))

			return
		}

		responseOK(ctx, http.StatusCreated, demos)
	}
}

func onAPIGetServerAdmins(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		perms, err := app.db.GetServerPermissions(ctx)
		if err != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusOK, perms)
	}
}

func onAPIPostPingMod(app *App) gin.HandlerFunc {
	type pingReq struct {
		ServerName string        `json:"server_name"`
		Name       string        `json:"name"`
		SteamID    steamid.SID64 `json:"steam_id"`
		Reason     string        `json:"reason"`
		Client     int           `json:"client"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req pingReq
		if errBind := ctx.BindJSON(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		players, found := app.Find(FindOpts{SteamID: req.SteamID})
		if !found {
			log.Error("Failed to find player on /mod call")
			responseErr(ctx, http.StatusFailedDependency, nil)

			return
		}

		// name := req.SteamID.String()
		// if playerInfo.InGame {
		// 	name = fmt.Sprintf("%s (%s)", name, playerInfo.Player.Name)
		// }
		var roleStrings []string
		for _, roleID := range app.conf.Discord.ModRoleIDs {
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

		for _, chanID := range app.conf.Discord.ModChannels {
			app.bot.SendPayload(discord.Payload{ChannelID: chanID, Embed: embed})
		}

		responseOK(ctx, http.StatusOK, gin.H{
			"client":  req.Client,
			"message": "Moderators have been notified",
		})
	}
}

func onAPIPostBanState(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportID, errID := getInt64Param(ctx, "report_id")
		if errID != nil || reportID <= 0 {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		var report store.Report
		if errReport := app.db.GetReport(ctx, reportID, &report); errReport != nil {
			if errors.Is(errReport, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}
		// app.bot.SendPayload(discord.Payload{ChannelID: "", Embed: nil})
	} //nolint:wsl
}

type apiUnbanRequest struct {
	UnbanReasonText string `json:"unban_reason_text"`
}

func onAPIPostSetBanAppealStatus(app *App) gin.HandlerFunc {
	type setStatusReq struct {
		AppealState store.AppealState `json:"appeal_state"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banID, banIDErr := getInt64Param(ctx, "ban_id")
		if banIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, "Invalid ban_id format")

			return
		}

		var req setStatusReq
		if errBind := ctx.BindJSON(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, "Invalid request")

			return
		}

		bannedPerson := store.NewBannedPerson()
		if banErr := app.db.GetBanByBanID(ctx, banID, &bannedPerson, false); banErr != nil {
			responseErr(ctx, http.StatusInternalServerError, "Failed to query")

			return
		}

		if bannedPerson.Ban.AppealState == req.AppealState {
			responseErr(ctx, http.StatusConflict, "State must be different than previous")

			return
		}

		original := bannedPerson.Ban.AppealState
		bannedPerson.Ban.AppealState = req.AppealState

		if errSave := app.db.SaveBan(ctx, &bannedPerson.Ban); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, "Failed to save appeal state changes")

			return
		}

		responseOK(ctx, http.StatusAccepted, nil)
		log.Info("Updated ban appeal state",
			zap.Int64("ban_id", banID),
			zap.Int("from_state", int(original)),
			zap.Int("to_state", int(req.AppealState)))
	}
}

func onAPIPostBanDelete(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banID, banIDErr := getInt64Param(ctx, "ban_id")
		if banIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, "Invalid ban_id format")

			return
		}

		var req apiUnbanRequest
		if errBind := ctx.BindJSON(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, "Invalid request")

			return
		}

		bannedPerson := store.NewBannedPerson()
		if banErr := app.db.GetBanByBanID(ctx, banID, &bannedPerson, false); banErr != nil {
			responseErr(ctx, http.StatusInternalServerError, "Failed to query")

			return
		}

		changed, errSave := app.Unban(ctx, bannedPerson.Person.SteamID, req.UnbanReasonText)
		if errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, "Failed to unban")

			return
		}

		if !changed {
			responseErr(ctx, http.StatusConflict, "Failed to save")

			return
		}

		responseOK(ctx, http.StatusAccepted, nil)
		log.Info("Ban deleted")
	}
}

func onAPIPostBansGroupCreate(app *App) gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   store.StringSID `json:"target_id"`
		GroupID    steamid.GID     `json:"group_id"`
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

		var (
			banSteamGroup store.BanGroup
			sid           = currentUserProfile(ctx).SteamID
		)

		if errBanSteamGroup := store.NewBanSteamGroup(ctx,
			store.StringSID(sid.String()),
			banRequest.TargetID,
			store.Duration(banRequest.Duration),
			banRequest.Reason,
			banRequest.ReasonText,
			"",
			store.Web,
			banRequest.GroupID,
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

func onAPIPostBansASNCreate(app *App) gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   store.StringSID `json:"target_id"`
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

		var (
			banASN store.BanASN
			sid    = currentUserProfile(ctx).SteamID
		)

		if errBanSteamGroup := store.NewBanASN(ctx,
			store.StringSID(sid.String()),
			banRequest.TargetID,
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

func onAPIPostBansCIDRCreate(app *App) gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   store.StringSID `json:"target_id"`
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

		var (
			banCIDR store.BanCIDR
			sid     = currentUserProfile(ctx).SteamID
		)

		if errBanCIDR := store.NewBanCIDR(ctx,
			store.StringSID(sid.String()),
			banRequest.TargetID,
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

func onAPIPostBanSteamCreate(app *App) gin.HandlerFunc {
	type apiBanRequest struct {
		SourceID   store.StringSID `json:"source_id"`
		TargetID   store.StringSID `json:"target_id"`
		Duration   string          `json:"duration"`
		BanType    store.BanType   `json:"ban_type"`
		Reason     store.Reason    `json:"reason"`
		ReasonText string          `json:"reason_text"`
		Note       string          `json:"note"`
		ReportID   int64           `json:"report_id"`
		DemoName   string          `json:"demo_name"`
		DemoTick   int             `json:"demo_tick"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var banRequest apiBanRequest
		if errBind := ctx.BindJSON(&banRequest); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, "Failed to perform ban")

			return
		}

		var (
			origin   = store.Web
			sid      = currentUserProfile(ctx).SteamID
			sourceID = store.StringSID(sid.String())
		)

		// srcds sourced bans provide a source_id to id the admin
		if banRequest.SourceID != "" {
			sourceID = banRequest.SourceID
			origin = store.InGame
		}

		var banSteam store.BanSteam
		if errBanSteam := store.NewBanSteam(ctx,
			sourceID,
			banRequest.TargetID,
			store.Duration(banRequest.Duration),
			banRequest.Reason,
			banRequest.ReasonText,
			banRequest.Note,
			origin,
			banRequest.ReportID,
			banRequest.BanType,
			&banSteam,
		); errBanSteam != nil {
			responseErr(ctx, http.StatusBadRequest, "Failed to parse options")

			return
		}

		if errBan := app.BanSteam(ctx, &banSteam); errBan != nil {
			log.Error("Failed to ban steam profile",
				zap.Error(errBan), zap.Int64("target_id", banSteam.TargetID.Int64()))

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

func onSAPIPostServerAuth(app *App) gin.HandlerFunc {
	type authReq struct {
		ServerName string `json:"server_name"`
		Key        string `json:"key"`
	}

	type authResp struct {
		Status bool   `json:"status"`
		Token  string `json:"token"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var request authReq
		if errBind := ctx.BindJSON(&request); errBind != nil {
			log.Error("Failed to decode auth request", zap.Error(errBind))
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		var server store.Server

		errGetServer := app.db.GetServerByName(ctx, request.ServerName, &server)
		if errGetServer != nil {
			log.Error("Failed to find server auth by name",
				zap.String("name", request.ServerName), zap.Error(errGetServer))
			responseErr(ctx, http.StatusNotFound, nil)

			return
		}

		if server.Password != request.Key {
			responseErr(ctx, http.StatusForbidden, nil)
			log.Error("Invalid server key used",
				zap.String("server", util.SanitizeLog(request.ServerName)))

			return
		}

		accessToken, errToken := newServerJWT(server.ServerID, app.conf.HTTP.CookieKey)
		if errToken != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to create new server access token", zap.Error(errToken))

			return
		}

		server.TokenCreatedOn = config.Now()
		if errSaveServer := app.db.SaveServer(ctx, &server); errSaveServer != nil {
			log.Error("Failed to updated server token", zap.Error(errSaveServer))
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusOK, authResp{Status: true, Token: accessToken})
		log.Info("Server authenticated successfully", zap.String("server", server.ServerName))
	}
}

func onAPIPostServerCheck(app *App) gin.HandlerFunc {
	type checkRequest struct {
		ClientID int         `json:"client_id"`
		SteamID  steamid.SID `json:"steam_id"`
		IP       net.IP      `json:"ip"`
		Name     string      `json:"name,omitempty"`
	}

	type checkResponse struct {
		ClientID        int              `json:"client_id"`
		SteamID         steamid.SID      `json:"steam_id"`
		BanType         store.BanType    `json:"ban_type"`
		PermissionLevel consts.Privilege `json:"permission_level"`
		Msg             string           `json:"msg"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

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
			log.Info("Player dropped", zap.String("drop_type", "group"),
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

		if errAddHist := app.db.AddConnectionHistory(ctx, &store.PersonConnection{
			IPAddr:      request.IP,
			SteamID:     steamid.SIDToSID64(request.SteamID),
			PersonaName: request.Name,
			CreatedOn:   config.Now(),
			IPInfo:      store.PersonIPRecord{},
		}); errAddHist != nil {
			log.Error("Failed to add conn history", zap.Error(errAddHist))
		}

		// Check IP first
		banNet, errGetBanNet := app.db.GetBanNetByAddress(responseCtx, request.IP)
		if errGetBanNet != nil {
			responseErr(ctx, http.StatusInternalServerError, checkResponse{
				BanType: store.Unknown,
				Msg:     "Error determining state",
			})
			log.Error("Could not get bannedPerson net results", zap.Error(errGetBanNet))

			return
		}

		if len(banNet) > 0 {
			resp.BanType = store.Banned
			resp.Msg = fmt.Sprintf("Network banned (C: %d)", len(banNet))
			responseOK(ctx, http.StatusOK, resp)
			log.Info("Player dropped", zap.String("drop_type", "cidr"),
				zap.Int64("sid64", steamID.Int64()))

			return
		}

		var asnRecord ip2location.ASNRecord

		errASN := app.db.GetASNRecordByIP(responseCtx, request.IP, &asnRecord)
		if errASN == nil {
			var asnBan store.BanASN
			if errASNBan := app.db.GetBanASN(responseCtx, int64(asnRecord.ASNum), &asnBan); errASNBan != nil {
				if !errors.Is(errASNBan, store.ErrNoResult) {
					log.Error("Failed to fetch asn bannedPerson", zap.Error(errASNBan))
				}
			} else {
				resp.BanType = store.Banned
				resp.Msg = store.ReasonString(asnBan.Reason)
				responseOK(ctx, http.StatusOK, resp)
				log.Info("Player dropped", zap.String("drop_type", "asn"),
					zap.Int64("sid64", steamID.Int64()))

				return
			}
		}

		bannedPerson := store.NewBannedPerson()
		if errGetBan := app.db.GetBanBySteamID(responseCtx, steamID, &bannedPerson, false); errGetBan != nil {
			if errors.Is(errGetBan, store.ErrNoResult) {
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

		var reason string

		switch {
		case bannedPerson.Ban.Reason == store.Custom && bannedPerson.Ban.ReasonText != "":
			reason = bannedPerson.Ban.ReasonText
		case bannedPerson.Ban.Reason == store.Custom && bannedPerson.Ban.ReasonText == "":
			reason = "Banned"
		default:
			reason = store.ReasonString(bannedPerson.Ban.Reason)
		}

		resp.Msg = fmt.Sprintf("Banned\nReason: %s\nAppeal: %s\nRemaining: %s", reason, bannedPerson.Ban.ToURL(app.conf),
			bannedPerson.Ban.ValidUntil.Sub(config.Now()).Round(time.Minute).String())

		responseOK(ctx, http.StatusOK, resp)

		if resp.BanType == store.NoComm {
			log.Info("Player muted", zap.Int64("sid64", steamID.Int64()))
		} else if resp.BanType == store.Banned {
			log.Info("Player dropped", zap.String("drop_type", "steam"),
				zap.Int64("sid64", steamID.Int64()))
		}
	}
}

//
// func (w *web) onAPIGetAnsibleHosts() gin.HandlerFunc {
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
// }

// https://prometheus.io/docs/prometheus/latest/configuration/configuration/#http_sd_config
func onAPIGetPrometheusHosts(app *App) gin.HandlerFunc {
	type promStaticConfig struct {
		Targets []string          `json:"targets"`
		Labels  map[string]string `json:"labels"`
	}

	type portMap struct {
		Type string
		Port int
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var staticConfigs []promStaticConfig

		servers, errGetServers := app.db.GetServers(ctx, true)
		if errGetServers != nil {
			log.Error("Failed to fetch servers", zap.Error(errGetServers))
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

// onAPIGetServerStates returns the current known cached server state.
func onAPIGetServerStates(app *App) gin.HandlerFunc {
	type UserServers struct {
		Servers []BaseServer        `json:"servers"`
		LatLong ip2location.LatLong `json:"lat_long"`
	}

	return func(ctx *gin.Context) {
		var (
			lat = getDefaultFloat64(ctx.GetHeader("cf-iplatitude"), 41.7774)
			lon = getDefaultFloat64(ctx.GetHeader("cf-iplongitude"), -87.6160)
			// region := ctx.GetHeader("cf-region-code")
			curState = app.state()
			servers  []BaseServer
		)

		for _, srv := range curState {
			servers = append(servers, BaseServer{
				Host:       srv.Host,
				Port:       srv.Port,
				Name:       srv.Name,
				NameShort:  srv.NameShort,
				Region:     srv.Region,
				CC:         srv.CC,
				ServerID:   srv.ServerID,
				Players:    srv.PlayerCount,
				MaxPlayers: srv.MaxPlayers,
				Bots:       srv.Bots,
				Map:        srv.Map,
				GameTypes:  []string{},
				Latitude:   srv.Latitude,
				Longitude:  srv.Longitude,
				Distance:   distance(srv.Latitude, srv.Longitude, lat, lon),
			})
		}

		sort.SliceStable(servers, func(i, j int) bool {
			return servers[i].Name < servers[j].Name
		})

		responseOK(ctx, http.StatusOK, UserServers{
			Servers: servers,
			LatLong: ip2location.LatLong{
				Latitude:  lat,
				Longitude: lon,
			},
		})
	}
}

func queryFilterFromContext(ctx *gin.Context) (store.QueryFilter, error) {
	var queryFilter store.QueryFilter
	if errBind := ctx.BindUri(&queryFilter); errBind != nil {
		return queryFilter, errors.Wrap(errBind, "Failed to bind URI parameters")
	}

	return queryFilter, nil
}

func onAPIGetPlayers(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		queryFilter, errFilterFromContext := queryFilterFromContext(ctx)
		if errFilterFromContext != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		people, errGetPeople := app.db.GetPeople(ctx, queryFilter)
		if errGetPeople != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusOK, people)
	}
}

func onAPIGetResolveProfile(app *App) gin.HandlerFunc {
	type queryParam struct {
		Query string `json:"query"`
	}

	return func(ctx *gin.Context) {
		var param queryParam
		if errBind := ctx.BindJSON(&param); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		steamID, errResolve := steamid.ResolveSID64(ctx, param.Query)
		if errResolve != nil {
			responseErr(ctx, http.StatusOK, nil)

			return
		}

		var person store.Person
		if errPerson := app.PersonBySID(ctx, steamID, &person); errPerson != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusOK, person)
	}
}

func onAPICurrentProfileNotifications(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userProfile := currentUserProfile(ctx)

		notifications, errNot := app.db.GetPersonNotifications(ctx, userProfile.SteamID)
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

func onAPICurrentProfile(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		userProfile := currentUserProfile(ctx)
		if !userProfile.SteamID.Valid() {
			log.Error("Failed to load user profile",
				zap.Int64("sid64", userProfile.SteamID.Int64()),
				zap.String("name", userProfile.Name),
				zap.String("permission_level", userProfile.PermissionLevel.String()))
			responseErr(ctx, http.StatusForbidden, nil)

			return
		}

		responseOK(ctx, http.StatusOK, userProfile)
	}
}

func onAPIExportBansValveSteamID(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		bans, errBans := app.db.GetBansSteam(ctx, store.BansQueryFilter{
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

func onAPIExportBansValveIP(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		bans, errBans := app.db.GetBansNet(ctx)
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

func onAPIExportSourcemodSimpleAdmins(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		privilegedIds, errPrivilegedIds := app.db.GetSteamIdsAbove(ctx, consts.PReserved)
		if errPrivilegedIds != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		players, errPlayers := app.db.GetPeopleBySteamID(ctx, privilegedIds)
		if errPlayers != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		sort.Slice(players, func(i, j int) bool {
			return players[i].PermissionLevel > players[j].PermissionLevel
		})

		bld := strings.Builder{}

		for _, player := range players {
			var perms string

			switch player.PermissionLevel {
			case consts.PAdmin:
				perms = "z"
			case consts.PModerator:
				perms = "abcdefgjk"
			case consts.PEditor:
				perms = "ak"
			case consts.PReserved:
				perms = "a"
			}

			if perms == "" {
				log.Warn("User has no perm string", zap.Int64("sid", player.SteamID.Int64()))
			} else {
				bld.WriteString(fmt.Sprintf("\"%s\" \"%s\"\n", steamid.SID64ToSID3(player.SteamID), perms))
			}
		}

		ctx.String(http.StatusOK, bld.String())
	}
}

func onAPIExportBansTF2BD(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// TODO limit / make specialized query since this returns all results
		bans, errBans := app.db.GetBansSteam(ctx, store.BansQueryFilter{
			QueryFilter: store.QueryFilter{},
			SteamID:     "",
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
				Authors:     []string{app.conf.General.SiteName},
				Description: "Players permanently banned for cheating",
				Title:       fmt.Sprintf("%s Cheater List", app.conf.General.SiteName),
				UpdateURL:   app.conf.ExtURL("/export/bans/tf2bd"),
			},
			Players: []thirdparty.Players{},
		}

		for _, ban := range filtered {
			out.Players = append(out.Players, thirdparty.Players{
				Attributes: []string{"cheater"},
				Steamid:    ban.Ban.TargetID,
				LastSeen: thirdparty.LastSeen{
					PlayerName: ban.Person.PersonaName,
					Time:       int(ban.Ban.UpdatedOn.Unix()),
				},
			})
		}

		ctx.JSON(http.StatusOK, out)
	}
}

func onAPIProfile(app *App) gin.HandlerFunc {
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

		var response resp

		friendList, errFetchFriends := steamweb.GetFriendList(requestCtx, person.SteamID)
		if errFetchFriends == nil {
			var friendIDs steamid.Collection
			for _, friend := range friendList {
				friendIDs = append(friendIDs, friend.SteamID)
			}

			// TODO add ctx to steamweb lib
			friends, errFetchSummaries := steamweb.PlayerSummaries(ctx, friendIDs)
			if errFetchSummaries != nil {
				app.log.Warn("Could not fetch summaries", zap.Error(errFetchSummaries))
			} else {
				response.Friends = friends
			}
		}

		response.Player = &person

		responseOK(ctx, http.StatusOK, response)
	}
}

func onAPIGetWordFilters(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		words, errGetFilters := app.db.GetFilters(ctx)
		if errGetFilters != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusOK, words)
	}
}

func onAPIPostWordMatch(app *App) gin.HandlerFunc {
	type matchRequest struct {
		Query string
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req matchRequest
		if errBind := ctx.BindJSON(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to parse request", zap.Error(errBind))

			return
		}

		words, errGetFilters := app.db.GetFilters(ctx)
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

func onAPIDeleteWordFilter(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		wordID, wordIDErr := getInt64Param(ctx, "word_id")
		if wordIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		var filter store.Filter
		if errGet := app.db.GetFilterByID(ctx, wordID, &filter); errGet != nil {
			if errors.Is(errGet, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		if errDrop := app.db.DropFilter(ctx, &filter); errDrop != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusOK, nil)
	}
}

func onAPIPostWordFilter(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var filter store.Filter
		if errBind := ctx.BindJSON(&filter); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to parse request", zap.Error(errBind))

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
			if errGet := app.db.GetFilterByID(ctx, filter.FilterID, &existingFilter); errGet != nil {
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
				AuthorID:  profile.SteamID,
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

func onAPIGetStats(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var stats store.Stats
		if errGetStats := app.db.GetStats(ctx, &stats); errGetStats != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		stats.ServersAlive = 1

		responseOK(ctx, http.StatusOK, stats)
	}
}

func loadBanMeta(_ *store.BannedPerson) {
}

func onAPIGetBanByID(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		curUser := currentUserProfile(ctx)

		banID, errID := getInt64Param(ctx, "ban_id")
		if errID != nil || banID == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		deletedOk := false

		fullValue, fullOk := ctx.GetQuery("deleted")
		if fullOk {
			deleted, deletedOkErr := strconv.ParseBool(fullValue)
			if deletedOkErr != nil {
				log.Error("Failed to parse ban full query value", zap.Error(deletedOkErr))
			} else {
				deletedOk = deleted
			}
		}

		bannedPerson := store.NewBannedPerson()
		if errGetBan := app.db.GetBanByBanID(ctx, banID, &bannedPerson, deletedOk); errGetBan != nil {
			responseErr(ctx, http.StatusNotFound, nil)
			log.Error("Failed to fetch bans", zap.Error(errGetBan))

			return
		}

		if !checkPrivilege(ctx, curUser, steamid.Collection{bannedPerson.Person.SteamID}, consts.PModerator) {
			return
		}

		loadBanMeta(&bannedPerson)
		responseOK(ctx, http.StatusOK, bannedPerson)
	}
}

func onAPIGetAppeals(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var queryFilter store.QueryFilter
		if errBind := ctx.BindJSON(&queryFilter); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		bans, errBans := app.db.GetAppealsByActivity(ctx, queryFilter)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to fetch bans", zap.Error(errBans))

			return
		}

		responseOK(ctx, http.StatusOK, bans)
	}
}

func onAPIGetBansSteam(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var queryFilter store.BansQueryFilter
		if errBind := ctx.BindJSON(&queryFilter); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		bans, errBans := app.db.GetBansSteam(ctx, queryFilter)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to fetch bans", zap.Error(errBans))

			return
		}

		responseOK(ctx, http.StatusOK, bans)
	}
}

func onAPIGetBansCIDR(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var queryFilter store.BansQueryFilter
		if errBind := ctx.BindJSON(&queryFilter); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		// TODO filters
		bans, errBans := app.db.GetBansNet(ctx)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to fetch bans", zap.Error(errBans))

			return
		}

		responseOK(ctx, http.StatusOK, bans)
	}
}

func onAPIDeleteBansCIDR(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		netID, netIDErr := getInt64Param(ctx, "net_id")
		if netIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		var req apiUnbanRequest
		if errBind := ctx.BindJSON(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, "Invalid request")

			return
		}

		var banCidr store.BanCIDR
		if errFetch := app.db.GetBanNetByID(ctx, netID, &banCidr); errFetch != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		banCidr.UnbanReasonText = req.UnbanReasonText
		banCidr.Deleted = true

		if errSave := app.db.SaveBanNet(ctx, &banCidr); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to delete cidr ban", zap.Error(errSave))

			return
		}

		banCidr.NetID = 0

		responseOK(ctx, http.StatusOK, banCidr)
	}
}

func onAPIGetBansGroup(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var queryFilter store.BansQueryFilter
		if errBind := ctx.BindJSON(&queryFilter); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		// TODO filters
		banGroups, errBans := app.db.GetBanGroups(ctx)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to fetch banGroups", zap.Error(errBans))

			return
		}

		responseOK(ctx, http.StatusOK, banGroups)
	}
}

func onAPIDeleteBansGroup(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		groupID, groupIDErr := getInt64Param(ctx, "ban_group_id")
		if groupIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		var req apiUnbanRequest
		if errBind := ctx.BindJSON(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, "Invalid request")

			return
		}

		var banGroup store.BanGroup
		if errFetch := app.db.GetBanGroupByID(ctx, groupID, &banGroup); errFetch != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		banGroup.UnbanReasonText = req.UnbanReasonText
		banGroup.Deleted = true

		if errSave := app.db.SaveBanGroup(ctx, &banGroup); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to delete asn ban", zap.Error(errSave))

			return
		}

		banGroup.BanGroupID = 0
		responseOK(ctx, http.StatusOK, banGroup)
	}
}

func onAPIGetBansASN(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var queryFilter store.BansQueryFilter
		if errBind := ctx.BindJSON(&queryFilter); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		// TODO filters
		banASN, errBans := app.db.GetBansASN(ctx)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to fetch banASN", zap.Error(errBans))

			return
		}

		responseOK(ctx, http.StatusOK, banASN)
	}
}

func onAPIDeleteBansASN(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		asnID, asnIDErr := getInt64Param(ctx, "asn_id")
		if asnIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		var req apiUnbanRequest
		if errBind := ctx.BindJSON(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, "Invalid request")

			return
		}

		var banAsn store.BanASN
		if errFetch := app.db.GetBanASN(ctx, asnID, &banAsn); errFetch != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		banAsn.UnbanReasonText = req.UnbanReasonText
		banAsn.Deleted = true

		if errSave := app.db.SaveBanASN(ctx, &banAsn); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to delete asn ban", zap.Error(errSave))

			return
		}

		banAsn.BanASNId = 0

		responseOK(ctx, http.StatusOK, banAsn)
	}
}

func onAPIGetServers(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		servers, errServers := app.db.GetServers(ctx, true)
		if errServers != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusOK, servers)
	}
}

type serverUpdateRequest struct {
	ServerName      string  `json:"server_name"`
	ServerNameShort string  `json:"server_name_short"`
	Host            string  `json:"host"`
	Port            int     `json:"port"`
	ReservedSlots   int     `json:"reserved_slots"`
	RCON            string  `json:"rcon"`
	Lat             float64 `json:"lat"`
	Lon             float64 `json:"lon"`
	CC              string  `json:"cc"`
	DefaultMap      string  `json:"default_map"`
	Region          string  `json:"region"`
	IsEnabled       bool    `json:"is_enabled"`
}

func onAPIPostServerUpdate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		serverID, idErr := getIntParam(ctx, "server_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		var server store.Server
		if errServer := app.db.GetServer(ctx, serverID, &server); errServer != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		var serverReq serverUpdateRequest
		if errBind := ctx.BindJSON(&serverReq); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to parse request to update server", zap.Error(errBind))

			return
		}

		server.ServerName = serverReq.ServerNameShort
		server.ServerNameLong = serverReq.ServerName
		server.Address = serverReq.Host
		server.Port = serverReq.Port
		server.ReservedSlots = serverReq.ReservedSlots
		server.RCON = serverReq.RCON
		server.Latitude = serverReq.Lat
		server.Longitude = serverReq.Lon
		server.CC = serverReq.CC
		server.Region = serverReq.Region
		server.IsEnabled = serverReq.IsEnabled

		if errSave := app.db.SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to update server", zap.Error(errSave))

			return
		}

		responseOK(ctx, http.StatusOK, server)

		log.Info("Server config updated",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ServerName))
	}
}

func onAPIPostServerDelete(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		serverID, idErr := getIntParam(ctx, "server_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		var server store.Server
		if errServer := app.db.GetServer(ctx, serverID, &server); errServer != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		server.Deleted = true

		if errSave := app.db.SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to delete server", zap.Error(errSave))

			return
		}

		responseOK(ctx, http.StatusOK, server)
		log.Info("Server config deleted",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ServerName))
	}
}

func onAPIPostServer(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var serverReq serverUpdateRequest
		if errBind := ctx.BindJSON(&serverReq); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to parse request for new server", zap.Error(errBind))

			return
		}

		server := store.NewServer(serverReq.ServerNameShort, serverReq.Host, serverReq.Port)
		server.ServerNameLong = serverReq.ServerName
		server.ReservedSlots = serverReq.ReservedSlots
		server.RCON = serverReq.RCON
		server.Latitude = serverReq.Lat
		server.Longitude = serverReq.Lon
		server.CC = serverReq.CC
		server.Region = serverReq.Region
		server.IsEnabled = serverReq.IsEnabled

		if errSave := app.db.SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to save new server", zap.Error(errSave))

			return
		}

		responseOK(ctx, http.StatusOK, server)

		log.Info("Server config created",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ServerName))
	}
}

func onAPIPostReportCreate(app *App) gin.HandlerFunc {
	type createReport struct {
		SourceID    store.StringSID `json:"source_id"`
		TargetID    store.StringSID `json:"target_id"`
		Description string          `json:"description"`
		Reason      store.Reason    `json:"reason"`
		ReasonText  string          `json:"reason_text"`
		DemoName    string          `json:"demo_name"`
		DemoTick    int             `json:"demo_tick"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		var newReport createReport
		if errBind := ctx.BindJSON(&newReport); errBind != nil {
			responseErrUser(ctx, http.StatusBadRequest, nil, "Invalid request")
			log.Error("Failed to bind report", zap.Error(errBind))

			return
		}

		// Server initiated requests will have a sourceID set by the server
		// Web based reports the source should not be set, the reporter will be taken from the
		// current session information instead
		if newReport.SourceID == "" {
			newReport.SourceID = store.StringSID(currentUser.SteamID.String())
		}

		sourceID, errSourceID := newReport.SourceID.SID64(ctx)
		if errSourceID != nil {
			responseErrUser(ctx, http.StatusBadRequest, nil, "Failed to resolve steam id")
			log.Error("Invalid steam_id", zap.Error(errSourceID))

			return
		}

		targetID, errTargetID := newReport.TargetID.SID64(ctx)
		if errTargetID != nil {
			responseErrUser(ctx, http.StatusBadRequest, nil, "Failed to resolve steam id")
			log.Error("Invalid target_id", zap.Error(errTargetID))

			return
		}

		if sourceID == targetID {
			responseErrUser(ctx, http.StatusForbidden, nil, "Cannot report yourself")

			return
		}

		var personSource store.Person
		if errCreatePerson := app.PersonBySID(ctx, sourceID, &personSource); errCreatePerson != nil {
			responseErrUser(ctx, http.StatusInternalServerError, nil, "Internal error")
			log.Error("Could not load player profile", zap.Error(errCreatePerson))

			return
		}

		var personTarget store.Person
		if errCreatePerson := app.PersonBySID(ctx, targetID, &personTarget); errCreatePerson != nil {
			responseErrUser(ctx, http.StatusInternalServerError, nil, "Internal error")
			log.Error("Could not load player profile", zap.Error(errCreatePerson))

			return
		}

		// Ensure the user doesn't already have an open report against the user
		var existing store.Report
		if errReports := app.db.GetReportBySteamID(ctx, currentUser.SteamID, targetID, &existing); errReports != nil {
			if !errors.Is(errReports, store.ErrNoResult) {
				log.Error("Failed to query reports by steam id", zap.Error(errReports))
				responseErr(ctx, http.StatusInternalServerError, nil)

				return
			}
		}

		if existing.ReportID > 0 {
			responseErrUser(ctx, http.StatusConflict, nil,
				"Must resolve existing report for user before creating another")

			return
		}

		// TODO encapsulate all operations in single tx
		report := store.NewReport()
		report.SourceID = sourceID
		report.ReportStatus = store.Opened
		report.Description = newReport.Description
		report.TargetID = targetID
		report.Reason = newReport.Reason
		report.ReasonText = newReport.ReasonText
		parts := strings.Split(newReport.DemoName, "/")
		report.DemoName = parts[len(parts)-1]
		report.DemoTick = newReport.DemoTick

		if errReportSave := app.db.SaveReport(ctx, &report); errReportSave != nil {
			responseErrUser(ctx, http.StatusInternalServerError, nil, "Failed to save report")
			log.Error("Failed to save report", zap.Error(errReportSave))

			return
		}

		responseOK(ctx, http.StatusCreated, report)

		embed := discord.RespOk(nil, "New user report created")
		embed.Description = report.Description
		embed.URL = report.ToURL(app.conf)
		discord.AddAuthorProfile(embed,
			currentUser.SteamID, currentUser.Name, currentUser.ToURL(app.conf))

		name := personSource.PersonaName

		if name == "" {
			name = report.TargetID.String()
		}

		discord.AddField(embed, "Subject", name)
		discord.AddField(embed, "Reason", store.ReasonString(report.Reason))

		if report.ReasonText != "" {
			discord.AddField(embed, "Custom Reason", report.ReasonText)
		}

		if report.DemoName != "" {
			discord.AddField(embed, "Demo", app.conf.ExtURL("/demos/name/%s", report.DemoName))
			discord.AddField(embed, "Demo Tick", fmt.Sprintf("%d", report.DemoTick))
		}

		discord.AddFieldsSteamID(embed, report.TargetID)
		discord.AddLink(embed, app.conf, report)

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.ReportLogChannelID,
			Embed:     embed,
		})
	}
}

func getSID64Param(c *gin.Context, key string) (steamid.SID64, error) {
	i, errGetParam := getInt64Param(c, key)
	if errGetParam != nil {
		return "", errGetParam
	}

	sid := steamid.New(i)
	if !sid.Valid() {
		return "", consts.ErrInvalidSID
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

func onAPIPostReportMessage(app *App) gin.HandlerFunc {
	type req struct {
		Message string `json:"message"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportID, errID := getInt64Param(ctx, "report_id")
		if errID != nil || reportID == 0 {
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
		if errReport := app.db.GetReport(ctx, reportID, &report); errReport != nil {
			if errors.Is(store.Err(errReport), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)

				return
			}

			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to load report", zap.Error(errReport))

			return
		}

		person := currentUserProfile(ctx)
		msg := store.NewUserMessage(reportID, person.SteamID, request.Message)

		if errSave := app.db.SaveReportMessage(ctx, &msg); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		responseOK(ctx, http.StatusCreated, msg)

		embed := &discordgo.MessageEmbed{
			Title:       "New report message posted",
			Description: msg.Contents,
		}

		discord.AddField(embed, "Author", report.SourceID.String())
		discord.AddLink(embed, app.conf, report)

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.ReportLogChannelID,
			Embed:     embed,
		})
	}
}

func onAPIEditReportMessage(app *App) gin.HandlerFunc {
	type editMessage struct {
		BodyMD string `json:"body_md"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportMessageID, errID := getInt64Param(ctx, "report_message_id")
		if errID != nil || reportMessageID == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		var existing store.UserMessage
		if errExist := app.db.GetReportMessageByID(ctx, reportMessageID, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		curUser := currentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, consts.PModerator) {
			return
		}

		var message editMessage
		if errBind := ctx.BindJSON(&message); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		if message.BodyMD == "" {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		if message.BodyMD == existing.Contents {
			responseErr(ctx, http.StatusConflict, nil)

			return
		}

		existing.Contents = message.BodyMD
		if errSave := app.db.SaveReportMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		responseOK(ctx, http.StatusCreated, message)

		embed := &discordgo.MessageEmbed{
			Title:       "New report message edited",
			Description: message.BodyMD,
		}

		discord.AddField(embed, "Old Message", existing.Contents)
		discord.AddField(embed, "Report Link", app.conf.ExtURL("/report/%d", existing.ParentID))
		discord.AddField(embed, "Author", curUser.SteamID.String())

		embed.Image = &discordgo.MessageEmbedImage{URL: curUser.Avatarfull}

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.ModLogChannelID,
			Embed:     embed,
		})
	}
}

func onAPIDeleteReportMessage(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportMessageID, errID := getInt64Param(ctx, "report_message_id")
		if errID != nil || reportMessageID == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		var existing store.UserMessage
		if errExist := app.db.GetReportMessageByID(ctx, reportMessageID, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		curUser := currentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, consts.PModerator) {
			return
		}

		existing.Deleted = true
		if errSave := app.db.SaveReportMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		responseOK(ctx, http.StatusNoContent, nil)

		embed := &discordgo.MessageEmbed{
			Title:       "User report message deleted",
			Description: existing.Contents,
		}

		discord.AddField(embed, "Author", curUser.SteamID.String())

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.ModLogChannelID,
			Embed:     embed,
		})
	}
}

type AuthorMessage struct {
	Author  store.Person      `json:"author"`
	Message store.UserMessage `json:"message"`
}

func onAPISetReportStatus(app *App) gin.HandlerFunc {
	type stateUpdateReq struct {
		Status store.ReportStatus `json:"status"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportID, errParam := getInt64Param(ctx, "report_id")
		if errParam != nil {
			responseErr(ctx, http.StatusNotFound, nil)

			return
		}

		var newStatus stateUpdateReq
		if errBind := ctx.BindJSON(&newStatus); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		var report store.Report
		if errGet := app.db.GetReport(ctx, reportID, &report); errGet != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to get report to set state", zap.Error(errGet))

			return
		}

		if report.ReportStatus == newStatus.Status {
			responseOK(ctx, http.StatusConflict, nil)

			return
		}

		original := report.ReportStatus

		report.ReportStatus = newStatus.Status
		if errSave := app.db.SaveReport(ctx, &report); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to save report state", zap.Error(errSave))

			return
		}

		responseOK(ctx, http.StatusAccepted, nil)
		log.Info("Report status changed",
			zap.Int64("report_id", report.ReportID),
			zap.String("from_status", original.String()),
			zap.String("to_status", report.ReportStatus.String()))
		// discord.SendDiscord(model.NotificationPayload{
		//	Sids:     steamid.Collection{report.SourceID},
		//	Severity: store.SeverityInfo,
		//	Message:  "Report status updated",
		//	Link:     report.ToURL(),
		// })
	} //nolint:wsl
}

func onAPIGetReportMessages(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportID, errParam := getInt64Param(ctx, "report_id")
		if errParam != nil {
			responseErr(ctx, http.StatusNotFound, nil)

			return
		}

		var report store.Report
		if errGetReport := app.db.GetReport(ctx, reportID, &report); errGetReport != nil {
			responseErr(ctx, http.StatusNotFound, nil)

			return
		}

		if !checkPrivilege(ctx, currentUserProfile(ctx), steamid.Collection{report.SourceID, report.TargetID}, consts.PModerator) {
			return
		}

		reportMessages, errGetReportMessages := app.db.GetReportMessages(ctx, reportID)
		if errGetReportMessages != nil {
			responseErr(ctx, http.StatusNotFound, nil)

			return
		}

		var ids steamid.Collection
		for _, msg := range reportMessages {
			ids = append(ids, msg.AuthorID)
		}

		authors, authorsErr := app.db.GetPeopleBySteamID(ctx, ids)
		if authorsErr != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		var (
			authorsMap     = authors.AsMap()
			authorMessages []AuthorMessage
		)

		for _, message := range reportMessages {
			authorMessages = append(authorMessages, AuthorMessage{
				Author:  authorsMap[message.AuthorID],
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

func onAPIGetReports(app *App) gin.HandlerFunc {
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

		reports, errReports := app.db.GetReports(ctx, opts)
		if errReports != nil {
			if errors.Is(store.Err(errReports), store.ErrNoResult) {
				responseOK(ctx, http.StatusNoContent, nil)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		var authorIds steamid.Collection
		for _, report := range reports {
			authorIds = append(authorIds, report.SourceID)
		}

		authors, errAuthors := app.db.GetPeopleBySteamID(ctx, fp.Uniq[steamid.SID64](authorIds))
		if errAuthors != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		authorMap := authors.AsMap()

		var subjectIds steamid.Collection
		for _, report := range reports {
			subjectIds = append(subjectIds, report.TargetID)
		}

		subjects, errSubjects := app.db.GetPeopleBySteamID(ctx, fp.Uniq[steamid.SID64](subjectIds))
		if errSubjects != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		subjectMap := subjects.AsMap()

		for _, report := range reports {
			userReports = append(userReports, reportWithAuthor{
				Author:  authorMap[report.SourceID],
				Report:  report,
				Subject: subjectMap[report.TargetID],
			})
		}

		sort.SliceStable(userReports, func(i, j int) bool {
			return userReports[i].Report.ReportID > userReports[j].Report.ReportID
		})

		responseOK(ctx, http.StatusOK, userReports)
	}
}

func onAPIGetReport(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportID, errParam := getInt64Param(ctx, "report_id")
		if errParam != nil {
			responseErr(ctx, http.StatusNotFound, nil)

			return
		}

		var report reportWithAuthor
		if errReport := app.db.GetReport(ctx, reportID, &report.Report); errReport != nil {
			if errors.Is(store.Err(errReport), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)

				return
			}

			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to load report", zap.Error(errReport))

			return
		}

		if !checkPrivilege(ctx, currentUserProfile(ctx), steamid.Collection{report.Report.SourceID}, consts.PModerator) {
			responseErr(ctx, http.StatusUnauthorized, nil)

			return
		}

		if errAuthor := app.PersonBySID(ctx, report.Report.SourceID, &report.Author); errAuthor != nil {
			if errors.Is(store.Err(errAuthor), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)

				return
			}

			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to load report author", zap.Error(errAuthor))

			return
		}

		if errSubject := app.PersonBySID(ctx, report.Report.TargetID, &report.Subject); errSubject != nil {
			if errors.Is(store.Err(errSubject), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)

				return
			}

			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to load report subject", zap.Error(errSubject))

			return
		}

		responseOK(ctx, http.StatusOK, report)
	}
}

func onAPIGetNewsLatest(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := app.db.GetNewsLatest(ctx, 50, false)
		if errGetNewsLatest != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusOK, newsLatest)
	}
}

func onAPIGetNewsAll(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := app.db.GetNewsLatest(ctx, 100, true)
		if errGetNewsLatest != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusOK, newsLatest)
	}
}

func onAPIPostNewsCreate(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var entry store.NewsEntry
		if errBind := ctx.BindJSON(&entry); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		if errSave := app.db.SaveNewsArticle(ctx, &entry); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusCreated, entry)

		go app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.ModLogChannelID,
			Embed: &discordgo.MessageEmbed{
				Title:       "News Created",
				Description: fmt.Sprintf("News Posted: %s", entry.Title),
			},
		})
	}
}

func onAPIPostNewsUpdate(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsID, errID := getIntParam(ctx, "news_id")
		if errID != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		var entry store.NewsEntry
		if errGet := app.db.GetNewsByID(ctx, newsID, &entry); errGet != nil {
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

		if errSave := app.db.SaveNewsArticle(ctx, &entry); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusAccepted, entry)

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.ModLogChannelID,
			Embed: &discordgo.MessageEmbed{
				Title:       "News Updated",
				Description: fmt.Sprintf("News Updated: %s", entry.Title),
			},
		})
	}
}

func onAPISaveMedia(app *App) gin.HandlerFunc {
	MediaSafeMimeTypesImages := []string{
		"image/gif",
		"image/jpeg",
		"image/png",
		"image/webp",
	}

	type UserUploadedFile struct {
		Content string `json:"content"`
		Name    string `json:"name"`
		Mime    string `json:"mime"`
		Size    int64  `json:"size"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

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
			log.Error("Invalid media uploaded", zap.Error(errMedia))
		}

		if !fp.Contains(MediaSafeMimeTypesImages, media.MimeType) {
			responseErrUser(ctx, http.StatusBadRequest, nil, "Invalid image format")
			log.Error("User tried uploading image with forbidden mimetype",
				zap.String("mime", media.MimeType), zap.String("name", media.Name))

			return
		}

		if errSave := app.db.SaveMedia(ctx, &media); errSave != nil {
			log.Error("Failed to save wiki media", zap.Error(errSave))

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

func onAPIGetWikiSlug(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slug := strings.ToLower(ctx.Param("slug"))
		if slug[0] == '/' {
			slug = slug[1:]
		}

		var page wiki.Page
		if errGetWikiSlug := app.db.GetWikiPageBySlug(ctx, slug, &page); errGetWikiSlug != nil {
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

func onGetMediaByID(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mediaID, idErr := getIntParam(ctx, "media_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		var media store.Media
		if errMedia := app.db.GetMediaByID(ctx, mediaID, &media); errMedia != nil {
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

func onAPISaveWikiSlug(app *App) gin.HandlerFunc {
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
		if errGetWikiSlug := app.db.GetWikiPageBySlug(ctx, request.Slug, &page); errGetWikiSlug != nil {
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
		if errSave := app.db.SaveWikiPage(ctx, &page); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusCreated, page)
	}
}

func onAPIGetMatches(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var opts store.MatchesQueryOpts
		if errBind := ctx.BindJSON(&opts); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		matches, matchesErr := app.db.Matches(ctx, opts)
		if matchesErr != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusOK, matches)
	}
}

func onAPIGetMatch(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		matchID, errID := getIntParam(ctx, "match_id")
		if errID != nil {
			log.Error("Invalid match_id value", zap.Error(errID))
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		match, errMatch := app.db.MatchGetByID(ctx, matchID)
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

func onAPIGetPersonConnections(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		steamID, errID := getSID64Param(ctx, "steam_id")
		if errID != nil {
			log.Error("Invalid steam_id value", zap.Error(errID))
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		// TODO paging
		ipHist, errIPHist := app.db.GetPersonIPHistory(ctx, steamID, 1000)
		if errIPHist != nil && !errors.Is(errIPHist, store.ErrNoResult) {
			log.Error("Failed to query connection history",
				zap.Error(errIPHist), zap.Int64("sid64", steamID.Int64()))
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		if ipHist == nil {
			ipHist = store.PersonConnections{}
		}

		responseOK(ctx, http.StatusOK, ipHist)
	}
}

func onAPIQueryMessages(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var query store.ChatHistoryQueryFilter
		if !bind(ctx, &query) {
			return
		}

		if query.Limit <= 0 || query.Limit > 1000 {
			query.Limit = 1000
		}

		// TODO paging
		chat, errChat := app.db.QueryChatHistory(ctx, query)
		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			log.Error("Failed to query chat history",
				zap.Error(errChat), zap.String("sid", query.SteamID))
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		if chat == nil {
			chat = store.PersonMessages{}
		}

		responseOK(ctx, http.StatusOK, chat)
	}
}

func onAPIGetMessageContext(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		messageID, errID := getInt64Param(ctx, "person_message_id")
		if errID != nil {
			log.Error("Invalid steam_id value", zap.Error(errID))
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		var message store.PersonMessage
		if errMsg := app.db.GetPersonMessageByID(ctx, messageID, &message); errMsg != nil {
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
		chat, errChat := app.db.QueryChatHistory(ctx, store.ChatHistoryQueryFilter{
			ServerID:    message.ServerID,
			SentAfter:   &after,
			SentBefore:  &before,
			QueryFilter: store.QueryFilter{},
		})

		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			log.Error("Failed to query chat history",
				zap.Error(errChat), zap.Int64("person_message_id", messageID))
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		if chat == nil {
			chat = store.PersonMessages{}
		}

		responseOK(ctx, http.StatusOK, chat)
	}
}

func onAPIGetPersonMessages(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		steamID, errID := getSID64Param(ctx, "steam_id")
		if errID != nil {
			log.Error("Invalid steam_id value", zap.Error(errID))
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		// TODO paging
		chat, errChat := app.db.QueryChatHistory(ctx, store.ChatHistoryQueryFilter{
			SteamID: steamID.String(),
			QueryFilter: store.QueryFilter{
				Limit: 1000,
			},
		})

		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			log.Error("Failed to query chat history",
				zap.Error(errChat), zap.Int64("sid64", steamID.Int64()))
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

func onAPIGetBanMessages(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banID, errParam := getInt64Param(ctx, "ban_id")
		if errParam != nil {
			responseErr(ctx, http.StatusNotFound, nil)

			return
		}

		banPerson := store.NewBannedPerson()
		if errGetBan := app.db.GetBanByBanID(ctx, banID, &banPerson, true); errGetBan != nil {
			responseErr(ctx, http.StatusNotFound, nil)

			return
		}

		if !checkPrivilege(ctx, currentUserProfile(ctx), steamid.Collection{banPerson.Ban.TargetID, banPerson.Ban.SourceID}, consts.PModerator) {
			return
		}

		banMessages, errGetBanMessages := app.db.GetBanMessages(ctx, banID)
		if errGetBanMessages != nil {
			responseErr(ctx, http.StatusNotFound, nil)

			return
		}

		var ids steamid.Collection
		for _, msg := range banMessages {
			ids = append(ids, msg.AuthorID)
		}

		authors, authorsErr := app.db.GetPeopleBySteamID(ctx, ids)
		if authorsErr != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		authorsMap := authors.AsMap()

		var authorMessages []AuthorBanMessage
		for _, message := range banMessages {
			authorMessages = append(authorMessages, AuthorBanMessage{
				Author:  authorsMap[message.AuthorID],
				Message: message,
			})
		}

		responseOK(ctx, http.StatusOK, authorMessages)
	}
}

func onAPIDeleteBanMessage(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banMessageID, errID := getIntParam(ctx, "ban_message_id")
		if errID != nil || banMessageID == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		var existing store.UserMessage
		if errExist := app.db.GetBanMessageByID(ctx, banMessageID, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		curUser := currentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, consts.PModerator) {
			return
		}

		existing.Deleted = true
		if errSave := app.db.SaveBanMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to save appeal message", zap.Error(errSave))

			return
		}

		responseOK(ctx, http.StatusNoContent, nil)

		embed := &discordgo.MessageEmbed{
			Title:       "User appeal message deleted",
			Description: existing.Contents,
		}
		discord.AddField(embed, "Author", curUser.SteamID.String())
		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.ReportLogChannelID,
			Embed:     embed,
		})
	}
}

func onAPIGetSourceBans(_ *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, errID := getSID64Param(ctx, "steam_id")
		if errID != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		records, errRecords := getSourceBans(ctx, steamID)
		if errRecords != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusOK, records)
	}
}

type sbBanRecord struct {
	BanID       int           `json:"ban_id"`
	SiteName    string        `json:"site_name"`
	SiteID      int           `json:"site_id"`
	PersonaName string        `json:"persona_name"`
	SteamID     steamid.SID64 `json:"steam_id"`
	Reason      string        `json:"reason"`
	Duration    time.Duration `json:"duration"`
	Permanent   bool          `json:"permanent"`
	CreatedOn   time.Time     `json:"created_on"`
}

func getSourceBans(ctx context.Context, steamID steamid.SID64) ([]sbBanRecord, error) {
	client := &http.Client{Timeout: time.Second * 10}
	url := fmt.Sprintf("https://bd-api.roto.lol/sourcebans/%s", steamID)

	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if errReq != nil {
		return nil, errors.Wrap(errReq, "Failed to create request")
	}

	resp, errResp := client.Do(req)
	if errResp != nil {
		return nil, errors.Wrap(errResp, "Failed to perform request")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, errBody := io.ReadAll(resp.Body)
	if errBody != nil {
		return nil, errors.Wrap(errBody, "Failed to read body")
	}

	var records []sbBanRecord
	if errJSON := json.Unmarshal(body, &records); errJSON != nil {
		return nil, errors.Wrap(errJSON, "Failed to decode body")
	}

	return records, nil
}

func onAPIPostBanMessage(app *App) gin.HandlerFunc {
	type req struct {
		Message string `json:"message"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banID, errID := getInt64Param(ctx, "ban_id")
		if errID != nil || banID == 0 {
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

		bannedPerson := store.NewBannedPerson()
		if errReport := app.db.GetBanByBanID(ctx, banID, &bannedPerson, true); errReport != nil {
			if errors.Is(store.Err(errReport), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)

				return
			}

			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to load ban", zap.Error(errReport))

			return
		}

		userProfile := currentUserProfile(ctx)
		if bannedPerson.Ban.AppealState != store.Open && userProfile.PermissionLevel < consts.PModerator {
			responseErr(ctx, http.StatusForbidden, nil)
			log.Warn("User tried to bypass posting restriction",
				zap.Int64("ban_id", bannedPerson.Ban.BanID), zap.Int64("steam_id", bannedPerson.Person.SteamID.Int64()))

			return
		}

		msg := store.NewUserMessage(banID, userProfile.SteamID, request.Message)
		if errSave := app.db.SaveBanMessage(ctx, &msg); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to save ban appeal message", zap.Error(errSave))

			return
		}

		responseOK(ctx, http.StatusCreated, msg)

		embed := &discordgo.MessageEmbed{
			Title:       "New ban appeal message posted",
			Description: msg.Contents,
			Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: userProfile.Avatarfull},
			Color:       app.bot.ColourLevels.Info,
			URL:         app.conf.ExtURL("/ban/%d", banID),
		}

		discord.AddAuthorProfile(embed,
			userProfile.SteamID, userProfile.Name, userProfile.ToURL(app.conf))

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.ReportLogChannelID,
			Embed:     embed,
		})
	}
}

func onAPIEditBanMessage(app *App) gin.HandlerFunc {
	type editMessage struct {
		BodyMD string `json:"body_md"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportMessageID, errID := getIntParam(ctx, "ban_message_id")
		if errID != nil || reportMessageID == 0 {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		var existing store.UserMessage
		if errExist := app.db.GetBanMessageByID(ctx, reportMessageID, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, nil)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		curUser := currentUserProfile(ctx)

		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, consts.PModerator) {
			return
		}

		var message editMessage
		if errBind := ctx.BindJSON(&message); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		if message.BodyMD == "" {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		if message.BodyMD == existing.Contents {
			responseErr(ctx, http.StatusConflict, nil)

			return
		}

		embed := &discordgo.MessageEmbed{
			Title:       "Ban appeal message edited",
			Description: util.DiffString(existing.Contents, message.BodyMD),
		}

		existing.Contents = message.BodyMD
		if errSave := app.db.SaveBanMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to save ban appeal message", zap.Error(errSave))

			return
		}

		responseOK(ctx, http.StatusCreated, message)

		discord.AddField(embed, "Author", curUser.SteamID.String())
		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.ReportLogChannelID,
			Embed:     embed,
		})
	}
}

func distance(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
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
	dist *= 1.609344 // convert to km

	return dist
}

func onAPIPostServerQuery(app *App) gin.HandlerFunc {
	type masterQueryRequest struct {
		// ctf,payload,cp,mvm,pd,passtime,mannpower,koth
		GameTypes  []string  `json:"game_types,omitempty"`
		AppID      int64     `json:"app_id,omitempty"`
		Maps       []string  `json:"maps,omitempty"`
		PlayersMin int       `json:"players_min,omitempty"`
		PlayersMax int       `json:"players_max,omitempty"`
		NotFull    bool      `json:"not_full,omitempty"`
		Location   []float64 `json:"location,omitempty"`
		Name       string    `json:"name,omitempty"`
		HasBots    bool      `json:"has_bots,omitempty"`
	}

	filterGameTypes := func(servers []state.ServerLocation, gameTypes []string) []state.ServerLocation {
		var valid []state.ServerLocation

		for _, server := range servers {
			serverTypes := strings.Split(server.GameType, ",")
			for _, gt := range gameTypes {
				if fp.Contains(serverTypes, gt) {
					valid = append(valid, server)

					break
				}
			}
		}

		return valid
	}

	filterMaps := func(servers []state.ServerLocation, mapNames []string) []state.ServerLocation {
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

	filterPlayersMin := func(servers []state.ServerLocation, minimum int) []state.ServerLocation {
		var valid []state.ServerLocation

		for _, server := range servers {
			if server.Players >= minimum {
				valid = append(valid, server)

				break
			}
		}

		return valid
	}

	filterPlayersMax := func(servers []state.ServerLocation, maximum int) []state.ServerLocation {
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
		if errLoc := app.db.GetLocationRecord(ctx, net.ParseIP("68.144.74.48"), &record); errLoc != nil {
			responseErr(ctx, http.StatusForbidden, nil)

			return
		}

		var req masterQueryRequest
		if errBind := ctx.BindJSON(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		app.stateMu.RLock()
		filtered := app.msl
		app.stateMu.RUnlock()

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
				Port: server.GamePort,
				Name: server.Name,
				// Region:     server.Region,
				Players:    server.Players,
				MaxPlayers: server.MaxPlayers,
				Bots:       server.Bots,
				Map:        server.Map,
				GameTypes:  strings.Split(server.GameType, ","),
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

func onAPIGetTF2Stats(app *App) gin.HandlerFunc {
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
			localStats, errGetStats := app.db.GetLocalTF2Stats(ctx, duration)
			if errGetStats != nil {
				responseErr(ctx, http.StatusInternalServerError, nil)

				return
			}

			responseOK(ctx, http.StatusOK, fp.Reverse(localStats))
		// case "global":
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

func onAPIGetPatreonCampaigns(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tiers, errTiers := app.patreon.Tiers()
		if errTiers != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusOK, tiers)
	}
}

func onAPIGetPatreonPledges(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Only leak specific details
		// type basicPledge struct {
		//	Name      string
		//	Amount    int
		//	CreatedAt time.Time
		// }
		// var basic []basicPledge
		// for _, p := range pledges {
		//	t0 := config.Now()
		//	if p.Attributes.CreatedAt.Valid {
		//		t0 = p.Attributes.CreatedAt.Time.UTC()
		//	}
		//	basic = append(basic, basicPledge{
		//		Name:      users[p.Relationships.Patron.Data.ID].Attributes.FullName,
		//		Amount:    p.Attributes.AmountCents,
		//		CreatedAt: t0,
		//	})
		// }
		pledges, _, errPledges := app.patreon.Pledges()
		if errPledges != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		responseOK(ctx, http.StatusOK, pledges)
	}
}

func serverFromCtx(ctx *gin.Context) int {
	serverIDUntyped, ok := ctx.Get("server_id")
	if !ok {
		return 0
	}

	serverID, castOk := serverIDUntyped.(int)
	if !castOk {
		return 0
	}

	return serverID
}

func onAPIPostServerState(app *App) gin.HandlerFunc {
	type newState struct {
		Hostname       string `json:"hostname"`
		ShortName      string `json:"short_name"`
		CurrentMap     string `json:"current_map"`
		PlayersReal    int    `json:"players_real"`
		PlayersTotal   int    `json:"players_total"`
		PlayersVisible int    `json:"players_visible"`
	}

	return func(ctx *gin.Context) {
		var req newState
		if errBind := ctx.BindJSON(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		serverID := serverFromCtx(ctx)
		if serverID == 0 {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		app.stateMu.Lock()
		defer app.stateMu.Unlock()

		curState, ok := app.serverState[serverID]
		if !ok {
			var server store.Server
			if errServer := app.db.GetServer(ctx, serverID, &server); errServer != nil {
				responseErr(ctx, http.StatusNotFound, nil)

				return
			}

			curState = ServerDetails{
				ServerID:  server.ServerID,
				NameShort: server.ServerName,
				Name:      server.ServerNameLong,
				Host:      server.Address,
				Port:      server.Port,
				Enabled:   server.IsEnabled,
				Region:    server.Region,
				CC:        server.CC,
				Latitude:  server.Latitude,
				Longitude: server.Longitude,
				Reserved:  server.ReservedSlots,
			}
		}

		curState.Name = req.Hostname
		curState.Map = req.CurrentMap
		curState.PlayerCount = req.PlayersReal
		curState.MaxPlayers = req.PlayersVisible
		curState.Bots = req.PlayersTotal - req.PlayersReal
		app.serverState[serverID] = curState

		responseOK(ctx, http.StatusNoContent, "")
	}
}
