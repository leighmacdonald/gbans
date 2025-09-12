package servers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/blocklist"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type srcdsHandler struct {
	srcds         *SRCDSUsecase
	servers       ServersUsecase
	persons       person.PersonUsecase
	state         StateUsecase
	notifications notification.NotificationUsecase
	config        *config.ConfigUsecase
	reports       ban.ReportUsecase
	assets        asset.AssetUsecase
	bans          ban.BanUsecase
	network       network.NetworkUsecase
	blocklist     blocklist.BlocklistUsecase
}

func NewHandlerSRCDS(engine *gin.Engine, srcds *SRCDSUsecase, servers ServersUsecase,
	persons person.PersonUsecase, assets asset.AssetUsecase, reports ban.ReportUsecase,
	bans ban.BanUsecase, network network.NetworkUsecase, auth httphelper.Authenticator,
	config *config.ConfigUsecase, notifications notification.NotificationUsecase, state StateUsecase,
	blocklist blocklist.BlocklistUsecase,
) {
	handler := srcdsHandler{
		srcds:         srcds,
		servers:       servers,
		persons:       persons,
		reports:       reports,
		bans:          bans,
		assets:        assets,
		network:       network,
		config:        config,
		notifications: notifications,
		state:         state,
		blocklist:     blocklist,
	}

	adminGroup := engine.Group("/")
	{
		admin := adminGroup.Use(auth.Middleware(permission.PAdmin))
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
		server := srcdsGroup.Use(MiddlewareServer(servers))
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
		ClientID int         `json:"client_id"`
		BanType  ban.BanType `json:"ban_type"`
		Msg      string      `json:"msg"`
	}

	return func(ctx *gin.Context) {
		var (
			currentUser, _ = session.CurrentUserProfile(ctx)
			req            checkRequest
		)

		if !httphelper.Bind(ctx, &req) {
			slog.Error("Failed to bind check request")

			return
		}

		defaultValue := checkResponse{
			ClientID: req.ClientID,
			BanType:  ban.OK,
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
			_, errPlayer := s.persons.GetOrCreatePersonBySteamID(ctx, nil, steamID)
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
						BanType:  ban.Banned,
						Msg:      "Evasion ban",
					}

					ctx.JSON(http.StatusOK, defaultValue)

					// s.notifications.Enqueue(ctx, domain.NewDiscordNotification(
					// 	domain.ChannelKickLog,
					// 	discord.KickPlayerOnConnectEmbed(steamID, req.Name, player, banState.BanSource)))

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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errImmunities))

			return
		}

		if immunities == nil {
			immunities = []SMGroupImmunity{}
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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errImmunity))

			return
		}

		ctx.JSON(http.StatusOK, immunity)
	}
}

func (s *srcdsHandler) onDeleteGroupImmunity() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupImmunityID, idFound := httphelper.GetIntParam(ctx, "group_immunity_id")
		if !idFound {
			return
		}

		if err := s.srcds.DelGroupImmunity(ctx, groupImmunityID); err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, err))

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
		groupID, idFound := httphelper.GetIntParam(ctx, "group_id")
		if !idFound {
			return
		}

		overrides, errOverrides := s.srcds.GroupOverrides(ctx, groupID)
		if errOverrides != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errOverrides))

			return
		}

		if overrides == nil {
			overrides = []SMGroupOverrides{}
		}

		ctx.JSON(http.StatusOK, overrides)
	}
}

type groupOverrideRequest struct {
	Name   string         `json:"name"`
	Type   OverrideType   `json:"type"`
	Access OverrideAccess `json:"access"`
}

func (s *srcdsHandler) onCreateGroupOverride() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupID, idFound := httphelper.GetIntParam(ctx, "group_id")
		if idFound {
			return
		}

		var req groupOverrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errOverride := s.srcds.AddGroupOverride(ctx, groupID, req.Name, req.Type, req.Access)
		if errOverride != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errOverride))

			return
		}

		ctx.JSON(http.StatusOK, override)
	}
}

func (s *srcdsHandler) onSaveGroupOverride() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupOverrideID, idFound := httphelper.GetIntParam(ctx, "group_override_id")
		if !idFound {
			return
		}

		var req groupOverrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errOverride := s.srcds.GetGroupOverride(ctx, groupOverrideID)
		if errOverride != nil {
			if errors.Is(errOverride, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, database.ErrNoResult))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errOverride))

			return
		}

		override.Type = req.Type
		override.Name = req.Name
		override.Access = req.Access

		edited, errSave := s.srcds.SaveGroupOverride(ctx, override)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errSave))

			return
		}

		ctx.JSON(http.StatusOK, edited)
	}
}

