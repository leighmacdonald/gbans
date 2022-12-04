package app

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/krayzpipes/cronticker/cronticker"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/query"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

func (app *App) IsSteamGroupBanned(steamId steamid.SID64) bool {
	app.bannedGroupMembersMu.RLock()
	defer app.bannedGroupMembersMu.RUnlock()
	for _, groupMembers := range app.bannedGroupMembers {
		for _, member := range groupMembers {
			if steamId == member {
				return true
			}
		}
	}
	return false
}

func (app *App) steamGroupMembershipUpdater(ctx context.Context, _ store.PersonStore) {
	var update = func() {
		localCtx, cancel := context.WithTimeout(ctx, time.Second*120)
		newMap := map[steamid.GID]steamid.Collection{}
		total := 0
		for _, gid := range config.General.BannedSteamGroupIds {
			members, errMembers := steamweb.GetGroupMembers(localCtx, gid)
			if errMembers != nil {
				log.Warnf("Failed to fetch group members")
				cancel()
				continue
			}
			newMap[gid] = members
			total += len(members)
		}
		app.bannedGroupMembersMu.Lock()
		app.bannedGroupMembers = newMap
		app.bannedGroupMembersMu.Unlock()
		cancel()
		log.WithFields(log.Fields{"count": total}).Debugf("Updated group member ban list")
	}
	update()
	ticker := time.NewTicker(time.Minute * 15)
	for {
		select {
		case <-ticker.C:
			update()
		case <-ctx.Done():
			log.Debugf("steamGroupMembershipUpdater shutting down")
			return
		}
	}
}

func (app *App) showReportMeta(ctx context.Context, database store.ReportStore) {
	type reportMeta struct {
		TotalOpen   int
		TotalClosed int
		Open        int
		NeedInfo    int
		Open1Day    int
		Open3Days   int
		OpenWeek    int
		OpenNew     int
	}
	var showReports = func() {
		reports, errReports := database.GetReports(ctx, store.AuthorQueryFilter{
			QueryFilter: store.QueryFilter{
				Limit: 0,
			},
		})
		if errReports != nil {
			log.Errorf("failed to fetch reports for report metadata")
			return
		}
		now := config.Now()
		var m reportMeta
		for _, report := range reports {
			if report.ReportStatus == model.ClosedWithAction || report.ReportStatus == model.ClosedWithoutAction {
				m.TotalClosed++
				continue
			}
			m.TotalOpen++
			if report.ReportStatus == model.NeedMoreInfo {
				m.NeedInfo++
			} else {
				m.Open++
			}
			if now.Sub(report.CreatedOn) > time.Hour*24*7 {
				m.OpenWeek++
			} else if now.Sub(report.CreatedOn) > time.Hour*24*3 {
				m.Open3Days++
			} else if now.Sub(report.CreatedOn) > time.Hour*24 {
				m.Open1Day++
			} else {
				m.OpenNew++
			}
		}
		reportNotice := &discordgo.MessageEmbed{
			URL:   config.ExtURL("/admin/reports"),
			Type:  discordgo.EmbedTypeRich,
			Title: "User Report Stats",
			Color: int(green),
		}
		if m.OpenWeek > 0 {
			reportNotice.Color = int(red)
		} else if m.Open3Days > 0 {
			reportNotice.Color = int(orange)
		}
		reportNotice.Description = "Current Open Report Counts"

		addFieldInline(reportNotice, "New", fmt.Sprintf("%d", m.Open1Day))
		addFieldInline(reportNotice, "Total Open", fmt.Sprintf("%d", m.TotalOpen))
		addFieldInline(reportNotice, "Total Closed", fmt.Sprintf("%d", m.TotalClosed))
		addFieldInline(reportNotice, ">1 Day", fmt.Sprintf("%d", m.Open1Day))
		addFieldInline(reportNotice, ">3 Days", fmt.Sprintf("%d", m.Open3Days))
		addFieldInline(reportNotice, ">1 Week", fmt.Sprintf("%d", m.OpenWeek))
		app.sendDiscordPayload(discordPayload{channelId: config.Discord.ReportLogChannelId, embed: reportNotice})
		//sendDiscordPayload(app.discordSendMsg)
	}
	time.Sleep(time.Second * 2)
	showReports()
	ticker := time.NewTicker(time.Hour * 24)
	for {
		select {
		case <-ticker.C:
			showReports()
		case <-ctx.Done():
			log.Debugf("showReportMeta shutting down")
			return
		}
	}
}

