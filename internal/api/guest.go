package api

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"go.uber.org/zap"
)

func onAPIPostDemosQuery(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req model.DemoFilter
		if !bind(ctx, log, &req) {
			return
		}

		demos, count, errDemos := env.Store().GetDemos(ctx, req)
		if errDemos != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)
			log.Error("Failed to query demos", zap.Error(errDemos))

			return
		}

		ctx.JSON(http.StatusCreated, newLazyResult(count, demos))
	}
}

// https://prometheus.io/docs/prometheus/latest/configuration/configuration/#http_sd_config
func onAPIGetPrometheusHosts(env Env) gin.HandlerFunc {
	type promStaticConfig struct {
		Targets []string          `json:"targets"`
		Labels  map[string]string `json:"labels"`
	}

	type portMap struct {
		Type string
		Port int
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var staticConfigs []promStaticConfig

		servers, _, errGetServers := env.Store().GetServers(ctx, model.ServerQueryFilter{})
		if errGetServers != nil {
			log.Error("Failed to fetch servers", zap.Error(errGetServers))
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			return
		}

		for _, nodePortConfig := range []portMap{{"node", 9100}} {
			staticConfig := promStaticConfig{Targets: nil, Labels: map[string]string{}}
			staticConfig.Labels["__meta_prometheus_job"] = nodePortConfig.Type

			for _, server := range servers {
				host := fmt.Sprintf("%s:%d", server.Address, nodePortConfig.Port)
				found := false

				for _, hostName := range staticConfig.Targets {
					if hostName == host {
						found = true

						break
					}
				}

				if !found {
					staticConfig.Targets = append(staticConfig.Targets, host)
				}
			}

			staticConfigs = append(staticConfigs, staticConfig)
		}

		// Don't wrap in our custom response format
		ctx.JSON(200, staticConfigs)
	}
}

func getDefaultFloat64(s string, def float64) float64 {
	if s != "" {
		l, errLat := strconv.ParseFloat(s, 64)
		if errLat != nil {
			return def
		}

		return l
	}

	return def
}

func onAPIGetServerStates(env Env) gin.HandlerFunc {
	type UserServers struct {
		Servers []model.BaseServer  `json:"servers"`
		LatLong ip2location.LatLong `json:"lat_long"`
	}

	return func(ctx *gin.Context) {
		var (
			lat = getDefaultFloat64(ctx.GetHeader("cf-iplatitude"), 41.7774)
			lon = getDefaultFloat64(ctx.GetHeader("cf-iplongitude"), -87.6160)
			// region := ctx.GetHeader("cf-region-code")
			curState = env.State().Current()
			servers  []model.BaseServer
		)

		for _, srv := range curState {
			servers = append(servers, model.BaseServer{
				Host:       srv.Host,
				Port:       srv.Port,
				IP:         srv.IP,
				Name:       srv.Name,
				NameShort:  srv.NameShort,
				Region:     srv.Region,
				CC:         srv.CC,
				ServerID:   srv.ServerID,
				Players:    srv.PlayerCount,
				MaxPlayers: srv.MaxPlayers,
				Bots:       srv.Bots,
				Map:        srv.Map,
				GameTypes:  []string{},
				Latitude:   srv.Latitude,
				Longitude:  srv.Longitude,
				Distance:   distance(srv.Latitude, srv.Longitude, lat, lon),
			})
		}

		sort.SliceStable(servers, func(i, j int) bool {
			return servers[i].Name < servers[j].Name
		})

		ctx.JSON(http.StatusOK, UserServers{
			Servers: servers,
			LatLong: ip2location.LatLong{
				Latitude:  lat,
				Longitude: lon,
			},
		})
	}
}

func onAPIExportBansValveSteamID(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		bans, _, errBans := env.Store().GetBansSteam(ctx, model.SteamBansQueryFilter{
			BansQueryFilter: model.BansQueryFilter{PermanentOnly: true},
		})

		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			return
		}

		var entries []string

		for _, ban := range bans {
			if ban.Deleted ||
				!ban.IsEnabled {
				continue
			}

			entries = append(entries, fmt.Sprintf("banid 0 %s", steamid.SID64ToSID(ban.TargetID)))
		}

		ctx.Data(http.StatusOK, "text/plain", []byte(strings.Join(entries, "\n")))
	}
}

