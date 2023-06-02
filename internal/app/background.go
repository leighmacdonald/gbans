package app

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/krayzpipes/cronticker/cronticker"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"sync"
	"time"
)

func IsSteamGroupBanned(steamId steamid.SID64) bool {
	bannedGroupMembersMu.RLock()
	defer bannedGroupMembersMu.RUnlock()
	for _, groupMembers := range bannedGroupMembers {
		for _, member := range groupMembers {
			if steamId == member {
				return true
			}
		}
	}
	return false
}

func steamGroupMembershipUpdater(ctx context.Context) {
	var update = func() {
		localCtx, cancel := context.WithTimeout(ctx, time.Second*120)
		newMap := map[steamid.GID]steamid.Collection{}
		total := 0
		for _, gid := range config.General.BannedSteamGroupIds {
			members, errMembers := steamweb.GetGroupMembers(localCtx, gid)
			if errMembers != nil {
				logger.Warn("Failed to fetch group members", zap.Error(errMembers))
				cancel()
				continue
			}
			newMap[gid] = members
			total += len(members)
		}
		bannedGroupMembersMu.Lock()
		bannedGroupMembers = newMap
		bannedGroupMembersMu.Unlock()
		cancel()
		logger.Debug("Updated group member ban list", zap.Int("count", total))
	}
	update()
	ticker := time.NewTicker(time.Minute * 15)
	for {
		select {
		case <-ticker.C:
			update()
		case <-ctx.Done():
			logger.Debug("steamGroupMembershipUpdater shutting down")
			return
		}
	}
}

func showReportMeta(ctx context.Context) {
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
		reports, errReports := store.GetReports(ctx, store.AuthorQueryFilter{
			QueryFilter: store.QueryFilter{
				Limit: 0,
			},
		})
		if errReports != nil {
			logger.Error("failed to fetch reports for report metadata", zap.Error(errReports))
			return
		}
		now := config.Now()
		var m reportMeta
		for _, report := range reports {
			if report.ReportStatus == store.ClosedWithAction || report.ReportStatus == store.ClosedWithoutAction {
				m.TotalClosed++
				continue
			}
			m.TotalOpen++
			if report.ReportStatus == store.NeedMoreInfo {
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
			Color: int(discord.Green),
		}
		if m.OpenWeek > 0 {
			reportNotice.Color = int(discord.Red)
		} else if m.Open3Days > 0 {
			reportNotice.Color = int(discord.Orange)
		}
		reportNotice.Description = "Current Open Report Counts"

		discord.AddFieldInline(reportNotice, "New", fmt.Sprintf(" %d", m.Open1Day))
		discord.AddFieldInline(reportNotice, "Total Open", fmt.Sprintf(" %d", m.TotalOpen))
		discord.AddFieldInline(reportNotice, "Total Closed", fmt.Sprintf(" %d", m.TotalClosed))
		discord.AddFieldInline(reportNotice, ">1 Day", fmt.Sprintf(" %d", m.Open1Day))
		discord.AddFieldInline(reportNotice, ">3 Days", fmt.Sprintf(" %d", m.Open3Days))
		discord.AddFieldInline(reportNotice, ">1 Week", fmt.Sprintf(" %d", m.OpenWeek))
		discord.SendPayload(discord.Payload{ChannelId: config.Discord.ReportLogChannelId, Embed: reportNotice})
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
			logger.Debug("showReportMeta shutting down")
			return
		}
	}
}

func demoCleaner(ctx context.Context) {
	ticker := time.NewTicker(time.Hour * 24)
	var update = func() {
		if err := store.FlushDemos(ctx); err != nil && !errors.Is(err, store.ErrNoResult) {
			logger.Error("Error pruning expired refresh tokens", zap.Error(err))
		}
		logger.Info("Old demos flushed")
	}
	update()
	for {
		select {
		case <-ticker.C:
			update()
		case <-ctx.Done():
			logger.Debug("profileUpdater shutting down")
			return
		}
	}
}

func cleanupTasks(ctx context.Context) {
	ticker := time.NewTicker(time.Hour * 24)
	for {
		select {
		case <-ticker.C:
			if err := store.PrunePersonAuth(ctx); err != nil && !errors.Is(err, store.ErrNoResult) {
				logger.Error("Error pruning expired refresh tokens", zap.Error(err))
			}
		case <-ctx.Done():
			logger.Debug("profileUpdater shutting down")
			return
		}
	}
}

