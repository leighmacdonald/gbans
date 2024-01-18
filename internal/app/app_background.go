package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// matchSummarizer is the central collection point for summarizing matches live from UDP log events.
func (app *App) matchSummarizer(ctx context.Context) {
	log := app.log.Named("matchSum")

	eventChan := make(chan logparse.ServerEvent)
	if errReg := app.eb.Consume(eventChan); errReg != nil {
		log.Error("logWriter Tried to register duplicate reader channel", zap.Error(errReg))
	}

	matches := map[int]*activeMatchContext{}

	for {
		select {
		case evt := <-eventChan:
			matchContext, exists := matches[evt.ServerID]

			if !exists {
				cancelCtx, cancel := context.WithCancel(ctx)
				matchContext = &activeMatchContext{
					match:          logparse.NewMatch(evt.ServerID, evt.ServerName),
					cancel:         cancel,
					log:            log.Named(evt.ServerName),
					incomingEvents: make(chan logparse.ServerEvent),
				}

				go matchContext.start(cancelCtx)

				app.matchUUIDMap.Set(evt.ServerID, matchContext.match.MatchID)

				matches[evt.ServerID] = matchContext
			}

			matchContext.incomingEvents <- evt

			switch evt.EventType {
			case logparse.WTeamFinalScore:
				matchContext.finalScores++
				if matchContext.finalScores < 2 {
					continue
				}

				fallthrough
			case logparse.LogStop:
				matchContext.cancel()

				state := app.state.current()
				server, found := state.byServerID(evt.ServerID)

				if found && server.Name != "" {
					matchContext.match.Title = server.Name
				}

				var fullServer model.Server
				if err := store.GetServer(ctx, app.db, evt.ServerID, &fullServer); err != nil {
					app.log.Error("Failed to load findMatch server",
						zap.Int("server", matchContext.match.ServerID), zap.Error(err))
					delete(matches, evt.ServerID)

					continue
				}

				if !fullServer.EnableStats {
					delete(matches, evt.ServerID)

					continue
				}

				if errSave := store.MatchSave(ctx, app.db, &matchContext.match, app.weaponMap); errSave != nil {
					if errors.Is(errSave, store.ErrInsufficientPlayers) {
						app.log.Warn("Failed to save findMatch",
							zap.Int("server", matchContext.match.ServerID), zap.Error(errSave))
					} else {
						app.log.Error("Failed to save findMatch",
							zap.Int("server", matchContext.match.ServerID), zap.Error(errSave))
					}

					delete(matches, evt.ServerID)

					continue
				}

				app.onMatchComplete(ctx, matchContext.match.MatchID)

				delete(matches, evt.ServerID)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (app *App) onMatchComplete(ctx context.Context, matchID uuid.UUID) {
	var result model.MatchResult
	if errResult := store.MatchGetByID(ctx, app.db, matchID, &result); errResult != nil {
		app.log.Error("Failed to load findMatch", zap.Error(errResult))

		return
	}

	conf := app.config()

	app.discord.SendPayload(discord.Payload{
		ChannelID: conf.Discord.PublicMatchLogChannelID,
		Embed:     app.genDiscordMatchEmbed(result).MessageEmbed,
	})
}

func (app *App) updateSteamBanMembers(ctx context.Context) (map[int64]steamid.Collection, error) {
	newMap := map[int64]steamid.Collection{}

	localCtx, cancel := context.WithTimeout(ctx, time.Second*120)
	defer cancel()

	opts := store.SteamBansQueryFilter{
		BansQueryFilter:    store.BansQueryFilter{QueryFilter: store.QueryFilter{Deleted: false}},
		IncludeFriendsOnly: true,
	}

	steamBans, _, errSteam := store.GetBansSteam(ctx, app.db, opts)
	if errSteam != nil {
		if errors.Is(errSteam, store.ErrNoResult) {
			return newMap, nil
		}

		return nil, errors.Wrap(errSteam, "Failed to fetch bans with friends included")
	}

	for _, steamBan := range steamBans {
		friends, errFriends := steamweb.GetFriendList(localCtx, steamBan.TargetID)
		if errFriends != nil {
			return nil, errors.Wrap(errFriends, "Failed to fetch friends")
		}

		if len(friends) == 0 {
			continue
		}

		var sids steamid.Collection

		for _, friend := range friends {
			sids = append(sids, friend.SteamID)
		}

		memberList := model.NewMembersList(steamBan.TargetID.Int64(), sids)
		if errQuery := store.GetMembersList(ctx, app.db, steamBan.TargetID.Int64(), &memberList); errQuery != nil {
			if !errors.Is(errQuery, store.ErrNoResult) {
				return nil, errors.Wrap(errQuery, "Failed to fetch members list")
			}
		}

		if errSave := store.SaveMembersList(ctx, app.db, &memberList); errSave != nil {
			return nil, errors.Wrap(errSave, "Failed to save banned steam friend member list")
		}

		newMap[steamBan.TargetID.Int64()] = memberList.Members
	}

	return newMap, nil
}

// todo move to separate struct.
func (app *App) updateBanChildren(ctx context.Context) {
	var (
		newMap = map[int64]steamid.Collection{}
		total  = 0
	)

	friendEntries, errFriendEntries := app.updateSteamBanMembers(ctx)
	if errFriendEntries == nil {
		for k, v := range friendEntries {
			total += len(v)
			newMap[k] = v
		}
	}

	app.log.Debug("Updated friend list member bans",
		zap.Int("friends", len(friendEntries)))
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
			reports, _, errReports := store.GetReports(ctx, app.db, store.ReportQueryFilter{
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
				if report.ReportStatus == model.ClosedWithAction || report.ReportStatus == model.ClosedWithoutAction {
					meta.TotalClosed++

					continue
				}

				meta.TotalOpen++

				if report.ReportStatus == model.NeedMoreInfo {
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

			conf := app.config()

			msgEmbed := discord.NewEmbed(conf, "User Report Stats")
			msgEmbed.
				Embed().
				SetColor(conf.Discord.ColourSuccess).
				SetURL(conf.ExtURLRaw("/admin/reports"))

			if meta.OpenWeek > 0 {
				msgEmbed.Embed().SetColor(conf.Discord.ColourError)
			} else if meta.Open3Days > 0 {
				msgEmbed.Embed().SetColor(conf.Discord.ColourWarn)
			}

			msgEmbed.Embed().
				SetDescription("Current Open Report Counts").
				AddField("New", fmt.Sprintf(" %d", meta.Open1Day)).MakeFieldInline().
				AddField("Total Open", fmt.Sprintf(" %d", meta.TotalOpen)).MakeFieldInline().
				AddField("Total Closed", fmt.Sprintf(" %d", meta.TotalClosed)).MakeFieldInline().
				AddField(">1 Day", fmt.Sprintf(" %d", meta.Open1Day)).MakeFieldInline().
				AddField(">3 Days", fmt.Sprintf(" %d", meta.Open3Days)).MakeFieldInline().
				AddField(">1 Week", fmt.Sprintf(" %d", meta.OpenWeek)).MakeFieldInline()

			app.discord.SendPayload(discord.Payload{
				ChannelID: conf.Discord.LogChannelID,
				Embed:     msgEmbed.Embed().Truncate().MessageEmbed,
			})
		case <-ctx.Done():
			app.log.Debug("showReportMeta shutting down")

			return
		}
	}
}

func (app *App) demoCleaner(ctx context.Context) {
	log := app.log.Named("demoCleaner")
	ticker := time.NewTicker(time.Hour)
	triggerChan := make(chan any)

	go func() {
		triggerChan <- true
	}()

	for {
		select {
		case <-ticker.C:
			triggerChan <- true
		case <-triggerChan:
			conf := app.config()

			if !conf.General.DemoCleanupEnabled {
				continue
			}

			log.Debug("Starting demo cleanup")

			expired, errExpired := store.ExpiredDemos(ctx, app.db, conf.General.DemoCountLimit)
			if errExpired != nil {
				if errors.Is(errExpired, store.ErrNoResult) {
					continue
				}

				log.Error("Failed to fetch expired demos", zap.Error(errExpired))
			}

			if len(expired) == 0 {
				continue
			}

			count := 0

			for _, demo := range expired {
				if errRemove := app.assetStore.Remove(ctx, conf.S3.BucketDemo, demo.Title); errRemove != nil {
					log.Error("Failed to remove demo asset from S3",
						zap.Error(errRemove), zap.String("bucket", conf.S3.BucketDemo), zap.String("name", demo.Title))

					continue
				}

				if errDrop := store.DropDemo(ctx, app.db, &model.DemoFile{DemoID: demo.DemoID, Title: demo.Title}); errDrop != nil {
					log.Error("Failed to remove demo", zap.Error(errDrop),
						zap.String("bucket", conf.S3.BucketDemo), zap.String("name", demo.Title))

					continue
				}

				log.Info("Demo expired and removed",
					zap.String("bucket", conf.S3.BucketDemo), zap.String("name", demo.Title))
				count++
			}

			log.Info("Old demos flushed", zap.Int("count", count))
		case <-ctx.Done():
			log.Debug("demoCleaner shutting down")

			return
		}
	}
}

func cleanupTasks(ctx context.Context, database store.Store, logger *zap.Logger) {
	var (
		log    = logger.Named("cleanupTasks")
		ticker = time.NewTicker(time.Hour * 24)
	)

	for {
		select {
		case <-ticker.C:
			if err := store.PrunePersonAuth(ctx, database); err != nil && !errors.Is(err, store.ErrNoResult) {
				log.Error("Error pruning expired refresh tokens", zap.Error(err))
			}
		case <-ctx.Done():
			log.Debug("cleanupTasks shutting down")

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

func (app *App) updateProfiles(ctx context.Context, people model.People) error {
	if len(people) > 100 {
		return errors.New("100 people max per call")
	}

	var (
		banStates           []steamweb.PlayerBanState
		summaries           []steamweb.PlayerSummary
		steamIDs            = people.ToSteamIDCollection()
		errGroup, cancelCtx = errgroup.WithContext(ctx)
	)

	errGroup.Go(func() error {
		newBanStates, errBans := thirdparty.FetchPlayerBans(cancelCtx, steamIDs)
		if errBans != nil {
			return errors.Wrap(errBans, "Failed to fetch ban status from steamapi")
		}

		banStates = newBanStates

		return nil
	})

	errGroup.Go(func() error {
		newSummaries, errSummaries := steamweb.PlayerSummaries(cancelCtx, steamIDs)
		if errSummaries != nil {
			return errors.Wrap(errSummaries, "Failed to fetch player summaries from steamapi")
		}

		summaries = newSummaries

		return nil
	})

	if errFetch := errGroup.Wait(); errFetch != nil {
		return errors.Wrap(errFetch, "Failed to fetch data from steamapi")
	}

	for _, curPerson := range people {
		person := curPerson
		person.IsNew = false
		person.UpdatedOnSteam = time.Now()

		for _, newSummary := range summaries {
			summary := newSummary
			if person.SteamID != summary.SteamID {
				continue
			}

			person.PlayerSummary = &summary

			break
		}

		for _, banState := range banStates {
			if person.SteamID != banState.SteamID {
				continue
			}

			person.CommunityBanned = banState.CommunityBanned
			person.VACBans = banState.NumberOfVACBans
			person.GameBans = banState.NumberOfGameBans
			person.EconomyBan = banState.EconomyBan
			person.CommunityBanned = banState.CommunityBanned
			person.DaysSinceLastBan = banState.DaysSinceLastBan
		}

		if errSavePerson := store.SavePerson(ctx, app.db, &person); errSavePerson != nil {
			return errors.Wrap(errSavePerson, "Failed to save person")
		}
	}

	app.log.Debug("Updated steam profiles and vac data", zap.Int("count", len(people)))

	return nil
}

// profileUpdater takes care of periodically querying the steam api for updates player summaries.
// The 100 oldest profiles are updated on each execution.
func (app *App) profileUpdater(ctx context.Context) {
	var (
		log    = app.log.Named("profileUpdate")
		run    = make(chan any)
		ticker = time.NewTicker(time.Second * 300)
	)

	go func() {
		run <- true
	}()

	for {
		select {
		case <-ticker.C:
			run <- true
		case <-run:
			localCtx, cancel := context.WithTimeout(ctx, time.Second*10)
			people, errGetExpired := store.GetExpiredProfiles(localCtx, app.db, 100)

			if errGetExpired != nil || len(people) == 0 {
				cancel()

				continue
			}

			if errUpdate := app.updateProfiles(localCtx, people); errUpdate != nil {
				log.Error("Failed to update profiles", zap.Error(errUpdate))
			}

			cancel()
		case <-ctx.Done():
			log.Debug("profileUpdater shutting down")

			return
		}
	}
}

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
			servers, _, errServers := store.GetServers(ctx, app.db, store.ServerQueryFilter{
				QueryFilter:     store.QueryFilter{Deleted: false},
				IncludeDisabled: false,
			})
			if errServers != nil && !errors.Is(errServers, store.ErrNoResult) {
				log.Error("Failed to fetch servers, cannot update state", zap.Error(errServers))

				continue
			}

			var configs []serverConfig
			for _, server := range servers {
				configs = append(configs, newServerConfig(
					server.ServerID,
					server.ShortName,
					server.Name,
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

			conf := app.config()
			if conf.Debug.AddRCONLogAddress != "" {
				app.state.logAddressAdd(conf.Debug.AddRCONLogAddress)
			}

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

				expiredBans, errExpiredBans := store.GetExpiredBans(ctx, app.db)
				if errExpiredBans != nil && !errors.Is(errExpiredBans, store.ErrNoResult) {
					log.Error("Failed to get expired expiredBans", zap.Error(errExpiredBans))
				} else {
					for _, expiredBan := range expiredBans {
						ban := expiredBan
						if errDrop := store.DropBan(ctx, app.db, &ban, false); errDrop != nil {
							log.Error("Failed to drop expired expiredBan", zap.Error(errDrop))
						} else {
							banType := "Ban"
							if ban.BanType == model.NoComm {
								banType = "Mute"
							}

							var person model.Person
							if errPerson := store.GetPersonBySteamID(ctx, app.db, ban.TargetID, &person); errPerson != nil {
								log.Error("Failed to get expired person", zap.Error(errPerson))

								continue
							}

							name := person.PersonaName
							if name == "" {
								name = person.SteamID.String()
							}

							conf := app.config()

							msgEmbed := discord.NewEmbed(conf, "Steam Ban Expired")
							msgEmbed.
								Embed().
								SetColor(conf.Discord.ColourInfo).
								AddField("Type", banType).
								SetImage(person.AvatarFull).
								AddField("Name", person.PersonaName).
								SetURL(conf.ExtURL(ban))

							msgEmbed.AddFieldsSteamID(person.SteamID)

							if expiredBan.BanType == model.NoComm {
								msgEmbed.Embed().SetColor(conf.Discord.ColourWarn)
							}

							app.bot().SendPayload(discord.Payload{
								ChannelID: conf.Discord.LogChannelID,
								Embed:     msgEmbed.Embed().Truncate().MessageEmbed,
							})

							log.Info("Ban expired", zap.String("type", banType),
								zap.String("reason", ban.Reason.String()),
								zap.Int64("sid64", ban.TargetID.Int64()), zap.String("name", name))
						}
					}
				}
			}()

			go func() {
				defer waitGroup.Done()

				expiredNetBans, errExpiredNetBans := store.GetExpiredNetBans(ctx, app.db)
				if errExpiredNetBans != nil && !errors.Is(errExpiredNetBans, store.ErrNoResult) {
					log.Warn("Failed to get expired network bans", zap.Error(errExpiredNetBans))
				} else {
					for _, expiredNetBan := range expiredNetBans {
						expiredBan := expiredNetBan
						if errDropBanNet := store.DropBanNet(ctx, app.db, &expiredBan); errDropBanNet != nil {
							log.Error("Failed to drop expired network expiredNetBan", zap.Error(errDropBanNet))
						} else {
							log.Info("IP ban expired", zap.String("cidr", expiredBan.String()))
						}
					}
				}
			}()

			go func() {
				defer waitGroup.Done()

				expiredASNBans, errExpiredASNBans := store.GetExpiredASNBans(ctx, app.db)
				if errExpiredASNBans != nil && !errors.Is(errExpiredASNBans, store.ErrNoResult) {
					log.Error("Failed to get expired asn bans", zap.Error(errExpiredASNBans))
				} else {
					for _, expiredASNBan := range expiredASNBans {
						expired := expiredASNBan
						if errDropASN := store.DropBanASN(ctx, app.db, &expired); errDropASN != nil {
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
