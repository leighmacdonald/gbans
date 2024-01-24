package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
)

var (
	errAssetCreateFailed  = errors.New("failed to create asset")
	errAssetPut           = errors.New("unable to create asset on remote store")
	errAssetSave          = errors.New("failed to save asset metadata")
	errInvalidFormat      = errors.New("invalid format")
	errDuplicateMediaName = errors.New("duplicate media name")
	errSaveMedia          = errors.New("could not save media")
	errFetchMedia         = errors.New("failed to fetch media asset")
	errEmptyToken         = errors.New("invalid access token decoded")
	errContestLoadEntries = errors.New("failed to load existing contest entries")
	errContestMaxEntries  = errors.New("entries count exceed max_submission limits")
	errEntryCreate        = errors.New("failed to create new contest entry")
	errEntrySave          = errors.New("failed to save contest entry")
	errThreadLocked       = errors.New("thread is locked")
)

func makeGetTokenKey(cookieKey string) func(_ *jwt.Token) (any, error) {
	return func(_ *jwt.Token) (any, error) {
		return []byte(cookieKey), nil
	}
}

const fingerprintCookieName = "fingerprint"

// onTokenRefresh handles generating new token pairs to access the api
// NOTE: All error code paths must return 401 (Unauthorized).
func onTokenRefresh(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		fingerprint, errCookie := ctx.Cookie(fingerprintCookieName)
		if errCookie != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			log.Warn("Failed to get fingerprint cookie", zap.Error(errCookie))

			return
		}

		refreshTokenString, errToken := tokenFromHeader(ctx, false)
		if errToken != nil {
			ctx.AbortWithStatus(http.StatusForbidden)

			return
		}

		userClaims := userAuthClaims{}

		refreshToken, errParseClaims := jwt.ParseWithClaims(refreshTokenString, &userClaims, makeGetTokenKey(env.Config().HTTP.CookieKey))
		if errParseClaims != nil {
			if errors.Is(errParseClaims, jwt.ErrSignatureInvalid) {
				log.Error("jwt signature invalid!", zap.Error(errParseClaims))
				ctx.AbortWithStatus(http.StatusUnauthorized)

				return
			}

			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		claims, ok := refreshToken.Claims.(*userAuthClaims)
		if !ok || !refreshToken.Valid {
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		hash := fingerprintHash(fingerprint)
		if claims.Fingerprint != hash {
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		var auth domain.PersonAuth
		if authError := env.Store().GetPersonAuthByRefreshToken(ctx, fingerprint, &auth); authError != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		tokens, errMakeToken := makeTokens(ctx, env, env.Config().HTTP.CookieKey, auth.SteamID, false)
		if errMakeToken != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			log.Error("Failed to create access token pair", zap.Error(errMakeToken))

			return
		}

		ctx.JSON(http.StatusOK, userToken{
			AccessToken: tokens.access,
		})
	}
}

