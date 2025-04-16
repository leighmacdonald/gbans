package demo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/viant/afs/option"
	"github.com/viant/afs/storage"
)

type UploadedDemo struct {
	Name    string
	Server  domain.Server
	Content []byte
}

type Fetcher struct {
	database       database.Database
	anticheat      domain.AntiCheatUsecase
	serversUsecase domain.ServersUsecase
	configUsecase  domain.ConfigUsecase
	assetUsecase   domain.AssetUsecase
	demoUsecase    domain.DemoUsecase
	matchUsecase   domain.MatchUsecase
	demoChan       chan UploadedDemo
	parserMu       *sync.Mutex
}

func NewFetcher(database database.Database, configUsecase domain.ConfigUsecase, serversUsecase domain.ServersUsecase,
	assetUsecase domain.AssetUsecase, demoUsecase domain.DemoUsecase, matchUsecase domain.MatchUsecase, anticheat domain.AntiCheatUsecase,
) *Fetcher {
	return &Fetcher{
		database:       database,
		configUsecase:  configUsecase,
		serversUsecase: serversUsecase,
		assetUsecase:   assetUsecase,
		demoUsecase:    demoUsecase,
		matchUsecase:   matchUsecase,
		anticheat:      anticheat,
		demoChan:       make(chan UploadedDemo),
		parserMu:       &sync.Mutex{},
	}
}

func (d Fetcher) OnDemoReceived(ctx context.Context, demo UploadedDemo) error {
	slog.Debug("Got new demo",
		slog.String("server", demo.Server.ShortName),
		slog.String("name", demo.Name))

	demoAsset, errNewAsset := d.assetUsecase.Create(ctx, steamid.New(d.configUsecase.Config().Owner),
		domain.BucketDemo, demo.Name, bytes.NewReader(demo.Content))
	if errNewAsset != nil {
		return errNewAsset
	}

	d.parserMu.Lock()
	defer d.parserMu.Unlock()

	_, errDemo := d.demoUsecase.CreateFromAsset(ctx, demoAsset, demo.Server.ServerID)
	if errDemo != nil {
		//_, errMatch := d.matchUsecase.CreateFromDemo(ctx, demo.Server.ServerID, demo)
		//		if errMatch != nil {
		// Cleanup the asset not attached to a demo
		if _, errDelete := d.assetUsecase.Delete(ctx, demoAsset.AssetID); errDelete != nil {
			return errors.Join(errDelete, errDelete)
		}
		//		}

		return errDemo
	}

	return nil
}

var (
	errFailedToList   = errors.New("failed to list files")
	errFailedOpenFile = errors.New("failed to open file")
	errFailedReadFile = errors.New("failed to read file")
	errCloseReader    = errors.New("failed to close file reader")
)

func (d Fetcher) fetchDemos(ctx context.Context, demoPathFmt string, server domain.Server, client storage.Storager) error {
	demoDir := fmt.Sprintf(demoPathFmt, server.ShortName)

	filelist, errFilelist := client.List(ctx, demoDir, option.NewPage(0, 1))
	if errFilelist != nil {
		slog.Error("remote list dir failed", log.ErrAttr(errFailedToList),
			slog.String("server", server.ShortName), slog.String("path", demoDir))

		return nil //nolint:nilerr
	}

	for _, file := range filelist {
		if !strings.HasSuffix(file.Name(), ".dem") {
			continue
		}

		demoPath := path.Join(demoDir, file.Name())

		slog.Info("Downloading demo", slog.String("name", file.Name()), slog.String("server", server.ShortName))

		reader, err := client.Open(ctx, demoPath)
		if err != nil {
			return errors.Join(err, errFailedOpenFile)
		}

		data, errRead := io.ReadAll(reader)
		if errRead != nil {
			_ = reader.Close()

			return errors.Join(errRead, errFailedReadFile)
		}

		if errClose := reader.Close(); errClose != nil {
			return errors.Join(errClose, errCloseReader)
		}

		// need Seeker, but afs does not provide
		demo := UploadedDemo{Name: file.Name(), Server: server, Content: data}

		if errDemo := d.OnDemoReceived(ctx, demo); errDemo != nil {
			if !errors.Is(errDemo, domain.ErrAssetTooLarge) {
				slog.Error("Failed to create new demo asset", log.ErrAttr(errDemo))

				continue
			}
		}

		if errDelete := client.Delete(ctx, demoPath); errDelete != nil {
			slog.Error("Failed to cleanup demo", log.ErrAttr(errDelete), slog.String("path", demoPath))
		}

		slog.Info("Deleted demo on remote host", slog.String("path", demoPath))
	}

	return nil
}

