package demo

import (
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
	"github.com/viant/afs/storage"
)

type demoUpdate struct {
	name      string
	server    domain.Server
	demoBytes []byte
}

type Fetcher struct {
	database       database.Database
	serversUsecase domain.ServersUsecase
	configUsecase  domain.ConfigUsecase
	demoChan       chan demoUpdate
}

func NewFetcher(database database.Database, configUsecase domain.ConfigUsecase, serversUsecase domain.ServersUsecase) Fetcher {
	return Fetcher{
		database:       database,
		configUsecase:  configUsecase,
		serversUsecase: serversUsecase,
		demoChan:       make(chan demoUpdate),
	}
}

func (d Fetcher) Start(ctx context.Context) {
	sshExec := network.NewSCPExecer(d.database, d.configUsecase, d.serversUsecase, d.OnClientConnect)
	go sshExec.Start(ctx)

	for {
		select {
		case newDemo := <-d.demoChan:
			slog.Info("got new demo",
				slog.String("server", newDemo.server.ShortName),
				slog.String("name", newDemo.name),
				slog.Int("size", len(newDemo.demoBytes)))
		case <-ctx.Done():
			return
		}
	}
}

var (
	errFailedToList   = errors.New("failed to list files")
	errFailedOpenFile = errors.New("failed to open file")
	errFailedReadFile = errors.New("failed to read file")
)

func (d Fetcher) OnClientConnect(ctx context.Context, client storage.Storager, servers []domain.Server) error {
	for _, server := range servers {
		demoDir := fmt.Sprintf("~/srcds-%s/tf/demos", server.ShortName)

		filelist, errFilelist := client.List(ctx, demoDir)
		if errFilelist != nil {
			slog.Error("remote list dir failed", log.ErrAttr(errFailedToList))

			continue
		}

		for _, file := range filelist {
			if !strings.HasSuffix(file.Name(), ".dem") {
				continue
			}

			reader, err := client.Open(ctx, path.Join(demoDir, file.Name()))
			if err != nil {
				return errors.Join(err, errFailedOpenFile)
			}

			data, errRead := io.ReadAll(reader)
			if errRead != nil {
				_ = reader.Close()

				return errors.Join(errRead, errFailedReadFile)
			}

			_ = reader.Close()

			d.demoChan <- demoUpdate{
				name:      file.Name(),
				server:    server,
				demoBytes: data,
			}
		}
	}

	return nil
}
