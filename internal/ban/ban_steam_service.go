package ban

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/exp/slices"
)

type banHandler struct {
	discord   domain.DiscordUsecase
	bansSteam domain.BanSteamUsecase
	persons   domain.PersonUsecase
	config    domain.ConfigUsecase
}

func NewHandlerSteam(engine *gin.Engine, bu domain.BanSteamUsecase, du domain.DiscordUsecase,
	pu domain.PersonUsecase, configUsecase domain.ConfigUsecase, ath domain.AuthUsecase,
) {
	handler := banHandler{bansSteam: bu, discord: du, persons: pu, config: configUsecase}

	if configUsecase.Config().Exports.BDEnabled {
		engine.GET("/export/bans/tf2bd", handler.onAPIExportBansTF2BD())
	}

	if configUsecase.Config().Exports.ValveEnabled {
		engine.GET("/export/bans/valve/steamid", handler.onAPIExportBansValveSteamID())
	}

	// auth
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(ath.Middleware(domain.PUser))
		authed.GET("/api/bans/steam/:ban_id", handler.onAPIGetBanByID())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.Middleware(domain.PModerator))

		mod.GET("/api/sourcebans/:steam_id", handler.onAPIGetSourceBans())
		mod.GET("/api/stats", handler.onAPIGetStats())
		mod.GET("/api/bans/steam", handler.onAPIGetBansSteam())
		mod.GET("/api/bans/steam_all/:steam_id", handler.onAPIGetBansSteamBySteamID())
		mod.GET("/api/bans/steamid/:steam_id", handler.onAPIGetBanBySteam())
		mod.POST("/api/bans/steam/create", handler.onAPIPostBanSteamCreate())
		mod.DELETE("/api/bans/steam/:ban_id", handler.onAPIPostBanDelete())
		mod.POST("/api/bans/steam/:ban_id", handler.onAPIPostBanUpdate())
		mod.POST("/api/bans/steam/:ban_id/status", handler.onAPIPostSetBanAppealStatus())
	}
}

func (h banHandler) onAPIPostSetBanAppealStatus() gin.HandlerFunc {
	type setStatusReq struct {
		AppealState domain.AppealState `json:"appeal_state"`
	}

	return func(ctx *gin.Context) {
		banID, idFound := httphelper.GetInt64Param(ctx, "ban_id")
		if !idFound {
			return
		}

		var req setStatusReq
		if !httphelper.Bind(ctx, &req) {
			return
		}

		bannedPerson, banErr := h.bansSteam.GetByBanID(ctx, banID, false, true)
		if banErr != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, banErr))

			return
		}

		if bannedPerson.AppealState == req.AppealState {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusConflict, domain.ErrStateUnchanged))

			return
		}

		original := bannedPerson.AppealState
		bannedPerson.AppealState = req.AppealState

		if errSave := h.bansSteam.Save(ctx, &bannedPerson.BanSteam); errSave != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSave))

			return
		}

		if req.AppealState == domain.Accepted {
			if _, err := h.bansSteam.Unban(ctx, bannedPerson.TargetID, "Appeal accepted", httphelper.CurrentUserProfile(ctx)); err != nil {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, err))

				return
			}
		}

		ctx.JSON(http.StatusAccepted, gin.H{})

		slog.Info("Updated ban appeal state",
			slog.Int64("ban_id", banID),
			slog.Int("from_state", int(original)),
			slog.Int("to_state", int(req.AppealState)))
	}
}

func (h banHandler) onAPIPostBanSteamCreate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.RequestBanSteamCreate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		ban, errBan := h.bansSteam.Ban(ctx, httphelper.CurrentUserProfile(ctx), domain.Web, req)
		if errBan != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errBan))

			return
		}

		ctx.JSON(http.StatusCreated, ban)
		slog.Info("New steam ban created", slog.Int64("ban_id", ban.BanID), slog.String("steam_id", ban.TargetID.String()))
	}
}

