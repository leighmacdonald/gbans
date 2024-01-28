package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/http_helper"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/demoparser"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type SRCDSHandler struct {
	ServerUsecase      domain.ServersUsecase
	PersonUsecase      domain.PersonUsecase
	StateUsecase       domain.StateUsecase
	DiscordUsecase     domain.DiscordUsecase
	ConfigUsecase      domain.ConfigUsecase
	S3Usecase          domain.AssetUsecase
	reportUsecase      domain.ReportUsecase
	assetUsecase       domain.AssetUsecase
	banUsecase         domain.BanUsecase
	netUsecase         domain.NetworkUsecase
	friendBlockUsecase domain.FriendBlockerUsecase
	netBlockerUsecase  domain.NetBlockerUsecase
	du                 domain.DemoUsecase
	groupBlocker       domain.GroupBlockerUsecase
	log                *zap.Logger
}

const authTokenDuration = time.Minute * 15

func NewSRCDSHandler(engine *gin.Engine, log *zap.Logger, su domain.ServersUsecase, pu domain.PersonUsecase,
	s3usecase domain.AssetUsecase, ru domain.ReportUsecase, au domain.AssetUsecase, bu domain.BanUsecase, nu domain.NetworkUsecase,
	fbu domain.FriendBlockerUsecase, nbu domain.NetBlockerUsecase, gbu domain.GroupBlockerUsecase, demoUsecase domain.DemoUsecase,
) {
	handler := SRCDSHandler{
		ServerUsecase:      su,
		PersonUsecase:      pu,
		reportUsecase:      ru,
		banUsecase:         bu,
		assetUsecase:       au,
		netUsecase:         nu,
		friendBlockUsecase: fbu,
		netBlockerUsecase:  nbu,
		groupBlocker:       gbu,
		du:                 demoUsecase,
		S3Usecase:          s3usecase,
		log:                log,
	}

	// unauthed
	engine.POST("/api/server/auth", handler.onSAPIPostServerAuth())

	// serverAuth := srvGrp.Use(authServerMiddleWare(env))
	engine.GET("/api/server/admins", handler.onAPIGetServerAdmins())
	engine.POST("/api/ping_mod", handler.onAPIPostPingMod())
	engine.POST("/api/check", handler.onAPIPostServerCheck())
	engine.POST("/api/demo", handler.onAPIPostDemo())

	// Duplicated since we need to authenticate via server middleware
	engine.POST("/api/sm/bans/steam/create", handler.onAPIPostBanSteamCreate())
	engine.POST("/api/sm/report/create", handler.onAPIPostReportCreate())
	engine.POST("/api/state_update", handler.onAPIPostServerState())
}

type ServerAuthReq struct {
	Key string `json:"key"`
}

type ServerAuthResp struct {
	Status bool   `json:"status"`
	Token  string `json:"token"`
}

func newServerToken(serverID int, cookieKey string) (string, error) {
	curTime := time.Now()

	claims := &domain.ServerAuthClaims{
		ServerID: serverID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(curTime.Add(authTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(curTime),
			NotBefore: jwt.NewNumericDate(curTime),
		},
	}

	tokenWithClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, errSigned := tokenWithClaims.SignedString([]byte(cookieKey))
	if errSigned != nil {
		return "", errors.Join(errSigned, domain.ErrSignJWT)
	}

	return signedToken, nil
}

func (s *SRCDSHandler) onSAPIPostServerAuth() gin.HandlerFunc {
	log := s.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req ServerAuthReq
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		var server domain.Server

		errGetServer := s.ServerUsecase.GetServerByPassword(ctx, req.Key, &server, true, false)
		if errGetServer != nil {
			http_helper.ResponseErr(ctx, http.StatusUnauthorized, domain.ErrPermissionDenied)
			log.Warn("Failed to find server auth by password", zap.Error(errGetServer))

			return
		}

		if server.Password != req.Key {
			http_helper.ResponseErr(ctx, http.StatusUnauthorized, domain.ErrPermissionDenied)
			log.Error("Invalid server key used")

			return
		}

		accessToken, errToken := newServerToken(server.ServerID, s.ConfigUsecase.Config().HTTP.CookieKey)
		if errToken != nil {
			http_helper.ResponseErr(ctx, http.StatusUnauthorized, domain.ErrPermissionDenied)
			log.Error("Failed to create new server access token", zap.Error(errToken))

			return
		}

		server.TokenCreatedOn = time.Now()
		if errSaveServer := s.ServerUsecase.SaveServer(ctx, &server); errSaveServer != nil {
			http_helper.ResponseErr(ctx, http.StatusUnauthorized, domain.ErrPermissionDenied)
			log.Error("Failed to updated server token", zap.Error(errSaveServer))

			return
		}

		ctx.JSON(http.StatusOK, ServerAuthResp{Status: true, Token: accessToken})
		log.Info("ServerStore authenticated successfully", zap.String("server", server.ShortName))
	}
}

