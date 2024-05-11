package demo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/viant/afs/storage"
)

type demoUpdate struct {
	name      string
	server    domain.Server
	demoBytes []byte
}

type DemoFetcher struct {
	serversUsecase domain.ServersUsecase
	configUsecase  domain.ConfigUsecase
	demoChan       chan demoUpdate
}

func NewDemoFetcher(configUsecase domain.ConfigUsecase, serversUsecase domain.ServersUsecase) DemoFetcher {
	return DemoFetcher{
		configUsecase:  configUsecase,
		serversUsecase: serversUsecase,
		demoChan:       make(chan demoUpdate),
	}
}

func (d DemoFetcher) Start(ctx context.Context) {
	sshExec := network.NewSCPExecer(d.configUsecase, d.serversUsecase, d.OnClientConnect)
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

func (d DemoFetcher) OnClientConnect(ctx context.Context, client storage.Storager, server domain.Server) error {
	demoDir := fmt.Sprintf("~/srcds-%s/tf/demos", server.ShortName)

	filelist, errFilelist := client.List(ctx, demoDir)
	if errFilelist != nil {
		return errors.Join(errFilelist, errFailedToList)
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

	return nil
}
