package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func makeGetTokenKey(cookieKey string) func(_ *jwt.Token) (any, error) {
	return func(_ *jwt.Token) (any, error) {
		return []byte(cookieKey), nil
	}
}

const fingerprintCookieName = "fingerprint"

// onTokenRefresh handles generating new token pairs to access the api
// NOTE: All error code paths must return 401 (Unauthorized).
func onTokenRefresh(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

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

		refreshToken, errParseClaims := jwt.ParseWithClaims(refreshTokenString, &userClaims, makeGetTokenKey(app.conf.HTTP.CookieKey))
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

		var auth store.PersonAuth
		if authError := app.db.GetPersonAuthByRefreshToken(ctx, fingerprint, &auth); authError != nil {
			ctx.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		tokens, errMakeToken := makeTokens(ctx, app.db, app.conf.HTTP.CookieKey, auth.SteamID, false)
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

func onOAuthDiscordCallback(app *App) gin.HandlerFunc {
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

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())
	client := util.NewHTTPClient()

	fetchDiscordID := func(ctx context.Context, accessToken string) (string, error) {
		req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, "https://discord.com/api/users/@me", nil)
		if errReq != nil {
			return "", errors.Wrap(errReq, "Failed to create new request")
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		resp, errResp := client.Do(req)

		if errResp != nil {
			return "", errors.Wrap(errResp, "Failed to perform http request")
		}

		defer func() {
			_ = resp.Body.Close()
		}()

		body, errBody := io.ReadAll(resp.Body)
		if errBody != nil {
			return "", errors.Wrap(errBody, "Failed to read response body")
		}

		var details discordUserDetail
		if errJSON := json.Unmarshal(body, &details); errJSON != nil {
			return "", errors.Wrap(errJSON, "Failed to unmarshal response")
		}

		return details.ID, nil
	}

	fetchToken := func(ctx context.Context, code string) (string, error) {
		// v, _ := go_oauth_pkce_code_verifier.CreateCodeVerifierFromBytes([]byte(code))
		form := url.Values{}
		form.Set("client_id", app.conf.Discord.AppID)
		form.Set("client_secret", app.conf.Discord.AppSecret)
		form.Set("redirect_uri", app.ExtURLRaw("/login/discord"))
		form.Set("code", code)
		form.Set("grant_type", "authorization_code")
		// form.Set("state", state.String())
		form.Set("scope", "identify")
		req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, "https://discord.com/api/oauth2/token", strings.NewReader(form.Encode()))

		if errReq != nil {
			return "", errors.Wrap(errReq, "Failed to create new request")
		}

		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		resp, errResp := client.Do(req)
		if errResp != nil {
			return "", errors.Wrap(errResp, "Failed to perform http request")
		}

		defer func() {
			_ = resp.Body.Close()
		}()

		body, errBody := io.ReadAll(resp.Body)
		if errBody != nil {
			return "", errors.Wrap(errBody, "Failed to read response body")
		}

		var atr accessTokenResp
		if errJSON := json.Unmarshal(body, &atr); errJSON != nil {
			return "", errors.Wrap(errJSON, "Failed to decode response body")
		}

		if atr.AccessToken == "" {
			return "", errors.New("Empty token returned")
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

		var discordPerson store.Person
		if errDp := app.db.GetPersonByDiscordID(ctx, discordID, &discordPerson); errDp != nil {
			if !errors.Is(errDp, store.ErrNoResult) {
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

		var person store.Person
		if errPerson := app.PersonBySID(ctx, sid, &person); errPerson != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		person.DiscordID = discordID

		if errSave := app.db.SavePerson(ctx, &person); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		ctx.JSON(http.StatusOK, nil)

		log.Info("Discord account linked successfully",
			zap.String("discord_id", discordID), zap.Int64("sid64", sid.Int64()))
	}
}

func onAPILogout(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		fingerprint, errCookie := ctx.Cookie(fingerprintCookieName)
		if errCookie != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		parsedExternal, errExternal := url.Parse(app.conf.General.ExternalURL)
		if errExternal != nil {
			ctx.Status(http.StatusInternalServerError)
			log.Error("Failed to parse ext url", zap.Error(errExternal))

			return
		}

		ctx.SetCookie(fingerprintCookieName, "", -1, "/api",
			parsedExternal.Hostname(), app.conf.General.Mode == ReleaseMode, true)

		auth := store.PersonAuth{}
		if errGet := app.db.GetPersonAuthByRefreshToken(ctx, fingerprint, &auth); errGet != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Warn("Failed to load person via fingerprint")

			return
		}

		if errDelete := app.db.DeletePersonAuth(ctx, auth.PersonAuthID); errDelete != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)
			log.Error("Failed to delete person auth on logout", zap.Error(errDelete))

			return
		}

		ctx.Status(http.StatusNoContent)
	}
}

func onAPICurrentProfile(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		profile := currentUserProfile(ctx)
		if !profile.SteamID.Valid() {
			log.Error("Failed to load user profile",
				zap.Int64("sid64", profile.SteamID.Int64()),
				zap.String("name", profile.Name),
				zap.String("permission_level", profile.PermissionLevel.String()))
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		ctx.JSON(http.StatusOK, profile)
	}
}