func notificationSender(ctx context.Context) {
	for {
		select {
		case notification := <-notificationChan:
			go func() {
				if errSend := sendNotification(ctx, notification); errSend != nil {
					logger.Error("Failed to send user notification", zap.Error(errSend))
				}
			}()
		case <-ctx.Done():
			return
		}
	}
}

// profileUpdater takes care of periodically querying the steam api for updates player summaries.
// The 100 oldest profiles are updated on each execution
func profileUpdater(ctx context.Context) {
	var update = func() {
		localCtx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()
		people, errGetExpired := store.GetExpiredProfiles(localCtx, 100)
		if errGetExpired != nil {
			logger.Error("Failed to get expired profiles", zap.Error(errGetExpired))
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
			logger.Error("Failed to get player summaries", zap.Error(errSummaries))
			return
		}
		for _, summary := range summaries {
			// TODO batch update upserts
			sid, errSid := steamid.SID64FromString(summary.Steamid)
			if errSid != nil {
				logger.Error("Failed to parse steamid from webapi", zap.Error(errSid))
				continue
			}
			person := store.NewPerson(sid)
			if errGetPerson := store.GetOrCreatePersonBySteamID(localCtx, sid, &person); errGetPerson != nil {
				logger.Error("Failed to get person", zap.Error(errGetPerson))
				continue
			}
			person.PlayerSummary = &summary
			if errSavePerson := store.SavePerson(localCtx, &person); errSavePerson != nil {
				logger.Error("Failed to save person", zap.Error(errSavePerson))
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
		case <-ctx.Done():
			logger.Debug("profileUpdater shutting down")
			return
		}
	}
}

func patreonUpdater(ctx context.Context) {
	updateTimer := time.NewTicker(time.Hour * 1)
	if patreonClient == nil {
		return
	}
	var update = func() {
		newCampaigns, errCampaigns := PatreonGetTiers(patreonClient)
		if errCampaigns != nil {
			logger.Error("Failed to refresh campaigns", zap.Error(errCampaigns))
			return
		}
		newPledges, _, errPledges := PatreonGetPledges(patreonClient)
		if errPledges != nil {
			logger.Error("Failed to refresh pledges", zap.Error(errPledges))
			return
		}
		patreonMu.Lock()
		patreonCampaigns = newCampaigns
		patreonPledges = newPledges
		//patreonUsers = newUsers
		patreonMu.Unlock()
		cents := 0
		totalCents := 0
		for _, p := range newPledges {
			cents += p.Attributes.AmountCents
			if p.Attributes.TotalHistoricalAmountCents != nil {
				totalCents += *p.Attributes.TotalHistoricalAmountCents
			}
		}
		logger.Info("Patreon Updated", zap.Int("campaign_count", len(newCampaigns)),
			zap.Int("current_cents", cents), zap.Int("total_cents", totalCents))
	}
	update()
	for {
		select {
		case <-updateTimer.C:
			update()
		case <-ctx.Done():
			return
		}
	}

}
func updateStateServerList(ctx context.Context) error {
	servers, errServers := store.GetServers(ctx, false)
	if errServers != nil {
		return errServers
	}
	var sc []*state.ServerConfig
	for _, server := range servers {
		sc = append(sc, state.NewServerConfig(logger.Named(fmt.Sprintf("state-%s", server.ServerNameShort)),
			server.ServerID, server.ServerNameLong, server.ServerNameShort, server.Addr(), server.RCON, server.Latitude,
			server.Longitude, server.Region, server.CC))
	}
	state.SetServers(sc)
	return nil
}

func stateUpdater(ctx context.Context, statusUpdateFreq time.Duration, materUpdateFreq time.Duration) {
	log := logger.Named("state")
	errChan := make(chan error)
	if errUpdate := updateStateServerList(ctx); errUpdate != nil {
		log.Error("Failed to update list", zap.Error(errUpdate))
	}

	if errStart := state.Start(ctx, statusUpdateFreq, materUpdateFreq, errChan); errStart != nil {
		logger.Error("start returned error", zap.Error(errStart))
	}
}

// banSweeper periodically will query the database for expired bans and remove them.
func banSweeper(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ticker.C:
			waitGroup := &sync.WaitGroup{}
			waitGroup.Add(3)
			go func() {
				defer waitGroup.Done()
				expiredBans, errExpiredBans := store.GetExpiredBans(ctx)
				if errExpiredBans != nil && !errors.Is(errExpiredBans, store.ErrNoResult) {
					logger.Error("Failed to get expired expiredBans", zap.Error(errExpiredBans))
				} else {
					for _, expiredBan := range expiredBans {
						if errDrop := store.DropBan(ctx, &expiredBan, false); errDrop != nil {
							logger.Error("Failed to drop expired expiredBan", zap.Error(errDrop))
						} else {
							banType := "Ban"
							if expiredBan.BanType == store.NoComm {
								banType = "Mute"
							}
							var person store.Person
							if errPerson := store.GetOrCreatePersonBySteamID(ctx, expiredBan.TargetId, &person); errPerson != nil {
								logger.Error("Failed to get expired person", zap.Error(errPerson))
								continue
							}
							name := person.PersonaName
							if name == "" {
								name = person.SteamID.String()
							}
							logger.Info("Ban expired", zap.String("type", banType),
								zap.String("reason", expiredBan.Reason.String()),
								zap.Int64("sid64", expiredBan.TargetId.Int64()), zap.String("name", name))
						}
					}
				}
			}()
			go func() {
				defer waitGroup.Done()
				expiredNetBans, errExpiredNetBans := store.GetExpiredNetBans(ctx)
				if errExpiredNetBans != nil && !errors.Is(errExpiredNetBans, store.ErrNoResult) {
					logger.Warn("Failed to get expired network bans", zap.Error(errExpiredNetBans))
				} else {
					for _, expiredNetBan := range expiredNetBans {
						if errDropBanNet := store.DropBanNet(ctx, &expiredNetBan); errDropBanNet != nil {
							logger.Error("Failed to drop expired network expiredNetBan", zap.Error(errDropBanNet))
						} else {
							logger.Info("CIDR ban expired", zap.String("cidr", expiredNetBan.String()))
						}
					}
				}
			}()
			go func() {
				defer waitGroup.Done()
				expiredASNBans, errExpiredASNBans := store.GetExpiredASNBans(ctx)
				if errExpiredASNBans != nil && !errors.Is(errExpiredASNBans, store.ErrNoResult) {
					logger.Error("Failed to get expired asn bans", zap.Error(errExpiredASNBans))
				} else {
					for _, expiredASNBan := range expiredASNBans {
						if errDropASN := store.DropBanASN(ctx, &expiredASNBan); errDropASN != nil {
							logger.Error("Failed to drop expired asn ban", zap.Error(errDropASN))
						} else {
							logger.Info("ASN ban expired", zap.Int64("ban_id", expiredASNBan.BanASNId))
						}
					}
				}
			}()
			waitGroup.Wait()
		case <-ctx.Done():
			logger.Debug("banSweeper shutting down")
			return
		}
	}
}

