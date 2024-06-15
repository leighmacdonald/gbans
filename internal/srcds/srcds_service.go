package srcds

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type srcdsHandler struct {
	srcds     domain.SRCDSUsecase
	servers   domain.ServersUsecase
	persons   domain.PersonUsecase
	state     domain.StateUsecase
	discord   domain.DiscordUsecase
	config    domain.ConfigUsecase
	reports   domain.ReportUsecase
	assets    domain.AssetUsecase
	bans      domain.BanSteamUsecase
	bansGroup domain.BanGroupUsecase
	bansASN   domain.BanASNUsecase
	bansNet   domain.BanNetUsecase
	network   domain.NetworkUsecase
	demos     domain.DemoUsecase
	blocklist domain.BlocklistUsecase
}

func NewSRCDSHandler(engine *gin.Engine, srcds domain.SRCDSUsecase, servers domain.ServersUsecase,
	persons domain.PersonUsecase, assets domain.AssetUsecase, reports domain.ReportUsecase,
	bans domain.BanSteamUsecase, network domain.NetworkUsecase, bansGroup domain.BanGroupUsecase,
	demos domain.DemoUsecase, auth domain.AuthUsecase, bansASNU domain.BanASNUsecase, bansNet domain.BanNetUsecase,
	config domain.ConfigUsecase, discord domain.DiscordUsecase, state domain.StateUsecase,
	blocklist domain.BlocklistUsecase,
) {
	handler := srcdsHandler{
		srcds:     srcds,
		servers:   servers,
		persons:   persons,
		reports:   reports,
		bans:      bans,
		assets:    assets,
		network:   network,
		bansGroup: bansGroup,
		demos:     demos,
		bansASN:   bansASNU,
		bansNet:   bansNet,
		config:    config,
		discord:   discord,
		state:     state,
		blocklist: blocklist,
	}

	adminGroup := engine.Group("/")
	{
		admin := adminGroup.Use(auth.AuthMiddleware(domain.PAdmin))
		// Groups
		admin.GET("/api/smadmin/groups", handler.onAPISMGroups())
		admin.POST("/api/smadmin/groups", handler.onCreateSMGroup())
		admin.POST("/api/smadmin/groups/:group_id", handler.onSaveSMGroup())
		admin.DELETE("/api/smadmin/groups/:group_id", handler.onDeleteSMGroup())
		admin.GET("/api/smadmin/groups/:group_id/overrides", handler.onGroupOverrides())
		admin.POST("/api/smadmin/groups/:group_id/overrides", handler.onCreateGroupOverride())
		admin.POST("/api/smadmin/groups_overrides/:group_override_id", handler.onSaveGroupOverride())
		admin.DELETE("/api/smadmin/groups_overrides/:group_override_id", handler.onDeleteGroupOverride())

		// Admins
		admin.GET("/api/smadmin/admins", handler.onGetSMAdmins())
		admin.POST("/api/smadmin/admins", handler.onCreateSMAdmin())
		admin.POST("/api/smadmin/admins/:admin_id", handler.onSaveSMAdmin())
		admin.DELETE("/api/smadmin/admins/:admin_id", handler.onDeleteSMAdmin())
		admin.POST("/api/smadmin/admins/:admin_id/groups", handler.onAddAdminGroup())
		admin.DELETE("/api/smadmin/admins/:admin_id/groups/:group_id", handler.onDeleteAdminGroup())

		// Global overrides
		admin.GET("/api/smadmin/overrides", handler.onGetOverrides())
		admin.POST("/api/smadmin/overrides", handler.onCreateOverrides())
		admin.POST("/api/smadmin/overrides/:override_id", handler.onSaveOverrides())
		admin.DELETE("/api/smadmin/overrides/:override_id", handler.onDeleteOverrides())

		// Group Immunities
		admin.GET("/api/smadmin/group_immunity", handler.onGetGroupImmunities())
		admin.POST("/api/smadmin/group_immunity", handler.onCreateGroupImmunity())
		admin.DELETE("/api/smadmin/group_immunity/:group_immunity_id", handler.onDeleteGroupImmunity())
	}

	// Endpoints called by sourcemod plugin
	srcdsGroup := engine.Group("/")
	{
		server := srcdsGroup.Use(auth.AuthServerMiddleWare())
		server.POST("/api/sm/check", handler.onAPICheckPlayer())
		server.GET("/api/sm/overrides", handler.onAPIGetServerOverrides())
		server.GET("/api/sm/users", handler.onAPIGetServerUsers())
		server.GET("/api/sm/groups", handler.onAPIGetServerGroups())
		server.POST("/api/sm/ping_mod", handler.onAPIPostPingMod())

		// Duplicated since we need to authenticate via server middleware
		server.POST("/api/sm/bans/steam/create", handler.onAPIPostBanSteamCreate())
		server.POST("/api/sm/report/create", handler.onAPIPostReportCreate())
		server.POST("/api/state_update", handler.onAPIPostServerState())
	}
}

