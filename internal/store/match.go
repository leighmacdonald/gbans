package store

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type MatchesQueryOpts struct {
	QueryFilter
	SteamID   steamid.SID64 `json:"steam_id"`
	ServerID  int           `json:"server_id"`
	Map       string        `json:"map"`
	TimeStart *time.Time    `json:"time_start,omitempty"`
	TimeEnd   *time.Time    `json:"time_end,omitempty"`
}

type MatchPlayerKillstreak struct {
	MatchKillstreakID int64                `json:"match_killstreak_id"`
	MatchPlayerID     int64                `json:"match_player_id"`
	PlayerClass       logparse.PlayerClass `json:"player_class"`
	Killstreak        int                  `json:"killstreak"`
	// Seconds
	Duration int `json:"duration"`
}

type MatchPlayerClass struct {
	MatchPlayerClassID int                  `json:"match_player_class_id"`
	MatchPlayerID      int64                `json:"match_player_id"`
	PlayerClass        logparse.PlayerClass `json:"player_class"`
	Kills              int                  `json:"kills"`
	Assists            int                  `json:"assists"`
	Deaths             int                  `json:"deaths"`
	Playtime           int                  `json:"playtime"`
	Dominations        int                  `json:"dominations"`
	Dominated          int                  `json:"dominated"`
	Revenges           int                  `json:"revenges"`
	Damage             int                  `json:"damage"`
	DamageTaken        int                  `json:"damage_taken"`
	HealingTaken       int                  `json:"healing_taken"`
	Captures           int                  `json:"captures"`
	CapturesBlocked    int                  `json:"captures_blocked"`
	BuildingDestroyed  int                  `json:"building_destroyed"`
}

type MatchPlayer struct {
	MatchPlayerID     int64                   `json:"match_player_id"`
	SteamID           steamid.SID64           `json:"steam_id"`
	Team              logparse.Team           `json:"team"`
	Name              string                  `json:"name"`
	AvatarHash        string                  `json:"avatar_hash"`
	TimeStart         time.Time               `json:"time_start"`
	TimeEnd           time.Time               `json:"time_end"`
	Kills             int                     `json:"kills"`
	Assists           int                     `json:"assists"`
	Deaths            int                     `json:"deaths"`
	Suicides          int                     `json:"suicides"`
	Dominations       int                     `json:"dominations"`
	Dominated         int                     `json:"dominated"`
	Revenges          int                     `json:"revenges"`
	Damage            int                     `json:"damage"`
	DamageTaken       int                     `json:"damage_taken"`
	HealingTaken      int                     `json:"healing_taken"`
	HealthPacks       int                     `json:"health_packs"`
	HealingPacks      int                     `json:"healing_packs"` // Healing from packs
	Captures          int                     `json:"captures"`
	CapturesBlocked   int                     `json:"captures_blocked"`
	Extinguishes      int                     `json:"extinguishes"`
	BuildingBuilt     int                     `json:"building_built"`
	BuildingDestroyed int                     `json:"building_destroyed"` // Opposing team buildings
	Backstabs         int                     `json:"backstabs"`
	Airshots          int                     `json:"airshots"`
	Headshots         int                     `json:"headshots"`
	Shots             int                     `json:"shots"`
	Hits              int                     `json:"hits"`
	MedicStats        *MatchHealer            `json:"medic_stats"`
	Classes           []MatchPlayerClass      `json:"classes"`
	Killstreaks       []MatchPlayerKillstreak `json:"killstreaks"`
}

func (player MatchPlayer) KDRatio() float64 {
	if player.Deaths <= 0 {
		return -1
	}

	return math.Ceil((float64(player.Kills)/float64(player.Deaths))*100) / 100
}

func (player MatchPlayer) KDARatio() float64 {
	if player.Deaths <= 0 {
		return -1
	}

	return math.Ceil((float64(player.Kills+player.Assists)/float64(player.Deaths))*100) / 100
}

func (player MatchPlayer) DamagePerMin() int {
	return int(float64(player.Damage) / player.TimeEnd.Sub(player.TimeStart).Minutes())
}

