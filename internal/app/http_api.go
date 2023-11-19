package app

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net"
	"net/http"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

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

func onAPIPostDemo(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	// {"76561198084134025": {"score": 0, "deaths": 0, "score_total": 0}}
	type demoForm struct {
		Stats      string `form:"stats"`
		ServerName string `form:"server_name"`
		MapName    string `form:"map_name"`
	}

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

		if errPut := app.assetStore.Put(ctx, app.conf.S3.BucketDemo, asset.Name, bytes.NewReader(result.demoRaw), asset.Size, asset.MimeType); errPut != nil {
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

func onAPIPostDemosQuery(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.DemoFilter
		if !bind(ctx, log, &req) {
			return
		}

		demos, count, errDemos := app.db.GetDemos(ctx, req)
		if errDemos != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to query demos", zap.Error(errDemos))

			return
		}

		ctx.JSON(http.StatusCreated, LazyResult{
			Count: count,
			Data:  demos,
		})
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
		if !bind(ctx, log, &req) {
			return
		}

		state := app.state.current()
		players := state.find(findOpts{SteamID: req.SteamID})

		if len(players) == 0 {
			log.Error("Failed to find player on /mod call")
			responseErr(ctx, http.StatusFailedDependency, consts.ErrInternal)

			return
		}

		msgEmbed := discord.
			NewEmbed("New User In-Game Report").
			SetDescription(fmt.Sprintf("%s | <@&%s>", req.Reason, app.conf.Discord.ModPingRoleID)).
			AddField("server", req.ServerName)

		app.addAuthor(ctx, msgEmbed, req.SteamID).Truncate()

		app.bot.SendPayload(discord.Payload{ChannelID: app.conf.Discord.LogChannelID, Embed: msgEmbed.MessageEmbed})

		ctx.JSON(http.StatusOK, gin.H{
			"client":  req.Client,
			"message": "Moderators have been notified",
		})
	}
}

func onAPIPostBanState(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportID, errID := getInt64Param(ctx, "report_id")
		if errID != nil || reportID <= 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var report store.Report
		if errReport := app.db.GetReport(ctx, reportID, &report); errReport != nil {
			if errors.Is(errReport, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		go app.bot.SendPayload(discord.Payload{ChannelID: app.conf.Discord.LogChannelID, Embed: nil})
	}
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
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req setStatusReq
		if !bind(ctx, log, &req) {
			return
		}

		bannedPerson := store.NewBannedPerson()
		if banErr := app.db.GetBanByBanID(ctx, banID, &bannedPerson, false); banErr != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if bannedPerson.AppealState == req.AppealState {
			responseErr(ctx, http.StatusConflict, errors.New("State must be different than previous"))

			return
		}

		original := bannedPerson.AppealState
		bannedPerson.AppealState = req.AppealState

		if errSave := app.db.SaveBan(ctx, &bannedPerson.BanSteam); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusAccepted, gin.H{})

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
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req apiUnbanRequest
		if !bind(ctx, log, &req) {
			return
		}

		bannedPerson := store.NewBannedPerson()
		if banErr := app.db.GetBanByBanID(ctx, banID, &bannedPerson, false); banErr != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		changed, errSave := app.Unban(ctx, bannedPerson.TargetID, req.UnbanReasonText)
		if errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if !changed {
			responseErr(ctx, http.StatusNotFound, errors.New("Failed to save unban"))

			return
		}

		ctx.JSON(http.StatusAccepted, gin.H{})
	}
}

func onAPIPostBanUpdate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type updateBanRequest struct {
		TargetID       store.StringSID `json:"target_id"`
		BanType        store.BanType   `json:"ban_type"`
		Reason         store.Reason    `json:"reason"`
		ReasonText     string          `json:"reason_text"`
		Note           string          `json:"note"`
		IncludeFriends bool            `json:"include_friends"`
		ValidUntil     time.Time       `json:"valid_until"`
	}

	return func(ctx *gin.Context) {
		banID, banIDErr := getInt64Param(ctx, "ban_id")
		if banIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req updateBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		if time.Since(req.ValidUntil) > 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		bannedPerson := store.NewBannedPerson()
		if banErr := app.db.GetBanByBanID(ctx, banID, &bannedPerson, false); banErr != nil {
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		if req.Reason == store.Custom {
			if req.ReasonText == "" {
				responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

				return
			}

			bannedPerson.ReasonText = req.ReasonText
		} else {
			bannedPerson.ReasonText = ""
		}

		bannedPerson.Note = req.Note
		bannedPerson.BanType = req.BanType
		bannedPerson.Reason = req.Reason
		bannedPerson.IncludeFriends = req.IncludeFriends
		bannedPerson.ValidUntil = req.ValidUntil

		if errSave := app.db.SaveBan(ctx, &bannedPerson.BanSteam); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save updated ban", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusAccepted, bannedPerson)
	}
}

func onAPIPostBansGroupCreate(app *App) gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   store.StringSID `json:"target_id"`
		GroupID    steamid.GID     `json:"group_id"`
		Duration   string          `json:"duration"`
		Note       string          `json:"note"`
		ValidUntil time.Time       `json:"valid_until"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var existing store.BanGroup
		if errExist := app.db.GetBanGroup(ctx, req.GroupID, &existing); errExist != nil {
			if !errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusConflict, consts.ErrDuplicate)

				return
			}
		}

		var (
			banSteamGroup store.BanGroup
			sid           = currentUserProfile(ctx).SteamID
		)

		duration, errDuration := calcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if errBanSteamGroup := store.NewBanSteamGroup(ctx,
			store.StringSID(sid.String()),
			req.TargetID,
			duration,
			req.Note,
			store.Web,
			req.GroupID,
			"",
			store.Banned,
			&banSteamGroup,
		); errBanSteamGroup != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)
			log.Error("Failed to save group ban", zap.Error(errBanSteamGroup))

			return
		}

		if errBan := app.BanSteamGroup(ctx, &banSteamGroup); errBan != nil {
			if errors.Is(errBan, store.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, consts.ErrDuplicate)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, banSteamGroup)
	}
}

func onAPIPostBansGroupUpdate(app *App) gin.HandlerFunc {
	type apiBanUpdateRequest struct {
		TargetID   store.StringSID `json:"target_id"`
		Note       string          `json:"note"`
		ValidUntil time.Time       `json:"valid_until"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banGroupID, banIDErr := getInt64Param(ctx, "ban_group_id")
		if banIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req apiBanUpdateRequest
		if !bind(ctx, log, &req) {
			return
		}

		sid, errSID := req.TargetID.SID64(ctx)
		if errSID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var ban store.BanGroup

		if errExist := app.db.GetBanGroupByID(ctx, banGroupID, &ban); errExist != nil {
			if !errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusConflict, consts.ErrDuplicate)

			return
		}

		ban.Note = req.Note
		ban.ValidUntil = req.ValidUntil
		ban.TargetID = sid

		if errSave := app.db.SaveBanGroup(ctx, &ban); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}