func onAPIExportSourcemodSimpleAdmins(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		privilegedIds, errPrivilegedIds := env.Store().GetSteamIdsAbove(ctx, model.PReserved)
		if errPrivilegedIds != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			return
		}

		players, errPlayers := env.Store().GetPeopleBySteamID(ctx, privilegedIds)
		if errPlayers != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			return
		}

		sort.Slice(players, func(i, j int) bool {
			return players[i].PermissionLevel > players[j].PermissionLevel
		})

		bld := strings.Builder{}

		for _, player := range players {
			var perms string

			switch player.PermissionLevel {
			case model.PAdmin:
				perms = "z"
			case model.PModerator:
				perms = "abcdefgjk"
			case model.PEditor:
				perms = "ak"
			case model.PReserved:
				perms = "a"
			}

			if perms == "" {
				log.Warn("User has no perm string", zap.Int64("sid", player.SteamID.Int64()))
			} else {
				bld.WriteString(fmt.Sprintf("\"%s\" \"%s\"\n", steamid.SID64ToSID3(player.SteamID), perms))
			}
		}

		ctx.String(http.StatusOK, bld.String())
	}
}

func onAPIExportBansTF2BD(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// TODO limit / make specialized query since this returns all results
		bans, _, errBans := env.Store().GetBansSteam(ctx, model.SteamBansQueryFilter{
			BansQueryFilter: model.BansQueryFilter{
				QueryFilter: model.QueryFilter{
					Deleted: false,
				},
			},
		})

		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			return
		}

		var filtered []model.BannedSteamPerson

		for _, ban := range bans {
			if ban.Reason != model.Cheating ||
				ban.Deleted ||
				!ban.IsEnabled {
				continue
			}

			filtered = append(filtered, ban)
		}

		conf := env.Config()

		out := thirdparty.TF2BDSchema{
			Schema: "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/schemas/v3/playerlist.schema.json",
			FileInfo: thirdparty.FileInfo{
				Authors:     []string{conf.General.SiteName},
				Description: "Players permanently banned for cheating",
				Title:       fmt.Sprintf("%s Cheater List", conf.General.SiteName),
				UpdateURL:   conf.ExtURLRaw("/export/bans/tf2bd"),
			},
			Players: []thirdparty.Players{},
		}

		for _, ban := range filtered {
			out.Players = append(out.Players, thirdparty.Players{
				Attributes: []string{"cheater"},
				Steamid:    ban.TargetID,
				LastSeen: thirdparty.LastSeen{
					PlayerName: ban.TargetPersonaname,
					Time:       int(ban.UpdatedOn.Unix()),
				},
			})
		}

		ctx.JSON(http.StatusOK, out)
	}
}

func onAPIProfile(env Env) gin.HandlerFunc {
	type profileQuery struct {
		Query string `form:"query"`
	}

	type resp struct {
		Player   *model.Person        `json:"player"`
		Friends  []steamweb.Friend    `json:"friends"`
		Settings model.PersonSettings `json:"settings"`
	}

	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		requestCtx, cancelRequest := context.WithTimeout(ctx, time.Second*15)
		defer cancelRequest()

		var req profileQuery
		if errBind := ctx.Bind(&req); errBind != nil {
			responseErr(ctx, http.StatusBadRequest, nil)

			return
		}

		sid, errResolveSID64 := steamid.ResolveSID64(requestCtx, req.Query)
		if errResolveSID64 != nil {
			responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)

			return
		}

		person := model.NewPerson(sid)
		if errGetProfile := env.Store().GetOrCreatePersonBySteamID(requestCtx, sid, &person); errGetProfile != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)
			log.Error("Failed to create person", zap.Error(errGetProfile))

			return
		}

		if person.Expired() {
			if err := thirdparty.UpdatePlayerSummary(ctx, &person); err != nil {
				log.Error("Failed to update player summary", zap.Error(err))
			} else {
				if errSave := env.Store().SavePerson(ctx, &person); errSave != nil {
					log.Error("Failed to save person summary", zap.Error(errSave))
				}
			}
		}

		var response resp

		friendList, errFetchFriends := steamweb.GetFriendList(requestCtx, person.SteamID)
		if errFetchFriends == nil {
			response.Friends = friendList
		}

		response.Player = &person

		var settings model.PersonSettings
		if err := env.Store().GetPersonSettings(ctx, sid, &settings); err != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)
			log.Error("Failed to load person settings", zap.Error(err))

			return
		}

		response.Settings = settings

		ctx.JSON(http.StatusOK, response)
	}
}

