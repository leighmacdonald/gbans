package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
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

// defaultImportWorkers caps concurrent demo imports. Each worker transiently
// holds several copies of a full demo in memory (raw bytes, parse request,
// parse response), so this is intentionally low.
const defaultImportWorkers = 4

// maxImportFileSize mirrors asset.maxDemoFileSize. Oversized files are
// rejected during import anyway; skipping them upfront avoids streaming
// gigabytes that can never be stored.
const maxImportFileSize = 500000000

func importDemoCmd() *cobra.Command {
	var (
		serverName         string
		force              bool
		workers            int
		skipProfileRefresh bool
	)

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

			if skipProfileRefresh {
				app.persons.DisableProfileRefresh()
			}

			if workers < 1 {
				workers = 1
			}

			importFn := func(arg string) error {
				if info, errStat := os.Stat(arg); errStat == nil && !info.IsDir() && info.Size() > maxImportFileSize {
					return fmt.Errorf("%w: file exceeds size limit: %s", demo.ErrDemoLoad, arg)
				}

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

				_, errImport := app.demos.ImportFile(ctx, serverID, arg, true, force)
				if errImport != nil {
					return errImport
				}

				return nil
			}

			for _, arg := range args {
				if isDir(arg) {
					var (
						sem       = make(chan struct{}, workers)
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
						sem <- struct{}{}
						go func(filePath string, entryName string) {
							defer waitGroup.Done()
							defer func() { <-sem }()

							if err := importFn(path.Join(filePath, entryName)); err != nil && !errors.Is(err, database.ErrDuplicate) {
								slog.Error("Failed to import dir", slog.String("error", err.Error()))
							}
						}(arg, entry.Name())
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
	cmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "Force re-import of already imported demos")
	cmd.PersistentFlags().IntVarP(&workers, "workers", "w", defaultImportWorkers, "Max concurrent demo imports")
	cmd.PersistentFlags().BoolVar(&skipProfileRefresh, "skip-profile-refresh", false, "Skip refreshing player profile data from Steam during import")

	return cmd
}

func isDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fileInfo.IsDir()
}
