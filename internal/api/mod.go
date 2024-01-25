package api

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

var (
	errUnbanFailed    = errors.New("failed to perform unban")
	errStateUnchanged = errors.New("state must be different than previous")
	errInvalidRegex   = errors.New("invalid regex format")
	errInvalidWeight  = errors.New("invalid weight value")
)

func onAPIPostNewsCreate(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.NewsEntry
		if !bind(ctx, log, &req) {
			return
		}

		if errSave := env.Store().SaveNewsArticle(ctx, &req); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusCreated, req)

		conf := env.Config()

		go env.SendPayload(conf.Discord.LogChannelID, discord.NewNewsMessage(req.BodyMD, req.Title))
	}
}

func onAPIPostNewsUpdate(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		newsID, errID := getIntParam(ctx, "news_id")
		if errID != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var entry domain.NewsEntry
		if errGet := env.Store().GetNewsByID(ctx, newsID, &entry); errGet != nil {
			if errors.Is(errs.DBErr(errGet), errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		if !bind(ctx, log, &entry) {
			return
		}

		if errSave := env.Store().SaveNewsArticle(ctx, &entry); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusAccepted, entry)

		conf := env.Config()
		env.SendPayload(conf.Discord.LogChannelID, discord.EditNewsMessages(entry.Title, entry.BodyMD))
	}
}

func onAPIGetNewsAll(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := env.Store().GetNewsLatest(ctx, 100, true)
		if errGetNewsLatest != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, newsLatest)
	}
}

func onAPIQueryWordFilters(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var opts domain.FiltersQueryFilter
		if !bind(ctx, log, &opts) {
			return
		}

		words, count, errGetFilters := env.Store().GetFilters(ctx, opts)
		if errGetFilters != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, words))
	}
}

func onAPIPostWordFilter(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.Filter
		if !bind(ctx, log, &req) {
			return
		}

		if req.Pattern == "" {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		_, errDur := util.ParseDuration(req.Duration)
		if errDur != nil {
			responseErr(ctx, http.StatusBadRequest, util.ErrInvalidDuration)

			return
		}

		if req.IsRegex {
			_, compErr := regexp.Compile(req.Pattern)
			if compErr != nil {
				responseErr(ctx, http.StatusBadRequest, errInvalidRegex)

				return
			}
		}

		if req.Weight < 1 {
			responseErr(ctx, http.StatusBadRequest, errInvalidWeight)

			return
		}

		now := time.Now()

		if req.FilterID > 0 {
			var existingFilter domain.Filter
			if errGet := env.Store().GetFilterByID(ctx, req.FilterID, &existingFilter); errGet != nil {
				if errors.Is(errGet, errs.ErrNoResult) {
					responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

					return
				}

				responseErr(ctx, http.StatusInternalServerError, errInternal)

				return
			}

			existingFilter.UpdatedOn = now
			existingFilter.Pattern = req.Pattern
			existingFilter.IsRegex = req.IsRegex
			existingFilter.IsEnabled = req.IsEnabled
			existingFilter.Action = req.Action
			existingFilter.Duration = req.Duration
			existingFilter.Weight = req.Weight

			if errSave := env.FilterAdd(ctx, &existingFilter); errSave != nil {
				responseErr(ctx, http.StatusInternalServerError, errInternal)

				return
			}

			req = existingFilter
		} else {
			profile := currentUserProfile(ctx)
			newFilter := domain.Filter{
				AuthorID:  profile.SteamID,
				Pattern:   req.Pattern,
				Action:    req.Action,
				Duration:  req.Duration,
				CreatedOn: now,
				UpdatedOn: now,
				IsRegex:   req.IsRegex,
				IsEnabled: req.IsEnabled,
				Weight:    req.Weight,
			}

			if errSave := env.FilterAdd(ctx, &newFilter); errSave != nil {
				responseErr(ctx, http.StatusInternalServerError, errInternal)

				return
			}

			req = newFilter
		}

		ctx.JSON(http.StatusOK, req)
	}
}

func onAPIDeleteWordFilter(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		wordID, wordIDErr := getInt64Param(ctx, "word_id")
		if wordIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var filter domain.Filter
		if errGet := env.Store().GetFilterByID(ctx, wordID, &filter); errGet != nil {
			if errors.Is(errGet, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		if errDrop := env.Store().DropFilter(ctx, &filter); errDrop != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusNoContent, nil)
	}
}

