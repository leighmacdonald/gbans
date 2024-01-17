package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func importCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import",
		Short: "Import data from 3rd party",
		Long:  `Import data from 3rd party`,
	}
}

func importConnectionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "chatlog-conn",
		Short: "Import chat logs connections",
		Long:  `Import chat logs connections`,
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			rootCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
			defer stop()

			var conf config.Config
			if errConfig := config.Read(&conf, false); errConfig != nil {
				panic("Failed to read config")
			}

			rootLogger := log.MustCreate(&conf, nil)
			defer func() {
				if conf.Log.File != "" {
					_ = rootLogger.Sync()
				}
			}()

			database := store.New(rootLogger, conf.DB.DSN, conf.DB.AutoMigrate, conf.DB.LogQueries)
			if errConnect := database.Connect(rootCtx); errConnect != nil {
				rootLogger.Fatal("Cannot initialize database", zap.Error(errConnect))
			}

			defer func() {
				if errClose := database.Close(); errClose != nil {
					rootLogger.Error("Failed to close database cleanly")
				}
			}()

			connFile, errConnFile := os.Open("connections.json")
			if errConnFile != nil {
				rootLogger.Fatal("Failed to open connection.json")
			}
			defer func() {
				_ = connFile.Close()
			}()

			scanner := bufio.NewScanner(connFile)
			scanner.Split(bufio.ScanLines)

			playerExistsCache := map[steamid.SID64]bool{}

			if errTruncate := database.Exec(ctx, "TRUNCATE person_connections"); errTruncate != nil {
				rootLogger.Fatal("Failed to truncate person_connections")
			}

			rootLogger.Info("Truncated person_connections")

			for scanner.Scan() {
				line := bytes.ReplaceAll(scanner.Bytes(), []byte("\\n "), []byte{' '})
				var conn map[string]any
				if errUnmarshal := json.Unmarshal(line, &conn); errUnmarshal != nil {
					rootLogger.Error("Failed to unmarshal row", zap.Error(errUnmarshal))

					continue
				}
				sidString, sidOk := conn["player_steamid3"].(string)
				if !sidOk {
					continue
				}

				sid := steamid.New(sidString)
				if !sid.Valid() {
					rootLogger.Error("Failed to decode steamid", zap.String("sid", sidString))

					continue
				}

				if _, playerFound := playerExistsCache[sid]; !playerFound {
					// Satisfy fk
					var person store.Person
					if errPerson := database.GetOrCreatePersonBySteamID(ctx, sid, &person); errPerson != nil {
						rootLogger.Error("Failed to get person", zap.Error(errPerson))

						continue
					}
					playerExistsCache[sid] = true
				}

				ipString, ipOk := conn["ip"].(string)
				if !ipOk {
					continue
				}

				ipAddr := net.ParseIP(ipString)
				if ipAddr == nil {
					rootLogger.Error("Failed to parse IP", zap.String("ip", ipString))

					continue
				}

				createdString, createdOk := conn["created_at"].(string)
				if !createdOk {
					continue
				}

				createdPieces := strings.Split(createdString, ".")
				if len(createdPieces) != 2 {
					rootLogger.Error("Failed to split time")

					continue
				}

				createdAt, errCreatedAt := time.Parse("2006-01-02T15:04:05", createdPieces[0])
				if errCreatedAt != nil {
					rootLogger.Error("Failed to parse time", zap.Error(errCreatedAt))

					continue
				}

				name, nameOk := conn["player_name"].(string)
				if !nameOk {
					continue
				}

				if errAdd := database.AddConnectionHistory(ctx, &store.PersonConnection{
					IPAddr:      ipAddr,
					SteamID:     sid,
					PersonaName: name,
					CreatedOn:   createdAt,
				}); errAdd != nil {
					rootLogger.Error("Failed to add conn", zap.Error(errAdd))

					continue
				}
			}
		},
	}
}

func importMessagesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "chatlog-messages",
		Short: "Import chat logs messages",
		Long:  `Import chat logs messages`,
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			rootCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
			defer stop()

			var conf config.Config
			if errConfig := config.Read(&conf, false); errConfig != nil {
				panic("Failed to read config")
			}

			rootLogger := log.MustCreate(&conf, nil)
			defer func() {
				if conf.Log.File != "" {
					_ = rootLogger.Sync()
				}
			}()

			database := store.New(rootLogger, conf.DB.DSN, conf.DB.AutoMigrate, conf.DB.LogQueries)
			if errConnect := database.Connect(rootCtx); errConnect != nil {
				rootLogger.Fatal("Cannot initialize database", zap.Error(errConnect))
			}

			defer util.LogCloser(database, rootLogger)

			connFile, errConnFile := os.Open("messages.json")
			if errConnFile != nil {
				rootLogger.Fatal("Failed to open messages.json")
			}

			defer util.LogCloser(connFile, rootLogger)

			scanner := bufio.NewScanner(connFile)
			scanner.Split(bufio.ScanLines)

			playerExistsCache := map[steamid.SID64]bool{}

			serverCache := map[string]int{}

			if errTruncate := database.Exec(ctx, "TRUNCATE person_messages"); errTruncate != nil {
				rootLogger.Fatal("Failed to truncate person_messages")
			}

			rootLogger.Info("Truncated person_messages")

			for scanner.Scan() {
				line := bytes.ReplaceAll(scanner.Bytes(), []byte("\\n "), []byte{' '})

				var msg map[string]any
				if errUnmarshal := json.Unmarshal(line, &msg); errUnmarshal != nil {
					rootLogger.Error("Failed to unmarshal row", zap.Error(errUnmarshal))

					continue
				}

				sidString, sidOk := msg["player_steamid3"].(string)
				if !sidOk {
					continue
				}

				sid := steamid.New(sidString)
				if !sid.Valid() {
					rootLogger.Error("Failed to decode steamid", zap.String("sid", sidString))

					continue
				}

				if _, playerFound := playerExistsCache[sid]; !playerFound {
					// Satisfy fk
					var person store.Person
					if errPerson := database.GetOrCreatePersonBySteamID(ctx, sid, &person); errPerson != nil {
						rootLogger.Error("Failed to get person", zap.Error(errPerson))

						continue
					}

					playerExistsCache[sid] = true
				}

				serverName, serverNameOk := msg["name"].(string)
				if !serverNameOk {
					continue
				}

				serverID, serverFound := serverCache[serverName]
				if !serverFound {
					var server store.Server
					if errServer := database.GetServerByName(ctx, serverName, &server, true, true); errServer != nil {
						rootLogger.Error("Failed to get server", zap.Error(errServer))

						continue
					}

					serverCache[serverName] = server.ServerID
					serverID = server.ServerID
				}

				createdString, createdOk := msg["created_at"].(string)
				if !createdOk {
					continue
				}

				createdPieces := strings.Split(createdString, ".")
				if len(createdPieces) != 2 {
					rootLogger.Error("Failed to split time")

					continue
				}

				createdAt, errCreatedAt := time.Parse("2006-01-02T15:04:05", createdPieces[0])
				if errCreatedAt != nil {
					rootLogger.Error("Failed to parse time", zap.Error(errCreatedAt))

					continue
				}

				name, nameOk := msg["player_name"].(string)
				if !nameOk {
					continue
				}

				message, messageOk := msg["message"].(string)
				if !messageOk {
					continue
				}

				team, teamOk := msg["team"].(bool)
				if !teamOk {
					continue
				}

				if errAdd := database.AddChatHistory(ctx, &store.PersonMessage{
					SteamID:     sid,
					PersonaName: name,
					ServerID:    serverID,
					Body:        message,
					Team:        team,
					CreatedOn:   createdAt,
				}); errAdd != nil {
					rootLogger.Error("Failed to add chat message", zap.Error(errAdd))

					continue
				}
			}
		},
	}
}