func onAPICurrentProfileNotifications(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		currentProfile := currentUserProfile(ctx)

		var req store.NotificationQuery
		if !bind(ctx, log, &req) {
			return
		}

		req.SteamID = currentProfile.SteamID

		notifications, count, errNot := app.db.GetPersonNotifications(ctx, req)
		if errNot != nil {
			if errors.Is(errNot, store.ErrNoResult) {
				ctx.JSON(http.StatusOK, []store.UserNotification{})

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, LazyResult{
			Count: count,
			Data:  notifications,
		})
	}
}

func onAPIGetReport(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportID, errParam := getInt64Param(ctx, "report_id")
		if errParam != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var report reportWithAuthor
		if errReport := app.db.GetReport(ctx, reportID, &report.Report); errReport != nil {
			if errors.Is(store.Err(errReport), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to load report", zap.Error(errReport))

			return
		}

		if !checkPrivilege(ctx, currentUserProfile(ctx), steamid.Collection{report.Report.SourceID}, consts.PModerator) {
			responseErr(ctx, http.StatusUnauthorized, consts.ErrPermissionDenied)

			return
		}

		if errAuthor := app.PersonBySID(ctx, report.Report.SourceID, &report.Author); errAuthor != nil {
			if errors.Is(store.Err(errAuthor), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)
			log.Error("Failed to load report author", zap.Error(errAuthor))

			return
		}

		if errSubject := app.PersonBySID(ctx, report.Report.TargetID, &report.Subject); errSubject != nil {
			if errors.Is(store.Err(errSubject), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)
			log.Error("Failed to load report subject", zap.Error(errSubject))

			return
		}

		ctx.JSON(http.StatusOK, report)
	}
}

type reportWithAuthor struct {
	Author  store.Person `json:"author"`
	Subject store.Person `json:"subject"`
	store.Report
}

