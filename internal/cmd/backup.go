package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/gokrazy/rsync/rsynccmd"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/spf13/cobra"
)

var (
	errBackup = errors.New("backup failed")
	username  = "" //nolint:gochecknoglobals
	backupDir = "" //nolint:gochecknoglobals
	baseDir   = "" //nolint:gochecknoglobals
)

func backupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backup",
		Short: "Backup Servers",
		Long:  `Download all saved data from active servers (logs & stvs) to a local directory`,
		RunE: func(cmd *cobra.Command, _ []string) error {
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

			servers, errServers := app.servers.Servers(ctx, servers.Query{IncludeDisabled: false, IncludeDeleted: false})
			if errServers != nil {
				return errors.Join(errServers, errBackup)
			}

			for _, server := range servers {
				for _, name := range []string{"logs" /* , "stv_demos"*/} {
					remotePath := path.Join(baseDir, "srcds-"+server.ShortName, "tf", name)
					targetURI := fmt.Sprintf("%s@%s:%s", username, server.AddressInternal, remotePath)
					outputPath := path.Join(backupDir, server.ShortName, name)
					if err := os.MkdirAll(outputPath, 0o775); err != nil {
						return errors.Join(err, errBackup)
					}
					// z isnt supported?? (nixos)
					slog.Info("Backing up game data", slog.String("server", server.ShortName), slog.String("path", outputPath))
					cmd := rsynccmd.Command("rsync", "-av", targetURI, outputPath)
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					if _, err := cmd.Run(context.Background()); err != nil {
						return errors.Join(err, errBackup)
					}
				}
			}

			return nil
		},
	}
}