func onAPIPostWordMatch(env Env) gin.HandlerFunc {
	type matchRequest struct {
		Query string
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req matchRequest
		if !bind(ctx, log, &req) {
			return
		}

		words, _, errGetFilters := env.Store().GetFilters(ctx, domain.FiltersQueryFilter{})
		if errGetFilters != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		var matches []domain.Filter

		for _, filter := range words {
			if filter.Match(req.Query) {
				matches = append(matches, filter)
			}
		}

		ctx.JSON(http.StatusOK, matches)
	}
}

func onAPIExportBansValveIP(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		bans, _, errBans := env.Store().GetBansNet(ctx, domain.CIDRBansQueryFilter{})
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

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

func onAPISearchPlayers(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var query domain.PlayerQuery
		if !bind(ctx, log, &query) {
			return
		}

		people, count, errGetPeople := env.Store().GetPeople(ctx, query)
		if errGetPeople != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, people))
	}
}

func onAPIPostBanState(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		reportID, errID := getInt64Param(ctx, "report_id")
		if errID != nil || reportID <= 0 {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var report domain.Report
		if errReport := env.Store().GetReport(ctx, reportID, &report); errReport != nil {
			if errors.Is(errReport, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		env.SendPayload(env.Config().Discord.LogChannelID, discord.EditBanAppealStatusMessage())
	}
}

type apiUnbanRequest struct {
	UnbanReasonText string `json:"unban_reason_text"`
}

func onAPIQueryPersonConnections(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.ConnectionHistoryQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		ipHist, totalCount, errIPHist := env.Store().QueryConnectionHistory(ctx, req)
		if errIPHist != nil && !errors.Is(errIPHist, errs.ErrNoResult) {
			log.Error("Failed to query connection history", zap.Error(errIPHist))
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(totalCount, ipHist))
	}
}

func onAPIQueryMessageContext(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		messageID, errMessageID := getInt64Param(ctx, "person_message_id")
		if errMessageID != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)
			log.Debug("Got invalid person_message_id", zap.Error(errMessageID))

			return
		}

		padding, errPadding := getIntParam(ctx, "padding")
		if errPadding != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)
			log.Debug("Got invalid padding", zap.Error(errPadding))

			return
		}

		var msg domain.QueryChatHistoryResult
		if errMsg := env.Store().GetPersonMessage(ctx, messageID, &msg); errMsg != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		messages, errQuery := env.Store().GetPersonMessageContext(ctx, msg.ServerID, messageID, padding)
		if errQuery != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, messages)
	}
}

func onAPIGetAppeals(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.AppealQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		bans, total, errBans := env.Store().GetAppealsByActivity(ctx, req)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to fetch appeals", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(total, bans))
	}
}

func onAPIGetBansSteam(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.SteamBansQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		bans, count, errBans := env.Store().GetBansSteam(ctx, req)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to fetch steam bans", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, bans))
	}
}

func onAPIPostBanDelete(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banID, banIDErr := getInt64Param(ctx, "ban_id")
		if banIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var req apiUnbanRequest
		if !bind(ctx, log, &req) {
			return
		}

		bannedPerson := domain.NewBannedPerson()
		if banErr := env.Store().GetBanByBanID(ctx, banID, &bannedPerson, false); banErr != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		changed, errSave := env.Unban(ctx, bannedPerson.TargetID, req.UnbanReasonText)
		if errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		if !changed {
			responseErr(ctx, http.StatusNotFound, errUnbanFailed)

			return
		}

		ctx.JSON(http.StatusAccepted, gin.H{})
	}
}