type MatchHealer struct {
	MatchMedicID        int64   `json:"match_medic_id"`
	MatchPlayerID       int64   `json:"match_player_id"`
	Healing             int     `json:"healing"`
	ChargesUber         int     `json:"charges_uber"`
	ChargesKritz        int     `json:"charges_kritz"`
	ChargesVacc         int     `json:"charges_vacc"`
	ChargesQuickfix     int     `json:"charges_quickfix"`
	Drops               int     `json:"drops"`
	NearFullChargeDeath int     `json:"near_full_charge_death"`
	AvgUberLength       float32 `json:"avg_uber_length"`
	MajorAdvLost        int     `json:"major_adv_lost"`
	BiggestAdvLost      int     `json:"biggest_adv_lost"`
}

func (h MatchHealer) HealingPerMin(matchDuration time.Duration) int {
	if h.Healing <= 0 {
		return 0
	}

	return int(float64(h.Healing) / matchDuration.Minutes())
}

type MatchWeapon struct {
	PlayerWeaponID int64 `json:"player_weapon_id"`
	MatchPlayerID  int64 `json:"match_player_id"`
}

type MatchResult struct {
	MatchID     uuid.UUID                `json:"match_id"`
	ServerID    int                      `json:"server_id"`
	Title       string                   `json:"title"`
	MapName     string                   `json:"map_name"`
	TeamScores  logparse.TeamScores      `json:"team_scores"`
	TimeStart   time.Time                `json:"time_start"`
	TimeEnd     time.Time                `json:"time_end"`
	PlayerStats []MatchPlayer            `json:"player_stats"`
	Chat        []QueryChatHistoryResult `json:"chat"`
}

func (match *MatchResult) TopPlayers() []MatchPlayer {
	players := match.PlayerStats

	sort.SliceStable(players, func(i, j int) bool {
		return players[i].Kills > players[j].Kills
	})

	return players
}

func (db *Store) matchGetPlayerClasses(ctx context.Context, matchID uuid.UUID) (map[steamid.SID64][]MatchPlayerClass, error) {
	const query = `
		SELECT mp.steam_id, c.match_player_class_id, c.match_player_id, c.player_class_id,c.kills, 
		   c.assists, c.deaths, c.playtime, c.dominations, c.dominated, c.revenges, c.damage, c.damage_taken, c.healing_taken,
		   c.captures, c.captures_blocked, c.buildings_destroyed
		FROM match_player_class c
		LEFT JOIN match_player mp on mp.match_player_id = c.match_player_id
		WHERE mp.match_id = $1`

	rows, errRows := db.Query(ctx, query, matchID)
	if errRows != nil {
		return nil, Err(errRows)
	}

	defer rows.Close()

	results := map[steamid.SID64][]MatchPlayerClass{}

	for rows.Next() {
		var (
			steamID int64
			stats   MatchPlayerClass
		)

		if errScan := rows.
			Scan(&steamID, &stats.MatchPlayerClassID, &stats.MatchPlayerID, &stats.PlayerClass,
				&stats.Kills, &stats.Assists, &stats.Deaths, &stats.Playtime, &stats.Dominations, &stats.Dominated,
				&stats.Revenges, &stats.Damage, &stats.DamageTaken, stats.HealingTaken, &stats.Captures,
				&stats.CapturesBlocked, &stats.BuildingDestroyed); errScan != nil {
			return nil, Err(errScan)
		}

		sid := steamid.New(steamID)

		res, found := results[sid]
		if !found {
			res = []MatchPlayerClass{}
		}

		results[sid] = append(res, stats)
	}

	return results, nil
}

