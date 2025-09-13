package scp

import (
	"context"
	"log/slog"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/pkg/log"
)

// type Fetcher struct {
// 	database      database.Database
// 	configUsecase *config.ConfigUsecase
// 	parserMu      *sync.Mutex
// }

// func NewFetcher(database database.Database, configUsecase *config.ConfigUsecase) *Fetcher {
// 	return &Fetcher{
// 		database:      database,
// 		configUsecase: configUsecase,
// 		parserMu:      &sync.Mutex{},
// 	}
// }

// func (d Fetcher) OnClientConnect(ctx context.Context, client storage.Storager, servers []servers.Server) error {
// 	config := d.configUsecase.Config()
// 	for _, server := range servers {
// 		if config.General.DemosEnabled {
// 			slog.Debug("Fetching demos")
// 			if err := d.fetchDemos(ctx, d.configUsecase.Config().SSH.DemoPathFmt, server, client); err != nil {
// 				slog.Error("Failed to fetch demos", log.ErrAttr(err))
// 			}
// 		}

// 		if config.Anticheat.Enabled {
// 			slog.Debug("Fetching anticheat logs", slog.String("server", server.ShortName))
// 			if err := d.fetchStacLogs(ctx, d.configUsecase.Config().SSH.StacPathFmt, server, client); err != nil {
// 				slog.Error("Failed to fetch stac logs", log.ErrAttr(err))
// 			}
// 		}
// 	}

// 	return nil
// }

func NewDownloader(config *config.ConfigUsecase, dbConn database.Database) Downloader {
	return Downloader{
		scpExec: NewSCPConnection(dbConn, config),
		config:  config,
	}
}

type Downloader struct {
	scpExec SCPConnection
	config  *config.ConfigUsecase
}

// Start begins the background task scheduler which peridodically will run the provided SCPExecer.Update function.
func (d Downloader) Start(ctx context.Context) {
	seconds := d.config.Config().SSH.UpdateInterval
	interval := time.Duration(seconds) * time.Second
	if interval < time.Minute*5 {
		slog.Warn("Interval is too short, overriding to 5 minutes", slog.Duration("interval", interval))
		interval = time.Minute * 5
	}

	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			conf := d.config.Config()
			if !conf.SSH.Enabled || !conf.General.DemosEnabled && !conf.Anticheat.Enabled {
				// Only perform SSH connection if we actually have at least one task that requires it enabled.
				continue
			}

			if err := d.scpExec.Update(ctx); err != nil {
				slog.Error("Error trying to download demos", log.ErrAttr(err))
			}
		case <-ctx.Done():
			return
		}
	}
}