func onAPIGetReports(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)

		var req store.ReportQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		if req.Limit <= 0 && req.Limit > 100 {
			req.Limit = 25
		}

		sourceID, errSourceID := req.SourceID.SID64(ctx)
		if errSourceID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if user.SteamID != sourceID {
			responseErr(ctx, http.StatusForbidden, consts.ErrPermissionDenied)

			return
		}

		var userReports []reportWithAuthor

		reports, count, errReports := app.db.GetReports(ctx, req)
		if errReports != nil {
			if errors.Is(store.Err(errReports), store.ErrNoResult) {
				ctx.JSON(http.StatusNoContent, nil)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var authorIds steamid.Collection
		for _, report := range reports {
			authorIds = append(authorIds, report.SourceID)
		}

		authors, errAuthors := app.db.GetPeopleBySteamID(ctx, fp.Uniq(authorIds))
		if errAuthors != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		authorMap := authors.AsMap()

		var subjectIds steamid.Collection
		for _, report := range reports {
			subjectIds = append(subjectIds, report.TargetID)
		}

		subjects, errSubjects := app.db.GetPeopleBySteamID(ctx, fp.Uniq(subjectIds))
		if errSubjects != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

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

func onAPISetReportStatus(app *App) gin.HandlerFunc {
	type stateUpdateReq struct {
		Status store.ReportStatus `json:"status"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportID, errParam := getInt64Param(ctx, "report_id")
		if errParam != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req stateUpdateReq
		if !bind(ctx, log, &req) {
			return
		}

		var report store.Report
		if errGet := app.db.GetReport(ctx, reportID, &report); errGet != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to get report to set state", zap.Error(errGet))

			return
		}

		if report.ReportStatus == req.Status {
			ctx.JSON(http.StatusConflict, consts.ErrDuplicate)

			return
		}

		original := report.ReportStatus

		report.ReportStatus = req.Status
		if errSave := app.db.SaveReport(ctx, &report); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
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
		//	Severity: store.SeverityInfo,
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

func onAPISaveMedia(app *App) gin.HandlerFunc {
	MediaSafeMimeTypesImages := []string{
		"image/gif",
		"image/jpeg",
		"image/png",
		"image/webp",
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req UserUploadedFile
		if !bind(ctx, log, &req) {
			return
		}

		content, decodeErr := base64.StdEncoding.DecodeString(req.Content)
		if decodeErr != nil {
			ctx.JSON(http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		media, errMedia := store.NewMedia(currentUserProfile(ctx).SteamID, req.Name, req.Mime, content)
		if errMedia != nil {
			ctx.JSON(http.StatusBadRequest, consts.ErrBadRequest)
			log.Error("Invalid media uploaded", zap.Error(errMedia))
		}

		asset, errAsset := store.NewAsset(content, app.conf.S3.BucketMedia, "")
		if errAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save asset"))

			return
		}

		if errPut := app.assetStore.Put(ctx, app.conf.S3.BucketMedia, asset.Name, bytes.NewReader(content), asset.Size, asset.MimeType); errPut != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save media"))

			log.Error("Failed to save user media to s3 backend", zap.Error(errPut))

			return
		}

		if errSaveAsset := app.db.SaveAsset(ctx, &asset); errSaveAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save asset"))

			log.Error("Failed to save user asset to s3 backend", zap.Error(errSaveAsset))
		}

		media.Asset = asset

		media.Contents = nil

		if !fp.Contains(MediaSafeMimeTypesImages, media.MimeType) {
			responseErr(ctx, http.StatusBadRequest, errors.New("Invalid image format"))
			log.Error("User tried uploading image with forbidden mimetype",
				zap.String("mime", media.MimeType), zap.String("name", media.Name))

			return
		}

		if errSave := app.db.SaveMedia(ctx, &media); errSave != nil {
			log.Error("Failed to save wiki media", zap.Error(errSave))

			if errors.Is(store.Err(errSave), store.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, errors.New("Duplicate media name"))

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save media"))

			return
		}

		ctx.JSON(http.StatusCreated, media)
	}
}

type AuthorMessage struct {
	Author  store.Person      `json:"author"`
	Message store.UserMessage `json:"message"`
}

func onAPIGetReportMessages(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportID, errParam := getInt64Param(ctx, "report_id")
		if errParam != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var report store.Report
		if errGetReport := app.db.GetReport(ctx, reportID, &report); errGetReport != nil {
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		if !checkPrivilege(ctx, currentUserProfile(ctx), steamid.Collection{report.SourceID, report.TargetID}, consts.PModerator) {
			return
		}

		reportMessages, errGetReportMessages := app.db.GetReportMessages(ctx, reportID)
		if errGetReportMessages != nil {
			responseErr(ctx, http.StatusNotFound, consts.ErrPlayerNotFound)

			return
		}

		var ids steamid.Collection
		for _, msg := range reportMessages {
			ids = append(ids, msg.AuthorID)
		}

		authors, authorsErr := app.db.GetPeopleBySteamID(ctx, ids)
		if authorsErr != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var (
			authorsMap     = authors.AsMap()
			authorMessages []AuthorMessage
		)

		for _, message := range reportMessages {
			authorMessages = append(authorMessages, AuthorMessage{
				Author:  authorsMap[message.AuthorID],
				Message: message,
			})
		}

		ctx.JSON(http.StatusOK, authorMessages)
	}
}

func onAPIPostReportMessage(app *App) gin.HandlerFunc {
	type newMessage struct {
		Message string `json:"message"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportID, errID := getInt64Param(ctx, "report_id")
		if errID != nil || reportID == 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req newMessage
		if !bind(ctx, log, &req) {
			return
		}

		if req.Message == "" {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var report store.Report
		if errReport := app.db.GetReport(ctx, reportID, &report); errReport != nil {
			if errors.Is(store.Err(errReport), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to load report", zap.Error(errReport))

			return
		}

		person := currentUserProfile(ctx)
		msg := store.NewUserMessage(reportID, person.SteamID, req.Message)

		if errSave := app.db.SaveReportMessage(ctx, &msg); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		report.UpdatedOn = time.Now()

		if errSave := app.db.SaveReport(ctx, &report); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to update report activity", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, msg)

		msgEmbed := discord.
			NewEmbed("New report message posted").
			SetDescription(msg.Contents).
			SetColor(app.bot.Colour.Success).
			SetURL(app.ExtURL(report))

		app.addAuthorUserProfile(msgEmbed, person).Truncate()

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.LogChannelID,
			Embed:     msgEmbed.MessageEmbed,
		})
	}
}

func onAPIEditReportMessage(app *App) gin.HandlerFunc {
	type editMessage struct {
		BodyMD string `json:"body_md"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportMessageID, errID := getInt64Param(ctx, "report_message_id")
		if errID != nil || reportMessageID == 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var existing store.UserMessage
		if errExist := app.db.GetReportMessageByID(ctx, reportMessageID, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrPlayerNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		curUser := currentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, consts.PModerator) {
			return
		}

		var req editMessage
		if !bind(ctx, log, &req) {
			return
		}

		if req.BodyMD == "" {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if req.BodyMD == existing.Contents {
			responseErr(ctx, http.StatusConflict, consts.ErrDuplicate)

			return
		}

		existing.Contents = req.BodyMD
		if errSave := app.db.SaveReportMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, req)

		msgEmbed := discord.
			NewEmbed("New report message edited").
			SetDescription(req.BodyMD).
			SetColor(app.bot.Colour.Warn).
			AddField("Old Message", existing.Contents).
			SetURL(app.ExtURLRaw("/report/%d", existing.ParentID))

		app.addAuthorUserProfile(msgEmbed, curUser).Truncate()

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.LogChannelID,
			Embed:     msgEmbed.MessageEmbed,
		})
	}
}

func onAPIDeleteReportMessage(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportMessageID, errID := getInt64Param(ctx, "report_message_id")
		if errID != nil || reportMessageID == 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var existing store.UserMessage
		if errExist := app.db.GetReportMessageByID(ctx, reportMessageID, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		curUser := currentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, consts.PModerator) {
			return
		}

		existing.Deleted = true
		if errSave := app.db.SaveReportMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save report message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusNoContent, nil)

		msgEmbed := discord.
			NewEmbed("User report message deleted").
			SetDescription(existing.Contents).
			SetColor(app.bot.Colour.Warn)

		app.addAuthorUserProfile(msgEmbed, curUser).
			Truncate()

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.LogChannelID,
			Embed:     msgEmbed.MessageEmbed,
		})
	}
}

func onAPIGetBanByID(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		curUser := currentUserProfile(ctx)

		banID, errID := getInt64Param(ctx, "ban_id")
		if errID != nil || banID == 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

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

		bannedPerson := store.NewBannedPerson()
		if errGetBan := app.db.GetBanByBanID(ctx, banID, &bannedPerson, deletedOk); errGetBan != nil {
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)
			log.Error("Failed to fetch steam ban", zap.Error(errGetBan))

			return
		}

		if !checkPrivilege(ctx, curUser, steamid.Collection{bannedPerson.TargetID}, consts.PModerator) {
			return
		}

		loadBanMeta(&bannedPerson)
		ctx.JSON(http.StatusOK, bannedPerson)
	}
}

type AuthorBanMessage struct {
	Author  store.Person      `json:"author"`
	Message store.UserMessage `json:"message"`
}

func onAPIGetBanMessages(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		banID, errParam := getInt64Param(ctx, "ban_id")
		if errParam != nil {
			responseErr(ctx, http.StatusNotFound, consts.ErrInvalidParameter)

			return
		}

		banPerson := store.NewBannedPerson()
		if errGetBan := app.db.GetBanByBanID(ctx, banID, &banPerson, true); errGetBan != nil {
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		if !checkPrivilege(ctx, currentUserProfile(ctx), steamid.Collection{banPerson.TargetID, banPerson.SourceID}, consts.PModerator) {
			return
		}

		banMessages, errGetBanMessages := app.db.GetBanMessages(ctx, banID)
		if errGetBanMessages != nil {
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		var ids steamid.Collection
		for _, msg := range banMessages {
			ids = append(ids, msg.AuthorID)
		}

		authors, authorsErr := app.db.GetPeopleBySteamID(ctx, ids)
		if authorsErr != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		authorsMap := authors.AsMap()

		var authorMessages []AuthorBanMessage
		for _, message := range banMessages {
			authorMessages = append(authorMessages, AuthorBanMessage{
				Author:  authorsMap[message.AuthorID],
				Message: message,
			})
		}

		ctx.JSON(http.StatusOK, authorMessages)
	}
}

func onAPIPostBanMessage(app *App) gin.HandlerFunc {
	type newMessage struct {
		Message string `json:"message"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banID, errID := getInt64Param(ctx, "ban_id")
		if errID != nil || banID == 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req newMessage
		if !bind(ctx, log, &req) {
			return
		}

		if req.Message == "" {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		bannedPerson := store.NewBannedPerson()
		if errReport := app.db.GetBanByBanID(ctx, banID, &bannedPerson, true); errReport != nil {
			if errors.Is(store.Err(errReport), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to load ban", zap.Error(errReport))

			return
		}

		curUserProfile := currentUserProfile(ctx)
		if bannedPerson.AppealState != store.Open && curUserProfile.PermissionLevel < consts.PModerator {
			responseErr(ctx, http.StatusForbidden, consts.ErrPermissionDenied)
			log.Warn("User tried to bypass posting restriction",
				zap.Int64("ban_id", bannedPerson.BanID), zap.Int64("target_id", bannedPerson.TargetID.Int64()))

			return
		}

		msg := store.NewUserMessage(banID, curUserProfile.SteamID, req.Message)
		if errSave := app.db.SaveBanMessage(ctx, &msg); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save ban appeal message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, msg)

		msgEmbed := discord.
			NewEmbed("New ban appeal message posted").
			SetColor(app.bot.Colour.Info).
			// SetThumbnail(bannedPerson.TargetAvatarhash).
			SetDescription(msg.Contents).
			SetURL(app.ExtURL(bannedPerson.BanSteam))

		app.addAuthorUserProfile(msgEmbed, curUserProfile).Truncate()

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.LogChannelID,
			Embed:     msgEmbed.MessageEmbed,
		})
	}
}

func onAPIEditBanMessage(app *App) gin.HandlerFunc {
	type editMessage struct {
		BodyMD string `json:"body_md"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		reportMessageID, errID := getIntParam(ctx, "ban_message_id")
		if errID != nil || reportMessageID == 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var existing store.UserMessage
		if errExist := app.db.GetBanMessageByID(ctx, reportMessageID, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		curUser := currentUserProfile(ctx)

		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, consts.PModerator) {
			return
		}

		var req editMessage
		if !bind(ctx, log, &req) {
			return
		}

		if req.BodyMD == "" {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if req.BodyMD == existing.Contents {
			responseErr(ctx, http.StatusConflict, consts.ErrDuplicate)

			return
		}

		existing.Contents = req.BodyMD
		if errSave := app.db.SaveBanMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save ban appeal message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, req)

		msgEmbed := discord.
			NewEmbed("Ban appeal message edited").
			SetDescription(util.DiffString(existing.Contents, req.BodyMD)).
			SetColor(app.bot.Colour.Warn)

		app.addAuthorUserProfile(msgEmbed, curUser).Truncate()

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.LogChannelID,
			Embed:     msgEmbed.MessageEmbed,
		})
	}
}

func onAPIDeleteBanMessage(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banMessageID, errID := getIntParam(ctx, "ban_message_id")
		if errID != nil || banMessageID == 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var existing store.UserMessage
		if errExist := app.db.GetBanMessageByID(ctx, banMessageID, &existing); errExist != nil {
			if errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		curUser := currentUserProfile(ctx)
		if !checkPrivilege(ctx, curUser, steamid.Collection{existing.AuthorID}, consts.PModerator) {
			return
		}

		existing.Deleted = true
		if errSave := app.db.SaveBanMessage(ctx, &existing); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save appeal message", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusNoContent, nil)

		msgEmbed := discord.
			NewEmbed("User appeal message deleted").
			SetDescription(existing.Contents)

		app.addAuthorUserProfile(msgEmbed, curUser).Truncate()

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.LogChannelID,
			Embed:     msgEmbed.MessageEmbed,
		})
	}
}

func onAPIGetSourceBans(_ *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		steamID, errID := getSID64Param(ctx, "steam_id")
		if errID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		records, errRecords := getSourceBans(ctx, steamID)
		if errRecords != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, records)
	}
}

func onAPIGetMatch(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		matchID, errID := getUUIDParam(ctx, "match_id")
		if errID != nil {
			log.Error("Invalid match_id value", zap.Error(errID))
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var match store.MatchResult

		errMatch := app.db.MatchGetByID(ctx, matchID, &match)

		if errMatch != nil {
			if errors.Is(errMatch, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, match)
	}
}

func onAPIGetMatches(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.MatchesQueryOpts
		if !bind(ctx, log, &req) {
			return
		}

		// Don't let normal users query anybody but themselves
		user := currentUserProfile(ctx)
		if user.PermissionLevel <= consts.PUser {
			if !req.SteamID.Valid() {
				responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

				return
			}

			if user.SteamID != req.SteamID {
				responseErr(ctx, http.StatusForbidden, consts.ErrPermissionDenied)

				return
			}
		}

		matches, totalCount, matchesErr := app.db.Matches(ctx, req)
		if matchesErr != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to perform query", zap.Error(matchesErr))

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(totalCount, matches))
	}
}

func onAPIQueryMessages(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.ChatHistoryQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		if req.Limit <= 0 || req.Limit > 1000 {
			req.Limit = 50
		}

		user := currentUserProfile(ctx)

		if user.PermissionLevel <= consts.PUser {
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

		messages, count, errChat := app.db.QueryChatHistory(ctx, req)
		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			log.Error("Failed to query messages history",
				zap.Error(errChat), zap.String("sid", string(req.SourceID)))
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, messages))
	}
}

func onAPIGetStatsWeaponsOverall(ctx context.Context, app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	updater := store.NewDataUpdater(log, time.Minute*10, func() ([]store.WeaponsOverallResult, error) {
		weaponStats, errUpdate := app.db.WeaponsOverall(ctx)
		if errUpdate != nil && !errors.Is(errUpdate, store.ErrNoResult) {
			return nil, errors.Wrap(errUpdate, "Failed to update weapon stats")
		}

		if weaponStats == nil {
			weaponStats = []store.WeaponsOverallResult{}
		}

		return weaponStats, nil
	})

	go updater.Start(ctx)

	return func(ctx *gin.Context) {
		stats := updater.Data()

		ctx.JSON(http.StatusOK, newLazyResult(int64(len(stats)), stats))
	}
}

func onAPIGetsStatsWeapon(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type resp struct {
		LazyResult
		Weapon store.Weapon `json:"weapon"`
	}

	return func(ctx *gin.Context) {
		weaponID, errWeaponID := getIntParam(ctx, "weapon_id")
		if errWeaponID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var weapon store.Weapon

		errWeapon := app.db.GetWeaponByID(ctx, weaponID, &weapon)

		if errWeapon != nil {
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		weaponStats, errChat := app.db.WeaponsOverallTopPlayers(ctx, weaponID)
		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			log.Error("Failed to get weapons overall top stats",
				zap.Error(errChat))
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if weaponStats == nil {
			weaponStats = []store.PlayerWeaponResult{}
		}

		ctx.JSON(http.StatusOK, resp{LazyResult: newLazyResult(int64(len(weaponStats)), weaponStats), Weapon: weapon})
	}
}

func onAPIGetStatsPlayersOverall(ctx context.Context, app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	updater := store.NewDataUpdater(log, time.Minute*10, func() ([]store.PlayerWeaponResult, error) {
		updatedStats, errChat := app.db.PlayersOverallByKills(ctx, 1000)
		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			return nil, errors.Wrap(errChat, "Failed to query overall players overall")
		}

		return updatedStats, nil
	})

	go updater.Start(ctx)

	return func(ctx *gin.Context) {
		stats := updater.Data()
		ctx.JSON(http.StatusOK, newLazyResult(int64(len(stats)), stats))
	}
}

func onAPIGetStatsHealersOverall(ctx context.Context, app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	updater := store.NewDataUpdater(log, time.Minute*10, func() ([]store.HealingOverallResult, error) {
		updatedStats, errChat := app.db.HealersOverallByHealing(ctx, 250)
		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			return nil, errors.Wrap(errChat, "Failed to query overall healers overall")
		}

		return updatedStats, nil
	})

	go updater.Start(ctx)

	return func(ctx *gin.Context) {
		stats := updater.Data()
		ctx.JSON(http.StatusOK, newLazyResult(int64(len(stats)), stats))
	}
}

func onAPIGetPlayerWeaponStatsOverall(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		steamID, errSteamID := getSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		weaponStats, errChat := app.db.WeaponsOverallByPlayer(ctx, steamID)
		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			log.Error("Failed to query player weapons stats",
				zap.Error(errChat))
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if weaponStats == nil {
			weaponStats = []store.WeaponsOverallResult{}
		}

		ctx.JSON(http.StatusOK, newLazyResult(int64(len(weaponStats)), weaponStats))
	}
}

func onAPIGetPlayerClassStatsOverall(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		steamID, errSteamID := getSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		classStats, errChat := app.db.PlayerOverallClassStats(ctx, steamID)
		if errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			log.Error("Failed to query player class stats",
				zap.Error(errChat))
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if classStats == nil {
			classStats = []store.PlayerClassOverallResult{}
		}

		ctx.JSON(http.StatusOK, newLazyResult(int64(len(classStats)), classStats))
	}
}

func onAPIGetPlayerStatsOverall(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		steamID, errSteamID := getSID64Param(ctx, "steam_id")
		if errSteamID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var por store.PlayerOverallResult
		if errChat := app.db.PlayerOverallStats(ctx, steamID, &por); errChat != nil && !errors.Is(errChat, store.ErrNoResult) {
			log.Error("Failed to query player stats overall",
				zap.Error(errChat))
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, por)
	}
}

func onAPISaveContestEntryMedia(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		contest, success := contestFromCtx(ctx, app)
		if !success {
			return
		}

		var req UserUploadedFile
		if !bind(ctx, log, &req) {
			return
		}

		content, decodeErr := base64.StdEncoding.DecodeString(req.Content)
		if decodeErr != nil {
			ctx.JSON(http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		media, errMedia := store.NewMedia(currentUserProfile(ctx).SteamID, req.Name, req.Mime, content)
		if errMedia != nil {
			ctx.JSON(http.StatusBadRequest, consts.ErrBadRequest)
			log.Error("Invalid media uploaded", zap.Error(errMedia))
		}

		asset, errAsset := store.NewAsset(content, app.conf.S3.BucketMedia, "")
		if errAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save asset"))

			return
		}

		if errPut := app.assetStore.Put(ctx, app.conf.S3.BucketMedia, asset.Name, bytes.NewReader(content), asset.Size, asset.MimeType); errPut != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save user contest media"))

			log.Error("Failed to save user contest entry media to s3 backend", zap.Error(errPut))

			return
		}

		if errSaveAsset := app.db.SaveAsset(ctx, &asset); errSaveAsset != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save asset"))

			log.Error("Failed to save user asset to s3 backend", zap.Error(errSaveAsset))
		}

		media.Asset = asset

		media.Contents = nil

		if !contest.MimeTypeAcceptable(media.MimeType) {
			responseErr(ctx, http.StatusUnsupportedMediaType, errors.New("Invalid file format"))
			log.Error("User tried uploading file with forbidden mimetype",
				zap.String("mime", media.MimeType), zap.String("name", media.Name))

			return
		}

		if errSave := app.db.SaveMedia(ctx, &media); errSave != nil {
			log.Error("Failed to save user contest media", zap.Error(errSave))

			if errors.Is(store.Err(errSave), store.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, errors.New("Duplicate media name"))

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save user contest media"))

			return
		}

		ctx.JSON(http.StatusCreated, media)
	}
}

func onAPISaveContestEntryVote(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type voteResult struct {
		CurrentVote string `json:"current_vote"`
	}

	return func(ctx *gin.Context) {
		contest, success := contestFromCtx(ctx, app)
		if !success {
			return
		}

		contestEntryID, errContestEntryID := getUUIDParam(ctx, "contest_entry_id")
		if errContestEntryID != nil {
			ctx.JSON(http.StatusNotFound, consts.ErrNotFound)
			log.Error("Invalid contest entry id option")

			return
		}

		direction := strings.ToLower(ctx.Param("direction"))
		if direction != "up" && direction != "down" {
			ctx.JSON(http.StatusBadRequest, consts.ErrBadRequest)
			log.Error("Invalid vote direction option")

			return
		}

		if !contest.Voting || !contest.DownVotes && direction != "down" {
			ctx.JSON(http.StatusBadRequest, consts.ErrBadRequest)
			log.Error("Voting not enabled")

			return
		}

		currentUser := currentUserProfile(ctx)

		if errVote := app.db.ContestEntryVote(ctx, contestEntryID, currentUser.SteamID, direction == "up"); errVote != nil {
			if errors.Is(errVote, store.ErrVoteDeleted) {
				ctx.JSON(http.StatusOK, voteResult{""})

				return
			}

			ctx.JSON(http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, voteResult{direction})
	}
}

func onAPISaveContestEntrySubmit(app *App) gin.HandlerFunc {
	type entryReq struct {
		Description string    `json:"description"`
		AssetID     uuid.UUID `json:"asset_id"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)
		contest, success := contestFromCtx(ctx, app)

		if !success {
			return
		}

		var req entryReq
		if !bind(ctx, log, &req) {
			return
		}

		if contest.MediaTypes != "" {
			var media store.Media
			if errMedia := app.db.GetMediaByAssetID(ctx, req.AssetID, &media); errMedia != nil {
				responseErr(ctx, http.StatusFailedDependency, errors.New("Could not load media asset"))

				return
			}

			if !contest.MimeTypeAcceptable(media.MimeType) {
				responseErr(ctx, http.StatusFailedDependency, errors.New("Invalid Mime Type"))

				return
			}
		}

		existingEntries, errEntries := app.db.ContestEntries(ctx, contest.ContestID)
		if errEntries != nil && !errors.Is(errEntries, store.ErrNoResult) {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not load existing contest entries"))

			return
		}

		own := 0

		for _, entry := range existingEntries {
			if entry.SteamID == user.SteamID {
				own++
			}

			if own >= contest.MaxSubmissions {
				responseErr(ctx, http.StatusForbidden, errors.New("Current entries count exceed max_submissions"))

				return
			}
		}

		steamID := currentUserProfile(ctx).SteamID

		entry, errEntry := contest.NewEntry(steamID, req.AssetID, req.Description)
		if errEntry != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not create content entry"))

			return
		}

		if errSave := app.db.ContestEntrySave(ctx, entry); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errors.New("Could not save entry"))

			return
		}

		ctx.JSON(http.StatusCreated, entry)

		log.Info("New contest entry submitted", zap.String("contest_id", contest.ContestID.String()))
	}
}