func (s *SRCDSHandler) onAPIPostServerState() gin.HandlerFunc {
	log := s.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.PartialStateUpdate
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		serverID := http_helper.ServerFromCtx(ctx) // TODO use generic func for int
		if serverID == 0 {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrParamInvalid)

			return
		}

		if errUpdate := s.StateUsecase.Update(serverID, req); errUpdate != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.AbortWithStatus(http.StatusNoContent)
	}
}

type apiCreateReportReq struct {
	SourceID        domain.StringSID `json:"source_id"`
	TargetID        domain.StringSID `json:"target_id"`
	Description     string           `json:"description"`
	Reason          domain.Reason    `json:"reason"`
	ReasonText      string           `json:"reason_text"`
	DemoName        string           `json:"demo_name"`
	DemoTick        int              `json:"demo_tick"`
	PersonMessageID int64            `json:"person_message_id"`
}

func (s *SRCDSHandler) onAPIPostReportCreate() gin.HandlerFunc {
	log := s.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		currentUser := http_helper.CurrentUserProfile(ctx)

		var req apiCreateReportReq
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		if req.Description == "" || len(req.Description) < 10 {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, fmt.Errorf("%w: description", domain.ErrParamInvalid))

			return
		}

		// ServerStore initiated requests will have a sourceID set by the server
		// Web based reports the source should not be set, the reporter will be taken from the
		// current session information instead
		if req.SourceID == "" {
			req.SourceID = domain.StringSID(currentUser.SteamID.String())
		}

		sourceID, errSourceID := req.SourceID.SID64(ctx)
		if errSourceID != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrSourceID)
			log.Error("Invalid steam_id", zap.Error(errSourceID))

			return
		}

		targetID, errTargetID := req.TargetID.SID64(ctx)
		if errTargetID != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrTargetID)
			log.Error("Invalid target_id", zap.Error(errTargetID))

			return
		}

		if sourceID == targetID {
			http_helper.ResponseErr(ctx, http.StatusConflict, domain.ErrSelfReport)

			return
		}

		var personSource domain.Person
		if errCreatePerson := s.PersonUsecase.GetPersonBySteamID(ctx, sourceID, &personSource); errCreatePerson != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Could not load player profile", zap.Error(errCreatePerson))

			return
		}

		var personTarget domain.Person
		if errCreatePerson := s.PersonUsecase.GetOrCreatePersonBySteamID(ctx, targetID, &personTarget); errCreatePerson != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Could not load player profile", zap.Error(errCreatePerson))

			return
		}

		if personTarget.Expired() {
			if err := thirdparty.UpdatePlayerSummary(ctx, &personTarget); err != nil {
				log.Error("Failed to update target player", zap.Error(err))
			} else {
				if errSave := s.PersonUsecase.SavePerson(ctx, &personTarget); errSave != nil {
					log.Error("Failed to save target player update", zap.Error(err))
				}
			}
		}

		// Ensure the user doesn't already have an open report against the user
		var existing domain.Report
		if errReports := s.reportUsecase.GetReportBySteamID(ctx, personSource.SteamID, targetID, &existing); errReports != nil {
			if !errors.Is(errReports, domain.ErrNoResult) {
				log.Error("Failed to query reports by steam id", zap.Error(errReports))
				http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

				return
			}
		}

		if existing.ReportID > 0 {
			http_helper.ResponseErr(ctx, http.StatusConflict, domain.ErrReportExists)

			return
		}

		// TODO encapsulate all operations in single tx
		report := domain.NewReport()
		report.SourceID = sourceID
		report.ReportStatus = domain.Opened
		report.Description = req.Description
		report.TargetID = targetID
		report.Reason = req.Reason
		report.ReasonText = req.ReasonText
		parts := strings.Split(req.DemoName, "/")
		report.DemoName = parts[len(parts)-1]
		report.DemoTick = req.DemoTick
		report.PersonMessageID = req.PersonMessageID

		if errReportSave := s.reportUsecase.SaveReport(ctx, &report); errReportSave != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
			log.Error("Failed to save report", zap.Error(errReportSave))

			return
		}

		ctx.JSON(http.StatusCreated, report)

		log.Info("New report created successfully", zap.Int64("report_id", report.ReportID))

		conf := s.ConfigUsecase.Config()

		if !conf.Discord.Enabled {
			return
		}

		demoURL := ""

		if report.DemoName != "" {
			demoURL = conf.ExtURLRaw("/demos/name/%s", report.DemoName)
		}

		msg := discord.NewInGameReportResponse(report, conf.ExtURL(report), currentUser, conf.ExtURL(currentUser), demoURL)

		s.DiscordUsecase.SendPayload(domain.ChannelModLog, msg)
	}
}

