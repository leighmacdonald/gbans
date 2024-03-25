package match

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type matchUsecase struct {
	mr           domain.MatchRepository
	su           domain.StateUsecase
	sv           domain.ServersUsecase
	events       chan logparse.ServerEvent
	du           domain.DiscordUsecase
	wm           fp.MutexMap[logparse.Weapon, int]
	broadcaster  *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]
	matchUUIDMap fp.MutexMap[int, uuid.UUID]
}

func NewMatchUsecase(broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent],
	matchRepository domain.MatchRepository, stateUsecase domain.StateUsecase, serversUsecase domain.ServersUsecase,
	discordUsecase domain.DiscordUsecase, weaponMap fp.MutexMap[logparse.Weapon, int],
) domain.MatchUsecase {
	return &matchUsecase{
		mr:           matchRepository,
		su:           stateUsecase,
		sv:           serversUsecase,
		du:           discordUsecase,
		wm:           weaponMap,
		events:       make(chan logparse.ServerEvent),
		broadcaster:  broadcaster,
		matchUUIDMap: fp.NewMutexMap[int, uuid.UUID](),
	}
}

func (m matchUsecase) GetMatchIDFromServerID(serverID int) (uuid.UUID, bool) {
	return m.matchUUIDMap.Get(serverID)
}

func (m matchUsecase) Start(ctx context.Context) {
	eventChan := make(chan logparse.ServerEvent)
	if errReg := m.broadcaster.Consume(eventChan); errReg != nil {
		slog.Error("logWriter Tried to register duplicate reader channel", log.ErrAttr(errReg))
	}

	matches := map[int]*Context{}

	for {
		select {
		case evt := <-eventChan:
			matchContext, exists := matches[evt.ServerID]

			if !exists {
				cancelCtx, cancel := context.WithCancel(ctx)
				matchContext = &Context{
					Match:          logparse.NewMatch(evt.ServerID, evt.ServerName),
					cancel:         cancel,
					incomingEvents: make(chan logparse.ServerEvent),
					stopChan:       make(chan bool),
				}

				go matchContext.start(cancelCtx)

				m.matchUUIDMap.Set(evt.ServerID, matchContext.Match.MatchID)

				matches[evt.ServerID] = matchContext
			}

			matchContext.incomingEvents <- evt

			switch evt.EventType {
			case logparse.WTeamFinalScore:
				matchContext.finalScores++
				if matchContext.finalScores < 2 {
					continue
				}

				fallthrough
			case logparse.LogStop:
				matchContext.stopChan <- true

				if err := m.onMatchComplete(ctx, matchContext); err != nil {
					switch {
					case errors.Is(err, domain.ErrInsufficientPlayers):
						slog.Warn("Insufficient data to save")
					case errors.Is(err, domain.ErrIncompleteMatch):
						slog.Warn("Incomplete match, ignoring")
					default:
						slog.Error("Failed to save Match results", log.ErrAttr(err))
					}
				}

				delete(matches, evt.ServerID)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m matchUsecase) onMatchComplete(ctx context.Context, matchContext *Context) error {
	const minPlayers = 6

	server, found := m.su.ByServerID(matchContext.Match.ServerID)

	if found && server.Name != "" {
		matchContext.Match.Title = server.Name
	}

	fullServer, err := m.sv.GetServer(ctx, server.ServerID)
	if err != nil {
		return errors.Join(err, domain.ErrLoadServer)
	}

	if !fullServer.EnableStats {
		return nil
	}

	if len(matchContext.Match.PlayerSums) < minPlayers {
		return domain.ErrInsufficientPlayers
	}

	if matchContext.Match.TimeStart == nil || matchContext.Match.MapName == "" {
		return domain.ErrIncompleteMatch
	}

	if errSave := m.MatchSave(ctx, &matchContext.Match, m.wm); errSave != nil {
		if errors.Is(errSave, domain.ErrInsufficientPlayers) {
			return domain.ErrInsufficientPlayers
		} else {
			return errors.Join(errSave, domain.ErrSaveMatch)
		}
	}

	var result domain.MatchResult
	if errResult := m.MatchGetByID(ctx, matchContext.Match.MatchID, &result); errResult != nil {
		return errors.Join(errResult, domain.ErrLoadMatch)
	}

	go m.du.SendPayload(domain.ChannelPublicMatchLog, discord.MatchMessage(result, ""))

	return nil
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
