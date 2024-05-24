package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/match"
	"github.com/leighmacdonald/gbans/internal/news"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/wiki"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/spf13/cobra"
)

func setupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Run Initial Setup",
		Long:  `Run Initial Setup`,
		Run: func(_ *cobra.Command, _ []string) {
			configUsecase := config.NewConfigUsecase(config.NewConfigRepository())
			if errConfig := configUsecase.Read(false); errConfig != nil {
				panic(fmt.Sprintf("Failed to read config: %v", errConfig))
			}

			conf := configUsecase.Config()

			logCloser := log.MustCreateLogger(conf.Log.File, conf.Log.Level)
			defer logCloser()

			ctx := context.Background()

			connCtx, cancelConn := context.WithTimeout(ctx, time.Second*5)
			defer cancelConn()
			databaseRepository := database.New(conf.DB.DSN, false, conf.DB.LogQueries)

			slog.Info("Connecting to database")
			if errConnect := databaseRepository.Connect(connCtx); errConnect != nil {
				slog.Error("Failed to connect to database", log.ErrAttr(errConnect))

				return
			}

			defer func() {
				if errClose := databaseRepository.Close(); errClose != nil {
					slog.Error("Failed to close database cleanly", log.ErrAttr(errClose))
				}
			}()

			if //goland:noinspection ALL
			errDelete := databaseRepository.Exec(ctx, "DELETE FROM person_messages_filter"); errDelete != nil {
				slog.Error("Failed to delete existing", log.ErrAttr(errDelete))

				return
			}
			broadcaster := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()
			weaponMap := fp.NewMutexMap[logparse.Weapon, int]()

			serversUsecase := servers.NewServersUsecase(servers.NewServersRepository(databaseRepository))
			stateUsecase := state.NewStateUsecase(broadcaster, state.NewStateRepository(state.NewCollector(serversUsecase)), configUsecase, serversUsecase)
			personUsecase := person.NewPersonUsecase(person.NewPersonRepository(databaseRepository), configUsecase)

			assetRepo := asset.NewLocalRepository(databaseRepository, configUsecase)
			if errAssetInit := assetRepo.Init(ctx); errAssetInit != nil {
				slog.Error("Failed to init local asset repo", log.ErrAttr(errAssetInit))

				return
			}

			newsUsecase := news.NewNewsUsecase(news.NewNewsRepository(databaseRepository))
			wikiUsecase := wiki.NewWikiUsecase(wiki.NewWikiRepository(databaseRepository))
			matchRepo := match.NewMatchRepository(broadcaster, databaseRepository, personUsecase, serversUsecase, nil, stateUsecase, weaponMap)
			matchUsecase := match.NewMatchUsecase(matchRepo, stateUsecase, serversUsecase, nil)

			owner, errRootUser := personUsecase.GetPersonBySteamID(ctx, steamid.New(conf.General.Owner))
			if errRootUser != nil {
				if !errors.Is(errRootUser, domain.ErrNoResult) {
					slog.Error("Failed checking owner state", log.ErrAttr(errRootUser))
				}

				newOwner := domain.NewPerson(steamid.New(conf.General.Owner))
				newOwner.PermissionLevel = domain.PAdmin

				if errSave := personUsecase.SavePerson(ctx, &newOwner); errSave != nil {
					slog.Error("Failed create new owner", log.ErrAttr(errSave))
				}

				newsEntry := domain.NewsEntry{
					Title:       "Welcome to gbans",
					BodyMD:      "This is an *example* **news** entry.",
					IsPublished: true,
					CreatedOn:   time.Now(),
					UpdatedOn:   time.Now(),
				}

				if errSave := newsUsecase.SaveNewsArticle(ctx, &newsEntry); errSave != nil {
					slog.Error("Failed create example news entry", log.ErrAttr(errSave))

					return
				}

				server := domain.NewServer("server-1", "127.0.0.1", 27015)
				server.CC = "jp"
				server.RCON = "example_rcon"
				server.Latitude = 35.652832
				server.Longitude = 139.839478
				server.Name = "Example ServerStore"
				server.LogSecret = 12345678
				server.Region = "asia"

				if errSave := serversUsecase.SaveServer(ctx, &server); errSave != nil {
					slog.Error("Failed create example server entry", log.ErrAttr(errSave))

					return
				}

				page := domain.WikiPage{
					Slug:      domain.RootSlug,
					BodyMD:    "# Welcome to the wiki",
					Revision:  1,
					CreatedOn: time.Now(),
					UpdatedOn: time.Now(),
				}

				_, errSave := wikiUsecase.SaveWikiPage(ctx, owner, page.Slug, page.BodyMD, page.PermissionLevel)
				if errSave != nil {
					slog.Error("Failed save example wiki entry", log.ErrAttr(errSave))
				}
			}

			if errWeapons := matchUsecase.LoadWeapons(ctx, weaponMap); errWeapons != nil {
				slog.Error("Failed to import weapons", log.ErrAttr(errWeapons))
			}

			os.Exit(0)
		},
	}
}