func onAPIPostBanUpdate(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type updateBanRequest struct {
		TargetID       domain.StringSID `json:"target_id"`
		BanType        domain.BanType   `json:"ban_type"`
		Reason         domain.Reason    `json:"reason"`
		ReasonText     string           `json:"reason_text"`
		Note           string           `json:"note"`
		IncludeFriends bool             `json:"include_friends"`
		ValidUntil     time.Time        `json:"valid_until"`
	}

	return func(ctx *gin.Context) {
		banID, banIDErr := getInt64Param(ctx, "ban_id")
		if banIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var req updateBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		if time.Since(req.ValidUntil) > 0 {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		bannedPerson := domain.NewBannedPerson()
		if banErr := env.Store().GetBanByBanID(ctx, banID, &bannedPerson, false); banErr != nil {
			responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		if req.Reason == domain.Custom {
			if req.ReasonText == "" {
				responseErr(ctx, http.StatusBadRequest, errBadRequest)

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

		if errSave := env.Store().SaveBan(ctx, &bannedPerson.BanSteam); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to save updated ban", zap.Error(errSave))

			return
		}

		ctx.JSON(http.StatusAccepted, bannedPerson)
	}
}

func onAPIPostSetBanAppealStatus(env Env) gin.HandlerFunc {
	type setStatusReq struct {
		AppealState domain.AppealState `json:"appeal_state"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banID, banIDErr := getInt64Param(ctx, "ban_id")
		if banIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var req setStatusReq
		if !bind(ctx, log, &req) {
			return
		}

		bannedPerson := domain.NewBannedPerson()
		if banErr := env.Store().GetBanByBanID(ctx, banID, &bannedPerson, false); banErr != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		if bannedPerson.AppealState == req.AppealState {
			responseErr(ctx, http.StatusConflict, errStateUnchanged)

			return
		}

		original := bannedPerson.AppealState
		bannedPerson.AppealState = req.AppealState

		if errSave := env.Store().SaveBan(ctx, &bannedPerson.BanSteam); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusAccepted, gin.H{})

		log.Info("Updated ban appeal state",
			zap.Int64("ban_id", banID),
			zap.Int("from_state", int(original)),
			zap.Int("to_state", int(req.AppealState)))
	}
}

func onAPIPostBansCIDRCreate(env Env) gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   domain.StringSID `json:"target_id"`
		Duration   string           `json:"duration"`
		Note       string           `json:"note"`
		Reason     domain.Reason    `json:"reason"`
		ReasonText string           `json:"reason_text"`
		CIDR       string           `json:"cidr"`
		ValidUntil time.Time        `json:"valid_until"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var (
			banCIDR domain.BanCIDR
			sid     = currentUserProfile(ctx).SteamID
		)

		duration, errDuration := util.CalcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		if errBanCIDR := domain.NewBanCIDR(ctx,
			domain.StringSID(sid.String()),
			req.TargetID,
			duration,
			req.Reason,
			req.ReasonText,
			req.Note,
			domain.Web,
			req.CIDR,
			domain.Banned,
			&banCIDR,
		); errBanCIDR != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		if errBan := env.BanCIDR(ctx, &banCIDR); errBan != nil {
			if errors.Is(errBan, errs.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, errs.ErrDuplicate)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to save cidr ban", zap.Error(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, banCIDR)
	}
}

func onAPIGetBansCIDR(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.CIDRBansQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		bans, count, errBans := env.Store().GetBansNet(ctx, req)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to fetch cidr bans", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, bans))
	}
}

func onAPIDeleteBansCIDR(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		netID, netIDErr := getInt64Param(ctx, "net_id")
		if netIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var req apiUnbanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var banCidr domain.BanCIDR
		if errFetch := env.Store().GetBanNetByID(ctx, netID, &banCidr); errFetch != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		banCidr.UnbanReasonText = req.UnbanReasonText
		banCidr.Deleted = true

		if errSave := env.Store().SaveBanNet(ctx, &banCidr); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to delete cidr ban", zap.Error(errSave))

			return
		}

		banCidr.NetID = 0

		ctx.JSON(http.StatusOK, banCidr)
	}
}

func onAPIPostBansCIDRUpdate(env Env) gin.HandlerFunc {
	type apiUpdateBanRequest struct {
		TargetID   domain.StringSID `json:"target_id"`
		Note       string           `json:"note"`
		Reason     domain.Reason    `json:"reason"`
		ReasonText string           `json:"reason_text"`
		CIDR       string           `json:"cidr"`
		ValidUntil time.Time        `json:"valid_until"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		netID, banIDErr := getInt64Param(ctx, "net_id")
		if banIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var ban domain.BanCIDR

		if errBan := env.Store().GetBanNetByID(ctx, netID, &ban); errBan != nil {
			if errors.Is(errBan, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		var req apiUpdateBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		sid, errSID := req.TargetID.SID64(ctx)
		if errSID != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		if req.Reason == domain.Custom && req.ReasonText == "" {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		_, ipNet, errParseCIDR := net.ParseCIDR(req.CIDR)
		if errParseCIDR != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		ban.Reason = req.Reason
		ban.ReasonText = req.ReasonText
		ban.CIDR = ipNet.String()
		ban.Note = req.Note
		ban.ValidUntil = req.ValidUntil
		ban.TargetID = sid

		if errSave := env.Store().SaveBanNet(ctx, &ban); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}

func onAPIPostBansASNCreate(env Env) gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   domain.StringSID `json:"target_id"`
		Note       string           `json:"note"`
		Reason     domain.Reason    `json:"reason"`
		ReasonText string           `json:"reason_text"`
		ASNum      int64            `json:"as_num"`
		Duration   string           `json:"duration"`
		ValidUntil time.Time        `json:"valid_until"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var (
			banASN domain.BanASN
			sid    = currentUserProfile(ctx).SteamID
		)

		duration, errDuration := util.CalcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		if errBanSteamGroup := domain.NewBanASN(ctx,
			domain.StringSID(sid.String()),
			req.TargetID,
			duration,
			req.Reason,
			req.ReasonText,
			req.Note,
			domain.Web,
			req.ASNum,
			domain.Banned,
			&banASN,
		); errBanSteamGroup != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		if errBan := env.BanASN(ctx, &banASN); errBan != nil {
			if errors.Is(errBan, errs.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, errs.ErrDuplicate)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)

			log.Error("Failed to save asn ban", zap.Error(errBan))

			return
		}

		ctx.JSON(http.StatusCreated, banASN)
	}
}

