package anticheat

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/discord"
	"io"
	"log/slog"
	"slices"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type antiCheatUsecase struct {
	parser        logparse.StacParser
	repo          domain.AntiCheatRepository
	person        domain.PersonUsecase
	ban           domain.BanSteamUsecase
	config        domain.ConfigUsecase
	notifications domain.NotificationUsecase
}

func NewAntiCheatUsecase(repo domain.AntiCheatRepository, person domain.PersonUsecase, ban domain.BanSteamUsecase, config domain.ConfigUsecase, notif domain.NotificationUsecase) domain.AntiCheatUsecase {
	return &antiCheatUsecase{
		parser:        logparse.NewStacParser(),
		repo:          repo,
		person:        person,
		ban:           ban,
		config:        config,
		notifications: notif,
	}
}

func (a antiCheatUsecase) DetectionsBySteamID(ctx context.Context, steamID steamid.SteamID) ([]logparse.StacEntry, error) {
	if !steamID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	return a.repo.DetectionsBySteamID(ctx, steamID)
}

func (a antiCheatUsecase) Handle(ctx context.Context, entries []logparse.StacEntry) error {
	results := map[steamid.SteamID]map[logparse.Detection]int{}
	conf := a.config.Config()

	owner, errOwner := a.person.GetPersonBySteamID(ctx, nil, steamid.New(conf.Owner))
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

		duration := "0"
		if conf.Anticheat.Duration > 0 {
			duration = fmt.Sprintf("%dm", conf.Anticheat.Duration)
		}

		ban, err := a.ban.Ban(ctx, owner.ToUserProfile(), domain.System, domain.RequestBanSteamCreate{
			SourceIDField:  domain.SourceIDField{SourceID: owner.SteamID.String()},
			TargetIDField:  domain.TargetIDField{TargetID: entry.SteamID.String()},
			Duration:       duration,
			ValidUntil:     time.Now().AddDate(10, 0, 0),
			BanType:        domain.Banned,
			Reason:         domain.Cheating,
			ReasonText:     "",
			Note:           entry.Summary + "\n\nRaw log:\n" + entry.RawLog,
			DemoName:       entry.DemoName,
			DemoTick:       entry.DemoTick,
			IncludeFriends: false,
			EvadeOk:        false,
		})

		if err != nil {
			slog.Error("Failed to ban cheater", slog.String("detection", string(entry.Detection)),
				slog.Int64("steam_id", entry.SteamID.Int64()), log.ErrAttr(err))
		} else if ban.BanID > 0 {
			slog.Info("Banned cheater", slog.String("detection", string(entry.Detection)),
				slog.Int64("steam_id", entry.SteamID.Int64()))
			hasBeenBanned = append(hasBeenBanned, entry.SteamID)
		}

		a.notifications.Enqueue(ctx, domain.NewDiscordNotification(domain.ChannelBanLog, discord.NewEmbed("").
			Message()))
	}

	return nil
}

func (a antiCheatUsecase) DetectionsByType(ctx context.Context, detectionType logparse.Detection) ([]logparse.StacEntry, error) {
	return a.repo.DetectionsByType(ctx, detectionType)
}

func (a antiCheatUsecase) Import(ctx context.Context, fileName string, reader io.ReadCloser, serverID int) ([]logparse.StacEntry, error) {
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
		player, err := a.person.GetOrCreatePersonBySteamID(ctx, nil, entry.SteamID)
		if err != nil {
			return nil, err
		}
		if player.PersonaName == "" && entry.Name != "" {
			player.PersonaName = entry.Name
			if errSave := a.person.SavePerson(ctx, nil, &player); errSave != nil {
				return nil, errSave
			}
		}
	}

	if err := a.repo.SaveEntries(ctx, entries); err != nil {
		return nil, err
	}

	return entries, nil
}

func (a antiCheatUsecase) SyncDemoIDs(ctx context.Context, limit uint64) error {
	if limit == 0 {
		limit = 100
	}

	return a.repo.SyncDemoIDs(ctx, limit)
}

func (a antiCheatUsecase) Query(ctx context.Context, query domain.AnticheatQuery) ([]domain.AnticheatEntry, error) {
	if query.SteamID != "" {
		sid := steamid.New(query.SteamID)
		if !sid.Valid() {
			return nil, domain.ErrInvalidSID
		}
	}

	return a.repo.Query(ctx, query)
}
