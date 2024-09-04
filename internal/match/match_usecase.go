package match

import (
	"context"
	"errors"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type matchUsecase struct {
	repository    domain.MatchRepository
	state         domain.StateUsecase
	servers       domain.ServersUsecase
	notifications domain.NotificationUsecase
}

func NewMatchUsecase(repository domain.MatchRepository, state domain.StateUsecase, servers domain.ServersUsecase,
	notifications domain.NotificationUsecase,
) domain.MatchUsecase {
	return &matchUsecase{
		repository:    repository,
		state:         state,
		servers:       servers,
		notifications: notifications,
	}
}

func (m matchUsecase) StartMatch(server domain.Server, mapName string, demoName string) (uuid.UUID, error) {
	matchUUID, errUUID := uuid.NewV4()
	if errUUID != nil {
		return uuid.UUID{}, errors.Join(errUUID, domain.ErrUUIDCreate)
	}

	trigger := domain.MatchTrigger{
		Type:     domain.MatchTriggerStart,
		UUID:     matchUUID,
		Server:   server,
		MapName:  mapName,
		DemoName: demoName,
	}

	m.repository.StartMatch(trigger)

	return matchUUID, nil
}

func (m matchUsecase) EndMatch(ctx context.Context, serverID int) (uuid.UUID, error) {
	matchID, found := m.repository.GetMatchIDFromServerID(serverID)
	if !found {
		return matchID, domain.ErrLoadMatch
	}

	server, errServer := m.servers.Server(ctx, serverID)
	if errServer != nil {
		return matchID, errors.Join(errServer, domain.ErrUnknownServer)
	}

	m.repository.EndMatch(domain.MatchTrigger{
		Type:   domain.MatchTriggerEnd,
		UUID:   matchID,
		Server: server,
	})

	return matchID, nil
}

func (m matchUsecase) GetMatchIDFromServerID(serverID int) (uuid.UUID, bool) {
	return m.repository.GetMatchIDFromServerID(serverID)
}

func (m matchUsecase) Matches(ctx context.Context, opts domain.MatchesQueryOpts) ([]domain.MatchSummary, int64, error) {
	return m.repository.Matches(ctx, opts)
}

func (m matchUsecase) MatchGetByID(ctx context.Context, matchID uuid.UUID, match *domain.MatchResult) error {
	return m.repository.MatchGetByID(ctx, matchID, match)
}

// todo hide.
func (m matchUsecase) MatchSave(ctx context.Context, match *logparse.Match, weaponMap fp.MutexMap[logparse.Weapon, int]) error {
	return m.repository.MatchSave(ctx, match, weaponMap)
}

func (m matchUsecase) StatsPlayerClass(ctx context.Context, sid64 steamid.SteamID) (domain.PlayerClassStatsCollection, error) {
	return m.repository.StatsPlayerClass(ctx, sid64)
}

func (m matchUsecase) StatsPlayerWeapons(ctx context.Context, sid64 steamid.SteamID) ([]domain.PlayerWeaponStats, error) {
	return m.repository.StatsPlayerWeapons(ctx, sid64)
}

func (m matchUsecase) StatsPlayerKillstreaks(ctx context.Context, sid64 steamid.SteamID) ([]domain.PlayerKillstreakStats, error) {
	return m.repository.StatsPlayerKillstreaks(ctx, sid64)
}

func (m matchUsecase) StatsPlayerMedic(ctx context.Context, sid64 steamid.SteamID) ([]domain.PlayerMedicStats, error) {
	return m.repository.StatsPlayerMedic(ctx, sid64)
}

func (m matchUsecase) PlayerStats(ctx context.Context, steamID steamid.SteamID, stats *domain.PlayerStats) error {
	return m.repository.PlayerStats(ctx, steamID, stats)
}

func (m matchUsecase) WeaponsOverall(ctx context.Context) ([]domain.WeaponsOverallResult, error) {
	return m.repository.WeaponsOverall(ctx)
}

func (m matchUsecase) GetMapUsageStats(ctx context.Context) ([]domain.MapUseDetail, error) {
	return m.repository.GetMapUsageStats(ctx)
}

func (m matchUsecase) Weapons(ctx context.Context) ([]domain.Weapon, error) {
	return m.repository.Weapons(ctx)
}

func (m matchUsecase) SaveWeapon(ctx context.Context, weapon *domain.Weapon) error {
	return m.repository.SaveWeapon(ctx, weapon)
}

func (m matchUsecase) GetWeaponByKey(ctx context.Context, key logparse.Weapon, weapon *domain.Weapon) error {
	return m.repository.GetWeaponByKey(ctx, key, weapon)
}

func (m matchUsecase) GetWeaponByID(ctx context.Context, weaponID int, weapon *domain.Weapon) error {
	return m.repository.GetWeaponByID(ctx, weaponID, weapon)
}

func (m matchUsecase) LoadWeapons(ctx context.Context, weaponMap fp.MutexMap[logparse.Weapon, int]) error {
	return m.repository.LoadWeapons(ctx, weaponMap)
}

func (m matchUsecase) WeaponsOverallTopPlayers(ctx context.Context, weaponID int) ([]domain.PlayerWeaponResult, error) {
	return m.repository.WeaponsOverallTopPlayers(ctx, weaponID)
}

func (m matchUsecase) WeaponsOverallByPlayer(ctx context.Context, steamID steamid.SteamID) ([]domain.WeaponsOverallResult, error) {
	return m.repository.WeaponsOverallByPlayer(ctx, steamID)
}

func (m matchUsecase) PlayersOverallByKills(ctx context.Context, count int) ([]domain.PlayerWeaponResult, error) {
	return m.repository.PlayersOverallByKills(ctx, count)
}

func (m matchUsecase) HealersOverallByHealing(ctx context.Context, count int) ([]domain.HealingOverallResult, error) {
	return m.repository.HealersOverallByHealing(ctx, count)
}

func (m matchUsecase) PlayerOverallClassStats(ctx context.Context, steamID steamid.SteamID) ([]domain.PlayerClassOverallResult, error) {
	return m.repository.PlayerOverallClassStats(ctx, steamID)
}

func (m matchUsecase) PlayerOverallStats(ctx context.Context, steamID steamid.SteamID, por *domain.PlayerOverallResult) error {
	return m.repository.PlayerOverallStats(ctx, steamID, por)
}
