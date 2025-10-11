package anticheat

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	banDomain "github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/internal/domain/network"
	"github.com/leighmacdonald/gbans/internal/network/scp"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/sosodev/duration"
	"github.com/viant/afs/option"
	"github.com/viant/afs/storage"
)

// Entry represents a stac log entry and some associated meta data.
type Entry struct {
	logparse.StacEntry
	Personaname string `json:"personaname"`
	AvatarHash  string `json:"avatar_hash"`
	Triggered   int    `json:"triggered"`
}

type Query struct {
	query.Filter
	Name      string             `json:"name" schema:"name"`
	SteamID   string             `json:"steam_id" schema:"steam_id"`
	ServerID  int                `json:"server_id" schema:"server_id"`
	Summary   string             `json:"summary" schema:"summary"`
	Detection logparse.Detection `json:"detection" schema:"detection"`
}

// AntiCheat handles parsing and processing of stac anti-cheat logs.
type AntiCheat struct {
	parser  logparse.StacParser
	repo    Repository
	persons *person.Persons
	ban     ban.Bans
	config  *config.Configuration
	notif   notification.Notifier
}

func NewAntiCheat(repo Repository, ban ban.Bans, config *config.Configuration, persons *person.Persons, notif notification.Notifier) AntiCheat {
	return AntiCheat{
		parser:  logparse.NewStacParser(),
		repo:    repo,
		ban:     ban,
		config:  config,
		persons: persons,
		notif:   notif,
	}
}

func (a AntiCheat) DownloadHandler(ctx context.Context, client storage.Storager, server scp.ServerInfo) error {
	for _, instance := range server.ServerIDs {
		logDir := server.GamePath(instance, "tf/addons/sourcemod/logs/stac")

		filelist, errFilelist := client.List(ctx, logDir, option.NewPage(0, 1))
		if errFilelist != nil {
			slog.Error("remote list dir failed", log.ErrAttr(errFilelist),
				slog.String("server", instance.ShortName), slog.String("path", logDir))

			return nil //nolint:nilerr
		}

		for _, file := range filelist {
			if !strings.HasSuffix(file.Name(), ".log") {
				continue
			}

			logPath := path.Join(logDir, file.Name())
			reader, err := client.Open(ctx, logPath)
			if err != nil {
				return errors.Join(err, network.ErrOpenClient)
			}

			slog.Debug("Importing stac log", slog.String("name", file.Name()), slog.String("server", instance.ShortName))
			entries, errImport := a.Import(ctx, file.Name(), reader, instance.ServerID)
			if errImport != nil && !errors.Is(errImport, database.ErrDuplicate) {
				slog.Error("Failed to import stac logs", log.ErrAttr(errImport))
			} else if len(entries) > 0 {
				if errHandle := a.Handle(ctx, entries); errHandle != nil {
					slog.Error("Failed to handle stac logs", log.ErrAttr(errHandle))
				}
			}

			if errClose := reader.Close(); errClose != nil {
				return errors.Join(errClose, network.ErrCloseReader)
			}
		}
	}

	return nil
}

// BySteamID returns all stac entries for the user.
func (a AntiCheat) BySteamID(ctx context.Context, steamID steamid.SteamID) ([]logparse.StacEntry, error) {
	if !steamID.Valid() {
		return nil, steamid.ErrInvalidSID
	}

	return a.repo.DetectionsBySteamID(ctx, steamID)
}

func (a AntiCheat) Handle(ctx context.Context, entries []logparse.StacEntry) error {
	results := map[steamid.SteamID]map[logparse.Detection]int{}
	conf := a.config.Config()

	owner, errOwner := a.persons.GetOrCreatePersonBySteamID(ctx, steamid.New(conf.Owner))
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

		var dur time.Duration
		if conf.Anticheat.Duration > 0 {
			dur = time.Duration(conf.Anticheat.Duration) * time.Second
		}

		newBan, err := a.ban.Create(ctx, ban.Opts{
			Origin:     banDomain.System,
			SourceID:   owner.GetSteamID(),
			TargetID:   entry.SteamID,
			Duration:   duration.FromTimeDuration(dur),
			BanType:    banDomain.Banned,
			Reason:     banDomain.Cheating,
			ReasonText: "",
			Note:       entry.Summary + "\n\nRaw log:\n" + entry.RawLog,
			DemoName:   entry.DemoName,
			DemoTick:   entry.DemoTick,
			EvadeOk:    false,
		})
		if err != nil && !errors.Is(err, database.ErrDuplicate) {
			slog.Error("Failed to ban cheater", slog.String("detection", string(entry.Detection)),
				slog.Int64("steam_id", entry.SteamID.Int64()), log.ErrAttr(err))
		} else if newBan.BanID > 0 {
			slog.Info("Banned cheater", slog.String("detection", string(entry.Detection)),
				slog.Int64("steam_id", entry.SteamID.Int64()))
			hasBeenBanned = append(hasBeenBanned, entry.SteamID)
			a.notif.Send(notification.NewDiscord(a.config.Config().Discord.AnticheatChannelID,
				NewAnticheatTrigger(newBan, a.config.Config().Anticheat.Action, entry, results[entry.SteamID][entry.Detection])))
		}
	}

	return nil
}

func (a AntiCheat) DetectionsByType(ctx context.Context, detectionType logparse.Detection) ([]logparse.StacEntry, error) {
	return a.repo.DetectionsByType(ctx, detectionType)
}

func (a AntiCheat) Import(ctx context.Context, fileName string, reader io.ReadCloser, serverID int) ([]logparse.StacEntry, error) {
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
		_, err := a.persons.GetOrCreatePersonBySteamID(ctx, entry.SteamID)
		if err != nil {
			return nil, err
		}
	}

	if err := a.repo.SaveEntries(ctx, entries); err != nil {
		return nil, err
	}

	return entries, nil
}

func (a AntiCheat) SyncDemoIDs(ctx context.Context, limit uint64) error {
	if limit == 0 {
		limit = 100
	}

	return a.repo.SyncDemoIDs(ctx, limit)
}

func (a AntiCheat) Query(ctx context.Context, query Query) ([]Entry, error) {
	if query.SteamID != "" {
		sid := steamid.New(query.SteamID)
		if !sid.Valid() {
			return nil, steamid.ErrInvalidSID
		}
	}

	return a.repo.Query(ctx, query)
}
