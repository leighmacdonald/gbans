package app

// Server API

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/demoparser"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type ServerAuthReq struct {
	Key string `json:"key"`
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

		errGetServer := app.db.GetServerByPassword(ctx, req.Key, &server, true, false)
		if errGetServer != nil {
			responseErr(ctx, http.StatusUnauthorized, consts.ErrPermissionDenied)
			log.Warn("Failed to find server auth by password", zap.Error(errGetServer))

			return
		}

		if server.Password != req.Key {
			responseErr(ctx, http.StatusUnauthorized, consts.ErrPermissionDenied)
			log.Error("Invalid server key used")

			return
		}

		accessToken, errToken := newServerToken(server.ServerID, app.config().HTTP.CookieKey)
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

		conf := app.config()
		state := app.state.current()
		players := state.find(findOpts{SteamID: req.SteamID})

		if len(players) == 0 && conf.General.Mode != config.TestMode {
			log.Error("Failed to find player on /mod call")
			responseErr(ctx, http.StatusFailedDependency, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"client":  req.Client,
			"message": "Moderators have been notified",
		})

		if !conf.Discord.Enabled {
			return
		}

		msgEmbed := discord.NewEmbed(conf, "New User In-Game Report")
		msgEmbed.
			Embed().
			SetDescription(fmt.Sprintf("%s | <@&%s>", req.Reason, conf.Discord.ModPingRoleID)).
			AddField("server", req.ServerName)

		var author store.Person
		if err := app.db.GetOrCreatePersonBySteamID(ctx, req.SteamID, &author); err != nil {
			log.Error("Failed to load user", zap.Error(err))
		} else {
			msgEmbed.AddAuthorPerson(author).Embed().Truncate()
		}

		app.discord.SendPayload(discord.Payload{
			ChannelID: conf.Discord.LogChannelID, Embed: msgEmbed.Message(),
		})
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