func (s *srcdsHandler) onDeleteGroupOverride() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupOverrideID, idFound := httphelper.GetIntParam(ctx, "group_override_id")
		if !idFound {
			return
		}

		if err := s.srcds.DelGroupOverride(ctx, groupOverrideID); err != nil {
			if errors.Is(err, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, database.ErrNoResult))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, err))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

type overrideRequest struct {
	Name  string       `json:"name"`
	Type  OverrideType `json:"type"`
	Flags string       `json:"flags"`
}

func (s *srcdsHandler) onSaveOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		overrideID, idFound := httphelper.GetIntParam(ctx, "override_id")
		if !idFound {
			return
		}

		var req overrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errOverride := s.srcds.GetOverride(ctx, overrideID)
		if errOverride != nil {
			if errors.Is(errOverride, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, database.ErrNoResult))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errOverride))

			return
		}

		override.Type = req.Type
		override.Name = req.Name
		override.Flags = req.Flags

		edited, errSave := s.srcds.SaveOverride(ctx, override)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errSave))

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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errCreate))

			return
		}

		ctx.JSON(http.StatusOK, override)
	}
}

func (s *srcdsHandler) onDeleteOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		override, idFound := httphelper.GetIntParam(ctx, "override_id")
		if !idFound {
			return
		}

		if errCreate := s.srcds.DelOverride(ctx, override); errCreate != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errCreate))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (s *srcdsHandler) onGetOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		overrides, errOverrides := s.srcds.Overrides(ctx)
		if errOverrides != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errOverrides))

			return
		}

		if overrides == nil {
			overrides = []SMOverrides{}
		}

		ctx.JSON(http.StatusOK, overrides)
	}
}

func (s *srcdsHandler) onAddAdminGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminID, idFound := httphelper.GetIntParam(ctx, "admin_id")
		if !idFound {
			return
		}

		var req groupRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		admin, err := s.srcds.AddAdminGroup(ctx, adminID, req.GroupID)
		if err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, err))

			return
		}

		ctx.JSON(http.StatusOK, admin)
	}
}

func (s *srcdsHandler) onDeleteAdminGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminID, idFound := httphelper.GetIntParam(ctx, "admin_id")
		if !idFound {
			return
		}

		groupID, groupIDFound := httphelper.GetIntParam(ctx, "group_id")
		if !groupIDFound {
			return
		}

		admin, errDel := s.srcds.DelAdminGroup(ctx, adminID, groupID)
		if errDel != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errDel))

			return
		}

		ctx.JSON(http.StatusOK, admin)
	}
}

func (s *srcdsHandler) onSaveSMAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminID, idFound := httphelper.GetIntParam(ctx, "admin_id")
		if !idFound {
			return
		}

		admin, errAdmin := s.srcds.GetAdminByID(ctx, adminID)
		if errAdmin != nil {
			if errors.Is(errAdmin, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, database.ErrNoResult))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errAdmin))

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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errSave))

			return
		}

		ctx.JSON(http.StatusOK, editedGroup)
	}
}

func (s *srcdsHandler) onDeleteSMAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminID, idFound := httphelper.GetIntParam(ctx, "admin_id")
		if !idFound {
			return
		}

		if err := s.srcds.DelAdmin(ctx, adminID); err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, err))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

type smAdminRequest struct {
	AuthType AuthType `json:"auth_type"`
	Identity string   `json:"identity"`
	Password string   `json:"password"`
	Flags    string   `json:"flags"`
	Name     string   `json:"name"`
	Immunity int      `json:"immunity"`
}

func (s *srcdsHandler) onCreateSMAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req smAdminRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		admin, errAdmin := s.srcds.AddAdmin(ctx, req.Name, req.AuthType, req.Identity, req.Flags, req.Immunity, req.Password)
		if errAdmin != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errAdmin))

			return
		}

		ctx.JSON(http.StatusOK, admin)
	}
}

