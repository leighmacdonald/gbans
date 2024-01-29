package app

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/metrics"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"go.uber.org/zap"
)

var (
	BuildVersion = "master" //nolint:gochecknoglobals
	BuildCommit  = ""       //nolint:gochecknoglobals
	BuildDate    = ""       //nolint:gochecknoglobals
)

func Version() domain.BuildInfo {
	return domain.BuildInfo{
		BuildVersion: BuildVersion,
		Commit:       BuildCommit,
		Date:         BuildDate,
	}
}

func (app *App) startWorkers(ctx context.Context) {
	go app.patreon.Start(ctx)
	go app.banSweeper(ctx)
	go app.profileUpdater(ctx)
	go app.warningTracker.Start(ctx)
	go app.logReader(ctx, app.Config().Debug.WriteUnhandledLogEvents)
	go app.initLogSrc(ctx)
	go metrics.logMetricsConsumer(ctx, app.mc, app.eb, app.log)
	go app.matchSummarizer.Start(ctx)
	go app.chatLogger.start(ctx)
	go app.playerConnectionWriter(ctx)
	go app.steamGroups.Start(ctx)
	go cleanupTasks(ctx, app.db, app.log)
	go app.showReportMeta(ctx)
	go app.notificationSender(ctx)
	go app.demoCleaner(ctx)
	go app.state.Start(ctx, func() config.Config {
		return app.Config()
	}, func() state.ServerStore {
		return app.Store()
	})
	go app.activityTracker.Start(ctx)
	go app.steamFriends.Start(ctx)
}

func FirstTimeSetup(ctx context.Context, conf domain.Config, pu domain.PersonUsecase,
	nu domain.NewsUsecase, sv domain.ServersUsecase, wu domain.WikiUsecase, mu domain.MatchUsecase,
	weaponMap fp.MutexMap[logparse.Weapon, int],
) error {
	if !conf.General.Owner.Valid() {
		return domain.ErrOwnerInvalid
	}

	localCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var owner domain.Person

	if errRootUser := pu.GetPersonBySteamID(localCtx, conf.General.Owner, &owner); errRootUser != nil {
		if !errors.Is(errRootUser, domain.ErrNoResult) {
			return errors.Join(errRootUser, domain.ErrCreateAdmin)
		}

		newOwner := domain.NewPerson(conf.General.Owner)
		newOwner.PermissionLevel = domain.PAdmin

		if errSave := pu.SavePerson(localCtx, &newOwner); errSave != nil {
			return errors.Join(errSave, domain.ErrSetupAdmin)
		}

		newsEntry := domain.NewsEntry{
			Title:       "Welcome to gbans",
			BodyMD:      "This is an *example* **news** entry.",
			IsPublished: true,
			CreatedOn:   time.Now(),
			UpdatedOn:   time.Now(),
		}

		if errSave := nu.SaveNewsArticle(localCtx, &newsEntry); errSave != nil {
			return errors.Join(errSave, domain.ErrSetupNews)
		}

		server := domain.NewServer("server-1", "127.0.0.1", 27015)
		server.CC = "jp"
		server.RCON = "example_rcon"
		server.Latitude = 35.652832
		server.Longitude = 139.839478
		server.Name = "Example ServerStore"
		server.LogSecret = 12345678
		server.Region = "asia"

		if errSave := sv.SaveServer(localCtx, &server); errSave != nil {
			return errors.Join(errSave, domain.ErrSetupServer)
		}

		page := domain.Page{
			Slug:      domain.RootSlug,
			BodyMD:    "# Welcome to the wiki",
			Revision:  1,
			CreatedOn: time.Now(),
			UpdatedOn: time.Now(),
		}

		if errSave := wu.SaveWikiPage(localCtx, &page); errSave != nil {
			return errors.Join(errSave, domain.ErrSetupWiki)
		}
	}

	if errWeapons := mu.LoadWeapons(ctx, weaponMap); errWeapons != nil {
		return errors.Join(errWeapons, domain.ErrSetupWeapons)
	}

	return nil
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
				log.Warn("Empty Person message body, skipping")

				continue
			}

			parsedAddr := net.ParseIP(newServerEvent.Address)
			if parsedAddr == nil {
				log.Warn("Received invalid address", zap.String("addr", newServerEvent.Address))

				continue
			}

			// Maybe ignore these and wait for connect call to create?
			var person domain.Person
			if errPerson := app.Store().GetOrCreatePersonBySteamID(ctx, newServerEvent.SID, &person); errPerson != nil {
				log.Error("Failed to load Person", zap.Error(errPerson))

				continue
			}

			conn := domain.PersonConnection{
				IPAddr:      parsedAddr,
				SteamID:     newServerEvent.SID,
				PersonaName: strings.ToValidUTF8(newServerEvent.Name, "_"),
				CreatedOn:   newServerEvent.CreatedOn,
				ServerID:    evt.ServerID,
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
// Registering receivers can be accomplished with app.eb.Broadcaster.
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

// UDP log sink.
func (app *App) initLogSrc(ctx context.Context) {
	logSrc, errLogSrc := logparse.NewUDPLogListener(app.log, app.Config().Log.SrcdsLogAddr,
		func(eventType logparse.EventType, event logparse.ServerEvent) {
			app.eb.Emit(event.EventType, event)
		})

	if errLogSrc != nil {
		app.log.Fatal("Failed to setup udp log src", zap.Error(errLogSrc))
	}

	app.logListener = logSrc

	// TODO run on server Config changes
	go app.updateSrcdsLogSecrets(ctx)

	app.logListener.Start(ctx)
}

func (app *App) updateSrcdsLogSecrets(ctx context.Context) {
	newSecrets := map[int]logparse.ServerIDMap{}
	serversCtx, cancelServers := context.WithTimeout(ctx, time.Second*5)

	defer cancelServers()

	servers, _, errServers := app.db.GetServers(serversCtx, domain.ServerQueryFilter{
		IncludeDisabled: false,
		QueryFilter:     domain.QueryFilter{Deleted: false},
	})
	if errServers != nil {
		app.log.Error("Failed to update srcds log secrets", zap.Error(errServers))

		return
	}

	for _, server := range servers {
		newSecrets[server.LogSecret] = logparse.ServerIDMap{
			ServerID:   server.ServerID,
			ServerName: server.ShortName,
		}
	}

	app.logListener.SetSecrets(newSecrets)
}

type NotificationHandler struct{}

func (app *App) SendNotification(ctx context.Context, notification NotificationPayload) error {
	return nil
}

// validateLink is used in the case of discord origin actions that require mapping the
// discord member ID to a SteamID so that we can track its use and apply permissions, etc.
//
// This function will replace the discord member id value in the target field with
// the found SteamID, if any.
// func validateLink(ctx context.Context, database db.postgreStore, sourceID action.Author, target *action.Author) error {
//	var p model.Person
//	if errGetPerson := database.GetPersonByDiscordID(ctx, string(sourceID), &p); errGetPerson != nil {
//		if errGetPerson == db.ErrNoResult {
//			return consts.ErrUnlinkedAccount
//		}
//		return consts.domain.ErrInternal
//	}
//	*target = action.Author(p.SteamID.String())
//	return nil
// }