func (db *Store) matchGetPlayerKillstreak(ctx context.Context, matchID uuid.UUID) (map[steamid.SID64][]MatchPlayerKillstreak, error) {
	const query = `
		SELECT mp.steam_id, k.match_player_id, k.player_class_id, k.killstreak, k.duration
		FROM match_player_killstreak k
		LEFT JOIN match_player mp on mp.match_player_id = k.match_player_id
		WHERE mp.match_id = $1`

	rows, errRows := db.Query(ctx, query, matchID)
	if errRows != nil {
		return nil, Err(errRows)
	}

	defer rows.Close()

	results := map[steamid.SID64][]MatchPlayerKillstreak{}

	for rows.Next() {
		var (
			steamID int64
			stats   MatchPlayerKillstreak
		)

		if errScan := rows.
			Scan(&steamID, &stats.MatchPlayerID, &stats.PlayerClass, &stats.Killstreak, &stats.Duration); errScan != nil {
			return nil, Err(errScan)
		}

		sid := steamid.New(steamID)

		res, found := results[sid]
		if !found {
			res = []MatchPlayerKillstreak{}
		}

		results[sid] = append(res, stats)
	}

	return results, nil
}

func (db *Store) matchGetPlayers(ctx context.Context, matchID uuid.UUID) ([]MatchPlayer, error) {
	const queryPlayer = `
		SELECT p.match_player_id,
			   p.steam_id,
			   p.team,
			   p.time_start,
			   p.time_end,
			   coalesce(SUM(w.kills), 0)    as kills,
			   coalesce(SUM(c.assists), 0)    as assists,
			   coalesce(SUM(c.deaths), 0)    as deaths,
			   coalesce(SUM(c.dominations), 0)    as dominations,
			   coalesce(SUM(c.dominated), 0)    as dominated,
			   coalesce(SUM(c.revenges), 0)    as revenges,
			   coalesce(SUM(w.damage), 0)    as damage,
			   coalesce(SUM(c.damage_taken), 0)    as damage_taken,
			   coalesce(SUM(c.healing_taken), 0)    as healing_taken,
			   p.health_packs,
			   coalesce(SUM(w.backstabs), 0) as backstabs,
			   coalesce(SUM(w.headshots), 0) as headshots,
			   coalesce(SUM(w.airshots), 0)  as airshots,
			   coalesce(SUM(c.captures), 0)    as captures,
			   coalesce(SUM(c.captures_blocked), 0)    as captures_blocked,
			   coalesce(SUM(w.shots), 0)     as shots,
			   p.extinguishes,
			   coalesce(SUM(w.hits), 0)      as hits,
			   p.buildings,
			   coalesce(SUM(c.buildings_destroyed), 0)    as buildings_destroyed,
			   pe.personaname,
			   pe.avatarhash
		FROM match_player p
		LEFT JOIN match_weapon w on p.match_player_id = w.match_player_id
		LEFT JOIN match_player_class c on c.match_player_id = p.match_player_id
		LEFT JOIN person pe on p.steam_id = pe.steam_id
		WHERE p.match_id = $1
		GROUP BY p.match_player_id, pe.personaname, pe.avatarhash`

	var players []MatchPlayer

	playerRows, errPlayer := db.Query(ctx, queryPlayer, matchID)
	if errPlayer != nil {
		if errors.Is(errPlayer, ErrNoResult) {
			return []MatchPlayer{}, nil
		}

		return nil, errors.Wrapf(errPlayer, "Failed to query match players")
	}

	defer playerRows.Close()

	for playerRows.Next() {
		var (
			mpSum   = MatchPlayer{}
			steamID int64
		)

		if errRow := playerRows.
			Scan(&mpSum.MatchPlayerID, &steamID, &mpSum.Team, &mpSum.TimeStart, &mpSum.TimeEnd,
				&mpSum.Kills, &mpSum.Assists, &mpSum.Deaths, &mpSum.Dominations, &mpSum.Dominated, &mpSum.Revenges,
				&mpSum.Damage, &mpSum.DamageTaken, &mpSum.HealingTaken, &mpSum.HealingPacks, &mpSum.Backstabs,
				&mpSum.Headshots, &mpSum.Airshots, &mpSum.Captures, &mpSum.CapturesBlocked, &mpSum.Shots,
				&mpSum.Extinguishes, &mpSum.Hits, &mpSum.BuildingBuilt, &mpSum.BuildingDestroyed, &mpSum.Name,
				&mpSum.AvatarHash); errRow != nil {
			return nil, errors.Wrapf(errPlayer, "Failed to scan match players")
		}

		mpSum.SteamID = steamid.New(steamID)
		players = append(players, mpSum)
	}

	return players, nil
}