func onAPIDeleteContestEntry(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)

		contestEntryID, idErr := getUUIDParam(ctx, "contest_entry_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var entry store.ContestEntry

		if errContest := app.db.ContestEntry(ctx, contestEntryID, &entry); errContest != nil {
			if errors.Is(errContest, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrUnknownID)

				return
			}

			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			log.Error("Error getting contest entry for deletion", zap.Error(errContest))

			return
		}

		// Only >=moderators or the entry author are allowed to delete entries.
		if !(user.PermissionLevel >= consts.PModerator || user.SteamID == entry.SteamID) {
			responseErr(ctx, http.StatusForbidden, consts.ErrPermissionDenied)

			return
		}

		var contest store.Contest

		if errContest := app.db.ContestByID(ctx, entry.ContestID, &contest); errContest != nil {
			if errors.Is(errContest, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrUnknownID)

				return
			}

			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			log.Error("Error getting contest", zap.Error(errContest))

			return
		}

		// Only allow mods to delete entries from expired contests.
		if user.SteamID == entry.SteamID && time.Since(contest.DateEnd) > 0 {
			responseErr(ctx, http.StatusForbidden, consts.ErrPermissionDenied)

			log.Error("User tried to delete entry from expired contest")

			return
		}

		if errDelete := app.db.ContestEntryDelete(ctx, entry.ContestEntryID); errDelete != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

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