func onAPIPostBansASNCreate(app *App) gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   store.StringSID `json:"target_id"`
		Note       string          `json:"note"`
		Reason     store.Reason    `json:"reason"`
		ReasonText string          `json:"reason_text"`
		ASNum      int64           `json:"as_num"`
		Duration   string          `json:"duration"`
		ValidUntil time.Time       `json:"valid_until"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var (
			banASN store.BanASN
			sid    = currentUserProfile(ctx).SteamID
		)

		duration, errDuration := calcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if errBanSteamGroup := store.NewBanASN(ctx,
			store.StringSID(sid.String()),
			req.TargetID,
			duration,
			req.Reason,
			req.ReasonText,
			req.Note,
			store.Web,
			req.ASNum,
			store.Banned,
			&banASN,
		); errBanSteamGroup != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if errBan := app.BanASN(ctx, &banASN); errBan != nil {
			if errors.Is(errBan, store.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, consts.ErrDuplicate)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Failed to save asn ban", zap.Error(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, banASN)
	}
}

func onAPIPostBansASNUpdate(app *App) gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   store.StringSID `json:"target_id"`
		Note       string          `json:"note"`
		Reason     store.Reason    `json:"reason"`
		ReasonText string          `json:"reason_text"`
		ValidUntil time.Time       `json:"valid_until"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		asnID, asnIDErr := getInt64Param(ctx, "asn_id")
		if asnIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var ban store.BanASN
		if errBan := app.db.GetBanASN(ctx, asnID, &ban); errBan != nil {
			if errors.Is(errBan, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var req apiBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		if ban.Reason == store.Custom && req.ReasonText == "" {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		sid, errSID := req.TargetID.SID64(ctx)
		if errSID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		ban.Note = req.Note
		ban.ValidUntil = req.ValidUntil
		ban.TargetID = sid
		ban.Reason = req.Reason
		ban.ReasonText = req.ReasonText

		if errSave := app.db.SaveBanASN(ctx, &ban); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}

func onAPIPostBansCIDRCreate(app *App) gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   store.StringSID `json:"target_id"`
		Duration   string          `json:"duration"`
		Note       string          `json:"note"`
		Reason     store.Reason    `json:"reason"`
		ReasonText string          `json:"reason_text"`
		CIDR       string          `json:"cidr"`
		ValidUntil time.Time       `json:"valid_until"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var (
			banCIDR store.BanCIDR
			sid     = currentUserProfile(ctx).SteamID
		)

		duration, errDuration := calcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if errBanCIDR := store.NewBanCIDR(ctx,
			store.StringSID(sid.String()),
			req.TargetID,
			duration,
			req.Reason,
			req.ReasonText,
			req.Note,
			store.Web,
			req.CIDR,
			store.Banned,
			&banCIDR,
		); errBanCIDR != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if errBan := app.BanCIDR(ctx, &banCIDR); errBan != nil {
			if errors.Is(errBan, store.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, consts.ErrDuplicate)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save cidr ban", zap.Error(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, banCIDR)
	}
}

func onAPIPostBansCIDRUpdate(app *App) gin.HandlerFunc {
	type apiUpdateBanRequest struct {
		TargetID   store.StringSID `json:"target_id"`
		Note       string          `json:"note"`
		Reason     store.Reason    `json:"reason"`
		ReasonText string          `json:"reason_text"`
		CIDR       string          `json:"cidr"`
		ValidUntil time.Time       `json:"valid_until"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		netID, banIDErr := getInt64Param(ctx, "net_id")
		if banIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var ban store.BanCIDR

		if errBan := app.db.GetBanNetByID(ctx, netID, &ban); errBan != nil {
			if errors.Is(errBan, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var req apiUpdateBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		sid, errSID := req.TargetID.SID64(ctx)
		if errSID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		if req.Reason == store.Custom && req.ReasonText == "" {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		_, ipNet, errParseCIDR := net.ParseCIDR(req.CIDR)
		if errParseCIDR != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		ban.Reason = req.Reason
		ban.ReasonText = req.ReasonText
		ban.CIDR = ipNet.String()
		ban.Note = req.Note
		ban.ValidUntil = req.ValidUntil
		ban.TargetID = sid

		if errSave := app.db.SaveBanNet(ctx, &ban); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}

func onAPIPostBanSteamCreate(app *App) gin.HandlerFunc {
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
		var req authReq
		if !bind(ctx, log, &req) {
			return
		}

		var server store.Server

		errGetServer := app.db.GetServerByName(ctx, req.ServerName, &server, true, false)
		if errGetServer != nil {
			log.Warn("Failed to find server auth by name",
				zap.String("name", req.ServerName), zap.Error(errGetServer))
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		if server.Password != req.Key {
			responseErr(ctx, http.StatusForbidden, consts.ErrPermissionDenied)
			log.Error("Invalid server key used",
				zap.String("server", util.SanitizeLog(req.ServerName)))

			return
		}

		accessToken, errToken := newServerToken(server.ServerID, app.conf.HTTP.CookieKey)
		if errToken != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to create new server access token", zap.Error(errToken))

			return
		}

		server.TokenCreatedOn = time.Now()
		if errSaveServer := app.db.SaveServer(ctx, &server); errSaveServer != nil {
			log.Error("Failed to updated server token", zap.Error(errSaveServer))
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, authResp{Status: true, Token: accessToken})
		log.Info("Server authenticated successfully", zap.String("server", server.ShortName))
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
		if errBind := ctx.BindJSON(&request); errBind != nil { // we don't currently use bind() for server api
			ctx.JSON(http.StatusInternalServerError, checkResponse{
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
			ctx.JSON(http.StatusBadRequest, resp)

			return
		}

		if parentID, banned := app.IsGroupBanned(steamID); banned {
			resp.BanType = store.Banned
			resp.Msg = fmt.Sprintf("Group/Steam Friend Ban (source: %d)", parentID)
			ctx.JSON(http.StatusOK, resp)
			log.Info("Player dropped", zap.String("drop_type", "group"),
				zap.Int64("sid64", steamID.Int64()))

			return
		}

		var person store.Person
		if errPerson := app.PersonBySID(responseCtx, steamID, &person); errPerson != nil {
			ctx.JSON(http.StatusInternalServerError, checkResponse{
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
			ctx.JSON(http.StatusInternalServerError, checkResponse{
				BanType: store.Unknown,
				Msg:     "Error determining state",
			})
			log.Error("Could not get bannedPerson net results", zap.Error(errGetBanNet))

			return
		}

		if len(banNet) > 0 {
			resp.BanType = store.Banned
			resp.Msg = fmt.Sprintf("Network banned (C: %d)", len(banNet))

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

		servers, _, errGetServers := app.db.GetServers(ctx, store.ServerQueryFilter{})
		if errGetServers != nil {
			log.Error("Failed to fetch servers", zap.Error(errGetServers))
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

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

func onAPIGetServerStates(app *App) gin.HandlerFunc {
	type UserServers struct {
		Servers []baseServer        `json:"servers"`
		LatLong ip2location.LatLong `json:"lat_long"`
	}

	return func(ctx *gin.Context) {
		var (
			lat = getDefaultFloat64(ctx.GetHeader("cf-iplatitude"), 41.7774)
			lon = getDefaultFloat64(ctx.GetHeader("cf-iplongitude"), -87.6160)
			// region := ctx.GetHeader("cf-region-code")
			curState = app.state.current()
			servers  []baseServer
		)

		for _, srv := range curState {
			servers = append(servers, baseServer{
				Host:       srv.Host,
				Port:       srv.Port,
				IP:         srv.IP,
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

		ctx.JSON(http.StatusOK, UserServers{
			Servers: servers,
			LatLong: ip2location.LatLong{
				Latitude:  lat,
				Longitude: lon,
			},
		})
	}
}

func onAPISearchPlayers(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var query store.PlayerQuery
		if !bind(ctx, log, &query) {
			return
		}

		people, count, errGetPeople := app.db.GetPeople(ctx, query)
		if errGetPeople != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, LazyResult{
			Count: count,
			Data:  people,
		})
	}
}

func onAPICurrentProfileNotifications(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentProfile := currentUserProfile(ctx)

		notifications, errNot := app.db.GetPersonNotifications(ctx, currentProfile.SteamID)
		if errNot != nil {
			if errors.Is(errNot, store.ErrNoResult) {
				ctx.JSON(http.StatusOK, []store.UserNotification{})

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, notifications)
	}
}

func onAPICurrentProfile(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		profile := currentUserProfile(ctx)
		if !profile.SteamID.Valid() {
			log.Error("Failed to load user profile",
				zap.Int64("sid64", profile.SteamID.Int64()),
				zap.String("name", profile.Name),
				zap.String("permission_level", profile.PermissionLevel.String()))
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		ctx.JSON(http.StatusOK, profile)
	}
}

func onAPIExportBansValveSteamID(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		bans, _, errBans := app.db.GetBansSteam(ctx, store.SteamBansQueryFilter{
			BansQueryFilter: store.BansQueryFilter{PermanentOnly: true},
		})

		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var entries []string

		for _, ban := range bans {
			if ban.Deleted ||
				!ban.IsEnabled {
				continue
			}

			entries = append(entries, fmt.Sprintf("banid 0 %s", steamid.SID64ToSID(ban.TargetID)))
		}

		ctx.Data(http.StatusOK, "text/plain", []byte(strings.Join(entries, "\n")))
	}
}

func onAPIExportBansValveIP(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		bans, _, errBans := app.db.GetBansNet(ctx, store.CIDRBansQueryFilter{})
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var entries []string

		for _, ban := range bans {
			if ban.Deleted ||
				!ban.IsEnabled {
				continue
			}
			// TODO Shouldn't be cidr?
			entries = append(entries, fmt.Sprintf("addip 0 %s", ban.CIDR))
		}

		ctx.Data(http.StatusOK, "text/plain", []byte(strings.Join(entries, "\n")))
	}
}

func onAPIExportSourcemodSimpleAdmins(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		privilegedIds, errPrivilegedIds := app.db.GetSteamIdsAbove(ctx, consts.PReserved)
		if errPrivilegedIds != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		players, errPlayers := app.db.GetPeopleBySteamID(ctx, privilegedIds)
		if errPlayers != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

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
		bans, _, errBans := app.db.GetBansSteam(ctx, store.SteamBansQueryFilter{
			BansQueryFilter: store.BansQueryFilter{
				QueryFilter: store.QueryFilter{
					Deleted: false,
				},
			},
		})

		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var filtered []store.BannedSteamPerson

		for _, ban := range bans {
			if ban.Reason != store.Cheating ||
				ban.Deleted ||
				!ban.IsEnabled {
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
				UpdateURL:   app.ExtURLRaw("/export/bans/tf2bd"),
			},
			Players: []thirdparty.Players{},
		}

		for _, ban := range filtered {
			out.Players = append(out.Players, thirdparty.Players{
				Attributes: []string{"cheater"},
				Steamid:    ban.TargetID,
				LastSeen: thirdparty.LastSeen{
					PlayerName: ban.TargetPersonaname,
					Time:       int(ban.UpdatedOn.Unix()),
				},
			})
		}

		ctx.JSON(http.StatusOK, out)
	}
}

func onAPIProfile(app *App) gin.HandlerFunc {
	type profileQuery struct {
		Query string `form:"query"`
	}

	type resp struct {
		Player  *store.Person     `json:"player"`
		Friends []steamweb.Friend `json:"friends"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		requestCtx, cancelRequest := context.WithTimeout(ctx, time.Second*15)
		defer cancelRequest()

		var req profileQuery
		if errBind := ctx.Bind(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		sid, errResolveSID64 := steamid.ResolveSID64(requestCtx, req.Query)
		if errResolveSID64 != nil {
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		person := store.NewPerson(sid)
		if errGetProfile := app.PersonBySID(requestCtx, sid, &person); errGetProfile != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to create person", zap.Error(errGetProfile))

			log.Error("Failed to create new profile", zap.Error(errGetProfile))

			return
		}

		var response resp

		friendList, errFetchFriends := steamweb.GetFriendList(requestCtx, person.SteamID)
		if errFetchFriends == nil {
			response.Friends = friendList
		}

		response.Player = &person

		ctx.JSON(http.StatusOK, response)
	}
}

func onAPIQueryWordFilters(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var opts store.FiltersQueryFilter
		if !bind(ctx, log, &opts) {
			return
		}

		words, count, errGetFilters := app.db.GetFilters(ctx, opts)
		if errGetFilters != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, LazyResult{
			Count: count,
			Data:  words,
		})
	}
}

func onAPIPostWordMatch(app *App) gin.HandlerFunc {
	type matchRequest struct {
		Query string
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req matchRequest
		if !bind(ctx, log, &req) {
			return
		}

		words, _, errGetFilters := app.db.GetFilters(ctx, store.FiltersQueryFilter{})
		if errGetFilters != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var matches []store.Filter

		for _, filter := range words {
			if filter.Match(req.Query) {
				matches = append(matches, filter)
			}
		}

		ctx.JSON(http.StatusOK, matches)
	}
}

func onAPIDeleteWordFilter(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		wordID, wordIDErr := getInt64Param(ctx, "word_id")
		if wordIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var filter store.Filter
		if errGet := app.db.GetFilterByID(ctx, wordID, &filter); errGet != nil {
			if errors.Is(errGet, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if errDrop := app.db.DropFilter(ctx, &filter); errDrop != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusNoContent, nil)
	}
}

func onAPIPostWordFilter(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.Filter
		if !bind(ctx, log, &req) {
			return
		}

		if req.Pattern == "" {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if req.IsRegex {
			_, compErr := regexp.Compile(req.Pattern)
			if compErr != nil {
				responseErr(ctx, http.StatusBadRequest, errors.New("invalid regex"))

				return
			}
		}

		now := time.Now()

		if req.FilterID > 0 {
			var existingFilter store.Filter
			if errGet := app.db.GetFilterByID(ctx, req.FilterID, &existingFilter); errGet != nil {
				if errors.Is(errGet, store.ErrNoResult) {
					responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

					return
				}

				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

				return
			}

			existingFilter.UpdatedOn = now
			existingFilter.Pattern = req.Pattern
			existingFilter.IsRegex = req.IsRegex
			existingFilter.IsEnabled = req.IsEnabled

			if errSave := app.FilterAdd(ctx, &existingFilter); errSave != nil {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

				return
			}

			req = existingFilter
		} else {
			profile := currentUserProfile(ctx)
			newFilter := store.Filter{
				AuthorID:  profile.SteamID,
				Pattern:   req.Pattern,
				CreatedOn: now,
				UpdatedOn: now,
				IsRegex:   req.IsRegex,
				IsEnabled: req.IsEnabled,
			}

			if errSave := app.FilterAdd(ctx, &newFilter); errSave != nil {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

				return
			}

			req = newFilter
		}

		ctx.JSON(http.StatusOK, req)
	}
}

func onAPIGetStats(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var stats store.Stats
		if errGetStats := app.db.GetStats(ctx, &stats); errGetStats != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		stats.ServersAlive = 1

		ctx.JSON(http.StatusOK, stats)
	}
}

func loadBanMeta(_ *store.BannedSteamPerson) {
}

func onAPIGetBanByID(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		curUser := currentUserProfile(ctx)

		banID, errID := getInt64Param(ctx, "ban_id")
		if errID != nil || banID == 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

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
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)
			log.Error("Failed to fetch steam ban", zap.Error(errGetBan))

			return
		}

		if !checkPrivilege(ctx, curUser, steamid.Collection{bannedPerson.TargetID}, consts.PModerator) {
			return
		}

		loadBanMeta(&bannedPerson)
		ctx.JSON(http.StatusOK, bannedPerson)
	}
}

func onAPIGetAppeals(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.AppealQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		bans, total, errBans := app.db.GetAppealsByActivity(ctx, req)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to fetch appeals", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, LazyResult{
			Count: total,
			Data:  bans,
		})
	}
}

func onAPIGetBansSteam(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.SteamBansQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		bans, count, errBans := app.db.GetBansSteam(ctx, req)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to fetch steam bans", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, LazyResult{
			Count: count,
			Data:  bans,
		})
	}
}

func onAPIGetBansCIDR(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.CIDRBansQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		bans, count, errBans := app.db.GetBansNet(ctx, req)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to fetch cidr bans", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, LazyResult{
			Count: count,
			Data:  bans,
		})
	}
}

func onAPIDeleteBansCIDR(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		netID, netIDErr := getInt64Param(ctx, "net_id")
		if netIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req apiUnbanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var banCidr store.BanCIDR
		if errFetch := app.db.GetBanNetByID(ctx, netID, &banCidr); errFetch != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		banCidr.UnbanReasonText = req.UnbanReasonText
		banCidr.Deleted = true

		if errSave := app.db.SaveBanNet(ctx, &banCidr); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to delete cidr ban", zap.Error(errSave))

			return
		}

		banCidr.NetID = 0

		ctx.JSON(http.StatusOK, banCidr)
	}
}

func onAPIGetBansGroup(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.GroupBansQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		banGroups, count, errBans := app.db.GetBanGroups(ctx, req)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to fetch banGroups", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, LazyResult{
			Count: count,
			Data:  banGroups,
		})
	}
}

func onAPIDeleteBansGroup(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		groupID, groupIDErr := getInt64Param(ctx, "ban_group_id")
		if groupIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInternal)

			return
		}

		var req apiUnbanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var banGroup store.BanGroup
		if errFetch := app.db.GetBanGroupByID(ctx, groupID, &banGroup); errFetch != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInternal)

			return
		}

		banGroup.UnbanReasonText = req.UnbanReasonText
		banGroup.Deleted = true

		if errSave := app.db.SaveBanGroup(ctx, &banGroup); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to delete asn ban", zap.Error(errSave))

			return
		}

		banGroup.BanGroupID = 0
		ctx.JSON(http.StatusOK, banGroup)
	}
}

func onAPIGetBansASN(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.ASNBansQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		bansASN, count, errBans := app.db.GetBansASN(ctx, req)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to fetch banASN", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, LazyResult{
			Count: count,
			Data:  bansASN,
		})
	}
}

func onAPIDeleteBansASN(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		asnID, asnIDErr := getInt64Param(ctx, "asn_id")
		if asnIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req apiUnbanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var banAsn store.BanASN
		if errFetch := app.db.GetBanASN(ctx, asnID, &banAsn); errFetch != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		banAsn.UnbanReasonText = req.UnbanReasonText
		banAsn.Deleted = true

		if errSave := app.db.SaveBanASN(ctx, &banAsn); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to delete asn ban", zap.Error(errSave))

			return
		}

		banAsn.BanASNId = 0

		ctx.JSON(http.StatusOK, banAsn)
	}
}

func onAPIGetServersAdmin(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var filter store.ServerQueryFilter
		if !bind(ctx, log, &filter) {
			return
		}

		servers, count, errServers := app.db.GetServers(ctx, filter)
		if errServers != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, LazyResult{
			Count: count,
			Data:  servers,
		})
	}
}

type serverInfoSafe struct {
	ServerNameLong string `json:"server_name_long"`
	ServerName     string `json:"server_name"`
	ServerID       int    `json:"server_id"`
	Colour         string `json:"colour"`
}

func onAPIGetServers(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fullServers, _, errServers := app.db.GetServers(ctx, store.ServerQueryFilter{})
		if errServers != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var servers []serverInfoSafe
		for _, server := range fullServers {
			servers = append(servers, serverInfoSafe{
				ServerNameLong: server.Name,
				ServerName:     server.ShortName,
				ServerID:       server.ServerID,
				Colour:         "",
			})
		}

		ctx.JSON(http.StatusOK, servers)
	}
}

func onAPIGetMapUsage(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mapUsages, errServers := app.db.GetMapUsageStats(ctx)
		if errServers != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, mapUsages)
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
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var server store.Server
		if errServer := app.db.GetServer(ctx, serverID, &server); errServer != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var req serverUpdateRequest
		if !bind(ctx, log, &req) {
			return
		}

		server.ShortName = req.ServerNameShort
		server.Name = req.ServerName
		server.Address = req.Host
		server.Port = req.Port
		server.ReservedSlots = req.ReservedSlots
		server.RCON = req.RCON
		server.Latitude = req.Lat
		server.Longitude = req.Lon
		server.CC = req.CC
		server.Region = req.Region
		server.IsEnabled = req.IsEnabled

		if errSave := app.db.SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to update server", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)

		log.Info("Server config updated",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ShortName))
	}
}

func onAPIPostServerDelete(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		serverID, idErr := getIntParam(ctx, "server_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var server store.Server
		if errServer := app.db.GetServer(ctx, serverID, &server); errServer != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		server.Deleted = true

		if errSave := app.db.SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to delete server", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)
		log.Info("Server config deleted",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ShortName))
	}
}

func onAPIPostServer(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req serverUpdateRequest
		if !bind(ctx, log, &req) {
			return
		}

		server := store.NewServer(req.ServerNameShort, req.Host, req.Port)
		server.Name = req.ServerName
		server.ReservedSlots = req.ReservedSlots
		server.RCON = req.RCON
		server.Latitude = req.Lat
		server.Longitude = req.Lon
		server.CC = req.CC
		server.Region = req.Region
		server.IsEnabled = req.IsEnabled

		if errSave := app.db.SaveServer(ctx, &server); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save new server", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, server)

		log.Info("Server config created",
			zap.Int("server_id", server.ServerID),
			zap.String("name", server.ShortName))
	}
}

func onAPIPostReportCreate(app *App) gin.HandlerFunc {
	type createReport struct {
		SourceID        store.StringSID `json:"source_id"`
		TargetID        store.StringSID `json:"target_id"`
		Description     string          `json:"description"`
		Reason          store.Reason    `json:"reason"`
		ReasonText      string          `json:"reason_text"`
		DemoName        string          `json:"demo_name"`
		DemoTick        int             `json:"demo_tick"`
		PersonMessageID int64           `json:"person_message_id"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		var req createReport
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
		if errReports := app.db.GetReportBySteamID(ctx, currentUser.SteamID, targetID, &existing); errReports != nil {
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

		log.Info("New report created successfully", zap.Int64("report_id", report.ReportID))
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

func getUUIDParam(ctx *gin.Context, key string) (uuid.UUID, error) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		return uuid.UUID{}, errors.Errorf("Failed to get %s", key)
	}

	parsedUUID, errString := uuid.FromString(valueStr)
	if errString != nil {
		return uuid.UUID{}, errors.Wrap(errString, "Failed to parse UUID")
	}

	return parsedUUID, nil
}

func onAPIPostReportMessage(app *App) gin.HandlerFunc {
	type newMessage struct {
		Message string `json:"message"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportID, errID := getInt64Param(ctx, "report_id")
		if errID != nil || reportID == 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req newMessage
		if !bind(ctx, log, &req) {
			return
		}

		if req.Message == "" {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var report store.Report
		if errReport := app.db.GetReport(ctx, reportID, &report); errReport != nil {
			if errors.Is(store.Err(errReport), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to load report", zap.Error(errReport))

			return
		}

		person := currentUserProfile(ctx)
		msg := store.NewUserMessage(reportID, person.SteamID, req.Message)

		if errSave := app.db.SaveReportMessage(ctx, &msg); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		report.UpdatedOn = time.Now()

		if errSave := app.db.SaveReport(ctx, &report); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to update report activity", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, msg)

		msgEmbed := discord.
			NewEmbed("New report message posted").
			SetDescription(msg.Contents).
			SetColor(app.bot.Colour.Success).
			SetURL(app.ExtURL(report))

		app.addAuthorUserProfile(msgEmbed, person).Truncate()

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.LogChannelID,
			Embed:     msgEmbed.MessageEmbed,
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
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var existing store.UserMessage
		if errExist := app.db.GetReportMessageByID(ctx, reportMessageID, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrPlayerNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		curUser := currentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, consts.PModerator) {
			return
		}

		var req editMessage
		if !bind(ctx, log, &req) {
			return
		}

		if req.BodyMD == "" {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if req.BodyMD == existing.Contents {
			responseErr(ctx, http.StatusConflict, consts.ErrDuplicate)

			return
		}

		existing.Contents = req.BodyMD
		if errSave := app.db.SaveReportMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, req)

		msgEmbed := discord.
			NewEmbed("New report message edited").
			SetDescription(req.BodyMD).
			SetColor(app.bot.Colour.Warn).
			AddField("Old Message", existing.Contents).
			SetURL(app.ExtURLRaw("/report/%d", existing.ParentID))

		app.addAuthorUserProfile(msgEmbed, curUser).Truncate()

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.LogChannelID,
			Embed:     msgEmbed.MessageEmbed,
		})
	}
}

func onAPIDeleteReportMessage(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportMessageID, errID := getInt64Param(ctx, "report_message_id")
		if errID != nil || reportMessageID == 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var existing store.UserMessage
		if errExist := app.db.GetReportMessageByID(ctx, reportMessageID, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		curUser := currentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, consts.PModerator) {
			return
		}

		existing.Deleted = true
		if errSave := app.db.SaveReportMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusNoContent, nil)

		msgEmbed := discord.
			NewEmbed("User report message deleted").
			SetDescription(existing.Contents).
			SetColor(app.bot.Colour.Warn)

		app.addAuthorUserProfile(msgEmbed, curUser).
			Truncate()

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.LogChannelID,
			Embed:     msgEmbed.MessageEmbed,
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
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req stateUpdateReq
		if !bind(ctx, log, &req) {
			return
		}

		var report store.Report
		if errGet := app.db.GetReport(ctx, reportID, &report); errGet != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to get report to set state", zap.Error(errGet))

			return
		}

		if report.ReportStatus == req.Status {
			ctx.JSON(http.StatusConflict, consts.ErrDuplicate)

			return
		}

		original := report.ReportStatus

		report.ReportStatus = req.Status
		if errSave := app.db.SaveReport(ctx, &report); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save report state", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusAccepted, nil)
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
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var report store.Report
		if errGetReport := app.db.GetReport(ctx, reportID, &report); errGetReport != nil {
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		if !checkPrivilege(ctx, currentUserProfile(ctx), steamid.Collection{report.SourceID, report.TargetID}, consts.PModerator) {
			return
		}

		reportMessages, errGetReportMessages := app.db.GetReportMessages(ctx, reportID)
		if errGetReportMessages != nil {
			responseErr(ctx, http.StatusNotFound, consts.ErrPlayerNotFound)

			return
		}

		var ids steamid.Collection
		for _, msg := range reportMessages {
			ids = append(ids, msg.AuthorID)
		}

		authors, authorsErr := app.db.GetPeopleBySteamID(ctx, ids)
		if authorsErr != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

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

		ctx.JSON(http.StatusOK, authorMessages)
	}
}

type reportWithAuthor struct {
	Author  store.Person `json:"author"`
	Subject store.Person `json:"subject"`
	store.Report
}

func onAPIGetReports(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.ReportQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		if req.Limit <= 0 && req.Limit > 100 {
			req.Limit = 25
		}

		var userReports []reportWithAuthor

		reports, count, errReports := app.db.GetReports(ctx, req)
		if errReports != nil {
			if errors.Is(store.Err(errReports), store.ErrNoResult) {
				ctx.JSON(http.StatusNoContent, nil)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var authorIds steamid.Collection
		for _, report := range reports {
			authorIds = append(authorIds, report.SourceID)
		}

		authors, errAuthors := app.db.GetPeopleBySteamID(ctx, fp.Uniq(authorIds))
		if errAuthors != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		authorMap := authors.AsMap()

		var subjectIds steamid.Collection
		for _, report := range reports {
			subjectIds = append(subjectIds, report.TargetID)
		}

		subjects, errSubjects := app.db.GetPeopleBySteamID(ctx, fp.Uniq(subjectIds))
		if errSubjects != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

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

		if userReports == nil {
			userReports = []reportWithAuthor{}
		}

		ctx.JSON(http.StatusOK, LazyResult{
			Count: count,
			Data:  userReports,
		})
	}
}

func onAPIGetReport(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportID, errParam := getInt64Param(ctx, "report_id")
		if errParam != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var report reportWithAuthor
		if errReport := app.db.GetReport(ctx, reportID, &report.Report); errReport != nil {
			if errors.Is(store.Err(errReport), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to load report", zap.Error(errReport))

			return
		}

		if !checkPrivilege(ctx, currentUserProfile(ctx), steamid.Collection{report.Report.SourceID}, consts.PModerator) {
			responseErr(ctx, http.StatusUnauthorized, consts.ErrPermissionDenied)

			return
		}

		if errAuthor := app.PersonBySID(ctx, report.Report.SourceID, &report.Author); errAuthor != nil {
			if errors.Is(store.Err(errAuthor), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)
			log.Error("Failed to load report author", zap.Error(errAuthor))

			return
		}

		if errSubject := app.PersonBySID(ctx, report.Report.TargetID, &report.Subject); errSubject != nil {
			if errors.Is(store.Err(errSubject), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)
			log.Error("Failed to load report subject", zap.Error(errSubject))

			return
		}

		ctx.JSON(http.StatusOK, report)
	}
}

func onAPIGetNewsLatest(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := app.db.GetNewsLatest(ctx, 50, false)
		if errGetNewsLatest != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newsLatest)
	}
}

func onAPIGetNewsAll(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := app.db.GetNewsLatest(ctx, 100, true)
		if errGetNewsLatest != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newsLatest)
	}
}

func onAPIPostNewsCreate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.NewsEntry
		if !bind(ctx, log, &req) {
			return
		}

		if errSave := app.db.SaveNewsArticle(ctx, &req); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, req)

		go app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.LogChannelID,
			Embed: discord.
				NewEmbed("News Created").
				SetDescription(req.BodyMD).
				AddField("Title", req.Title).MessageEmbed,
		})
	}
}

func onAPIPostNewsUpdate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		newsID, errID := getIntParam(ctx, "news_id")
		if errID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var entry store.NewsEntry
		if errGet := app.db.GetNewsByID(ctx, newsID, &entry); errGet != nil {
			if errors.Is(store.Err(errGet), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if !bind(ctx, log, &entry) {
			return
		}

		if errSave := app.db.SaveNewsArticle(ctx, &entry); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusAccepted, entry)

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.LogChannelID,
			Embed: discord.
				NewEmbed("News Updated").
				AddField("Title", entry.Title).
				SetDescription(entry.BodyMD).
				MessageEmbed,
		})
	}
}

type UserUploadedFile struct {
	Content string `json:"content"`
	Name    string `json:"name"`
	Mime    string `json:"mime"`
	Size    int64  `json:"size"`
}

func onAPISaveMedia(app *App) gin.HandlerFunc {
	MediaSafeMimeTypesImages := []string{
		"image/gif",
		"image/jpeg",
		"image/png",
		"image/webp",
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req UserUploadedFile
		if !bind(ctx, log, &req) {
			return
		}

		content, decodeErr := base64.StdEncoding.DecodeString(req.Content)
		if decodeErr != nil {
			ctx.JSON(http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		media, errMedia := store.NewMedia(currentUserProfile(ctx).SteamID, req.Name, req.Mime, content)
		if errMedia != nil {
			ctx.JSON(http.StatusBadRequest, consts.ErrBadRequest)
			log.Error("Invalid media uploaded", zap.Error(errMedia))
		}

		asset, errAsset := store.NewAsset(content, app.conf.S3.BucketMedia, "")
		if errAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save asset"))

			return
		}

		if errPut := app.assetStore.Put(ctx, app.conf.S3.BucketMedia, asset.Name, bytes.NewReader(content), asset.Size, asset.MimeType); errPut != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save media"))

			log.Error("Failed to save user media to s3 backend", zap.Error(errPut))

			return
		}

		if errSaveAsset := app.db.SaveAsset(ctx, &asset); errSaveAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save asset"))

			log.Error("Failed to save user asset to s3 backend", zap.Error(errSaveAsset))
		}

		media.Asset = asset

		media.Contents = nil

		if !fp.Contains(MediaSafeMimeTypesImages, media.MimeType) {
			responseErr(ctx, http.StatusBadRequest, errors.New("Invalid image format"))
			log.Error("User tried uploading image with forbidden mimetype",
				zap.String("mime", media.MimeType), zap.String("name", media.Name))

			return
		}

		if errSave := app.db.SaveMedia(ctx, &media); errSave != nil {
			log.Error("Failed to save wiki media", zap.Error(errSave))

			if errors.Is(store.Err(errSave), store.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, errors.New("Duplicate media name"))

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save media"))

			return
		}

		ctx.JSON(http.StatusCreated, media)
	}
}

func onAPISaveContestEntrySubmit(app *App) gin.HandlerFunc {
	type entryReq struct {
		Description string    `json:"description"`
		AssetID     uuid.UUID `json:"asset_id"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)
		contest, success := contestFromCtx(ctx, app)

		if !success {
			return
		}

		var req entryReq
		if !bind(ctx, log, &req) {
			return
		}

		if contest.MediaTypes != "" {
			var media store.Media
			if errMedia := app.db.GetMediaByAssetID(ctx, req.AssetID, &media); errMedia != nil {
				responseErr(ctx, http.StatusFailedDependency, errors.New("Could not load media asset"))

				return
			}

			if !contest.MimeTypeAcceptable(media.MimeType) {
				responseErr(ctx, http.StatusFailedDependency, errors.New("Invalid Mime Type"))

				return
			}
		}

		existingEntries, errEntries := app.db.ContestEntries(ctx, contest.ContestID)
		if errEntries != nil && !errors.Is(errEntries, store.ErrNoResult) {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not load existing contest entries"))

			return
		}

		own := 0

		for _, entry := range existingEntries {
			if entry.SteamID == user.SteamID {
				own++
			}

			if own >= contest.MaxSubmissions {
				responseErr(ctx, http.StatusForbidden, errors.New("Current entries count exceed max_submissions"))

				return
			}
		}

		steamID := currentUserProfile(ctx).SteamID

		entry, errEntry := contest.NewEntry(steamID, req.AssetID, req.Description)
		if errEntry != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not create content entry"))

			return
		}

		if errSave := app.db.ContestEntrySave(ctx, entry); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save entry"))

			return
		}

		ctx.JSON(http.StatusCreated, entry)

		log.Info("New contest entry submitted", zap.String("contest_id", contest.ContestID.String()))
	}
}

func contestFromCtx(ctx *gin.Context, app *App) (store.Contest, bool) {
	contestID, idErr := getUUIDParam(ctx, "contest_id")
	if idErr != nil {
		responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

		return store.Contest{}, false
	}

	var contest store.Contest
	if errContests := app.db.ContestByID(ctx, contestID, &contest); errContests != nil {
		responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

		return store.Contest{}, false
	}

	if !contest.Public && currentUserProfile(ctx).PermissionLevel < consts.PModerator {
		responseErr(ctx, http.StatusForbidden, consts.ErrNotFound)

		return store.Contest{}, false
	}

	return contest, true
}

func onAPISaveContestEntryVote(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type voteResult struct {
		CurrentVote string `json:"current_vote"`
	}

	return func(ctx *gin.Context) {
		contest, success := contestFromCtx(ctx, app)
		if !success {
			return
		}

		contestEntryID, errContestEntryID := getUUIDParam(ctx, "contest_entry_id")
		if errContestEntryID != nil {
			ctx.JSON(http.StatusNotFound, consts.ErrNotFound)
			log.Error("Invalid contest entry id option")

			return
		}

		direction := strings.ToLower(ctx.Param("direction"))
		if direction != "up" && direction != "down" {
			ctx.JSON(http.StatusBadRequest, consts.ErrBadRequest)
			log.Error("Invalid vote direction option")

			return
		}

		if !contest.Voting || !contest.DownVotes && direction != "down" {
			ctx.JSON(http.StatusBadRequest, consts.ErrBadRequest)
			log.Error("Voting not enabled")

			return
		}

		currentUser := currentUserProfile(ctx)

		if errVote := app.db.ContestEntryVote(ctx, contestEntryID, currentUser.SteamID, direction == "up"); errVote != nil {
			if errors.Is(errVote, store.ErrVoteDeleted) {
				ctx.JSON(http.StatusOK, voteResult{""})

				return
			}

			ctx.JSON(http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, voteResult{direction})
	}
}

func onAPISaveContestEntryMedia(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		contest, success := contestFromCtx(ctx, app)
		if !success {
			return
		}

		var req UserUploadedFile
		if !bind(ctx, log, &req) {
			return
		}

		content, decodeErr := base64.StdEncoding.DecodeString(req.Content)
		if decodeErr != nil {
			ctx.JSON(http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		media, errMedia := store.NewMedia(currentUserProfile(ctx).SteamID, req.Name, req.Mime, content)
		if errMedia != nil {
			ctx.JSON(http.StatusBadRequest, consts.ErrBadRequest)
			log.Error("Invalid media uploaded", zap.Error(errMedia))
		}

		asset, errAsset := store.NewAsset(content, app.conf.S3.BucketMedia, "")
		if errAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save asset"))

			return
		}

		if errPut := app.assetStore.Put(ctx, app.conf.S3.BucketMedia, asset.Name, bytes.NewReader(content), asset.Size, asset.MimeType); errPut != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save user contest media"))

			log.Error("Failed to save user contest entry media to s3 backend", zap.Error(errPut))

			return
		}

		if errSaveAsset := app.db.SaveAsset(ctx, &asset); errSaveAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save asset"))

			log.Error("Failed to save user asset to s3 backend", zap.Error(errSaveAsset))
		}

		media.Asset = asset

		media.Contents = nil

		if !contest.MimeTypeAcceptable(media.MimeType) {
			responseErr(ctx, http.StatusUnsupportedMediaType, errors.New("Invalid file format"))
			log.Error("User tried uploading file with forbidden mimetype",
				zap.String("mime", media.MimeType), zap.String("name", media.Name))

			return
		}

		if errSave := app.db.SaveMedia(ctx, &media); errSave != nil {
			log.Error("Failed to save user contest media", zap.Error(errSave))

			if errors.Is(store.Err(errSave), store.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, errors.New("Duplicate media name"))

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save user contest media"))

			return
		}

		ctx.JSON(http.StatusCreated, media)
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
				ctx.JSON(http.StatusOK, page)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, page)
	}
}

func onGetMediaByID(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mediaID, idErr := getIntParam(ctx, "media_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var media store.Media
		if errMedia := app.db.GetMediaByID(ctx, mediaID, &media); errMedia != nil {
			if errors.Is(store.Err(errMedia), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)
			} else {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			}

			return
		}

		ctx.Data(http.StatusOK, media.MimeType, media.Contents)
	}
}

func onAPISaveWikiSlug(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req wiki.Page
		if !bind(ctx, log, &req) {
			return
		}

		if req.Slug == "" || req.BodyMD == "" {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var page wiki.Page
		if errGetWikiSlug := app.db.GetWikiPageBySlug(ctx, req.Slug, &page); errGetWikiSlug != nil {
			if errors.Is(errGetWikiSlug, store.ErrNoResult) {
				page.CreatedOn = time.Now()
				page.Revision += 1
				page.Slug = req.Slug
			} else {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

				return
			}
		} else {
			page = page.NewRevision()
		}

		page.BodyMD = req.BodyMD
		if errSave := app.db.SaveWikiPage(ctx, &page); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, page)
	}
}

type MatchQueryResults struct {
	ResultsCount
	Matches []store.MatchSummary `json:"matches"`
}

func onAPIGetMatches(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.MatchesQueryOpts
		if !bind(ctx, log, &req) {
			return
		}

		// Don't let normal users query anybody but themselves
		user := currentUserProfile(ctx)
		if user.PermissionLevel <= consts.PUser {
			if !req.SteamID.Valid() {
				responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

				return
			}

			if user.SteamID != req.SteamID {
				responseErr(ctx, http.StatusForbidden, consts.ErrPermissionDenied)

				return
			}
		}

		matches, totalCount, matchesErr := app.db.Matches(ctx, req)
		if matchesErr != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to perform query", zap.Error(matchesErr))

			return
		}

		ctx.JSON(http.StatusOK, MatchQueryResults{
			ResultsCount: ResultsCount{Count: totalCount},
			Matches:      matches,
		})
	}
}

func onAPIGetMatch(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		matchID, errID := getUUIDParam(ctx, "match_id")
		if errID != nil {
			log.Error("Invalid match_id value", zap.Error(errID))
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var match store.MatchResult

		errMatch := app.db.MatchGetByID(ctx, matchID, &match)

		if errMatch != nil {
			if errors.Is(errMatch, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, match)
	}
}

type ResultsCount struct {
	Count int64 `json:"count"`
}

func onAPIQueryPersonConnections(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.ConnectionHistoryQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		ipHist, totalCount, errIPHist := app.db.QueryConnectionHistory(ctx, req)
		if errIPHist != nil && !errors.Is(errIPHist, store.ErrNoResult) {
			log.Error("Failed to query connection history", zap.Error(errIPHist))
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, LazyResult{
			Count: totalCount,
			Data:  ipHist,
		})
	}
}

func onAPIQueryMessageContext(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		messageID, errMessageID := getInt64Param(ctx, "person_message_id")
		if errMessageID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)
			log.Debug("Got invalid person_message_id", zap.Error(errMessageID))

			return
		}

		padding, errPadding := getIntParam(ctx, "padding")
		if errPadding != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)
			log.Debug("Got invalid padding", zap.Error(errPadding))

			return
		}

		var msg store.QueryChatHistoryResult
		if errMsg := app.db.GetPersonMessage(ctx, messageID, &msg); errMsg != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		messages, errQuery := app.db.GetPersonMessageContext(ctx, msg.ServerID, messageID, padding)
		if errQuery != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, messages)
	}
}

