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
	"go.uber.org/zap"
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

func (app *App) steamGroupMembershipUpdater() {
	var update = func() {
		localCtx, cancel := context.WithTimeout(app.ctx, time.Second*120)
		newMap := map[steamid.GID]steamid.Collection{}
		total := 0
		for _, gid := range config.General.BannedSteamGroupIds {
			members, errMembers := steamweb.GetGroupMembers(localCtx, gid)
			if errMembers != nil {
				app.logger.Warn("Failed to fetch group members", zap.Error(errMembers))
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
		app.logger.Debug("Updated group member ban list", zap.Int("count", total))
	}
	update()
	ticker := time.NewTicker(time.Minute * 15)
	for {
		select {
		case <-ticker.C:
			update()
		case <-app.ctx.Done():
			app.logger.Debug("steamGroupMembershipUpdater shutting down")
			return
		}
	}
}

func (app *App) showReportMeta() {
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
		reports, errReports := app.store.GetReports(app.ctx, store.AuthorQueryFilter{
			QueryFilter: store.QueryFilter{
				Limit: 0,
			},
		})
		if errReports != nil {
			app.logger.Error("failed to fetch reports for report metadata", zap.Error(errReports))
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

		addFieldInline(reportNotice, app.logger, "New", fmt.Sprintf(" %d", m.Open1Day))
		addFieldInline(reportNotice, app.logger, "Total Open", fmt.Sprintf(" %d", m.TotalOpen))
		addFieldInline(reportNotice, app.logger, "Total Closed", fmt.Sprintf(" %d", m.TotalClosed))
		addFieldInline(reportNotice, app.logger, ">1 Day", fmt.Sprintf(" %d", m.Open1Day))
		addFieldInline(reportNotice, app.logger, ">3 Days", fmt.Sprintf(" %d", m.Open3Days))
		addFieldInline(reportNotice, app.logger, ">1 Week", fmt.Sprintf(" %d", m.OpenWeek))
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
		case <-app.ctx.Done():
			app.logger.Debug("showReportMeta shutting down")
			return
		}
	}
}

func (app *App) demoCleaner() {
	ticker := time.NewTicker(time.Hour * 24)
	var update = func() {
		if err := app.store.FlushDemos(app.ctx); err != nil && !errors.Is(err, store.ErrNoResult) {
			app.logger.Error("Error pruning expired refresh tokens", zap.Error(err))
		}
		app.logger.Info("Old demos flushed")
	}
	update()
	for {
		select {
		case <-ticker.C:
			update()
		case <-app.ctx.Done():
			app.logger.Debug("profileUpdater shutting down")
			return
		}
	}
}

func (app *App) cleanupTasks() {
	ticker := time.NewTicker(time.Hour * 24)
	for {
		select {
		case <-ticker.C:
			if err := app.store.PrunePersonAuth(app.ctx); err != nil && !errors.Is(err, store.ErrNoResult) {
				app.logger.Error("Error pruning expired refresh tokens", zap.Error(err))
			}
		case <-app.ctx.Done():
			app.logger.Debug("profileUpdater shutting down")
			return
		}
	}
}

type notificationPayload struct {
	minPerms model.Privilege
	sids     steamid.Collection
	severity model.NotificationSeverity
	message  string
	link     string
}

func (app *App) notificationSender() {
	for {
		select {
		case notification := <-app.notificationChan:
			go func() {
				if errSend := app.sendNotification(notification); errSend != nil {
					app.logger.Error("Failed to send user notification", zap.Error(errSend))
				}
			}()
		case <-app.ctx.Done():
			return
		}
	}
}

// profileUpdater takes care of periodically querying the steam api for updates player summaries.
// The 100 oldest profiles are updated on each execution
func (app *App) profileUpdater() {
	var update = func() {
		localCtx, cancel := context.WithTimeout(app.ctx, time.Second*10)
		defer cancel()
		people, errGetExpired := app.store.GetExpiredProfiles(localCtx, 100)
		if errGetExpired != nil {
			app.logger.Error("Failed to get expired profiles", zap.Error(errGetExpired))
			return
		}
		if len(people) == 0 {
			return
		}
		var sids steamid.Collection
		for _, person := range people {
			sids = append(sids, person.SteamID)
		}
		summaries, errSummaries := steamweb.PlayerSummaries(sids)
		if errSummaries != nil {
			app.logger.Error("Failed to get player summaries", zap.Error(errSummaries))
			return
		}
		for _, summary := range summaries {
			// TODO batch update upserts
			sid, errSid := steamid.SID64FromString(summary.Steamid)
			if errSid != nil {
				app.logger.Error("Failed to parse steamid from webapi", zap.Error(errSid))
				continue
			}
			person := model.NewPerson(sid)
			if errGetPerson := app.store.GetOrCreatePersonBySteamID(localCtx, sid, &person); errGetPerson != nil {
				app.logger.Error("Failed to get person", zap.Error(errGetPerson))
				continue
			}
			person.PlayerSummary = &summary
			if errSavePerson := app.store.SavePerson(localCtx, &person); errSavePerson != nil {
				app.logger.Error("Failed to save person", zap.Error(errSavePerson))
				continue
			}
		}
	}
	update()
	ticker := time.NewTicker(time.Second * 60)
	for {
		select {
		case <-ticker.C:
			update()
		case <-app.ctx.Done():
			app.logger.Debug("profileUpdater shutting down")
			return
		}
	}
}

func (app *App) serverA2SStatusUpdater(updateFreq time.Duration) {
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
				newStatus, errA := query.A2SQueryServer(app.logger, server)
				if errA != nil {
					app.logger.Debug("Failed to update a2s status", zap.Error(errA))
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
		case <-app.ctx.Done():
			return
		case <-ticker.C:
			if errUpdate := updateStatus(app.ctx, app.store); errUpdate != nil {
				app.logger.Error("Error trying to updateStatus server status state", zap.Error(errUpdate))
			}
		}
	}
}

