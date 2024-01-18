package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func onAPIPostDemosQuery(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var req store.DemoFilter
		if !bind(ctx, log, &req) {
			return
		}

		demos, count, errDemos := store.GetDemos(ctx, app.db, req)
		if errDemos != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to query demos", zap.Error(errDemos))

			return
		}

		ctx.JSON(http.StatusCreated, newLazyResult(count, demos))
	}
}

// https://prometheus.io/docs/prometheus/latest/configuration/configuration/#http_sd_config
func onAPIGetPrometheusHosts(app *App) gin.HandlerFunc {
	type promStaticConfig struct {
		Targets []string          `json:"targets"`
		Labels  map[string]string `json:"labels"`
	}

	type portMap struct {
		Type string
		Port int
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var staticConfigs []promStaticConfig

		servers, _, errGetServers := store.GetServers(ctx, app.db, store.ServerQueryFilter{})
		if errGetServers != nil {
			log.Error("Failed to fetch servers", zap.Error(errGetServers))
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

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

func onAPIGetServerStates(app *App) gin.HandlerFunc {
	type UserServers struct {
		Servers []baseServer        `json:"servers"`
		LatLong ip2location.LatLong `json:"lat_long"`
	}

	return func(ctx *gin.Context) {
		var (
			lat = getDefaultFloat64(ctx.GetHeader("cf-iplatitude"), 41.7774)
			lon = getDefaultFloat64(ctx.GetHeader("cf-iplongitude"), -87.6160)
			// region := ctx.GetHeader("cf-region-code")
			curState = app.state.current()
			servers  []baseServer
		)

		for _, srv := range curState {
			servers = append(servers, baseServer{
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

func onAPIExportBansValveSteamID(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		bans, _, errBans := store.GetBansSteam(ctx, app.db, store.SteamBansQueryFilter{
			BansQueryFilter: store.BansQueryFilter{PermanentOnly: true},
		})

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

			entries = append(entries, fmt.Sprintf("banid 0 %s", steamid.SID64ToSID(ban.TargetID)))
		}

		ctx.Data(http.StatusOK, "text/plain", []byte(strings.Join(entries, "\n")))
	}
}

func onAPIExportSourcemodSimpleAdmins(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		privilegedIds, errPrivilegedIds := store.GetSteamIdsAbove(ctx, app.db, consts.PReserved)
		if errPrivilegedIds != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		players, errPlayers := store.GetPeopleBySteamID(ctx, app.db, privilegedIds)
		if errPlayers != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		sort.Slice(players, func(i, j int) bool {
			return players[i].PermissionLevel > players[j].PermissionLevel
		})

		bld := strings.Builder{}

		for _, player := range players {
			var perms string

			switch player.PermissionLevel {
			case consts.PAdmin:
				perms = "z"
			case consts.PModerator:
				perms = "abcdefgjk"
			case consts.PEditor:
				perms = "ak"
			case consts.PReserved:
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

func onAPIExportBansTF2BD(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// TODO limit / make specialized query since this returns all results
		bans, _, errBans := store.GetBansSteam(ctx, app.db, store.SteamBansQueryFilter{
			BansQueryFilter: store.BansQueryFilter{
				QueryFilter: store.QueryFilter{
					Deleted: false,
				},
			},
		})

		if errBans != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

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

		conf := app.config()

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

func onAPIProfile(app *App) gin.HandlerFunc {
	type profileQuery struct {
		Query string `form:"query"`
	}

	type resp struct {
		Player   *model.Person        `json:"player"`
		Friends  []steamweb.Friend    `json:"friends"`
		Settings model.PersonSettings `json:"settings"`
	}

	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

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
			responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)

			return
		}

		person := model.NewPerson(sid)
		if errGetProfile := PersonBySID(requestCtx, app.db, sid, &person); errGetProfile != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to create person", zap.Error(errGetProfile))

			return
		}

		var response resp

		friendList, errFetchFriends := steamweb.GetFriendList(requestCtx, person.SteamID)
		if errFetchFriends == nil {
			response.Friends = friendList
		}

		response.Player = &person

		var settings model.PersonSettings
		if err := store.GetPersonSettings(ctx, app.db, sid, &settings); err != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to load person settings", zap.Error(err))

			return
		}

		response.Settings = settings

		ctx.JSON(http.StatusOK, response)
	}
}

func onAPIGetStats(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var stats model.Stats
		if errGetStats := store.GetStats(ctx, app.db, &stats); errGetStats != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

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

func onAPIGetServers(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fullServers, _, errServers := store.GetServers(ctx, app.db, store.ServerQueryFilter{})
		if errServers != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

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

func onAPIGetMapUsage(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mapUsages, errServers := store.GetMapUsageStats(ctx, app.db)
		if errServers != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, mapUsages)
	}
}

func getSID64Param(c *gin.Context, key string) (steamid.SID64, error) {
	i, errGetParam := getInt64Param(c, key)
	if errGetParam != nil {
		return "", errGetParam
	}

	sid := steamid.New(i)
	if !sid.Valid() {
		return "", consts.ErrInvalidSID
	}

	return sid, nil
}

func getInt64Param(ctx *gin.Context, key string) (int64, error) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		return 0, errors.Errorf("Failed to get %s", key)
	}

	value, valueErr := strconv.ParseInt(valueStr, 10, 64)
	if valueErr != nil {
		return 0, errors.Errorf("Failed to parse %s: %v", key, valueErr)
	}

	if value <= 0 {
		return 0, errors.Errorf("Invalid %s: %v", key, valueErr)
	}

	return value, nil
}

func getIntParam(ctx *gin.Context, key string) (int, error) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		return 0, errors.Errorf("Failed to get %s", key)
	}

	return util.StringToInt(valueStr), nil
}

func getUUIDParam(ctx *gin.Context, key string) (uuid.UUID, error) {
	valueStr := ctx.Param(key)
	if valueStr == "" {
		return uuid.UUID{}, errors.Errorf("Failed to get %s", key)
	}

	parsedUUID, errString := uuid.FromString(valueStr)
	if errString != nil {
		return uuid.UUID{}, errors.Wrap(errString, "Failed to parse UUID")
	}

	return parsedUUID, nil
}

func onAPIGetNewsLatest(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		newsLatest, errGetNewsLatest := store.GetNewsLatest(ctx, app.db, 50, false)
		if errGetNewsLatest != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newsLatest)
	}
}