func (h banHandler) onAPIGetBanByID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		curUser := httphelper.CurrentUserProfile(ctx)

		banID, idFound := httphelper.GetInt64Param(ctx, "ban_id")
		if !idFound {
			return
		}

		if banID == 0 {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusBadRequest, domain.ErrBadRequest))

			return
		}

		deletedOk := false

		fullValue, fullOk := ctx.GetQuery("deleted")
		if fullOk {
			deleted, deletedOkErr := strconv.ParseBool(fullValue)
			if deletedOkErr != nil {
				slog.Error("Failed to parse ban full query value", log.ErrAttr(deletedOkErr))
			} else {
				deletedOk = deleted
			}
		}

		bannedPerson, errGet := h.bansSteam.GetByBanID(ctx, banID, deletedOk, true)
		if errGet != nil {
			if errors.Is(errGet, domain.ErrNoResult) {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusNotFound, domain.ErrNoResult))

				return
			}

			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errGet))

			return
		}

		if !httphelper.HasPrivilege(curUser, steamid.Collection{bannedPerson.TargetID}, domain.PModerator) {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusForbidden, domain.ErrPermissionDenied))

			return
		}

		ctx.JSON(http.StatusOK, bannedPerson)
	}
}

func (h banHandler) onAPIGetSourceBans() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, idFound := httphelper.GetSID64Param(ctx, "steam_id")
		if !idFound {
			return
		}

		records, errRecords := thirdparty.BDSourceBans(ctx, steamID)
		if errRecords != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errRecords))

			return
		}

		userRecords, found := records[steamID.Int64()]
		if !found {
			ctx.JSON(http.StatusOK, []thirdparty.BDSourceBansRecord{})

			return
		}

		ctx.JSON(http.StatusOK, userRecords)
	}
}

func (h banHandler) onAPIGetStats() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var stats domain.Stats
		if errGetStats := h.bansSteam.Stats(ctx, &stats); errGetStats != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errGetStats))

			return
		}

		stats.ServersAlive = 1

		ctx.JSON(http.StatusOK, stats)
	}
}

func (h banHandler) onAPIExportBansValveSteamID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizedKeys := strings.Split(h.config.Config().Exports.AuthorizedKeys, ",")

		if len(authorizedKeys) > 0 {
			key, ok := ctx.GetQuery("key")
			if !ok || !slices.Contains(authorizedKeys, key) {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusForbidden, domain.ErrPermissionDenied))

				return
			}
		}

		// TODO limit to perm?
		bans, errBans := h.bansSteam.Get(ctx, domain.SteamBansQueryFilter{})
		if errBans != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errBans))

			return
		}

		var entries strings.Builder

		for _, ban := range bans {
			if ban.Deleted ||
				!ban.IsEnabled {
				continue
			}

			entries.WriteString(fmt.Sprintf("banid 0 %s\n", ban.TargetID.Steam(false)))
		}

		ctx.Data(http.StatusOK, "text/plain", []byte(entries.String()))
	}
}

func (h banHandler) onAPIExportBansTF2BD() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizedKeys := strings.Split(h.config.Config().Exports.AuthorizedKeys, ",")

		if len(authorizedKeys) > 0 {
			key, ok := ctx.GetQuery("key")
			if !ok || !slices.Contains(authorizedKeys, key) {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusForbidden, domain.ErrPermissionDenied))

				return
			}
		}

		bans, errBans := h.bansSteam.Get(ctx, domain.SteamBansQueryFilter{})
		if errBans != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errBans))

			return
		}

		var filtered []domain.BannedSteamPerson

		for _, ban := range bans {
			if ban.Reason != domain.Cheating || ban.Deleted || !ban.IsEnabled {
				continue
			}

			filtered = append(filtered, ban)
		}

		config := h.config.Config()

		out := thirdparty.TF2BDSchema{
			Schema: "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/schemas/v3/playerlist.schema.json",
			FileInfo: thirdparty.FileInfo{
				Authors:     []string{config.General.SiteName},
				Description: "Players permanently banned for cheating",
				Title:       config.General.SiteName + " Cheater List",
				UpdateURL:   h.config.ExtURLRaw("/export/bans/tf2bd"),
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

func (h banHandler) onAPIGetBansSteam() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var params domain.SteamBansQueryFilter
		if !httphelper.BindQuery(ctx, &params) {
			return
		}

		bans, errBans := h.bansSteam.Get(ctx, params)
		if errBans != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errBans))

			return
		}

		ctx.JSON(http.StatusOK, bans)
	}
}

