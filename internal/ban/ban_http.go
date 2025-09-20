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
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"golang.org/x/exp/slices"
)

type banHandler struct {
	bans   Bans
	config *config.Configuration
}

func NewHandlerSteam(engine *gin.Engine, bans Bans,
	config *config.Configuration, authenticator httphelper.Authenticator,
) {
	handler := banHandler{bans: bans, config: config}

	if config.Config().Exports.BDEnabled {
		engine.GET("/export/bans/tf2bd", handler.onAPIExportBansTF2BD())
	}

	if config.Config().Exports.ValveEnabled {
		engine.GET("/export/bans/valve/steamid", handler.onAPIExportBansValveSteamID())
	}

	// auth
	authedGrp := engine.Group("/")
	{
		authed := authedGrp.Use(authenticator.Middleware(permission.PUser))
		authed.GET("/api/bans/:ban_id", handler.onAPIGetBanByID())
	}

	// mod
	modGrp := engine.Group("/")
	{
		mod := modGrp.Use(authenticator.Middleware(permission.PModerator))

		mod.GET("/api/sourcebans/:steam_id", handler.onAPIGetSourceBans())
		mod.GET("/api/stats", handler.onAPIGetStats())
		mod.GET("/api/bans/query", handler.onAPIGetBans())
		mod.POST("/api/bans/create", handler.onAPIPostBanSteamCreate())
		mod.GET("/api/bans/all/:steam_id", handler.onAPIGetBansSteamBySteamID())
		mod.GET("/api/bans/steamid/:steam_id", handler.onAPIGetBanBySteam())
		mod.DELETE("/api/ban/:ban_id", handler.onAPIPostBanDelete())
		mod.POST("/api/ban/:ban_id", handler.onAPIPostBanUpdate())
		mod.POST("/api/ban/:ban_id/status", handler.onAPIPostSetBanAppealStatus())
	}
}

func (h banHandler) onAPIPostSetBanAppealStatus() gin.HandlerFunc {
	type setStatusReq struct {
		AppealState AppealState `json:"appeal_state"`
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

		bannedPerson, banErr := h.bans.QueryOne(ctx, QueryOpts{BanID: banID, EvadeOk: true})
		if banErr != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, banErr))

			return
		}

		if bannedPerson.AppealState == req.AppealState {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusConflict, httphelper.ErrBadRequest,
				"New state must be different than previous state"))

			return
		}

		original := bannedPerson.AppealState
		bannedPerson.AppealState = req.AppealState

		if errSave := h.bans.Save(ctx, &bannedPerson); errSave != nil {
			switch {
			case errors.Is(errSave, domain.ErrPersonTarget):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrBadRequest,
					"Ban target steam_id invalid"))
			case errors.Is(errSave, domain.ErrPersonSource):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrBadRequest,
					"Ban author steam_id invalid"))
			case errors.Is(errSave, database.ErrDuplicate):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, httphelper.ErrBadRequest,
					"Ban typ (nocomm/ban/network) cannot be the same as existng ban"))
			case errors.Is(errSave, ErrGetBan):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusNotFound, httphelper.ErrNotFound, "Could not load ban to update"))
			default:
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal)))
			}

			return
		}

		if req.AppealState == Accepted {
			user, _ := session.CurrentUserProfile(ctx)
			if _, err := h.bans.Unban(ctx, bannedPerson.TargetID, "Appeal accepted", user); err != nil {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusInternalServerError, errors.Join(err, httphelper.ErrInternal),
					"Could not perform unban request"))

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
		var req BanOpts
		if !httphelper.Bind(ctx, &req) {
			return
		}

		user, _ := session.CurrentUserProfile(ctx)
		if !req.SourceID.Valid() {
			req.SourceID = user.GetSteamID()
		}

		newBan, errBan := h.bans.Create(ctx, req)
		if errBan != nil {
			switch {
			case errors.Is(errBan, database.ErrDuplicate):
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusConflict, database.ErrDuplicate,
					"Ban already active for steam_id: %s", req.TargetID.String()))
			default:
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errBan, httphelper.ErrInternal)))
			}

			return
		}

		ctx.JSON(http.StatusCreated, newBan)
		slog.Info("New steam ban created", slog.Int64("ban_id", newBan.BanID), slog.String("steam_id", newBan.TargetID.String()))
	}
}

func (h banHandler) onAPIGetBanByID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, _ := session.CurrentUserProfile(ctx)
		banID, idFound := httphelper.GetInt64Param(ctx, "ban_id")
		if !idFound {
			return
		}

		if banID == 0 {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrBadRequest,
				"Ban ID must be > 0"))

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
		bannedPerson, errGet := h.bans.QueryOne(ctx, QueryOpts{
			BanID:   banID,
			Deleted: deletedOk,
			EvadeOk: true,
		})
		if errGet != nil {
			if errors.Is(errGet, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, database.ErrNoResult))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errGet, httphelper.ErrInternal)))

			return
		}

		if !httphelper.HasPrivilege(user, steamid.Collection{bannedPerson.TargetID}, permission.PModerator) {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusForbidden, httphelper.ErrPermissionDenied,
				"You do not have permission to access this ban."))

			return
		}

		ctx.JSON(http.StatusOK, bannedPerson)
	}
}

type BDSourceBansRecord struct {
	BanID       int             `json:"ban_id"`
	SiteName    string          `json:"site_name"`
	SiteID      int             `json:"site_id"`
	PersonaName string          `json:"persona_name"`
	SteamID     steamid.SteamID `json:"steam_id"`
	Reason      string          `json:"reason"`
	Duration    time.Duration   `json:"duration"`
	Permanent   bool            `json:"permanent"`
	CreatedOn   time.Time       `json:"created_on"`
}

