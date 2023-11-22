package app

// Server API

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type ServerAuthReq struct {
	ServerName string `json:"server_name"`
	Key        string `json:"key"`
}

type ServerAuthResp struct {
	Status bool   `json:"status"`
	Token  string `json:"token"`
}

func onSAPIPostServerAuth(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req ServerAuthReq
		if !bind(ctx, log, &req) {
			return
		}

		var server store.Server

		errGetServer := app.db.GetServerByName(ctx, req.ServerName, &server, true, false)
		if errGetServer != nil {
			responseErr(ctx, http.StatusUnauthorized, consts.ErrPermissionDenied)
			log.Warn("Failed to find server auth by name",
				zap.String("name", req.ServerName), zap.Error(errGetServer))

			return
		}

		if server.Password != req.Key {
			responseErr(ctx, http.StatusUnauthorized, consts.ErrPermissionDenied)
			log.Error("Invalid server key used",
				zap.String("server", util.SanitizeLog(req.ServerName)))

			return
		}

		accessToken, errToken := newServerToken(server.ServerID, app.conf.HTTP.CookieKey)
		if errToken != nil {
			responseErr(ctx, http.StatusUnauthorized, consts.ErrPermissionDenied)
			log.Error("Failed to create new server access token", zap.Error(errToken))

			return
		}

		server.TokenCreatedOn = time.Now()
		if errSaveServer := app.db.SaveServer(ctx, &server); errSaveServer != nil {
			responseErr(ctx, http.StatusUnauthorized, consts.ErrPermissionDenied)
			log.Error("Failed to updated server token", zap.Error(errSaveServer))

			return
		}

		ctx.JSON(http.StatusOK, ServerAuthResp{Status: true, Token: accessToken})
		log.Info("Server authenticated successfully", zap.String("server", server.ShortName))
	}
}

func onAPIGetServerAdmins(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		perms, err := app.db.GetServerPermissions(ctx)
		if err != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, perms)
	}
}

type pingReq struct {
	ServerName string        `json:"server_name"`
	Name       string        `json:"name"`
	SteamID    steamid.SID64 `json:"steam_id"`
	Reason     string        `json:"reason"`
	Client     int           `json:"client"`
}

func onAPIPostPingMod(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req pingReq
		if !bind(ctx, log, &req) {
			return
		}

		state := app.state.current()
		players := state.find(findOpts{SteamID: req.SteamID})

		if len(players) == 0 && app.conf.General.Mode != TestMode {
			log.Error("Failed to find player on /mod call")
			responseErr(ctx, http.StatusFailedDependency, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"client":  req.Client,
			"message": "Moderators have been notified",
		})

		if !app.conf.Discord.Enabled {
			return
		}

		msgEmbed := discord.
			NewEmbed("New User In-Game Report").
			SetDescription(fmt.Sprintf("%s | <@&%s>", req.Reason, app.conf.Discord.ModPingRoleID)).
			AddField("server", req.ServerName)

		app.addAuthor(ctx, msgEmbed, req.SteamID).Truncate()

		app.bot.SendPayload(discord.Payload{ChannelID: app.conf.Discord.LogChannelID, Embed: msgEmbed.MessageEmbed})
	}
}

type CheckRequest struct {
	ClientID int         `json:"client_id"`
	SteamID  steamid.SID `json:"steam_id"`
	IP       net.IP      `json:"ip"`
	Name     string      `json:"name,omitempty"`
}

type CheckResponse struct {
	ClientID        int              `json:"client_id"`
	SteamID         steamid.SID      `json:"steam_id"`
	BanType         store.BanType    `json:"ban_type"`
	PermissionLevel consts.Privilege `json:"permission_level"`
	Msg             string           `json:"msg"`
}