func onAPIGetStatsWeaponsOverall(ctx context.Context, app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	updater := store.NewDataUpdater(log, time.Minute*10, func() ([]store.WeaponsOverallResult, error) {
		weaponStats, errUpdate := app.db.WeaponsOverall(ctx)
		if errUpdate != nil && !errors.Is(errUpdate, store.ErrNoResult) {
			return nil, errors.Wrap(errUpdate, "Failed to update weapon stats")
		}

		if weaponStats == nil {
			weaponStats = []store.WeaponsOverallResult{}
		}

		return weaponStats, nil
	})

	go updater.Start(ctx)

	return func(ctx *gin.Context) {
		stats := updater.Data()

		ctx.JSON(http.StatusOK, LazyResult{
			Count: int64(len(stats)),
			Data:  stats,
		})
	}
}

func onAPIGetStatsPlayersOverall(ctx context.Context, app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	updater := store.NewDataUpdater(log, time.Minute*10, func() ([]store.PlayerWeaponResult, error) {
		updatedStats, errChat := app.db.PlayersOverallByKills(ctx, 1000)
		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			return nil, errors.Wrap(errChat, "Failed to query overall players overall")
		}

		return updatedStats, nil
	})

	go updater.Start(ctx)

	return func(ctx *gin.Context) {
		stats := updater.Data()
		ctx.JSON(http.StatusOK, LazyResult{Count: int64(len(stats)), Data: stats})
	}
}