func (app *App) cleanupTasks(ctx context.Context, database store.AuthStore) {
	ticker := time.NewTicker(time.Hour * 24)
	for {
		select {
		case <-ticker.C:
			if err := database.PrunePersonAuth(ctx); err != nil && !errors.Is(err, store.ErrNoResult) {
				log.WithError(err).Errorf("Error pruning expired refresh tokens")
			}
		case <-ctx.Done():
			log.Debugf("profileUpdater shutting down")
			return
		}
	}
}

// profileUpdater takes care of periodically querying the steam api for updates player summaries.
// The 100 oldest profiles are updated on each execution
func profileUpdater(ctx context.Context, database store.PersonStore) {
	var update = func() {
		localCtx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()
		people, errGetExpired := database.GetExpiredProfiles(localCtx, 100)
		if errGetExpired != nil {
			log.Errorf("Failed to get expired profiles: %v", errGetExpired)
			return
		}
		var sids steamid.Collection
		for _, person := range people {
			sids = append(sids, person.SteamID)
		}
		summaries, errSummaries := steamweb.PlayerSummaries(sids)
		if errSummaries != nil {
			log.Errorf("Failed to get Player summaries: %v", errSummaries)
			return
		}
		for _, summary := range summaries {
			// TODO batch update upserts
			sid, errSid := steamid.SID64FromString(summary.Steamid)
			if errSid != nil {
				log.Errorf("Failed to parse steamid from webapi: %v", errSid)
				continue
			}
			person := model.NewPerson(sid)
			if errGetPerson := database.GetOrCreatePersonBySteamID(localCtx, sid, &person); errGetPerson != nil {
				log.Errorf("Failed to get person: %v", errGetPerson)
				continue
			}
			person.PlayerSummary = &summary
			if errSavePerson := database.SavePerson(localCtx, &person); errSavePerson != nil {
				log.Errorf("Failed to save person: %v", errSavePerson)
				continue
			}
		}
		log.WithFields(log.Fields{"count": len(summaries)}).Trace("Profiles updated")
	}
	update()
	ticker := time.NewTicker(time.Second * 60)
	for {
		select {
		case <-ticker.C:
			update()
		case <-ctx.Done():
			log.Debugf("profileUpdater shutting down")
			return
		}
	}
}

func (app *App) serverA2SStatusUpdater(ctx context.Context, database store.ServerStore, updateFreq time.Duration) {
	var updateStatus = func(localCtx context.Context, localDb store.ServerStore) error {
		cancelCtx, cancel := context.WithTimeout(localCtx, updateFreq/2)
		defer cancel()
		servers, errGetServers := localDb.GetServers(cancelCtx, false)
		if errGetServers != nil {
			return errors.Wrapf(errGetServers, "Failed to get servers")
		}
		waitGroup := &sync.WaitGroup{}
		for _, srv := range servers {
			waitGroup.Add(1)
			go func(server model.Server) {
				defer waitGroup.Done()
				newStatus, errA := query.A2SQueryServer(server)
				if errA != nil {
					log.Tracef("Failed to update a2s status: %v", errA)
					return
				}
				app.serverStateA2SMu.Lock()
				app.serverStateA2S[server.ServerNameShort] = *newStatus
				app.serverStateA2SMu.Unlock()
			}(srv)
		}
		waitGroup.Wait()
		return nil
	}
	ticker := time.NewTicker(updateFreq)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.WithFields(log.Fields{"state": "started"}).Tracef("Server status state updateStatus")
			if errUpdate := updateStatus(ctx, database); errUpdate != nil {
				log.Errorf("Error trying to updateStatus server status state: %v", errUpdate)
			}
			log.WithFields(log.Fields{"state": "success"}).Tracef("Server status state updateStatus")
		}
	}
}