func (s *SRCDSHandler) onAPIPostDemo() gin.HandlerFunc {
	log := s.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		serverID := http_helper.ServerFromCtx(ctx)
		if serverID <= 0 {
			http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrBadRequest)

			return
		}

		var server domain.Server
		if errGetServer := s.ServerUsecase.GetServer(ctx, serverID, &server); errGetServer != nil {
			log.Error("ServerStore not found", zap.Int("server_id", serverID))
			http_helper.ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)

			return
		}

		demoFormFile, errDemoFile := ctx.FormFile("demo")
		if errDemoFile != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		demoHandle, errDemoHandle := demoFormFile.Open()
		if errDemoHandle != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		demoContent, errRead := io.ReadAll(demoHandle)
		if errRead != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		dir, errDir := os.MkdirTemp("", "gbans-demo")
		if errDir != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		defer func() {
			if err := os.RemoveAll(dir); err != nil {
				log.Error("Failed to cleanup temp demo path", zap.Error(err))
			}
		}()

		namePartsAll := strings.Split(demoFormFile.Filename, "-")

		var mapName string

		if strings.Contains(demoFormFile.Filename, "workshop-") {
			// 20231221-042605-workshop-cp_overgrown_rc8-ugc503939302.dem
			mapName = namePartsAll[3]
		} else {
			// 20231112-063943-koth_harvest_final.dem
			nameParts := strings.Split(namePartsAll[2], ".")
			mapName = nameParts[0]
		}

		tempPath := filepath.Join(dir, demoFormFile.Filename)

		localFile, errLocalFile := os.Create(tempPath)
		if errLocalFile != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		if _, err := localFile.Write(demoContent); err != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		_ = localFile.Close()

		var demoInfo demoparser.DemoInfo
		if errParse := demoparser.Parse(ctx, tempPath, &demoInfo); errParse != nil {
			http_helper.ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)

			return
		}

		intStats := map[steamid.SID64]gin.H{}

		for _, steamID := range demoInfo.SteamIDs() {
			intStats[steamID] = gin.H{}
		}

		newDemo, errCreateDemo := s.du.Create(ctx, demoFormFile.Filename, demoContent, mapName, intStats, serverID)
		if errCreateDemo != nil {
			http_helper.HandleErrInternal(ctx)

			return
		}

		ctx.JSON(http.StatusCreated, gin.H{"demo_id": newDemo.DemoID})
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

