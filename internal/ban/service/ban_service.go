package service

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
	"net"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type BanHandler struct {
	du  domain.DiscordUsecase
	bu  domain.BanUsecase
	pu  domain.PersonUsecase
	cu  domain.ConfigUsecase
	log *zap.Logger
}

func NewBanHandler(logger *zap.Logger, engine *gin.Engine, bu domain.BanUsecase, du domain.DiscordUsecase,
	pu domain.PersonUsecase, cu domain.ConfigUsecase) {
	handler := BanHandler{log: logger, bu: bu, du: du, pu: pu, cu: cu}

	engine.GET("/api/stats", handler.onAPIGetStats())
	engine.GET("/export/bans/tf2bd", handler.onAPIExportBansTF2BD())
	engine.GET("/export/bans/valve/steamid", handler.onAPIExportBansValveSteamID())

	// auth
	engine.GET("/api/bans/steam/:ban_id", handler.onAPIGetBanByID())
	engine.GET("/api/sourcebans/:steam_id", handler.onAPIGetSourceBans())

	// mod
	engine.POST("/api/bans/steam", handler.onAPIGetBansSteam())
	engine.POST("/api/bans/steam/create", handler.onAPIPostBanSteamCreate())
	engine.DELETE("/api/bans/steam/:ban_id", handler.onAPIPostBanDelete())
	engine.POST("/api/bans/steam/:ban_id", handler.onAPIPostBanUpdate())
	engine.POST("/api/bans/steam/:ban_id/status", handler.onAPIPostSetBanAppealStatus())

	engine.POST("/api/bans/cidr/create", handler.onAPIPostBansCIDRCreate())
	engine.POST("/api/bans/cidr", handler.onAPIGetBansCIDR())
	engine.DELETE("/api/bans/cidr/:net_id", handler.onAPIDeleteBansCIDR())
	engine.POST("/api/bans/cidr/:net_id", handler.onAPIPostBansCIDRUpdate())

	engine.POST("/api/bans/asn/create", handler.onAPIPostBansASNCreate())
	engine.POST("/api/bans/asn", handler.onAPIGetBansASN())
	engine.DELETE("/api/bans/asn/:asn_id", handler.onAPIDeleteBansASN())
	engine.POST("/api/bans/asn/:asn_id", handler.onAPIPostBansASNUpdate())
}

func (h BanHandler) onAPIPostSetBanAppealStatus() gin.HandlerFunc {
	type setStatusReq struct {
		AppealState domain.AppealState `json:"appeal_state"`
	}

	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banID, banIDErr := http_helper.GetInt64Param(ctx, "ban_id")
		if banIDErr != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req setStatusReq
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		bannedPerson := domain.NewBannedPerson()
		if banErr := h.bu.GetBanByBanID(ctx, banID, &bannedPerson, false); banErr != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		if bannedPerson.AppealState == req.AppealState {
			http_helper.ResponseErr(ctx, http.StatusConflict, domain.ErrStateUnchanged)

			return
		}

		original := bannedPerson.AppealState
		bannedPerson.AppealState = req.AppealState

		if errSave := h.bu.SaveBan(ctx, &bannedPerson.BanSteam); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusAccepted, gin.H{})

		log.Info("Updated ban appeal state",
			zap.Int64("ban_id", banID),
			zap.Int("from_state", int(original)),
			zap.Int("to_state", int(req.AppealState)))
	}
}

type apiBanRequest struct {
	SourceID       domain.StringSID `json:"source_id"`
	TargetID       domain.StringSID `json:"target_id"`
	Duration       string           `json:"duration"`
	ValidUntil     time.Time        `json:"valid_until"`
	BanType        domain.BanType   `json:"ban_type"`
	Reason         domain.Reason    `json:"reason"`
	ReasonText     string           `json:"reason_text"`
	Note           string           `json:"note"`
	ReportID       int64            `json:"report_id"`
	DemoName       string           `json:"demo_name"`
	DemoTick       int              `json:"demo_tick"`
	IncludeFriends bool             `json:"include_friends"`
}