// onAPIPostServerCheck takes care of checking if the player connecting to the server is
// allowed to connect, or otherwise has restrictions such as being mutes. It performs
// the following actions/checks in order:
//
// - Add ip to connection history
// - Check if is a part of a steam group ban
// - Check if ip belongs to banned 3rd party CIDR block, like VPNs.
// - Check if ip belongs to one or more local CIDR bans
// - Check if ip belongs to a banned AS Number range
// - Check if steam_id is part of a local steam ban
// - Check if player is connecting from a IP that belongs to a banned player
//
// Returns a ok/muted/banned status for the player.
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

		steamID := steamid.SIDToSID64(request.SteamID)
		if !steamID.Valid() {
			resp.Msg = "Invalid steam id"
			ctx.JSON(http.StatusBadRequest, resp)

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

		if errAddHist := app.db.AddConnectionHistory(ctx, &store.PersonConnection{
			IPAddr:      request.IP,
			SteamID:     steamID,
			PersonaName: request.Name,
			CreatedOn:   time.Now(),
			ServerID:    ctx.GetInt("server_id"),
		}); errAddHist != nil {
			log.Error("Failed to add conn history", zap.Error(errAddHist))
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

		resp.PermissionLevel = person.PermissionLevel

		if cidrBanned, source := app.netBlock.IsMatch(request.IP); cidrBanned {
			resp.BanType = store.Network
			resp.Msg = "Network Range Banned.\nIf you using a VPN try disabling it"

			ctx.JSON(http.StatusOK, resp)
			log.Info("Player network blocked", zap.Int64("sid64", steamID.Int64()),
				zap.String("source", source), zap.String("ip", request.IP.String()))

			return
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
				if app.isOnIPWithBan(ctx, steamid.SIDToSID64(request.SteamID), request.IP) {
					log.Info("Player connected from IP of a banned player",
						zap.String("steam_id", steamid.SIDToSID64(request.SteamID).String()),
						zap.String("ip", request.IP.String()))

					resp.BanType = store.Banned
					resp.Msg = "Ban evasion. Previous ban updated to permanent if not already permanent"

					ctx.JSON(http.StatusOK, resp)

					return
				}

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

		conf := app.config()

		resp.Msg = fmt.Sprintf("Banned\nReason: %s\nAppeal: %s\nRemaining: %s", reason, conf.ExtURL(bannedPerson.BanSteam),
			time.Until(bannedPerson.ValidUntil).Round(time.Minute).String())

		ctx.JSON(http.StatusOK, resp)

		//goland:noinspection GoSwitchMissingCasesForIotaConsts
		switch resp.BanType {
		case store.NoComm:
			log.Info("Player muted", zap.Int64("sid64", steamID.Int64()))
		case store.Banned:
			log.Info("Player dropped", zap.String("drop_type", "steam"),
				zap.Int64("sid64", steamID.Int64()))
		}
	}
}

func onAPIPostDemo(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		serverID := serverFromCtx(ctx)
		if serverID <= 0 {
			responseErr(ctx, http.StatusNotFound, consts.ErrBadRequest)

			return
		}

		var server store.Server
		if errGetServer := app.db.GetServer(ctx, serverID, &server); errGetServer != nil {
			log.Error("Server not found", zap.Int("server_id", serverID))
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		demoFormFile, errDemoFile := ctx.FormFile("demo")
		if errDemoFile != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		demoHandle, errDemoHandle := demoFormFile.Open()
		if errDemoHandle != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		demoContent, errRead := io.ReadAll(demoHandle)
		if errRead != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		dir, errDir := os.MkdirTemp("", "gbans-demo")
		if errDir != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		defer func() {
			if err := os.RemoveAll(dir); err != nil {
				log.Error("Failed to cleanup temp demo path", zap.Error(err))
			}
		}()

		namePartsAll := strings.Split(demoFormFile.Filename, "-")

		var mapName string

		if strings.Contains(demoFormFile.Filename, "workshop-") {
			// 20231221-042605-workshop-cp_overgrown_rc8-ugc503939302.dem
			mapName = namePartsAll[3]
		} else {
			// 20231112-063943-koth_harvest_final.dem
			nameParts := strings.Split(namePartsAll[2], ".")
			mapName = nameParts[0]
		}

		tempPath := filepath.Join(dir, demoFormFile.Filename)

		localFile, errLocalFile := os.Create(tempPath)
		if errLocalFile != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if _, err := localFile.Write(demoContent); err != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		_ = localFile.Close()

		var demoInfo demoparser.DemoInfo
		if errParse := demoparser.Parse(ctx, tempPath, &demoInfo); errParse != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		intStats := map[steamid.SID64]gin.H{}

		for _, steamID := range demoInfo.SteamIDs() {
			intStats[steamID] = gin.H{}
		}

		conf := app.config()

		asset, errAsset := store.NewAsset(demoContent, conf.S3.BucketDemo, demoFormFile.Filename)
		if errAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save asset"))

			return
		}

		if errPut := app.assetStore.Put(ctx, conf.S3.BucketDemo, asset.Name,
			bytes.NewReader(demoContent), asset.Size, asset.MimeType); errPut != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save media"))

			log.Error("Failed to save user media to s3 backend", zap.Error(errPut))

			return
		}

		if errSaveAsset := app.db.SaveAsset(ctx, &asset); errSaveAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save asset"))

			log.Error("Failed to save user asset to s3 backend", zap.Error(errSaveAsset))
		}

		newDemo := store.DemoFile{
			ServerID:  serverID,
			Title:     asset.Name,
			CreatedOn: time.Now(),
			MapName:   mapName,
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

		duration, errDuration := util.CalcDuration(req.Duration, req.ValidUntil)
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

		if req.Description == "" || len(req.Description) < 10 {
			responseErr(ctx, http.StatusBadRequest, errors.New("Description too short"))

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

		conf := app.config()

		if !conf.Discord.Enabled {
			return
		}

		msgEmbed := discord.NewEmbed(conf, "New User Report Created")
		msgEmbed.
			Embed().
			SetDescription(report.Description).
			SetColor(conf.Discord.ColourSuccess).
			SetURL(conf.ExtURL(report))

		msgEmbed.AddAuthorUserProfile(currentUser)

		name := personSource.PersonaName

		if name == "" {
			name = report.TargetID.String()
		}

		msgEmbed.
			Embed().
			AddField("Subject", name).
			AddField("Reason", report.Reason.String())

		if report.ReasonText != "" {
			msgEmbed.Embed().AddField("Custom Reason", report.ReasonText)
		}

		if report.DemoName != "" {
			msgEmbed.Embed().AddField("Demo", conf.ExtURLRaw("/demos/name/%s", report.DemoName))
			msgEmbed.Embed().AddField("Demo Tick", fmt.Sprintf("%d", report.DemoTick))
		}

		msgEmbed.AddFieldsSteamID(report.TargetID).Embed().Truncate()

		app.discord.SendPayload(discord.Payload{
			ChannelID: conf.Discord.LogChannelID,
			Embed:     msgEmbed.Message(),
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