func (d Fetcher) fetchStacLogs(ctx context.Context, stactPathFmt string, server domain.Server, client storage.Storager) error {
	logDir := fmt.Sprintf(stactPathFmt, server.ShortName)

	filelist, errFilelist := client.List(ctx, logDir, option.NewPage(0, 1))
	if errFilelist != nil {
		slog.Error("remote list dir failed", log.ErrAttr(errFailedToList),
			slog.String("server", server.ShortName), slog.String("path", logDir))

		return nil //nolint:nilerr
	}

	for _, file := range filelist {
		if !strings.HasSuffix(file.Name(), ".log") {
			continue
		}

		logPath := path.Join(logDir, file.Name())

		reader, err := client.Open(ctx, logPath)
		if err != nil {
			return errors.Join(err, errFailedOpenFile)
		}

		slog.Debug("Importing stac log", slog.String("name", file.Name()), slog.String("server", server.ShortName))
		entries, errImport := d.anticheat.Import(ctx, file.Name(), reader, server.ServerID)
		if errImport != nil && !errors.Is(errImport, domain.ErrDuplicate) {
			slog.Error("Failed to import stac logs", log.ErrAttr(errImport))
		} else if len(entries) > 0 {
			if errHandle := d.anticheat.Handle(ctx, entries); errHandle != nil {
				slog.Error("Failed to handle stac logs", log.ErrAttr(errHandle))
			}
		}

		if errClose := reader.Close(); errClose != nil {
			return errors.Join(errClose, errCloseReader)
		}
	}

	return nil
}

func (d Fetcher) OnClientConnect(ctx context.Context, client storage.Storager, servers []domain.Server) error {
	config := d.configUsecase.Config()
	for _, server := range servers {
		if config.General.DemosEnabled {
			slog.Debug("Fetching demos")
			if err := d.fetchDemos(ctx, d.configUsecase.Config().SSH.DemoPathFmt, server, client); err != nil {
				slog.Error("Failed to fetch demos", log.ErrAttr(err))
			}
		}

		if config.Anticheat.Enabled {
			slog.Debug("Fetching anticheat logs", slog.String("server", server.ShortName))
			if err := d.fetchStacLogs(ctx, d.configUsecase.Config().SSH.StacPathFmt, server, client); err != nil {
				slog.Error("Failed to fetch stac logs", log.ErrAttr(err))
			}
		}
	}

	return nil
}

func NewDownloader(config domain.ConfigUsecase, dbConn database.Database, servers domain.ServersUsecase,
	assets domain.AssetUsecase, demos domain.DemoUsecase, matchUsecase domain.MatchUsecase, anticheat domain.AntiCheatUsecase,
) Downloader {
	fetcher := NewFetcher(dbConn, config, servers, assets, demos, matchUsecase, anticheat)

	return Downloader{
		fetcher: fetcher,
		scpExec: network.NewSCPExecer(dbConn, config, servers, fetcher.OnClientConnect),
		config:  config,
	}
}

type Downloader struct {
	fetcher *Fetcher
	scpExec network.SCPExecer
	config  domain.ConfigUsecase
}

// Start begins the background task scheduler which peridodically will run the provided SCPExecer.Update function.
func (d Downloader) Start(ctx context.Context) {
	seconds := d.config.Config().SSH.UpdateInterval
	interval := time.Duration(seconds) * time.Second
	if interval < time.Minute*5 {
		slog.Warn("Interval is too short, overriding to 5 minutes", slog.Duration("interval", interval))
		//		interval = time.Minute * 5
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
