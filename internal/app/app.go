// Package app is the main application and entry point. It implements the action.Executor and io.Closer interfaces.
package app

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/mxpv/patreon-go.v1"
)

var (
	// BuildVersion holds the current git revision, as of build time.
	BuildVersion = "master"

	logFileChan          chan *model.LogFilePayload
	logger               *zap.Logger
	warningChan          chan newUserWarning
	notificationChan     chan NotificationPayload
	serverStateMu        *sync.RWMutex
	serverState          state.ServerStateCollection
	bannedGroupMembers   map[steamid.GID]steamid.Collection
	bannedGroupMembersMu *sync.RWMutex
	patreonClient        *patreon.Client
	patreonMu            *sync.RWMutex
	patreonCampaigns     []patreon.Campaign
	patreonPledges       []patreon.Pledge
)

type userWarning struct {
	WarnReason    store.Reason
	Message       string
	Matched       string
	MatchedFilter *store.Filter
	CreatedOn     time.Time
}

func init() {
	logFileChan = make(chan *model.LogFilePayload, 10)
	warningChan = make(chan newUserWarning)
	notificationChan = make(chan NotificationPayload, 5)
	serverStateMu = &sync.RWMutex{}
	serverState = state.ServerStateCollection{}
	bannedGroupMembers = map[steamid.GID]steamid.Collection{}
	bannedGroupMembersMu = &sync.RWMutex{}
	patreonMu = &sync.RWMutex{}
}

func firstTimeSetup(ctx context.Context) error {
	if !config.General.Owner.Valid() {
		return errors.New("Configured owner is not a valid steam64")
	}
	localCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	var owner store.Person
	if errRootUser := store.GetPersonBySteamID(localCtx, config.General.Owner, &owner); errRootUser != nil {
		if !errors.Is(errRootUser, store.ErrNoResult) {
			return errors.Wrapf(errRootUser, "Failed first time setup")
		}
		logger.Info("Performing initial setup")
		newOwner := store.NewPerson(config.General.Owner)
		newOwner.PermissionLevel = consts.PAdmin
		if errSave := store.SavePerson(localCtx, &newOwner); errSave != nil {
			return errors.Wrap(errSave, "Failed to create admin user")
		}
		newsEntry := store.NewsEntry{
			Title:       "Welcome to gbans",
			BodyMD:      "This is an *example* **news** entry.",
			IsPublished: true,
			CreatedOn:   time.Now(),
			UpdatedOn:   time.Now(),
		}
		if errSave := store.SaveNewsArticle(localCtx, &newsEntry); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample news entry")
		}
		server := store.NewServer("server-1", "127.0.0.1", 27015)
		server.CC = "jp"
		server.RCON = "example_rcon"
		server.Latitude = 35.652832
		server.Longitude = 139.839478
		server.ServerNameLong = "Example Server"
		server.LogSecret = 12345678
		server.Region = "asia"
		if errSave := store.SaveServer(localCtx, &server); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample server entry")
		}
		var page wiki.Page
		page.BodyMD = "# Welcome to the wiki"
		page.UpdatedOn = time.Now()
		page.CreatedOn = time.Now()
		page.Revision = 1
		page.Slug = wiki.RootSlug
		if errSave := store.SaveWikiPage(localCtx, &page); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample wiki entry")
		}
	}
	return nil
}

func PatreonPledges() []patreon.Pledge {
	patreonMu.RLock()
	pledges := patreonPledges
	// users := web.app.patreonUsers
	patreonMu.RUnlock()
	return pledges
}

func PatreonCampaigns() []patreon.Campaign {
	patreonMu.RLock()
	campaigns := patreonCampaigns
	patreonMu.RUnlock()
	return campaigns
}

