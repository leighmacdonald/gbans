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
	mr domain.MatchRepository
	su domain.StateUsecase
	sv domain.ServersUsecase
	du domain.DiscordUsecase
}

func NewMatchUsecase(matchRepository domain.MatchRepository, stateUsecase domain.StateUsecase, serversUsecase domain.ServersUsecase,
	discordUsecase domain.DiscordUsecase,
) domain.MatchUsecase {
	return &matchUsecase{
		mr: matchRepository,
		su: stateUsecase,
		sv: serversUsecase,
		du: discordUsecase,
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

	m.mr.StartMatch(trigger)

	return matchUUID, nil
}

func (m matchUsecase) EndMatch(ctx context.Context, serverID int) (uuid.UUID, error) {
	matchID, found := m.mr.GetMatchIDFromServerID(serverID)
	if !found {
		return matchID, domain.ErrLoadMatch
	}

	server, errServer := m.sv.GetServer(ctx, serverID)
	if errServer != nil {
		return matchID, errors.Join(errServer, domain.ErrUnknownServer)
	}

	m.mr.EndMatch(domain.MatchTrigger{
		Type:   domain.MatchTriggerEnd,
		UUID:   matchID,
		Server: server,
	})

	return matchID, nil
}

func (m matchUsecase) GetMatchIDFromServerID(serverID int) (uuid.UUID, bool) {
	return m.mr.GetMatchIDFromServerID(serverID)
}

func (m matchUsecase) Matches(ctx context.Context, opts domain.MatchesQueryOpts) ([]domain.MatchSummary, int64, error) {
	return m.mr.Matches(ctx, opts)
}

func (m matchUsecase) MatchGetByID(ctx context.Context, matchID uuid.UUID, match *domain.MatchResult) error {
	return m.mr.MatchGetByID(ctx, matchID, match)
}

// todo hide.
func (m matchUsecase) MatchSave(ctx context.Context, match *logparse.Match, weaponMap fp.MutexMap[logparse.Weapon, int]) error {
	return m.mr.MatchSave(ctx, match, weaponMap)
}

func (m matchUsecase) StatsPlayerClass(ctx context.Context, sid64 steamid.SteamID) (domain.PlayerClassStatsCollection, error) {
	return m.mr.StatsPlayerClass(ctx, sid64)
}

func (m matchUsecase) StatsPlayerWeapons(ctx context.Context, sid64 steamid.SteamID) ([]domain.PlayerWeaponStats, error) {
	return m.mr.StatsPlayerWeapons(ctx, sid64)
}

func (m matchUsecase) StatsPlayerKillstreaks(ctx context.Context, sid64 steamid.SteamID) ([]domain.PlayerKillstreakStats, error) {
	return m.mr.StatsPlayerKillstreaks(ctx, sid64)
}

func (m matchUsecase) StatsPlayerMedic(ctx context.Context, sid64 steamid.SteamID) ([]domain.PlayerMedicStats, error) {
	return m.mr.StatsPlayerMedic(ctx, sid64)
}

func (m matchUsecase) PlayerStats(ctx context.Context, steamID steamid.SteamID, stats *domain.PlayerStats) error {
	return m.mr.PlayerStats(ctx, steamID, stats)
}

func (m matchUsecase) WeaponsOverall(ctx context.Context) ([]domain.WeaponsOverallResult, error) {
	return m.mr.WeaponsOverall(ctx)
}

func (m matchUsecase) GetMapUsageStats(ctx context.Context) ([]domain.MapUseDetail, error) {
	return m.mr.GetMapUsageStats(ctx)
}

func (m matchUsecase) Weapons(ctx context.Context) ([]domain.Weapon, error) {
	return m.mr.Weapons(ctx)
}

func (m matchUsecase) SaveWeapon(ctx context.Context, weapon *domain.Weapon) error {
	return m.mr.SaveWeapon(ctx, weapon)
}

func (m matchUsecase) GetWeaponByKey(ctx context.Context, key logparse.Weapon, weapon *domain.Weapon) error {
	return m.mr.GetWeaponByKey(ctx, key, weapon)
}

func (m matchUsecase) GetWeaponByID(ctx context.Context, weaponID int, weapon *domain.Weapon) error {
	return m.mr.GetWeaponByID(ctx, weaponID, weapon)
}

func (m matchUsecase) LoadWeapons(ctx context.Context, weaponMap fp.MutexMap[logparse.Weapon, int]) error {
	return m.mr.LoadWeapons(ctx, weaponMap)
}

func (m matchUsecase) WeaponsOverallTopPlayers(ctx context.Context, weaponID int) ([]domain.PlayerWeaponResult, error) {
	return m.mr.WeaponsOverallTopPlayers(ctx, weaponID)
}

func (m matchUsecase) WeaponsOverallByPlayer(ctx context.Context, steamID steamid.SteamID) ([]domain.WeaponsOverallResult, error) {
	return m.mr.WeaponsOverallByPlayer(ctx, steamID)
}

func (m matchUsecase) PlayersOverallByKills(ctx context.Context, count int) ([]domain.PlayerWeaponResult, error) {
	return m.mr.PlayersOverallByKills(ctx, count)
}

func (m matchUsecase) HealersOverallByHealing(ctx context.Context, count int) ([]domain.HealingOverallResult, error) {
	return m.mr.HealersOverallByHealing(ctx, count)
}

func (m matchUsecase) PlayerOverallClassStats(ctx context.Context, steamID steamid.SteamID) ([]domain.PlayerClassOverallResult, error) {
	return m.mr.PlayerOverallClassStats(ctx, steamID)
}

func (m matchUsecase) PlayerOverallStats(ctx context.Context, steamID steamid.SteamID, por *domain.PlayerOverallResult) error {
	return m.mr.PlayerOverallStats(ctx, steamID, por)
}