func (app *App) serverRCONStatusUpdater(updateFreq time.Duration) {
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
					app.logger.Error("Failed to query server status", zap.Error(queryErr))
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
		case <-app.ctx.Done():
			return
		case <-ticker.C:
			if errUpdate := updateStatus(app.ctx, app.store); errUpdate != nil {
				app.logger.Error("Error trying to updateStatus server status state", zap.Error(errUpdate))
			}
		}
	}
}

// serverStateRefresher periodically compiles and caches the current known db, rcon & a2s server state
// into a ServerState instance
func (app *App) serverStateRefresher(updateFreq time.Duration) {
	var refreshState = func() error {
		var newState model.ServerStateCollection
		servers, errServers := app.store.GetServers(app.ctx, false)
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
		case <-app.ctx.Done():
			return
		case <-ticker.C:
			if errUpdate := refreshState(); errUpdate != nil {
				app.logger.Error("Failed to refreshState server state: %v", zap.Error(errUpdate))
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
			app.logger.Error("Failed to refresh campaigns", zap.Error(errCampaigns))
			return
		}
		newPledges, newUsers, errPledges := thirdparty.PatreonGetPledges(app.patreon)
		if errPledges != nil {
			app.logger.Error("Failed to refresh pledges", zap.Error(errPledges))
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
		app.logger.Info("Patreon Updated", zap.Int("campaign_count", len(newCampaigns)),
			zap.Int("current_cents", cents), zap.Int("total_cents", totalCents))
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

// banSweeper periodically will query the database for expired bans and remove them.
func (app *App) banSweeper() {
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ticker.C:
			waitGroup := &sync.WaitGroup{}
			waitGroup.Add(3)
			go func() {
				defer waitGroup.Done()
				expiredBans, errExpiredBans := app.store.GetExpiredBans(app.ctx)
				if errExpiredBans != nil && !errors.Is(errExpiredBans, store.ErrNoResult) {
					app.logger.Error("Failed to get expired expiredBans", zap.Error(errExpiredBans))
				} else {
					for _, expiredBan := range expiredBans {
						if errDrop := app.store.DropBan(app.ctx, &expiredBan, false); errDrop != nil {
							app.logger.Error("Failed to drop expired expiredBan", zap.Error(errDrop))
						} else {
							banType := "Ban"
							if expiredBan.BanType == model.NoComm {
								banType = "Mute"
							}
							var person model.Person
							if errPerson := app.store.GetOrCreatePersonBySteamID(app.ctx, expiredBan.TargetId, &person); errPerson != nil {
								app.logger.Error("Failed to get expired person", zap.Error(errPerson))
								continue
							}
							name := person.PersonaName
							if name == "" {
								name = person.SteamID.String()
							}
							app.logger.Info("Ban expired", zap.String("type", banType),
								zap.String("reason", expiredBan.Reason.String()),
								zap.Int64("sid64", expiredBan.TargetId.Int64()), zap.String("name", name))
						}
					}
				}
			}()
			go func() {
				defer waitGroup.Done()
				expiredNetBans, errExpiredNetBans := app.store.GetExpiredNetBans(app.ctx)
				if errExpiredNetBans != nil && !errors.Is(errExpiredNetBans, store.ErrNoResult) {
					app.logger.Warn("Failed to get expired network bans", zap.Error(errExpiredNetBans))
				} else {
					for _, expiredNetBan := range expiredNetBans {
						if errDropBanNet := app.store.DropBanNet(app.ctx, &expiredNetBan); errDropBanNet != nil {
							app.logger.Error("Failed to drop expired network expiredNetBan", zap.Error(errDropBanNet))
						} else {
							app.logger.Info("CIDR ban expired", zap.String("cidr", expiredNetBan.String()))
						}
					}
				}
			}()
			go func() {
				defer waitGroup.Done()
				expiredASNBans, errExpiredASNBans := app.store.GetExpiredASNBans(app.ctx)
				if errExpiredASNBans != nil && !errors.Is(errExpiredASNBans, store.ErrNoResult) {
					app.logger.Error("Failed to get expired asn bans", zap.Error(errExpiredASNBans))
				} else {
					for _, expiredASNBan := range expiredASNBans {
						if errDropASN := app.store.DropBanASN(app.ctx, &expiredASNBan); errDropASN != nil {
							app.logger.Error("Failed to drop expired asn ban", zap.Error(errDropASN))
						} else {
							app.logger.Info("ASN ban expired", zap.Int64("ban_id", expiredASNBan.BanASNId))
						}
					}
				}
			}()
			waitGroup.Wait()
		case <-app.ctx.Done():
			app.logger.Debug("banSweeper shutting down")
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

func (app *App) localStatUpdater() {
	var build = func() {
		if errBuild := app.store.BuildLocalTF2Stats(app.ctx); errBuild != nil {
			app.logger.Error("Error building local stats", zap.Error(errBuild))
		}
	}
	saveTicker, errSaveTicker := cronticker.NewTicker("0 */5 * * * *")
	if errSaveTicker != nil {
		app.logger.Fatal("Invalid save ticker cron format", zap.Error(errSaveTicker))
		return
	}
	// Rebuild stats every hour
	buildTicker, errBuildTicker := cronticker.NewTicker("0 * * * * *")
	if errBuildTicker != nil {
		app.logger.Fatal("Invalid build ticker cron format", zap.Error(errBuildTicker))
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
			servers, errServers := app.store.GetServers(app.ctx, false)
			if errServers != nil {
				app.logger.Error("Failed to fetch servers to build local cache", zap.Error(errServers))
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
					app.logger.Error("Cannot find server name", zap.String("name", serverName))
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
			if errSave := app.store.SaveLocalTF2Stats(app.ctx, store.Live, stats); errSave != nil {
				app.logger.Error("Failed to save local stats state", zap.Error(errSave))
				continue
			}
		case <-app.ctx.Done():
			return
		}
	}
}

func (app *App) masterServerListUpdater(updateFreq time.Duration) {
	prevStats := model.NewGlobalTF2Stats()
	locationCache := map[string]ip2location.LatLong{}
	var build = func() {
		if errBuild := app.store.BuildGlobalTF2Stats(app.ctx); errBuild != nil {
			app.logger.Error("Error building stats", zap.Error(errBuild))
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
				if errLocation := app.store.GetLocationRecord(app.ctx, ip, &locRecord); errLocation != nil {
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
		return nil
	}
	build()
	_ = update()
	updateTicker := time.NewTicker(updateFreq)
	// Fetch new stats every 30 minutes
	saveTicker, errSaveTicker := cronticker.NewTicker("0 */30 * * * *")
	if errSaveTicker != nil {
		app.logger.Fatal("Invalid save ticker cron format", zap.Error(errSaveTicker))
		return
	}
	// Rebuild stats every hour
	buildTicker, errBuildTicker := cronticker.NewTicker("0 * * * * *")
	if errBuildTicker != nil {
		app.logger.Fatal("Invalid build ticker cron format", zap.Error(errBuildTicker))
		return
	}
	for {
		select {
		case <-updateTicker.C:
			if errUpdate := update(); errUpdate != nil {
				app.logger.Error("Failed to update master server state", zap.Error(errUpdate))
			}
		case <-buildTicker.C:
			build()
		case saveTime := <-saveTicker.C:
			prevStats.CreatedOn = saveTime
			if errSave := app.store.SaveGlobalTF2Stats(app.ctx, store.Live, prevStats); errSave != nil {
				app.logger.Error("Failed to save global stats state", zap.Error(errSave))
				continue
			}
		case <-app.ctx.Done():
			return
		}
	}
}
