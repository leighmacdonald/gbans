package app

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func onAPISaveWikiSlug(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req wiki.Page
		if !bind(ctx, log, &req) {
			return
		}

		if req.Slug == "" || req.BodyMD == "" {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var page wiki.Page
		if errGetWikiSlug := app.db.GetWikiPageBySlug(ctx, req.Slug, &page); errGetWikiSlug != nil {
			if errors.Is(errGetWikiSlug, store.ErrNoResult) {
				page.CreatedOn = time.Now()
				page.Revision += 1
				page.Slug = req.Slug
			} else {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

				return
			}
		} else {
			page = page.NewRevision()
		}

		page.BodyMD = req.BodyMD
		if errSave := app.db.SaveWikiPage(ctx, &page); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, page)
	}
}

func onAPIPostNewsCreate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.NewsEntry
		if !bind(ctx, log, &req) {
			return
		}

		if errSave := app.db.SaveNewsArticle(ctx, &req); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, req)

		go app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.LogChannelID,
			Embed: discord.
				NewEmbed("News Created").
				SetDescription(req.BodyMD).
				AddField("Title", req.Title).MessageEmbed,
		})
	}
}

func onAPIPostNewsUpdate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		newsID, errID := getIntParam(ctx, "news_id")
		if errID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var entry store.NewsEntry
		if errGet := app.db.GetNewsByID(ctx, newsID, &entry); errGet != nil {
			if errors.Is(store.Err(errGet), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if !bind(ctx, log, &entry) {
			return
		}

		if errSave := app.db.SaveNewsArticle(ctx, &entry); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusAccepted, entry)

		app.bot.SendPayload(discord.Payload{
			ChannelID: app.conf.Discord.LogChannelID,
			Embed: discord.
				NewEmbed("News Updated").
				AddField("Title", entry.Title).
				SetDescription(entry.BodyMD).
				MessageEmbed,
		})
	}
}

func onAPIGetNewsAll(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := app.db.GetNewsLatest(ctx, 100, true)
		if errGetNewsLatest != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newsLatest)
	}
}

func onAPIQueryWordFilters(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var opts store.FiltersQueryFilter
		if !bind(ctx, log, &opts) {
			return
		}

		words, count, errGetFilters := app.db.GetFilters(ctx, opts)
		if errGetFilters != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, words))
	}
}

func onAPIPostWordFilter(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.Filter
		if !bind(ctx, log, &req) {
			return
		}

		if req.Pattern == "" {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if req.IsRegex {
			_, compErr := regexp.Compile(req.Pattern)
			if compErr != nil {
				responseErr(ctx, http.StatusBadRequest, errors.New("invalid regex"))

				return
			}
		}

		now := time.Now()

		if req.FilterID > 0 {
			var existingFilter store.Filter
			if errGet := app.db.GetFilterByID(ctx, req.FilterID, &existingFilter); errGet != nil {
				if errors.Is(errGet, store.ErrNoResult) {
					responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

					return
				}

				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

				return
			}

			existingFilter.UpdatedOn = now
			existingFilter.Pattern = req.Pattern
			existingFilter.IsRegex = req.IsRegex
			existingFilter.IsEnabled = req.IsEnabled

			if errSave := app.FilterAdd(ctx, &existingFilter); errSave != nil {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

				return
			}

			req = existingFilter
		} else {
			profile := currentUserProfile(ctx)
			newFilter := store.Filter{
				AuthorID:  profile.SteamID,
				Pattern:   req.Pattern,
				CreatedOn: now,
				UpdatedOn: now,
				IsRegex:   req.IsRegex,
				IsEnabled: req.IsEnabled,
			}

			if errSave := app.FilterAdd(ctx, &newFilter); errSave != nil {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

				return
			}

			req = newFilter
		}

		ctx.JSON(http.StatusOK, req)
	}
}

func onAPIDeleteWordFilter(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		wordID, wordIDErr := getInt64Param(ctx, "word_id")
		if wordIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var filter store.Filter
		if errGet := app.db.GetFilterByID(ctx, wordID, &filter); errGet != nil {
			if errors.Is(errGet, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if errDrop := app.db.DropFilter(ctx, &filter); errDrop != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusNoContent, nil)
	}
}

