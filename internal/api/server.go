package api

// ServerStore API

import (
	"bytes"
	"context"
	"errors"
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
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/demoparser"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

var (
	errSelfReport   = errors.New("cannot self report")
	errReportExists = errors.New("cannot create report while existing report open")
)

type ServerAuthReq struct {
	Key string `json:"key"`
}

type ServerAuthResp struct {
	Status bool   `json:"status"`
	Token  string `json:"token"`
}

func onSAPIPostServerAuth(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req ServerAuthReq
		if !bind(ctx, log, &req) {
			return
		}

		var server model.Server

		errGetServer := env.Store().GetServerByPassword(ctx, req.Key, &server, true, false)
		if errGetServer != nil {
			responseErr(ctx, http.StatusUnauthorized, errPermissionDenied)
			log.Warn("Failed to find server auth by password", zap.Error(errGetServer))

			return
		}

		if server.Password != req.Key {
			responseErr(ctx, http.StatusUnauthorized, errPermissionDenied)
			log.Error("Invalid server key used")

			return
		}

		accessToken, errToken := newServerToken(server.ServerID, env.Config().HTTP.CookieKey)
		if errToken != nil {
			responseErr(ctx, http.StatusUnauthorized, errPermissionDenied)
			log.Error("Failed to create new server access token", zap.Error(errToken))

			return
		}

		server.TokenCreatedOn = time.Now()
		if errSaveServer := env.Store().SaveServer(ctx, &server); errSaveServer != nil {
			responseErr(ctx, http.StatusUnauthorized, errPermissionDenied)
			log.Error("Failed to updated server token", zap.Error(errSaveServer))

			return
		}

		ctx.JSON(http.StatusOK, ServerAuthResp{Status: true, Token: accessToken})
		log.Info("ServerStore authenticated successfully", zap.String("server", server.ShortName))
	}
}