func (db *Store) matchGetMedics(ctx context.Context, matchID uuid.UUID) (map[steamid.SID64]MatchHealer, error) {
	const query = `
		SELECT m.match_medic_id,
			   m.match_player_id,
			   m.healing,
			   m.drops,
			   m.near_full_charge_death,
			   m.avg_uber_length,
			   m.major_adv_lost,
			   m.biggest_adv_lost,
			   m.charge_uber,
			   m.charge_kritz,
			   m.charge_vacc,
			   m.charge_quickfix
		FROM match_medic m
		LEFT JOIN match_player mp on mp.match_player_id = m.match_player_id
		WHERE mp.match_id = $1`

	medics := map[steamid.SID64]MatchHealer{}

	medicRows, errMedics := db.Query(ctx, query, matchID)
	if errMedics != nil {
		if errors.Is(errMedics, ErrNoResult) {
			return medics, nil
		}

		return nil, errors.Wrapf(errMedics, "Failed to query match healers")
	}

	defer medicRows.Close()

	for medicRows.Next() {
		var (
			mpSum   = MatchHealer{}
			steamID int64
		)

		if errRow := medicRows.
			Scan(&mpSum.MatchMedicID, &steamID, &mpSum.Healing, &mpSum.Drops,
				&mpSum.NearFullChargeDeath, &mpSum.AvgUberLength, &mpSum.MajorAdvLost,
				&mpSum.BiggestAdvLost, &mpSum.ChargesUber, &mpSum.ChargesKritz,
				&mpSum.ChargesVacc, &mpSum.ChargesQuickfix); errRow != nil {
			return nil, errors.Wrapf(errMedics, "Failed to scan match healer")
		}

		sid := steamid.New(steamID)

		medics[sid] = mpSum
	}

	if medicRows.Err() != nil {
		return medics, errors.Wrap(medicRows.Err(), "medicRows error returned")
	}

	return medics, nil
}

func (db *Store) MatchGetByID(ctx context.Context, matchID uuid.UUID, match *MatchResult) error {
	const query = `
		SELECT match_id, server_id, map, created_on, title, score_red, score_blu, time_end 
		FROM match WHERE match_id = $1`

	if errMatch := db.
		QueryRow(ctx, query, matchID).
		Scan(&match.MatchID, &match.ServerID, &match.MapName, &match.TimeStart,
			&match.Title, &match.TeamScores.Red, &match.TeamScores.Blu, &match.TimeEnd); errMatch != nil {
		return errors.Wrapf(errMatch, "Failed to load root match")
	}

	playerStats, errPlayerStats := db.matchGetPlayers(ctx, matchID)
	if errPlayerStats != nil {
		return errors.Wrap(errPlayerStats, "Failed to fetch match players")
	}

	match.PlayerStats = playerStats

	playerClasses, errPlayerClasses := db.matchGetPlayerClasses(ctx, matchID)
	if errPlayerClasses != nil {
		return errors.Wrap(errPlayerClasses, "Failed to fetch player class stats")
	}

	for _, player := range playerStats {
		if classes, found := playerClasses[player.SteamID]; found {
			player.Classes = classes
		}
	}

	playerKillstreaks, errPlayerKillstreaks := db.matchGetPlayerKillstreak(ctx, matchID)
	if errPlayerKillstreaks != nil {
		return errors.Wrap(errPlayerKillstreaks, "Failed to fetch player killstreak stats")
	}

	for _, player := range playerStats {
		if killstreaks, found := playerKillstreaks[player.SteamID]; found {
			player.Killstreaks = killstreaks
		}
	}

	medicStats, errMedics := db.matchGetMedics(ctx, matchID)
	if errMedics != nil {
		return errors.Wrap(errMedics, "Failed to fetch match medics")
	}

	for steamID, stats := range medicStats {
		localStats := stats

		for _, player := range playerStats {
			if player.SteamID == steamID {
				player.MedicStats = &localStats

				break
			}
		}
	}

	chat, _, errChat := db.QueryChatHistory(ctx, ChatHistoryQueryFilter{
		ServerID:      match.ServerID,
		SentAfter:     &match.TimeStart,
		SentBefore:    &match.TimeEnd,
		Unrestricted:  true,
		DontCalcTotal: true,
	})

	if errChat != nil && !errors.Is(errChat, ErrNoResult) {
		return errors.Wrap(errMedics, "Failed to fetch match chat history")
	}

	match.Chat = chat

	return nil
}

