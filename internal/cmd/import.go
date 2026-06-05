package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"runtime"
	"strconv"
	"sync"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/demo"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/spf13/cobra"
)

func importCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import existing data",
	}
	cmd.AddCommand(importDemoCmd())

	return cmd
}

func importDemoCmd() *cobra.Command {
	var serverName string
	cmd := &cobra.Command{
		Use:   "demo [DEMOFILE] [DEMODIR]",
		Short: "Import a demo",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if serverName == "" {
				return demo.ErrDemoLoad
			}

			ctx := cmd.Context()
			app, errApp := New()
			if errApp != nil {
				return errApp
			}

			defer func() {
				if errClose := app.Shutdown(ctx); errClose != nil {
					slog.Error("Error closing", slog.String("error", errClose.Error()))
				}
			}()

			if errSetup := app.Init(ctx); errSetup != nil {
				return errSetup
			}

			importFn := func(arg string) error {
				var serverID int32
				parsedID, errID := strconv.ParseInt(arg, 10, 32)
				if errID != nil {
					server, errServer := app.servers.Servers(ctx, servers.Query{ShortName: serverName, IncludeDisabled: true})
					if errServer != nil {
						return errors.Join(errServer, demo.ErrDemoLoad)
					}
					if len(server) != 1 {
						return fmt.Errorf("%w: Invalid server %s", demo.ErrDemoLoad, serverName)
					}
					serverID = server[0].ServerID
				} else {
					serverID = int32(parsedID)
				}

				importedDemo, errImport := app.demos.ImportFile(ctx, serverID, arg)
				if errImport != nil {
					return errImport
				}

				slog.Info("Imported demo", slog.String("name", importedDemo.ServerNameShort), slog.Int("serverID", int(serverID)))

				return nil
			}

			for _, arg := range args {
				if isDir(arg) {
					var (
						sem       = make(chan struct{}, runtime.NumCPU())
						waitGroup sync.WaitGroup
					)

					entries, err := os.ReadDir(arg)
					if err != nil {
						return errors.Join(err, demo.ErrDemoLoad)
					}
					for _, entry := range entries {
						if entry.IsDir() {
							continue
						}
						waitGroup.Add(1)
						go func(filePath string) {
							defer waitGroup.Done()
							sem <- struct{}{}
							defer func() { <-sem }()

							if err := importFn(path.Join(filePath, entry.Name())); err != nil && !errors.Is(err, database.ErrDuplicate) {
								slog.Error("Failed to import dir", slog.String("error", err.Error()))
							}
						}(arg)
					}
					waitGroup.Wait()
				} else {
					if err := importFn(arg); err != nil {
						slog.Error("Failed to import", slog.String("error", err.Error()))

						return errors.Join(err, demo.ErrDemoLoad)
					}
				}
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&serverName, "server", "s", "", "Shorthand name of the server (srv-1), or its numerical server id (6)")

	return cmd
}

func isDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fileInfo.IsDir()
}