func onAPIPostWordMatch(app *App) gin.HandlerFunc {
	type matchRequest struct {
		Query string
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req matchRequest
		if !bind(ctx, log, &req) {
			return
		}

		words, _, errGetFilters := app.db.GetFilters(ctx, store.FiltersQueryFilter{})
		if errGetFilters != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var matches []store.Filter

		for _, filter := range words {
			if filter.Match(req.Query) {
				matches = append(matches, filter)
			}
		}

		ctx.JSON(http.StatusOK, matches)
	}
}

func onAPIExportBansValveIP(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		bans, _, errBans := app.db.GetBansNet(ctx, store.CIDRBansQueryFilter{})
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var entries []string

		for _, ban := range bans {
			if ban.Deleted ||
				!ban.IsEnabled {
				continue
			}
			// TODO Shouldn't be cidr?
			entries = append(entries, fmt.Sprintf("addip 0 %s", ban.CIDR))
		}

		ctx.Data(http.StatusOK, "text/plain", []byte(strings.Join(entries, "\n")))
	}
}

func onAPISearchPlayers(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var query store.PlayerQuery
		if !bind(ctx, log, &query) {
			return
		}

		people, count, errGetPeople := app.db.GetPeople(ctx, query)
		if errGetPeople != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, people))
	}
}

func onAPIPostBanState(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportID, errID := getInt64Param(ctx, "report_id")
		if errID != nil || reportID <= 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var report store.Report
		if errReport := app.db.GetReport(ctx, reportID, &report); errReport != nil {
			if errors.Is(errReport, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		go app.bot.SendPayload(discord.Payload{ChannelID: app.conf.Discord.LogChannelID, Embed: nil})
	}
}

type apiUnbanRequest struct {
	UnbanReasonText string `json:"unban_reason_text"`
}

func onAPIQueryPersonConnections(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.ConnectionHistoryQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		ipHist, totalCount, errIPHist := app.db.QueryConnectionHistory(ctx, req)
		if errIPHist != nil && !errors.Is(errIPHist, store.ErrNoResult) {
			log.Error("Failed to query connection history", zap.Error(errIPHist))
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(totalCount, ipHist))
	}
}

func onAPIQueryMessageContext(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		messageID, errMessageID := getInt64Param(ctx, "person_message_id")
		if errMessageID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)
			log.Debug("Got invalid person_message_id", zap.Error(errMessageID))

			return
		}

		padding, errPadding := getIntParam(ctx, "padding")
		if errPadding != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)
			log.Debug("Got invalid padding", zap.Error(errPadding))

			return
		}

		var msg store.QueryChatHistoryResult
		if errMsg := app.db.GetPersonMessage(ctx, messageID, &msg); errMsg != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		messages, errQuery := app.db.GetPersonMessageContext(ctx, msg.ServerID, messageID, padding)
		if errQuery != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, messages)
	}
}

func onAPIGetAppeals(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.AppealQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		bans, total, errBans := app.db.GetAppealsByActivity(ctx, req)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to fetch appeals", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(total, bans))
	}
}

func onAPIGetBansSteam(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.SteamBansQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		bans, count, errBans := app.db.GetBansSteam(ctx, req)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to fetch steam bans", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, bans))
	}
}

func onAPIPostBanDelete(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banID, banIDErr := getInt64Param(ctx, "ban_id")
		if banIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req apiUnbanRequest
		if !bind(ctx, log, &req) {
			return
		}

		bannedPerson := store.NewBannedPerson()
		if banErr := app.db.GetBanByBanID(ctx, banID, &bannedPerson, false); banErr != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		changed, errSave := app.Unban(ctx, bannedPerson.TargetID, req.UnbanReasonText)
		if errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if !changed {
			responseErr(ctx, http.StatusNotFound, errors.New("Failed to save unban"))

			return
		}

		ctx.JSON(http.StatusAccepted, gin.H{})
	}
}

