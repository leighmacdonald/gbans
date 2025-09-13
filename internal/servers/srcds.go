package servers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrSMInvalidAuthName   = errors.New("invalid auth name")
	ErrSMImmunity          = errors.New("invalid immunity level, must be between 0-100")
	ErrSMGroupName         = errors.New("group name cannot be empty")
	ErrSMAdminGroupExists  = errors.New("admin group already exists")
	ErrSMAdminExists       = errors.New("admin already exists")
	ErrSMAdminFlagInvalid  = errors.New("invalid admin flag")
	ErrSMRequirePassword   = errors.New("name auth type requires password")
	ErrInsufficientDetails = errors.New("insufficient details")
)

type BanSource string

const (
	BanSourceNone        BanSource = ""
	BanSourceSteam       BanSource = "ban_steam"
	BanSourceSteamFriend BanSource = "ban_steam_friend"
	BanSourceSteamGroup  BanSource = "steam_group"
	BanSourceSteamNet    BanSource = "ban_net"
	BanSourceCIDR        BanSource = "cidr_block"
	BanSourceASN         BanSource = "ban_asn"
)

type PlayerBanState struct {
	SteamID    steamid.SteamID `json:"steam_id"`
	BanSource  BanSource       `json:"ban_source"`
	BanID      int             `json:"ban_id"`
	BanType    ban.BanType     `json:"ban_type"`
	Reason     ban.Reason      `json:"reason"`
	EvadeOK    bool            `json:"evade_ok"`
	ValidUntil time.Time       `json:"valid_until"`
}

type AuthType string

const (
	AuthTypeSteam AuthType = "steam"
	AuthTypeName  AuthType = "name"
	AuthTypeIP    AuthType = "ip"
)

type OverrideType string

const (
	OverrideTypeCommand OverrideType = "command"
	OverrideTypeGroup   OverrideType = "group"
)

type OverrideAccess string

const (
	OverrideAccessAllow OverrideAccess = "allow"
	OverrideAccessDeny  OverrideAccess = "deny"
)

type SMAdmin struct {
	AdminID   int             `json:"admin_id"`
	SteamID   steamid.SteamID `json:"steam_id"`
	AuthType  AuthType        `json:"auth_type"` // steam | name |ip
	Identity  string          `json:"identity"`
	Password  string          `json:"password"`
	Flags     string          `json:"flags"`
	Name      string          `json:"name"`
	Immunity  int             `json:"immunity"`
	Groups    []SMGroups      `json:"groups"`
	CreatedOn time.Time       `json:"created_on"`
	UpdatedOn time.Time       `json:"updated_on"`
}

type SMGroups struct {
	GroupID       int       `json:"group_id"`
	Flags         string    `json:"flags"`
	Name          string    `json:"name"`
	ImmunityLevel int       `json:"immunity_level"`
	CreatedOn     time.Time `json:"created_on"`
	UpdatedOn     time.Time `json:"updated_on"`
}

type SMGroupImmunity struct {
	GroupImmunityID int       `json:"group_immunity_id"`
	Group           SMGroups  `json:"group"`
	Other           SMGroups  `json:"other"`
	CreatedOn       time.Time `json:"created_on"`
}

type SMGroupOverrides struct {
	GroupOverrideID int            `json:"group_override_id"`
	GroupID         int            `json:"group_id"`
	Type            OverrideType   `json:"type"` // command | group
	Name            string         `json:"name"`
	Access          OverrideAccess `json:"access"` // allow | deny
	CreatedOn       time.Time      `json:"created_on"`
	UpdatedOn       time.Time      `json:"updated_on"`
}

type SMOverrides struct {
	OverrideID int          `json:"override_id"`
	Type       OverrideType `json:"type"` // command | group
	Name       string       `json:"name"`
	Flags      string       `json:"flags"`
	CreatedOn  time.Time    `json:"created_on"`
	UpdatedOn  time.Time    `json:"updated_on"`
}

type SMAdminGroups struct {
	AdminID      int       `json:"admin_id"`
	GroupID      int       `json:"group_id"`
	InheritOrder int       `json:"inherit_order"`
	CreatedOn    time.Time `json:"created_on"`
	UpdatedOn    time.Time `json:"updated_on"`
}

type SMConfig struct {
	CfgKey   string `json:"cfg_key"`
	CfgValue string `json:"cfg_value"`
}

type ServerAuthReq struct {
	Key string `json:"key"`
}

func MiddlewareServer(serversUC ServersUsecase) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reqAuthHeader := ctx.GetHeader("Authorization")
		if reqAuthHeader == "" {
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		if strings.HasPrefix(reqAuthHeader, "Bearer ") {
			parts := strings.Split(reqAuthHeader, " ")
			if len(parts) != 2 {
				ctx.AbortWithStatus(http.StatusUnauthorized)

				return
			}

			reqAuthHeader = parts[1]
		}

		var server Server
		if errServer := serversUC.GetByPassword(ctx, reqAuthHeader, &server, false, false); errServer != nil {
			slog.Error("Failed to load server during auth", log.ErrAttr(errServer), slog.String("token", reqAuthHeader), slog.String("IP", ctx.ClientIP()))
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		ctx.Set("server_id", server.ServerID)

		if app.SentryDSN != "" {
			if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetUser(sentry.User{
						ID:        strconv.Itoa(server.ServerID),
						IPAddress: server.Addr(),
						Name:      server.ShortName,
					})
				})
			}
		}

		ctx.Next()
	}
}