func onOAuthDiscordCallback(env Env) gin.HandlerFunc {
	type accessTokenResp struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
		TokenType    string `json:"token_type"`
	}

	type discordUserDetail struct {
		ID               string      `json:"id"`
		Username         string      `json:"username"`
		Avatar           string      `json:"avatar"`
		AvatarDecoration interface{} `json:"avatar_decoration"`
		Discriminator    string      `json:"discriminator"`
		PublicFlags      int         `json:"public_flags"`
		Flags            int         `json:"flags"`
		Banner           interface{} `json:"banner"`
		BannerColor      interface{} `json:"banner_color"`
		AccentColor      interface{} `json:"accent_color"`
		Locale           string      `json:"locale"`
		MfaEnabled       bool        `json:"mfa_enabled"`
		PremiumType      int         `json:"premium_type"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())
	client := util.NewHTTPClient()

	fetchDiscordID := func(ctx context.Context, accessToken string) (string, error) {
		req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, "https://discord.com/api/users/@me", nil)
		if errReq != nil {
			return "", errors.Join(errReq, errs.ErrCreateRequest)
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		resp, errResp := client.Do(req)

		if errResp != nil {
			return "", errors.Join(errResp, errs.ErrRequestPerform)
		}

		defer func() {
			_ = resp.Body.Close()
		}()

		var details discordUserDetail
		if errJSON := json.NewDecoder(resp.Body).Decode(&details); errJSON != nil {
			return "", errors.Join(errJSON, errs.ErrRequestDecode)
		}

		return details.ID, nil
	}

	fetchToken := func(ctx context.Context, code string) (string, error) {
		// v, _ := go_oauth_pkce_code_verifier.CreateCodeVerifierFromBytes([]byte(code))
		conf := env.Config()
		form := url.Values{}
		form.Set("client_id", conf.Discord.AppID)
		form.Set("client_secret", conf.Discord.AppSecret)
		form.Set("redirect_uri", conf.ExtURLRaw("/login/discord"))
		form.Set("code", code)
		form.Set("grant_type", "authorization_code")
		// form.Set("state", state.String())
		form.Set("scope", "identify")
		req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, "https://discord.com/api/oauth2/token", strings.NewReader(form.Encode()))

		if errReq != nil {
			return "", errors.Join(errReq, errs.ErrCreateRequest)
		}

		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		resp, errResp := client.Do(req)
		if errResp != nil {
			return "", errors.Join(errResp, errs.ErrRequestPerform)
		}

		defer func() {
			_ = resp.Body.Close()
		}()

		var atr accessTokenResp
		if errJSON := json.NewDecoder(resp.Body).Decode(&atr); errJSON != nil {
			return "", errors.Join(errJSON, errs.ErrRequestDecode)
		}

		if atr.AccessToken == "" {
			return "", errEmptyToken
		}

		return atr.AccessToken, nil
	}

	return func(ctx *gin.Context) {
		code := ctx.Query("code")
		if code == "" {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to get code from query")

			return
		}

		token, errToken := fetchToken(ctx, code)
		if errToken != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to fetch token", zap.Error(errToken))

			return
		}

		discordID, errID := fetchDiscordID(ctx, token)
		if errID != nil {
			responseErr(ctx, http.StatusBadRequest, nil)
			log.Error("Failed to fetch discord ID", zap.Error(errID))

			return
		}

		if discordID == "" {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Empty discord id received")

			return
		}

		var discordPerson domain.Person
		if errDp := env.Store().GetPersonByDiscordID(ctx, discordID, &discordPerson); errDp != nil {
			if !errors.Is(errDp, errs.ErrNoResult) {
				responseErr(ctx, http.StatusInternalServerError, nil)

				return
			}
		}

		if discordPerson.DiscordID != "" {
			responseErr(ctx, http.StatusConflict, nil)
			log.Error("Failed to update persons discord id")

			return
		}

		sid := currentUserProfile(ctx).SteamID

		var person domain.Person
		if errPerson := env.Store().GetPersonBySteamID(ctx, sid, &person); errPerson != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		if person.Expired() {
			if errGetProfile := thirdparty.UpdatePlayerSummary(ctx, &person); errGetProfile != nil {
				log.Error("Failed to fetch user profile", zap.Error(errGetProfile))
				responseErr(ctx, http.StatusInternalServerError, nil)

				return
			}

			if errSave := env.Store().SavePerson(ctx, &person); errSave != nil {
				log.Error("Failed to save player summary update", zap.Error(errSave))
			}
		}

		person.DiscordID = discordID

		if errSave := env.Store().SavePerson(ctx, &person); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		ctx.JSON(http.StatusOK, nil)

		log.Info("Discord account linked successfully",
			zap.String("discord_id", discordID), zap.Int64("sid64", sid.Int64()))
	}
}

func onAPILogout(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		fingerprint, errCookie := ctx.Cookie(fingerprintCookieName)
		if errCookie != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		conf := env.Config()

		parsedExternal, errExternal := url.Parse(conf.General.ExternalURL)
		if errExternal != nil {
			ctx.Status(http.StatusInternalServerError)
			log.Error("Failed to parse ext url", zap.Error(errExternal))

			return
		}

		ctx.SetCookie(fingerprintCookieName, "", -1, "/api",
			parsedExternal.Hostname(), conf.General.Mode == config.ReleaseMode, true)

		auth := domain.PersonAuth{}
		if errGet := env.Store().GetPersonAuthByRefreshToken(ctx, fingerprint, &auth); errGet != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Warn("Failed to load person via fingerprint")

			return
		}

		if errDelete := env.Store().DeletePersonAuth(ctx, auth.PersonAuthID); errDelete != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to delete person auth on logout", zap.Error(errDelete))

			return
		}

		ctx.Status(http.StatusNoContent)
	}
}

func onAPICurrentProfile(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		profile := currentUserProfile(ctx)
		if !profile.SteamID.Valid() {
			log.Error("Failed to load user profile",
				zap.Int64("sid64", profile.SteamID.Int64()),
				zap.String("name", profile.Name),
				zap.String("permission_level", profile.PermissionLevel.String()))
			responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		ctx.JSON(http.StatusOK, profile)
	}
}

func onAPICurrentProfileNotifications(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		currentProfile := currentUserProfile(ctx)

		var req domain.NotificationQuery
		if !bind(ctx, log, &req) {
			return
		}

		req.SteamID = currentProfile.SteamID

		notifications, count, errNot := env.Store().GetPersonNotifications(ctx, req)
		if errNot != nil {
			if errors.Is(errNot, errs.ErrNoResult) {
				ctx.JSON(http.StatusOK, []domain.UserNotification{})

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, LazyResult{
			Count: count,
			Data:  notifications,
		})
	}
}

func onAPIGetReport(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportID, errParam := getInt64Param(ctx, "report_id")
		if errParam != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var report reportWithAuthor
		if errReport := env.Store().GetReport(ctx, reportID, &report.Report); errReport != nil {
			if errors.Is(errs.DBErr(errReport), errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to load report", zap.Error(errReport))

			return
		}

		if !checkPrivilege(ctx, currentUserProfile(ctx), steamid.Collection{report.Report.SourceID}, domain.PModerator) {
			responseErr(ctx, http.StatusUnauthorized, errPermissionDenied)

			return
		}

		if errAuthor := env.Store().GetPersonBySteamID(ctx, report.Report.SourceID, &report.Author); errAuthor != nil {
			if errors.Is(errs.DBErr(errAuthor), errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusBadRequest, errBadRequest)
			log.Error("Failed to load report author", zap.Error(errAuthor))

			return
		}

		if errSubject := env.Store().GetPersonBySteamID(ctx, report.Report.TargetID, &report.Subject); errSubject != nil {
			if errors.Is(errs.DBErr(errSubject), errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusBadRequest, errBadRequest)
			log.Error("Failed to load report subject", zap.Error(errSubject))

			return
		}

		ctx.JSON(http.StatusOK, report)
	}
}

type reportWithAuthor struct {
	Author  domain.Person `json:"author"`
	Subject domain.Person `json:"subject"`
	domain.Report
}

func onAPIGetReports(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)

		var req domain.ReportQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		if req.Limit <= 0 && req.Limit > 100 {
			req.Limit = 25
		}

		// Make sure the person requesting is either a moderator, or a user
		// only able to request their own reports
		var sourceID steamid.SID64

		if user.PermissionLevel < domain.PModerator {
			sourceID = user.SteamID
		} else if req.SourceID != "" {
			sid, errSourceID := req.SourceID.SID64(ctx)
			if errSourceID != nil {
				responseErr(ctx, http.StatusBadRequest, errBadRequest)

				return
			}

			sourceID = sid
		}

		if sourceID.Valid() {
			req.SourceID = domain.StringSID(sourceID.String())
		}

		var userReports []reportWithAuthor

		reports, count, errReports := env.Store().GetReports(ctx, req)
		if errReports != nil {
			if errors.Is(errs.DBErr(errReports), errs.ErrNoResult) {
				ctx.JSON(http.StatusNoContent, nil)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		var authorIds steamid.Collection
		for _, report := range reports {
			authorIds = append(authorIds, report.SourceID)
		}

		authors, errAuthors := env.Store().GetPeopleBySteamID(ctx, fp.Uniq(authorIds))
		if errAuthors != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		authorMap := authors.AsMap()

		var subjectIds steamid.Collection
		for _, report := range reports {
			subjectIds = append(subjectIds, report.TargetID)
		}

		subjects, errSubjects := env.Store().GetPeopleBySteamID(ctx, fp.Uniq(subjectIds))
		if errSubjects != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		subjectMap := subjects.AsMap()

		for _, report := range reports {
			userReports = append(userReports, reportWithAuthor{
				Author:  authorMap[report.SourceID],
				Report:  report,
				Subject: subjectMap[report.TargetID],
			})
		}

		if userReports == nil {
			userReports = []reportWithAuthor{}
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, userReports))
	}
}

func onAPISetReportStatus(env Env) gin.HandlerFunc {
	type stateUpdateReq struct {
		Status domain.ReportStatus `json:"status"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportID, errParam := getInt64Param(ctx, "report_id")
		if errParam != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var req stateUpdateReq
		if !bind(ctx, log, &req) {
			return
		}

		var report domain.Report
		if errGet := env.Store().GetReport(ctx, reportID, &report); errGet != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to get report to set state", zap.Error(errGet))

			return
		}

		if report.ReportStatus == req.Status {
			ctx.JSON(http.StatusConflict, errs.ErrDuplicate)

			return
		}

		original := report.ReportStatus

		report.ReportStatus = req.Status
		if errSave := env.Store().SaveReport(ctx, &report); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to save report state", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusAccepted, nil)
		log.Info("Report status changed",
			zap.Int64("report_id", report.ReportID),
			zap.String("from_status", original.String()),
			zap.String("to_status", report.ReportStatus.String()))
		// discord.SendDiscord(model.NotificationPayload{
		//	Sids:     steamid.Collection{report.SourceID},
		//	Severity: db.SeverityInfo,
		//	Message:  "Report status updated",
		//	Link:     report.ToURL(),
		// })
	} //nolint:wsl
}