func onAPIGetServerAdmins(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		perms, err := env.Store().GetServerPermissions(ctx)
		if err != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

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

func onAPIPostPingMod(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req pingReq
		if !bind(ctx, log, &req) {
			return
		}

		conf := env.Config()
		currentState := env.State()
		players := currentState.Find("", req.SteamID, nil, nil)

		if len(players) == 0 && conf.General.Mode != config.TestMode {
			log.Error("Failed to find player on /mod call")
			responseErr(ctx, http.StatusFailedDependency, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, gin.H{"client": req.Client, "message": "Moderators have been notified"})

		if !conf.Discord.Enabled {
			return
		}

		var author model.Person
		if err := env.Store().GetOrCreatePersonBySteamID(ctx, req.SteamID, &author); err != nil {
			log.Error("Failed to load user", zap.Error(err))

			return
		}

		env.SendPayload(conf.Discord.LogChannelID,
			discord.PingModMessage(author, conf.ExtURL(author), req.Reason, req.ServerName, conf.Discord.ModPingRoleID))
	}
}

type CheckRequest struct {
	ClientID int         `json:"client_id"`
	SteamID  steamid.SID `json:"steam_id"`
	IP       net.IP      `json:"ip"`
	Name     string      `json:"name,omitempty"`
}

type CheckResponse struct {
	ClientID        int             `json:"client_id"`
	SteamID         steamid.SID     `json:"steam_id"`
	BanType         model.BanType   `json:"ban_type"`
	PermissionLevel model.Privilege `json:"permission_level"`
	Msg             string          `json:"msg"`
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
// - Check if player is connecting from an IP that belongs to a banned player
//
// Returns an ok/muted/banned status for the player.
func onAPIPostServerCheck(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var request CheckRequest
		if errBind := ctx.BindJSON(&request); errBind != nil { // we don't currently use bind() for server api
			ctx.JSON(http.StatusInternalServerError, CheckResponse{
				BanType: model.Unknown,
				Msg:     "Error determining state",
			})

			return
		}

		log.Debug("Player connecting",
			zap.String("ip", request.IP.String()),
			zap.Int64("sid64", steamid.SIDToSID64(request.SteamID).Int64()),
			zap.String("name", request.Name))

		resp := CheckResponse{ClientID: request.ClientID, SteamID: request.SteamID, BanType: model.Unknown, Msg: ""}

		responseCtx, cancelResponse := context.WithTimeout(ctx, time.Second*15)
		defer cancelResponse()

		steamID := steamid.SIDToSID64(request.SteamID)
		if !steamID.Valid() {
			resp.Msg = "Invalid steam id"
			ctx.JSON(http.StatusBadRequest, resp)

			return
		}

		var person model.Person

		if errPerson := env.Store().GetOrCreatePersonBySteamID(responseCtx, steamID, &person); errPerson != nil {
			log.Error("Failed to create connecting player", zap.Error(errPerson))
		} else if person.Expired() {
			if err := thirdparty.UpdatePlayerSummary(ctx, &person); err != nil {
				log.Error("Failed to update connecting player", zap.Error(err))
			} else {
				if errSave := env.Store().SavePerson(ctx, &person); errSave != nil {
					log.Error("Failed to save connecting player summary", zap.Error(err))
				}
			}
		}

		if errAddHist := env.Store().AddConnectionHistory(ctx, &model.PersonConnection{
			IPAddr:      request.IP,
			SteamID:     steamID,
			PersonaName: request.Name,
			CreatedOn:   time.Now(),
			ServerID:    ctx.GetInt("server_id"),
		}); errAddHist != nil {
			log.Error("Failed to add conn history", zap.Error(errAddHist))
		}

		resp.PermissionLevel = person.PermissionLevel

		if checkGroupBan(ctx, log, env, steamID, &resp) || checkFriendBan(ctx, log, env, steamID, &resp) {
			return
		}

		if checkNetBlockBan(ctx, log, env, steamID, request.IP, &resp) {
			return
		}

		if checkIPBan(ctx, log, env, steamID, request.IP, responseCtx, &resp) {
			return
		}

		if checkASN(ctx, log, env, steamID, request.IP, responseCtx, &resp) {
			return
		}

		bannedPerson := model.NewBannedPerson()
		if errGetBan := env.Store().GetBanBySteamID(responseCtx, steamID, &bannedPerson, false); errGetBan != nil {
			if errors.Is(errGetBan, errs.ErrNoResult) {
				if IsOnIPWithBan(ctx, env, steamid.SIDToSID64(request.SteamID), request.IP) {
					log.Info("Player connected from IP of a banned player",
						zap.String("steam_id", steamid.SIDToSID64(request.SteamID).String()),
						zap.String("ip", request.IP.String()))

					resp.BanType = model.Banned
					resp.Msg = "Ban evasion. Previous ban updated to permanent if not already permanent"

					ctx.JSON(http.StatusOK, resp)

					return
				}

				// No ban, exit early
				resp.BanType = model.OK
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
		case bannedPerson.Reason == model.Custom && bannedPerson.ReasonText != "":
			reason = bannedPerson.ReasonText
		case bannedPerson.Reason == model.Custom && bannedPerson.ReasonText == "":
			reason = "Banned"
		default:
			reason = bannedPerson.Reason.String()
		}

		conf := env.Config()

		resp.Msg = fmt.Sprintf("Banned\nReason: %s\nAppeal: %s\nRemaining: %s", reason, conf.ExtURL(bannedPerson.BanSteam),
			time.Until(bannedPerson.ValidUntil).Round(time.Minute).String())

		ctx.JSON(http.StatusOK, resp)

		//goland:noinspection GoSwitchMissingCasesForIotaConsts
		switch resp.BanType {
		case model.NoComm:
			log.Info("Player muted", zap.Int64("sid64", steamID.Int64()))
		case model.Banned:
			log.Info("Player dropped", zap.String("drop_type", "steam"),
				zap.Int64("sid64", steamID.Int64()))
		}
	}
}

func checkASN(ctx *gin.Context, log *zap.Logger, env Env, steamID steamid.SID64, addr net.IP, responseCtx context.Context, resp *CheckResponse) bool {
	var asnRecord ip2location.ASNRecord

	errASN := env.Store().GetASNRecordByIP(responseCtx, addr, &asnRecord)
	if errASN == nil {
		var asnBan model.BanASN
		if errASNBan := env.Store().GetBanASN(responseCtx, int64(asnRecord.ASNum), &asnBan); errASNBan != nil {
			if !errors.Is(errASNBan, errs.ErrNoResult) {
				log.Error("Failed to fetch asn bannedPerson", zap.Error(errASNBan))
			}
		} else {
			resp.BanType = model.Banned
			resp.Msg = asnBan.Reason.String()
			ctx.JSON(http.StatusOK, resp)
			log.Info("Player dropped", zap.String("drop_type", "asn"),
				zap.Int64("sid64", steamID.Int64()))

			return true
		}
	}

	return false
}

func checkIPBan(ctx *gin.Context, log *zap.Logger, env Env, steamID steamid.SID64, addr net.IP, responseCtx context.Context, resp *CheckResponse) bool {
	// Check IP first
	banNet, errGetBanNet := env.Store().GetBanNetByAddress(responseCtx, addr)
	if errGetBanNet != nil {
		ctx.JSON(http.StatusInternalServerError, CheckResponse{
			BanType: model.Unknown,
			Msg:     "Error determining state",
		})
		log.Error("Could not get bannedPerson net results", zap.Error(errGetBanNet))

		return true
	}

	if len(banNet) > 0 {
		resp.BanType = model.Banned
		resp.Msg = "Banned"

		ctx.JSON(http.StatusOK, resp)

		log.Info("Player dropped", zap.String("drop_type", "cidr"),
			zap.Int64("sid64", steamID.Int64()))

		return true
	}

	return false
}

func checkNetBlockBan(ctx *gin.Context, log *zap.Logger, env Env, steamID steamid.SID64, addr net.IP, resp *CheckResponse) bool {
	if cidrBanned, source := env.NetBlocks().IsMatch(addr); cidrBanned {
		resp.BanType = model.Network
		resp.Msg = "Network Range Banned.\nIf you using a VPN try disabling it"

		ctx.JSON(http.StatusOK, resp)
		log.Info("Player network blocked", zap.Int64("sid64", steamID.Int64()),
			zap.String("source", source), zap.String("ip", addr.String()))

		return true
	}

	return false
}

func checkGroupBan(ctx *gin.Context, log *zap.Logger, env Env, steamID steamid.SID64, resp *CheckResponse) bool {
	if groupID, banned := env.Groups().IsMember(steamID); banned {
		resp.BanType = model.Banned
		resp.Msg = fmt.Sprintf("Group Banned (gid: %d)", groupID.Int64())

		ctx.JSON(http.StatusOK, resp)
		log.Info("Player dropped", zap.String("drop_type", "group"),
			zap.Int64("sid64", steamID.Int64()))

		return true
	}

	return false
}

func checkFriendBan(ctx *gin.Context, log *zap.Logger, env Env, steamID steamid.SID64, resp *CheckResponse) bool {
	if parentFriendID, banned := env.Groups().IsMember(steamID); banned {
		resp.BanType = model.Banned

		resp.Msg = fmt.Sprintf("Banned (sid: %d)", parentFriendID.Int64())

		ctx.JSON(http.StatusOK, resp)
		log.Info("Player dropped", zap.String("drop_type", "friend"),
			zap.Int64("sid64", steamID.Int64()))

		return true
	}

	return false
}

// IsOnIPWithBan checks if the address matches an existing user who is currently banned already. This
// function will always fail-open and allow players in if an error occurs.
func IsOnIPWithBan(ctx context.Context, env Env, steamID steamid.SID64, address net.IP) bool {
	existing := model.NewBannedPerson()
	if errMatch := env.Store().GetBanByLastIP(ctx, address, &existing, false); errMatch != nil {
		if errors.Is(errMatch, errs.ErrNoResult) {
			return false
		}

		env.Log().Error("Could not load player by ip", zap.Error(errMatch))

		return false
	}

	duration, errDuration := util.ParseUserStringDuration("10y")
	if errDuration != nil {
		env.Log().Error("Could not parse ban duration", zap.Error(errDuration))

		return false
	}

	existing.BanSteam.ValidUntil = time.Now().Add(duration)

	if errSave := env.Store().SaveBan(ctx, &existing.BanSteam); errSave != nil {
		env.Log().Error("Could not update previous ban.", zap.Error(errSave))

		return false
	}

	var newBan model.BanSteam
	if errNewBan := model.NewBanSteam(ctx,
		model.StringSID(env.Config().General.Owner.String()),
		model.StringSID(steamID.String()), duration, model.Evading, model.Evading.String(),
		"Connecting from same IP as banned player", model.System,
		0, model.Banned, false, &newBan); errNewBan != nil {
		env.Log().Error("Could not create evade ban", zap.Error(errDuration))

		return false
	}

	if errSave := env.BanSteam(ctx, &newBan); errSave != nil {
		env.Log().Error("Could not save evade ban", zap.Error(errSave))

		return false
	}

	return true
}

func onAPIPostDemo(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		serverID := serverFromCtx(ctx)
		if serverID <= 0 {
			responseErr(ctx, http.StatusNotFound, errBadRequest)

			return
		}

		var server model.Server
		if errGetServer := env.Store().GetServer(ctx, serverID, &server); errGetServer != nil {
			log.Error("ServerStore not found", zap.Int("server_id", serverID))
			responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		demoFormFile, errDemoFile := ctx.FormFile("demo")
		if errDemoFile != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		demoHandle, errDemoHandle := demoFormFile.Open()
		if errDemoHandle != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		demoContent, errRead := io.ReadAll(demoHandle)
		if errRead != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		dir, errDir := os.MkdirTemp("", "gbans-demo")
		if errDir != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

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
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		if _, err := localFile.Write(demoContent); err != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		_ = localFile.Close()

		var demoInfo demoparser.DemoInfo
		if errParse := demoparser.Parse(ctx, tempPath, &demoInfo); errParse != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		intStats := map[steamid.SID64]gin.H{}

		for _, steamID := range demoInfo.SteamIDs() {
			intStats[steamID] = gin.H{}
		}

		conf := env.Config()

		asset, errAsset := model.NewAsset(demoContent, conf.S3.BucketDemo, demoFormFile.Filename)
		if errAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errAssetCreateFailed)

			return
		}

		if errPut := env.Assets().Put(ctx, conf.S3.BucketDemo, asset.Name,
			bytes.NewReader(demoContent), asset.Size, asset.MimeType); errPut != nil {
			responseErr(ctx, http.StatusInternalServerError, errAssetPut)

			log.Error("Failed to save user media to s3 backend", zap.Error(errPut))

			return
		}

		if errSaveAsset := env.Store().SaveAsset(ctx, &asset); errSaveAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errAssetSave)

			log.Error("Failed to save user asset to s3 backend", zap.Error(errSaveAsset))
		}

		newDemo := model.DemoFile{
			ServerID:  serverID,
			Title:     asset.Name,
			CreatedOn: time.Now(),
			MapName:   mapName,
			Stats:     intStats,
			AssetID:   asset.AssetID,
		}

		if errSave := env.Store().SaveDemo(ctx, &newDemo); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to save demo", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, gin.H{"demo_id": newDemo.DemoID})
	}
}