func onAPIPostBanUpdate(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type updateBanRequest struct {
		TargetID       store.StringSID `json:"target_id"`
		BanType        store.BanType   `json:"ban_type"`
		Reason         store.Reason    `json:"reason"`
		ReasonText     string          `json:"reason_text"`
		Note           string          `json:"note"`
		IncludeFriends bool            `json:"include_friends"`
		ValidUntil     time.Time       `json:"valid_until"`
	}

	return func(ctx *gin.Context) {
		banID, banIDErr := getInt64Param(ctx, "ban_id")
		if banIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req updateBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		if time.Since(req.ValidUntil) > 0 {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		bannedPerson := store.NewBannedPerson()
		if banErr := app.db.GetBanByBanID(ctx, banID, &bannedPerson, false); banErr != nil {
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		if req.Reason == store.Custom {
			if req.ReasonText == "" {
				responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

				return
			}

			bannedPerson.ReasonText = req.ReasonText
		} else {
			bannedPerson.ReasonText = ""
		}

		bannedPerson.Note = req.Note
		bannedPerson.BanType = req.BanType
		bannedPerson.Reason = req.Reason
		bannedPerson.IncludeFriends = req.IncludeFriends
		bannedPerson.ValidUntil = req.ValidUntil

		if errSave := app.db.SaveBan(ctx, &bannedPerson.BanSteam); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save updated ban", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusAccepted, bannedPerson)
	}
}

func onAPIPostSetBanAppealStatus(app *App) gin.HandlerFunc {
	type setStatusReq struct {
		AppealState store.AppealState `json:"appeal_state"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banID, banIDErr := getInt64Param(ctx, "ban_id")
		if banIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req setStatusReq
		if !bind(ctx, log, &req) {
			return
		}

		bannedPerson := store.NewBannedPerson()
		if banErr := app.db.GetBanByBanID(ctx, banID, &bannedPerson, false); banErr != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if bannedPerson.AppealState == req.AppealState {
			responseErr(ctx, http.StatusConflict, errors.New("State must be different than previous"))

			return
		}

		original := bannedPerson.AppealState
		bannedPerson.AppealState = req.AppealState

		if errSave := app.db.SaveBan(ctx, &bannedPerson.BanSteam); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusAccepted, gin.H{})

		log.Info("Updated ban appeal state",
			zap.Int64("ban_id", banID),
			zap.Int("from_state", int(original)),
			zap.Int("to_state", int(req.AppealState)))
	}
}

func onAPIPostBansCIDRCreate(app *App) gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   store.StringSID `json:"target_id"`
		Duration   string          `json:"duration"`
		Note       string          `json:"note"`
		Reason     store.Reason    `json:"reason"`
		ReasonText string          `json:"reason_text"`
		CIDR       string          `json:"cidr"`
		ValidUntil time.Time       `json:"valid_until"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var (
			banCIDR store.BanCIDR
			sid     = currentUserProfile(ctx).SteamID
		)

		duration, errDuration := calcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if errBanCIDR := store.NewBanCIDR(ctx,
			store.StringSID(sid.String()),
			req.TargetID,
			duration,
			req.Reason,
			req.ReasonText,
			req.Note,
			store.Web,
			req.CIDR,
			store.Banned,
			&banCIDR,
		); errBanCIDR != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if errBan := app.BanCIDR(ctx, &banCIDR); errBan != nil {
			if errors.Is(errBan, store.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, consts.ErrDuplicate)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to save cidr ban", zap.Error(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, banCIDR)
	}
}

func onAPIGetBansCIDR(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.CIDRBansQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		bans, count, errBans := app.db.GetBansNet(ctx, req)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to fetch cidr bans", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, bans))
	}
}

func onAPIDeleteBansCIDR(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		netID, netIDErr := getInt64Param(ctx, "net_id")
		if netIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req apiUnbanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var banCidr store.BanCIDR
		if errFetch := app.db.GetBanNetByID(ctx, netID, &banCidr); errFetch != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		banCidr.UnbanReasonText = req.UnbanReasonText
		banCidr.Deleted = true

		if errSave := app.db.SaveBanNet(ctx, &banCidr); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to delete cidr ban", zap.Error(errSave))

			return
		}

		banCidr.NetID = 0

		ctx.JSON(http.StatusOK, banCidr)
	}
}