func (h BanHandler) onAPIPostBanSteamCreate() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		var (
			origin   = domain.Web
			sid      = http_helper.CurrentUserProfile(ctx).SteamID
			sourceID = domain.StringSID(sid.String())
		)

		// srcds sourced bans provide a source_id to id the admin
		if req.SourceID != "" {
			sourceID = req.SourceID
			origin = domain.InGame
		}

		duration, errDuration := util.CalcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		var banSteam domain.BanSteam
		if errBanSteam := domain.NewBanSteam(ctx,
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
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if errBan := h.bu.BanSteam(ctx, &banSteam); errBan != nil {
			log.Error("Failed to ban steam profile",
				zap.Error(errBan), zap.Int64("target_id", banSteam.TargetID.Int64()))

			if errors.Is(errBan, domain.ErrDuplicate) {
				http_helper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to save new steam ban", zap.Error(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, banSteam)
	}
}

func (h BanHandler) onAPIGetBanByID() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		curUser := http_helper.CurrentUserProfile(ctx)

		banID, errID := http_helper.GetInt64Param(ctx, "ban_id")
		if errID != nil || banID == 0 {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

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

		bannedPerson := domain.NewBannedPerson()
		if errGetBan := h.bu.GetBanByBanID(ctx, banID, &bannedPerson, deletedOk); errGetBan != nil {
			http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)
			log.Error("Failed to fetch steam ban", zap.Error(errGetBan))

			return
		}

		if !http_helper.CheckPrivilege(ctx, curUser, steamid.Collection{bannedPerson.TargetID}, domain.PModerator) {
			return
		}

		loadBanMeta(&bannedPerson)

		ctx.JSON(http.StatusOK, bannedPerson)
	}
}

func (h BanHandler) onAPIGetSourceBans() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, errID := http_helper.GetSID64Param(ctx, "steam_id")
		if errID != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		records, errRecords := thirdparty.BDSourceBans(ctx, steamID)
		if errRecords != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, records)
	}
}

func (h BanHandler) onAPIGetStats() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var stats domain.Stats
		if errGetStats := h.bu.GetStats(ctx, &stats); errGetStats != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		stats.ServersAlive = 1

		ctx.JSON(http.StatusOK, stats)
	}
}

func (h BanHandler) onAPIExportBansValveSteamID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		bans, _, errBans := h.bu.GetBansSteam(ctx, domain.SteamBansQueryFilter{
			BansQueryFilter: domain.BansQueryFilter{PermanentOnly: true},
		})

		if errBans != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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

func (h BanHandler) onAPIExportSourcemodSimpleAdmins() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		privilegedIds, errPrivilegedIds := h.pu.GetSteamIdsAbove(ctx, domain.PReserved)
		if errPrivilegedIds != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		players, errPlayers := h.pu.GetPeopleBySteamID(ctx, privilegedIds)
		if errPlayers != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		sort.Slice(players, func(i, j int) bool {
			return players[i].PermissionLevel > players[j].PermissionLevel
		})

		bld := strings.Builder{}

		for _, player := range players {
			var perms string

			switch player.PermissionLevel {
			case domain.PAdmin:
				perms = "z"
			case domain.PModerator:
				perms = "abcdefgjk"
			case domain.PEditor:
				perms = "ak"
			case domain.PReserved:
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

func (h BanHandler) onAPIExportBansTF2BD() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// TODO limit / make specialized query since this returns all results
		bans, _, errBans := h.bu.GetBansSteam(ctx, domain.SteamBansQueryFilter{
			BansQueryFilter: domain.BansQueryFilter{
				QueryFilter: domain.QueryFilter{
					Deleted: false,
				},
			},
		})

		if errBans != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		var filtered []domain.BannedSteamPerson

		for _, ban := range bans {
			if ban.Reason != domain.Cheating ||
				ban.Deleted ||
				!ban.IsEnabled {
				continue
			}

			filtered = append(filtered, ban)
		}

		config := h.cu.Config()

		out := thirdparty.TF2BDSchema{
			Schema: "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/schemas/v3/playerlist.schema.json",
			FileInfo: thirdparty.FileInfo{
				Authors:     []string{config.General.SiteName},
				Description: "Players permanently banned for cheating",
				Title:       fmt.Sprintf("%s Cheater List", config.General.SiteName),
				UpdateURL:   h.cu.ExtURLRaw("/export/bans/tf2bd"),
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

func (h BanHandler) onAPIExportBansValveIP() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		bans, _, errBans := h.bu.GetBansNet(ctx, domain.CIDRBansQueryFilter{})
		if errBans != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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

func (h BanHandler) onAPIGetBansSteam() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.SteamBansQueryFilter
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		bans, count, errBans := h.bu.GetBansSteam(ctx, req)
		if errBans != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to fetch steam bans", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, bans))
	}
}

func (h BanHandler) onAPIPostBanDelete() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banID, banIDErr := http_helper.GetInt64Param(ctx, "ban_id")
		if banIDErr != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req apiUnbanRequest
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		bannedPerson := domain.NewBannedPerson()
		if banErr := h.bu.GetBanByBanID(ctx, banID, &bannedPerson, false); banErr != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		changed, errSave := h.bu.Unban(ctx, bannedPerson.TargetID, req.UnbanReasonText)
		if errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		if !changed {
			http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrUnbanFailed)

			return
		}

		ctx.JSON(http.StatusAccepted, gin.H{})
	}
}