type ServerAuthResp struct {
	Status bool   `json:"status"`
	Token  string `json:"token"`
}

func (s *srcdsHandler) onAPICheckPlayer() gin.HandlerFunc {
	type checkRequest struct {
		domain.SteamIDField
		ClientID int        `json:"client_id"`
		IP       netip.Addr `json:"ip"`
		Name     string     `json:"name"`
	}

	type checkResponse struct {
		ClientID int            `json:"client_id"`
		BanType  domain.BanType `json:"ban_type"`
		Msg      string         `json:"msg"`
	}

	return func(ctx *gin.Context) {
		var (
			currentUser = httphelper.CurrentUserProfile(ctx)
			req         checkRequest
		)

		if !httphelper.Bind(ctx, &req) {
			slog.Error("Failed to bind check request")

			return
		}

		defaultValue := checkResponse{
			ClientID: req.ClientID,
			BanType:  domain.OK,
			Msg:      "",
		}

		steamID, valid := req.SteamID(ctx)
		if !valid {
			ctx.JSON(http.StatusOK, defaultValue)
			slog.Error("Did not receive valid steamid for check response", log.ErrAttr(domain.ErrInvalidSID))

			return
		}

		banState, msg, errBS := s.srcds.GetBanState(ctx, steamID, req.IP)
		if errBS != nil {
			slog.Error("failed to get ban state", log.ErrAttr(errBS))

			// Fail Open
			ctx.JSON(http.StatusOK, defaultValue)

			return
		}

		if banState.BanID != 0 {
			player, errPlayer := s.persons.GetOrCreatePersonBySteamID(ctx, steamID)
			if errPlayer != nil {
				slog.Error("Failed to load or create player on connect")
				ctx.JSON(http.StatusOK, defaultValue)

				return
			}

			if banState.SteamID != steamID && !banState.EvadeOK {
				evadeBanned, err := s.bans.CheckEvadeStatus(ctx, currentUser, steamID, req.IP)
				if err != nil {
					ctx.JSON(http.StatusOK, defaultValue)

					return
				}

				if evadeBanned {
					defaultValue = checkResponse{
						ClientID: req.ClientID,
						BanType:  domain.Banned,
						Msg:      "Evasion ban",
					}

					ctx.JSON(http.StatusOK, defaultValue)

					s.discord.SendPayload(domain.ChannelKickLog,
						discord.KickPlayerOnConnectEmbed(steamID, req.Name, player, banState.BanSource))

					return
				}
			}

			if banState.SteamID != steamID && banState.EvadeOK {
				ctx.JSON(http.StatusOK, defaultValue)

				return
			}

			ctx.JSON(http.StatusOK, checkResponse{
				ClientID: req.ClientID,
				BanType:  banState.BanType,
				Msg:      msg,
			})

			return
		}

		ctx.JSON(http.StatusOK, defaultValue)
	}
}