func Init(ctx context.Context, l *zap.Logger) error {
	logger = l.Named("gbans")
	if setupErr := firstTimeSetup(ctx); setupErr != nil {
		logger.Fatal("Failed to do first time setup", zap.Error(setupErr))
	}

	discord.SetOnConnect(func() {
		_ = SendNotification(context.TODO(), NotificationPayload{
			MinPerms: consts.PAdmin, Severity: consts.SeverityInfo, Message: "Discord connected",
		})
	})
	discord.SetOnDisconnect(func() {
		_ = SendNotification(context.TODO(), NotificationPayload{
			MinPerms: consts.PAdmin, Severity: consts.SeverityInfo, Message: "Discord disconnected",
		})
	})

	pc, errPatreon := NewPatreonClient(ctx)
	if errPatreon == nil {
		patreonClient = pc
	}

	// Load in the external network block / ip ban lists to memory if enabled
	if config.Net.Enabled {
		if errNetBans := initNetBans(ctx); errNetBans != nil {
			return errors.Wrap(errNetBans, "Failed to load net bans")
		}
	} else {
		logger.Warn("External Network ban lists not enabled")
	}

	// Start the background goroutine workers
	initWorkers(ctx)

	// Load the filtered word set into memory
	if config.Filter.Enabled {
		if errFilter := initFilters(ctx); errFilter != nil {
			return errors.Wrap(errFilter, "Failed to load filters")
		}
		logger.Info("Loaded filter list", zap.Int("count", len(wordFilters)))
	}

	return nil
}

type newUserWarning struct {
	ServerEvent model.ServerEvent
	Message     string
	userWarning
}

