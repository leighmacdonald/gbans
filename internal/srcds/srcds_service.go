package srcds

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type srcdsHandler struct {
	srcdsUsecase     domain.SRCDSUsecase
	serverUsecase    domain.ServersUsecase
	personUsecase    domain.PersonUsecase
	stateUsecase     domain.StateUsecase
	discordUsecase   domain.DiscordUsecase
	configUsecase    domain.ConfigUsecase
	reportUsecase    domain.ReportUsecase
	assetUsecase     domain.AssetUsecase
	banUsecase       domain.BanSteamUsecase
	banGroupUsecase  domain.BanGroupUsecase
	banASNUsecase    domain.BanASNUsecase
	banNetUsecase    domain.BanNetUsecase
	networkUsecase   domain.NetworkUsecase
	demoUsecase      domain.DemoUsecase
	blocklistUsecase domain.BlocklistUsecase
}

func NewSRCDSHandler(engine *gin.Engine, srcdsUsecase domain.SRCDSUsecase, serversUsecase domain.ServersUsecase,
	personUsecase domain.PersonUsecase, assetUsecase domain.AssetUsecase, reportUsecase domain.ReportUsecase,
	banUsecase domain.BanSteamUsecase, networkUsecase domain.NetworkUsecase, banGroupUsecase domain.BanGroupUsecase,
	demoUsecase domain.DemoUsecase, authUsecase domain.AuthUsecase, banASNUsecase domain.BanASNUsecase, banNetUsecase domain.BanNetUsecase,
	configUsecase domain.ConfigUsecase, discordUsecase domain.DiscordUsecase, stateUsecase domain.StateUsecase,
	blocklistUsecase domain.BlocklistUsecase,
) {
	handler := srcdsHandler{
		srcdsUsecase:     srcdsUsecase,
		serverUsecase:    serversUsecase,
		personUsecase:    personUsecase,
		reportUsecase:    reportUsecase,
		banUsecase:       banUsecase,
		assetUsecase:     assetUsecase,
		networkUsecase:   networkUsecase,
		banGroupUsecase:  banGroupUsecase,
		demoUsecase:      demoUsecase,
		banASNUsecase:    banASNUsecase,
		banNetUsecase:    banNetUsecase,
		configUsecase:    configUsecase,
		discordUsecase:   discordUsecase,
		stateUsecase:     stateUsecase,
		blocklistUsecase: blocklistUsecase,
	}

	adminGroup := engine.Group("/")
	{
		admin := adminGroup.Use(authUsecase.AuthMiddleware(domain.PAdmin))
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
		server := srcdsGroup.Use(authUsecase.AuthServerMiddleWare())
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
		SteamID  string `json:"steam_id"`
		ClientID int    `json:"client_id"`
		IP       net.IP `json:"ip"`
		Name     string `json:"name"`
	}

	type checkResponse struct {
		ClientID int            `json:"client_id"`
		BanType  domain.BanType `json:"ban_type"`
		Msg      string         `json:"msg"`
	}

	return func(ctx *gin.Context) {
		var req checkRequest
		if !httphelper.Bind(ctx, &req) {
			slog.Error("Failed to bind check request")

			return
		}

		steamID := steamid.New(req.SteamID)
		if !steamID.Valid() {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrInvalidSID)

			return
		}

		banState, msg, errBS := s.srcdsUsecase.GetBanState(ctx, steamID, req.IP)
		if errBS != nil {
			slog.Error("failed to get ban state", log.ErrAttr(errBS))

			// Fail Open
			ctx.JSON(http.StatusOK, checkResponse{})

			return
		}

		if banState.BanID == 0 {
			ctx.JSON(http.StatusOK, checkResponse{})

			return
		}

		ctx.JSON(http.StatusOK, checkResponse{
			ClientID: req.ClientID,
			BanType:  banState.BanType,
			Msg:      msg,
		})
	}
}