func (s *srcdsHandler) onGetGroupImmunities() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		immunities, errImmunities := s.srcds.GetGroupImmunities(ctx)
		if errImmunities != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get group immunities", log.ErrAttr(errImmunities))

			return
		}

		if immunities == nil {
			immunities = []domain.SMGroupImmunity{}
		}

		ctx.JSON(http.StatusOK, immunities)
	}
}

type groupImmunityRequest struct {
	GroupID int `json:"group_id"`
	OtherID int `json:"other_id"`
}

func (s *srcdsHandler) onCreateGroupImmunity() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req groupImmunityRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		immunity, errImmunity := s.srcds.AddGroupImmunity(ctx, req.GroupID, req.OtherID)
		if errImmunity != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to add group immunity", log.ErrAttr(errImmunity))

			return
		}

		ctx.JSON(http.StatusOK, immunity)
	}
}

func (s *srcdsHandler) onDeleteGroupImmunity() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupImmunityID, errID := httphelper.GetIntParam(ctx, "group_immunity_id")
		if errID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get group_immunity_id", log.ErrAttr(errID))

			return
		}

		if err := s.srcds.DelGroupImmunity(ctx, groupImmunityID); err != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to delete group immunity", log.ErrAttr(err), slog.Int("group_immunity_id", groupImmunityID))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

type groupRequest struct {
	GroupID int `json:"group_id"`
}

func (s *srcdsHandler) onGroupOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupID, errGroupID := httphelper.GetIntParam(ctx, "group_id")
		if errGroupID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Got invalid group_id", log.ErrAttr(errGroupID))

			return
		}

		overrides, errOverrides := s.srcds.GroupOverrides(ctx, groupID)
		if errOverrides != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get group overrides", log.ErrAttr(errOverrides))

			return
		}

		if overrides == nil {
			overrides = []domain.SMGroupOverrides{}
		}

		ctx.JSON(http.StatusOK, overrides)
	}
}

type groupOverrideRequest struct {
	Name   string                `json:"name"`
	Type   domain.OverrideType   `json:"type"`
	Access domain.OverrideAccess `json:"access"`
}

func (s *srcdsHandler) onCreateGroupOverride() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupID, errGroupID := httphelper.GetIntParam(ctx, "group_id")
		if errGroupID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get group_id", log.ErrAttr(errGroupID))

			return
		}

		var req groupOverrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errOverride := s.srcds.AddGroupOverride(ctx, groupID, req.Name, req.Type, req.Access)
		if errOverride != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to add group override", log.ErrAttr(errOverride))

			return
		}

		ctx.JSON(http.StatusOK, override)
	}
}

func (s *srcdsHandler) onSaveGroupOverride() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupOverrideID, errGroupID := httphelper.GetIntParam(ctx, "group_override_id")
		if errGroupID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get group_override_id", log.ErrAttr(errGroupID))

			return
		}

		var req groupOverrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errOverride := s.srcds.GetGroupOverride(ctx, groupOverrideID)
		if errOverride != nil {
			if errors.Is(errOverride, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get group override", log.ErrAttr(errOverride))

			return
		}

		override.Type = req.Type
		override.Name = req.Name
		override.Access = req.Access

		edited, errSave := s.srcds.SaveGroupOverride(ctx, override)
		if errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to save group override", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, edited)
	}
}

func (s *srcdsHandler) onDeleteGroupOverride() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupOverrideID, errGroupID := httphelper.GetIntParam(ctx, "group_override_id")
		if errGroupID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Got invalid group_override_id", log.ErrAttr(errGroupID))

			return
		}

		if err := s.srcds.DelGroupOverride(ctx, groupOverrideID); err != nil {
			if errors.Is(err, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to delete group override", log.ErrAttr(err))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

type overrideRequest struct {
	Name  string              `json:"name"`
	Type  domain.OverrideType `json:"type"`
	Flags string              `json:"flags"`
}

func (s *srcdsHandler) onSaveOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		overrideID, errOverrideID := httphelper.GetIntParam(ctx, "override_id")
		if errOverrideID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to ver override_id", log.ErrAttr(errOverrideID))

			return
		}

		var req overrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errOverride := s.srcds.GetOverride(ctx, overrideID)
		if errOverride != nil {
			if errors.Is(errOverride, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get override", log.ErrAttr(errOverride))

			return
		}

		override.Type = req.Type
		override.Name = req.Name
		override.Flags = req.Flags

		edited, errSave := s.srcds.SaveOverride(ctx, override)
		if errSave != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Error("Failed to save override", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, edited)
	}
}