func (h BanHandler) onAPIPostBanUpdate() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type updateBanRequest struct {
		TargetID       domain.StringSID `json:"target_id"`
		BanType        domain.BanType   `json:"ban_type"`
		Reason         domain.Reason    `json:"reason"`
		ReasonText     string           `json:"reason_text"`
		Note           string           `json:"note"`
		IncludeFriends bool             `json:"include_friends"`
		ValidUntil     time.Time        `json:"valid_until"`
	}

	return func(ctx *gin.Context) {
		banID, banIDErr := http_helper.GetInt64Param(ctx, "ban_id")
		if banIDErr != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req updateBanRequest
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		if time.Since(req.ValidUntil) > 0 {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		bannedPerson := domain.NewBannedPerson()
		if banErr := h.bu.GetBanByBanID(ctx, banID, &bannedPerson, false); banErr != nil {
			http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

			return
		}

		if req.Reason == domain.Custom {
			if req.ReasonText == "" {
				http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

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

		if errSave := h.bu.SaveBan(ctx, &bannedPerson.BanSteam); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to save updated ban", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusAccepted, bannedPerson)
	}
}

func (h BanHandler) onAPIPostBansCIDRCreate() gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   domain.StringSID `json:"target_id"`
		Duration   string           `json:"duration"`
		Note       string           `json:"note"`
		Reason     domain.Reason    `json:"reason"`
		ReasonText string           `json:"reason_text"`
		CIDR       string           `json:"cidr"`
		ValidUntil time.Time        `json:"valid_until"`
	}

	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		var (
			banCIDR domain.BanCIDR
			sid     = http_helper.CurrentUserProfile(ctx).SteamID
		)

		duration, errDuration := util.CalcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if errBanCIDR := domain.NewBanCIDR(ctx,
			domain.StringSID(sid.String()),
			req.TargetID,
			duration,
			req.Reason,
			req.ReasonText,
			req.Note,
			domain.Web,
			req.CIDR,
			domain.Banned,
			&banCIDR,
		); errBanCIDR != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if errBan := h.bu.BanCIDR(ctx, &banCIDR); errBan != nil {
			if errors.Is(errBan, domain.ErrDuplicate) {
				http_helper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to save cidr ban", zap.Error(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, banCIDR)
	}
}

func (h BanHandler) onAPIGetBansCIDR() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.CIDRBansQueryFilter
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		bans, count, errBans := h.bu.GetBansNet(ctx, req)
		if errBans != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to fetch cidr bans", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, bans))
	}
}

func (h BanHandler) onAPIDeleteBansCIDR() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		netID, netIDErr := http_helper.GetInt64Param(ctx, "net_id")
		if netIDErr != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req apiUnbanRequest
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		var banCidr domain.BanCIDR
		if errFetch := h.bu.GetBanNetByID(ctx, netID, &banCidr); errFetch != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		banCidr.UnbanReasonText = req.UnbanReasonText
		banCidr.Deleted = true

		if errSave := h.bu.SaveBanNet(ctx, &banCidr); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to delete cidr ban", zap.Error(errSave))

			return
		}

		banCidr.NetID = 0

		ctx.JSON(http.StatusOK, banCidr)
	}
}