func (s *SRCDSHandler) onAPIPostBanSteamCreate() gin.HandlerFunc {
	log := s.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

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

		if errBan := s.banUsecase.BanSteam(ctx, &banSteam); errBan != nil {
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

func (s *SRCDSHandler) onAPIGetServerAdmins() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		perms, err := s.ServerUsecase.GetServerPermissions(ctx)
		if err != nil {
			http_helper.ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, perms)
	}
}

type pingReq struct {
	ServerName string        `json:"server_name"`
	Name       string        `json:"name"`
	SteamID    steamid.SID64 `json:"steam_id"`
	Reason     string        `json:"reason"`
	Client     int           `json:"client"`
}

func (s *SRCDSHandler) onAPIPostPingMod() gin.HandlerFunc {
	log := s.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req pingReq
		if !http_helper.Bind(ctx, log, &req) {
			return
		}

		conf := s.ConfigUsecase.Config()
		players := s.StateUsecase.Find("", req.SteamID, nil, nil)

		if len(players) == 0 && conf.General.Mode != domain.TestMode {
			log.Error("Failed to find player on /mod call")
			http_helper.ResponseErr(ctx, http.StatusFailedDependency, domain.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, gin.H{"client": req.Client, "message": "Moderators have been notified"})

		if !conf.Discord.Enabled {
			return
		}

		var author domain.Person
		if err := s.PersonUsecase.GetOrCreatePersonBySteamID(ctx, req.SteamID, &author); err != nil {
			log.Error("Failed to load user", zap.Error(err))

			return
		}

		s.DiscordUsecase.SendPayload(domain.ChannelMod,
			discord.PingModMessage(author, conf.ExtURL(author), req.Reason, req.ServerName, conf.Discord.ModPingRoleID))
	}
}

type CheckRequest struct {
	ClientID int         `json:"client_id"`
	SteamID  steamid.SID `json:"steam_id"`
	IP       net.IP      `json:"ip"`
	Name     string      `json:"name,omitempty"`
}

type CheckResponse struct {
	ClientID        int              `json:"client_id"`
	SteamID         steamid.SID      `json:"steam_id"`
	BanType         domain.BanType   `json:"ban_type"`
	PermissionLevel domain.Privilege `json:"permission_level"`
	Msg             string           `json:"msg"`
}

