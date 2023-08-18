// Package app is the main application and entry point. It implements the action.Executor and io.Closer interfaces.
package app

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// BuildVersion holds the current git revision, as of build time.
var BuildVersion = "master" //nolint:gochecknoglobals

type App struct {
	conf                 *Config
	bot                  *discord.Bot
	db                   *store.Store
	log                  *zap.Logger
	logFileChan          chan *logFilePayload
	notificationChan     chan NotificationPayload
	incomingGameChat     chan store.PersonMessage
	state                *serverStateCollector
	bannedGroupMembers   map[steamid.GID]steamid.Collection
	bannedGroupMembersMu *sync.RWMutex
	patreon              *patreonManager
	eb                   *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]
	wordFilters          *wordFilters
	mc                   *metricCollector
	logListener          *logparse.UDPLogListener
}

func New(conf *Config, database *store.Store, bot *discord.Bot, logger *zap.Logger) App {
	application := App{
		bot:                  bot,
		eb:                   fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent](),
		db:                   database,
		conf:                 conf,
		log:                  logger,
		logFileChan:          make(chan *logFilePayload, 10),
		notificationChan:     make(chan NotificationPayload, 5),
		incomingGameChat:     make(chan store.PersonMessage, 5),
		bannedGroupMembers:   map[steamid.GID]steamid.Collection{},
		bannedGroupMembersMu: &sync.RWMutex{},
		patreon:              newPatreonManager(logger, conf, database),
		wordFilters:          newWordFilters(),
		mc:                   newMetricCollector(),
		state:                newServerStateCollector(logger),
	}

	if conf.Discord.Enabled {
		if errReg := application.registerDiscordHandlers(); errReg != nil {
			panic(errReg)
		}
	}

	return application
}

type userWarning struct {
	WarnReason    store.Reason
	Message       string
	Matched       string
	MatchedFilter *store.Filter
	CreatedOn     time.Time
}

func firstTimeSetup(ctx context.Context, conf *Config, database *store.Store) error {
	if !conf.General.Owner.Valid() {
		return errors.New("Configured owner is not a valid steam64")
	}

	localCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var owner store.Person

	if errRootUser := database.GetPersonBySteamID(localCtx, conf.General.Owner, &owner); errRootUser != nil {
		if !errors.Is(errRootUser, store.ErrNoResult) {
			return errors.Wrapf(errRootUser, "Failed first time setup")
		}

		newOwner := store.NewPerson(conf.General.Owner)
		newOwner.PermissionLevel = consts.PAdmin

		if errSave := database.SavePerson(localCtx, &newOwner); errSave != nil {
			return errors.Wrap(errSave, "Failed to create admin user")
		}

		newsEntry := store.NewsEntry{
			Title:       "Welcome to gbans",
			BodyMD:      "This is an *example* **news** entry.",
			IsPublished: true,
			CreatedOn:   time.Now(),
			UpdatedOn:   time.Now(),
		}

		if errSave := database.SaveNewsArticle(localCtx, &newsEntry); errSave != nil {
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

		if errSave := database.SaveServer(localCtx, &server); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample server entry")
		}

		page := wiki.Page{
			Slug:      wiki.RootSlug,
			BodyMD:    "# Welcome to the wiki",
			Revision:  1,
			CreatedOn: time.Now(),
			UpdatedOn: time.Now(),
		}

		if errSave := database.SaveWikiPage(localCtx, &page); errSave != nil {
			return errors.Wrap(errSave, "Failed to create sample wiki entry")
		}
	}

	return nil
}