func localStatUpdater(ctx context.Context) {
	var build = func() {
		if errBuild := store.BuildLocalTF2Stats(ctx); errBuild != nil {
			logger.Error("Error building local stats", zap.Error(errBuild))
		}
	}
	saveTicker, errSaveTicker := cronticker.NewTicker("0 */5 * * * *")
	if errSaveTicker != nil {
		logger.Fatal("Invalid save ticker cron format", zap.Error(errSaveTicker))
		return
	}
	// Rebuild stats every hour
	buildTicker, errBuildTicker := cronticker.NewTicker("0 * * * * *")
	if errBuildTicker != nil {
		logger.Fatal("Invalid build ticker cron format", zap.Error(errBuildTicker))
		return
	}
	build()
	for {
		select {
		case <-buildTicker.C:
			build()
		case saveTime := <-saveTicker.C:
			stats := store.NewLocalTF2Stats()
			stats.CreatedOn = saveTime
			servers, errServers := store.GetServers(ctx, false)
			if errServers != nil {
				logger.Error("Failed to fetch servers to build local cache", zap.Error(errServers))
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
			serverStateMu.RLock()
			for _, ss := range serverState {
				sn := fmt.Sprintf("%s:%d", ss.Host, ss.Port)
				serverName, nameFound := serverNameMap[sn]
				if !nameFound {
					logger.Error("Cannot find server name", zap.String("name", serverName))
					continue
				}
				stats.Servers[serverName] = ss.PlayerCount
				stats.Players += ss.PlayerCount
				_, foundRegion := stats.Regions[ss.Region]
				if !foundRegion {
					stats.Regions[ss.Region] = 0
				}
				stats.Regions[ss.Region] += ss.PlayerCount

				mapType := state.GuessMapType(ss.Map)
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
			serverStateMu.RUnlock()
			if errSave := store.SaveLocalTF2Stats(ctx, store.Live, stats); errSave != nil {
				logger.Error("Failed to save local stats state", zap.Error(errSave))
				continue
			}
		case <-ctx.Done():
			return
		}
	}
}
