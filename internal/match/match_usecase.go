package match

import (
	"context"
	"errors"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type MatchUsecase struct {
	repository    MatchRepository
	state         state.StateUsecase
	servers       servers.ServersUsecase
	notifications notification.NotificationUsecase
}

func NewMatchUsecase(repository MatchRepository, state state.StateUsecase, servers servers.ServersUsecase,
	notifications notification.NotificationUsecase,
) MatchUsecase {
	return MatchUsecase{
		repository:    repository,
		state:         state,
		servers:       servers,
		notifications: notifications,
	}
}

func (m MatchUsecase) StartMatch(server servers.Server, mapName string, demoName string) (uuid.UUID, error) {
	matchUUID, errUUID := uuid.NewV4()
	if errUUID != nil {
		return uuid.UUID{}, errors.Join(errUUID, domain.ErrUUIDCreate)
	}

	trigger := MatchTrigger{
		Type:     MatchTriggerStart,
		UUID:     matchUUID,
		Server:   server,
		MapName:  mapName,
		DemoName: demoName,
	}

	m.repository.StartMatch(trigger)

	return matchUUID, nil
}

func (m MatchUsecase) EndMatch(ctx context.Context, serverID int) (uuid.UUID, error) {
	matchID, found := m.repository.GetMatchIDFromServerID(serverID)
	if !found {
		return matchID, domain.ErrLoadMatch
	}

	server, errServer := m.servers.Server(ctx, serverID)
	if errServer != nil {
		return matchID, errors.Join(errServer, servers.ErrUnknownServer)
	}

	m.repository.EndMatch(MatchTrigger{
		Type:   MatchTriggerEnd,
		UUID:   matchID,
		Server: server,
	})

	return matchID, nil
}

func (m MatchUsecase) GetMatchIDFromServerID(serverID int) (uuid.UUID, bool) {
	return m.repository.GetMatchIDFromServerID(serverID)
}

func (m MatchUsecase) Matches(ctx context.Context, opts MatchesQueryOpts) ([]MatchSummary, int64, error) {
	return m.repository.Matches(ctx, opts)
}

func (m MatchUsecase) MatchGetByID(ctx context.Context, matchID uuid.UUID, match *MatchResult) error {
	return m.repository.MatchGetByID(ctx, matchID, match)
}

// todo hide.
func (m MatchUsecase) MatchSave(ctx context.Context, match *logparse.Match, weaponMap fp.MutexMap[logparse.Weapon, int]) error {
	return m.repository.MatchSave(ctx, match, weaponMap)
}

func (m MatchUsecase) StatsPlayerClass(ctx context.Context, sid64 steamid.SteamID) (PlayerClassStatsCollection, error) {
	return m.repository.StatsPlayerClass(ctx, sid64)
}

func (m MatchUsecase) StatsPlayerWeapons(ctx context.Context, sid64 steamid.SteamID) ([]PlayerWeaponStats, error) {
	return m.repository.StatsPlayerWeapons(ctx, sid64)
}

func (m MatchUsecase) StatsPlayerKillstreaks(ctx context.Context, sid64 steamid.SteamID) ([]PlayerKillstreakStats, error) {
	return m.repository.StatsPlayerKillstreaks(ctx, sid64)
}

func (m MatchUsecase) StatsPlayerMedic(ctx context.Context, sid64 steamid.SteamID) ([]PlayerMedicStats, error) {
	return m.repository.StatsPlayerMedic(ctx, sid64)
}

func (m MatchUsecase) PlayerStats(ctx context.Context, steamID steamid.SteamID, stats *PlayerStats) error {
	return m.repository.PlayerStats(ctx, steamID, stats)
}

func (m MatchUsecase) WeaponsOverall(ctx context.Context) ([]WeaponsOverallResult, error) {
	return m.repository.WeaponsOverall(ctx)
}

func (m MatchUsecase) GetMapUsageStats(ctx context.Context) ([]MapUseDetail, error) {
	return m.repository.GetMapUsageStats(ctx)
}

func (m MatchUsecase) Weapons(ctx context.Context) ([]Weapon, error) {
	return m.repository.Weapons(ctx)
}

func (m MatchUsecase) SaveWeapon(ctx context.Context, weapon *Weapon) error {
	return m.repository.SaveWeapon(ctx, weapon)
}

func (m MatchUsecase) GetWeaponByKey(ctx context.Context, key logparse.Weapon, weapon *Weapon) error {
	return m.repository.GetWeaponByKey(ctx, key, weapon)
}

func (m MatchUsecase) GetWeaponByID(ctx context.Context, weaponID int, weapon *Weapon) error {
	return m.repository.GetWeaponByID(ctx, weaponID, weapon)
}

func (m MatchUsecase) LoadWeapons(ctx context.Context, weaponMap fp.MutexMap[logparse.Weapon, int]) error {
	return m.repository.LoadWeapons(ctx, weaponMap)
}

func (m MatchUsecase) WeaponsOverallTopPlayers(ctx context.Context, weaponID int) ([]PlayerWeaponResult, error) {
	return m.repository.WeaponsOverallTopPlayers(ctx, weaponID)
}

func (m MatchUsecase) WeaponsOverallByPlayer(ctx context.Context, steamID steamid.SteamID) ([]WeaponsOverallResult, error) {
	return m.repository.WeaponsOverallByPlayer(ctx, steamID)
}

func (m MatchUsecase) PlayersOverallByKills(ctx context.Context, count int) ([]PlayerWeaponResult, error) {
	return m.repository.PlayersOverallByKills(ctx, count)
}

func (m MatchUsecase) HealersOverallByHealing(ctx context.Context, count int) ([]HealingOverallResult, error) {
	return m.repository.HealersOverallByHealing(ctx, count)
}

func (m MatchUsecase) PlayerOverallClassStats(ctx context.Context, steamID steamid.SteamID) ([]PlayerClassOverallResult, error) {
	return m.repository.PlayerOverallClassStats(ctx, steamID)
}

func (m MatchUsecase) PlayerOverallStats(ctx context.Context, steamID steamid.SteamID, por *PlayerOverallResult) error {
	return m.repository.PlayerOverallStats(ctx, steamID, por)
}