func onAPIThreadCreate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type CreateThreadRequest struct {
		Title  string `json:"title"`
		BodyMD string `json:"body_md"`
		Sticky bool   `json:"sticky"`
		Locked bool   `json:"locked"`
	}

	type ThreadWithMessage struct {
		store.ForumThread
		Message store.ForumMessage `json:"message"`
	}

	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)

		app.touchPerson(user)

		forumID, errForumID := getIntParam(ctx, "forum_id")
		if errForumID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var req CreateThreadRequest
		if !bind(ctx, log, &req) {
			return
		}

		if len(req.BodyMD) <= 1 {
			responseErr(ctx, http.StatusBadRequest, errors.New("Message too short"))

			return
		}

		if len(req.Title) <= 4 {
			responseErr(ctx, http.StatusBadRequest, errors.New("Title too short"))

			return
		}

		var forum store.Forum
		if errForum := app.db.Forum(ctx, forumID, &forum); errForum != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		thread := forum.NewThread(req.Title, user.SteamID)
		thread.Sticky = req.Sticky
		thread.Locked = req.Locked

		if errSaveThread := app.db.ForumThreadSave(ctx, &thread); errSaveThread != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Failed to save new thread", zap.Error(errSaveThread))

			return
		}

		message := thread.NewMessage(user.SteamID, req.BodyMD)

		if errSaveMessage := app.db.ForumMessageSave(ctx, &message); errSaveMessage != nil {
			// Drop created thread.
			// TODO transaction
			if errRollback := app.db.ForumThreadDelete(ctx, thread.ForumThreadID); errRollback != nil {
				log.Error("Failed to rollback new thread", zap.Error(errRollback))
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Failed to save new forum message", zap.Error(errSaveMessage))

			return
		}

		if errIncr := app.db.ForumIncrMessageCount(ctx, forum.ForumID, true); errIncr != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

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

func onAPIThreadUpdate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type threadUpdate struct {
		Title  string `json:"title"`
		Sticky bool   `json:"sticky"`
		Locked bool   `json:"locked"`
	}

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		forumThreadID, errForumTheadID := getInt64Param(ctx, "forum_thread_id")
		if errForumTheadID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var req threadUpdate
		if !bind(ctx, log, &req) {
			return
		}

		req.Title = util.SanitizeUGC(req.Title)

		if len(req.Title) < 2 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var thread store.ForumThread
		if errGet := app.db.ForumThread(ctx, forumThreadID, &thread); errGet != nil {
			if errors.Is(errGet, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)
			} else {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			}

			return
		}

		if thread.SourceID != currentUser.SteamID && !(currentUser.PermissionLevel >= consts.PModerator) {
			responseErr(ctx, http.StatusForbidden, consts.ErrInternal)

			return
		}

		thread.Title = req.Title
		thread.Sticky = req.Sticky
		thread.Locked = req.Locked

		if errDelete := app.db.ForumThreadSave(ctx, &thread); errDelete != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to update thread", zap.Error(errDelete))

			return
		}

		ctx.JSON(http.StatusOK, thread)
		log.Info("Thread updated", zap.Int64("forum_thread_id", thread.ForumThreadID))
	}
}

