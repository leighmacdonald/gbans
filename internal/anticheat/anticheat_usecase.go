package anticheat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	banDomain "github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/viant/afs/option"
	"github.com/viant/afs/storage"
)

type AntiCheatUsecase struct {
	parser  logparse.StacParser
	repo    anticheatRepository
	persons person.PersonUsecase
	ban     ban.BanUsecase
	config  *config.ConfigUsecase
}

func NewAntiCheatUsecase(repo anticheatRepository, ban ban.BanUsecase, config *config.ConfigUsecase, persons person.PersonUsecase) AntiCheatUsecase {
	return AntiCheatUsecase{
		parser:  logparse.NewStacParser(),
		repo:    repo,
		ban:     ban,
		config:  config,
		persons: persons,
	}
}

// TODO fix
func (a AntiCheatUsecase) FetchStacLogs(ctx context.Context, stactPathFmt string, server servers.Server, client storage.Storager) error {
	logDir := fmt.Sprintf(stactPathFmt, server.ShortName)

	filelist, errFilelist := client.List(ctx, logDir, option.NewPage(0, 1))
	if errFilelist != nil {
		slog.Error("remote list dir failed", log.ErrAttr(errFilelist),
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
			return err
		}

		slog.Debug("Importing stac log", slog.String("name", file.Name()), slog.String("server", server.ShortName))
		entries, errImport := a.Import(ctx, file.Name(), reader, server.ServerID)
		if errImport != nil && !errors.Is(errImport, database.ErrDuplicate) {
			slog.Error("Failed to import stac logs", log.ErrAttr(errImport))
		} else if len(entries) > 0 {
			if errHandle := a.Handle(ctx, entries); errHandle != nil {
				slog.Error("Failed to handle stac logs", log.ErrAttr(errHandle))
			}
		}

		if errCloseReader := reader.Close(); errCloseReader != nil {
			return errCloseReader
		}
	}

	return nil
}

func (a AntiCheatUsecase) DetectionsBySteamID(ctx context.Context, steamID steamid.SteamID) ([]logparse.StacEntry, error) {
	if !steamID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	return a.repo.DetectionsBySteamID(ctx, steamID)
}