func onAPIGetStats(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var stats model.Stats
		if errGetStats := env.Store().GetStats(ctx, &stats); errGetStats != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			return
		}

		stats.ServersAlive = 1

		ctx.JSON(http.StatusOK, stats)
	}
}

func loadBanMeta(_ *model.BannedSteamPerson) {
}

type serverInfoSafe struct {
	ServerNameLong string `json:"server_name_long"`
	ServerName     string `json:"server_name"`
	ServerID       int    `json:"server_id"`
	Colour         string `json:"colour"`
}

func onAPIGetServers(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fullServers, _, errServers := env.Store().GetServers(ctx, model.ServerQueryFilter{})
		if errServers != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			return
		}

		var servers []serverInfoSafe
		for _, server := range fullServers {
			servers = append(servers, serverInfoSafe{
				ServerNameLong: server.Name,
				ServerName:     server.ShortName,
				ServerID:       server.ServerID,
				Colour:         "",
			})
		}

		ctx.JSON(http.StatusOK, servers)
	}
}

func onAPIGetMapUsage(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mapUsages, errServers := env.Store().GetMapUsageStats(ctx)
		if errServers != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, mapUsages)
	}
}

func onAPIGetNewsLatest(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := env.Store().GetNewsLatest(ctx, 50, false)
		if errGetNewsLatest != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newsLatest)
	}
}

func onAPIGetWikiSlug(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		slug := strings.ToLower(ctx.Param("slug"))
		if slug[0] == '/' {
			slug = slug[1:]
		}

		var page wiki.Page
		if errGetWikiSlug := env.Store().GetWikiPageBySlug(ctx, slug, &page); errGetWikiSlug != nil {
			if errors.Is(errGetWikiSlug, errs.ErrNoResult) {
				ctx.JSON(http.StatusOK, page)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			return
		}

		if page.PermissionLevel > currentUser.PermissionLevel {
			responseErr(ctx, http.StatusForbidden, errs.ErrPermissionDenied)

			return
		}

		ctx.JSON(http.StatusOK, page)
	}
}

func onGetMediaByID(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mediaID, idErr := getIntParam(ctx, "media_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, errs.ErrInvalidParameter)

			return
		}

		var media model.Media
		if errMedia := env.Store().GetMediaByID(ctx, mediaID, &media); errMedia != nil {
			if errors.Is(errs.DBErr(errMedia), errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)
			} else {
				responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)
			}

			return
		}

		ctx.Data(http.StatusOK, media.MimeType, media.Contents)
	}
}

func distance(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	radianLat1 := math.Pi * lat1 / 180
	radianLat2 := math.Pi * lat2 / 180
	theta := lng1 - lng2
	radianTheta := math.Pi * theta / 180

	dist := math.Sin(radianLat1)*math.Sin(radianLat2) + math.Cos(radianLat1)*math.Cos(radianLat2)*math.Cos(radianTheta)
	if dist > 1 {
		dist = 1
	}

	dist = math.Acos(dist)
	dist = dist * 180 / math.Pi
	dist = dist * 60 * 1.1515
	dist *= 1.609344 // convert to km

	return dist
}

func onAPIGetPatreonCampaigns(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tiers, errTiers := env.Patreon().Tiers()
		if errTiers != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		ctx.JSON(http.StatusOK, tiers)
	}
}

func onAPIGetContests(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)
		publicOnly := user.PermissionLevel < model.PModerator
		contests, errContests := env.Store().Contests(ctx, publicOnly)

		if errContests != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(int64(len(contests)), contests))
	}
}

func onAPIGetContest(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		contest, success := contestFromCtx(ctx, env)
		if !success {
			return
		}

		ctx.JSON(http.StatusOK, contest)
	}
}

func onAPIGetContestEntries(env Env) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		contest, success := contestFromCtx(ctx, env)
		if !success {
			return
		}

		entries, errEntries := env.Store().ContestEntries(ctx, contest.ContestID)
		if errEntries != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, entries)
	}
}

func onAPIForumOverview(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type Overview struct {
		Categories []model.ForumCategory `json:"categories"`
	}

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		env.Activity().Touch(currentUser)

		categories, errCats := env.Store().ForumCategories(ctx)
		if errCats != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			log.Error("Could not load categories")

			return
		}

		forums, errForums := env.Store().Forums(ctx)
		if errForums != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			log.Error("Could not load forums", zap.Error(errForums))

			return
		}

		for index := range categories {
			for _, forum := range forums {
				if currentUser.PermissionLevel < forum.PermissionLevel {
					continue
				}

				if categories[index].ForumCategoryID == forum.ForumCategoryID {
					categories[index].Forums = append(categories[index].Forums, forum)
				}
			}

			if categories[index].Forums == nil {
				categories[index].Forums = []model.Forum{}
			}
		}

		ctx.JSON(http.StatusOK, Overview{Categories: categories})
	}
}