func onAPIThreadDelete(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		forumThreadID, errForumTheadID := getInt64Param(ctx, "forum_thread_id")
		if errForumTheadID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var thread store.ForumThread
		if errGet := app.db.ForumThread(ctx, forumThreadID, &thread); errGet != nil {
			if errors.Is(errGet, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)
			} else {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			}

			return
		}

		if errDelete := app.db.ForumThreadDelete(ctx, thread.ForumThreadID); errDelete != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to delete thread", zap.Error(errDelete))

			return
		}

		var forum store.Forum
		if errForum := app.db.Forum(ctx, thread.ForumID, &forum); errForum != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to load forum", zap.Error(errForum))

			return
		}

		forum.CountThreads -= 1

		if errSave := app.db.ForumSave(ctx, &forum); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save thread count", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func onAPIThreadMessageUpdate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type MessageUpdate struct {
		BodyMD string `json:"body_md"`
	}

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		app.touchPerson(currentUser)

		forumMessageID, errForumMessageID := getInt64Param(ctx, "forum_message_id")
		if errForumMessageID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var req MessageUpdate
		if !bind(ctx, log, &req) {
			return
		}

		var message store.ForumMessage
		if errMessage := app.db.ForumMessage(ctx, forumMessageID, &message); errMessage != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if message.SourceID != currentUser.SteamID && !(currentUser.PermissionLevel >= consts.PModerator) {
			responseErr(ctx, http.StatusForbidden, consts.ErrInternal)

			return
		}

		req.BodyMD = util.SanitizeUGC(req.BodyMD)

		if len(req.BodyMD) < 10 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		message.BodyMD = req.BodyMD

		if errSave := app.db.ForumMessageSave(ctx, &message); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, message)
	}
}