type UserUploadedFile struct {
	Content string `json:"content"`
	Name    string `json:"name"`
	Mime    string `json:"mime"`
	Size    int64  `json:"size"`
}

func onAPISaveMedia(env Env) gin.HandlerFunc {
	MediaSafeMimeTypesImages := []string{
		"image/gif",
		"image/jpeg",
		"image/png",
		"image/webp",
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req UserUploadedFile
		if !bind(ctx, log, &req) {
			return
		}

		content, decodeErr := base64.StdEncoding.DecodeString(req.Content)
		if decodeErr != nil {
			ctx.JSON(http.StatusBadRequest, errBadRequest)

			return
		}

		media, errMedia := domain.NewMedia(currentUserProfile(ctx).SteamID, req.Name, req.Mime, content)
		if errMedia != nil {
			ctx.JSON(http.StatusBadRequest, errBadRequest)
			log.Error("Invalid media uploaded", zap.Error(errMedia))
		}

		conf := env.Config()

		asset, errAsset := domain.NewAsset(content, conf.S3.BucketMedia, "")
		if errAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errAssetCreateFailed)

			return
		}

		if errPut := env.Assets().Put(ctx, conf.S3.BucketMedia, asset.Name, bytes.NewReader(content), asset.Size, asset.MimeType); errPut != nil {
			responseErr(ctx, http.StatusInternalServerError, errAssetPut)

			log.Error("Failed to save user media to s3 backend", zap.Error(errPut))

			return
		}

		if errSaveAsset := env.Store().SaveAsset(ctx, &asset); errSaveAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errAssetSave)

			log.Error("Failed to save user asset to s3 backend", zap.Error(errSaveAsset))
		}

		media.Asset = asset

		media.Contents = nil

		if !slices.Contains(MediaSafeMimeTypesImages, media.MimeType) {
			responseErr(ctx, http.StatusBadRequest, errInvalidFormat)
			log.Error("User tried uploading image with forbidden mimetype",
				zap.String("mime", media.MimeType), zap.String("name", media.Name))

			return
		}

		if errSave := env.Store().SaveMedia(ctx, &media); errSave != nil {
			log.Error("Failed to save wiki media", zap.Error(errSave))

			if errors.Is(errSave, errs.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, errDuplicateMediaName)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errSaveMedia)

			return
		}

		ctx.JSON(http.StatusCreated, media)
	}
}

func onAPIGetReportMessages(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportID, errParam := getInt64Param(ctx, "report_id")
		if errParam != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var report domain.Report
		if errGetReport := env.Store().GetReport(ctx, reportID, &report); errGetReport != nil {
			responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		if !checkPrivilege(ctx, currentUserProfile(ctx), steamid.Collection{report.SourceID, report.TargetID}, domain.PModerator) {
			return
		}

		reportMessages, errGetReportMessages := env.Store().GetReportMessages(ctx, reportID)
		if errGetReportMessages != nil {
			responseErr(ctx, http.StatusNotFound, errs.ErrPlayerNotFound)

			return
		}

		if reportMessages == nil {
			reportMessages = []domain.ReportMessage{}
		}

		ctx.JSON(http.StatusOK, reportMessages)
	}
}

