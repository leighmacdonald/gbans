package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/krayzpipes/cronticker/cronticker"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (app *App) IsSteamGroupBanned(steamID steamid.SID64) bool {
	app.bannedGroupMembersMu.RLock()
	defer app.bannedGroupMembersMu.RUnlock()

	for _, groupMembers := range app.bannedGroupMembers {
		for _, member := range groupMembers {
			if steamID == member {
				return true
			}
		}
	}

	return false
}

func (app *App) steamGroupMembershipUpdater(ctx context.Context) {
	log := app.log.Named("steamGroupMembership")
	ticker := time.NewTicker(time.Minute * 15)
	updateChan := make(chan any)

	go func() {
		updateChan <- true
	}()

	for {
		select {
		case <-ticker.C:
			updateChan <- true
		case <-updateChan:
			localCtx, cancel := context.WithTimeout(ctx, time.Second*120)
			newMap := map[steamid.GID]steamid.Collection{}
			total := 0

			for _, gid := range app.conf.General.BannedSteamGroupIds {
				members, errMembers := steamweb.GetGroupMembers(localCtx, gid)
				if errMembers != nil {
					log.Warn("Failed to fetch group members", zap.Error(errMembers))
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
			log.Debug("Updated group member ban list", zap.Int("count", total))
		case <-ctx.Done():
			log.Debug("steamGroupMembershipUpdater shutting down")

			return
		}
	}
}

func (app *App) showReportMeta(ctx context.Context) {
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

	ticker := time.NewTicker(time.Hour * 24)
	updateChan := make(chan any)

	go func() {
		time.Sleep(time.Second * 20)
		updateChan <- true
	}()

	for {
		select {
		case <-ticker.C:
			updateChan <- true
		case <-updateChan:
			reports, errReports := app.db.GetReports(ctx, store.AuthorQueryFilter{
				QueryFilter: store.QueryFilter{
					Limit: 0,
				},
			})
			if errReports != nil {
				app.log.Error("failed to fetch reports for report metadata", zap.Error(errReports))

				continue
			}

			var (
				now  = time.Now()
				meta reportMeta
			)

			for _, report := range reports {
				if report.ReportStatus == store.ClosedWithAction || report.ReportStatus == store.ClosedWithoutAction {
					meta.TotalClosed++

					continue
				}

				meta.TotalOpen++

				if report.ReportStatus == store.NeedMoreInfo {
					meta.NeedInfo++
				} else {
					meta.Open++
				}

				switch {
				case now.Sub(report.CreatedOn) > time.Hour*24*7:
					meta.OpenWeek++
				case now.Sub(report.CreatedOn) > time.Hour*24*3:
					meta.Open3Days++
				case now.Sub(report.CreatedOn) > time.Hour*24:
					meta.Open1Day++
				default:
					meta.OpenNew++
				}
			}

			msgEmbed := discord.
				NewEmbed("User Report Stats").
				SetColor(app.bot.Colour.Success).
				SetURL(app.ExtURLRaw("/admin/reports"))

			if meta.OpenWeek > 0 {
				msgEmbed.SetColor(app.bot.Colour.Error)
			} else if meta.Open3Days > 0 {
				msgEmbed.SetColor(app.bot.Colour.Warn)
			}

			msgEmbed.SetDescription("Current Open Report Counts")

			msgEmbed.AddField("New", fmt.Sprintf(" %d", meta.Open1Day)).MakeFieldInline()
			msgEmbed.AddField("Total Open", fmt.Sprintf(" %d", meta.TotalOpen)).MakeFieldInline()
			msgEmbed.AddField("Total Closed", fmt.Sprintf(" %d", meta.TotalClosed)).MakeFieldInline()
			msgEmbed.AddField(">1 Day", fmt.Sprintf(" %d", meta.Open1Day)).MakeFieldInline()
			msgEmbed.AddField(">3 Days", fmt.Sprintf(" %d", meta.Open3Days)).MakeFieldInline()
			msgEmbed.AddField(">1 Week", fmt.Sprintf(" %d", meta.OpenWeek)).MakeFieldInline()

			app.bot.SendPayload(discord.Payload{ChannelID: app.conf.Discord.LogChannelID, Embed: msgEmbed.Truncate().MessageEmbed})
		case <-ctx.Done():
			app.log.Debug("showReportMeta shutting down")

			return
		}
	}
}

func demoCleaner(ctx context.Context, database *store.Store, logger *zap.Logger) {
	var (
		log         = logger.Named("demoCleaner")
		ticker      = time.NewTicker(time.Hour * 24)
		triggerChan = make(chan any)
	)

	defer func() {
		triggerChan <- true
	}()

	for {
		select {
		case <-ticker.C:
			triggerChan <- true
		case <-triggerChan:
			if err := database.FlushDemos(ctx); err != nil && !errors.Is(err, store.ErrNoResult) {
				log.Error("Error pruning expired refresh tokens", zap.Error(err))
			}

			log.Info("Old demos flushed")
		case <-ctx.Done():
			log.Debug("profileUpdater shutting down")

			return
		}
	}
}

func cleanupTasks(ctx context.Context, database *store.Store, logger *zap.Logger) {
	var (
		log    = logger.Named("cleanupTasks")
		ticker = time.NewTicker(time.Hour * 24)
	)

	for {
		select {
		case <-ticker.C:
			if err := database.PrunePersonAuth(ctx); err != nil && !errors.Is(err, store.ErrNoResult) {
				log.Error("Error pruning expired refresh tokens", zap.Error(err))
			}
		case <-ctx.Done():
			log.Debug("profileUpdater shutting down")

			return
		}
	}
}

func (app *App) notificationSender(ctx context.Context) {
	log := app.log.Named("notificationSender")

	for {
		select {
		case notification := <-app.notificationChan:
			go func() {
				if errSend := app.SendNotification(ctx, notification); errSend != nil {
					log.Error("Failed to send user notification", zap.Error(errSend))
				}
			}()
		case <-ctx.Done():
			return
		}
	}
}

// profileUpdater takes care of periodically querying the steam api for updates player summaries.
// The 100 oldest profiles are updated on each execution
// func profileUpdater(ctx context.Context) {
//	var update = func() {
//		localCtx, cancel := context.WithTimeout(ctx, time.Second*10)
//		defer cancel()
//		people, errGetExpired := store.GetExpiredProfiles(localCtx, 100)
//		if errGetExpired != nil {
//			logger.Error("Failed to get expired profiles", zap.Error(errGetExpired))
//			return
//		}
//		if len(people) == 0 {
//			return
//		}
//		var sids steamid.Collection
//		for _, person := range people {
//			sids = append(sids, person.SteamID)
//		}
//		summaries, errSummaries := steamweb.PlayerSummaries(sids)
//		if errSummaries != nil {
//			logger.Error("Failed to get player summaries", zap.Error(errSummaries))
//			return
//		}
//		for _, summary := range summaries {
//			// TODO batch update upserts
//			sid, errSid := steamid.SID64FromString(summary.Steamid)
//			if errSid != nil {
//				logger.Error("Failed to parse steamid from webapi", zap.Error(errSid))
//				continue
//			}
//			person := store.NewPerson(sid)
//			if errGetPerson := store.GetOrCreatePersonBySteamID(localCtx, sid, &person); errGetPerson != nil {
//				logger.Error("Failed to get person", zap.Error(errGetPerson))
//				continue
//			}
//			person.PlayerSummary = &summary
//			person.UpdatedOnSteam = config.Now()
//			if errSavePerson := store.SavePerson(localCtx, &person); errSavePerson != nil {
//				logger.Error("Failed to save person", zap.Error(errSavePerson))
//				continue
//			}
//		}
//	}
//	update()
//	ticker := time.NewTicker(time.Second * 60)
//	for {
//		select {
//		case <-ticker.C:
//			update()
//		case <-ctx.Done():
//			logger.Debug("profileUpdater shutting down")
//			return
//		}
//	}
// }

func (app *App) stateUpdater(ctx context.Context) {
	var (
		log          = app.log.Named("state")
		trigger      = make(chan any)
		updateTicker = time.NewTicker(time.Minute * 30)
	)

	go app.state.start(ctx)

	go func() {
		trigger <- true
	}()

	for {
		select {
		case <-updateTicker.C:
			trigger <- true
		case <-trigger:
			servers, errServers := app.db.GetServers(ctx, false)
			if errServers != nil {
				log.Error("Failed to fetch servers, cannot update state", zap.Error(errServers))

				continue
			}

			var configs []serverConfig
			for _, server := range servers {
				configs = append(configs, newServerConfig(
					server.ServerID,
					server.ServerName,
					server.ServerNameLong,
					server.Address,
					server.Port,
					server.RCON,
					server.ReservedSlots,
					server.CC,
					server.Region,
					server.Latitude,
					server.Longitude,
				))
			}

			app.state.setServerConfigs(configs)
		case <-ctx.Done():
			return
		}
	}
}

// banSweeper periodically will query the database for expired bans and remove them.
func (app *App) banSweeper(ctx context.Context) {
	var (
		log    = app.log.Named("banSweeper")
		ticker = time.NewTicker(time.Minute)
	)

	for {
		select {
		case <-ticker.C:
			waitGroup := &sync.WaitGroup{}
			waitGroup.Add(3)

			go func() {
				defer waitGroup.Done()

				expiredBans, errExpiredBans := app.db.GetExpiredBans(ctx)
				if errExpiredBans != nil && !errors.Is(errExpiredBans, store.ErrNoResult) {
					log.Error("Failed to get expired expiredBans", zap.Error(errExpiredBans))
				} else {
					for _, expiredBan := range expiredBans {
						ban := expiredBan
						if errDrop := app.db.DropBan(ctx, &ban, false); errDrop != nil {
							log.Error("Failed to drop expired expiredBan", zap.Error(errDrop))
						} else {
							banType := "Ban"
							if ban.BanType == store.NoComm {
								banType = "Mute"
							}

							var person store.Person
							if errPerson := app.PersonBySID(ctx, ban.TargetID, &person); errPerson != nil {
								log.Error("Failed to get expired person", zap.Error(errPerson))

								continue
							}

							name := person.PersonaName
							if name == "" {
								name = person.SteamID.String()
							}

							msgEmbed := discord.
								NewEmbed("Steam Ban Expired").
								SetColor(app.bot.Colour.Info).
								AddField("Type", banType).
								SetImage(person.AvatarFull).
								AddField("Name", person.PersonaName).
								SetURL(app.ExtURL(ban))

							discord.AddFieldsSteamID(msgEmbed, person.SteamID)

							if expiredBan.BanType == store.NoComm {
								msgEmbed.SetColor(app.bot.Colour.Warn)
							}

							app.bot.SendPayload(discord.Payload{
								ChannelID: app.conf.Discord.LogChannelID,
								Embed:     msgEmbed.Truncate().MessageEmbed,
							})

							log.Info("Ban expired", zap.String("type", banType),
								zap.String("reason", store.ReasonString(ban.Reason)),
								zap.Int64("sid64", ban.TargetID.Int64()), zap.String("name", name))
						}
					}
				}
			}()
			go func() {
				defer waitGroup.Done()

				expiredNetBans, errExpiredNetBans := app.db.GetExpiredNetBans(ctx)
				if errExpiredNetBans != nil && !errors.Is(errExpiredNetBans, store.ErrNoResult) {
					log.Warn("Failed to get expired network bans", zap.Error(errExpiredNetBans))
				} else {
					for _, expiredNetBan := range expiredNetBans {
						expiredBan := expiredNetBan
						if errDropBanNet := app.db.DropBanNet(ctx, &expiredBan); errDropBanNet != nil {
							log.Error("Failed to drop expired network expiredNetBan", zap.Error(errDropBanNet))
						} else {
							log.Info("CIDR ban expired", zap.String("cidr", expiredBan.String()))
						}
					}
				}
			}()
			go func() {
				defer waitGroup.Done()

				expiredASNBans, errExpiredASNBans := app.db.GetExpiredASNBans(ctx)
				if errExpiredASNBans != nil && !errors.Is(errExpiredASNBans, store.ErrNoResult) {
					log.Error("Failed to get expired asn bans", zap.Error(errExpiredASNBans))
				} else {
					for _, expiredASNBan := range expiredASNBans {
						expired := expiredASNBan
						if errDropASN := app.db.DropBanASN(ctx, &expired); errDropASN != nil {
							log.Error("Failed to drop expired asn ban", zap.Error(errDropASN))
						} else {
							log.Info("ASN ban expired", zap.Int64("ban_id", expired.BanASNId))
						}
					}
				}
			}()
			waitGroup.Wait()
		case <-ctx.Done():
			log.Debug("banSweeper shutting down")

			return
		}
	}
}

func (app *App) localStatUpdater(ctx context.Context) {
	log := app.log.Named("localStatUpdater")
	build := func() {
		if errBuild := app.db.BuildLocalTF2Stats(ctx); errBuild != nil {
			log.Error("Error building local stats", zap.Error(errBuild))
		}
	}

	saveTicker, errSaveTicker := cronticker.NewTicker("0 */5 * * * *")
	if errSaveTicker != nil {
		log.Fatal("Invalid save ticker cron format", zap.Error(errSaveTicker))

		return
	}

	// Rebuild stats every hour
	buildTicker, errBuildTicker := cronticker.NewTicker("0 * * * * *")
	if errBuildTicker != nil {
		log.Fatal("Invalid build ticker cron format", zap.Error(errBuildTicker))

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

			servers, errServers := app.db.GetServers(ctx, false)
			if errServers != nil {
				log.Error("Failed to fetch servers to build local cache", zap.Error(errServers))

				continue
			}

			serverNameMap := map[string]string{}
			for _, server := range servers {
				serverNameMap[fmt.Sprintf("%s:%d", server.Address, server.Port)] = server.ServerName

				ipAddr, errIP := server.IP(ctx)
				if errIP != nil {
					continue
				}

				serverNameMap[fmt.Sprintf("%s:%d", ipAddr.String(), server.Port)] = server.ServerName
			}

			currentState := app.state.current()
			for _, curState := range currentState {
				sn := fmt.Sprintf("%s:%d", curState.Host, curState.Port)

				serverName, nameFound := serverNameMap[sn]
				if !nameFound {
					log.Error("Cannot find server name", zap.String("name", serverName))

					continue
				}

				stats.Servers[serverName] = curState.PlayerCount
				stats.Players += curState.PlayerCount

				_, foundRegion := stats.Regions[curState.Region]
				if !foundRegion {
					stats.Regions[curState.Region] = 0
				}

				stats.Regions[curState.Region] += curState.PlayerCount

				mapType := guessMapType(curState.Map)

				_, mapTypeFound := stats.MapTypes[mapType]
				if !mapTypeFound {
					stats.MapTypes[mapType] = 0
				}

				stats.MapTypes[mapType] += curState.PlayerCount

				switch {
				case curState.PlayerCount >= curState.MaxPlayers && curState.MaxPlayers > 0:
					stats.CapacityFull++
				case curState.PlayerCount == 0:
					stats.CapacityEmpty++
				default:
					stats.CapacityPartial++
				}
			}

			if errSave := app.db.SaveLocalTF2Stats(ctx, store.Live, stats); errSave != nil {
				log.Error("Failed to save local stats state", zap.Error(errSave))

				continue
			}
		case <-ctx.Done():
			return
		}
	}
}