func onAPIMessageDelete(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		forumMessageID, errForumMessageID := getInt64Param(ctx, "forum_message_id")
		if errForumMessageID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var message store.ForumMessage
		if err := app.db.ForumMessage(ctx, forumMessageID, &message); err != nil {
			if errors.Is(err, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)
			} else {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			}

			return
		}

		var thread store.ForumThread
		if err := app.db.ForumThread(ctx, message.ForumThreadID, &thread); err != nil {
			if errors.Is(err, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)
			} else {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			}

			return
		}

		if thread.Locked {
			responseErr(ctx, http.StatusForbidden, errors.New("Locked thread"))

			return
		}

		messages, count, errMessage := app.db.ForumMessages(ctx, store.ThreadMessagesQueryFilter{ForumThreadID: message.ForumThreadID})
		if errMessage != nil || count <= 0 {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		isThreadParent := messages[0].ForumMessageID == message.ForumMessageID

		if isThreadParent {
			if err := app.db.ForumThreadDelete(ctx, message.ForumThreadID); err != nil {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
				log.Error("Failed to delete forum thread", zap.Error(err))

				return
			}

			// Delete the thread if it's the first message
			var forum store.Forum
			if errForum := app.db.Forum(ctx, thread.ForumID, &forum); errForum != nil {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
				log.Error("Failed to load forum", zap.Error(errForum))

				return
			}

			forum.CountThreads -= 1

			if errSave := app.db.ForumSave(ctx, &forum); errSave != nil {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
				log.Error("Failed to save thread count", zap.Error(errSave))

				return
			}

			log.Error("Thread deleted due to parent deletion", zap.Int64("forum_thread_id", thread.ForumThreadID))
		} else {
			if errDelete := app.db.ForumMessageDelete(ctx, message.ForumMessageID); errDelete != nil {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
				log.Error("Failed to delete message", zap.Error(errDelete))

				return
			}

			log.Info("Thread message deleted", zap.Int64("forum_message_id", message.ForumMessageID))
		}

		ctx.JSON(http.StatusOK, gin.H{})
	}
}