func (a AntiCheatUsecase) Handle(ctx context.Context, entries []logparse.StacEntry) error {
	results := map[steamid.SteamID]map[logparse.Detection]int{}
	conf := a.config.Config()

	owner, errOwner := a.persons.GetOrCreatePersonBySteamID(ctx, nil, steamid.New(conf.Owner))
	if errOwner != nil {
		return errOwner
	}

	var hasBeenBanned []steamid.SteamID

	for _, entry := range entries {
		if _, ok := results[entry.SteamID]; !ok {
			results[entry.SteamID] = map[logparse.Detection]int{}
		}

		if _, ok := results[entry.SteamID][entry.Detection]; !ok {
			results[entry.SteamID][entry.Detection] = 0
		}

		results[entry.SteamID][entry.Detection]++

		isban := false

		switch entry.Detection {
		case logparse.SilentAim:
			isban = conf.Anticheat.MaxPsilent > 0 && results[entry.SteamID][entry.Detection] >= conf.Anticheat.MaxPsilent
		case logparse.AimSnap:
			isban = conf.Anticheat.MaxAimSnap > 0 && results[entry.SteamID][entry.Detection] >= conf.Anticheat.MaxAimSnap
		case logparse.BHop:
			isban = conf.Anticheat.MaxBhop > 0 && results[entry.SteamID][entry.Detection] >= conf.Anticheat.MaxBhop
		case logparse.CmdNumSpike:
			isban = conf.Anticheat.MaxCmdNum > 0 && results[entry.SteamID][entry.Detection] >= conf.Anticheat.MaxCmdNum
		case logparse.EyeAngles:
			isban = conf.Anticheat.MaxFakeAng > 0 && results[entry.SteamID][entry.Detection] >= conf.Anticheat.MaxFakeAng
		case logparse.InvalidUserCmd:
			isban = conf.Anticheat.MaxInvalidUserCmd > 0 && results[entry.SteamID][entry.Detection] >= conf.Anticheat.MaxInvalidUserCmd
		case logparse.OOBCVar:
			isban = conf.Anticheat.MaxOOBVar > 0 && results[entry.SteamID][entry.Detection] >= conf.Anticheat.MaxOOBVar
		case logparse.CheatCVar:
			isban = conf.Anticheat.MaxCheatCvar > 0 && results[entry.SteamID][entry.Detection] >= conf.Anticheat.MaxCheatCvar
		case logparse.TooManyConnectiona:
			isban = conf.Anticheat.MaxTooManyConnections > 0 && results[entry.SteamID][entry.Detection] >= conf.Anticheat.MaxTooManyConnections
		default:
			slog.Warn("Got unknown stac detection", slog.String("summary", entry.Summary))
		}

		if !isban || slices.Contains(hasBeenBanned, entry.SteamID) {
			continue
		}

		var duration time.Duration
		if conf.Anticheat.Duration > 0 {
			duration = time.Duration(conf.Anticheat.Duration) * time.Second
		}

		newBan, err := a.ban.Ban(ctx, ban.BanOpts{
			Origin:         banDomain.System,
			SourceID:       owner.SteamID,
			TargetID:       entry.SteamID,
			Duration:       duration,
			BanType:        banDomain.Banned,
			Reason:         banDomain.Cheating,
			ReasonText:     "",
			Note:           entry.Summary + "\n\nRaw log:\n" + entry.RawLog,
			DemoName:       entry.DemoName,
			DemoTick:       entry.DemoTick,
			IncludeFriends: false,
			EvadeOk:        false,
		})
		if err != nil && !errors.Is(err, database.ErrDuplicate) {
			slog.Error("Failed to ban cheater", slog.String("detection", string(entry.Detection)),
				slog.Int64("steam_id", entry.SteamID.Int64()), log.ErrAttr(err))
		} else if newBan.BanID > 0 {
			slog.Info("Banned cheater", slog.String("detection", string(entry.Detection)),
				slog.Int64("steam_id", entry.SteamID.Int64()))
			hasBeenBanned = append(hasBeenBanned, entry.SteamID)

			// go a.notifications.Enqueue(ctx, notification.NewDiscordNotification(a.config.Config().Discord.AnticheatChannelID,
			// 	discord.NewAnticheatTrigger(newBan, conf, entry, results[entry.SteamID][entry.Detection])))
		}
	}

	return nil
}

func (a AntiCheatUsecase) DetectionsByType(ctx context.Context, detectionType logparse.Detection) ([]logparse.StacEntry, error) {
	return a.repo.DetectionsByType(ctx, detectionType)
}

func (a AntiCheatUsecase) Import(ctx context.Context, fileName string, reader io.ReadCloser, serverID int) ([]logparse.StacEntry, error) {
	entries, errEntries := a.parser.Parse(fileName, reader)
	if errEntries != nil {
		return nil, errEntries
	}

	if len(entries) == 0 {
		return nil, nil
	}

	for i := range entries {
		entries[i].ServerID = serverID
	}

	for _, entry := range entries {
		_, err := a.persons.GetOrCreatePersonBySteamID(ctx, nil, entry.SteamID)
		if err != nil {
			return nil, err
		}
	}

	if err := a.repo.SaveEntries(ctx, entries); err != nil {
		return nil, err
	}

	return entries, nil
}

func (a AntiCheatUsecase) SyncDemoIDs(ctx context.Context, limit uint64) error {
	if limit == 0 {
		limit = 100
	}

	return a.repo.SyncDemoIDs(ctx, limit)
}

func (a AntiCheatUsecase) Query(ctx context.Context, query AnticheatQuery) ([]AnticheatEntry, error) {
	if query.SteamID != "" {
		sid := steamid.New(query.SteamID)
		if !sid.Valid() {
			return nil, domain.ErrInvalidSID
		}
	}

	return a.repo.Query(ctx, query)
}