var (
	ErrIncompleteMatch     = errors.New("Insufficient match data")
	ErrInsufficientPlayers = errors.New("Insufficient match players")
)

func (db *Store) MatchSave(ctx context.Context, match *logparse.Match) error {
	const (
		minPlayers = 6
		query      = `
		INSERT INTO match (match_id, server_id, map, created_on, title, match_raw, score_red, score_blu, time_end) 
		VALUES ($1, $2, $3, $4, $5, $6,$7, $8, $9) 
		RETURNING match_id`
	)

	if match.CreatedOn == nil || match.MapName == "" {
		return ErrIncompleteMatch
	}

	if len(match.PlayerSums) < minPlayers {
		return ErrInsufficientPlayers
	}

	transaction, errTx := db.conn.Begin(ctx)
	if errTx != nil {
		return errors.Wrap(errTx, "Failed to create match tx")
	}

	if errQuery := transaction.
		QueryRow(ctx, query, match.MatchID, match.ServerID, match.MapName, match.CreatedOn, match.Title, match,
			match.TeamScores.Red, match.TeamScores.Blu, match.TimeEnd).
		Scan(&match.MatchID); errQuery != nil {
		if errRollback := transaction.Rollback(ctx); errRollback != nil {
			db.log.Error("Failed to rollback tx", zap.Error(errRollback))
		}

		return errors.Wrap(errQuery, "Failed to create match")
	}

	for _, player := range match.PlayerSums {
		if !player.SteamID.Valid() {
			// TODO Why can this happen? stv host?
			continue
		}

		var loadPlayerTest Person
		if errPlayer := db.GetOrCreatePersonBySteamID(ctx, player.SteamID, &loadPlayerTest); errPlayer != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				db.log.Error("Failed to rollback tx", zap.Error(errRollback))
			}

			return errors.Wrapf(errPlayer, "Failed to load person")
		}

		if errSave := saveMatchPlayerStats(ctx, transaction, match, player); errSave != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				db.log.Error("Failed to rollback tx", zap.Error(errRollback))
			}

			return errSave
		}

		if errSave := saveMatchWeaponStats(ctx, transaction, player); errSave != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				db.log.Error("Failed to rollback tx", zap.Error(errRollback))
			}

			return errSave
		}

		if errSave := saveMatchPlayerClassStats(ctx, transaction, player); errSave != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				db.log.Error("Failed to rollback tx", zap.Error(errRollback))
			}

			return errSave
		}

		if errSave := saveMatchKillstreakStats(ctx, transaction, player); errSave != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				db.log.Error("Failed to rollback tx", zap.Error(errRollback))
			}

			return errSave
		}

		if player.HealingStats != nil {
			if errSave := saveMatchMedicStats(ctx, transaction, player.MatchPlayerID, player.HealingStats); errSave != nil {
				if errRollback := transaction.Rollback(ctx); errRollback != nil {
					db.log.Error("Failed to rollback tx", zap.Error(errRollback))
				}

				return errSave
			}
		}
	}

	if errCommit := transaction.Commit(ctx); errCommit != nil {
		return errors.Wrapf(errCommit, "Failed to commit match")
	}

	return nil
}