func onAPIPostServerCheck(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var request CheckRequest
		if errBind := ctx.BindJSON(&request); errBind != nil { // we don't currently use bind() for server api
			ctx.JSON(http.StatusInternalServerError, CheckResponse{
				BanType: store.Unknown,
				Msg:     "Error determining state",
			})

			return
		}

		log.Debug("Player connecting",
			zap.String("ip", request.IP.String()),
			zap.Int64("sid64", steamid.SIDToSID64(request.SteamID).Int64()),
			zap.String("name", request.Name))

		resp := CheckResponse{
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
			ctx.JSON(http.StatusBadRequest, resp)

			return
		}

		if parentID, banned := app.IsGroupBanned(steamID); banned {
			resp.BanType = store.Banned

			if parentID >= steamid.BaseGID {
				resp.Msg = fmt.Sprintf("Group Banned (gid: %d)", parentID)
			} else {
				resp.Msg = fmt.Sprintf("Banned (sid: %d)", parentID)
			}

			ctx.JSON(http.StatusOK, resp)
			log.Info("Player dropped", zap.String("drop_type", "group"),
				zap.Int64("sid64", steamID.Int64()))

			return
		}

		var person store.Person
		if errPerson := app.PersonBySID(responseCtx, steamID, &person); errPerson != nil {
			ctx.JSON(http.StatusInternalServerError, CheckResponse{
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
			CreatedOn:   time.Now(),
		}); errAddHist != nil {
			log.Error("Failed to add conn history", zap.Error(errAddHist))
		}

		// Check IP first
		banNet, errGetBanNet := app.db.GetBanNetByAddress(responseCtx, request.IP)
		if errGetBanNet != nil {
			ctx.JSON(http.StatusInternalServerError, CheckResponse{
				BanType: store.Unknown,
				Msg:     "Error determining state",
			})
			log.Error("Could not get bannedPerson net results", zap.Error(errGetBanNet))

			return
		}

		if len(banNet) > 0 {
			resp.BanType = store.Banned
			resp.Msg = "Banned"

			ctx.JSON(http.StatusOK, resp)

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
				resp.Msg = asnBan.Reason.String()
				ctx.JSON(http.StatusOK, resp)
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
				ctx.JSON(http.StatusOK, resp)

				return
			}

			resp.Msg = "Error determining state"

			ctx.JSON(http.StatusInternalServerError, resp)

			return
		}

		resp.BanType = bannedPerson.BanType

		var reason string

		switch {
		case bannedPerson.Reason == store.Custom && bannedPerson.ReasonText != "":
			reason = bannedPerson.ReasonText
		case bannedPerson.Reason == store.Custom && bannedPerson.ReasonText == "":
			reason = "Banned"
		default:
			reason = bannedPerson.Reason.String()
		}

		resp.Msg = fmt.Sprintf("Banned\nReason: %s\nAppeal: %s\nRemaining: %s", reason, app.ExtURL(bannedPerson.BanSteam),
			time.Until(bannedPerson.ValidUntil).Round(time.Minute).String())

		ctx.JSON(http.StatusOK, resp)

		if resp.BanType == store.NoComm {
			log.Info("Player muted", zap.Int64("sid64", steamID.Int64()))
		} else if resp.BanType == store.Banned {
			log.Info("Player dropped", zap.String("drop_type", "steam"),
				zap.Int64("sid64", steamID.Int64()))
		}
	}
}

type demoUploadContent struct {
	demoName string
	demoRaw  []byte
	jsonRaw  []byte
}

func readDemoZipContainer(log *zap.Logger, demoContainerZip *multipart.FileHeader) (*demoUploadContent, error) {
	zipHandle, errZipHandle := demoContainerZip.Open()
	if errZipHandle != nil {
		return nil, errors.Wrap(errZipHandle, "Failed to open zip container")
	}

	zipReader, errContainer := zip.NewReader(zipHandle, demoContainerZip.Size)
	if errContainer != nil {
		return nil, errors.Wrap(errContainer, "Cannot open zip reader")
	}

	var (
		rawDemoBuffer = &bytes.Buffer{}
		rawJSONBuffer = &bytes.Buffer{}
		rawDemoWriter = bufio.NewWriter(rawDemoBuffer)
		rawJSONWriter = bufio.NewWriter(rawJSONBuffer)
		demoName      string
	)

	for _, file := range zipReader.File {
		reader, errReader := file.Open()
		if errReader != nil {
			return nil, errors.Wrap(errReader, "Cannot open zip file")
		}

		var (
			err  error
			size int64
		)

		if file.Name == "stats.json" {
			log.Info("Got stats", zap.String("stats", file.Name))

			size, err = io.CopyN(rawJSONWriter, reader, file.FileInfo().Size())
		} else {
			log.Info("Got demo", zap.String("demo", file.Name))

			demoName = file.Name

			size, err = io.CopyN(rawDemoWriter, reader, file.FileInfo().Size())
		}

		if err != nil {
			return nil, errors.Wrap(errContainer, "Failed to read zip file content")
		}

		log.Debug("Copied contents", zap.Int64("size", size))

		if errClose := reader.Close(); errClose != nil {
			return nil, errors.Wrap(errContainer, "Failed to close reader")
		}
	}

	return &demoUploadContent{
		demoName: demoName,
		demoRaw:  rawDemoBuffer.Bytes(),
		jsonRaw:  rawJSONBuffer.Bytes(),
	}, nil
}