func onAPIThreadCreateReply(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type ThreadReply struct {
		BodyMD string `json:"body_md"`
	}

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		app.touchPerson(currentUser)

		forumThreadID, errForumID := getInt64Param(ctx, "forum_thread_id")
		if errForumID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var thread store.ForumThread
		if errThread := app.db.ForumThread(ctx, forumThreadID, &thread); errThread != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if thread.Locked && currentUser.PermissionLevel < consts.PEditor {
			responseErr(ctx, http.StatusForbidden, errors.New("Cannot reply to locked threads"))

			return
		}

		var req ThreadReply
		if !bind(ctx, log, &req) {
			return
		}

		req.BodyMD = util.SanitizeUGC(req.BodyMD)

		if len(req.BodyMD) < 3 {
			responseErr(ctx, http.StatusBadRequest, errors.New("Body too short"))

			return
		}

		newMessage := thread.NewMessage(currentUser.SteamID, req.BodyMD)
		if errSave := app.db.ForumMessageSave(ctx, &newMessage); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var message store.ForumMessage
		if errFetch := app.db.ForumMessage(ctx, newMessage.ForumMessageID, &message); errFetch != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if errIncr := app.db.ForumIncrMessageCount(ctx, thread.ForumID, true); errIncr != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Failed to increment message count", zap.Error(errIncr))
		}

		newMessage.Personaname = currentUser.Name
		newMessage.Avatarhash = currentUser.Avatarhash
		newMessage.PermissionLevel = currentUser.PermissionLevel
		newMessage.Online = true

		ctx.JSON(http.StatusCreated, newMessage)
	}
}