func (s *srcdsHandler) onCreateOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req overrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errCreate := s.srcds.AddOverride(ctx, req.Name, req.Type, req.Flags)
		if errCreate != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to add override", log.ErrAttr(errCreate))

			return
		}

		ctx.JSON(http.StatusOK, override)
	}
}

func (s *srcdsHandler) onDeleteOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		override, errOverride := httphelper.GetIntParam(ctx, "override_id")
		if errOverride != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get override_id", log.ErrAttr(errOverride))

			return
		}

		if errCreate := s.srcds.DelOverride(ctx, override); errCreate != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to delete override", log.ErrAttr(errCreate))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (s *srcdsHandler) onGetOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		overrides, errOverrides := s.srcds.Overrides(ctx)
		if errOverrides != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get overrides", log.ErrAttr(errOverrides))

			return
		}

		if overrides == nil {
			overrides = []domain.SMOverrides{}
		}

		ctx.JSON(http.StatusOK, overrides)
	}
}

func (s *srcdsHandler) onAddAdminGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminID, errAdminID := httphelper.GetIntParam(ctx, "admin_id")
		if errAdminID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get admin_id", log.ErrAttr(errAdminID))

			return
		}

		var req groupRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		admin, err := s.srcds.AddAdminGroup(ctx, adminID, req.GroupID)
		if err != nil {
			httphelper.HandleErrInternal(ctx)

			return
		}

		ctx.JSON(http.StatusOK, admin)
	}
}

func (s *srcdsHandler) onDeleteAdminGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminID, errAdminID := httphelper.GetIntParam(ctx, "admin_id")
		if errAdminID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get admin_id", log.ErrAttr(errAdminID))

			return
		}

		groupID, errGroupID := httphelper.GetIntParam(ctx, "group_id")
		if errGroupID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get group_id", log.ErrAttr(errGroupID))

			return
		}

		admin, errDel := s.srcds.DelAdminGroup(ctx, adminID, groupID)
		if errDel != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to delete admin group", log.ErrAttr(errDel))

			return
		}

		ctx.JSON(http.StatusOK, admin)
	}
}

func (s *srcdsHandler) onSaveSMAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminID, errAdminID := httphelper.GetIntParam(ctx, "admin_id")
		if errAdminID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get admin_id", log.ErrAttr(errAdminID))

			return
		}

		admin, errAdmin := s.srcds.GetAdminByID(ctx, adminID)
		if errAdmin != nil {
			if errors.Is(errAdmin, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get admin by id", log.ErrAttr(errAdmin))

			return
		}

		var req smAdminRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		admin.Name = req.Name
		admin.Flags = req.Flags
		admin.Immunity = req.Immunity
		admin.AuthType = req.AuthType
		admin.Identity = req.Identity
		admin.Password = req.Password

		editedGroup, errSave := s.srcds.SaveAdmin(ctx, admin)
		if errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to save admin", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, editedGroup)
	}
}

func (s *srcdsHandler) onDeleteSMAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminID, errAdminID := httphelper.GetIntParam(ctx, "admin_id")
		if errAdminID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get admin_id", log.ErrAttr(errAdminID))

			return
		}

		if err := s.srcds.DelAdmin(ctx, adminID); err != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to delete admin", log.ErrAttr(err))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