func onAPIPostReportMessage(env Env) gin.HandlerFunc {
	type newMessage struct {
		Message string `json:"message"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportID, errID := getInt64Param(ctx, "report_id")
		if errID != nil || reportID == 0 {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var req newMessage
		if !bind(ctx, log, &req) {
			return
		}

		if req.Message == "" {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		var report domain.Report
		if errReport := env.Store().GetReport(ctx, reportID, &report); errReport != nil {
			if errors.Is(errs.DBErr(errReport), errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to load report", zap.Error(errReport))

			return
		}

		person := currentUserProfile(ctx)
		msg := domain.NewReportMessage(reportID, person.SteamID, req.Message)

		if errSave := env.Store().SaveReportMessage(ctx, &msg); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		report.UpdatedOn = time.Now()

		if errSave := env.Store().SaveReport(ctx, &report); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to update report activity", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, msg)

		conf := env.Config()

		env.SendPayload(conf.Discord.LogChannelID,
			discord.NewReportMessageResponse(msg.MessageMD, conf.ExtURL(report), person, conf.ExtURL(person)))
	}
}

func onAPIEditReportMessage(env Env) gin.HandlerFunc {
	type editMessage struct {
		BodyMD string `json:"body_md"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportMessageID, errID := getInt64Param(ctx, "report_message_id")
		if errID != nil || reportMessageID == 0 {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var existing domain.ReportMessage
		if errExist := env.Store().GetReportMessageByID(ctx, reportMessageID, &existing); errExist != nil {
			if errors.Is(errExist, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrPlayerNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		curUser := currentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
			return
		}

		var req editMessage
		if !bind(ctx, log, &req) {
			return
		}

		if req.BodyMD == "" {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		if req.BodyMD == existing.MessageMD {
			responseErr(ctx, http.StatusConflict, errs.ErrDuplicate)

			return
		}

		existing.MessageMD = req.BodyMD
		if errSave := env.Store().SaveReportMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, req)

		conf := env.Config()
		msg := discord.EditReportMessageResponse(req.BodyMD, existing.MessageMD,
			conf.ExtURLRaw("/report/%d", existing.ReportID), curUser, conf.ExtURL(curUser))
		env.SendPayload(env.Config().Discord.LogChannelID, msg)
	}
}

func onAPIDeleteReportMessage(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportMessageID, errID := getInt64Param(ctx, "report_message_id")
		if errID != nil || reportMessageID == 0 {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var existing domain.ReportMessage
		if errExist := env.Store().GetReportMessageByID(ctx, reportMessageID, &existing); errExist != nil {
			if errors.Is(errExist, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		curUser := currentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
			return
		}

		existing.Deleted = true
		if errSave := env.Store().SaveReportMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusNoContent, nil)

		conf := env.Config()

		env.SendPayload(conf.Discord.LogChannelID, discord.DeleteReportMessage(existing, curUser, conf.ExtURL(curUser)))
	}
}

func onAPIGetBanByID(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		curUser := currentUserProfile(ctx)

		banID, errID := getInt64Param(ctx, "ban_id")
		if errID != nil || banID == 0 {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		deletedOk := false

		fullValue, fullOk := ctx.GetQuery("deleted")
		if fullOk {
			deleted, deletedOkErr := strconv.ParseBool(fullValue)
			if deletedOkErr != nil {
				log.Error("Failed to parse ban full query value", zap.Error(deletedOkErr))
			} else {
				deletedOk = deleted
			}
		}

		bannedPerson := domain.NewBannedPerson()
		if errGetBan := env.Store().GetBanByBanID(ctx, banID, &bannedPerson, deletedOk); errGetBan != nil {
			responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)
			log.Error("Failed to fetch steam ban", zap.Error(errGetBan))

			return
		}

		if !checkPrivilege(ctx, curUser, steamid.Collection{bannedPerson.TargetID}, domain.PModerator) {
			return
		}

		loadBanMeta(&bannedPerson)
		ctx.JSON(http.StatusOK, bannedPerson)
	}
}

func onAPIGetBanMessages(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banID, errParam := getInt64Param(ctx, "ban_id")
		if errParam != nil {
			responseErr(ctx, http.StatusNotFound, errInvalidParameter)

			return
		}

		banPerson := domain.NewBannedPerson()
		if errGetBan := env.Store().GetBanByBanID(ctx, banID, &banPerson, true); errGetBan != nil {
			responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		if !checkPrivilege(ctx, currentUserProfile(ctx), steamid.Collection{banPerson.TargetID, banPerson.SourceID}, domain.PModerator) {
			return
		}

		banMessages, errGetBanMessages := env.Store().GetBanMessages(ctx, banID)
		if errGetBanMessages != nil {
			responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		ctx.JSON(http.StatusOK, banMessages)
	}
}

func onAPIPostBanMessage(env Env) gin.HandlerFunc {
	type newMessage struct {
		Message string `json:"message"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banID, errID := getInt64Param(ctx, "ban_id")
		if errID != nil || banID == 0 {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var req newMessage
		if !bind(ctx, log, &req) {
			return
		}

		if req.Message == "" {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		bannedPerson := domain.NewBannedPerson()
		if errReport := env.Store().GetBanByBanID(ctx, banID, &bannedPerson, true); errReport != nil {
			if errors.Is(errs.DBErr(errReport), errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to load ban", zap.Error(errReport))

			return
		}

		curUserProfile := currentUserProfile(ctx)
		if bannedPerson.AppealState != domain.Open && curUserProfile.PermissionLevel < domain.PModerator {
			responseErr(ctx, http.StatusForbidden, errPermissionDenied)
			log.Warn("User tried to bypass posting restriction",
				zap.Int64("ban_id", bannedPerson.BanID), zap.Int64("target_id", bannedPerson.TargetID.Int64()))

			return
		}

		msg := domain.NewBanAppealMessage(banID, curUserProfile.SteamID, req.Message)
		if errSave := env.Store().SaveBanMessage(ctx, &msg); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to save ban appeal message", zap.Error(errSave))

			return
		}

		msg.PermissionLevel = curUserProfile.PermissionLevel
		msg.Personaname = curUserProfile.Name
		msg.Avatarhash = curUserProfile.Avatarhash

		ctx.JSON(http.StatusCreated, msg)

		conf := env.Config()

		var target domain.Person
		if errTarget := env.Store().GetPersonBySteamID(ctx, bannedPerson.TargetID, &target); errTarget != nil {
			env.Log().Error("Failed to load target", zap.Error(errTarget))

			return
		}

		var source domain.Person
		if errSource := env.Store().GetPersonBySteamID(ctx, bannedPerson.SourceID, &source); errSource != nil {
			env.Log().Error("Failed to load source", zap.Error(errSource))

			return
		}

		env.SendPayload(conf.Discord.LogChannelID,
			discord.NewAppealMessage(msg.MessageMD, conf.ExtURL(bannedPerson.BanSteam), curUserProfile, conf.ExtURL(curUserProfile)))
	}
}

func onAPIEditBanMessage(env Env) gin.HandlerFunc {
	type editMessage struct {
		BodyMD string `json:"body_md"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportMessageID, errID := getIntParam(ctx, "ban_message_id")
		if errID != nil || reportMessageID == 0 {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var existing domain.BanAppealMessage
		if errExist := env.Store().GetBanMessageByID(ctx, reportMessageID, &existing); errExist != nil {
			if errors.Is(errExist, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		curUser := currentUserProfile(ctx)

		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
			return
		}

		var req editMessage
		if !bind(ctx, log, &req) {
			return
		}

		if req.BodyMD == "" {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		if req.BodyMD == existing.MessageMD {
			responseErr(ctx, http.StatusConflict, errs.ErrDuplicate)

			return
		}

		existing.MessageMD = req.BodyMD
		if errSave := env.Store().SaveBanMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to save ban appeal message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, req)

		conf := env.Config()

		env.SendPayload(conf.Discord.LogChannelID, discord.EditAppealMessage(existing, req.BodyMD, curUser, conf.ExtURL(curUser)))
	}
}

func onAPIDeleteBanMessage(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banMessageID, errID := getIntParam(ctx, "ban_message_id")
		if errID != nil || banMessageID == 0 {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var existing domain.BanAppealMessage
		if errExist := env.Store().GetBanMessageByID(ctx, banMessageID, &existing); errExist != nil {
			if errors.Is(errExist, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		curUser := currentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
			return
		}

		existing.Deleted = true
		if errSave := env.Store().SaveBanMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to save appeal message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusNoContent, nil)

		conf := env.Config()

		env.SendPayload(conf.Discord.LogChannelID, discord.DeleteAppealMessage(existing, curUser, conf.ExtURL(curUser)))
	}
}

func onAPIGetSourceBans(_ Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, errID := getSID64Param(ctx, "steam_id")
		if errID != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		records, errRecords := thirdparty.BDSourceBans(ctx, steamID)
		if errRecords != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, records)
	}
}

func onAPIGetMatch(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		matchID, errID := getUUIDParam(ctx, "match_id")
		if errID != nil {
			log.Error("Invalid match_id value", zap.Error(errID))
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var match domain.MatchResult

		errMatch := env.Store().MatchGetByID(ctx, matchID, &match)

		if errMatch != nil {
			if errors.Is(errMatch, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, match)
	}
}

func onAPIGetMatches(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.MatchesQueryOpts
		if !bind(ctx, log, &req) {
			return
		}

		// Don't let normal users query anybody but themselves
		user := currentUserProfile(ctx)
		if user.PermissionLevel <= domain.PUser {
			if !req.SteamID.Valid() {
				responseErr(ctx, http.StatusBadRequest, errBadRequest)

				return
			}

			if user.SteamID != req.SteamID {
				responseErr(ctx, http.StatusForbidden, errPermissionDenied)

				return
			}
		}

		matches, totalCount, matchesErr := env.Store().Matches(ctx, req)
		if matchesErr != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to perform query", zap.Error(matchesErr))

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(totalCount, matches))
	}
}

func onAPIQueryMessages(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.ChatHistoryQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		if req.Limit <= 0 || req.Limit > 1000 {
			req.Limit = 50
		}

		user := currentUserProfile(ctx)

		if user.PermissionLevel <= domain.PUser {
			req.Unrestricted = false
			beforeLimit := time.Now().Add(-time.Minute * 20)

			if req.DateEnd != nil && req.DateEnd.After(beforeLimit) {
				req.DateEnd = &beforeLimit
			}

			if req.DateEnd == nil {
				req.DateEnd = &beforeLimit
			}
		} else {
			req.Unrestricted = true
		}

		messages, count, errChat := env.Store().QueryChatHistory(ctx, req)
		if errChat != nil && !errors.Is(errChat, errs.ErrNoResult) {
			log.Error("Failed to query messages history",
				zap.Error(errChat), zap.String("sid", string(req.SourceID)))
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, messages))
	}
}

