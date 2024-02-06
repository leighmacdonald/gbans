package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/internal/match"
	"github.com/leighmacdonald/gbans/internal/media"
	"github.com/leighmacdonald/gbans/internal/news"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/wiki"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func setupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Run Initial Setup",
		Long:  `Run Initial Setup`,
		Run: func(cmd *cobra.Command, args []string) {
			configUsecase := config.NewConfigUsecase(config.NewConfigRepository())
			if errConfig := configUsecase.Read(false); errConfig != nil {
				panic(fmt.Sprintf("Failed to read config: %v", errConfig))
			}

			conf := configUsecase.Config()
			rootLogger := log.MustCreate(conf, nil)
			defer func() {
				_ = rootLogger.Sync()
			}()

			defer func() {
				_ = rootLogger.Sync()
			}()

			ctx := context.Background()

			connCtx, cancelConn := context.WithTimeout(ctx, time.Second*5)
			defer cancelConn()
			databaseRepository := database.New(rootLogger, conf.DB.DSN, false, conf.DB.LogQueries)

			rootLogger.Info("Connecting to database")
			if errConnect := databaseRepository.Connect(connCtx); errConnect != nil {
				rootLogger.Fatal("Failed to connect to database", zap.Error(errConnect))
			}
			defer func() {
				if errClose := databaseRepository.Close(); errClose != nil {
					rootLogger.Error("Failed to close database cleanly", zap.Error(errClose))
				}
			}()

			if //goland:noinspection ALL
			errDelete := databaseRepository.Exec(ctx, "DELETE FROM person_messages_filter"); errDelete != nil {
				rootLogger.Fatal("Failed to delete existing", zap.Error(errDelete))
			}
			broadcaster := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()
			weaponMap := fp.NewMutexMap[logparse.Weapon, int]()

			serversUsecase := servers.NewServersUsecase(servers.NewServersRepository(databaseRepository))
			stateUsecase := state.NewStateUsecase(rootLogger, broadcaster, state.NewStateRepository(state.NewCollector(rootLogger, serversUsecase)), configUsecase, serversUsecase)
			personUsecase := person.NewPersonUsecase(rootLogger, person.NewPersonRepository(databaseRepository), configUsecase)
			assetUsecase := asset.NewAssetUsecase(asset.NewS3Repository(rootLogger, databaseRepository, nil, conf.S3.Region))
			mediaUsecase := media.NewMediaUsecase(conf.S3.BucketMedia, media.NewMediaRepository(databaseRepository), assetUsecase)
			newsUsecase := news.NewNewsUsecase(news.NewNewsRepository(databaseRepository))
			wikiUsecase := wiki.NewWikiUsecase(wiki.NewWikiRepository(databaseRepository, mediaUsecase))
			matchUsecase := match.NewMatchUsecase(rootLogger, broadcaster, match.NewMatchRepository(databaseRepository, personUsecase), stateUsecase, serversUsecase, nil, weaponMap)

			owner, errRootUser := personUsecase.GetPersonBySteamID(ctx, conf.General.Owner)
			if errRootUser != nil {
				if !errors.Is(errRootUser, domain.ErrNoResult) {
					rootLogger.Fatal("Failed checking owner state", zap.Error(errRootUser))
				}

				newOwner := domain.NewPerson(conf.General.Owner)
				newOwner.PermissionLevel = domain.PAdmin

				if errSave := personUsecase.SavePerson(ctx, &newOwner); errSave != nil {
					rootLogger.Fatal("Failed create new owner", zap.Error(errSave))
				}

				newsEntry := domain.NewsEntry{
					Title:       "Welcome to gbans",
					BodyMD:      "This is an *example* **news** entry.",
					IsPublished: true,
					CreatedOn:   time.Now(),
					UpdatedOn:   time.Now(),
				}

				if errSave := newsUsecase.SaveNewsArticle(ctx, &newsEntry); errSave != nil {
					rootLogger.Fatal("Failed create example news entry", zap.Error(errSave))
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
					rootLogger.Fatal("Failed create example server entry", zap.Error(errSave))
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
					rootLogger.Fatal("Failed save example wiki entry", zap.Error(errSave))
				}
			}

			if errWeapons := matchUsecase.LoadWeapons(ctx, weaponMap); errWeapons != nil {
				rootLogger.Fatal("Failed to import weapons", zap.Error(errWeapons))
			}

			os.Exit(0)
		},
	}
}