func (app *App) Init(ctx context.Context) error {
	if setupErr := firstTimeSetup(ctx, app.conf, app.db); setupErr != nil {
		app.log.Fatal("Failed to do first time setup", zap.Error(setupErr))
	}

	if errWeapons := app.db.LoadWeapons(ctx); errWeapons != nil {
		app.log.Fatal("Failed to load weapons", zap.Error(errWeapons))
	}

	// Load in the external network block / ip ban lists to memory if enabled
	if app.conf.NetBans.Enabled {
		if errNetBans := initNetBans(ctx, app.conf); errNetBans != nil {
			return errors.Wrap(errNetBans, "Failed to load net bans")
		}
	} else {
		app.log.Warn("External Network ban lists not enabled")
	}

	// start the background goroutine workers
	app.startWorkers(ctx)

	// Load the filtered word set into memory
	if app.conf.Filter.Enabled {
		if errFilter := app.LoadFilters(ctx); errFilter != nil {
			return errors.Wrap(errFilter, "Failed to load filters")
		}

		app.log.Info("Loaded filter list", zap.Int("count", len(app.wordFilters.wordFilters)))
	}

	return nil
}

func (app *App) StartHTTP(ctx context.Context) error {
	app.log.Info("Service status changed", zap.String("state", "ready"))
	defer app.log.Info("Service status changed", zap.String("state", "stopped"))

	if app.conf.General.Mode == ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	httpServer := newHTTPServer(ctx, app)

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)

		defer cancel()

		if errShutdown := httpServer.Shutdown(shutdownCtx); errShutdown != nil { //nolint:contextcheck
			app.log.Error("Error shutting down http service", zap.Error(errShutdown))
		}
	}()

	errServe := httpServer.ListenAndServe()
	if errServe != nil && !errors.Is(errServe, http.ErrServerClosed) {
		return errors.Wrap(errServe, "HTTP listener returned error")
	}

	return nil
}

type LinkablePath interface {
	Path() string
}

func (app *App) ExtURL(obj LinkablePath) string {
	return app.ExtURLRaw(obj.Path())
}

func (app *App) ExtURLRaw(path string, args ...any) string {
	return strings.TrimRight(app.conf.General.ExternalURL, "/") + fmt.Sprintf(strings.TrimLeft(path, "."), args...)
}

type newUserWarning struct {
	userMessage store.PersonMessage
	userWarning
}