func onAPIGetStatsWeaponsOverall(ctx context.Context, env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	updater := NewDataUpdater(log, time.Minute*10, func() ([]domain.WeaponsOverallResult, error) {
		weaponStats, errUpdate := env.Store().WeaponsOverall(ctx)
		if errUpdate != nil && !errors.Is(errUpdate, errs.ErrNoResult) {
			return nil, errors.Join(errUpdate, ErrDataUpdate)
		}

		if weaponStats == nil {
			weaponStats = []domain.WeaponsOverallResult{}
		}

		return weaponStats, nil
	})

	go updater.Start(ctx)

	return func(ctx *gin.Context) {
		stats := updater.Data()

		ctx.JSON(http.StatusOK, newLazyResult(int64(len(stats)), stats))
	}
}

func onAPIGetsStatsWeapon(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type resp struct {
		LazyResult
		Weapon domain.Weapon `json:"weapon"`
	}

	return func(ctx *gin.Context) {
		weaponID, errWeaponID := getIntParam(ctx, "weapon_id")
		if errWeaponID != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var weapon domain.Weapon

		errWeapon := env.Store().GetWeaponByID(ctx, weaponID, &weapon)

		if errWeapon != nil {
			responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		weaponStats, errChat := env.Store().WeaponsOverallTopPlayers(ctx, weaponID)
		if errChat != nil && !errors.Is(errChat, errs.ErrNoResult) {
			log.Error("Failed to get weapons overall top stats",
				zap.Error(errChat))
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		if weaponStats == nil {
			weaponStats = []domain.PlayerWeaponResult{}
		}

		ctx.JSON(http.StatusOK, resp{LazyResult: newLazyResult(int64(len(weaponStats)), weaponStats), Weapon: weapon})
	}
}

func onAPIGetStatsPlayersOverall(ctx context.Context, env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	updater := NewDataUpdater(log, time.Minute*10, func() ([]domain.PlayerWeaponResult, error) {
		updatedStats, errChat := env.Store().PlayersOverallByKills(ctx, 1000)
		if errChat != nil && !errors.Is(errChat, errs.ErrNoResult) {
			return nil, errors.Join(errChat, ErrDataUpdate)
		}

		return updatedStats, nil
	})

	go updater.Start(ctx)

	return func(ctx *gin.Context) {
		stats := updater.Data()
		ctx.JSON(http.StatusOK, newLazyResult(int64(len(stats)), stats))
	}
}

func onAPIGetStatsHealersOverall(ctx context.Context, env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	updater := NewDataUpdater(log, time.Minute*10, func() ([]domain.HealingOverallResult, error) {
		updatedStats, errChat := env.Store().HealersOverallByHealing(ctx, 250)
		if errChat != nil && !errors.Is(errChat, errs.ErrNoResult) {
			return nil, errors.Join(errChat, ErrDataUpdate)
		}

		return updatedStats, nil
	})

	go updater.Start(ctx)

	return func(ctx *gin.Context) {
		stats := updater.Data()
		ctx.JSON(http.StatusOK, newLazyResult(int64(len(stats)), stats))
	}
}

func onAPIGetPlayerWeaponStatsOverall(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		steamID, errSteamID := getSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		weaponStats, errChat := env.Store().WeaponsOverallByPlayer(ctx, steamID)
		if errChat != nil && !errors.Is(errChat, errs.ErrNoResult) {
			log.Error("Failed to query player weapons stats",
				zap.Error(errChat))
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		if weaponStats == nil {
			weaponStats = []domain.WeaponsOverallResult{}
		}

		ctx.JSON(http.StatusOK, newLazyResult(int64(len(weaponStats)), weaponStats))
	}
}

func onAPIGetPlayerClassStatsOverall(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		steamID, errSteamID := getSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		classStats, errChat := env.Store().PlayerOverallClassStats(ctx, steamID)
		if errChat != nil && !errors.Is(errChat, errs.ErrNoResult) {
			log.Error("Failed to query player class stats",
				zap.Error(errChat))
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		if classStats == nil {
			classStats = []domain.PlayerClassOverallResult{}
		}

		ctx.JSON(http.StatusOK, newLazyResult(int64(len(classStats)), classStats))
	}
}

func onAPIGetPlayerStatsOverall(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		steamID, errSteamID := getSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var por domain.PlayerOverallResult
		if errChat := env.Store().PlayerOverallStats(ctx, steamID, &por); errChat != nil && !errors.Is(errChat, errs.ErrNoResult) {
			log.Error("Failed to query player stats overall",
				zap.Error(errChat))
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, por)
	}
}

func onAPISaveContestEntryMedia(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		contest, success := contestFromCtx(ctx, env)
		if !success {
			return
		}

		var req UserUploadedFile
		if !bind(ctx, log, &req) {
			return
		}

		content, decodeErr := base64.StdEncoding.DecodeString(req.Content)
		if decodeErr != nil {
			ctx.JSON(http.StatusBadRequest, errBadRequest)

			return
		}

		media, errMedia := domain.NewMedia(currentUserProfile(ctx).SteamID, req.Name, req.Mime, content)
		if errMedia != nil {
			ctx.JSON(http.StatusBadRequest, errBadRequest)
			log.Error("Invalid media uploaded", zap.Error(errMedia))
		}

		conf := env.Config()

		asset, errAsset := domain.NewAsset(content, conf.S3.BucketMedia, "")
		if errAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errAssetCreateFailed)

			return
		}

		if errPut := env.Assets().Put(ctx, conf.S3.BucketMedia, asset.Name, bytes.NewReader(content), asset.Size, asset.MimeType); errPut != nil {
			responseErr(ctx, http.StatusInternalServerError, errAssetPut)

			log.Error("Failed to save user contest entry media to s3 backend", zap.Error(errPut))

			return
		}

		if errSaveAsset := env.Store().SaveAsset(ctx, &asset); errSaveAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errAssetSave)

			log.Error("Failed to save user asset to s3 backend", zap.Error(errSaveAsset))
		}

		media.Asset = asset

		media.Contents = nil

		if !contest.MimeTypeAcceptable(media.MimeType) {
			responseErr(ctx, http.StatusUnsupportedMediaType, errInvalidFormat)
			log.Error("User tried uploading file with forbidden mimetype",
				zap.String("mime", media.MimeType), zap.String("name", media.Name))

			return
		}

		if errSave := env.Store().SaveMedia(ctx, &media); errSave != nil {
			log.Error("Failed to save user contest media", zap.Error(errSave))

			if errors.Is(errs.DBErr(errSave), errs.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, errDuplicateMediaName)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errSaveMedia)

			return
		}

		ctx.JSON(http.StatusCreated, media)
	}
}