// onAPIPostServerCheck takes care of checking if the player connecting to the server is
// allowed to connect, or otherwise has restrictions such as being mutes. It performs
// the following actions/checks in order:
//
// - Add ip to connection history
// - Check if is a part of a steam group ban
// - Check if ip belongs to banned 3rd party CIDR block, like VPNs.
// - Check if ip belongs to one or more local CIDR bans
// - Check if ip belongs to a banned AS Number range
// - Check if steam_id is part of a local steam ban
// - Check if player is connecting from an IP that belongs to a banned player
//
// Returns an ok/muted/banned status for the player.
func (s *SRCDSHandler) onAPIPostServerCheck() gin.HandlerFunc {
	log := s.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var request CheckRequest
		if errBind := ctx.BindJSON(&request); errBind != nil { // we don't currently use bind() for server api
			ctx.JSON(http.StatusInternalServerError, CheckResponse{
				BanType: domain.Unknown,
				Msg:     "Error determining state",
			})

			return
		}

		log.Debug("Player connecting",
			zap.String("ip", request.IP.String()),
			zap.Int64("sid64", steamid.SIDToSID64(request.SteamID).Int64()),
			zap.String("name", request.Name))

		resp := CheckResponse{ClientID: request.ClientID, SteamID: request.SteamID, BanType: domain.Unknown, Msg: ""}

		responseCtx, cancelResponse := context.WithTimeout(ctx, time.Second*15)
		defer cancelResponse()

		steamID := steamid.SIDToSID64(request.SteamID)
		if !steamID.Valid() {
			resp.Msg = "Invalid steam id"
			ctx.JSON(http.StatusBadRequest, resp)

			return
		}

		var person domain.Person

		if errPerson := s.PersonUsecase.GetOrCreatePersonBySteamID(responseCtx, steamID, &person); errPerson != nil {
			log.Error("Failed to create connecting player", zap.Error(errPerson))
		} else if person.Expired() {
			if err := thirdparty.UpdatePlayerSummary(ctx, &person); err != nil {
				log.Error("Failed to update connecting player", zap.Error(err))
			} else {
				if errSave := s.PersonUsecase.SavePerson(ctx, &person); errSave != nil {
					log.Error("Failed to save connecting player summary", zap.Error(err))
				}
			}
		}

		if errAddHist := s.PersonUsecase.AddConnectionHistory(ctx, &domain.PersonConnection{
			IPAddr:      request.IP,
			SteamID:     steamID,
			PersonaName: request.Name,
			CreatedOn:   time.Now(),
			ServerID:    ctx.GetInt("server_id"),
		}); errAddHist != nil {
			log.Error("Failed to add conn history", zap.Error(errAddHist))
		}

		resp.PermissionLevel = person.PermissionLevel

		if s.checkGroupBan(ctx, log, steamID, &resp) || s.checkFriendBan(ctx, log, steamID, &resp) {
			return
		}

		if s.checkNetBlockBan(ctx, log, steamID, request.IP, &resp) {
			return
		}

		if s.checkIPBan(ctx, log, steamID, request.IP, responseCtx, &resp) {
			return
		}

		if s.checkASN(ctx, log, steamID, request.IP, responseCtx, &resp) {
			return
		}

		bannedPerson := domain.NewBannedPerson()
		if errGetBan := s.banUsecase.GetBanBySteamID(responseCtx, steamID, &bannedPerson, false); errGetBan != nil {
			if errors.Is(errGetBan, domain.ErrNoResult) {
				isShared, errShared := s.banUsecase.IsOnIPWithBan(ctx, steamid.SIDToSID64(request.SteamID), request.IP)
				if errShared != nil {
					log.Error("Failed to check shared ip state", zap.Error(errShared))

					ctx.JSON(http.StatusOK, resp)

					return
				}
				if isShared {
					log.Info("Player connected from IP of a banned player",
						zap.String("steam_id", steamid.SIDToSID64(request.SteamID).String()),
						zap.String("ip", request.IP.String()))

					resp.BanType = domain.Banned
					resp.Msg = "Ban evasion. Previous ban updated to permanent if not already permanent"

					ctx.JSON(http.StatusOK, resp)

					return
				}

				// No ban, exit early
				resp.BanType = domain.OK
				ctx.JSON(http.StatusOK, resp)

				return
			}

			resp.Msg = "Error determining state"

			ctx.JSON(http.StatusInternalServerError, resp)

			return
		}

		resp.BanType = bannedPerson.BanType

		var reason string

		switch {
		case bannedPerson.Reason == domain.Custom && bannedPerson.ReasonText != "":
			reason = bannedPerson.ReasonText
		case bannedPerson.Reason == domain.Custom && bannedPerson.ReasonText == "":
			reason = "Banned"
		default:
			reason = bannedPerson.Reason.String()
		}

		conf := s.ConfigUsecase.Config()

		resp.Msg = fmt.Sprintf("Banned\nReason: %s\nAppeal: %s\nRemaining: %s", reason, conf.ExtURL(bannedPerson.BanSteam),
			time.Until(bannedPerson.ValidUntil).Round(time.Minute).String())

		ctx.JSON(http.StatusOK, resp)

		//goland:noinspection GoSwitchMissingCasesForIotaConsts
		switch resp.BanType {
		case domain.NoComm:
			log.Info("Player muted", zap.Int64("sid64", steamID.Int64()))
		case domain.Banned:
			log.Info("Player dropped", zap.String("drop_type", "steam"),
				zap.Int64("sid64", steamID.Int64()))
		}
	}
}