type apiCreateReportReq struct {
	SourceID        model.StringSID `json:"source_id"`
	TargetID        model.StringSID `json:"target_id"`
	Description     string          `json:"description"`
	Reason          model.Reason    `json:"reason"`
	ReasonText      string          `json:"reason_text"`
	DemoName        string          `json:"demo_name"`
	DemoTick        int             `json:"demo_tick"`
	PersonMessageID int64           `json:"person_message_id"`
}

type apiBanRequest struct {
	SourceID       model.StringSID `json:"source_id"`
	TargetID       model.StringSID `json:"target_id"`
	Duration       string          `json:"duration"`
	ValidUntil     time.Time       `json:"valid_until"`
	BanType        model.BanType   `json:"ban_type"`
	Reason         model.Reason    `json:"reason"`
	ReasonText     string          `json:"reason_text"`
	Note           string          `json:"note"`
	ReportID       int64           `json:"report_id"`
	DemoName       string          `json:"demo_name"`
	DemoTick       int             `json:"demo_tick"`
	IncludeFriends bool            `json:"include_friends"`
}

func onAPIPostBanSteamCreate(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var (
			origin   = model.Web
			sid      = currentUserProfile(ctx).SteamID
			sourceID = model.StringSID(sid.String())
		)

		// srcds sourced bans provide a source_id to id the admin
		if req.SourceID != "" {
			sourceID = req.SourceID
			origin = model.InGame
		}

		duration, errDuration := util.CalcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		var banSteam model.BanSteam
		if errBanSteam := model.NewBanSteam(ctx,
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
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		if errBan := env.BanSteam(ctx, &banSteam); errBan != nil {
			log.Error("Failed to ban steam profile",
				zap.Error(errBan), zap.Int64("target_id", banSteam.TargetID.Int64()))

			if errors.Is(errBan, errs.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, errs.ErrDuplicate)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to save new steam ban", zap.Error(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, banSteam)
	}
}

func onAPIPostReportCreate(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		var req apiCreateReportReq
		if !bind(ctx, log, &req) {
			return
		}

		if req.Description == "" || len(req.Description) < 10 {
			responseErr(ctx, http.StatusBadRequest, fmt.Errorf("%w: description", errParamInvalid))

			return
		}

		// ServerStore initiated requests will have a sourceID set by the server
		// Web based reports the source should not be set, the reporter will be taken from the
		// current session information instead
		if req.SourceID == "" {
			req.SourceID = model.StringSID(currentUser.SteamID.String())
		}

		sourceID, errSourceID := req.SourceID.SID64(ctx)
		if errSourceID != nil {
			responseErr(ctx, http.StatusBadRequest, errs.ErrSourceID)
			log.Error("Invalid steam_id", zap.Error(errSourceID))

			return
		}

		targetID, errTargetID := req.TargetID.SID64(ctx)
		if errTargetID != nil {
			responseErr(ctx, http.StatusBadRequest, errs.ErrTargetID)
			log.Error("Invalid target_id", zap.Error(errTargetID))

			return
		}

		if sourceID == targetID {
			responseErr(ctx, http.StatusConflict, errSelfReport)

			return
		}

		var personSource model.Person
		if errCreatePerson := env.Store().GetPersonBySteamID(ctx, sourceID, &personSource); errCreatePerson != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Could not load player profile", zap.Error(errCreatePerson))

			return
		}

		var personTarget model.Person
		if errCreatePerson := env.Store().GetOrCreatePersonBySteamID(ctx, targetID, &personTarget); errCreatePerson != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Could not load player profile", zap.Error(errCreatePerson))

			return
		}

		if personTarget.Expired() {
			if err := thirdparty.UpdatePlayerSummary(ctx, &personTarget); err != nil {
				log.Error("Failed to update target player", zap.Error(err))
			} else {
				if errSave := env.Store().SavePerson(ctx, &personTarget); errSave != nil {
					log.Error("Failed to save target player update", zap.Error(err))
				}
			}
		}

		// Ensure the user doesn't already have an open report against the user
		var existing model.Report
		if errReports := env.Store().GetReportBySteamID(ctx, personSource.SteamID, targetID, &existing); errReports != nil {
			if !errors.Is(errReports, errs.ErrNoResult) {
				log.Error("Failed to query reports by steam id", zap.Error(errReports))
				responseErr(ctx, http.StatusInternalServerError, errInternal)

				return
			}
		}

		if existing.ReportID > 0 {
			responseErr(ctx, http.StatusConflict, errReportExists)

			return
		}

		// TODO encapsulate all operations in single tx
		report := model.NewReport()
		report.SourceID = sourceID
		report.ReportStatus = model.Opened
		report.Description = req.Description
		report.TargetID = targetID
		report.Reason = req.Reason
		report.ReasonText = req.ReasonText
		parts := strings.Split(req.DemoName, "/")
		report.DemoName = parts[len(parts)-1]
		report.DemoTick = req.DemoTick
		report.PersonMessageID = req.PersonMessageID

		if errReportSave := env.Store().SaveReport(ctx, &report); errReportSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to save report", zap.Error(errReportSave))

			return
		}

		ctx.JSON(http.StatusCreated, report)

		log.Info("New report created successfully", zap.Int64("report_id", report.ReportID))

		conf := env.Config()

		if !conf.Discord.Enabled {
			return
		}

		demoURL := ""

		if report.DemoName != "" {
			demoURL = conf.ExtURLRaw("/demos/name/%s", report.DemoName)
		}

		msg := discord.NewInGameReportResponse(report, conf.ExtURL(report), currentUser, conf.ExtURL(currentUser), demoURL)

		env.SendPayload(conf.Discord.LogChannelID, msg)
	}
}

func onAPIPostServerState(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req model.PartialStateUpdate
		if !bind(ctx, log, &req) {
			return
		}

		serverID := serverFromCtx(ctx) // TODO use generic func for int
		if serverID == 0 {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		if errUpdate := env.State().Update(serverID, req); errUpdate != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.AbortWithStatus(http.StatusNoContent)
	}
}
