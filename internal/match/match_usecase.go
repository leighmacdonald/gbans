package match

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

// func parseMapName(name string) string {
//	if strings.HasPrefix(name, "workshop/") {
//		parts := strings.Split(strings.TrimPrefix(name, "workshop/"), ".ugc")
//		name = parts[0]
//	}
//
//	return name
// }

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

func (m matchUsecase) CreateFromDemo(_ context.Context, serverID int, details demoparse.Demo) (domain.MatchResult, error) {
	server, errServer := m.servers.Server(context.Background(), serverID)
	if errServer != nil {
		return domain.MatchResult{}, errServer
	}
	newID, errID := uuid.NewV4()
	if errID != nil {
		return domain.MatchResult{}, errors.Join(errID, domain.ErrUUIDCreate)
	}

	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(int(math.Ceil(details.Duration))))
	result := domain.MatchResult{
		MatchID:  newID,
		ServerID: server.ServerID,
		Title:    details.Server,
		MapName:  details.Map,
		TeamScores: logparse.TeamScores{
			Red:     details.State.Results.ScoreRed,
			RedTime: details.State.Results.RedTime,
			Blu:     details.State.Results.ScoreBlu,
			BluTime: details.State.Results.BluTime,
		},
		TimeStart: startTime,
		TimeEnd:   endTime,
		Winner:    logparse.BLU,
		Players:   nil,
		Chat:      nil,
	}

	for _, chat := range details.State.Chat {
		result.Chat = append(result.Chat, domain.MatchChat{
			SteamID:     steamid.New(chat.SteamID),
			PersonaName: chat.PersonaName,
			Body:        chat.Body,
			Team:        chat.Team,
		})
	}

	for uid, player := range details.State.Players {
		user := details.State.Users[uid]
		matchPlayer := domain.MatchPlayer{
			CommonPlayerStats: domain.CommonPlayerStats{
				SteamID:           steamid.New(user.SteamID),
				Name:              user.Name,
				Kills:             player.Kills,
				Assists:           player.Assists,
				Deaths:            player.Deaths,
				Suicides:          0,
				Dominations:       player.Dominations,
				Dominated:         0,
				Revenges:          player.Revenges,
				Damage:            player.DamageDealt,
				DamageTaken:       player.DamageTaken,
				HealingTaken:      player.HealingTaken,
				HealthPacks:       player.HealthPacks,
				HealingPacks:      player.HealingPacks,
				Captures:          player.Captures,
				CapturesBlocked:   player.Defenses,
				Extinguishes:      player.Extinguishes,
				BuildingBuilt:     player.BuildingBuilt,
				BuildingDestroyed: player.BuildingDestroyed,
				Backstabs:         player.Backstabs,
				Airshots:          player.Airshots,
				Headshots:         player.Headshots,
				Shots:             player.Shots,
				Hits:              player.Hits,
			},
			Team:        user.Team,
			TimeStart:   time.Time{},
			TimeEnd:     time.Time{},
			MedicStats:  nil,
			Classes:     nil,
			Killstreaks: nil,
			Weapons:     nil,
		}

		result.Players = append(result.Players, &matchPlayer)
	}

	return result, nil
}

func (m matchUsecase) GetMatchIDFromServerID(serverID int) (uuid.UUID, bool) {
	return m.repository.GetMatchIDFromServerID(serverID)
}

func (m matchUsecase) Matches(ctx context.Context, opts domain.MatchesQueryOpts) ([]domain.MatchResult, int64, error) {
	return m.repository.Matches(ctx, opts)
}

func (m matchUsecase) MatchGetByID(ctx context.Context, matchID uuid.UUID, match *domain.MatchResult) error {
	return m.repository.MatchGetByID(ctx, matchID, match)
}

// todo hide.
func (m matchUsecase) MatchSave(ctx context.Context, match *domain.MatchResult) error {
	return m.repository.MatchSave(ctx, match)
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