func (h banHandler) onAPIGetBansSteamBySteamID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, idFound := httphelper.GetSID64Param(ctx, "steam_id")
		if !idFound {
			return
		}

		params := domain.SteamBansQueryFilter{
			TargetIDField: domain.TargetIDField{TargetID: sid.String()},
			Deleted:       true,
		}
		bans, errBans := h.bansSteam.Get(ctx, params)
		if errBans != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errBans))

			return
		}

		ctx.JSON(http.StatusOK, bans)
	}
}

func (h banHandler) onAPIPostBanDelete() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banID, idFound := httphelper.GetInt64Param(ctx, "ban_id")
		if !idFound {
			return
		}

		var req domain.RequestUnban
		if !httphelper.Bind(ctx, &req) {
			return
		}

		bannedPerson, errBan := h.bansSteam.GetByBanID(ctx, banID, false, true)
		if errBan != nil {
			if errors.Is(errBan, domain.ErrNoResult) {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusNotFound, domain.ErrNotFound))

				return
			}

			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errBan))

			return
		}

		changed, errSave := h.bansSteam.Unban(ctx, bannedPerson.TargetID, req.UnbanReasonText, httphelper.CurrentUserProfile(ctx))
		if errSave != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSave))

			return
		}

		if !changed {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusOK, domain.ErrUnbanFailed))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

type RequestBanSteamUpdate struct {
	TargetID       steamid.SteamID `json:"target_id"`
	BanType        domain.BanType  `json:"ban_type"`
	Reason         domain.Reason   `json:"reason"`
	ReasonText     string          `json:"reason_text"`
	Note           string          `json:"note"`
	IncludeFriends bool            `json:"include_friends"`
	EvadeOk        bool            `json:"evade_ok"`
	ValidUntil     time.Time       `json:"valid_until"`
}

func (h banHandler) onAPIPostBanUpdate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banID, idFound := httphelper.GetInt64Param(ctx, "ban_id")
		if !idFound {
			return
		}

		var req RequestBanSteamUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if time.Since(req.ValidUntil) > 0 {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusBadRequest, domain.ErrBadRequest))

			return
		}

		bannedPerson, banErr := h.bansSteam.GetByBanID(ctx, banID, false, true)
		if banErr != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusNotFound, domain.ErrNotFound))

			return
		}

		if req.Reason == domain.Custom {
			if req.ReasonText == "" {
				_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusBadRequest, domain.ErrBadRequest))

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
		bannedPerson.EvadeOk = req.EvadeOk
		bannedPerson.ValidUntil = req.ValidUntil

		if errSave := h.bansSteam.Save(ctx, &bannedPerson.BanSteam); errSave != nil {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errSave))

			return
		}

		ctx.JSON(http.StatusOK, bannedPerson)
	}
}

func (h banHandler) onAPIGetBanBySteam() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, idFound := httphelper.GetSID64Param(ctx, "steam_id")
		if !idFound {
			return
		}

		ban, errBans := h.bansSteam.GetBySteamID(ctx, steamID, false, false)
		if errBans != nil && !errors.Is(errBans, domain.ErrNoResult) {
			_ = ctx.Error(httphelper.NewAPIError(ctx, http.StatusInternalServerError, errBans))

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}