func onAPIGetStatsHealersOverall(ctx context.Context, app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	updater := store.NewDataUpdater(log, time.Minute*10, func() ([]store.HealingOverallResult, error) {
		updatedStats, errChat := app.db.HealersOverallByHealing(ctx, 250)
		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			return nil, errors.Wrap(errChat, "Failed to query overall healers overall")
		}

		return updatedStats, nil
	})

	go updater.Start(ctx)

	return func(ctx *gin.Context) {
		stats := updater.Data()
		ctx.JSON(http.StatusOK, LazyResult{Count: int64(len(stats)), Data: stats})
	}
}

func onAPIGetPlayerWeaponStatsOverall(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		steamID, errSteamID := getSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		weaponStats, errChat := app.db.WeaponsOverallByPlayer(ctx, steamID)
		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			log.Error("Failed to query player weapons stats",
				zap.Error(errChat))
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if weaponStats == nil {
			weaponStats = []store.WeaponsOverallResult{}
		}

		ctx.JSON(http.StatusOK, LazyResult{
			Count: int64(len(weaponStats)),
			Data:  weaponStats,
		})
	}
}

type LazyResult struct {
	Count int64 `json:"count"`
	Data  any   `json:"data"`
}

func onAPIGetPlayerClassStatsOverall(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		steamID, errSteamID := getSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		classStats, errChat := app.db.PlayerOverallClassStats(ctx, steamID)
		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			log.Error("Failed to query player class stats",
				zap.Error(errChat))
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if classStats == nil {
			classStats = []store.PlayerClassOverallResult{}
		}

		ctx.JSON(http.StatusOK, LazyResult{Count: int64(len(classStats)), Data: classStats})
	}
}