// warnWorker handles tracking and applying warnings based on incoming events.
func (app *App) warnWorker(ctx context.Context) { //nolint:maintidx
	var (
		log         = app.log.Named("warnWorker")
		warnings    = map[steamid.SID64][]userWarning{}
		ticker      = time.NewTicker(1 * time.Second)
		warningChan = make(chan newUserWarning)
	)

	warningHandler := func() {
		for {
			warnDur := app.conf.General.WarningTimeout.Duration()
			select {
			case now := <-ticker.C:
				for steamID := range warnings {
					for warnIdx, warning := range warnings[steamID] {
						if now.Sub(warning.CreatedOn) > warnDur {
							if len(warnings[steamID]) > 1 {
								warnings[steamID] = append(warnings[steamID][:warnIdx], warnings[steamID][warnIdx+1])
							} else {
								delete(warnings, steamID)
							}
						}
					}
				}
			case newWarn := <-warningChan:
				if !newWarn.userMessage.SteamID.Valid() {
					continue
				}

				newWarn.MatchedFilter.TriggerCount++
				if errSave := app.db.SaveFilter(ctx, newWarn.MatchedFilter); errSave != nil {
					log.Error("Failed to update filter trigger count", zap.Error(errSave))
				}

				var person store.Person
				if personErr := app.PersonBySID(ctx, newWarn.userMessage.SteamID, &person); personErr != nil {
					log.Error("Failed to get person for warning", zap.Error(personErr))

					continue
				}

				if !newWarn.MatchedFilter.IsEnabled {
					continue
				}

				title := fmt.Sprintf("Language Warning (#%d/%d)", len(warnings[newWarn.userMessage.SteamID])+1, app.conf.General.WarningLimit)
				if app.conf.Filter.Dry {
					title = "[DRYRUN] " + title
				}

				msgEmbed := discord.
					NewEmbed(title).
					SetDescription(newWarn.userWarning.Message).
					SetColor(app.bot.Colour.Warn).
					AddField("Filter ID", fmt.Sprintf("%d", newWarn.MatchedFilter.FilterID)).
					AddField("Matched", newWarn.Matched).
					AddField("Server", newWarn.userMessage.ServerName).InlineAllFields().
					AddField("Pattern", newWarn.MatchedFilter.Pattern)

				app.addAuthor(ctx, msgEmbed, newWarn.userMessage.SteamID)

				discord.AddFieldsSteamID(msgEmbed, newWarn.userMessage.SteamID)

				if !newWarn.MatchedFilter.IsEnabled {
					continue
				}

				if !app.conf.Filter.Dry {
					_, found := warnings[newWarn.userMessage.SteamID]
					if !found {
						warnings[newWarn.userMessage.SteamID] = []userWarning{}
					}

					warnings[newWarn.userMessage.SteamID] = append(warnings[newWarn.userMessage.SteamID], newWarn.userWarning)

					if len(warnings[newWarn.userMessage.SteamID]) > app.conf.General.WarningLimit {
						log.Info("Warn limit exceeded",
							zap.Int64("sid64", newWarn.userMessage.SteamID.Int64()),
							zap.Int("count", len(warnings[newWarn.userMessage.SteamID])))

						var (
							errBan   error
							banSteam store.BanSteam
							expIn    = "Permanent"
							expAt    = expIn
						)

						if errNewBan := store.NewBanSteam(ctx, store.StringSID(app.conf.General.Owner.String()),
							store.StringSID(newWarn.userMessage.SteamID.String()),
							store.Duration(app.conf.General.WarningExceededDuration),
							newWarn.WarnReason,
							"",
							"Automatic warning ban",
							store.System,
							0,
							store.NoComm,
							&banSteam); errNewBan != nil {
							log.Error("Failed to create warning ban", zap.Error(errNewBan))

							continue
						}

						switch app.conf.General.WarningExceededAction {
						case Gag:
							banSteam.BanType = store.NoComm
							errBan = app.BanSteam(ctx, &banSteam)
						case Ban:
							banSteam.BanType = store.Banned
							errBan = app.BanSteam(ctx, &banSteam)
						case Kick:
							errBan = app.Kick(ctx, store.System, newWarn.userMessage.SteamID, app.conf.General.Owner, newWarn.WarnReason)
						}

						if errBan != nil {
							log.Error("Failed to apply warning action",
								zap.Error(errBan),
								zap.String("action", string(app.conf.General.WarningExceededAction)))
						}

						msgEmbed.AddField("Name", person.PersonaName)

						if banSteam.ValidUntil.Year()-time.Now().Year() < 5 {
							expIn = FmtDuration(banSteam.ValidUntil)
							expAt = FmtTimeShort(banSteam.ValidUntil)
						}

						msgEmbed.AddField("Expires In", expIn)
						msgEmbed.AddField("Expires At", expAt)
					} else {
						msg := fmt.Sprintf("[WARN #%d] Please refrain from using slurs/toxicity (see: rules & MOTD). "+
							"Further offenses will result in mutes/bans", len(warnings[newWarn.userMessage.SteamID]))

						if errPSay := app.PSay(ctx, newWarn.userMessage.SteamID, msg); errPSay != nil {
							log.Error("Failed to send user warning psay message", zap.Error(errPSay))
						}
					}
				}

				if app.conf.Filter.PingDiscord {
					app.bot.SendPayload(discord.Payload{
						ChannelID: app.conf.Discord.LogChannelID,
						Embed:     msgEmbed.MessageEmbed,
					})
				}

			case <-ctx.Done():
				return
			}
		}
	}

	go warningHandler()

	for {
		select {
		case userMessage := <-app.incomingGameChat:
			matchedWord, matchedFilter := app.wordFilters.findFilteredWordMatch(userMessage.Body)
			if matchedFilter != nil {
				if errSaveMatch := app.db.AddMessageFilterMatch(ctx, userMessage.PersonMessageID, matchedFilter.FilterID); errSaveMatch != nil {
					log.Error("Failed to save message match status", zap.Error(errSaveMatch))
				}
				warningChan <- newUserWarning{
					userMessage: userMessage,
					userWarning: userWarning{
						WarnReason:    store.Language,
						Message:       userMessage.Body,
						Matched:       matchedWord,
						MatchedFilter: matchedFilter,
						CreatedOn:     time.Now(),
					},
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func defaultTable(writer io.Writer) *tablewriter.Table {
	tbl := tablewriter.NewWriter(writer)
	tbl.SetAutoFormatHeaders(true)
	tbl.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	tbl.SetCenterSeparator("")
	tbl.SetColumnSeparator("")
	tbl.SetRowSeparator("")
	tbl.SetHeaderLine(false)
	tbl.SetTablePadding(" ")
	tbl.SetAlignment(tablewriter.ALIGN_LEFT)

	return tbl
}

func infString(f float64) string {
	if f == -1 {
		return "∞"
	}

	return fmt.Sprintf("%.1f", f)
}

func matchASCIITable(match store.MatchResult) string {
	writerPlayers := &strings.Builder{}
	tablePlayers := defaultTable(writerPlayers)
	tablePlayers.SetHeader([]string{"Name", "K", "A", "D", "K:D", "KA:D", "DA", "DAm", "B H A C"})

	players := match.TopPlayers()

	for i, player := range players {
		if i == 30 {
			break
		}

		name := player.SteamID.String()
		if player.Name != "" {
			name = player.Name
		}

		if len(name) > 17 {
			name = name[0:17]
		}

		tablePlayers.Append([]string{
			name,
			fmt.Sprintf("%d", player.Kills),
			fmt.Sprintf("%d", player.Assists),
			fmt.Sprintf("%d", player.Deaths),
			infString(player.KDRatio()),
			infString(player.KDARatio()),
			fmt.Sprintf("%d", player.Damage),
			fmt.Sprintf("%d", player.DamagePerMin()),
			fmt.Sprintf("%d %d %d %d",
				player.Backstabs, player.Headshots, player.Airshots, player.Captures),
		})
	}

	tablePlayers.Render()

	writerHealers := &strings.Builder{}
	tableHealers := defaultTable(writerPlayers)
	tableHealers.SetHeader([]string{"Name", "A", "D", "Healing", "H/M", "U/K/Q/V", "Dr"})

	for _, player := range match.PlayerStats {
		if player.MedicStats == nil {
			continue
		}

		name := player.SteamID.String()
		if player.Name != "" {
			name = player.Name
		}

		if len(name) > 17 {
			name = name[0:17]
		}

		tableHealers.Append([]string{
			name,
			fmt.Sprintf("%d", player.Assists),
			fmt.Sprintf("%d", player.Deaths),
			fmt.Sprintf("%d", player.MedicStats.Healing),
			fmt.Sprintf("%d", player.MedicStats.HealingPerMin(player.TimeEnd.Sub(player.TimeStart))),
			fmt.Sprintf("%d/%d/%d/%d",
				player.MedicStats.ChargesUber, player.MedicStats.ChargesKritz,
				player.MedicStats.ChargesQuickfix, player.MedicStats.ChargesVacc),
			fmt.Sprintf("%d", player.MedicStats.Drops),
		})
	}

	resp := fmt.Sprintf("`%s\n%s`",
		strings.Trim(writerPlayers.String(), "\n"),
		strings.Trim(writerHealers.String(), "\n"))

	return resp
}

func (app *App) chatRecorder(ctx context.Context) {
	var (
		log             = app.log.Named("chatRecorder")
		serverEventChan = make(chan logparse.ServerEvent)
	)

	if errRegister := app.eb.Consume(serverEventChan, logparse.Say, logparse.SayTeam); errRegister != nil {
		log.Warn("logWriter Tried to register duplicate reader channel", zap.Error(errRegister))

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
				newServerEvent, ok := evt.Event.(logparse.SayEvt)
				if !ok {
					continue
				}

				if newServerEvent.Msg == "" {
					log.Warn("Empty person message body, skipping")

					continue
				}

				var author store.Person
				if errPerson := app.PersonBySID(ctx, newServerEvent.SID, &author); errPerson != nil {
					log.Error("Failed to add chat history, could not get author", zap.Error(errPerson))

					continue
				}

				msg := store.PersonMessage{
					SteamID:     newServerEvent.SID,
					PersonaName: strings.ToValidUTF8(newServerEvent.Name, "_"),
					ServerName:  evt.ServerName,
					ServerID:    evt.ServerID,
					Body:        strings.ToValidUTF8(newServerEvent.Msg, "_"),
					Team:        newServerEvent.Team,
					CreatedOn:   newServerEvent.CreatedOn,
				}

				if errChat := app.db.AddChatHistory(ctx, &msg); errChat != nil {
					log.Error("Failed to add chat history", zap.Error(errChat))

					continue
				}

				// log.Debug("Chat message",
				//	zap.Int64("id", msg.PersonMessageID),
				//	zap.String("server", evt.ServerName),
				//	zap.String("name", newServerEvent.Name),
				//	zap.String("steam_id", newServerEvent.SID.String()),
				//	zap.Bool("team", msg.Team),
				//	zap.String("message", msg.Body))

				app.incomingGameChat <- msg
			}
		}
	}
}

func (app *App) playerConnectionWriter(ctx context.Context) {
	log := app.log.Named("playerConnectionWriter")

	serverEventChan := make(chan logparse.ServerEvent)
	if errRegister := app.eb.Consume(serverEventChan, logparse.Connected); errRegister != nil {
		log.Warn("logWriter Tried to register duplicate reader channel", zap.Error(errRegister))

		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-serverEventChan:
			newServerEvent, ok := evt.Event.(logparse.ConnectedEvt)
			if !ok {
				continue
			}

			if newServerEvent.Address == "" {
				log.Warn("Empty person message body, skipping")

				continue
			}

			parsedAddr := net.ParseIP(newServerEvent.Address)
			if parsedAddr == nil {
				log.Warn("Received invalid address", zap.String("addr", newServerEvent.Address))

				continue
			}

			// Maybe ignore these and wait for connect call to create?
			var person store.Person
			if errPerson := app.PersonBySID(ctx, newServerEvent.SID, &person); errPerson != nil {
				log.Error("Failed to load person", zap.Error(errPerson))

				continue
			}

			conn := store.PersonConnection{
				IPAddr:      parsedAddr,
				SteamID:     newServerEvent.SID,
				PersonaName: strings.ToValidUTF8(newServerEvent.Name, "_"),
				CreatedOn:   newServerEvent.CreatedOn,
			}

			lCtx, cancel := context.WithTimeout(ctx, time.Second*5)
			if errChat := app.db.AddConnectionHistory(lCtx, &conn); errChat != nil {
				log.Error("Failed to add connection history", zap.Error(errChat))
			}

			cancel()
		}
	}
}

type logFilePayload struct {
	ServerID   int
	ServerName string
	Lines      []string
	Map        string
}

// logReader is the fan-out orchestrator for game log events
// Registering receivers can be accomplished with RegisterLogEventReader.
func (app *App) logReader(ctx context.Context, writeUnhandled bool) {
	var (
		log  = app.log.Named("logReader")
		file *os.File
	)

	if writeUnhandled {
		var errCreateFile error
		file, errCreateFile = os.Create("./unhandled_messages.log")

		if errCreateFile != nil {
			log.Fatal("Failed to open debug message log", zap.Error(errCreateFile))
		}

		defer func() {
			if errClose := file.Close(); errClose != nil {
				log.Error("Failed to close unhandled_messages.log", zap.Error(errClose))
			}
		}()
	}

	parser := logparse.NewLogParser()

	// playerStateCache := newPlayerCache(app.logger)
	for {
		select {
		case logFile := <-app.logFileChan:
			emitted := 0
			failed := 0
			unknown := 0
			ignored := 0

			for _, logLine := range logFile.Lines {
				parseResult, errParse := parser.Parse(logLine)
				if errParse != nil {
					continue
				}

				newServerEvent := logparse.ServerEvent{
					ServerName: logFile.ServerName,
					ServerID:   logFile.ServerID,
					Results:    parseResult,
				}

				if newServerEvent.EventType == logparse.IgnoredMsg {
					ignored++

					continue
				} else if newServerEvent.EventType == logparse.UnknownMsg {
					unknown++
					if writeUnhandled {
						if _, errWrite := file.WriteString(logLine + "\n"); errWrite != nil {
							log.Error("Failed to write debug log", zap.Error(errWrite))
						}
					}
				}

				app.eb.Emit(newServerEvent.EventType, newServerEvent)
				emitted++
			}

			log.Debug("Completed emitting logfile events",
				zap.Int("ok", emitted), zap.Int("failed", failed),
				zap.Int("unknown", unknown), zap.Int("ignored", ignored))
		case <-ctx.Done():
			log.Debug("logReader shutting down")

			return
		}
	}
}

func (app *App) LoadFilters(ctx context.Context) error {
	// TODO load external lists via http
	localCtx, cancel := context.WithTimeout(ctx, time.Second*15)
	defer cancel()

	words, errGetFilters := app.db.GetFilters(localCtx)
	if errGetFilters != nil {
		if errors.Is(errGetFilters, store.ErrNoResult) {
			return nil
		}

		return errors.Wrap(errGetFilters, "Failed to fetch filters")
	}

	app.wordFilters.importFilteredWords(words)

	app.log.Debug("Loaded word filters", zap.Int("count", len(words)))

	return nil
}

func (app *App) startWorkers(ctx context.Context) {
	go app.patreon.updater(ctx)
	go app.banSweeper(ctx)
	go app.profileUpdater(ctx)
	go app.warnWorker(ctx)
	go app.logReader(ctx, app.conf.Debug.WriteUnhandledLogEvents)
	go app.initLogSrc(ctx)
	go logMetricsConsumer(ctx, app.mc, app.eb, app.log)
	go app.matchSummarizer(ctx)
	go app.chatRecorder(ctx)
	go app.playerConnectionWriter(ctx)
	go app.steamGroupMembershipUpdater(ctx)
	go cleanupTasks(ctx, app.db, app.log)
	go app.showReportMeta(ctx)
	go app.notificationSender(ctx)
	go demoCleaner(ctx, app.db, app.log)
	go app.stateUpdater(ctx)
}

// UDP log sink.
func (app *App) initLogSrc(ctx context.Context) {
	logSrc, errLogSrc := logparse.NewUDPLogListener(app.log, app.conf.Log.SrcdsLogAddr, func(eventType logparse.EventType, event logparse.ServerEvent) {
		app.eb.Emit(event.EventType, event)
	})

	if errLogSrc != nil {
		app.log.Fatal("Failed to setup udp log src", zap.Error(errLogSrc))
	}

	app.logListener = logSrc

	// TODO run on server config changes
	go app.updateSrcdsLogSecrets(ctx)

	app.logListener.Start(ctx)
}

func (app *App) updateSrcdsLogSecrets(ctx context.Context) {
	newSecrets := map[int]logparse.ServerIDMap{}
	serversCtx, cancelServers := context.WithTimeout(ctx, time.Second*5)

	defer cancelServers()

	servers, errServers := app.db.GetServers(serversCtx, true)
	if errServers != nil {
		app.log.Error("Failed to update srcds log secrets", zap.Error(errServers))

		return
	}

	for _, server := range servers {
		newSecrets[server.LogSecret] = logparse.ServerIDMap{
			ServerID:   server.ServerID,
			ServerName: server.ServerName,
		}
	}

	app.logListener.SetSecrets(newSecrets)
}

// PersonBySID fetches the person from the database, updating the PlayerSummary if it out of date.
func (app *App) PersonBySID(ctx context.Context, sid steamid.SID64, person *store.Person) error {
	if errGetPerson := app.db.GetOrCreatePersonBySteamID(ctx, sid, person); errGetPerson != nil {
		return errors.Wrapf(errGetPerson, "Failed to get person instance: %s", sid)
	}

	if person.IsNew || time.Since(person.UpdatedOnSteam) > time.Hour*24 {
		summaries, errSummaries := steamweb.PlayerSummaries(ctx, steamid.Collection{sid})
		if errSummaries != nil {
			return errors.Wrapf(errSummaries, "Failed to get Player summary: %v", errSummaries)
		}

		if len(summaries) > 0 {
			s := summaries[0]
			person.PlayerSummary = &s
		} else {
			app.log.Warn("Failed to update profile summary", zap.Error(errSummaries), zap.Int64("sid", sid.Int64()))
			// return errors.Errorf("Failed to fetch Player summary for %d", sid)
		}

		vac, errBans := thirdparty.FetchPlayerBans(ctx, steamid.Collection{sid})
		if errBans != nil || len(vac) != 1 {
			// return errors.Wrapf(errBans, "Failed to get Player ban state: %v", errBans)
			app.log.Warn("Failed to update ban status", zap.Error(errBans), zap.Int64("sid", sid.Int64()))
		} else {
			person.CommunityBanned = vac[0].CommunityBanned
			person.VACBans = vac[0].NumberOfVACBans
			person.GameBans = vac[0].NumberOfGameBans
			person.EconomyBan = steamweb.EconBanNone
			person.CommunityBanned = vac[0].CommunityBanned
			person.DaysSinceLastBan = vac[0].DaysSinceLastBan
		}

		person.UpdatedOnSteam = time.Now()
	}

	person.SteamID = sid
	if errSavePerson := app.db.SavePerson(ctx, person); errSavePerson != nil {
		return errors.Wrapf(errSavePerson, "Failed to save person")
	}

	return nil
}

// resolveSID is just a simple helper for calling steamid.ResolveSID64 with a timeout.
func resolveSID(ctx context.Context, sidStr string) (steamid.SID64, error) {
	localCtx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	sid64, errString := steamid.StringToSID64(sidStr)
	if errString == nil && sid64.Valid() {
		return sid64, nil
	}

	sid, errResolve := steamid.ResolveSID64(localCtx, sidStr)
	if errResolve != nil {
		return "", errors.Wrap(errResolve, "Failed to resolve vanity")
	}

	return sid, nil
}

func initNetBans(ctx context.Context, conf *Config) error {
	for _, banList := range conf.NetBans.Sources {
		if _, errImport := thirdparty.Import(ctx, banList, conf.NetBans.CachePath, conf.NetBans.MaxAge); errImport != nil {
			return errors.Wrap(errImport, "Failed to import net bans")
		}
	}

	return nil
}

type NotificationHandler struct{}

type NotificationPayload struct {
	MinPerms consts.Privilege
	Sids     steamid.Collection
	Severity consts.NotificationSeverity
	Message  string
	Link     string
}

func (app *App) SendNotification(ctx context.Context, notification NotificationPayload) error {
	// Collect all required ids
	if notification.MinPerms >= consts.PUser {
		sids, errIds := app.db.GetSteamIdsAbove(ctx, notification.MinPerms)
		if errIds != nil {
			return errors.Wrap(errIds, "Failed to fetch steamids for notification")
		}

		notification.Sids = append(notification.Sids, sids...)
	}

	uniqueIds := fp.Uniq(notification.Sids)

	people, errPeople := app.db.GetPeopleBySteamID(ctx, uniqueIds)
	if errPeople != nil && !errors.Is(errPeople, store.ErrNoResult) {
		return errors.Wrap(errPeople, "Failed to fetch people for notification")
	}

	var discordIds []string

	for _, p := range people {
		if p.DiscordID != "" {
			discordIds = append(discordIds, p.DiscordID)
		}
	}

	go func(ids []string, payload NotificationPayload) {
		for _, discordID := range ids {
			msgEmbed := discord.NewEmbed("Notification", payload.Message)
			if payload.Link != "" {
				msgEmbed.SetURL(payload.Link)
			}

			app.bot.SendPayload(discord.Payload{ChannelID: discordID, Embed: msgEmbed.Truncate().MessageEmbed})
		}
	}(discordIds, notification)

	// Broadcast to
	for _, sid := range uniqueIds {
		// Todo, prep stmt at least.
		if errSend := app.db.SendNotification(ctx, sid, notification.Severity,
			notification.Message, notification.Link); errSend != nil {
			app.log.Error("Failed to send notification", zap.Error(errSend))

			break
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