func onAPIGetBansASN(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.ASNBansQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		bansASN, count, errBans := env.Store().GetBansASN(ctx, req)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to fetch banASN", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, bansASN))
	}
}

func onAPIDeleteBansASN(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		asnID, asnIDErr := getInt64Param(ctx, "asn_id")
		if asnIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var req apiUnbanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var banAsn domain.BanASN
		if errFetch := env.Store().GetBanASN(ctx, asnID, &banAsn); errFetch != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		banAsn.UnbanReasonText = req.UnbanReasonText
		banAsn.Deleted = true

		if errSave := env.Store().SaveBanASN(ctx, &banAsn); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to delete asn ban", zap.Error(errSave))

			return
		}

		banAsn.BanASNId = 0

		ctx.JSON(http.StatusOK, banAsn)
	}
}

func onAPIPostBansASNUpdate(env Env) gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   domain.StringSID `json:"target_id"`
		Note       string           `json:"note"`
		Reason     domain.Reason    `json:"reason"`
		ReasonText string           `json:"reason_text"`
		ValidUntil time.Time        `json:"valid_until"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		asnID, asnIDErr := getInt64Param(ctx, "asn_id")
		if asnIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var ban domain.BanASN
		if errBan := env.Store().GetBanASN(ctx, asnID, &ban); errBan != nil {
			if errors.Is(errBan, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		var req apiBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		if ban.Reason == domain.Custom && req.ReasonText == "" {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		sid, errSID := req.TargetID.SID64(ctx)
		if errSID != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		ban.Note = req.Note
		ban.ValidUntil = req.ValidUntil
		ban.TargetID = sid
		ban.Reason = req.Reason
		ban.ReasonText = req.ReasonText

		if errSave := env.Store().SaveBanASN(ctx, &ban); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}

func onAPIPostBansGroupCreate(env Env) gin.HandlerFunc {
	type apiBanRequest struct {
		TargetID   domain.StringSID `json:"target_id"`
		GroupID    steamid.GID      `json:"group_id"`
		Duration   string           `json:"duration"`
		Note       string           `json:"note"`
		ValidUntil time.Time        `json:"valid_until"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req apiBanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var existing domain.BanGroup
		if errExist := env.Store().GetBanGroup(ctx, req.GroupID, &existing); errExist != nil {
			if !errors.Is(errExist, errs.ErrNoResult) {
				responseErr(ctx, http.StatusConflict, errs.ErrDuplicate)

				return
			}
		}

		var (
			banSteamGroup domain.BanGroup
			sid           = currentUserProfile(ctx).SteamID
		)

		duration, errDuration := util.CalcDuration(req.Duration, req.ValidUntil)
		if errDuration != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)

			return
		}

		if errBanSteamGroup := domain.NewBanSteamGroup(ctx,
			domain.StringSID(sid.String()),
			req.TargetID,
			duration,
			req.Note,
			domain.Web,
			req.GroupID,
			"",
			domain.Banned,
			&banSteamGroup,
		); errBanSteamGroup != nil {
			responseErr(ctx, http.StatusBadRequest, errBadRequest)
			log.Error("Failed to save group ban", zap.Error(errBanSteamGroup))

			return
		}

		if errBan := env.BanSteamGroup(ctx, &banSteamGroup); errBan != nil {
			if errors.Is(errBan, errs.ErrDuplicate) {
				responseErr(ctx, http.StatusConflict, errs.ErrDuplicate)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusCreated, banSteamGroup)
	}
}