func (s *SRCDSHandler) checkASN(ctx *gin.Context, log *zap.Logger, steamID steamid.SID64, addr net.IP, responseCtx context.Context, resp *CheckResponse) bool {
	var asnRecord ip2location.ASNRecord

	errASN := s.netUsecase.GetASNRecordByIP(responseCtx, addr, &asnRecord)
	if errASN == nil {
		var asnBan domain.BanASN
		if errASNBan := s.banUsecase.GetBanASN(responseCtx, int64(asnRecord.ASNum), &asnBan); errASNBan != nil {
			if !errors.Is(errASNBan, domain.ErrNoResult) {
				log.Error("Failed to fetch asn bannedPerson", zap.Error(errASNBan))
			}
		} else {
			resp.BanType = domain.Banned
			resp.Msg = asnBan.Reason.String()
			ctx.JSON(http.StatusOK, resp)
			log.Info("Player dropped", zap.String("drop_type", "asn"),
				zap.Int64("sid64", steamID.Int64()))

			return true
		}
	}

	return false
}

func (s *SRCDSHandler) checkIPBan(ctx *gin.Context, log *zap.Logger, steamID steamid.SID64, addr net.IP, responseCtx context.Context, resp *CheckResponse) bool {
	// Check IP first
	banNet, errGetBanNet := s.banUsecase.GetBanNetByAddress(responseCtx, addr)
	if errGetBanNet != nil {
		ctx.JSON(http.StatusInternalServerError, CheckResponse{
			BanType: domain.Unknown,
			Msg:     "Error determining state",
		})
		log.Error("Could not get bannedPerson net results", zap.Error(errGetBanNet))

		return true
	}

	if len(banNet) > 0 {
		resp.BanType = domain.Banned
		resp.Msg = "Banned"

		ctx.JSON(http.StatusOK, resp)

		log.Info("Player dropped", zap.String("drop_type", "cidr"),
			zap.Int64("sid64", steamID.Int64()))

		return true
	}

	return false
}

func (s *SRCDSHandler) checkNetBlockBan(ctx *gin.Context, log *zap.Logger, steamID steamid.SID64, addr net.IP, resp *CheckResponse) bool {
	if source, cidrBanned := s.netBlockerUsecase.IsMatch(addr); cidrBanned {
		resp.BanType = domain.Network
		resp.Msg = "Network Range Banned.\nIf you using a VPN try disabling it"

		ctx.JSON(http.StatusOK, resp)
		log.Info("Player network blocked", zap.Int64("sid64", steamID.Int64()),
			zap.String("source", source), zap.String("ip", addr.String()))

		return true
	}

	return false
}

func (s *SRCDSHandler) checkGroupBan(ctx *gin.Context, log *zap.Logger, steamID steamid.SID64, resp *CheckResponse) bool {
	if groupID, banned := s.groupBlocker.IsMember(steamID); banned {
		resp.BanType = domain.Banned
		resp.Msg = fmt.Sprintf("Group Banned (gid: %d)", groupID.Int64())

		ctx.JSON(http.StatusOK, resp)
		log.Info("Player dropped", zap.String("drop_type", "group"),
			zap.Int64("sid64", steamID.Int64()))

		return true
	}

	return false
}

func (s *SRCDSHandler) checkFriendBan(ctx *gin.Context, log *zap.Logger, steamID steamid.SID64, resp *CheckResponse) bool {
	if parentFriendID, banned := s.groupBlocker.IsMember(steamID); banned {
		resp.BanType = domain.Banned

		resp.Msg = fmt.Sprintf("Banned (sid: %d)", parentFriendID.Int64())

		ctx.JSON(http.StatusOK, resp)
		log.Info("Player dropped", zap.String("drop_type", "friend"),
			zap.Int64("sid64", steamID.Int64()))

		return true
	}

	return false
}