// warnWorker handles tracking and applying warnings based on incoming events.
func warnWorker(ctx context.Context) {
	warnings := map[steamid.SID64][]userWarning{}
	eventChan := make(chan model.ServerEvent)
	if errRegister := Consume(eventChan, []logparse.EventType{logparse.Say, logparse.SayTeam}); errRegister != nil {
		logger.Fatal("Failed to register event reader", zap.Error(errRegister))
	}
	ticker := time.NewTicker(1 * time.Second)
	warningHandler := func() {
		for {
			select {
			case now := <-ticker.C:
				for steamID := range warnings {
					for warnIdx, warning := range warnings[steamID] {
						if now.Sub(warning.CreatedOn) > config.General.WarningTimeout {
							if len(warnings[steamID]) > 1 {
								warnings[steamID] = append(warnings[steamID][:warnIdx], warnings[steamID][warnIdx+1])
							} else {
								delete(warnings, steamID)
							}
						}
					}
				}
			case newWarn := <-warningChan:
				evt, ok := newWarn.ServerEvent.Event.(logparse.SayEvt)
				if !ok {
					continue
				}
				if !evt.SID.Valid() {
					continue
				}
				newWarn.MatchedFilter.TriggerCount++
				if errSave := store.SaveFilter(ctx, newWarn.MatchedFilter); errSave != nil {
					logger.Error("Failed to update filter trigger count", zap.Error(errSave))
				}
				logger.Info("User triggered word filter",
					zap.String("matched", newWarn.Matched),
					zap.String("message", newWarn.Message),
					zap.Int64("filter_id", newWarn.MatchedFilter.FilterID))
				var person store.Person
				if personErr := PersonBySID(ctx, evt.SID, &person); personErr != nil {
					logger.Error("Failed to get person for warning", zap.Error(personErr))
					continue
				}
				if newWarn.MatchedFilter.IsEnabled {
					_, found := warnings[evt.SID]
					if !found {
						warnings[evt.SID] = []userWarning{}
					}
					warnings[evt.SID] = append(warnings[evt.SID], newWarn.userWarning)
				}

				title := fmt.Sprintf("Language Warning (#%d/%d)", len(warnings[evt.SID]), config.General.WarningLimit)
				if !newWarn.MatchedFilter.IsEnabled {
					title = "[DISABLED] Language Warning"
				}
				warnNotice := &discordgo.MessageEmbed{
					URL:   config.ExtURL("/profiles/%d", evt.SID),
					Type:  discordgo.EmbedTypeRich,
					Title: title,
					Color: int(discord.Green),
					Image: &discordgo.MessageEmbedImage{URL: person.AvatarFull},
				}
				discord.AddField(warnNotice, "Matched", newWarn.Matched)
				discord.AddField(warnNotice, "Message", newWarn.userWarning.Message)
				if newWarn.MatchedFilter.IsEnabled {
					if len(warnings[evt.SID]) > config.General.WarningLimit {
						logger.Info("Warn limit exceeded",
							zap.Int64("sid64", evt.SID.Int64()),
							zap.Int("count", len(warnings[evt.SID])))
						var errBan error
						var banSteam store.BanSteam
						if errNewBan := store.NewBanSteam(ctx, store.StringSID(config.General.Owner.String()),
							store.StringSID(evt.SID.String()),
							store.Duration(config.General.WarningExceededDurationValue),
							newWarn.WarnReason,
							"",
							"Automatic warning ban",
							store.System,
							0,
							store.NoComm,
							&banSteam); errNewBan != nil {
							logger.Error("Failed to create warning ban", zap.Error(errNewBan))
							continue
						}
						switch config.General.WarningExceededAction {
						case config.Gag:
							banSteam.BanType = store.NoComm
							errBan = BanSteam(ctx, &banSteam)
						case config.Ban:
							banSteam.BanType = store.Banned
							errBan = BanSteam(ctx, &banSteam)
						case config.Kick:
							errBan = Kick(ctx, store.System, evt.SID, config.General.Owner, newWarn.WarnReason)
						}
						if errBan != nil {
							logger.Error("Failed to apply warning action",
								zap.Error(errBan),
								zap.String("action", string(config.General.WarningExceededAction)))
						}
						discord.AddField(warnNotice, "Name", person.PersonaName)
						expIn := "Permanent"
						expAt := "Permanent"
						if banSteam.ValidUntil.Year()-config.Now().Year() < 5 {
							expIn = config.FmtDuration(banSteam.ValidUntil)
							expAt = config.FmtTimeShort(banSteam.ValidUntil)
						}
						discord.AddField(warnNotice, "Expires In", expIn)
						discord.AddField(warnNotice, "Expires At", expAt)
					} else {
						msg := fmt.Sprintf("[WARN #%d] Please refrain from using slurs/toxicity (see: rules & MOTD). "+
							"Further offenses will result in mutes/bans", len(warnings[evt.SID]))
						if errPSay := PSay(ctx, 0, evt.SID, msg); errPSay != nil {
							logger.Error("Failed to send user warning psay message", zap.Error(errPSay))
						}
					}
				}
				discord.AddField(warnNotice, "Pattern", newWarn.MatchedFilter.Pattern)
				discord.AddFieldsSteamID(warnNotice, evt.SID)
				discord.AddFieldInt64Inline(warnNotice, "Filter ID", newWarn.MatchedFilter.FilterID)
				discord.AddFieldInline(warnNotice, "Server", newWarn.ServerEvent.Server.ServerNameShort)
				discord.SendPayload(discord.Payload{
					ChannelID: config.Discord.ModLogChannelID,
					Embed:     warnNotice,
				})

			case <-ctx.Done():
				return
			}
		}
	}

	go warningHandler()

	for {
		select {
		case serverEvent := <-eventChan:
			evt, ok := serverEvent.Event.(logparse.SayEvt)
			if !ok {
				logger.Error("Got invalid type?")
				continue
			}
			if evt.Msg == "" {
				continue
			}
			matchedWord, matchedFilter := findFilteredWordMatch(evt.Msg)
			if matchedFilter != nil {
				warningChan <- newUserWarning{
					ServerEvent: serverEvent,
					userWarning: userWarning{
						WarnReason:    store.Language,
						Message:       evt.Msg,
						Matched:       matchedWord,
						MatchedFilter: matchedFilter,
						CreatedOn:     config.Now(),
					},
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func matchSummarizer(ctx context.Context) {
	eventChan := make(chan model.ServerEvent)
	if errReg := Consume(eventChan, []logparse.EventType{logparse.Any}); errReg != nil {
		logger.Error("logWriter Tried to register duplicate reader channel", zap.Error(errReg))
	}
	matches := map[int]logparse.Match{}

	var curServer store.Server
	for {
		select {
		case evt := <-eventChan:
			match, found := matches[evt.Server.ServerID]
			if !found && evt.EventType != logparse.MapLoad {
				// Wait for new map
				continue
			}
			if evt.EventType == logparse.LogStart {
				logger.Info("New match created (new game)", zap.String("server", evt.Server.ServerNameShort))
				matches[evt.Server.ServerID] = logparse.NewMatch(logger, evt.Server.ServerID, evt.Server.ServerNameLong)
			}
			// Apply the update before any secondary side effects trigger
			if errApply := match.Apply(evt.Results); errApply != nil {
				logger.Error("Error applying event",
					zap.String("server", evt.Server.ServerNameShort),
					zap.Error(errApply))
			}
			switch evt.EventType {
			case logparse.LogStop:
				fallthrough
			case logparse.WGameOver:
				go func(completeMatch logparse.Match) {
					if errSave := store.MatchSave(ctx, &completeMatch); errSave != nil {
						logger.Error("Failed to save match",
							zap.String("server", evt.Server.ServerNameShort), zap.Error(errSave))
					} else {
						sendDiscordMatchResults(curServer, completeMatch)
					}
				}(match)
				delete(matches, evt.Server.ServerID)
			}
		case <-ctx.Done():
			return
		}
	}
}

func sendDiscordMatchResults(server store.Server, match logparse.Match) {
	embed := &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeRich,
		Title:       fmt.Sprintf("Match #%d - %s - %s", match.MatchID, server.ServerNameShort, match.MapName),
		Description: "Match results",
		Color:       int(discord.Green),
		URL:         config.ExtURL("/log/%d", match.MatchID),
	}
	redScore := 0
	bluScore := 0
	for _, round := range match.Rounds {
		redScore += round.Score.Red
		bluScore += round.Score.Blu
	}

	found := 0
	for _, teamStats := range match.TeamSums {
		discord.AddFieldInline(embed, fmt.Sprintf("%s Kills", teamStats.Team.String()), fmt.Sprintf("%d", teamStats.Kills))
		discord.AddFieldInline(embed, fmt.Sprintf("%s Damage", teamStats.Team.String()), fmt.Sprintf("%d", teamStats.Damage))
		discord.AddFieldInline(embed, fmt.Sprintf("%s Ubers/Drops", teamStats.Team.String()), fmt.Sprintf("%d/%d", teamStats.Charges, teamStats.Drops))
		found++
	}
	discord.AddFieldInline(embed, "Red Score", fmt.Sprintf("%d", redScore))
	discord.AddFieldInline(embed, "Blu Score", fmt.Sprintf("%d", bluScore))
	discord.AddFieldInline(embed, "Duration", fmt.Sprintf("%.2f Minutes", time.Since(match.CreatedOn).Minutes()))
	discord.SendPayload(discord.Payload{ChannelID: config.Discord.LogChannelID, Embed: embed})
}

func playerMessageWriter(ctx context.Context) {
	serverEventChan := make(chan model.ServerEvent)
	if errRegister := Consume(serverEventChan, []logparse.EventType{
		logparse.Say,
		logparse.SayTeam,
	}); errRegister != nil {
		logger.Warn("logWriter Tried to register duplicate reader channel", zap.Error(errRegister))
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-serverEventChan:
			switch evt.EventType {
			case logparse.Say:
				fallthrough
			case logparse.SayTeam:
				e, ok := evt.Event.(logparse.SayEvt)
				if !ok {
					continue
				}
				if e.Msg == "" {
					logger.Warn("Empty person message body, skipping")
					continue
				}
				msg := store.PersonMessage{
					SteamID:     e.SID,
					PersonaName: e.Name,
					ServerName:  evt.Server.ServerNameLong,
					ServerID:    evt.Server.ServerID,
					Body:        e.Msg,
					Team:        evt.EventType == logparse.SayTeam,
					CreatedOn:   e.CreatedOn,
				}
				lCtx, cancel := context.WithTimeout(ctx, time.Second*5)
				if errChat := store.AddChatHistory(lCtx, &msg); errChat != nil {
					logger.Error("Failed to add chat history", zap.Error(errChat))
				}
				cancel()
				logger.Debug("Saved user chat message", zap.String("message", msg.Body))
			}
		}
	}
}

func playerConnectionWriter(ctx context.Context) {
	serverEventChan := make(chan model.ServerEvent)
	if errRegister := Consume(serverEventChan, []logparse.EventType{logparse.Connected}); errRegister != nil {
		logger.Warn("logWriter Tried to register duplicate reader channel", zap.Error(errRegister))
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-serverEventChan:
			e, ok := evt.Event.(logparse.ConnectedEvt)
			if !ok {
				continue
			}
			if e.Address == "" {
				logger.Warn("Empty person message body, skipping")
				continue
			}
			parsedAddr := net.ParseIP(e.Address)
			if parsedAddr == nil {
				logger.Warn("Received invalid address", zap.String("addr", e.Address))
				continue
			}
			conn := store.PersonConnection{
				IPAddr:      parsedAddr,
				SteamID:     e.SID,
				PersonaName: e.Name,
				CreatedOn:   e.CreatedOn,
			}
			lCtx, cancel := context.WithTimeout(ctx, time.Second*5)
			if errChat := store.AddConnectionHistory(lCtx, &conn); errChat != nil {
				logger.Error("Failed to add connection history", zap.Error(errChat))
			}
			cancel()
		}
	}
}

// logReader is the fan-out orchestrator for game log events
// Registering receivers can be accomplished with RegisterLogEventReader.
func logReader(ctx context.Context) {
	var file *os.File
	if config.Debug.WriteUnhandledLogEvents {
		var errCreateFile error
		file, errCreateFile = os.Create("./unhandled_messages.log")
		if errCreateFile != nil {
			logger.Fatal("Failed to open debug message log", zap.Error(errCreateFile))
		}
		defer func() {
			if errClose := file.Close(); errClose != nil {
				logger.Error("Failed to close unhandled_messages.log", zap.Error(errClose))
			}
		}()
	}
	// playerStateCache := newPlayerCache(app.logger)
	for {
		select {
		case logFile := <-logFileChan:
			emitted := 0
			failed := 0
			unknown := 0
			ignored := 0
			for _, logLine := range logFile.Lines {
				parseResult, errParse := logparse.Parse(logLine)
				if errParse != nil {
					continue
				}
				serverEvent := model.ServerEvent{
					Server:  logFile.Server,
					Results: parseResult,
				}
				if serverEvent.EventType == logparse.IgnoredMsg {
					ignored++
					continue
				} else if serverEvent.EventType == logparse.UnknownMsg {
					unknown++
					if config.Debug.WriteUnhandledLogEvents {
						if _, errWrite := file.WriteString(logLine + "\n"); errWrite != nil {
							logger.Error("Failed to write debug log", zap.Error(errWrite))
						}
					}
				}
				Emit(serverEvent)
				emitted++
			}
			logger.Debug("Completed emitting logfile events",
				zap.Int("ok", emitted), zap.Int("failed", failed),
				zap.Int("unknown", unknown), zap.Int("ignored", ignored))
		case <-ctx.Done():
			logger.Debug("logReader shutting down")
			return
		}
	}
}

func initFilters(ctx context.Context) error {
	// TODO load external lists via http
	localCtx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()
	words, errGetFilters := store.GetFilters(localCtx)
	if errGetFilters != nil {
		if errors.Is(errGetFilters, store.ErrNoResult) {
			return nil
		}
		return errGetFilters
	}
	importFilteredWords(words)
	return nil
}

func initWorkers(ctx context.Context) {
	go patreonUpdater(ctx)
	go banSweeper(ctx)
	// go profileUpdater(ctx)
	go warnWorker(ctx)
	go logReader(ctx)
	go initLogSrc(ctx)
	go logMetricsConsumer(ctx)
	go matchSummarizer(ctx)
	go playerMessageWriter(ctx)
	go playerConnectionWriter(ctx)
	go steamGroupMembershipUpdater(ctx)
	go localStatUpdater(ctx)
	go cleanupTasks(ctx)
	go showReportMeta(ctx)
	go notificationSender(ctx)
	go demoCleaner(ctx)
	go stateUpdater(ctx, time.Second*30, time.Second*180)
}

// UDP log sink.
func initLogSrc(ctx context.Context) {
	logSrc, errLogSrc := newRemoteSrcdsLogSource(logger)
	if errLogSrc != nil {
		logger.Fatal("Failed to setup udp log src", zap.Error(errLogSrc))
	}
	logSrc.start(ctx)
}

// func SendUserNotification(pl NotificationPayload) {
//	select {
//	case notificationChan <- pl:
//	default:
//		logger.Error("Failed to write user notification payload, channel full")
//	}
// }

func initNetBans(ctx context.Context) error {
	for _, banList := range config.Net.Sources {
		if _, errImport := thirdparty.Import(ctx, banList); errImport != nil {
			return errImport
		}
	}
	return nil
}

// validateLink is used in the case of discord origin actions that require mapping the
// discord member ID to a SteamID so that we can track its use and apply permissions, etc.
//
// This function will replace the discord member id value in the target field with
// the found SteamID, if any.
// func validateLink(ctx context.Context, database store.Store, sourceID action.Author, target *action.Author) error {
//	var p model.Person
//	if errGetPerson := database.GetPersonByDiscordID(ctx, string(sourceID), &p); errGetPerson != nil {
//		if errGetPerson == store.ErrNoResult {
//			return consts.ErrUnlinkedAccount
//		}
//		return consts.ErrInternal
//	}
//	*target = action.Author(p.SteamID.String())
//	return nil
// }
