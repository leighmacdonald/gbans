package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/discord"
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

				var fullServer store.Server
				if err := app.db.GetServer(ctx, evt.ServerID, &fullServer); err != nil {
					app.log.Error("Failed to load match server",
						zap.Int("server", matchContext.match.ServerID), zap.Error(err))
					delete(matches, evt.ServerID)

					continue
				}

				if !fullServer.EnableStats {
					delete(matches, evt.ServerID)

					continue
				}

				if errSave := app.db.MatchSave(ctx, &matchContext.match); errSave != nil {
					app.log.Error("Failed to save match",
						zap.Int("server", matchContext.match.ServerID), zap.Error(errSave))

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
	var result store.MatchResult
	if errResult := app.db.MatchGetByID(ctx, matchID, &result); errResult != nil {
		app.log.Error("Failed to load match", zap.Error(errResult))

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

	steamBans, _, errSteam := app.db.GetBansSteam(ctx, opts)
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

		memberList := store.NewMembersList(steamBan.TargetID.Int64(), sids)
		if errQuery := app.db.GetMembersList(ctx, steamBan.TargetID.Int64(), &memberList); errQuery != nil {
			if !errors.Is(errQuery, store.ErrNoResult) {
				return nil, errors.Wrap(errQuery, "Failed to fetch members list")
			}
		}

		if errSave := app.db.SaveMembersList(ctx, &memberList); errSave != nil {
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
			reports, _, errReports := app.db.GetReports(ctx, store.ReportQueryFilter{
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

			expired, errExpired := app.db.ExpiredDemos(ctx, conf.General.DemoCountLimit)
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

				if errDrop := app.db.DropDemo(ctx, &store.DemoFile{DemoID: demo.DemoID, Title: demo.Title}); errDrop != nil {
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

func (app *App) updateProfiles(ctx context.Context, people store.People) error {
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
		newBanStates, errBans := thirdparty.FetchPlayerBans(cancelCtx, app.log, steamIDs)
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

		if errSavePerson := app.db.SavePerson(ctx, &person); errSavePerson != nil {
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
			people, errGetExpired := app.db.GetExpiredProfiles(localCtx, 100)

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
			servers, _, errServers := app.db.GetServers(ctx, store.ServerQueryFilter{
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
			app.initLogAddress()
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

							if expiredBan.BanType == store.NoComm {
								msgEmbed.Embed().SetColor(conf.Discord.ColourWarn)
							}

							app.discord.SendPayload(discord.Payload{
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

				expiredNetBans, errExpiredNetBans := app.db.GetExpiredNetBans(ctx)
				if errExpiredNetBans != nil && !errors.Is(errExpiredNetBans, store.ErrNoResult) {
					log.Warn("Failed to get expired network bans", zap.Error(errExpiredNetBans))
				} else {
					for _, expiredNetBan := range expiredNetBans {
						expiredBan := expiredNetBan
						if errDropBanNet := app.db.DropBanNet(ctx, &expiredBan); errDropBanNet != nil {
							log.Error("Failed to drop expired network expiredNetBan", zap.Error(errDropBanNet))
						} else {
							log.Info("IP ban expired", zap.String("cidr", expiredBan.String()))
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

type forumActivity struct {
	person       userProfile
	lastActivity time.Time
}

func (activity forumActivity) expired() bool {
	return time.Since(activity.lastActivity) > time.Minute*5
}

func (app *App) touchPerson(person userProfile) {
	if !person.SteamID.Valid() {
		return
	}

	valid := []forumActivity{{lastActivity: time.Now(), person: person}}

	app.activityMu.Lock()
	defer app.activityMu.Unlock()

	for _, activity := range app.activity {
		if activity.person.SteamID == person.SteamID {
			continue
		}

		valid = append(valid, activity)
	}

	app.activity = valid
}

func (app *App) forumActivityUpdater(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 30)
	log := app.log.Named("forumActivityUpdater")

	for {
		select {
		case <-ticker.C:
			var current []forumActivity

			app.activityMu.Lock()

			for _, entry := range app.activity {
				if entry.expired() {
					log.Debug("Player forum activity expired", zap.Int64("steam_id", entry.person.SteamID.Int64()))

					continue
				}

				current = append(current, entry)
			}

			app.activity = current

			app.activityMu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