func contestFromCtx(ctx *gin.Context, app *App) (model.Contest, bool) {
	contestID, idErr := getUUIDParam(ctx, "contest_id")
	if idErr != nil {
		responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

		return model.Contest{}, false
	}

	var contest model.Contest
	if errContests := store.ContestByID(ctx, app.db, contestID, &contest); errContests != nil {
		responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

		return model.Contest{}, false
	}

	if !contest.Public && currentUserProfile(ctx).PermissionLevel < consts.PModerator {
		responseErr(ctx, http.StatusForbidden, consts.ErrNotFound)

		return model.Contest{}, false
	}

	return contest, true
}

func onAPIGetWikiSlug(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		slug := strings.ToLower(ctx.Param("slug"))
		if slug[0] == '/' {
			slug = slug[1:]
		}

		var page wiki.Page
		if errGetWikiSlug := store.GetWikiPageBySlug(ctx, app.db, slug, &page); errGetWikiSlug != nil {
			if errors.Is(errGetWikiSlug, store.ErrNoResult) {
				ctx.JSON(http.StatusOK, page)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		if page.PermissionLevel > currentUser.PermissionLevel {
			responseErr(ctx, http.StatusForbidden, consts.ErrPermissionDenied)

			return
		}

		ctx.JSON(http.StatusOK, page)
	}
}

func onGetMediaByID(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		mediaID, idErr := getIntParam(ctx, "media_id")
		if idErr != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var media model.Media
		if errMedia := store.GetMediaByID(ctx, app.db, mediaID, &media); errMedia != nil {
			if errors.Is(store.DBErr(errMedia), store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)
			} else {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			}

			return
		}

		ctx.Data(http.StatusOK, media.MimeType, media.Contents)
	}
}

type sbBanRecord struct {
	BanID       int           `json:"ban_id"`
	SiteName    string        `json:"site_name"`
	SiteID      int           `json:"site_id"`
	PersonaName string        `json:"persona_name"`
	SteamID     steamid.SID64 `json:"steam_id"`
	Reason      string        `json:"reason"`
	Duration    time.Duration `json:"duration"`
	Permanent   bool          `json:"permanent"`
	CreatedOn   time.Time     `json:"created_on"`
}

func getSourceBans(ctx context.Context, steamID steamid.SID64) ([]sbBanRecord, error) {
	client := &http.Client{Timeout: time.Second * 10}
	url := fmt.Sprintf("https://bd-api.roto.lol/sourcebans/%s", steamID)

	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if errReq != nil {
		return nil, errors.Wrap(errReq, "Failed to create request")
	}

	resp, errResp := client.Do(req)
	if errResp != nil {
		return nil, errors.Wrap(errResp, "Failed to perform request")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, errBody := io.ReadAll(resp.Body)
	if errBody != nil {
		return nil, errors.Wrap(errBody, "Failed to read body")
	}

	var records []sbBanRecord
	if errJSON := json.Unmarshal(body, &records); errJSON != nil {
		return nil, errors.Wrap(errJSON, "Failed to decode body")
	}

	return records, nil
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

func onAPIGetPatreonCampaigns(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tiers, errTiers := app.patreon.tiers()
		if errTiers != nil {
			responseErr(ctx, http.StatusInternalServerError, nil)

			return
		}

		ctx.JSON(http.StatusOK, tiers)
	}
}

func onAPIGetContests(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)
		publicOnly := user.PermissionLevel < consts.PModerator
		contests, errContests := store.Contests(ctx, app.db, publicOnly)

		if errContests != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(int64(len(contests)), contests))
	}
}

