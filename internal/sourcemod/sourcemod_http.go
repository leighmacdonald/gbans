package sourcemod

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/netip"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type EvadeChecker interface {
	CheckEvadeStatus(ctx context.Context, steamID steamid.SteamID, address netip.Addr) (bool, error)
}

type Handler struct {
	Sourcemod

	persons person.Provider
	evades  EvadeChecker
}

// MiddlewareServer(servers, sentryDSN).
func NewHandler(engine *gin.Engine, auth httphelper.Authenticator, serverAuth httphelper.ServerAuthenticator, sourcemod Sourcemod) {
	handler := Handler{Sourcemod: sourcemod}

	adminGroup := engine.Group("/")
	{
		admin := adminGroup.Use(auth.Middleware(permission.Admin))
		// Groups
		admin.GET("/api/smadmin/groups", handler.onAPISMGroups())
		admin.POST("/api/smadmin/groups", handler.onCreateSMGroup())
		admin.PUT("/api/smadmin/groups/:group_id", handler.onSaveSMGroup())
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
		server := srcdsGroup.Use(serverAuth.Middleware)
		server.GET("/api/sm/check", handler.onAPICheckPlayer())
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

type CheckRequest struct {
	SteamID  string `json:"steam_id"`
	ClientID int    `json:"client_id"`
	IP       string `json:"ip"`
	Name     string `json:"name"`
}

type CheckResponse struct {
	ClientID int          `json:"client_id"`
	BanType  bantype.Type `json:"ban_type"`
	Msg      string       `json:"msg"`
}

func (s *Handler) onAPICheckPlayer() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req CheckRequest

		if !httphelper.Bind(ctx, &req) {
			return
		}

		defaultValue := CheckResponse{
			ClientID: req.ClientID,
			BanType:  bantype.OK,
			Msg:      "",
		}
		steamID := steamid.New(req.SteamID)
		// steamID, valid := req.SteamID
		// if !valid {
		// 	ctx.JSON(http.StatusOK, defaultValue)
		// 	slog.Error("Did not receive valid steamid for check response", log.ErrAttr(steamid.ErrDecodeSID))

		// 	return
		// }

		ipAddr, errIP := netip.ParseAddr(req.IP)
		if errIP != nil {
			ctx.JSON(http.StatusOK, defaultValue)
			slog.Error("Failed to parse IP", slog.String("error", errIP.Error()))

			return
		}

		banState, msg, errBS := s.GetBanState(ctx, steamID, ipAddr)
		if errBS != nil {
			slog.Error("failed to get ban state", slog.String("error", errBS.Error()))

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
				evadeBanned, err := s.evades.CheckEvadeStatus(ctx, steamID, ipAddr)
				if err != nil {
					ctx.JSON(http.StatusOK, defaultValue)

					return
				}

				if evadeBanned {
					defaultValue = CheckResponse{
						ClientID: req.ClientID,
						BanType:  bantype.Banned,
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

			ctx.JSON(http.StatusOK, CheckResponse{
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
		immunities, errImmunities := s.GroupImmunities(ctx)
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

type GroupImmunityRequest struct {
	GroupID int `json:"group_id"`
	OtherID int `json:"other_id"`
}

func (s *Handler) onCreateGroupImmunity() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req GroupImmunityRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		immunity, errImmunity := s.AddGroupImmunity(ctx, req.GroupID, req.OtherID)
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

		if err := s.DelGroupImmunity(ctx, groupImmunityID); err != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, err))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (s *Handler) onGroupOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupID, idFound := httphelper.GetIntParam(ctx, "group_id")
		if !idFound {
			return
		}

		overrides, errOverrides := s.GroupOverrides(ctx, groupID)
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

type GroupOverrideRequest struct {
	Name   string         `json:"name"`
	Type   OverrideType   `json:"type"`
	Access OverrideAccess `json:"access"`
}

func (s *Handler) onCreateGroupOverride() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groupID, idFound := httphelper.GetIntParam(ctx, "group_id")
		if !idFound {
			return
		}

		var req GroupOverrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errOverride := s.AddGroupOverride(ctx, groupID, req.Name, req.Type, req.Access)
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

		var req GroupOverrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errOverride := s.GroupOverride(ctx, groupOverrideID)
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

		edited, errSave := s.SaveGroupOverride(ctx, override)
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

		if err := s.DelGroupOverride(ctx, groupOverrideID); err != nil {
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

type OverrideRequest struct {
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

		var req OverrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errOverride := s.Override(ctx, overrideID)
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

		edited, errSave := s.SaveOverride(ctx, override)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errSave))

			return
		}

		ctx.JSON(http.StatusOK, edited)
	}
}

func (s *Handler) onCreateOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req OverrideRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		override, errCreate := s.AddOverride(ctx, req.Name, req.Type, req.Flags)
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

		if errCreate := s.DelOverride(ctx, override); errCreate != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errCreate))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func (s *Handler) onGetOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		overrides, errOverrides := s.Overrides(ctx)
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