func onAPIGetPlayerStatsOverall(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		steamID, errSteamID := getSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var por store.PlayerOverallResult
		if errChat := app.db.PlayerOverallStats(ctx, steamID, &por); errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			log.Error("Failed to query player stats overall",
				zap.Error(errChat))
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, por)
	}
}

func onAPIGetsStatsWeapon(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type resp struct {
		LazyResult
		Weapon store.Weapon `json:"weapon"`
	}

	return func(ctx *gin.Context) {
		weaponID, errWeaponID := getIntParam(ctx, "weapon_id")
		if errWeaponID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var weapon store.Weapon

		errWeapon := app.db.GetWeaponByID(ctx, weaponID, &weapon)

		if errWeapon != nil {
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		weaponStats, errChat := app.db.WeaponsOverallTopPlayers(ctx, weaponID)
		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			log.Error("Failed to get weapons overall top stats",
				zap.Error(errChat))
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if weaponStats == nil {
			weaponStats = []store.PlayerWeaponResult{}
		}

		ctx.JSON(http.StatusOK, resp{
			LazyResult: LazyResult{
				Count: int64(len(weaponStats)),
				Data:  weaponStats,
			}, Weapon: weapon,
		})
	}
}

func onAPIQueryMessages(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.ChatHistoryQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		if req.Limit <= 0 || req.Limit > 1000 {
			req.Limit = 50
		}

		user := currentUserProfile(ctx)

		if user.PermissionLevel <= consts.PUser {
			req.Unrestricted = false
			beforeLimit := time.Now().Add(-time.Minute * 20)

			if req.DateEnd != nil && req.DateEnd.After(beforeLimit) {
				req.DateEnd = &beforeLimit
			}

			if req.DateEnd == nil {
				req.DateEnd = &beforeLimit
			}
		} else {
			req.Unrestricted = true
		}

		messages, count, errChat := app.db.QueryChatHistory(ctx, req)
		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			log.Error("Failed to query messages history",
				zap.Error(errChat), zap.String("sid", string(req.SourceID)))
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, LazyResult{
			Count: count,
			Data:  messages,
		})
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
			responseErr(ctx, http.StatusNotFound, consts.ErrInvalidParameter)

			return
		}

		banPerson := store.NewBannedPerson()
		if errGetBan := app.db.GetBanByBanID(ctx, banID, &banPerson, true); errGetBan != nil {
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		if !checkPrivilege(ctx, currentUserProfile(ctx), steamid.Collection{banPerson.TargetID, banPerson.SourceID}, consts.PModerator) {
			return
		}

		banMessages, errGetBanMessages := app.db.GetBanMessages(ctx, banID)
		if errGetBanMessages != nil {
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		var ids steamid.Collection
		for _, msg := range banMessages {
			ids = append(ids, msg.AuthorID)
		}

		authors, authorsErr := app.db.GetPeopleBySteamID(ctx, ids)
		if authorsErr != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

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

		ctx.JSON(http.StatusOK, authorMessages)
	}
}