func onAPIForumThreads(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		env.Activity().Touch(currentUser)

		var tqf model.ThreadQueryFilter
		if !bind(ctx, log, &tqf) {
			return
		}

		threads, count, errThreads := env.Store().ForumThreads(ctx, tqf)
		if errThreads != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			log.Error("Could not load threads", zap.Error(errThreads))

			return
		}

		var forum model.Forum
		if err := env.Store().Forum(ctx, tqf.ForumID, &forum); err != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			log.Error("Could not load forum", zap.Error(errThreads))

			return
		}

		if forum.PermissionLevel > currentUser.PermissionLevel {
			responseErr(ctx, http.StatusUnauthorized, errs.ErrPermissionDenied)

			log.Error("User does not have access to forum")

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, threads))
	}
}

func onAPIForumThread(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		env.Activity().Touch(currentUser)

		forumThreadID, errID := getInt64Param(ctx, "forum_thread_id")
		if errID != nil {
			responseErr(ctx, http.StatusBadRequest, errs.ErrInvalidParameter)

			return
		}

		var thread model.ForumThread
		if errThreads := env.Store().ForumThread(ctx, forumThreadID, &thread); errThreads != nil {
			if errors.Is(errThreads, errs.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, errs.ErrNotFound)
			} else {
				responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)
				log.Error("Could not load threads")
			}

			return
		}

		ctx.JSON(http.StatusOK, thread)

		if err := env.Store().ForumThreadIncrView(ctx, forumThreadID); err != nil {
			log.Error("Failed to increment thread view count", zap.Error(err))
		}
	}
}

func onAPIForum(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		forumID, errForumID := getIntParam(ctx, "forum_id")
		if errForumID != nil {
			responseErr(ctx, http.StatusBadRequest, errs.ErrBadRequest)

			return
		}

		var forum model.Forum

		if errForum := env.Store().Forum(ctx, forumID, &forum); errForum != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			log.Error("Could not load forum")

			return
		}

		if forum.PermissionLevel > currentUser.PermissionLevel {
			responseErr(ctx, http.StatusForbidden, errs.ErrPermissionDenied)

			return
		}

		ctx.JSON(http.StatusOK, forum)
	}
}

func onAPIForumMessages(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var queryFilter model.ThreadMessagesQueryFilter
		if !bind(ctx, log, &queryFilter) {
			return
		}

		messages, count, errMessages := env.Store().ForumMessages(ctx, queryFilter)
		if errMessages != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			log.Error("Could not load thread messages", zap.Error(errMessages))

			return
		}

		activeUsers := env.Activity().Current()

		for idx := range messages {
			for _, activity := range activeUsers {
				if messages[idx].SourceID == activity.Person.SteamID {
					messages[idx].Online = true

					break
				}
			}
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, messages))
	}
}

func onAPIActiveUsers(env Env) gin.HandlerFunc {
	type userActivity struct {
		SteamID         steamid.SID64   `json:"steam_id"`
		Personaname     string          `json:"personaname"`
		PermissionLevel model.Privilege `json:"permission_level"`
		CreatedOn       time.Time       `json:"created_on"`
	}

	return func(ctx *gin.Context) {
		var results []userActivity

		for _, act := range env.Activity().Current() {
			results = append(results, userActivity{
				SteamID:         act.Person.SteamID,
				Personaname:     act.Person.Name,
				PermissionLevel: act.Person.PermissionLevel,
				CreatedOn:       act.LastActivity,
			})
		}

		ctx.JSON(http.StatusOK, results)
	}
}

func onAPIForumMessagesRecent(env Env) gin.HandlerFunc {
	log := env.Log().Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)

		messages, errThreads := env.Store().ForumRecentActivity(ctx, 5, user.PermissionLevel)
		if errThreads != nil {
			responseErr(ctx, http.StatusInternalServerError, errs.ErrInternal)

			log.Error("Could not load thread messages")

			return
		}

		if messages == nil {
			messages = []model.ForumMessage{}
		}

		ctx.JSON(http.StatusOK, messages)
	}
}