func onAPIGetContest(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		contest, success := contestFromCtx(ctx, app)
		if !success {
			return
		}

		ctx.JSON(http.StatusOK, contest)
	}
}

func onAPIGetContestEntries(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		contest, success := contestFromCtx(ctx, app)
		if !success {
			return
		}

		entries, errEntries := store.ContestEntries(ctx, app.db, contest.ContestID)
		if errEntries != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, entries)
	}
}

func onAPIForumOverview(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	type Overview struct {
		Categories []model.ForumCategory `json:"categories"`
	}

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		app.activityTracker.touch(currentUser)

		categories, errCats := store.ForumCategories(ctx, app.db)
		if errCats != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Could not load categories")

			return
		}

		forums, errForums := store.Forums(ctx, app.db)
		if errForums != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

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

func onAPIForumThreads(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		app.activityTracker.touch(currentUser)

		var tqf store.ThreadQueryFilter
		if !bind(ctx, log, &tqf) {
			return
		}

		threads, count, errThreads := store.ForumThreads(ctx, app.db, tqf)
		if errThreads != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Could not load threads", zap.Error(errThreads))

			return
		}

		var forum model.Forum
		if err := store.Forum(ctx, app.db, tqf.ForumID, &forum); err != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Could not load forum", zap.Error(errThreads))

			return
		}

		if forum.PermissionLevel > currentUser.PermissionLevel {
			responseErr(ctx, http.StatusUnauthorized, consts.ErrPermissionDenied)

			log.Error("User does not have access to forum")

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, threads))
	}
}

func onAPIForumThread(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		app.activityTracker.touch(currentUser)

		forumThreadID, errID := getInt64Param(ctx, "forum_thread_id")
		if errID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var thread model.ForumThread
		if errThreads := store.ForumThread(ctx, app.db, forumThreadID, &thread); errThreads != nil {
			if errors.Is(errThreads, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)
			} else {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
				log.Error("Could not load threads")
			}

			return
		}

		ctx.JSON(http.StatusOK, thread)

		if err := store.ForumThreadIncrView(ctx, app.db, forumThreadID); err != nil {
			log.Error("Failed to increment thread view count", zap.Error(err))
		}
	}
}

func onAPIForum(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		forumID, errForumID := getIntParam(ctx, "forum_id")
		if errForumID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var forum model.Forum

		if errForum := store.Forum(ctx, app.db, forumID, &forum); errForum != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Could not load forum")

			return
		}

		if forum.PermissionLevel > currentUser.PermissionLevel {
			responseErr(ctx, http.StatusForbidden, consts.ErrPermissionDenied)

			return
		}

		ctx.JSON(http.StatusOK, forum)
	}
}

func onAPIForumMessages(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var queryFilter store.ThreadMessagesQueryFilter
		if !bind(ctx, log, &queryFilter) {
			return
		}

		messages, count, errMessages := store.ForumMessages(ctx, app.db, queryFilter)
		if errMessages != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Could not load thread messages", zap.Error(errMessages))

			return
		}

		activeUsers := app.activityTracker.current()

		for idx := range messages {
			for _, activity := range activeUsers {
				if messages[idx].SourceID == activity.person.SteamID {
					messages[idx].Online = true

					break
				}
			}
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, messages))
	}
}

func onAPIActiveUsers(app *App) gin.HandlerFunc {
	type userActivity struct {
		SteamID         steamid.SID64    `json:"steam_id"`
		Personaname     string           `json:"personaname"`
		PermissionLevel consts.Privilege `json:"permission_level"`
		CreatedOn       time.Time        `json:"created_on"`
	}

	return func(ctx *gin.Context) {
		var results []userActivity

		for _, act := range app.activityTracker.current() {
			results = append(results, userActivity{
				SteamID:         act.person.SteamID,
				Personaname:     act.person.Name,
				PermissionLevel: act.person.PermissionLevel,
				CreatedOn:       act.lastActivity,
			})
		}

		ctx.JSON(http.StatusOK, results)
	}
}

func onAPIForumMessagesRecent(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)

		messages, errThreads := store.ForumRecentActivity(ctx, app.db, 5, user.PermissionLevel)
		if errThreads != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Could not load thread messages")

			return
		}

		if messages == nil {
			messages = []model.ForumMessage{}
		}

		ctx.JSON(http.StatusOK, messages)
	}
}
