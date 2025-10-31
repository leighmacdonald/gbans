package sourcemod

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/netip"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type EvadeChecker interface {
	CheckEvadeStatus(ctx context.Context, steamID steamid.SteamID, address netip.Addr) (bool, error)
}

type Handler struct {
	sourcemod *Sourcemod
	persons   person.Provider
	evades    EvadeChecker
}

// MiddlewareServer(servers, sentryDSN).
func NewHandler(engine *gin.Engine, auth httphelper.Authenticator, serverAuth gin.HandlerFunc, sourcemod *Sourcemod) {
	handler := Handler{
		sourcemod: sourcemod,
	}

	adminGroup := engine.Group("/")
	{
		admin := adminGroup.Use(auth.Middleware(permission.Admin))
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
		server := srcdsGroup.Use(serverAuth)
		server.POST("/api/sm/check", handler.onAPICheckPlayer())
		server.GET("/api/sm/overrides", handler.onAPIGetServerOverrides())
		server.GET("/api/sm/users", handler.onAPIGetServerUsers())
		server.GET("/api/sm/groups", handler.onAPIGetServerGroups())

		// Duplicated since we need to authenticate via server middleware
		// server.POST("/api/sm/bans/steam/create", handler.onAPIPostBanSteamCreate())
		// server.POST("/api/sm/report/create", handler.onAPIPostReportCreate())
	}
}

type ServerAuthResp struct {
	Status bool   `json:"status"`
	Token  string `json:"token"`
}

func (s *Handler) onAPICheckPlayer() gin.HandlerFunc {
	type checkRequest struct {
		httphelper.SteamIDField
		ClientID int        `json:"client_id"`
		IP       netip.Addr `json:"ip"`
		Name     string     `json:"name"`
	}

	type checkResponse struct {
		ClientID int      `json:"client_id"`
		BanType  ban.Type `json:"ban_type"`
		Msg      string   `json:"msg"`
	}

	return func(ctx *gin.Context) {
		var req checkRequest

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
			slog.Error("Did not receive valid steamid for check response", log.ErrAttr(steamid.ErrDecodeSID))

			return
		}

		banState, msg, errBS := s.sourcemod.GetBanState(ctx, steamID, req.IP)
		if errBS != nil {
			slog.Error("failed to get ban state", log.ErrAttr(errBS))

			// Fail Open
			ctx.JSON(http.StatusOK, defaultValue)

			return
		}

		if banState.BanID != 0 {
			_, errPlayer := s.persons.GetOrCreatePersonBySteamID(ctx, steamID)
			if errPlayer != nil {
				slog.Error("Failed to load or create player on connect")
				ctx.JSON(http.StatusOK, defaultValue)

				return
			}

			if banState.SteamID != steamID && !banState.EvadeOK {
				evadeBanned, err := s.evades.CheckEvadeStatus(ctx, steamID, req.IP)
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

func (s *Handler) onGetGroupImmunities() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		immunities, errImmunities := s.sourcemod.GroupImmunities(ctx)
		if errImmunities != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errImmunities))

			return
		}

		if immunities == nil {
			immunities = []GroupImmunity{}
		}

		ctx.JSON(http.StatusOK, immunities)
	}
}

type groupImmunityRequest struct {
	GroupID int `json:"group_id"`
	OtherID int `json:"other_id"`
}

func (s *Handler) onCreateGroupImmunity() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req groupImmunityRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		immunity, errImmunity := s.sourcemod.AddGroupImmunity(ctx, req.GroupID, req.OtherID)
		if errImmunity != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errImmunity))

			return
		}

		ctx.JSON(http.StatusOK, immunity)
	}
}

func (s *Handler) onDeleteGroupImmunity() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupImmunityID, idFound := httphelper.GetIntParam(ctx, "group_immunity_id")
		if !idFound {
			return
		}

		if err := s.sourcemod.DelGroupImmunity(ctx, groupImmunityID); err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, err))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

type groupRequest struct {
	GroupID int `json:"group_id"`
}

func (s *Handler) onGroupOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupID, idFound := httphelper.GetIntParam(ctx, "group_id")
		if !idFound {
			return
		}

		overrides, errOverrides := s.sourcemod.GroupOverrides(ctx, groupID)
		if errOverrides != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errOverrides))

			return
		}

		if overrides == nil {
			overrides = []GroupOverrides{}
		}

		ctx.JSON(http.StatusOK, overrides)
	}
}