func saveMatchPlayerStats(ctx context.Context, transaction pgx.Tx, match *logparse.Match, stats *logparse.PlayerStats) error {
	const playerQuery = `
		INSERT INTO match_player (
			match_id, steam_id, team, time_start, time_end, health_packs, extinguishes, buildings
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
		RETURNING match_player_id`

	endTime := stats.TimeEnd

	if endTime == nil {
		// Use match end time
		endTime = match.TimeEnd
	}

	if errPlayerExec := transaction.
		QueryRow(ctx, playerQuery, match.MatchID, stats.SteamID.Int64(), stats.Team, stats.TimeStart,
			endTime, stats.HealthPacks(), stats.Extinguishes(), stats.BuildingBuilt).
		Scan(&stats.MatchPlayerID); errPlayerExec != nil {
		return errors.Wrapf(errPlayerExec, "Failed to write player sum")
	}

	return nil
}

func saveMatchMedicStats(ctx context.Context, transaction pgx.Tx, matchPlayerID int64, stats *logparse.HealingStats) error {
	const medicQuery = `
		INSERT INTO match_medic (
			match_player_id, healing, drops, near_full_charge_death, avg_uber_length,  major_adv_lost, biggest_adv_lost, 
            charge_kritz, charge_quickfix, charge_uber, charge_vacc)
        VALUES ($1, $2, $3, $4, $5,$6, $7, $8, $9, $10,$11) 
		RETURNING match_medic_id`

	if errMedExec := transaction.
		QueryRow(ctx, medicQuery, matchPlayerID, stats.Healing,
			stats.DropsTotal(), stats.NearFullChargeDeath, stats.AverageUberLength(), stats.MajorAdvLost, stats.BiggestAdvLost,
			stats.Charges[logparse.Kritzkrieg], stats.Charges[logparse.QuickFix],
			stats.Charges[logparse.Uber], stats.Charges[logparse.Vaccinator]).
		Scan(&stats.MatchMedicID); errMedExec != nil {
		return errors.Wrapf(errMedExec, "Failed to write medic sum")
	}

	return nil
}

func saveMatchWeaponStats(ctx context.Context, transaction pgx.Tx, player *logparse.PlayerStats) error {
	const query = `
		INSERT INTO match_weapon (match_player_id, weapon_id, kills, damage, shots, hits, backstabs, headshots, airshots) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) 
		RETURNING player_weapon_id`

	for weapon, info := range player.WeaponInfo {
		if _, errWeapon := transaction.
			Exec(ctx, query, player.MatchPlayerID, weapon, info.Kills, info.Damage, info.Shots, info.Hits,
				info.BackStabs, info.Headshots, info.Airshots); errWeapon != nil {
			return errors.Wrapf(errWeapon, "Failed to write weapon stats")
		}
	}

	return nil
}

func saveMatchPlayerClassStats(ctx context.Context, transaction pgx.Tx, player *logparse.PlayerStats) error {
	const query = `
		INSERT INTO match_player_class (
			match_player_id, player_class_id, kills, assists, deaths, playtime, dominations, dominated, revenges, 
		    damage, damage_taken, healing_taken, captures, captures_blocked, buildings_destroyed) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	for class, stats := range player.Classes {
		if _, errWeapon := transaction.
			Exec(ctx, query, player.MatchPlayerID, class, stats.Kills, stats.Assists, stats.Deaths, stats.Playtime,
				stats.Dominations, stats.Dominated, stats.Revenges, stats.Damage, stats.DamageTaken, stats.HealingTaken,
				stats.Captures, stats.CapturesBlocked, stats.BuildingsDestroyed); errWeapon != nil {
			return errors.Wrapf(errWeapon, "Failed to write player class stats")
		}
	}

	return nil
}

func saveMatchKillstreakStats(ctx context.Context, transaction pgx.Tx, player *logparse.PlayerStats) error {
	const query = `
		INSERT INTO match_player_killstreak (match_player_id, player_class_id, killstreak, duration) 
		VALUES ($1, $2, $3, $4)`

	for class, stats := range player.KillStreaks {
		if _, errWeapon := transaction.
			Exec(ctx, query, player.MatchPlayerID, class, stats.Killstreak, stats.Duration); errWeapon != nil {
			return errors.Wrapf(errWeapon, "Failed to write player class stats")
		}
	}

	return nil
}