type demoForm struct {
	// Stats      string `form:"stats"` // {"76561198084134025": {"score": 0, "deaths": 0, "score_total": 0}}
	ServerName string `form:"server_name"`
	MapName    string `form:"map_name"`
}

func onAPIPostDemo(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var form demoForm
		if err := ctx.ShouldBind(&form); err != nil {
			ctx.String(http.StatusBadRequest, "bad request: %v", err)

			return
		}

		if form.ServerName == "" || form.MapName == "" {
			log.Error("Missing demo params",
				zap.String("server_name", util.SanitizeLog(form.ServerName)),
				zap.String("map_name", util.SanitizeLog(form.MapName)))
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		demoContainerZip, errDemoFile := ctx.FormFile("demo")
		if errDemoFile != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		result, errZip := readDemoZipContainer(log, demoContainerZip)
		if errZip != nil {
			log.Error("Could not read demo zip", zap.Error(errZip))
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		var server store.Server
		if errGetServer := app.db.GetServerByName(ctx, form.ServerName, &server, false, false); errGetServer != nil {
			log.Error("Server not found", zap.String("server", util.SanitizeLog(form.ServerName)))
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		var metaData store.DemoMetaData
		if errJSON := json.Unmarshal(result.jsonRaw, &metaData); errJSON != nil {
			log.Error("Failed to unmarshal meta data", zap.Error(errJSON))
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		// Convert string based sid to int64
		intStats := map[steamid.SID64]store.DemoPlayerStats{}

		for steamID, PlayerStat := range metaData.Scores {
			sid64, errSid := steamid.SID64FromString(steamID)
			if errSid != nil {
				log.Error("Failed to parse score steam id", zap.Error(errSid))

				continue
			}

			intStats[sid64] = PlayerStat
		}

		asset, errAsset := store.NewAsset(result.demoRaw, app.conf.S3.BucketDemo, result.demoName)
		if errAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save asset"))

			return
		}

		if errPut := app.assetStore.Put(ctx, app.conf.S3.BucketDemo, asset.Name,
			bytes.NewReader(result.demoRaw), asset.Size, asset.MimeType); errPut != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save media"))

			log.Error("Failed to save user media to s3 backend", zap.Error(errPut))

			return
		}

		if errSaveAsset := app.db.SaveAsset(ctx, &asset); errSaveAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save asset"))

			log.Error("Failed to save user asset to s3 backend", zap.Error(errSaveAsset))
		}

		newDemo := store.DemoFile{
			ServerID:  server.ServerID,
			Title:     asset.Name,
			CreatedOn: time.Now(),
			MapName:   form.MapName,
			Stats:     intStats,
			AssetID:   asset.AssetID,
		}

		if errSave := app.db.SaveDemo(ctx, &newDemo); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save demo", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, gin.H{"demo_id": newDemo.DemoID})
	}
}

type apiCreateReportReq struct {
	SourceID        store.StringSID `json:"source_id"`
	TargetID        store.StringSID `json:"target_id"`
	Description     string          `json:"description"`
	Reason          store.Reason    `json:"reason"`
	ReasonText      string          `json:"reason_text"`
	DemoName        string          `json:"demo_name"`
	DemoTick        int             `json:"demo_tick"`
	PersonMessageID int64           `json:"person_message_id"`
}

type apiBanRequest struct {
	SourceID       store.StringSID `json:"source_id"`
	TargetID       store.StringSID `json:"target_id"`
	Duration       string          `json:"duration"`
	ValidUntil     time.Time       `json:"valid_until"`
	BanType        store.BanType   `json:"ban_type"`
	Reason         store.Reason    `json:"reason"`
	ReasonText     string          `json:"reason_text"`
	Note           string          `json:"note"`
	ReportID       int64           `json:"report_id"`
	DemoName       string          `json:"demo_name"`
	DemoTick       int             `json:"demo_tick"`
	IncludeFriends bool            `json:"include_friends"`
}

func onAPIPostBanSteamCreate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var (
			origin   = store.Web
			sid      = currentUserProfile(ctx).SteamID
			sourceID = store.StringSID(sid.String())
		)

		// srcds sourced bans provide a source_id to id the admin
		if req.SourceID != "" {
			sourceID = req.SourceID
			origin = store.InGame
		}

		duration, errDuration := calcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var banSteam store.BanSteam
		if errBanSteam := store.NewBanSteam(ctx,
			sourceID,
			req.TargetID,
			duration,
			req.Reason,
			req.ReasonText,
			req.Note,
			origin,
			req.ReportID,
			req.BanType,
			req.IncludeFriends,
			&banSteam,
		); errBanSteam != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if errBan := app.BanSteam(ctx, &banSteam); errBan != nil {
			log.Error("Failed to ban steam profile",
				zap.Error(errBan), zap.Int64("target_id", banSteam.TargetID.Int64()))

			if errors.Is(errBan, store.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, consts.ErrDuplicate)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save new steam ban", zap.Error(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, banSteam)
	}
}