func (h BanHandler) onAPIPostBansCIDRUpdate() gin.HandlerFunc {
	type apiUpdateBanRequest struct {
		TargetID   domain.StringSID `json:"target_id"`
		Note       string           `json:"note"`
		Reason     domain.Reason    `json:"reason"`
		ReasonText string           `json:"reason_text"`
		CIDR       string           `json:"cidr"`
		ValidUntil time.Time        `json:"valid_until"`
	}

	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		netID, banIDErr := http_helper.GetInt64Param(ctx, "net_id")
		if banIDErr != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var ban domain.BanCIDR

		if errBan := h.bu.GetBanNetByID(ctx, netID, &ban); errBan != nil {
			if errors.Is(errBan, domain.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		var req apiUpdateBanRequest
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		sid, errSID := req.TargetID.SID64(ctx)
		if errSID != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		if req.Reason == domain.Custom && req.ReasonText == "" {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		_, ipNet, errParseCIDR := net.ParseCIDR(req.CIDR)
		if errParseCIDR != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		ban.Reason = req.Reason
		ban.ReasonText = req.ReasonText
		ban.CIDR = ipNet.String()
		ban.Note = req.Note
		ban.ValidUntil = req.ValidUntil
		ban.TargetID = sid

		if errSave := h.bu.SaveBanNet(ctx, &ban); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}

func (h BanHandler) onAPIPostBansASNCreate() gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   domain.StringSID `json:"target_id"`
		Note       string           `json:"note"`
		Reason     domain.Reason    `json:"reason"`
		ReasonText string           `json:"reason_text"`
		ASNum      int64            `json:"as_num"`
		Duration   string           `json:"duration"`
		ValidUntil time.Time        `json:"valid_until"`
	}

	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		var (
			banASN domain.BanASN
			sid    = http_helper.CurrentUserProfile(ctx).SteamID
		)

		duration, errDuration := util.CalcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if errBanSteamGroup := domain.NewBanASN(ctx,
			domain.StringSID(sid.String()),
			req.TargetID,
			duration,
			req.Reason,
			req.ReasonText,
			req.Note,
			domain.Web,
			req.ASNum,
			domain.Banned,
			&banASN,
		); errBanSteamGroup != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if errBan := h.bu.BanASN(ctx, &banASN); errBan != nil {
			if errors.Is(errBan, domain.ErrDuplicate) {
				http_helper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			log.Error("Failed to save asn ban", zap.Error(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, banASN)
	}
}

func (h BanHandler) onAPIGetBansASN() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.ASNBansQueryFilter
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		bansASN, count, errBans := h.bu.GetBansASN(ctx, req)
		if errBans != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to fetch banASN", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, domain.NewLazyResult(count, bansASN))
	}
}

func (h BanHandler) onAPIDeleteBansASN() gin.HandlerFunc {
	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		asnID, asnIDErr := http_helper.GetInt64Param(ctx, "asn_id")
		if asnIDErr != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req apiUnbanRequest
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		var banAsn domain.BanASN
		if errFetch := h.bu.GetBanASN(ctx, asnID, &banAsn); errFetch != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		banAsn.UnbanReasonText = req.UnbanReasonText
		banAsn.Deleted = true

		if errSave := h.bu.SaveBanASN(ctx, &banAsn); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to delete asn ban", zap.Error(errSave))

			return
		}

		banAsn.BanASNId = 0

		ctx.JSON(http.StatusOK, banAsn)
	}
}

func (h BanHandler) onAPIPostBansASNUpdate() gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   domain.StringSID `json:"target_id"`
		Note       string           `json:"note"`
		Reason     domain.Reason    `json:"reason"`
		ReasonText string           `json:"reason_text"`
		ValidUntil time.Time        `json:"valid_until"`
	}

	log := h.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		asnID, asnIDErr := http_helper.GetInt64Param(ctx, "asn_id")
		if asnIDErr != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var ban domain.BanASN
		if errBan := h.bu.GetBanASN(ctx, asnID, &ban); errBan != nil {
			if errors.Is(errBan, domain.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		var req apiBanRequest
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		if ban.Reason == domain.Custom && req.ReasonText == "" {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		sid, errSID := req.TargetID.SID64(ctx)
		if errSID != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		ban.Note = req.Note
		ban.ValidUntil = req.ValidUntil
		ban.TargetID = sid
		ban.Reason = req.Reason
		ban.ReasonText = req.ReasonText

		if errSave := h.bu.SaveBanASN(ctx, &ban); errSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}

func (h BanHandler) onAPIPostBanState() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportID, errID := http_helper.GetInt64Param(ctx, "report_id")
		if errID != nil || reportID <= 0 {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var report domain.Report
		if errReport := env.Store().GetReport(ctx, reportID, &report); errReport != nil {
			if errors.Is(errReport, errs.ErrNoResult) {
				http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		h.du.SendPayload(domain.ChannelModLog, discord.EditBanAppealStatusMessage())
	}
}

type apiUnbanRequest struct {
	UnbanReasonText string `json:"unban_reason_text"`
}