func onAPIPostBansCIDRUpdate(app *App) gin.HandlerFunc {
	type apiUpdateBanRequest struct {
		TargetID   store.StringSID `json:"target_id"`
		Note       string          `json:"note"`
		Reason     store.Reason    `json:"reason"`
		ReasonText string          `json:"reason_text"`
		CIDR       string          `json:"cidr"`
		ValidUntil time.Time       `json:"valid_until"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		netID, banIDErr := getInt64Param(ctx, "net_id")
		if banIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var ban store.BanCIDR

		if errBan := app.db.GetBanNetByID(ctx, netID, &ban); errBan != nil {
			if errors.Is(errBan, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var req apiUpdateBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		sid, errSID := req.TargetID.SID64(ctx)
		if errSID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		if req.Reason == store.Custom && req.ReasonText == "" {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		_, ipNet, errParseCIDR := net.ParseCIDR(req.CIDR)
		if errParseCIDR != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		ban.Reason = req.Reason
		ban.ReasonText = req.ReasonText
		ban.CIDR = ipNet.String()
		ban.Note = req.Note
		ban.ValidUntil = req.ValidUntil
		ban.TargetID = sid

		if errSave := app.db.SaveBanNet(ctx, &ban); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}

func onAPIPostBansASNCreate(app *App) gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   store.StringSID `json:"target_id"`
		Note       string          `json:"note"`
		Reason     store.Reason    `json:"reason"`
		ReasonText string          `json:"reason_text"`
		ASNum      int64           `json:"as_num"`
		Duration   string          `json:"duration"`
		ValidUntil time.Time       `json:"valid_until"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var (
			banASN store.BanASN
			sid    = currentUserProfile(ctx).SteamID
		)

		duration, errDuration := calcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if errBanSteamGroup := store.NewBanASN(ctx,
			store.StringSID(sid.String()),
			req.TargetID,
			duration,
			req.Reason,
			req.ReasonText,
			req.Note,
			store.Web,
			req.ASNum,
			store.Banned,
			&banASN,
		); errBanSteamGroup != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if errBan := app.BanASN(ctx, &banASN); errBan != nil {
			if errors.Is(errBan, store.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, consts.ErrDuplicate)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Failed to save asn ban", zap.Error(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, banASN)
	}
}

func onAPIGetBansASN(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.ASNBansQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		bansASN, count, errBans := app.db.GetBansASN(ctx, req)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to fetch banASN", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, bansASN))
	}
}

func onAPIDeleteBansASN(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		asnID, asnIDErr := getInt64Param(ctx, "asn_id")
		if asnIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req apiUnbanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var banAsn store.BanASN
		if errFetch := app.db.GetBanASN(ctx, asnID, &banAsn); errFetch != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		banAsn.UnbanReasonText = req.UnbanReasonText
		banAsn.Deleted = true

		if errSave := app.db.SaveBanASN(ctx, &banAsn); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to delete asn ban", zap.Error(errSave))

			return
		}

		banAsn.BanASNId = 0

		ctx.JSON(http.StatusOK, banAsn)
	}
}

