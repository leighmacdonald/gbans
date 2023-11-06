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

// IsGroupBanned checks membership in the currently known banned group members or friends of banned users
// marked with `include_friends`.
func (app *App) IsGroupBanned(steamID steamid.SID64) (int64, bool) {
	app.bannedGroupMembersMu.RLock()
	defer app.bannedGroupMembersMu.RUnlock()

	for parentID, groupMembers := range app.bannedGroupMembers {
		for _, member := range groupMembers {
			if steamID == member {
				return parentID, true
			}
		}
	}

	return 0, false
}

// activeMatchContext represents the current match on any given server instance.
type activeMatchContext struct {
	match          logparse.Match
	cancel         context.CancelFunc
	incomingEvents chan logparse.ServerEvent
	log            *zap.Logger
	finalScores    int
}

func (am *activeMatchContext) start(ctx context.Context) {
	am.log.Info("Match started", zap.String("match_id", am.match.MatchID.String()))

	for {
		select {
		case evt := <-am.incomingEvents:
			if errApply := am.match.Apply(evt.Results); errApply != nil && !errors.Is(errApply, logparse.ErrIgnored) {
				am.log.Error("Error applying event",
					zap.String("server", evt.ServerName),
					zap.Error(errApply))
			}
		case <-ctx.Done():
			am.log.Info("Match Closed", zap.String("match_id", am.match.MatchID.String()))

			return
		}
	}
}

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

	app.bot.SendPayload(discord.Payload{
		ChannelID: app.conf.Discord.PublicMatchLogChannelID,
		Embed:     app.genDiscordMatchEmbed(result).MessageEmbed,
	})
}

func (app *App) updateSteamBanMembers(ctx context.Context) (map[int64]steamid.Collection, error) {
	newMap := map[int64]steamid.Collection{}

	localCtx, cancel := context.WithTimeout(ctx, time.Second*120)
	defer cancel()

	opts := store.BansQueryFilter{
		QueryFilter:        store.QueryFilter{Deleted: false},
		IncludeFriendsOnly: true,
	}

	steamBans, errSteam := app.db.GetBansSteam(ctx, opts)
	if errSteam != nil {
		if errors.Is(errSteam, store.ErrNoResult) {
			return newMap, nil
		}

		return nil, errors.Wrap(errSteam, "Failed to fetch bans with friends included")
	}

	for _, steamBan := range steamBans {
		friends, errFriends := steamweb.GetFriendList(localCtx, steamBan.Ban.TargetID)
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

		memberList := store.NewMembersList(steamBan.Person.SteamID.Int64(), sids)
		if errQuery := app.db.GetMembersList(ctx, steamBan.Person.SteamID.Int64(), &memberList); errQuery != nil {
			if !errors.Is(errQuery, store.ErrNoResult) {
				return nil, errors.Wrap(errQuery, "Failed to fetch members list")
			}
		}

		if errSave := app.db.SaveMembersList(ctx, &memberList); errSave != nil {
			return nil, errors.Wrap(errSave, "Failed to save banned steam friend member list")
		}

		newMap[steamBan.Person.SteamID.Int64()] = memberList.Members
	}

	return newMap, nil
}

func (app *App) updateGroupBanMembers(ctx context.Context) (map[int64]steamid.Collection, error) {
	newMap := map[int64]steamid.Collection{}

	localCtx, cancel := context.WithTimeout(ctx, time.Second*120)
	defer cancel()

	groups, errGroups := app.db.GetBanGroups(ctx)
	if errGroups != nil {
		if errors.Is(errGroups, store.ErrNoResult) {
			return newMap, nil
		}

		return nil, errors.Wrap(errGroups, "Failed to fetch banned groups")
	}

	for _, group := range groups {
		members, errMembers := steamweb.GetGroupMembers(localCtx, group.GroupID)
		if errMembers != nil {
			return nil, errors.Wrapf(errMembers, "Failed to fetch group members")
		}

		if len(members) == 0 {
			continue
		}

		memberList := store.NewMembersList(group.GroupID.Int64(), members)
		if errQuery := app.db.GetMembersList(ctx, group.GroupID.Int64(), &memberList); errQuery != nil {
			if !errors.Is(errQuery, store.ErrNoResult) {
				return nil, errors.Wrap(errQuery, "Failed to fetch members list")
			}
		}

		if errSave := app.db.SaveMembersList(ctx, &memberList); errSave != nil {
			return nil, errors.Wrap(errSave, "Failed to save banned groups member list")
		}

		newMap[group.GroupID.Int64()] = members
	}

	return newMap, nil
}

func (app *App) steamGroupMembershipUpdater(ctx context.Context) {
	log := app.log.Named("steamGroupMembership")
	ticker := time.NewTicker(time.Minute * 60)
	updateChan := make(chan any)

	go func() {
		updateChan <- true
	}()

	for {
		select {
		case <-ticker.C:
			updateChan <- true
		case <-updateChan:
			newMap := map[int64]steamid.Collection{}
			total := 0

			groupEntries, errGroupEntries := app.updateGroupBanMembers(ctx)
			if errGroupEntries == nil {
				for k, v := range groupEntries {
					newMap[k] = v
				}
			}

			friendEntries, errFriendEntries := app.updateSteamBanMembers(ctx)
			if errFriendEntries == nil {
				for k, v := range friendEntries {
					newMap[k] = v
				}
			}

			app.bannedGroupMembersMu.Lock()
			app.bannedGroupMembers = newMap
			app.bannedGroupMembersMu.Unlock()

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
			reports, _, errReports := app.db.GetReports(ctx, store.ReportQueryFilter{
				AuthorQueryFilter: store.AuthorQueryFilter{
					QueryFilter: store.QueryFilter{
						Limit: 0,
					},
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
		newBanStates, errBans := thirdparty.FetchPlayerBans(cancelCtx, steamIDs)
		if errBans != nil || len(newBanStates) != 1 {
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

	return nil
}

// profileUpdater takes care of periodically querying the steam api for updates player summaries.
// The 100 oldest profiles are updated on each execution.
func (app *App) profileUpdater(ctx context.Context) {
	var (
		log    = app.log.Named("profileUpdate")
		run    = make(chan any)
		ticker = time.NewTicker(time.Second * 60)
	)

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
			servers, errServers := app.db.GetServers(ctx, false)
			if errServers != nil && !errors.Is(errServers, store.ErrNoResult) {
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