func onAPIGetBansGroup(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req domain.GroupBansQueryFilter
		if !bind(ctx, log, &req) {
			return
		}

		banGroups, count, errBans := env.Store().GetBanGroups(ctx, req)
		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to fetch banGroups", zap.Error(errBans))

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, banGroups))
	}
}

func onAPIDeleteBansGroup(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		groupID, groupIDErr := getInt64Param(ctx, "ban_group_id")
		if groupIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, errInternal)

			return
		}

		var req apiUnbanRequest
		if !bind(ctx, log, &req) {
			return
		}

		var banGroup domain.BanGroup
		if errFetch := env.Store().GetBanGroupByID(ctx, groupID, &banGroup); errFetch != nil {
			responseErr(ctx, http.StatusBadRequest, errInternal)

			return
		}

		banGroup.UnbanReasonText = req.UnbanReasonText
		banGroup.Deleted = true

		if errSave := env.Store().SaveBanGroup(ctx, &banGroup); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)
			log.Error("Failed to delete asn ban", zap.Error(errSave))

			return
		}

		banGroup.BanGroupID = 0

		ctx.JSON(http.StatusOK, banGroup)
	}
}

func onAPIPostBansGroupUpdate(env Env) gin.HandlerFunc {
	type apiBanUpdateRequest struct {
		TargetID   domain.StringSID `json:"target_id"`
		Note       string           `json:"note"`
		ValidUntil time.Time        `json:"valid_until"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		banGroupID, banIDErr := getInt64Param(ctx, "ban_group_id")
		if banIDErr != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var req apiBanUpdateRequest
		if !bind(ctx, log, &req) {
			return
		}

		sid, errSID := req.TargetID.SID64(ctx)
		if errSID != nil {
			responseErr(ctx, http.StatusBadRequest, errInvalidParameter)

			return
		}

		var ban domain.BanGroup

		if errExist := env.Store().GetBanGroupByID(ctx, banGroupID, &ban); errExist != nil {
			if !errors.Is(errExist, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

				return
			}

			responseErr(ctx, http.StatusConflict, errs.ErrDuplicate)

			return
		}

		ban.Note = req.Note
		ban.ValidUntil = req.ValidUntil
		ban.TargetID = sid

		if errSave := env.Store().SaveBanGroup(ctx, &ban); errSave != nil {
			responseErr(ctx, http.StatusInternalServerError, errInternal)

			return
		}

		ctx.JSON(http.StatusOK, ban)
	}
}

func onAPIGetPatreonPledges(env Env) gin.HandlerFunc {
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
		pledges, _, errPledges := env.Patreon().Pledges()
		if errPledges != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		ctx.JSON(http.StatusOK, pledges)
	}
}

func onAPIGetWarningState(env Env) gin.HandlerFunc {
	// log := app.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())
	type warningState struct {
		MaxWeight int                  `json:"max_weight"`
		Current   []domain.UserWarning `json:"current"`
	}

	return func(ctx *gin.Context) {
		state := env.Warnings().State()

		outputState := warningState{MaxWeight: env.Config().Filter.MaxWeight}

		for _, warn := range state {
			outputState.Current = append(outputState.Current, warn...)
		}

		ctx.JSON(http.StatusOK, outputState)
	}
}
