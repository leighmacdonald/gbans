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
			cu := config.NewConfigUsecase(config.NewConfigRepository())
			if errConfig := cu.Read(false); errConfig != nil {
				panic(fmt.Sprintf("Failed to read config: %v", errConfig))
			}

			conf := cu.Config()
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
			db := database.New(rootLogger, conf.DB.DSN, false, conf.DB.LogQueries)

			rootLogger.Info("Connecting to database")
			if errConnect := db.Connect(connCtx); errConnect != nil {
				rootLogger.Fatal("Failed to connect to database", zap.Error(errConnect))
			}
			defer func() {
				if errClose := db.Close(); errClose != nil {
					rootLogger.Error("Failed to close database cleanly", zap.Error(errClose))
				}
			}()

			if errDelete := db.Exec(ctx, "DELETE FROM person_messages_filter"); errDelete != nil {
				rootLogger.Fatal("Failed to delete existing", zap.Error(errDelete))
			}
			eb := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()
			wm := fp.NewMutexMap[logparse.Weapon, int]()

			sv := servers.NewServersUsecase(servers.NewServersRepository(db))
			st := state.NewStateUsecase(rootLogger, eb, state.NewStateRepository(state.NewCollector(rootLogger, sv)), cu, sv)
			pu := person.NewPersonUsecase(rootLogger, person.NewPersonRepository(db))
			au := asset.NewAssetUsecase(asset.NewS3Repository(rootLogger, db, nil, conf.S3.Region))
			meu := media.NewMediaUsecase(conf.S3.BucketMedia, media.NewMediaRepository(db), au)
			neu := news.NewNewsUsecase(news.NewNewsRepository(db))
			wu := wiki.NewWikiUsecase(wiki.NewWikiRepository(db, meu))
			mu := match.NewMatchUsecase(rootLogger, eb, match.NewMatchRepository(db, pu), st, sv, nil, wm)

			var owner domain.Person

			if errRootUser := pu.GetPersonBySteamID(ctx, conf.General.Owner, &owner); errRootUser != nil {
				if !errors.Is(errRootUser, domain.ErrNoResult) {
					rootLogger.Fatal("Failed checking owner state", zap.Error(errRootUser))
				}

				newOwner := domain.NewPerson(conf.General.Owner)
				newOwner.PermissionLevel = domain.PAdmin

				if errSave := pu.SavePerson(ctx, &newOwner); errSave != nil {
					rootLogger.Fatal("Failed create new owner", zap.Error(errSave))
				}

				newsEntry := domain.NewsEntry{
					Title:       "Welcome to gbans",
					BodyMD:      "This is an *example* **news** entry.",
					IsPublished: true,
					CreatedOn:   time.Now(),
					UpdatedOn:   time.Now(),
				}

				if errSave := neu.SaveNewsArticle(ctx, &newsEntry); errSave != nil {
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

				if errSave := sv.SaveServer(ctx, &server); errSave != nil {
					rootLogger.Fatal("Failed create example server entry", zap.Error(errSave))
				}

				page := domain.Page{
					Slug:      domain.RootSlug,
					BodyMD:    "# Welcome to the wiki",
					Revision:  1,
					CreatedOn: time.Now(),
					UpdatedOn: time.Now(),
				}

				if errSave := wu.SaveWikiPage(ctx, &page); errSave != nil {
					rootLogger.Fatal("Failed save example wiki entry", zap.Error(errSave))
				}
			}

			if errWeapons := mu.LoadWeapons(ctx, wm); errWeapons != nil {
				rootLogger.Fatal("Failed to import weapons", zap.Error(errWeapons))
			}

			os.Exit(0)
		},
	}
}
