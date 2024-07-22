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

func NewBanHandler(engine *gin.Engine, bu domain.BanSteamUsecase, du domain.DiscordUsecase,
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
		authed := authedGrp.Use(ath.AuthMiddleware(domain.PUser))
		authed.GET("/api/bans/steam/:ban_id", handler.onAPIGetBanByID())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(ath.AuthMiddleware(domain.PModerator))

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
		banID, banIDErr := httphelper.GetInt64Param(ctx, "ban_id")
		if banIDErr != nil {
			httphelper.ResponseApiErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req setStatusReq
		if !httphelper.Bind(ctx, &req) {
			return
		}

		bannedPerson, banErr := h.bansSteam.GetByBanID(ctx, banID, false, true)
		if banErr != nil {
			httphelper.ResponseApiErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		if bannedPerson.AppealState == req.AppealState {
			httphelper.ResponseApiErr(ctx, http.StatusConflict, domain.ErrStateUnchanged)

			return
		}

		original := bannedPerson.AppealState
		bannedPerson.AppealState = req.AppealState

		if errSave := h.bansSteam.Save(ctx, &bannedPerson.BanSteam); errSave != nil {
			httphelper.ResponseApiErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
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
			httphelper.HandleErrs(ctx, errBan)
			slog.Error("Failed to save new steam ban", log.ErrAttr(errBan), slog.String("steam_id", req.TargetID))

			return
		}

		ctx.JSON(http.StatusCreated, ban)
		slog.Info("New steam ban created", slog.Int64("ban_id", ban.BanID), slog.String("steam_id", ban.TargetID.String()))
	}
}

func (h banHandler) onAPIGetBanByID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		curUser := httphelper.CurrentUserProfile(ctx)

		banID, errID := httphelper.GetInt64Param(ctx, "ban_id")
		if errID != nil || banID == 0 {
			httphelper.ResponseApiErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

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
				httphelper.HandleErrNotFound(ctx)

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get requested ban", log.ErrAttr(errGet))

			return
		}

		if !httphelper.HasPrivilege(curUser, steamid.Collection{bannedPerson.TargetID}, domain.PModerator) {
			return
		}

		ctx.JSON(http.StatusOK, bannedPerson)
	}
}

func (h banHandler) onAPIGetSourceBans() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, errID := httphelper.GetSID64Param(ctx, "steam_id")
		if errID != nil {
			httphelper.ResponseApiErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		records, errRecords := thirdparty.BDSourceBans(ctx, steamID)
		if errRecords != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Error querying bdapi sourcebans", log.ErrAttr(errRecords), slog.String("steam_id", steamID.String()))

			return
		}

		ctx.JSON(http.StatusOK, records)
	}
}

func (h banHandler) onAPIGetStats() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var stats domain.Stats
		if errGetStats := h.bansSteam.Stats(ctx, &stats); errGetStats != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to fetch ban stats", log.ErrAttr(errGetStats))

			return
		}

		stats.ServersAlive = 1

		ctx.JSON(http.StatusOK, stats)
	}
}

func (h banHandler) onAPIExportBansValveSteamID() gin.HandlerFunc {
	authorizedKeys := h.config.Config().Exports.AuthorizedKeys

	return func(ctx *gin.Context) {
		if len(authorizedKeys) > 0 {
			key, ok := ctx.GetQuery("key")
			if !ok || !slices.Contains(authorizedKeys, key) {
				httphelper.ResponseApiErr(ctx, http.StatusForbidden, domain.ErrPermissionDenied)

				return
			}
		}

		// TODO limit to perm?
		bans, errBans := h.bansSteam.Get(ctx, domain.SteamBansQueryFilter{})
		if errBans != nil {
			httphelper.ResponseApiErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
	authorizedKeys := h.config.Config().Exports.AuthorizedKeys

	return func(ctx *gin.Context) {
		if len(authorizedKeys) > 0 {
			key, ok := ctx.GetQuery("key")
			if !ok || !slices.Contains(authorizedKeys, key) {
				httphelper.ResponseApiErr(ctx, http.StatusForbidden, domain.ErrPermissionDenied)

				return
			}
		}

		bans, errBans := h.bansSteam.Get(ctx, domain.SteamBansQueryFilter{})

		if errBans != nil {
			httphelper.ResponseApiErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
			httphelper.ResponseApiErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to fetch steam bans", log.ErrAttr(errBans))

			return
		}

		ctx.JSON(http.StatusOK, bans)
	}
}

func (h banHandler) onAPIGetBansSteamBySteamID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sid, errSID := httphelper.GetSID64Param(ctx, "steam_id")
		if errSID != nil {
			httphelper.HandleErrs(ctx, errSID)
			slog.Warn("Got invalid steam_id param", log.ErrAttr(errSID))

			return
		}

		params := domain.SteamBansQueryFilter{
			TargetIDField: domain.TargetIDField{TargetID: sid.String()},
			Deleted:       true,
		}
		bans, errBans := h.bansSteam.Get(ctx, params)
		if errBans != nil {
			httphelper.ResponseApiErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to fetch steam bans", log.ErrAttr(errBans))

			return
		}

		ctx.JSON(http.StatusOK, bans)
	}
}

func (h banHandler) onAPIPostBanDelete() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banID, banIDErr := httphelper.GetInt64Param(ctx, "ban_id")
		if banIDErr != nil {
			httphelper.ResponseApiErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req domain.RequestUnban
		if !httphelper.Bind(ctx, &req) {
			return
		}

		bannedPerson, banErr := h.bansSteam.GetByBanID(ctx, banID, false, true)
		if banErr != nil {
			if errors.Is(banErr, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)

				return
			}

			httphelper.HandleErrInternal(ctx)

			return
		}

		changed, errSave := h.bansSteam.Unban(ctx, bannedPerson.TargetID, req.UnbanReasonText)
		if errSave != nil {
			httphelper.HandleErrInternal(ctx)

			return
		}

		if !changed {
			httphelper.ResponseApiErr(ctx, http.StatusOK, domain.ErrUnbanFailed)

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
		banID, banIDErr := httphelper.GetInt64Param(ctx, "ban_id")
		if banIDErr != nil {
			httphelper.ResponseApiErr(ctx, http.StatusBadRequest, domain.ErrInvalidParameter)

			return
		}

		var req RequestBanSteamUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		if time.Since(req.ValidUntil) > 0 {
			httphelper.ResponseApiErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		bannedPerson, banErr := h.bansSteam.GetByBanID(ctx, banID, false, true)
		if banErr != nil {
			httphelper.ResponseApiErr(ctx, http.StatusNotFound, domain.ErrNotFound)

			return
		}

		if req.Reason == domain.Custom {
			if req.ReasonText == "" {
				httphelper.ResponseApiErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

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
			httphelper.ResponseApiErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to save updated ban", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, bannedPerson)
	}
}

func (h banHandler) onAPIGetBanBySteam() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, err := httphelper.GetSID64Param(ctx, "steam_id")
		if err != nil {
			httphelper.ResponseApiErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)
			slog.Error("Failed to get steamid", log.ErrAttr(err))

			return
		}

		ban, errBans := h.bansSteam.GetBySteamID(ctx, steamID, false, false)
		if errBans != nil && !errors.Is(errBans, domain.ErrNoResult) {
			httphelper.ResponseApiErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to get ban record for steamid", log.ErrAttr(errBans))

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}