func (h banHandler) onAPIGetSourceBans() gin.HandlerFunc {
	client, errClient := thirdparty.NewClientWithResponses("https://tf-api.roto.lol")
	if errClient != nil {
		panic(errClient)
	}

	return func(ctx *gin.Context) {
		steamID, idFound := httphelper.GetSID64Param(ctx, "steam_id")
		if !idFound {
			return
		}

		records := []BDSourceBansRecord{}

		resp, errResp := client.BansSearchWithResponse(ctx, &thirdparty.BansSearchParams{Steamids: steamID.String()})
		if errResp != nil {
			return
		}

		if resp.JSON200 != nil {
			for _, ban := range *resp.JSON200 {
				records = append(records, BDSourceBansRecord{
					SiteName:    ban.SiteName,
					SiteID:      0,
					PersonaName: ban.Name,
					SteamID:     steamid.New(ban.SteamId),
					Reason:      ban.Reason,
					Duration:    ban.ExpiresOn.Sub(ban.CreatedOn),
					Permanent:   ban.Permanent,
					CreatedOn:   ban.CreatedOn,
				})
			}
		}

		ctx.JSON(http.StatusOK, records)
	}
}

func (h banHandler) onAPIGetStats() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var stats Stats
		if errGetStats := h.bans.Stats(ctx, &stats); errGetStats != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errGetStats, httphelper.ErrInternal)))

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
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusForbidden, httphelper.ErrPermissionDenied,
					"You do not have permission to access this resource. You can try contacting the administrator to obtain an api key."))

				return
			}
		}

		// TODO limit to perm?
		bans, errBans := h.bans.Query(ctx, QueryOpts{})
		if errBans != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errBans, httphelper.ErrInternal)))

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
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusForbidden, httphelper.ErrPermissionDenied,
					"You do not have permission to access this resource. You can try contacting the administrator to obtain an api key."))

				return
			}
		}

		bans, errBans := h.bans.Query(ctx, QueryOpts{})
		if errBans != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errBans, httphelper.ErrInternal)))

			return
		}

		var filtered []Ban

		for _, curBan := range bans {
			if curBan.Reason != ban.Cheating || curBan.Deleted || !curBan.IsEnabled {
				continue
			}

			filtered = append(filtered, curBan)
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
					PlayerName: ban.TargetID.String(),
					Time:       int(ban.UpdatedOn.Unix()),
				},
			})
		}

		ctx.JSON(http.StatusOK, out)
	}
}

func (h banHandler) onAPIGetBans() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var params BansQueryFilter
		if !httphelper.BindQuery(ctx, &params) {
			return
		}

		bans, errBans := h.bans.Query(ctx, QueryOpts{Deleted: params.Deleted})
		if errBans != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errBans, httphelper.ErrInternal)))

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

		bans, errBans := h.bans.Query(ctx, QueryOpts{TargetID: sid, Deleted: true})
		if errBans != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errBans, httphelper.ErrInternal)))

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

		var req RequestUnban
		if !httphelper.Bind(ctx, &req) {
			return
		}

		bannedPerson, errBan := h.bans.QueryOne(ctx, QueryOpts{BanID: banID, EvadeOk: true})
		if errBan != nil {
			if errors.Is(errBan, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, httphelper.ErrNotFound))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errBan, httphelper.ErrInternal)))

			return
		}

		user, _ := session.CurrentUserProfile(ctx)
		changed, errSave := h.bans.Unban(ctx, bannedPerson.TargetID, req.UnbanReasonText, user)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errors.Join(errSave, httphelper.ErrInternal)))

			return
		}

		if !changed {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusOK, domain.ErrUnbanFailed, "Ban status is unchanged"))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

type RequestBanSteamUpdate struct {
	TargetID   steamid.SteamID `json:"target_id"`
	BanType    ban.BanType     `json:"ban_type"`
	Reason     ban.Reason      `json:"reason"`
	ReasonText string          `json:"reason_text"`
	Note       string          `json:"note"`
	EvadeOk    bool            `json:"evade_ok"`
	ValidUntil time.Time       `json:"valid_until"`
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
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrBadRequest,
				"Valid until date cannot be in the past."))

			return
		}

		bannedPerson, banErr := h.bans.QueryOne(ctx, QueryOpts{BanID: banID, Deleted: true, EvadeOk: true})
		if banErr != nil {
			httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusNotFound, httphelper.ErrNotFound,
				"Failed to find existing ban with id: %d", banID))

			return
		}

		if req.Reason == ban.Custom {
			if req.ReasonText == "" {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusBadRequest, httphelper.ErrBadRequest,
					"Reason cannot be empty."))

				return
			}

			bannedPerson.ReasonText = req.ReasonText
		} else {
			bannedPerson.ReasonText = ""
		}

		bannedPerson.Note = req.Note
		bannedPerson.BanType = req.BanType
		bannedPerson.Reason = req.Reason
		bannedPerson.EvadeOk = req.EvadeOk
		bannedPerson.ValidUntil = req.ValidUntil

		if errSave := h.bans.Save(ctx, &bannedPerson); errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errSave))

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

		ban, errBans := h.bans.Query(ctx, QueryOpts{TargetID: steamID})
		if errBans != nil && !errors.Is(errBans, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errBans))

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}