type smAdminRequest struct {
	AuthType domain.AuthType `json:"auth_type"`
	Identity string          `json:"identity"`
	Password string          `json:"password"`
	Flags    string          `json:"flags"`
	Name     string          `json:"name"`
	Immunity int             `json:"immunity"`
}

func (s *srcdsHandler) onCreateSMAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req smAdminRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		admin, errAdmin := s.srcds.AddAdmin(ctx, req.Name, req.AuthType, req.Identity, req.Flags, req.Immunity, req.Password)
		if errAdmin != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Error("Failed to add admin", log.ErrAttr(errAdmin))

			return
		}

		ctx.JSON(http.StatusOK, admin)
	}
}

func (s *srcdsHandler) onDeleteSMGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupID, errGroupID := httphelper.GetIntParam(ctx, "group_id")
		if errGroupID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get group_id", log.ErrAttr(errGroupID))

			return
		}

		if err := s.srcds.DelGroup(ctx, groupID); err != nil {
			if errors.Is(err, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to delete group", log.ErrAttr(err))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

type smGroupRequest struct {
	Name     string `json:"name"`
	Immunity int    `json:"immunity"`
	Flags    string `json:"flags"`
}

func (s *srcdsHandler) onSaveSMGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupID, errGroupID := httphelper.GetIntParam(ctx, "group_id")
		if errGroupID != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get group_id", log.ErrAttr(errGroupID))

			return
		}

		group, errGroup := s.srcds.GetGroupByID(ctx, groupID)
		if errGroup != nil {
			if errors.Is(errGroup, domain.ErrNoResult) {
				httphelper.HandleErrNotFound(ctx)

				return
			}

			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get group by id", log.ErrAttr(errGroup))

			return
		}

		var req smGroupRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		group.Name = req.Name
		group.Flags = req.Flags
		group.ImmunityLevel = req.Immunity

		editedGroup, errSave := s.srcds.SaveGroup(ctx, group)
		if errSave != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to save group", log.ErrAttr(errSave))

			return
		}

		ctx.JSON(http.StatusOK, editedGroup)
	}
}

func (s *srcdsHandler) onCreateSMGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req smGroupRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		group, errGroup := s.srcds.AddGroup(ctx, req.Name, req.Flags, req.Immunity)
		if errGroup != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to add group", log.ErrAttr(errGroup))

			return
		}

		ctx.JSON(http.StatusCreated, group)
	}
}

func (s *srcdsHandler) onAPISMGroups() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groups, errGroups := s.srcds.Groups(ctx)
		if errGroups != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get groups", log.ErrAttr(errGroups))

			return
		}

		ctx.JSON(http.StatusOK, groups)
	}
}

func (s *srcdsHandler) onGetSMAdmins() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		admins, errAdmins := s.srcds.Admins(ctx)
		if errAdmins != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get admins", log.ErrAttr(errAdmins))

			return
		}

		if admins == nil {
			admins = []domain.SMAdmin{}
		}

		ctx.JSON(http.StatusOK, admins)
	}
}

func (s *srcdsHandler) onAPIPostServerState() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.PartialStateUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		serverID, err := httphelper.GetIntParam(ctx, "server_id")
		if err != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Warn("Failed to get server_id", log.ErrAttr(err))

			return
		}

		if errUpdate := s.state.Update(serverID, req); errUpdate != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to update server", log.ErrAttr(errUpdate))

			return
		}

		ctx.AbortWithStatus(http.StatusNoContent)
	}
}

func (s *srcdsHandler) onAPIPostReportCreate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser := httphelper.CurrentUserProfile(ctx)

		var req domain.CreateReportReq
		if !httphelper.Bind(ctx, &req) {
			return
		}

		report, errReport := s.srcds.Report(ctx, currentUser, req)
		if errReport != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to create report", log.ErrAttr(errReport))

			return
		}

		ctx.JSON(http.StatusCreated, report)
	}
}