func (app *App) serverRCONStatusUpdater(ctx context.Context, database store.ServerStore, updateFreq time.Duration) {
	var updateStatus = func(localCtx context.Context, localDb store.ServerStore) error {
		cancelCtx, cancel := context.WithTimeout(localCtx, updateFreq/2)
		defer cancel()
		servers, errGetServers := localDb.GetServers(cancelCtx, false)
		if errGetServers != nil {
			return errors.Wrapf(errGetServers, "Failed to get servers")
		}
		waitGroup := &sync.WaitGroup{}
		for _, srv := range servers {
			waitGroup.Add(1)
			go func(c context.Context, server model.Server) {
				defer waitGroup.Done()
				newStatus, queryErr := query.GetServerStatus(c, server)
				if queryErr != nil {
					log.Tracef("Failed to query server status: %v", queryErr)
					return
				}
				app.serverStateStatusMu.Lock()
				app.serverStateStatus[server.ServerNameShort] = newStatus
				app.serverStateStatusMu.Unlock()
			}(cancelCtx, srv)
		}
		waitGroup.Wait()
		return nil
	}
	ticker := time.NewTicker(updateFreq)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.WithFields(log.Fields{"state": "started"}).Tracef("Server status state updateStatus")
			if errUpdate := updateStatus(ctx, database); errUpdate != nil {
				log.Errorf("Error trying to updateStatus server status state: %v", errUpdate)
			}
			log.WithFields(log.Fields{"state": "success"}).Tracef("Server status state updateStatus")
		}
	}
}

// serverStateRefresher periodically compiles and caches the current known db, rcon & a2s server state
// into a ServerState instance
func (app *App) serverStateRefresher(ctx context.Context, database store.ServerStore, updateFreq time.Duration) {
	var refreshState = func() error {
		var newState model.ServerStateCollection
		servers, errServers := database.GetServers(ctx, false)
		if errServers != nil {
			return errors.Errorf("Failed to fetch servers: %v", errServers)
		}
		app.serverStateA2SMu.RLock()
		defer app.serverStateA2SMu.RUnlock()
		app.serverStateStatusMu.RLock()
		defer app.serverStateStatusMu.RUnlock()
		for _, server := range servers {
			var state model.ServerState
			// use existing state for start?
			state.ServerId = server.ServerID
			state.Name = server.ServerNameLong
			state.NameShort = server.ServerNameShort
			state.Host = server.Address
			state.Port = server.Port
			state.Enabled = server.IsEnabled
			state.Region = server.Region
			state.CountryCode = server.CC
			state.Latitude = server.Latitude
			state.Longitude = server.Longitude
			a2sInfo, a2sFound := app.serverStateA2S[server.ServerNameShort]
			if a2sFound {
				if a2sInfo.Name != "" {
					state.Name = a2sInfo.Name
				}
				state.NameA2S = a2sInfo.Name
				state.Protocol = a2sInfo.Protocol
				state.Map = a2sInfo.Map
				state.Folder = a2sInfo.Folder
				state.Game = a2sInfo.Game
				state.AppId = a2sInfo.ID
				state.PlayerCount = int(a2sInfo.Players)
				state.MaxPlayers = int(a2sInfo.MaxPlayers)
				state.Bots = int(a2sInfo.Bots)
				state.ServerType = a2sInfo.ServerType.String()
				state.ServerOS = a2sInfo.ServerOS.String()
				state.Password = !a2sInfo.Visibility
				state.VAC = a2sInfo.VAC
				state.Version = a2sInfo.Version
				if a2sInfo.SourceTV != nil {
					state.STVPort = a2sInfo.SourceTV.Port
					state.STVName = a2sInfo.SourceTV.Name
				}
				if a2sInfo.ExtendedServerInfo != nil {
					state.SteamID = steamid.SID64(a2sInfo.ExtendedServerInfo.SteamID)
					state.GameID = a2sInfo.ExtendedServerInfo.GameID
					state.Keywords = strings.Split(a2sInfo.ExtendedServerInfo.Keywords, ",")
				}
			}
			statusInfo, statusFound := app.serverStateStatus[server.ServerNameShort]
			if statusFound {
				if state.Name != "" {
					state.Name = statusInfo.ServerName
				}
				if state.PlayerCount < statusInfo.PlayersCount {
					state.PlayerCount = statusInfo.PlayersCount
				}
				// rcon status doesn't respect sv_visiblemaxplayers (sp?) so this doesn't work well
				//if state.MaxPlayers < statusInfo.PlayersMax {
				//	state.MaxPlayers = statusInfo.PlayersMax
				//}
				if state.Map != "" && state.Map != statusInfo.Map {
					state.Map = statusInfo.Map
				}
				var knownPlayers []model.ServerStatePlayer
				for _, player := range statusInfo.Players {
					var newPlayer model.ServerStatePlayer
					newPlayer.UserID = player.UserID
					newPlayer.Name = player.Name
					newPlayer.SID = player.SID
					newPlayer.ConnectedTime = player.ConnectedTime
					newPlayer.State = player.State
					newPlayer.Ping = player.Ping
					newPlayer.Loss = player.Loss
					newPlayer.IP = player.IP
					newPlayer.Port = player.Port
					knownPlayers = append(knownPlayers, newPlayer)
				}
				state.Players = knownPlayers
			}
			newState = append(newState, state)
		}
		app.serverStateMu.Lock()
		app.serverState = newState
		app.serverStateMu.Unlock()
		return nil
	}
	ticker := time.NewTicker(updateFreq)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if errUpdate := refreshState(); errUpdate != nil {
				log.Errorf("Failed to refreshState server state: %v", errUpdate)
			}
		}
	}
}