func (s *srcdsHandler) onGetGroupImmunities() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		immunities, errImmunities := s.srcdsUsecase.GetGroupImmunities(ctx)
		if errImmunities != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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

		immunity, errImmunity := s.srcdsUsecase.AddGroupImmunity(ctx, req.GroupID, req.OtherID)
		if errImmunity != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, immunity)
	}
}

func (s *srcdsHandler) onDeleteGroupImmunity() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupImmunityID, errID := httphelper.GetIntParam(ctx, "group_immunity_id")
		if errID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if err := s.srcdsUsecase.DelGroupImmunity(ctx, groupImmunityID); err != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		overrides, errOverrides := s.srcdsUsecase.GroupOverrides(ctx, groupID)
		if errOverrides != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		var req groupOverrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errOverride := s.srcdsUsecase.AddGroupOverride(ctx, groupID, req.Name, req.Type, req.Access)
		if errOverride != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, override)
	}
}

func (s *srcdsHandler) onSaveGroupOverride() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupOverrideID, errGroupID := httphelper.GetIntParam(ctx, "group_override_id")
		if errGroupID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		var req groupOverrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errOverride := s.srcdsUsecase.GetGroupOverride(ctx, groupOverrideID)
		if errOverride != nil {
			if errors.Is(errOverride, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		override.Type = req.Type
		override.Name = req.Name
		override.Access = req.Access

		edited, errSave := s.srcdsUsecase.SaveGroupOverride(ctx, override)
		if errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, edited)
	}
}

func (s *srcdsHandler) onDeleteGroupOverride() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupOverrideID, errGroupID := httphelper.GetIntParam(ctx, "group_override_id")
		if errGroupID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if err := s.srcdsUsecase.DelGroupOverride(ctx, groupOverrideID); err != nil {
			if errors.Is(err, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		var req overrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errOverride := s.srcdsUsecase.GetOverride(ctx, overrideID)
		if errOverride != nil {
			if errors.Is(errOverride, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		override.Type = req.Type
		override.Name = req.Name
		override.Flags = req.Flags

		edited, errSave := s.srcdsUsecase.SaveOverride(ctx, override)
		if errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

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

		override, errCreate := s.srcdsUsecase.AddOverride(ctx, req.Name, req.Type, req.Flags)
		if errCreate != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, override)
	}
}

func (s *srcdsHandler) onDeleteOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		override, errOverride := httphelper.GetIntParam(ctx, "override_id")
		if errOverride != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if errCreate := s.srcdsUsecase.DelOverride(ctx, override); errCreate != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (s *srcdsHandler) onGetOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		overrides, errOverrides := s.srcdsUsecase.Overrides(ctx)
		if errOverrides != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		var req groupRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		admin, err := s.srcdsUsecase.AddAdminGroup(ctx, adminID, req.GroupID)
		if err != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, admin)
	}
}

func (s *srcdsHandler) onDeleteAdminGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminID, errAdminID := httphelper.GetIntParam(ctx, "admin_id")
		if errAdminID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		groupID, errGroupID := httphelper.GetIntParam(ctx, "group_id")
		if errGroupID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		admin, errDel := s.srcdsUsecase.DelAdminGroup(ctx, adminID, groupID)
		if errDel != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, admin)
	}
}

func (s *srcdsHandler) onSaveSMAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminID, errAdminID := httphelper.GetIntParam(ctx, "admin_id")
		if errAdminID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		admin, errAdmin := s.srcdsUsecase.GetAdminByID(ctx, adminID)
		if errAdmin != nil {
			if errors.Is(errAdmin, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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

		editedGroup, errSave := s.srcdsUsecase.SaveAdmin(ctx, admin)
		if errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, editedGroup)
	}
}

func (s *srcdsHandler) onDeleteSMAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminID, errAdminID := httphelper.GetIntParam(ctx, "admin_id")
		if errAdminID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if err := s.srcdsUsecase.DelAdmin(ctx, adminID); err != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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

		admin, errAdmin := s.srcdsUsecase.AddAdmin(ctx, req.Name, req.AuthType, req.Identity, req.Flags, req.Immunity, req.Password)
		if errAdmin != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		ctx.JSON(http.StatusOK, admin)
	}
}