type groupOverrideRequest struct {
	Name   string         `json:"name"`
	Type   OverrideType   `json:"type"`
	Access OverrideAccess `json:"access"`
}

func (s *Handler) onCreateGroupOverride() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupID, idFound := httphelper.GetIntParam(ctx, "group_id")
		if idFound {
			return
		}

		var req groupOverrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errOverride := s.sourcemod.AddGroupOverride(ctx, groupID, req.Name, req.Type, req.Access)
		if errOverride != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errOverride))

			return
		}

		ctx.JSON(http.StatusOK, override)
	}
}

func (s *Handler) onSaveGroupOverride() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupOverrideID, idFound := httphelper.GetIntParam(ctx, "group_override_id")
		if !idFound {
			return
		}

		var req groupOverrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errOverride := s.sourcemod.GroupOverride(ctx, groupOverrideID)
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

		edited, errSave := s.sourcemod.SaveGroupOverride(ctx, override)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errSave))

			return
		}

		ctx.JSON(http.StatusOK, edited)
	}
}

func (s *Handler) onDeleteGroupOverride() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupOverrideID, idFound := httphelper.GetIntParam(ctx, "group_override_id")
		if !idFound {
			return
		}

		if err := s.sourcemod.DelGroupOverride(ctx, groupOverrideID); err != nil {
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

func (s *Handler) onSaveOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		overrideID, idFound := httphelper.GetIntParam(ctx, "override_id")
		if !idFound {
			return
		}

		var req overrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errOverride := s.sourcemod.Override(ctx, overrideID)
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

		edited, errSave := s.sourcemod.SaveOverride(ctx, override)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errSave))

			return
		}

		ctx.JSON(http.StatusOK, edited)
	}
}

func (s *Handler) onCreateOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req overrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errCreate := s.sourcemod.AddOverride(ctx, req.Name, req.Type, req.Flags)
		if errCreate != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errCreate))

			return
		}

		ctx.JSON(http.StatusOK, override)
	}
}

func (s *Handler) onDeleteOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		override, idFound := httphelper.GetIntParam(ctx, "override_id")
		if !idFound {
			return
		}

		if errCreate := s.sourcemod.DelOverride(ctx, override); errCreate != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errCreate))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (s *Handler) onGetOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		overrides, errOverrides := s.sourcemod.Overrides(ctx)
		if errOverrides != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errOverrides))

			return
		}

		if overrides == nil {
			overrides = []Overrides{}
		}

		ctx.JSON(http.StatusOK, overrides)
	}
}

func (s *Handler) onAddAdminGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminID, idFound := httphelper.GetIntParam(ctx, "admin_id")
		if !idFound {
			return
		}

		var req groupRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		admin, err := s.sourcemod.AddAdminGroup(ctx, adminID, req.GroupID)
		if err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, err))

			return
		}

		ctx.JSON(http.StatusOK, admin)
	}
}

func (s *Handler) onDeleteAdminGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminID, idFound := httphelper.GetIntParam(ctx, "admin_id")
		if !idFound {
			return
		}

		groupID, groupIDFound := httphelper.GetIntParam(ctx, "group_id")
		if !groupIDFound {
			return
		}

		admin, errDel := s.sourcemod.DelAdminGroup(ctx, adminID, groupID)
		if errDel != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errDel))

			return
		}

		ctx.JSON(http.StatusOK, admin)
	}
}

func (s *Handler) onSaveSMAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminID, idFound := httphelper.GetIntParam(ctx, "admin_id")
		if !idFound {
			return
		}

		admin, errAdmin := s.sourcemod.AdminByID(ctx, adminID)
		if errAdmin != nil {
			if errors.Is(errAdmin, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, database.ErrNoResult))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errAdmin))

			return
		}

		var req CreateAdminRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		admin.Name = req.Name
		admin.Flags = req.Flags
		admin.Immunity = req.Immunity
		admin.AuthType = req.AuthType
		admin.Identity = req.Identity
		admin.Password = req.Password

		editedGroup, errSave := s.sourcemod.SaveAdmin(ctx, admin)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errSave))

			return
		}

		ctx.JSON(http.StatusOK, editedGroup)
	}
}

func (s *Handler) onDeleteSMAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		adminID, idFound := httphelper.GetIntParam(ctx, "admin_id")
		if !idFound {
			return
		}

		if err := s.sourcemod.DelAdmin(ctx, adminID); err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, err))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