func (app *App) patreonUpdater() {
	updateTimer := time.NewTicker(time.Hour * 1)
	if app.patreon == nil {
		return
	}
	var update = func() {
		newCampaigns, errCampaigns := thirdparty.PatreonGetTiers(app.patreon)
		if errCampaigns != nil {
			log.WithError(errCampaigns).Errorf("Failed to refresh campaigns")
			return
		}
		newPledges, newUsers, errPledges := thirdparty.PatreonGetPledges(app.patreon)
		if errPledges != nil {
			log.WithError(errPledges).Errorf("Failed to refresh pledges")
			return
		}
		app.patreonMu.Lock()
		app.patreonCampaigns = newCampaigns
		app.patreonPledges = newPledges
		app.patreonUsers = newUsers
		app.patreonMu.Unlock()
		cents := 0
		totalCents := 0
		for _, p := range newPledges {
			cents += p.Attributes.AmountCents
			if p.Attributes.TotalHistoricalAmountCents != nil {
				totalCents += *p.Attributes.TotalHistoricalAmountCents
			}
		}
		log.WithFields(log.Fields{
			"campaign_count": len(newCampaigns),
			"current_cents":  cents,
			"total_cents":    totalCents,
		}).Infof("Patreon Updated")
	}
	update()
	for {
		select {
		case <-updateTimer.C:
			update()
		case <-app.ctx.Done():
			return
		}
	}

}