func (s *srcdsHandler) onDeleteSMGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupID, errGroupID := httphelper.GetIntParam(ctx, "group_id")
		if errGroupID != nil {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if err := s.srcdsUsecase.DelGroup(ctx, groupID); err != nil {
			if errors.Is(err, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

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
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		group, errGroup := s.srcdsUsecase.GetGroupByID(ctx, groupID)
		if errGroup != nil {
			if errors.Is(errGroup, domain.ErrNoResult) {
				httphelper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

				return
			}

			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		var req smGroupRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		group.Name = req.Name
		group.Flags = req.Flags
		group.ImmunityLevel = req.Immunity

		editedGroup, errSave := s.srcdsUsecase.SaveGroup(ctx, group)
		if errSave != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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

		group, errGroup := s.srcdsUsecase.AddGroup(ctx, req.Name, req.Flags, req.Immunity)
		if errGroup != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, group)
	}
}

func (s *srcdsHandler) onAPISMGroups() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groups, errGroups := s.srcdsUsecase.Groups(ctx)
		if errGroups != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, groups)
	}
}

func (s *srcdsHandler) onGetSMAdmins() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		admins, errAdmins := s.srcdsUsecase.Admins(ctx)
		if errAdmins != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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

		serverID := httphelper.ServerIDFromCtx(ctx) // TODO use generic func for int
		if serverID == 0 {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrParamInvalid)

			return
		}

		if errUpdate := s.stateUsecase.Update(serverID, req); errUpdate != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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

		report, errReport := s.srcdsUsecase.Report(ctx, currentUser, req)
		if errReport != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, errReport)
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
			sourceID = steamid.New(s.configUsecase.Config().General.Owner)
		}

		targetID, valid := req.TargetSteamID(ctx)
		if !valid {
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)
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
			httphelper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if errBan := s.banUsecase.Ban(ctx, curUser, &banSteam); errBan != nil {
			slog.Error("Failed to ban steam profile",
				log.ErrAttr(errBan), slog.Int64("target_id", banSteam.TargetID.Int64()))

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
		overrides, errOverrides := s.srcdsUsecase.Overrides(ctx)
		if errOverrides != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
		groups, errGroups := s.srcdsUsecase.Groups(ctx)
		if errGroups != nil && !errors.Is(errGroups, domain.ErrNoResult) {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		immunities, errImmunities := s.srcdsUsecase.GetGroupImmunities(ctx)
		if errImmunities != nil && !errors.Is(errImmunities, domain.ErrNoResult) {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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
		users, errUsers := s.srcdsUsecase.Admins(ctx)
		if errUsers != nil {
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

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

		conf := s.configUsecase.Config()
		players := s.stateUsecase.FindBySteamID(req.SteamID)

		if len(players) == 0 && conf.General.Mode != domain.TestMode {
			slog.Error("Failed to find player on /mod call")
			httphelper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, gin.H{"client": req.Client, "message": "Moderators have been notified"})

		if !conf.Discord.Enabled {
			return
		}

		author, err := s.personUsecase.GetOrCreatePersonBySteamID(ctx, req.SteamID)
		if err != nil {
			slog.Error("Failed to load user", log.ErrAttr(err))

			return
		}

		server, errServer := s.serverUsecase.GetServer(ctx, players[0].ServerID)
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

		s.discordUsecase.SendPayload(domain.ChannelMod,
			discord.PingModMessage(author, conf.ExtURL(author), req.Reason, server, conf.Discord.ModPingRoleID, connect))

		if errSay := s.stateUsecase.PSay(ctx, author.SteamID, "Moderators have been notified"); errSay != nil {
			slog.Error("Failed to reply to user", log.ErrAttr(errSay))
		}
	}
}