func onAPIPostReportCreate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		var req apiCreateReportReq
		if !bind(ctx, log, &req) {
			return
		}

		// Server initiated requests will have a sourceID set by the server
		// Web based reports the source should not be set, the reporter will be taken from the
		// current session information instead
		if req.SourceID == "" {
			req.SourceID = store.StringSID(currentUser.SteamID.String())
		}

		sourceID, errSourceID := req.SourceID.SID64(ctx)
		if errSourceID != nil {
			responseErr(ctx, http.StatusBadRequest, errors.New("Failed to resolve author steam id"))
			log.Error("Invalid steam_id", zap.Error(errSourceID))

			return
		}

		targetID, errTargetID := req.TargetID.SID64(ctx)
		if errTargetID != nil {
			responseErr(ctx, http.StatusBadRequest, errors.New("Failed to resolve target steam id"))
			log.Error("Invalid target_id", zap.Error(errTargetID))

			return
		}

		if sourceID == targetID {
			responseErr(ctx, http.StatusConflict, errors.New("Cannot report yourself"))

			return
		}

		var personSource store.Person
		if errCreatePerson := app.PersonBySID(ctx, sourceID, &personSource); errCreatePerson != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Could not load player profile", zap.Error(errCreatePerson))

			return
		}

		var personTarget store.Person
		if errCreatePerson := app.PersonBySID(ctx, targetID, &personTarget); errCreatePerson != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Could not load player profile", zap.Error(errCreatePerson))

			return
		}

		// Ensure the user doesn't already have an open report against the user
		var existing store.Report
		if errReports := app.db.GetReportBySteamID(ctx, personSource.SteamID, targetID, &existing); errReports != nil {
			if !errors.Is(errReports, store.ErrNoResult) {
				log.Error("Failed to query reports by steam id", zap.Error(errReports))
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

				return
			}
		}

		if existing.ReportID > 0 {
			responseErr(ctx, http.StatusConflict, errors.New("Must resolve existing report for user before creating another"))

			return
		}

		// TODO encapsulate all operations in single tx
		report := store.NewReport()
		report.SourceID = sourceID
		report.ReportStatus = store.Opened
		report.Description = req.Description
		report.TargetID = targetID
		report.Reason = req.Reason
		report.ReasonText = req.ReasonText
		parts := strings.Split(req.DemoName, "/")
		report.DemoName = parts[len(parts)-1]
		report.DemoTick = req.DemoTick
		report.PersonMessageID = req.PersonMessageID

		if errReportSave := app.db.SaveReport(ctx, &report); errReportSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save report", zap.Error(errReportSave))

			return
		}

		ctx.JSON(http.StatusCreated, report)

		log.Info("New report created successfully", zap.Int64("report_id", report.ReportID))

		if !app.conf.Discord.Enabled {
			return
		}

		msgEmbed := discord.
			NewEmbed("New User Report Created").
			SetDescription(report.Description).
			SetColor(app.bot.Colour.Success).
			SetURL(app.ExtURL(report))

		app.addAuthorUserProfile(msgEmbed, currentUser)

		name := personSource.PersonaName

		if name == "" {
			name = report.TargetID.String()
		}

		msgEmbed.AddField("Subject", name)
		msgEmbed.AddField("Reason", report.Reason.String())

		if report.ReasonText != "" {
			msgEmbed.AddField("Custom Reason", report.ReasonText)
		}

		if report.DemoName != "" {
			msgEmbed.AddField("Demo", app.ExtURLRaw("/demos/name/%s", report.DemoName))
			msgEmbed.AddField("Demo Tick", fmt.Sprintf("%d", report.DemoTick))
		}

		discord.AddFieldsSteamID(msgEmbed, report.TargetID)

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.LogChannelID,
			Embed:     msgEmbed.Truncate().MessageEmbed,
		})
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
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req partialStateUpdate
		if !bind(ctx, log, &req) {
			return
		}

		serverID := serverFromCtx(ctx) // TODO use generic func for int
		if serverID == 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		if errUpdate := app.state.updateState(serverID, req); errUpdate != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.AbortWithStatus(http.StatusNoContent)
	}
}