// mapChanger watches over servers and checks for servers on maps with 0 players.
// If there is no player for a long enough duration and the map is not one of the
// maps in the default map set, a changelevel request will be made to the server
//
// Relevant config values:
// - general.map_changer_enabled
// - general.default_map
func (app *App) mapChanger(ctx context.Context, database store.ServerStore, timeout time.Duration) {
	type at struct {
		lastActive time.Time
		triggered  bool
	}
	activityMap := map[string]*at{}
	ticker := time.NewTicker(time.Second * 60)
	for {
		select {
		case <-ticker.C:
			if !config.General.MapChangerEnabled {
				continue
			}
			stateCopy := app.ServerState()
			for _, state := range stateCopy {
				activity, activityFound := activityMap[state.NameShort]
				if !activityFound || len(state.Players) > 0 {
					activityMap[state.NameShort] = &at{config.Now(), false}
					continue
				}
				if !activity.triggered && time.Since(activity.lastActive) > timeout {
					isDefaultMap := false
					for _, m := range config.General.DefaultMaps {
						if m == state.Map {
							isDefaultMap = true
							break
						}
					}
					if isDefaultMap {
						continue
					}
					var server model.Server
					if errGetServer := database.GetServerByName(ctx, state.NameShort, &server); errGetServer != nil {
						log.Errorf("Failed to get server for map changer: %v", errGetServer)
						continue
					}
					nextMap := server.DefaultMap
					if nextMap == "" {
						nextMap = config.General.DefaultMaps[rand.Intn(len(config.General.DefaultMaps))]
					}
					if nextMap == "" {
						log.Errorf("Failed to get valid nextMap value")
						continue
					}
					if server.DefaultMap == state.Map {
						continue
					}
					go func(s model.Server, mapName string) {
						var logger = log.WithFields(log.Fields{"map": nextMap, "reason": "no_activity", "server": state.ServerId})
						logger.Infof("Idle map change triggered")
						if _, errExecRCON := query.ExecRCON(ctx, server, fmt.Sprintf("changelevel %s", mapName)); errExecRCON != nil {
							logger.Errorf("failed to exec mapchanger rcon: %v", errExecRCON)
						}
						logger.Infof("Idle map change complete")
					}(server, nextMap)
					activity.triggered = true
					continue
				}
				if activity.triggered {
					activity.triggered = false
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// banSweeper periodically will query the database for expired bans and remove them.
func banSweeper(ctx context.Context, database store.Store) {
	log.WithFields(log.Fields{"service": "ban_sweeper", "status": "ready"}).Debugf("Service status changed")
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ticker.C:
			waitGroup := &sync.WaitGroup{}
			waitGroup.Add(3)
			go func() {
				defer waitGroup.Done()
				expiredBans, errExpiredBans := database.GetExpiredBans(ctx)
				if errExpiredBans != nil && !errors.Is(errExpiredBans, store.ErrNoResult) {
					log.Warnf("Failed to get expired expiredBans: %v", errExpiredBans)
				} else {
					for _, expiredBan := range expiredBans {
						if errDrop := database.DropBan(ctx, &expiredBan, false); errDrop != nil {
							log.Errorf("Failed to drop expired expiredBan: %v", errDrop)
						} else {
							banType := "Ban"
							if expiredBan.BanType == model.NoComm {
								banType = "Mute"
							}
							var person model.Person
							if errPerson := database.GetOrCreatePersonBySteamID(ctx, expiredBan.TargetId, &person); errPerson != nil {
								log.Errorf("Failed to get expired person: %v", errPerson)
								continue
							}
							name := person.PersonaName
							if name == "" {
								name = person.SteamID.String()
							}
							fields := log.Fields{
								"sid":    expiredBan.TargetId,
								"name":   name,
								"origin": expiredBan.Origin.String(),
								"reason": expiredBan.Reason.String(),
							}
							if expiredBan.ReasonText != "" {
								fields["custom"] = expiredBan.ReasonText
							}
							log.WithFields(fields).Infof("%s expired", banType)
						}
					}
				}
			}()
			go func() {
				defer waitGroup.Done()
				expiredNetBans, errExpiredNetBans := database.GetExpiredNetBans(ctx)
				if errExpiredNetBans != nil && !errors.Is(errExpiredNetBans, store.ErrNoResult) {
					log.Warnf("Failed to get expired netbans: %v", errExpiredNetBans)
				} else {
					for _, expiredNetBan := range expiredNetBans {
						if errDropBanNet := database.DropBanNet(ctx, &expiredNetBan); errDropBanNet != nil {
							log.Errorf("Failed to drop expired network expiredNetBan: %v", errDropBanNet)
						} else {
							log.Infof("CIDR ban expired: %v", expiredNetBan)
						}
					}
				}
			}()
			go func() {
				defer waitGroup.Done()
				expiredASNBans, errExpiredASNBans := database.GetExpiredASNBans(ctx)
				if errExpiredASNBans != nil && !errors.Is(errExpiredASNBans, store.ErrNoResult) {
					log.Warnf("Failed to get expired asnbans: %v", errExpiredASNBans)
				} else {
					for _, expiredASNBan := range expiredASNBans {
						if errDropASN := database.DropBanASN(ctx, &expiredASNBan); errDropASN != nil {
							log.Errorf("Failed to drop expired asn ban: %v", errDropASN)
						} else {
							log.Infof("ASN ban expired: %v", expiredASNBan)
						}
					}
				}
			}()
			waitGroup.Wait()
		case <-ctx.Done():
			log.Debugf("banSweeper shutting down")
			return
		}
	}
}

func guessMapType(mapName string) string {
	mapName = strings.TrimPrefix(mapName, "workshop/")
	pieces := strings.SplitN(mapName, "_", 2)
	if len(pieces) == 1 {
		return "unknown"
	} else {
		return strings.ToLower(pieces[0])
	}
}

type SvRegion int

const (
	RegionNaEast SvRegion = iota
	RegionNAWest
	RegionSouthAmerica
	RegionEurope
	RegionAsia
	RegionAustralia
	RegionMiddleEast
	RegionAfrica
	RegionWorld SvRegion = 255
)

func SteamRegionIdString(region SvRegion) string {
	switch region {
	case RegionNaEast:
		return "ne"
	case RegionNAWest:
		return "nw"
	case RegionSouthAmerica:
		return "sa"
	case RegionEurope:
		return "eu"
	case RegionAsia:
		return "as"
	case RegionAustralia:
		return "au"
	case RegionMiddleEast:
		return "me"
	case RegionAfrica:
		return "af"
	case RegionWorld:
		fallthrough
	default:
		return "wd"
	}
}

func (app *App) localStatUpdater(ctx context.Context, database store.Store) {
	var build = func() {
		if errBuild := database.BuildLocalTF2Stats(ctx); errBuild != nil {
			log.WithError(errBuild).Error("Error building local stats")
		}
	}
	saveTicker, errSaveTicker := cronticker.NewTicker("0 */5 * * * *")
	if errSaveTicker != nil {
		log.WithError(errSaveTicker).Panicf("Invalid save ticker cron format")
		return
	}
	// Rebuild stats every hour
	buildTicker, errBuildTicker := cronticker.NewTicker("0 * * * * *")
	if errBuildTicker != nil {
		log.WithError(errBuildTicker).Panicf("Invalid build ticker cron format")
		return
	}
	build()
	for {
		select {
		case <-buildTicker.C:
			build()
		case saveTime := <-saveTicker.C:
			stats := model.NewLocalTF2Stats()
			stats.CreatedOn = saveTime
			servers, errServers := database.GetServers(ctx, false)
			if errServers != nil {
				log.WithError(errServers).Errorf("Failed to fetch servers to build local cache")
				continue
			}
			serverNameMap := map[string]string{}
			for _, server := range servers {
				serverNameMap[fmt.Sprintf("%s:%d", server.Address, server.Port)] = server.ServerNameShort
				ipAddr, errIp := server.IP()
				if errIp != nil {
					continue
				}
				serverNameMap[fmt.Sprintf("%s:%d", ipAddr.String(), server.Port)] = server.ServerNameShort
			}
			app.serverStateMu.RLock()
			for _, ss := range app.serverState {
				sn := fmt.Sprintf("%s:%d", ss.Host, ss.Port)
				serverName, nameFound := serverNameMap[sn]
				if !nameFound {
					log.WithFields(log.Fields{"name": serverName}).Errorf("Cannot find server name")
					continue
				}
				stats.Servers[serverName] = ss.PlayerCount
				stats.Players += ss.PlayerCount
				_, foundRegion := stats.Regions[ss.Region]
				if !foundRegion {
					stats.Regions[ss.Region] = 0
				}
				stats.Regions[ss.Region] += ss.PlayerCount

				mapType := guessMapType(ss.Map)
				_, mapTypeFound := stats.MapTypes[mapType]
				if !mapTypeFound {
					stats.MapTypes[mapType] = 0
				}
				stats.MapTypes[mapType] += ss.PlayerCount
				if ss.PlayerCount >= ss.MaxPlayers && ss.MaxPlayers > 0 {
					stats.CapacityFull++
				} else if ss.PlayerCount == 0 {
					stats.CapacityEmpty++
				} else {
					stats.CapacityPartial++
				}
			}
			app.serverStateMu.RUnlock()
			if errSave := database.SaveLocalTF2Stats(ctx, store.Live, stats); errSave != nil {
				log.WithError(errSave).Error("Failed to save local stats state")
				continue
			}
		case <-ctx.Done():
			return
		}
	}
}

func (app *App) masterServerListUpdater(ctx context.Context, database store.Store, updateFreq time.Duration) {
	prevStats := model.NewGlobalTF2Stats()
	locationCache := map[string]ip2location.LatLong{}
	var build = func() {
		if errBuild := database.BuildGlobalTF2Stats(ctx); errBuild != nil {
			log.WithError(errBuild).Error("Error building stats")
			return
		}
	}

	var update = func() error {
		allServers, errServers := steamweb.GetServerList(map[string]string{
			"appid":     "440",
			"dedicated": "1",
		})
		if errServers != nil {
			return errors.Wrap(errServers, "Failed to fetch server list")
		}
		var communityServers []model.ServerLocation
		stats := model.NewGlobalTF2Stats()
		for _, baseServer := range allServers {
			server := model.ServerLocation{
				LatLong: ip2location.LatLong{},
				Server:  baseServer,
			}
			hostParts := strings.SplitN(server.Addr, ":", 2)
			ipAddr := hostParts[0]
			_, found := locationCache[ipAddr]
			if !found {
				var locRecord ip2location.LocationRecord
				ip := net.ParseIP(ipAddr)
				if errLocation := database.GetLocationRecord(ctx, ip, &locRecord); errLocation != nil {
					continue
				}
				locationCache[ipAddr] = locRecord.LatLong
			}
			server.LatLong = locationCache[ipAddr]
			stats.ServersTotal++
			stats.Players += server.Players
			stats.Bots += server.Bots
			if server.MaxPlayers > 0 && server.Players >= server.MaxPlayers {
				stats.CapacityFull++
			} else if server.Players == 0 {
				stats.CapacityEmpty++
			} else {
				stats.CapacityPartial++
			}
			if server.Secure {
				stats.Secure++
			}
			region := SteamRegionIdString(SvRegion(server.Region))
			_, regionFound := stats.Regions[region]
			if !regionFound {
				stats.Regions[region] = 0
			}
			stats.Regions[region] += server.Players
			mapType := guessMapType(server.Map)
			_, mapTypeFound := stats.MapTypes[mapType]
			if !mapTypeFound {
				stats.MapTypes[mapType] = 0
			}
			stats.MapTypes[mapType]++
			if strings.Contains(server.Gametype, "valve") ||
				!server.Dedicated ||
				!server.Secure {
				stats.ServersCommunity++
				continue
			}
			communityServers = append(communityServers, server)
		}
		app.masterServerListMu.Lock()
		app.masterServerList = communityServers
		app.masterServerListMu.Unlock()
		prevStats = stats
		log.WithFields(log.Fields{
			"community": fmt.Sprintf("%d/%d", stats.ServersCommunity, stats.ServersTotal),
			"players":   stats.Players,
			"bots":      stats.Bots,
			"servers":   fmt.Sprintf("%d/%d/%d", stats.CapacityEmpty, stats.CapacityPartial, stats.CapacityFull),
		}).Tracef("Updated master server list")
		return nil
	}
	build()
	_ = update()
	updateTicker := time.NewTicker(updateFreq)
	// Fetch new stats every 30 minutes
	saveTicker, errSaveTicker := cronticker.NewTicker("0 */30 * * * *")
	if errSaveTicker != nil {
		log.WithError(errSaveTicker).Panicf("Invalid save ticker cron format")
		return
	}
	// Rebuild stats every hour
	buildTicker, errBuildTicker := cronticker.NewTicker("0 * * * * *")
	if errBuildTicker != nil {
		log.WithError(errBuildTicker).Panicf("Invalid build ticker cron format")
		return
	}
	for {
		select {
		case <-updateTicker.C:
			if errUpdate := update(); errUpdate != nil {
				log.WithError(errUpdate).Error("Failed to update master server state")
			}
		case <-buildTicker.C:
			build()
		case saveTime := <-saveTicker.C:
			prevStats.CreatedOn = saveTime
			if errSave := database.SaveGlobalTF2Stats(ctx, store.Live, prevStats); errSave != nil {
				log.WithError(errSave).Error("Failed to save global stats state")
				continue
			}
		case <-ctx.Done():
			return
		}
	}
}