type groupRequest struct {
	GroupID int `json:"group_id"`
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

		admin, err := s.AddAdminGroup(ctx, adminID, req.GroupID)
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

		admin, errDel := s.DelAdminGroup(ctx, adminID, groupID)
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

		admin, errAdmin := s.AdminByID(ctx, adminID)
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

		editedGroup, errSave := s.SaveAdmin(ctx, admin)
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

		if err := s.DelAdmin(ctx, adminID); err != nil {
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

		admin, errAdmin := s.AddAdmin(ctx, req.Name, req.AuthType, req.Identity, req.Flags, req.Immunity, req.Password)
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

		if err := s.DelGroup(ctx, groupID); err != nil {
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

type CreateGroupRequest struct {
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

		group, errGroup := s.GetGroupByID(ctx, groupID)
		if errGroup != nil {
			if errors.Is(errGroup, database.ErrNoResult) {
				httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, database.ErrNoResult))

				return
			}

			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errGroup))

			return
		}

		var req CreateGroupRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		group.Name = req.Name
		group.Flags = req.Flags
		group.ImmunityLevel = req.Immunity

		editedGroup, errSave := s.SaveGroup(ctx, group)
		if errSave != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errSave))

			return
		}

		ctx.JSON(http.StatusOK, editedGroup)
	}
}

func (s *Handler) onCreateSMGroup() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req CreateGroupRequest
		if !httphelper.Bind(ctx, &req) {
			return
		}

		group, errGroup := s.AddGroup(ctx, req.Name, req.Flags, req.Immunity)
		if errGroup != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errGroup))

			return
		}

		ctx.JSON(http.StatusCreated, group)
	}
}

func (s *Handler) onAPISMGroups() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groups, errGroups := s.Groups(ctx)
		if errGroups != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errGroups))

			return
		}

		ctx.JSON(http.StatusOK, groups)
	}
}

func (s *Handler) onGetSMAdmins() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		admins, errAdmins := s.Admins(ctx)
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

//			ctx.JSON(http.StatusCreated, ban)
//			slog.Info("Created new ban successfully", slog.Int64("ban_id", ban.BanID))
//		}
//	}

type Override struct {
	Type  OverrideType `json:"type"`
	Name  string       `json:"name"`
	Flags string       `json:"flags"`
}

func (s *Handler) onAPIGetServerOverrides() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		overrides, errOverrides := s.Overrides(ctx)
		if errOverrides != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errOverrides))

			return
		}

		//goland:noinspection ALL
		smOverrides := []Override{}
		for _, group := range overrides {
			smOverrides = append(smOverrides, Override{
				Flags: group.Flags,
				Name:  group.Name,
				Type:  group.Type,
			})
		}

		ctx.JSON(http.StatusOK, smOverrides)
	}
}

type Group struct {
	Flags         string `json:"flags"`
	Name          string `json:"name"`
	ImmunityLevel int    `json:"immunity_level"`
}

type GroupImmunityResp struct {
	GroupName string `json:"group_name"`
	OtherName string `json:"other_name"`
}

type GroupsResp struct {
	Groups     []Group             `json:"groups"`
	Immunities []GroupImmunityResp `json:"immunities"`
}

func (s *Handler) onAPIGetServerGroups() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		groups, errGroups := s.Groups(ctx)
		if errGroups != nil && !errors.Is(errGroups, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusNotFound, httphelper.ErrNotFound))

			return
		}

		immunities, errImmunities := s.GroupImmunities(ctx)
		if errImmunities != nil && !errors.Is(errImmunities, database.ErrNoResult) {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errImmunities))

			return
		}

		resp := GroupsResp{
			// Make sure we return an empty list instead of null
			Groups:     []Group{},
			Immunities: []GroupImmunityResp{},
		}

		//goland:noinspection ALL
		for _, group := range groups {
			resp.Groups = append(resp.Groups, Group{
				Flags:         group.Flags,
				Name:          group.Name,
				ImmunityLevel: group.ImmunityLevel,
			})
		}

		for _, immunity := range immunities {
			resp.Immunities = append(resp.Immunities, GroupImmunityResp{
				GroupName: immunity.Group.Name,
				OtherName: immunity.Other.Name,
			})
		}

		ctx.JSON(http.StatusOK, resp)
	}
}

type User struct {
	ID       int      `json:"id"`
	Authtype AuthType `json:"authtype"`
	Identity string   `json:"identity"`
	Password string   `json:"password"`
	Flags    string   `json:"flags"`
	Name     string   `json:"name"`
	Immunity int      `json:"immunity"`
}

type UserGroup struct {
	AdminID   int    `json:"admin_id"`
	GroupName string `json:"group_name"`
}

type UsersResponse struct {
	Users      []User      `json:"users"`
	UserGroups []UserGroup `json:"user_groups"`
}

func (s *Handler) onAPIGetServerUsers() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		users, errUsers := s.Admins(ctx)
		if errUsers != nil {
			httphelper.SetError(ctx, httphelper.NewAPIError(http.StatusInternalServerError, errUsers))

			return
		}

		smResp := UsersResponse{
			Users:      []User{},
			UserGroups: []UserGroup{},
		}

		for _, user := range users {
			smResp.Users = append(smResp.Users, User{
				ID:       user.AdminID,
				Authtype: user.AuthType,
				Identity: user.Identity,
				Password: user.Password,
				Flags:    user.Flags,
				Name:     user.Name,
				Immunity: user.Immunity,
			})

			for _, ug := range user.Groups {
				smResp.UserGroups = append(smResp.UserGroups, UserGroup{
					AdminID:   user.AdminID,
					GroupName: ug.Name,
				})
			}
		}

		ctx.JSON(http.StatusOK, smResp)
	}
}