func onAPIPostBansASNUpdate(app *App) gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   store.StringSID `json:"target_id"`
		Note       string          `json:"note"`
		Reason     store.Reason    `json:"reason"`
		ReasonText string          `json:"reason_text"`
		ValidUntil time.Time       `json:"valid_until"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		asnID, asnIDErr := getInt64Param(ctx, "asn_id")
		if asnIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var ban store.BanASN
		if errBan := app.db.GetBanASN(ctx, asnID, &ban); errBan != nil {
			if errors.Is(errBan, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var req apiBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		if ban.Reason == store.Custom && req.ReasonText == "" {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		sid, errSID := req.TargetID.SID64(ctx)
		if errSID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		ban.Note = req.Note
		ban.ValidUntil = req.ValidUntil
		ban.TargetID = sid
		ban.Reason = req.Reason
		ban.ReasonText = req.ReasonText

		if errSave := app.db.SaveBanASN(ctx, &ban); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}

func onAPIPostBansGroupCreate(app *App) gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   store.StringSID `json:"target_id"`
		GroupID    steamid.GID     `json:"group_id"`
		Duration   string          `json:"duration"`
		Note       string          `json:"note"`
		ValidUntil time.Time       `json:"valid_until"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var existing store.BanGroup
		if errExist := app.db.GetBanGroup(ctx, req.GroupID, &existing); errExist != nil {
			if !errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusConflict, consts.ErrDuplicate)

				return
			}
		}

		var (
			banSteamGroup store.BanGroup
			sid           = currentUserProfile(ctx).SteamID
		)

		duration, errDuration := calcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		if errBanSteamGroup := store.NewBanSteamGroup(ctx,
			store.StringSID(sid.String()),
			req.TargetID,
			duration,
			req.Note,
			store.Web,
			req.GroupID,
			"",
			store.Banned,
			&banSteamGroup,
		); errBanSteamGroup != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)
			log.Error("Failed to save group ban", zap.Error(errBanSteamGroup))

			return
		}

		if errBan := app.BanSteamGroup(ctx, &banSteamGroup); errBan != nil {
			if errors.Is(errBan, store.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, consts.ErrDuplicate)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusCreated, banSteamGroup)

		// Immediately update group members
		go app.updateBanChildren(ctx)
	}
}

func onAPIGetBansGroup(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.GroupBansQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		banGroups, count, errBans := app.db.GetBanGroups(ctx, req)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to fetch banGroups", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, banGroups))
	}
}

func onAPIDeleteBansGroup(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		groupID, groupIDErr := getInt64Param(ctx, "ban_group_id")
		if groupIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInternal)

			return
		}

		var req apiUnbanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var banGroup store.BanGroup
		if errFetch := app.db.GetBanGroupByID(ctx, groupID, &banGroup); errFetch != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInternal)

			return
		}

		banGroup.UnbanReasonText = req.UnbanReasonText
		banGroup.Deleted = true

		if errSave := app.db.SaveBanGroup(ctx, &banGroup); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to delete asn ban", zap.Error(errSave))

			return
		}

		banGroup.BanGroupID = 0
		ctx.JSON(http.StatusOK, banGroup)

		go app.updateBanChildren(ctx)
	}
}

