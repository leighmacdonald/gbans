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

		demos, count, errDemos := app.db.GetDemos(ctx, req)
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

		servers, _, errGetServers := app.db.GetServers(ctx, store.ServerQueryFilter{})
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
		bans, _, errBans := app.db.GetBansSteam(ctx, store.SteamBansQueryFilter{
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
		privilegedIds, errPrivilegedIds := app.db.GetSteamIdsAbove(ctx, consts.PReserved)
		if errPrivilegedIds != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		players, errPlayers := app.db.GetPeopleBySteamID(ctx, privilegedIds)
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
		bans, _, errBans := app.db.GetBansSteam(ctx, store.SteamBansQueryFilter{
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

		var filtered []store.BannedSteamPerson

		for _, ban := range bans {
			if ban.Reason != store.Cheating ||
				ban.Deleted ||
				!ban.IsEnabled {
				continue
			}

			filtered = append(filtered, ban)
		}

		out := thirdparty.TF2BDSchema{
			Schema: "https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/schemas/v3/playerlist.schema.json",
			FileInfo: thirdparty.FileInfo{
				Authors:     []string{app.conf.General.SiteName},
				Description: "Players permanently banned for cheating",
				Title:       fmt.Sprintf("%s Cheater List", app.conf.General.SiteName),
				UpdateURL:   app.ExtURLRaw("/export/bans/tf2bd"),
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
		Player  *store.Person     `json:"player"`
		Friends []steamweb.Friend `json:"friends"`
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

		person := store.NewPerson(sid)
		if errGetProfile := app.PersonBySID(requestCtx, sid, &person); errGetProfile != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
			log.Error("Failed to create person", zap.Error(errGetProfile))

			log.Error("Failed to create new profile", zap.Error(errGetProfile))

			return
		}

		var response resp

		friendList, errFetchFriends := steamweb.GetFriendList(requestCtx, person.SteamID)
		if errFetchFriends == nil {
			response.Friends = friendList
		}

		response.Player = &person

		ctx.JSON(http.StatusOK, response)
	}
}

func onAPIGetStats(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var stats store.Stats
		if errGetStats := app.db.GetStats(ctx, &stats); errGetStats != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		stats.ServersAlive = 1

		ctx.JSON(http.StatusOK, stats)
	}
}

func loadBanMeta(_ *store.BannedSteamPerson) {
}

type serverInfoSafe struct {
	ServerNameLong string `json:"server_name_long"`
	ServerName     string `json:"server_name"`
	ServerID       int    `json:"server_id"`
	Colour         string `json:"colour"`
}

func onAPIGetServers(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fullServers, _, errServers := app.db.GetServers(ctx, store.ServerQueryFilter{})
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
		mapUsages, errServers := app.db.GetMapUsageStats(ctx)
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
		newsLatest, errGetNewsLatest := app.db.GetNewsLatest(ctx, 50, false)
		if errGetNewsLatest != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			return
		}

		ctx.JSON(http.StatusOK, newsLatest)
	}
}

func contestFromCtx(ctx *gin.Context, app *App) (store.Contest, bool) {
	contestID, idErr := getUUIDParam(ctx, "contest_id")
	if idErr != nil {
		responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

		return store.Contest{}, false
	}

	var contest store.Contest
	if errContests := app.db.ContestByID(ctx, contestID, &contest); errContests != nil {
		responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

		return store.Contest{}, false
	}

	if !contest.Public && currentUserProfile(ctx).PermissionLevel < consts.PModerator {
		responseErr(ctx, http.StatusForbidden, consts.ErrNotFound)

		return store.Contest{}, false
	}

	return contest, true
}

func onAPIGetWikiSlug(app *App) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		slug := strings.ToLower(ctx.Param("slug"))
		if slug[0] == '/' {
			slug = slug[1:]
		}

		var page wiki.Page
		if errGetWikiSlug := app.db.GetWikiPageBySlug(ctx, slug, &page); errGetWikiSlug != nil {
			if errors.Is(errGetWikiSlug, store.ErrNoResult) {
				ctx.JSON(http.StatusOK, page)

				return
			}

			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

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

		var media store.Media
		if errMedia := app.db.GetMediaByID(ctx, mediaID, &media); errMedia != nil {
			if errors.Is(store.Err(errMedia), store.ErrNoResult) {
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
		contests, errContests := app.db.Contests(ctx, publicOnly)

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

		entries, errEntries := app.db.ContestEntries(ctx, contest.ContestID)
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
		Categories []store.ForumCategory `json:"categories"`
	}

	return func(ctx *gin.Context) {
		currentUser := currentUserProfile(ctx)

		categories, errCats := app.db.ForumCategories(ctx)
		if errCats != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Could not load categories")

			return
		}

		forums, errForums := app.db.Forums(ctx)
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
				categories[index].Forums = []store.Forum{}
			}
		}

		ctx.JSON(http.StatusOK, Overview{Categories: categories})
	}
}

func onAPIForumThreads(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var tqf store.ThreadQueryFilter
		if !bind(ctx, log, &tqf) {
			return
		}

		threads, count, errThreads := app.db.ForumThreads(ctx, tqf)
		if errThreads != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Could not load threads", zap.Error(errThreads))

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, threads))
	}
}

func onAPIForumThread(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		forumThreadID, errID := getInt64Param(ctx, "forum_thread_id")
		if errID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrInvalidParameter)

			return
		}

		var thread store.ForumThread
		if errThreads := app.db.ForumThread(ctx, forumThreadID, &thread); errThreads != nil {
			if errors.Is(errThreads, store.ErrNoResult) {
				responseErr(ctx, http.StatusNotFound, consts.ErrNotFound)
			} else {
				responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)
				log.Error("Could not load threads")
			}

			return
		}

		ctx.JSON(http.StatusOK, thread)

		if err := app.db.ForumThreadIncrView(ctx, forumThreadID); err != nil {
			log.Error("Failed to increment thread view count", zap.Error(err))
		}
	}
}

func onAPIForum(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		forumID, errForumID := getIntParam(ctx, "forum_id")
		if errForumID != nil {
			responseErr(ctx, http.StatusBadRequest, consts.ErrBadRequest)

			return
		}

		var forum store.Forum

		if errForum := app.db.Forum(ctx, forumID, &forum); errForum != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Could not load forum")

			return
		}

		ctx.JSON(http.StatusOK, forum)
	}
}

func onAPIForumMessages(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		var tmqf store.ThreadMessagesQueryFilter
		if !bind(ctx, log, &tmqf) {
			return
		}

		threads, count, errThreads := app.db.ForumMessages(ctx, tmqf)
		if errThreads != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Could not load thread messages")

			return
		}

		ctx.JSON(http.StatusOK, newLazyResult(count, threads))
	}
}

func onAPIForumMessagesRecent(app *App) gin.HandlerFunc {
	log := app.log.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(ctx *gin.Context) {
		user := currentUserProfile(ctx)

		messages, errThreads := app.db.ForumRecentActivity(ctx, 5, user.PermissionLevel)
		if errThreads != nil {
			responseErr(ctx, http.StatusInternalServerError, consts.ErrInternal)

			log.Error("Could not load thread messages")

			return
		}

		if messages == nil {
			messages = []store.ForumMessage{}
		}

		ctx.JSON(http.StatusOK, messages)
	}
}
