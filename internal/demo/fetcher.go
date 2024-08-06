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
	serversUsecase domain.ServersUsecase
	configUsecase  domain.ConfigUsecase
	assetUsecase   domain.AssetUsecase
	demoUsecase    domain.DemoUsecase
	demoChan       chan UploadedDemo
}

func NewFetcher(database database.Database, configUsecase domain.ConfigUsecase, serversUsecase domain.ServersUsecase,
	assetUsecase domain.AssetUsecase, demoUsecase domain.DemoUsecase,
) Fetcher {
	return Fetcher{
		database:       database,
		configUsecase:  configUsecase,
		serversUsecase: serversUsecase,
		assetUsecase:   assetUsecase,
		demoUsecase:    demoUsecase,
		demoChan:       make(chan UploadedDemo),
	}
}

func (d Fetcher) Start(ctx context.Context) {
	sshExec := network.NewSCPExecer(d.database, d.configUsecase, d.serversUsecase, d.OnClientConnect)
	sshExec.Start(ctx)
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

	_, errDemo := d.demoUsecase.CreateFromAsset(ctx, demoAsset, demo.Server.ServerID)
	if errDemo != nil {
		// Cleanup the asset not attached to a demo
		if _, errDelete := d.assetUsecase.Delete(ctx, demoAsset.AssetID); errDelete != nil {
			return errors.Join(errDelete, errDelete)
		}

		return errDemo
	}

	return nil
}

var (
	errFailedToList   = errors.New("failed to list files")
	errFailedOpenFile = errors.New("failed to open file")
	errFailedReadFile = errors.New("failed to read file")
)

func (d Fetcher) OnClientConnect(ctx context.Context, client storage.Storager, servers []domain.Server) error {
	demoPathFmt := d.configUsecase.Config().SSH.DemoPathFmt

	for _, server := range servers {
		demoDir := fmt.Sprintf(demoPathFmt, server.ShortName)

		filelist, errFilelist := client.List(ctx, demoDir, option.NewPage(0, 1))
		if errFilelist != nil {
			slog.Error("remote list dir failed", log.ErrAttr(errFailedToList),
				slog.String("server", server.ShortName), slog.String("path", demoDir))

			continue
		}

		for _, file := range filelist {
			if !strings.HasSuffix(file.Name(), ".dem") {
				continue
			}

			demoPath := path.Join(demoDir, file.Name())

			slog.Debug("Downloading demo", slog.String("name", file.Name()))

			reader, err := client.Open(ctx, demoPath)
			if err != nil {
				return errors.Join(err, errFailedOpenFile)
			}

			data, errRead := io.ReadAll(reader)
			if errRead != nil {
				_ = reader.Close()

				return errors.Join(errRead, errFailedReadFile)
			}

			_ = reader.Close()

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

			slog.Debug("Deleted demo on remote host", slog.String("path", demoPath))
		}
	}

	return nil
}