func onAPIPostBansGroupUpdate(app *App) gin.HandlerFunc {
	type apiBanUpdateRequest struct {
		TargetID   store.StringSID `json:"target_id"`
		Note       string          `json:"note"`
		ValidUntil time.Time       `json:"valid_until"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banGroupID, banIDErr := getInt64Param(ctx, "ban_group_id")
		if banIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var req apiBanUpdateRequest
		if !bind(ctx, log, &req) {
			return
		}

		sid, errSID := req.TargetID.SID64(ctx)
		if errSID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var ban store.BanGroup

		if errExist := app.db.GetBanGroupByID(ctx, banGroupID, &ban); errExist != nil {
			if !errors.Is(errExist, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusConflict, consts.ErrDuplicate)

			return
		}

		ban.Note = req.Note
		ban.ValidUntil = req.ValidUntil
		ban.TargetID = sid

		if errSave := app.db.SaveBanGroup(ctx, &ban); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}

func onAPIGetPatreonPledges(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Only leak specific details
		// type basicPledge struct {
		//	Name      string
		//	Amount    int
		//	CreatedAt time.Time
		// }
		// var basic []basicPledge
		// for _, p := range pledges {
		//	t0 := config.Now()
		//	if p.Attributes.CreatedAt.Valid {
		//		t0 = p.Attributes.CreatedAt.Time.UTC()
		//	}
		//	basic = append(basic, basicPledge{
		//		Name:      users[p.Relationships.Patron.Data.ID].Attributes.FullName,
		//		Amount:    p.Attributes.AmountCents,
		//		CreatedAt: t0,
		//	})
		// }
		pledges, _, errPledges := app.patreon.pledges()
		if errPledges != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		ctx.JSON(http.StatusOK, pledges)
	}
}

func onAPIPostContest(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		newContest, _ := store.NewContest("", "", time.Now(), time.Now(), false)
		if !bind(ctx, log, &newContest) {
			return
		}

		if newContest.ContestID.IsNil() {
			newID, errID := uuid.NewV4()
			if errID != nil {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

				return
			}

			newContest.ContestID = newID
		}

		if errSave := app.db.ContestSave(ctx, &newContest); errSave != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		ctx.JSON(http.StatusOK, newContest)
	}
}

func onAPIDeleteContest(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		contestID, idErr := getUUIDParam(ctx, "contest_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var contest store.Contest

		if errContest := app.db.ContestByID(ctx, contestID, &contest); errContest != nil {
			if errors.Is(errContest, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrUnknownID)

				return
			}

			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			log.Error("Error getting contest for deletion", zap.Error(errContest))

			return
		}

		if errDelete := app.db.ContestDelete(ctx, contest.ContestID); errDelete != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Error deleting contest", zap.Error(errDelete))

			return
		}

		ctx.Status(http.StatusAccepted)

		log.Info("Contest deleted",
			zap.String("contest_id", contestID.String()),
			zap.String("title", contest.Title))
	}
}

func onAPIUpdateContest(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		if _, success := contestFromCtx(ctx, app); !success {
			return
		}

		var contest store.Contest
		if !bind(ctx, log, &contest) {
			return
		}

		if errSave := app.db.ContestSave(ctx, &contest); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Error updating contest", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusAccepted, contest)

		log.Info("Contest updated",
			zap.String("contest_id", contest.ContestID.String()),
			zap.String("title", contest.Title))
	}
}

type ForumCategoryRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Ordering    int    `json:"ordering"`
}

func onAPICreateForumCategory(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req ForumCategoryRequest
		if !bind(ctx, log, &req) {
			return
		}

		category := store.ForumCategory{
			Title:       req.Title,
			Description: req.Description,
			Ordering:    req.Ordering,
			TimeStamped: store.NewTimeStamped(),
		}

		if errSave := app.db.ForumCategorySave(ctx, &category); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Error creating new forum category", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, category)

		log.Info("New forum category created", zap.String("title", category.Title))
	}
}

func onAPIUpdateForumCategory(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		categoryID, errCategoryID := getIntParam(ctx, "forum_category_id")
		if errCategoryID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var category store.ForumCategory
		if errGet := app.db.ForumCategory(ctx, categoryID, &category); errGet != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var req ForumCategoryRequest
		if !bind(ctx, log, &req) {
			return
		}

		category.Title = req.Title
		category.Description = req.Description
		category.Ordering = req.Ordering

		if errSave := app.db.ForumCategorySave(ctx, &category); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Error creating new forum category", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, category)

		log.Info("New forum category updated", zap.String("title", category.Title))
	}
}

type ForumForumRequest struct {
	ForumCategoryID int `json:"forum_category_id"`
	ForumCategoryRequest
}

func onAPICreateForumForum(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req ForumForumRequest
		if !bind(ctx, log, &req) {
			return
		}

		forum := store.Forum{
			ForumCategoryID: req.ForumCategoryID,
			Title:           req.Title,
			Description:     req.Description,
			Ordering:        req.Ordering,
			TimeStamped:     store.NewTimeStamped(),
		}

		if errSave := app.db.ForumSave(ctx, &forum); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Error creating new forum", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusCreated, forum)

		log.Info("New forum created", zap.String("title", forum.Title))
	}
}

func onAPIUpdateForumForum(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		categoryID, errCategoryID := getIntParam(ctx, "forum_category_id")
		if errCategoryID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var forum store.Forum
		if errGet := app.db.Forum(ctx, categoryID, &forum); errGet != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		var req ForumForumRequest
		if !bind(ctx, log, &req) {
			return
		}

		forum.ForumCategoryID = req.ForumCategoryID
		forum.Title = req.Title
		forum.Description = req.Description
		forum.Ordering = req.Ordering

		if errSave := app.db.ForumSave(ctx, &forum); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Error updating forum", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusOK, forum)

		log.Info("Forum updated", zap.String("title", forum.Title))
	}
}