func (s *srcdsHandler) onDeleteSMGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupID, idFound := httphelper.GetIntParam(ctx, "group_id")
		if !idFound {
			return
		}

		if err := s.srcds.DelGroup(ctx, groupID); err != nil {
			if errors.Is(err, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, database.ErrNoResult))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, err))

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
		groupID, idFound := httphelper.GetIntParam(ctx, "group_id")
		if !idFound {
			return
		}

		group, errGroup := s.srcds.GetGroupByID(ctx, groupID)
		if errGroup != nil {
			if errors.Is(errGroup, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, database.ErrNoResult))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errGroup))

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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errSave))

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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errGroup))

			return
		}

		ctx.JSON(http.StatusCreated, group)
	}
}

func (s *srcdsHandler) onAPISMGroups() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groups, errGroups := s.srcds.Groups(ctx)
		if errGroups != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errGroups))

			return
		}

		ctx.JSON(http.StatusOK, groups)
	}
}

func (s *srcdsHandler) onGetSMAdmins() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		admins, errAdmins := s.srcds.Admins(ctx)
		if errAdmins != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errAdmins))

			return
		}

		if admins == nil {
			admins = []SMAdmin{}
		}

		ctx.JSON(http.StatusOK, admins)
	}
}

func (s *srcdsHandler) onAPIPostServerState() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req servers.PartialStateUpdate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		serverID, idFound := httphelper.GetIntParam(ctx, "server_id")
		if !idFound {
			return
		}

		if errUpdate := s.state.Update(serverID, req); errUpdate != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errUpdate))

			return
		}

		ctx.AbortWithStatus(http.StatusNoContent)
	}
}

func (s *srcdsHandler) onAPIPostReportCreate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser, _ := session.CurrentUserProfile(ctx)

		var req ban.RequestReportCreate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		report, errReport := s.srcds.Report(ctx, currentUser, req)
		if errReport != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errReport))

			return
		}

		ctx.JSON(http.StatusCreated, report)
		slog.Info("New report created successfully", slog.Int64("report_id", report.ReportID), slog.String("method", "in-game"))
	}
}

func (s *srcdsHandler) onAPIPostBanSteamCreate() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req domain.RequestBanCreate
		if !httphelper.Bind(ctx, &req) {
			return
		}

		user, _ := session.CurrentUserProfile(ctx)
		ban, errBan := s.bans.Ban(ctx, user, ban.InGame, req)
		if errBan != nil {
			if errors.Is(errBan, database.ErrDuplicate) {
				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusConflict, database.ErrDuplicate,
					"This user is already currently banned"))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errBan))

			return
		}

		ctx.JSON(http.StatusCreated, ban)
		slog.Info("Created new ban successfully", slog.Int64("ban_id", ban.BanID))
	}
}

func (s *srcdsHandler) onAPIGetServerOverrides() gin.HandlerFunc {
	type smOverride struct {
		Type  OverrideType `json:"type"`
		Name  string       `json:"name"`
		Flags string       `json:"flags"`
	}

	return func(ctx *gin.Context) {
		overrides, errOverrides := s.srcds.Overrides(ctx)
		if errOverrides != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errOverrides))

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
		if errGroups != nil && !errors.Is(errGroups, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, httphelper.ErrNotFound))

			return
		}

		immunities, errImmunities := s.srcds.GetGroupImmunities(ctx)
		if errImmunities != nil && !errors.Is(errImmunities, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errImmunities))

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
		ID       int      `json:"id"`
		Authtype AuthType `json:"authtype"`
		Identity string   `json:"identity"`
		Password string   `json:"password"`
		Flags    string   `json:"flags"`
		Name     string   `json:"name"`
		Immunity int      `json:"immunity"`
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
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errUsers))

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

		if len(players) == 0 && conf.General.Mode != config.TestMode {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, httphelper.ErrInternal))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{"client": req.Client, "message": "Moderators have been notified"})

		author, err := s.persons.GetOrCreatePersonBySteamID(ctx, nil, req.SteamID)
		if err != nil {
			slog.Error("Failed to load user", log.ErrAttr(err))

			return
		}

		server, errServer := s.servers.Server(ctx, players[0].ServerID)
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

		// s.notifications.Enqueue(ctx, notification.NewDiscordNotification(
		// 	domain.ChannelMod,
		// 	discord.PingModMessage(author, conf.ExtURL(author), req.Reason, server, conf.Discord.ModPingRoleID, connect)))

		if errSay := s.state.PSay(ctx, author.SteamID, "Moderators have been notified"); errSay != nil {
			slog.Error("Failed to reply to user", log.ErrAttr(errSay))
		}
	}
}