type apiSMBanRequest struct {
	domain.SourceIDField
	domain.TargetIDField
	Duration       int            `json:"duration"`
	ValidUntil     time.Time      `json:"valid_until"`
	BanType        domain.BanType `json:"ban_type"`
	Reason         domain.Reason  `json:"reason"`
	ReasonText     string         `json:"reason_text"`
	Note           string         `json:"note"`
	ReportID       int64          `json:"report_id"`
	DemoName       string         `json:"demo_name"`
	DemoTick       int            `json:"demo_tick"`
	IncludeFriends bool           `json:"include_friends"`
}

func (s *srcdsHandler) onAPIPostBanSteamCreate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req apiSMBanRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		var (
			origin   = domain.InGame
			curUser  = httphelper.CurrentUserProfile(ctx)
			sourceID steamid.SteamID
		)

		// srcds sourced bans provide a source_id to id the admin
		if sid, valid := req.SourceSteamID(ctx); valid {
			sourceID = sid
		} else {
			sourceID = steamid.New(s.config.Config().Owner)
		}

		targetID, valid := req.TargetSteamID(ctx)
		if !valid {
			httphelper.HandleErrBadRequest(ctx)
			slog.Error("SM sent invalid target ID", slog.String("target_id", req.TargetID))

			return
		}

		duration := time.Hour * 24 * 365 * 10
		if req.Duration > 0 {
			duration = time.Duration(req.Duration) * time.Second
		}

		var banSteam domain.BanSteam
		if errBanSteam := domain.NewBanSteam(sourceID, targetID, duration, req.Reason, req.ReasonText, req.Note, origin,
			req.ReportID, req.BanType, req.IncludeFriends, false, &banSteam); errBanSteam != nil {
			httphelper.HandleErrBadRequest(ctx)
			slog.Error("Failed to create new ban", log.ErrAttr(errBanSteam))

			return
		}

		if errBan := s.bans.Ban(ctx, curUser, &banSteam); errBan != nil {
			if errors.Is(errBan, domain.ErrDuplicate) {
				httphelper.ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			slog.Error("Failed to save new steam ban", log.ErrAttr(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, banSteam)
	}
}

func (s *srcdsHandler) onAPIGetServerOverrides() gin.HandlerFunc {
	type smOverride struct {
		Type  domain.OverrideType `json:"type"`
		Name  string              `json:"name"`
		Flags string              `json:"flags"`
	}

	return func(ctx *gin.Context) {
		overrides, errOverrides := s.srcds.Overrides(ctx)
		if errOverrides != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get overrides", log.ErrAttr(errOverrides))

			return
		}

		//goland:noinspection ALL
		smOverrides := []smOverride{}
		for _, group := range overrides {
			smOverrides = append(smOverrides, smOverride{
				Flags: group.Flags,
				Name:  group.Name,
				Type:  group.Type,
			})
		}

		ctx.JSON(http.StatusOK, smOverrides)
	}
}

func (s *srcdsHandler) onAPIGetServerGroups() gin.HandlerFunc {
	type smGroup struct {
		Flags         string `json:"flags"`
		Name          string `json:"name"`
		ImmunityLevel int    `json:"immunity_level"`
	}

	type smGroupImmunity struct {
		GroupName string `json:"group_name"`
		OtherName string `json:"other_name"`
	}

	type smGroupsResp struct {
		Groups     []smGroup         `json:"groups"`
		Immunities []smGroupImmunity `json:"immunities"`
	}

	return func(ctx *gin.Context) {
		groups, errGroups := s.srcds.Groups(ctx)
		if errGroups != nil && !errors.Is(errGroups, domain.ErrNoResult) {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get groups", log.ErrAttr(errGroups))

			return
		}

		immunities, errImmunities := s.srcds.GetGroupImmunities(ctx)
		if errImmunities != nil && !errors.Is(errImmunities, domain.ErrNoResult) {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get group immunities", log.ErrAttr(errImmunities))

			return
		}

		resp := smGroupsResp{
			// Make sure we return an empty list instead of null
			Groups:     []smGroup{},
			Immunities: []smGroupImmunity{},
		}

		//goland:noinspection ALL
		for _, group := range groups {
			resp.Groups = append(resp.Groups, smGroup{
				Flags:         group.Flags,
				Name:          group.Name,
				ImmunityLevel: group.ImmunityLevel,
			})
		}

		for _, immunity := range immunities {
			resp.Immunities = append(resp.Immunities, smGroupImmunity{
				GroupName: immunity.Group.Name,
				OtherName: immunity.Other.Name,
			})
		}

		ctx.JSON(http.StatusOK, resp)
	}
}