func onAPIDeleteBanMessage(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banMessageID, errID := getIntParam(ctx, "ban_message_id")
		if errID != nil || banMessageID == 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var existing store.UserMessage
		if errExist := app.db.GetBanMessageByID(ctx, banMessageID, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		curUser := currentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, consts.PModerator) {
			return
		}

		existing.Deleted = true
		if errSave := app.db.SaveBanMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save appeal message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusNoContent, nil)

		msgEmbed := discord.
			NewEmbed("User appeal message deleted").
			SetDescription(existing.Contents)

		app.addAuthorUserProfile(msgEmbed, curUser).Truncate()

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.LogChannelID,
			Embed:     msgEmbed.MessageEmbed,
		})
	}
}

func onAPIGetSourceBans(_ *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, errID := getSID64Param(ctx, "steam_id")
		if errID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		records, errRecords := getSourceBans(ctx, steamID)
		if errRecords != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, records)
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
	type newMessage struct {
		Message string `json:"message"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banID, errID := getInt64Param(ctx, "ban_id")
		if errID != nil || banID == 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req newMessage
		if !bind(ctx, log, &req) {
			return
		}

		if req.Message == "" {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		bannedPerson := store.NewBannedPerson()
		if errReport := app.db.GetBanByBanID(ctx, banID, &bannedPerson, true); errReport != nil {
			if errors.Is(store.Err(errReport), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to load ban", zap.Error(errReport))

			return
		}

		curUserProfile := currentUserProfile(ctx)
		if bannedPerson.AppealState != store.Open && curUserProfile.PermissionLevel < consts.PModerator {
			responseErr(ctx, http.StatusForbidden, consts.ErrPermissionDenied)
			log.Warn("User tried to bypass posting restriction",
				zap.Int64("ban_id", bannedPerson.BanID), zap.Int64("target_id", bannedPerson.TargetID.Int64()))

			return
		}

		msg := store.NewUserMessage(banID, curUserProfile.SteamID, req.Message)
		if errSave := app.db.SaveBanMessage(ctx, &msg); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save ban appeal message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, msg)

		msgEmbed := discord.
			NewEmbed("New ban appeal message posted").
			SetColor(app.bot.Colour.Info).
			// SetThumbnail(bannedPerson.TargetAvatarhash).
			SetDescription(msg.Contents).
			SetURL(app.ExtURL(bannedPerson.BanSteam))

		app.addAuthorUserProfile(msgEmbed, curUserProfile).Truncate()

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.LogChannelID,
			Embed:     msgEmbed.MessageEmbed,
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
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var existing store.UserMessage
		if errExist := app.db.GetBanMessageByID(ctx, reportMessageID, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		curUser := currentUserProfile(ctx)

		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, consts.PModerator) {
			return
		}

		var req editMessage
		if !bind(ctx, log, &req) {
			return
		}

		if req.BodyMD == "" {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if req.BodyMD == existing.Contents {
			responseErr(ctx, http.StatusConflict, consts.ErrDuplicate)

			return
		}

		existing.Contents = req.BodyMD
		if errSave := app.db.SaveBanMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save ban appeal message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, req)

		msgEmbed := discord.
			NewEmbed("Ban appeal message edited").
			SetDescription(util.DiffString(existing.Contents, req.BodyMD)).
			SetColor(app.bot.Colour.Warn)

		app.addAuthorUserProfile(msgEmbed, curUser).Truncate()

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.LogChannelID,
			Embed:     msgEmbed.MessageEmbed,
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

func onAPIGetPatreonCampaigns(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tiers, errTiers := app.patreon.tiers()
		if errTiers != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		ctx.JSON(http.StatusOK, tiers)
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
		pledges, _, errPledges := app.patreon.pledges()
		if errPledges != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		ctx.JSON(http.StatusOK, pledges)
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

		ctx.JSON(http.StatusNoContent, "")
	}
}

func onAPIDeleteContest(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		contestID, idErr := getUUIDParam(ctx, "contest_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var contest store.Contest

		if errContest := app.db.ContestByID(ctx, contestID, &contest); errContest != nil {
			if errors.Is(errContest, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrUnknownID)

				return
			}

			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			log.Error("Error getting contest for deletion", zap.Error(errContest))

			return
		}

		if errDelete := app.db.ContestDelete(ctx, contest.ContestID); errDelete != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Error deleting contest", zap.Error(errDelete))

			return
		}

		ctx.Status(http.StatusAccepted)

		log.Info("Contest deleted",
			zap.String("contest_id", contestID.String()),
			zap.String("title", contest.Title))
	}
}

func onAPIUpdateContest(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		if _, success := contestFromCtx(ctx, app); !success {
			return
		}

		var contest store.Contest
		if !bind(ctx, log, &contest) {
			return
		}

		if errSave := app.db.ContestSave(ctx, &contest); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Error updating contest", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusAccepted, contest)

		log.Info("Contest updated",
			zap.String("contest_id", contest.ContestID.String()),
			zap.String("title", contest.Title))
	}
}

func onAPIPostContest(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		newContest, _ := store.NewContest("", "", time.Now(), time.Now(), false)
		if !bind(ctx, log, &newContest) {
			return
		}

		if newContest.ContestID.IsNil() {
			newID, errID := uuid.NewV4()
			if errID != nil {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

				return
			}

			newContest.ContestID = newID
		}

		if errSave := app.db.ContestSave(ctx, &newContest); errSave != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		ctx.JSON(http.StatusOK, newContest)
	}
}

func onAPIGetContests(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)
		publicOnly := user.PermissionLevel < consts.PModerator
		contests, errContests := app.db.Contests(ctx, publicOnly)

		if errContests != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, LazyResult{
			Count: int64(len(contests)),
			Data:  contests,
		})
	}
}

func onAPIGetContest(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		contest, success := contestFromCtx(ctx, app)
		if !success {
			return
		}

		ctx.JSON(http.StatusOK, contest)
	}
}

func onAPIGetContestEntries(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		contest, success := contestFromCtx(ctx, app)
		if !success {
			return
		}

		entries, errEntries := app.db.ContestEntries(ctx, contest.ContestID)
		if errEntries != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, entries)
	}
}

func onAPIDeleteContestEntry(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)

		contestEntryID, idErr := getUUIDParam(ctx, "contest_entry_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var entry store.ContestEntry

		if errContest := app.db.ContestEntry(ctx, contestEntryID, &entry); errContest != nil {
			if errors.Is(errContest, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrUnknownID)

				return
			}

			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			log.Error("Error getting contest entry for deletion", zap.Error(errContest))

			return
		}

		// Only >=moderators or the entry author are allowed to delete entries.
		if !(user.PermissionLevel >= consts.PModerator || user.SteamID == entry.SteamID) {
			responseErr(ctx, http.StatusForbidden, consts.ErrPermissionDenied)

			return
		}

		var contest store.Contest

		if errContest := app.db.ContestByID(ctx, entry.ContestID, &contest); errContest != nil {
			if errors.Is(errContest, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrUnknownID)

				return
			}

			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			log.Error("Error getting contest", zap.Error(errContest))

			return
		}

		// Only allow mods to delete entries from expired contests.
		if user.SteamID == entry.SteamID && time.Since(contest.DateEnd) > 0 {
			responseErr(ctx, http.StatusForbidden, consts.ErrPermissionDenied)

			log.Error("User tried to delete entry from expired contest")

			return
		}

		if errDelete := app.db.ContestEntryDelete(ctx, entry.ContestEntryID); errDelete != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Error deleting contest entry", zap.Error(errDelete))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})

		log.Info("Contest deleted",
			zap.String("contest_id", entry.ContestID.String()),
			zap.String("contest_entry_id", entry.ContestEntryID.String()),
			zap.String("title", contest.Title))
	}
}

func onAPIPutPlayerPermission(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type updatePpermissionLevel struct {
		PermissionLevel consts.Privilege
	}

	return func(ctx *gin.Context) {
		steamID, errParam := getSID64Param(ctx, "")
		if errParam != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var req updatePpermissionLevel
		if !bind(ctx, log, &req) {
			return
		}

		var person store.Person
		if errGet := app.db.GetPersonBySteamID(ctx, steamID, &person); errGet != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Failed to load person", zap.Error(errGet))

			return
		}

		if steamID == app.conf.General.Owner {
			responseErr(ctx, http.StatusConflict, errors.New("Cannot alter site owner permissions"))

			return
		}

		person.PermissionLevel = req.PermissionLevel

		if errSave := app.db.SavePerson(ctx, &person); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Failed to save person", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, person)

		log.Info("Player permission updated",
			zap.Int64("steam_id", steamID.Int64()),
			zap.String("permissions", person.PermissionLevel.String()))
	}
}
