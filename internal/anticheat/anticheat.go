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

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/network/scp"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/viant/afs/option"
	"github.com/viant/afs/storage"
)

type Action string

const (
	ActionGag  Action = "gag"
	ActionKick Action = "kick"
	ActionBan  Action = "ban"
)

type OnEntry func(ctx context.Context, entry logparse.StacEntry, duration time.Duration, count int) error

var ErrOpenClient = errors.New("failed to open client")

type Config struct {
	Enabled               bool   `mapstructure:"enabled" json:"enabled"`
	Action                Action `mapstructure:"action" json:"action"`
	Duration              int    `mapstructure:"duration" json:"duration"`
	MaxAimSnap            int    `mapstructure:"max_aim_snap" json:"max_aim_snap"`
	MaxPsilent            int    `mapstructure:"max_psilent" json:"max_psilent"`
	MaxBhop               int    `mapstructure:"max_bhop" json:"max_bhop"`
	MaxFakeAng            int    `mapstructure:"max_fake_ang" json:"max_fake_ang"`
	MaxCmdNum             int    `mapstructure:"max_cmd_num" json:"max_cmd_num"`
	MaxTooManyConnections int    `mapstructure:"max_too_many_connections" json:"max_too_many_connections"`
	MaxCheatCvar          int    `mapstructure:"max_cheat_cvar" json:"max_cheat_cvar"`
	MaxOOBVar             int    `mapstructure:"max_oob_var" json:"max_oob_var"`
	MaxInvalidUserCmd     int    `mapstructure:"max_invalid_user_cmd" json:"max_invalid_user_cmd"`
}

type ConfigStore struct {
	PathRoot string `json:"path_root"`
}

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
	Config

	parser  logparse.StacParser
	repo    Repository
	notif   notification.Notifier
	handler OnEntry
}

func NewAntiCheat(repo Repository, config Config, notif notification.Notifier, handler OnEntry) AntiCheat {
	return AntiCheat{
		Config:  config,
		parser:  logparse.NewStacParser(),
		repo:    repo,
		notif:   notif,
		handler: handler,
	}
}

func (a AntiCheat) DownloadHandler(ctx context.Context, client storage.Storager, server scp.ServerInfo, config scp.Config) error {
	for _, instance := range server.ServerIDs {
		logDir := server.GamePath(config.StacPathFmt, instance)
		filelist, errFilelist := client.List(ctx, logDir, option.NewPage(0, 1))
		if errFilelist != nil {
			slog.Error("remote list dir failed", slog.String("error", errFilelist.Error()),
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
				return errors.Join(err, ErrOpenClient)
			}

			slog.Debug("Importing stac log", slog.String("name", file.Name()), slog.String("server", instance.ShortName))
			entries, errImport := a.Import(ctx, file.Name(), reader, instance.ServerID)
			if errImport != nil && !errors.Is(errImport, database.ErrDuplicate) {
				slog.Error("Failed to import stac logs", slog.String("error", errImport.Error()))
			} else if len(entries) > 0 {
				if errHandle := a.Handle(ctx, entries); errHandle != nil {
					slog.Error("Failed to handle stac logs", slog.String("error", errHandle.Error()))
				}
			}

			_ = reader.Close()
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

func (a AntiCheat) Handle(ctx context.Context, entries []logparse.StacEntry) error { //nolint:cyclop
	var ( //nolint:prealloc
		results       = map[steamid.SteamID]map[logparse.Detection]int{}
		hasBeenBanned []steamid.SteamID
	)
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
			isban = a.MaxPsilent > 0 && results[entry.SteamID][entry.Detection] >= a.MaxPsilent
		case logparse.AimSnap:
			isban = a.MaxAimSnap > 0 && results[entry.SteamID][entry.Detection] >= a.MaxAimSnap
		case logparse.BHop:
			isban = a.MaxBhop > 0 && results[entry.SteamID][entry.Detection] >= a.MaxBhop
		case logparse.CmdNumSpike:
			isban = a.MaxCmdNum > 0 && results[entry.SteamID][entry.Detection] >= a.MaxCmdNum
		case logparse.EyeAngles:
			isban = a.MaxFakeAng > 0 && results[entry.SteamID][entry.Detection] >= a.MaxFakeAng
		case logparse.InvalidUserCmd:
			isban = a.MaxInvalidUserCmd > 0 && results[entry.SteamID][entry.Detection] >= a.MaxInvalidUserCmd
		case logparse.OOBCVar:
			isban = a.MaxOOBVar > 0 && results[entry.SteamID][entry.Detection] >= a.MaxOOBVar
		case logparse.CheatCVar:
			isban = a.MaxCheatCvar > 0 && results[entry.SteamID][entry.Detection] >= a.MaxCheatCvar
		case logparse.TooManyConnectiona:
			isban = a.MaxTooManyConnections > 0 && results[entry.SteamID][entry.Detection] >= a.MaxTooManyConnections
		default:
			slog.Warn("Got unknown stac detection", slog.String("summary", entry.Summary))
		}

		if !isban || slices.Contains(hasBeenBanned, entry.SteamID) {
			continue
		}

		var dur time.Duration
		if a.Duration > 0 {
			dur = time.Duration(a.Duration) * time.Second
		}

		if err := a.handler(ctx, entry, dur, results[entry.SteamID][entry.Detection]); err != nil {
			slog.Error("Failed to run antichat handler", slog.String("detection", string(entry.Detection)),
				slog.Int64("steam_id", entry.SteamID.Int64()), slog.String("error", err.Error()))

			continue
		}
		hasBeenBanned = append(hasBeenBanned, entry.SteamID)
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