func (s *srcdsHandler) onAPIGetServerUsers() gin.HandlerFunc {
	type smUser struct {
		ID       int             `json:"id"`
		Authtype domain.AuthType `json:"authtype"`
		Identity string          `json:"identity"`
		Password string          `json:"password"`
		Flags    string          `json:"flags"`
		Name     string          `json:"name"`
		Immunity int             `json:"immunity"`
	}

	type smUserGroup struct {
		AdminID   int    `json:"admin_id"`
		GroupName string `json:"group_name"`
	}

	type smUsersResponse struct {
		Users      []smUser      `json:"users"`
		UserGroups []smUserGroup `json:"user_groups"`
	}

	return func(ctx *gin.Context) {
		users, errUsers := s.srcds.Admins(ctx)
		if errUsers != nil {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to get admins", log.ErrAttr(errUsers))

			return
		}

		smResp := smUsersResponse{
			Users:      []smUser{},
			UserGroups: []smUserGroup{},
		}

		for _, user := range users {
			smResp.Users = append(smResp.Users, smUser{
				ID:       user.AdminID,
				Authtype: user.AuthType,
				Identity: user.Identity,
				Password: user.Password,
				Flags:    user.Flags,
				Name:     user.Name,
				Immunity: user.Immunity,
			})

			for _, ug := range user.Groups {
				smResp.UserGroups = append(smResp.UserGroups, smUserGroup{
					AdminID:   user.AdminID,
					GroupName: ug.Name,
				})
			}
		}

		ctx.JSON(http.StatusOK, smResp)
	}
}

type pingReq struct {
	ServerName string          `json:"server_name"`
	Name       string          `json:"name"`
	SteamID    steamid.SteamID `json:"steam_id"`
	Reason     string          `json:"reason"`
	Client     int             `json:"client"`
}

func (s *srcdsHandler) onAPIPostPingMod() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req pingReq
		if !httphelper.Bind(ctx, &req) {
			return
		}

		conf := s.config.Config()
		players := s.state.FindBySteamID(req.SteamID)

		if len(players) == 0 && conf.General.Mode != domain.TestMode {
			httphelper.HandleErrInternal(ctx)
			slog.Error("Failed to find player on /mod call")

			return
		}

		ctx.JSON(http.StatusOK, gin.H{"client": req.Client, "message": "Moderators have been notified"})

		author, err := s.persons.GetOrCreatePersonBySteamID(ctx, req.SteamID)
		if err != nil {
			slog.Error("Failed to load user", log.ErrAttr(err))

			return
		}

		server, errServer := s.servers.GetServer(ctx, players[0].ServerID)
		if errServer != nil {
			slog.Error("Failed to load server", log.ErrAttr(errServer))

			return
		}

		var connect string

		if addr, errIP := server.IP(ctx); errIP != nil {
			slog.Error("Failed to resolve server ip", log.ErrAttr(errIP))
		} else {
			connect = fmt.Sprintf("steam://connect/%s:%d", addr.String(), server.Port)
		}

		s.discord.SendPayload(domain.ChannelMod,
			discord.PingModMessage(author, conf.ExtURL(author), req.Reason, server, conf.Discord.ModPingRoleID, connect))

		if errSay := s.state.PSay(ctx, author.SteamID, "Moderators have been notified"); errSay != nil {
			slog.Error("Failed to reply to user", log.ErrAttr(errSay))
		}
	}
}