func onAPISaveContestEntryVote(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type voteResult struct {
		CurrentVote string `json:"current_vote"`
	}

	return func(ctx *gin.Context) {
		contest, success := contestFromCtx(ctx, env)
		if !success {
			return
		}

		contestEntryID, errContestEntryID := getUUIDParam(ctx, "contest_entry_id")
		if errContestEntryID != nil {
			ctx.JSON(http.StatusNotFound, errs.ErrNotFound)
			log.Error("Invalid contest entry id option")

			return
		}

		direction := strings.ToLower(ctx.Param("direction"))
		if direction != "up" && direction != "down" {
			ctx.JSON(http.StatusBadRequest, errBadRequest)
			log.Error("Invalid vote direction option")

			return
		}

		if !contest.Voting || !contest.DownVotes && direction != "down" {
			ctx.JSON(http.StatusBadRequest, errBadRequest)
			log.Error("Voting not enabled")

			return
		}

		currentUser := currentUserProfile(ctx)

		if errVote := env.Store().ContestEntryVote(ctx, contestEntryID, currentUser.SteamID, direction == "up"); errVote != nil {
			if errors.Is(errVote, errs.ErrVoteDeleted) {
				ctx.JSON(http.StatusOK, voteResult{""})

				return
			}

			ctx.JSON(http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusCreated, voteResult{direction})
	}
}

func onAPISaveContestEntrySubmit(env Env) gin.HandlerFunc {
	type entryReq struct {
		Description string    `json:"description"`
		AssetID     uuid.UUID `json:"asset_id"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)
		contest, success := contestFromCtx(ctx, env)

		if !success {
			return
		}

		var req entryReq
		if !bind(ctx, log, &req) {
			return
		}

		if contest.MediaTypes != "" {
			var media domain.Media
			if errMedia := env.Store().GetMediaByAssetID(ctx, req.AssetID, &media); errMedia != nil {
				responseErr(ctx, http.StatusFailedDependency, errFetchMedia)

				return
			}

			if !contest.MimeTypeAcceptable(media.MimeType) {
				responseErr(ctx, http.StatusFailedDependency, errInvalidFormat)

				return
			}
		}

		existingEntries, errEntries := env.Store().ContestEntries(ctx, contest.ContestID)
		if errEntries != nil && !errors.Is(errEntries, errs.ErrNoResult) {
			responseErr(ctx, http.StatusInternalServerError, errContestLoadEntries)

			return
		}

		own := 0

		for _, entry := range existingEntries {
			if entry.SteamID == user.SteamID {
				own++
			}

			if own >= contest.MaxSubmissions {
				responseErr(ctx, http.StatusForbidden, errContestMaxEntries)

				return
			}
		}

		steamID := currentUserProfile(ctx).SteamID

		entry, errEntry := contest.NewEntry(steamID, req.AssetID, req.Description)
		if errEntry != nil {
			responseErr(ctx, http.StatusInternalServerError, errEntryCreate)

			return
		}

		if errSave := env.Store().ContestEntrySave(ctx, entry); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errEntrySave)

			return
		}

		ctx.JSON(http.StatusCreated, entry)

		log.Info("New contest entry submitted", zap.String("contest_id", contest.ContestID.String()))
	}
}

func onAPIDeleteContestEntry(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)

		contestEntryID, idErr := getUUIDParam(ctx, "contest_entry_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		var entry domain.ContestEntry

		if errContest := env.Store().ContestEntry(ctx, contestEntryID, &entry); errContest != nil {
			if errors.Is(errContest, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrUnknownID)

				return
			}

			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			log.Error("Error getting contest entry for deletion", zap.Error(errContest))

			return
		}

		// Only >=moderators or the entry author are allowed to delete entries.
		if !(user.PermissionLevel >= domain.PModerator || user.SteamID == entry.SteamID) {
			responseErr(ctx, http.StatusForbidden, errPermissionDenied)

			return
		}

		var contest domain.Contest

		if errContest := env.Store().ContestByID(ctx, entry.ContestID, &contest); errContest != nil {
			if errors.Is(errContest, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrUnknownID)

				return
			}

			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			log.Error("Error getting contest", zap.Error(errContest))

			return
		}

		// Only allow mods to delete entries from expired contests.
		if user.SteamID == entry.SteamID && time.Since(contest.DateEnd) > 0 {
			responseErr(ctx, http.StatusForbidden, errPermissionDenied)

			log.Error("User tried to delete entry from expired contest")

			return
		}

		if errDelete := env.Store().ContestEntryDelete(ctx, entry.ContestEntryID); errDelete != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			log.Error("Error deleting contest entry", zap.Error(errDelete))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})

		log.Info("Contest deleted",
			zap.String("contest_id", entry.ContestID.String()),
			zap.String("contest_entry_id", entry.ContestEntryID.String()),
			zap.String("title", contest.Title))
	}
}

func onAPIThreadCreate(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type CreateThreadRequest struct {
		Title  string `json:"title"`
		BodyMD string `json:"body_md"`
		Sticky bool   `json:"sticky"`
		Locked bool   `json:"locked"`
	}

	type ThreadWithMessage struct {
		domain.ForumThread
		Message domain.ForumMessage `json:"message"`
	}

	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)

		env.Activity().Touch(user)

		forumID, errForumID := getIntParam(ctx, "forum_id")
		if errForumID != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		var req CreateThreadRequest
		if !bind(ctx, log, &req) {
			return
		}

		if len(req.BodyMD) <= 1 {
			responseErr(ctx, http.StatusBadRequest, fmt.Errorf("body: %w", store.ErrTooShort))

			return
		}

		if len(req.Title) <= 4 {
			responseErr(ctx, http.StatusBadRequest, fmt.Errorf("title: %w", store.ErrTooShort))

			return
		}

		var forum domain.Forum
		if errForum := env.Store().Forum(ctx, forumID, &forum); errForum != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		thread := forum.NewThread(req.Title, user.SteamID)
		thread.Sticky = req.Sticky
		thread.Locked = req.Locked

		if errSaveThread := env.Store().ForumThreadSave(ctx, &thread); errSaveThread != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			log.Error("Failed to save new thread", zap.Error(errSaveThread))

			return
		}

		message := thread.NewMessage(user.SteamID, req.BodyMD)

		if errSaveMessage := env.Store().ForumMessageSave(ctx, &message); errSaveMessage != nil {
			// Drop created thread.
			// TODO transaction
			if errRollback := env.Store().ForumThreadDelete(ctx, thread.ForumThreadID); errRollback != nil {
				log.Error("Failed to rollback new thread", zap.Error(errRollback))
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)

			log.Error("Failed to save new forum message", zap.Error(errSaveMessage))

			return
		}

		if errIncr := env.Store().ForumIncrMessageCount(ctx, forum.ForumID, true); errIncr != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			log.Error("Failed to increment message count", zap.Error(errIncr))

			return
		}

		ctx.JSON(http.StatusCreated, ThreadWithMessage{
			ForumThread: thread,
			Message:     message,
		})

		log.Info("Thread created")
	}
}

func onAPIThreadUpdate(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type threadUpdate struct {
		Title  string `json:"title"`
		Sticky bool   `json:"sticky"`
		Locked bool   `json:"locked"`
	}

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		forumThreadID, errForumTheadID := getInt64Param(ctx, "forum_thread_id")
		if errForumTheadID != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		var req threadUpdate
		if !bind(ctx, log, &req) {
			return
		}

		req.Title = util.SanitizeUGC(req.Title)

		if len(req.Title) < 2 {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		var thread domain.ForumThread
		if errGet := env.Store().ForumThread(ctx, forumThreadID, &thread); errGet != nil {
			if errors.Is(errGet, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)
			} else {
				responseErr(ctx, http.StatusInternalServerError, errInternal)
			}

			return
		}

		if thread.SourceID != currentUser.SteamID && !(currentUser.PermissionLevel >= domain.PModerator) {
			responseErr(ctx, http.StatusForbidden, errInternal)

			return
		}

		thread.Title = req.Title
		thread.Sticky = req.Sticky
		thread.Locked = req.Locked

		if errDelete := env.Store().ForumThreadSave(ctx, &thread); errDelete != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to update thread", zap.Error(errDelete))

			return
		}

		ctx.JSON(http.StatusOK, thread)
		log.Info("Thread updated", zap.Int64("forum_thread_id", thread.ForumThreadID))
	}
}

func onAPIThreadDelete(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		forumThreadID, errForumTheadID := getInt64Param(ctx, "forum_thread_id")
		if errForumTheadID != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		var thread domain.ForumThread
		if errGet := env.Store().ForumThread(ctx, forumThreadID, &thread); errGet != nil {
			if errors.Is(errGet, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)
			} else {
				responseErr(ctx, http.StatusInternalServerError, errInternal)
			}

			return
		}

		if errDelete := env.Store().ForumThreadDelete(ctx, thread.ForumThreadID); errDelete != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to delete thread", zap.Error(errDelete))

			return
		}

		var forum domain.Forum
		if errForum := env.Store().Forum(ctx, thread.ForumID, &forum); errForum != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to load forum", zap.Error(errForum))

			return
		}

		forum.CountThreads -= 1

		if errSave := env.Store().ForumSave(ctx, &forum); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to save thread count", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func onAPIThreadMessageUpdate(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type MessageUpdate struct {
		BodyMD string `json:"body_md"`
	}

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		env.Activity().Touch(currentUser)

		forumMessageID, errForumMessageID := getInt64Param(ctx, "forum_message_id")
		if errForumMessageID != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		var req MessageUpdate
		if !bind(ctx, log, &req) {
			return
		}

		var message domain.ForumMessage
		if errMessage := env.Store().ForumMessage(ctx, forumMessageID, &message); errMessage != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		if message.SourceID != currentUser.SteamID && !(currentUser.PermissionLevel >= domain.PModerator) {
			responseErr(ctx, http.StatusForbidden, errInternal)

			return
		}

		req.BodyMD = util.SanitizeUGC(req.BodyMD)

		if len(req.BodyMD) < 10 {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		message.BodyMD = req.BodyMD

		if errSave := env.Store().ForumMessageSave(ctx, &message); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, message)
	}
}

func onAPIMessageDelete(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		forumMessageID, errForumMessageID := getInt64Param(ctx, "forum_message_id")
		if errForumMessageID != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		var message domain.ForumMessage
		if err := env.Store().ForumMessage(ctx, forumMessageID, &message); err != nil {
			if errors.Is(err, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)
			} else {
				responseErr(ctx, http.StatusInternalServerError, errInternal)
			}

			return
		}

		var thread domain.ForumThread
		if err := env.Store().ForumThread(ctx, message.ForumThreadID, &thread); err != nil {
			if errors.Is(err, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)
			} else {
				responseErr(ctx, http.StatusInternalServerError, errInternal)
			}

			return
		}

		if thread.Locked {
			responseErr(ctx, http.StatusForbidden, errThreadLocked)

			return
		}

		messages, count, errMessage := env.Store().ForumMessages(ctx, domain.ThreadMessagesQueryFilter{ForumThreadID: message.ForumThreadID})
		if errMessage != nil || count <= 0 {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		isThreadParent := messages[0].ForumMessageID == message.ForumMessageID

		if isThreadParent {
			if err := env.Store().ForumThreadDelete(ctx, message.ForumThreadID); err != nil {
				responseErr(ctx, http.StatusInternalServerError, errInternal)
				log.Error("Failed to delete forum thread", zap.Error(err))

				return
			}

			// Delete the thread if it's the first message
			var forum domain.Forum
			if errForum := env.Store().Forum(ctx, thread.ForumID, &forum); errForum != nil {
				responseErr(ctx, http.StatusInternalServerError, errInternal)
				log.Error("Failed to load forum", zap.Error(errForum))

				return
			}

			forum.CountThreads -= 1

			if errSave := env.Store().ForumSave(ctx, &forum); errSave != nil {
				responseErr(ctx, http.StatusInternalServerError, errInternal)
				log.Error("Failed to save thread count", zap.Error(errSave))

				return
			}

			log.Error("Thread deleted due to parent deletion", zap.Int64("forum_thread_id", thread.ForumThreadID))
		} else {
			if errDelete := env.Store().ForumMessageDelete(ctx, message.ForumMessageID); errDelete != nil {
				responseErr(ctx, http.StatusInternalServerError, errInternal)
				log.Error("Failed to delete message", zap.Error(errDelete))

				return
			}

			log.Info("Thread message deleted", zap.Int64("forum_message_id", message.ForumMessageID))
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func onAPIThreadCreateReply(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type ThreadReply struct {
		BodyMD string `json:"body_md"`
	}

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		env.Activity().Touch(currentUser)

		forumThreadID, errForumID := getInt64Param(ctx, "forum_thread_id")
		if errForumID != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		var thread domain.ForumThread
		if errThread := env.Store().ForumThread(ctx, forumThreadID, &thread); errThread != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		if thread.Locked && currentUser.PermissionLevel < domain.PEditor {
			responseErr(ctx, http.StatusForbidden, errThreadLocked)

			return
		}

		var req ThreadReply
		if !bind(ctx, log, &req) {
			return
		}

		req.BodyMD = util.SanitizeUGC(req.BodyMD)

		if len(req.BodyMD) < 3 {
			responseErr(ctx, http.StatusBadRequest, fmt.Errorf("body: %w", store.ErrTooShort))

			return
		}

		newMessage := thread.NewMessage(currentUser.SteamID, req.BodyMD)
		if errSave := env.Store().ForumMessageSave(ctx, &newMessage); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		var message domain.ForumMessage
		if errFetch := env.Store().ForumMessage(ctx, newMessage.ForumMessageID, &message); errFetch != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		if errIncr := env.Store().ForumIncrMessageCount(ctx, thread.ForumID, true); errIncr != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			log.Error("Failed to increment message count", zap.Error(errIncr))
		}

		newMessage.Personaname = currentUser.Name
		newMessage.Avatarhash = currentUser.Avatarhash
		newMessage.PermissionLevel = currentUser.PermissionLevel
		newMessage.Online = true

		ctx.JSON(http.StatusCreated, newMessage)
	}
}

func onAPIGetPersonSettings(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)

		var settings domain.PersonSettings

		if err := env.Store().GetPersonSettings(ctx, user.SteamID, &settings); err != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to fetch person settings", zap.Error(err), zap.Int64("steam_id", user.SteamID.Int64()))

			return
		}

		ctx.JSON(http.StatusOK, settings)
	}
}

func onAPIPostPersonSettings(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type settingsUpdateReq struct {
		ForumSignature       string `json:"forum_signature"`
		ForumProfileMessages bool   `json:"forum_profile_messages"`
		StatsHidden          bool   `json:"stats_hidden"`
	}

	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)

		var req settingsUpdateReq

		if !bind(ctx, log, &req) {
			return
		}

		var settings domain.PersonSettings

		if err := env.Store().GetPersonSettings(ctx, user.SteamID, &settings); err != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to fetch person settings", zap.Error(err), zap.Int64("steam_id", user.SteamID.Int64()))

			return
		}

		settings.ForumProfileMessages = req.ForumProfileMessages
		settings.StatsHidden = req.StatsHidden
		settings.ForumSignature = util.SanitizeUGC(req.ForumSignature)

		if err := env.Store().SavePersonSettings(ctx, &settings); err != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to save person settings", zap.Error(err), zap.Int64("steam_id", user.SteamID.Int64()))

			return
		}

		ctx.JSON(http.StatusOK, settings)
	}
}