type CreateAdminRequest struct {
	AuthType AuthType `json:"auth_type"`
	Identity string   `json:"identity"`
	Password string   `json:"password"`
	Flags    string   `json:"flags"`
	Name     string   `json:"name"`
	Immunity int      `json:"immunity"`
}

func (s *Handler) onCreateSMAdmin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req CreateAdminRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		admin, errAdmin := s.sourcemod.AddAdmin(ctx, req.Name, req.AuthType, req.Identity, req.Flags, req.Immunity, req.Password)
		if errAdmin != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errAdmin))

			return
		}

		ctx.JSON(http.StatusOK, admin)
	}
}

func (s *Handler) onDeleteSMGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupID, idFound := httphelper.GetIntParam(ctx, "group_id")
		if !idFound {
			return
		}

		if err := s.sourcemod.DelGroup(ctx, groupID); err != nil {
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

func (s *Handler) onSaveSMGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupID, idFound := httphelper.GetIntParam(ctx, "group_id")
		if !idFound {
			return
		}

		group, errGroup := s.sourcemod.GetGroupByID(ctx, groupID)
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

		editedGroup, errSave := s.sourcemod.SaveGroup(ctx, group)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errSave))

			return
		}

		ctx.JSON(http.StatusOK, editedGroup)
	}
}

func (s *Handler) onCreateSMGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req smGroupRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		group, errGroup := s.sourcemod.AddGroup(ctx, req.Name, req.Flags, req.Immunity)
		if errGroup != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errGroup))

			return
		}

		ctx.JSON(http.StatusCreated, group)
	}
}

func (s *Handler) onAPISMGroups() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groups, errGroups := s.sourcemod.Groups(ctx)
		if errGroups != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errGroups))

			return
		}

		ctx.JSON(http.StatusOK, groups)
	}
}

func (s *Handler) onGetSMAdmins() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		admins, errAdmins := s.sourcemod.Admins(ctx)
		if errAdmins != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errAdmins))

			return
		}

		if admins == nil {
			admins = []Admin{}
		}

		ctx.JSON(http.StatusOK, admins)
	}
}

// func (s *srcdsHandler) onAPIPostReportCreate() gin.HandlerFunc {
// 	return func(ctx *gin.Context) {
// 		currentUser, _ := session.CurrentUserProfile(ctx)

// 		var req ban.RequestReportCreate
// 		if !httphelper.Bind(ctx, &req) {
// 			return
// 		}

// 		report, errReport := s.srcds.Report(ctx, currentUser, req)
// 		if errReport != nil {
// 			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errReport))

// 			return
// 		}

// 		ctx.JSON(http.StatusCreated, report)
// 		slog.Info("New report created successfully", slog.Int64("report_id", report.ReportID), slog.String("method", "in-game"))
// 	}
// }

// TODO move to ban package
// func (s *srcdsHandler) onAPIPostBanSteamCreate() gin.HandlerFunc {
// 	return func(ctx *gin.Context) {
// 		var req domain.RequestBanCreate
// 		if !httphelper.Bind(ctx, &req) {
// 			return
// 		}

// 		user, _ := session.CurrentUserProfile(ctx)
// 		ban, errBan := s.bans.Ban(ctx, user, ban.InGame, req)
// 		if errBan != nil {
// 			if errors.Is(errBan, database.ErrDuplicate) {
// 				httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusConflict, database.ErrDuplicate,
// 					"This user is already currently banned"))

// 				return
// 			}

// 			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errBan))

// 			return
// 		}

// 		ctx.JSON(http.StatusCreated, ban)
// 		slog.Info("Created new ban successfully", slog.Int64("ban_id", ban.BanID))
// 	}
// }

func (s *Handler) onAPIGetServerOverrides() gin.HandlerFunc {
	type smOverride struct {
		Type  OverrideType `json:"type"`
		Name  string       `json:"name"`
		Flags string       `json:"flags"`
	}

	return func(ctx *gin.Context) {
		overrides, errOverrides := s.sourcemod.Overrides(ctx)
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

func (s *Handler) onAPIGetServerGroups() gin.HandlerFunc {
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
		groups, errGroups := s.sourcemod.Groups(ctx)
		if errGroups != nil && !errors.Is(errGroups, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, httphelper.ErrNotFound))

			return
		}

		immunities, errImmunities := s.sourcemod.GroupImmunities(ctx)
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

func (s *Handler) onAPIGetServerUsers() gin.HandlerFunc {
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
		users, errUsers := s.sourcemod.Admins(ctx)
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
