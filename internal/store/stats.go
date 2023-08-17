package store

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type LocalTF2StatsSnapshot struct {
	StatID          int64          `json:"stat_id"`
	Players         int            `json:"players"`
	CapacityFull    int            `json:"capacity_full"`
	CapacityEmpty   int            `json:"capacity_empty"`
	CapacityPartial int            `json:"capacity_partial"`
	MapTypes        map[string]int `json:"map_types"`
	Servers         map[string]int `json:"servers"`
	Regions         map[string]int `json:"regions"`
	CreatedOn       time.Time      `json:"created_on"`
}

func NewLocalTF2Stats() LocalTF2StatsSnapshot {
	return LocalTF2StatsSnapshot{
		MapTypes:  map[string]int{},
		Regions:   map[string]int{},
		Servers:   map[string]int{},
		CreatedOn: time.Now(),
	}
}

type Stats struct {
	BansTotal     int `json:"bans_total"`
	BansDay       int `json:"bans_day"`
	BansWeek      int `json:"bans_week"`
	BansMonth     int `json:"bans_month"`
	Bans3Month    int `json:"bans3_month"`
	Bans6Month    int `json:"bans6_month"`
	BansYear      int `json:"bans_year"`
	BansCIDRTotal int `json:"bans_cidr_total"`
	AppealsOpen   int `json:"appeals_open"`
	AppealsClosed int `json:"appeals_closed"`
	FilteredWords int `json:"filtered_words"`
	ServersAlive  int `json:"servers_alive"`
	ServersTotal  int `json:"servers_total"`
}

func (db *Store) LoadWeapons(ctx context.Context) error {
	const query = `INSERT INTO weapon (weapon_id, name) VALUES ($1, $2)`

	index := 1

	for {
		weapon := logparse.Weapon(index)

		if errSave := db.Exec(ctx, query, weapon, weapon.String()); errSave != nil && errors.Is(errSave, ErrDuplicate) {
			return errSave
		}

		if weapon == logparse.Wrench {
			break
		}

		index++
	}

	return nil
}

func (db *Store) MatchSave(ctx context.Context, match *logparse.Match) error {
	const query = `
		INSERT INTO match (server_id, map, created_on, title, match_raw, score_red, score_blu, time_end) 
		VALUES ($1, $2, $3, $4, $5, $6,$7, $8) 
		RETURNING match_id`

	transaction, errTx := db.conn.Begin(ctx)
	if errTx != nil {
		return errors.Wrap(errTx, "Failed to create match tx")
	}

	if errQuery := transaction.
		QueryRow(ctx, query, match.ServerID, match.MapName, match.CreatedOn, match.Title, match,
			match.TeamScores.Red, match.TeamScores.Blu, match.TimeEnd).
		Scan(&match.MatchID); errQuery != nil {
		_ = transaction.Rollback(ctx)

		return errors.Wrap(errQuery, "Failed to save match")
	}

	for _, playerStats := range match.PlayerSums {
		if !playerStats.SteamID.Valid() {
			// TODO Why can this happen? stv host?
			continue
		}

		var player Person
		if errPlayer := db.GetOrCreatePersonBySteamID(ctx, playerStats.SteamID, &player); errPlayer != nil {
			_ = transaction.Rollback(ctx)

			return errors.Wrapf(errPlayer, "Failed to create person")
		}
	}

	for _, playerSum := range match.PlayerSums {
		if !playerSum.SteamID.Valid() {
			// TODO Why can this happen? stv host?
			continue
		}

		endTime := match.CreatedOn

		if playerSum.TimeEnd != nil {
			// Use match end time
			endTime = match.TimeEnd
		}

		if errMedic := saveMatchPlayerStats(ctx, transaction, match.MatchID, playerSum, endTime); errMedic != nil {
			_ = transaction.Rollback(ctx)

			return errMedic
		}
	}

	for _, medic := range match.Healers() {
		if errMedic := saveMatchMedicStats(ctx, transaction, match.MatchID, medic.SteamID, medic.HealingStats); errMedic != nil {
			_ = transaction.Rollback(ctx)

			return errMedic
		}
	}

	for _, player := range match.PlayerSums {
		if errWi := saveMatchWeaponStats(ctx, transaction, player); errWi != nil {
			_ = transaction.Rollback(ctx)

			return errWi
		}
	}

	if errCommit := transaction.Commit(ctx); errCommit != nil {
		return errors.Wrapf(errCommit, "Failed to commit match")
	}

	return nil
}

func saveMatchPlayerStats(ctx context.Context, transaction pgx.Tx, matchID int, stats *logparse.PlayerStats, endTime *time.Time) error {
	const playerQuery = `
		INSERT INTO match_player (
			match_id, steam_id, team, time_start, time_end, assists, deaths, dominations, dominated,
			revenges, damage_taken, healing_taken, health_packs, captures, extinguishes, buildings, buildings_destroyed,
		    captures_blocked)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18) 
		RETURNING match_player_id`

	if errPlayerExec := transaction.
		QueryRow(ctx, playerQuery, matchID, stats.SteamID.Int64(), stats.Team, stats.TimeStart,
			endTime, stats.Assists, stats.Deaths(), stats.DominationCount(),
			stats.DominatedCount(), stats.RevengeCount(), stats.DamageTaken(), stats.HealingTaken(),
			stats.HealthPacks(), stats.CaptureCount(), stats.Extinguishes(), stats.BuildingBuilt,
			stats.BuildingDestroyed, stats.CapturesBlockedCount()).
		Scan(&stats.MatchPlayerID); errPlayerExec != nil {
		_ = transaction.Rollback(ctx)

		return errors.Wrapf(errPlayerExec, "Failed to write player sum")
	}

	return nil
}

func saveMatchMedicStats(ctx context.Context, transaction pgx.Tx, matchID int, steamID steamid.SID64, stats *logparse.HealingStats) error {
	const medicQuery = `
		INSERT INTO match_medic (
			match_id, steam_id, healing, drops, near_full_charge_death, avg_uber_length,  major_adv_lost, biggest_adv_lost, 
            charge_kritz, charge_quickfix, charge_uber, charge_vacc)
        VALUES ($1, $2, $3, $4, $5,$6, $7, $8, $9, $10,$11, $12) 
		RETURNING match_medic_id`

	if errMedExec := transaction.
		QueryRow(ctx, medicQuery, matchID, steamID.Int64(), stats.Healing,
			stats.DropsTotal(), stats.NearFullChargeDeath, stats.AverageUberLength(), stats.MajorAdvLost, stats.BiggestAdvLost,
			stats.Charges[logparse.Kritzkrieg], stats.Charges[logparse.QuickFix],
			stats.Charges[logparse.Uber], stats.Charges[logparse.Vaccinator]).
		Scan(&stats.MatchMedicID); errMedExec != nil {
		_ = transaction.Rollback(ctx)

		return errors.Wrapf(errMedExec, "Failed to write medic sum")
	}

	return nil
}

func saveMatchWeaponStats(ctx context.Context, transaction pgx.Tx, player *logparse.PlayerStats) error {
	const weaponQuery = `
		INSERT INTO match_weapon (match_player_id, weapon_id, kills, damage, shots, hits, backstabs, headshots, airshots) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) 
		RETURNING player_weapon_id`

	for weapon, info := range player.WeaponInfo {
		if _, errWeapon := transaction.
			Exec(ctx, weaponQuery, player.MatchPlayerID, weapon, info.Kills, info.Damage, info.Shots, info.Hits,
				info.BackStabs, info.Headshots, info.Airshots); errWeapon != nil {
			_ = transaction.Rollback(ctx)

			return errors.Wrapf(errWeapon, "Failed to write weapon stats")
		}
	}

	return nil
}

type Weapon struct {
	WeaponID logparse.Weapon `json:"weapon_id"`
	Name     string          `json:"name"`
}

func (db *Store) SaveWeapon(ctx context.Context, weapon *Weapon) error {
	if weapon.WeaponID > 0 {
		updateQuery, updateArgs, errUpdateQuery := db.sb.
			Update("weapon").
			Set("name", weapon.Name).
			Where(sq.Eq{"weapon_id": weapon.WeaponID}).
			ToSql()

		if errUpdateQuery != nil {
			return errors.Wrap(errUpdateQuery, "Failed to make query")
		}

		if errSave := db.Exec(ctx, updateQuery, updateArgs...); errSave != nil {
			return errSave
		}

		return nil
	}

	const wq = `INSERT INTO weapon (weapon_id, name) VALUES ($1, $2)`

	if errSave := db.
		Exec(ctx, wq, weapon.WeaponID, weapon.Name); errSave != nil {
		return errSave
	}

	return nil
}

func (db *Store) Weapons(ctx context.Context) ([]Weapon, error) {
	query, args, errQuery := db.sb.
		Select("weapon_id", "key", "name").
		From("weapon").
		ToSql()

	if errQuery != nil {
		return nil, errors.Wrap(errQuery, "Failed to make weapons query")
	}

	rows, errRows := db.Query(ctx, query, args...)
	if errRows != nil {
		return nil, errRows
	}
	defer rows.Close()

	var weapons []Weapon

	for rows.Next() {
		var weapon Weapon
		if errScan := rows.Scan(&weapon.WeaponID, &weapon.Name); errScan != nil {
			return nil, errors.Wrap(errScan, "Failed to scan weapon")
		}

		weapons = append(weapons, weapon)
	}

	if errRow := rows.Err(); errRow != nil {
		return nil, errors.Wrap(errRow, "weapons rows error")
	}

	return weapons, nil
}

type MatchesQueryOpts struct {
	QueryFilter
	SteamID   steamid.SID64 `json:"steam_id"`
	ServerID  int           `json:"server_id"`
	Map       string        `json:"map"`
	TimeStart *time.Time    `json:"time_start,omitempty"`
	TimeEnd   *time.Time    `json:"time_end,omitempty"`
}

// func (db *Store) Matches(ctx context.Context, opts MatchesQueryOpts) (logparse.MatchSummaryCollection, error) {
//	builder := db.sb.
//		Select("m.match_id", "m.server_id", "m.map", "m.created_on", "COALESCE(sum(mp.kills), 0)", "COALESCE(sum(mp.assists), 0)", "COALESCE(sum(mp.damage), 0)", "COALESCE(sum(mp.healing), 0)", "COALESCE(sum(mp.airshots), 0)").
//		From("match m").
//		LeftJoin("match_player mp on m.match_id = mp.match_id").
//		GroupBy("m.match_id")
//	if opts.Map != "" {
//		builder = builder.Where(sq.Eq{"m.map_name": opts.Map})
//	}
//
//	if opts.SteamID.Valid() {
//		builder = builder.Where(sq.Eq{"mp.steam_id": opts.SteamID.Int64()})
//	}
//
//	if opts.Desc {
//		builder = builder.OrderBy("m.match_id DESC")
//	} else {
//		builder = builder.OrderBy("m.match_id ASC")
//	}
//
//	if opts.Limit > 0 {
//		builder = builder.Limit(opts.Limit)
//	}
//
//	query, args, errQueryArgs := builder.ToSql()
//	if errQueryArgs != nil {
//		return nil, errors.Wrapf(errQueryArgs, "Failed to build query")
//	}
//
//	rows, errQuery := db.Query(ctx, query, args...)
//	if errQuery != nil {
//		return nil, errors.Wrapf(errQuery, "Failed to query matches")
//	}
//
//	defer rows.Close()
//
//	var matches logparse.MatchSummaryCollection
//
//	for rows.Next() {
//		var m logparse.MatchSummary
//		if errScan := rows.Scan(&m.MatchID, &m.ServerID, &m.MapName, &m.CreatedOn /*&m.PlayerCount,*/, &m.Kills, &m.Assists, &m.Damage, &m.MedicStats, &m.Airshots); errScan != nil {
//			return nil, errors.Wrapf(errScan, "Failed to scan match row")
//		}
//
//		matches = append(matches, &m)
//	}
//
//	return matches, nil
// }

type MatchPlayer struct {
	MatchPlayerID     int64                  `json:"match_player_id"`
	SteamID           steamid.SID64          `json:"steam_id"`
	Team              logparse.Team          `json:"team"`
	Name              string                 `json:"name"`
	TimeStart         time.Time              `json:"time_start"`
	TimeEnd           time.Time              `json:"time_end"`
	Assists           int                    `json:"assists"`
	Deaths            int                    `json:"deaths"`
	Suicides          int                    `json:"suicides"`
	Dominations       int                    `json:"dominations"`
	Dominated         int                    `json:"dominated"`
	Revenges          int                    `json:"revenges"`
	Damage            int                    `json:"damage"`
	DamageTaken       int                    `json:"damage_taken"`
	HealingTaken      int                    `json:"healing_taken"`
	HealthPacks       int                    `json:"health_packs"`
	HealingPacks      int                    `json:"healing_packs"` // Healing from packs
	Captures          int                    `json:"captures"`
	CapturesBlocked   int                    `json:"captures_blocked"`
	Extinguishes      int                    `json:"extinguishes"`
	BuildingBuilt     int                    `json:"building_built"`
	BuildingDestroyed int                    `json:"building_destroyed"` // Opposing team buildings
	Classes           []logparse.PlayerClass `json:"classes"`
	KillStreaks       []int                  `json:"kill_streaks"`
	Backstabs         int                    `json:"backstabs"`
	Airshots          int                    `json:"airshots"`
	Headshots         int                    `json:"headshots"`
	Shots             int                    `json:"shots"`
	Hits              int                    `json:"hits"`
}

type MatchHealer struct {
	MatchMedicID        int64 `json:"match_medic_id"`
	SteamID             steamid.SID64
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

type MatchWeapon struct {
	PlayerWeaponID int64 `json:"player_weapon_id"`
	MatchPlayerID  int64 `json:"match_player_id"`
}

type MatchResult struct {
	MatchID     int                 `json:"match_id"`
	ServerID    int                 `json:"server_id"`
	Title       string              `json:"title"`
	MapName     string              `json:"map_name"`
	TeamScores  logparse.TeamScores `json:"team_scores"`
	TimeStart   time.Time           `json:"time_start"`
	TimeEnd     time.Time           `json:"time_end"`
	PlayerStats []MatchPlayer       `json:"player_stats"`
	MedicStats  []MatchHealer       `json:"medic_stats"`
	Players     []Person            `json:"players"`
}

func (db *Store) matchGetPlayers(ctx context.Context, matchID int) ([]MatchPlayer, error) {
	const queryPlayer = `
		SELECT p.match_player_id,
			   p.steam_id,
			   p.team,
			   p.time_start,
			   p.time_end,
			   p.assists,
			   p.deaths,
			   p.dominations,
			   p.dominated,
			   p.revenges,
			   coalesce(SUM(w.damage), 0)    as damage,
			   p.damage_taken,
			   p.healing_taken,
			   p.health_packs,
			   coalesce(SUM(w.backstabs), 0) as backstabs,
			   coalesce(SUM(w.headshots), 0) as headshots,
			   coalesce(SUM(w.airshots), 0)  as airshots,
			   captures,
			   coalesce(SUM(w.shots), 0)     as shots,
			   extinguishes,
			   coalesce(SUM(w.hits), 0)      as hits,
			   buildings,
			   buildings_destroyed
		FROM match_player p
		LEFT JOIN match_weapon w on p.match_player_id = w.match_player_id
		WHERE p.match_id = $1
		GROUP BY p.match_player_id`

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
				&mpSum.Assists, &mpSum.Deaths, &mpSum.Dominations, &mpSum.Dominated, &mpSum.Revenges,
				&mpSum.DamageTaken, &mpSum.DamageTaken, &mpSum.HealingTaken, &mpSum.HealingPacks, &mpSum.Backstabs,
				&mpSum.Headshots, &mpSum.Airshots, &mpSum.Captures, &mpSum.Shots, &mpSum.Extinguishes, &mpSum.Hits,
				&mpSum.BuildingBuilt, &mpSum.BuildingDestroyed); errRow != nil {
			return nil, errors.Wrapf(errPlayer, "Failed to scan match players")
		}

		mpSum.SteamID = steamid.New(steamID)
		players = append(players, mpSum)
	}

	return players, nil
}

func (db *Store) matchGetMedics(ctx context.Context, matchID int) ([]MatchHealer, error) {
	const query = `
		SELECT m.match_medic_id,
			   m.match_id,
			   m.steam_id,
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
		WHERE m.match_id = $1`

	var medics []MatchHealer

	medicRows, errMedics := db.Query(ctx, query, matchID)
	if errMedics != nil {
		if errors.Is(errMedics, ErrNoResult) {
			return []MatchHealer{}, nil
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

		mpSum.SteamID = steamid.New(steamID)
		medics = append(medics, mpSum)
	}

	if medicRows.Err() != nil {
		return []MatchHealer{}, errors.Wrap(medicRows.Err(), "medicRows error returned")
	}

	return medics, nil
}

func (db *Store) MatchGetByID(ctx context.Context, matchID int) (*MatchResult, error) {
	const query = `
		SELECT match_id, server_id, map, created_on, title, score_red, score_blu, time_end 
		FROM match WHERE match_id = $1`

	var match MatchResult
	if errMatch := db.
		QueryRow(ctx, query, matchID).
		Scan(&match.MatchID, &match.ServerID, &match.MapName, &match.TimeStart,
			&match.Title, &match.TeamScores.Red, &match.TeamScores.Blu, &match.TimeEnd); errMatch != nil {
		return nil, errors.Wrapf(errMatch, "Failed to load root match")
	}

	playerStats, errPlayerStats := db.matchGetPlayers(ctx, matchID)
	if errPlayerStats != nil {
		return nil, errors.Wrap(errPlayerStats, "Failed to fetch match players")
	}

	match.PlayerStats = playerStats

	medicStats, errMedics := db.matchGetMedics(ctx, matchID)
	if errMedics != nil {
		return nil, errors.Wrap(errMedics, "Failed to fetch match medics")
	}

	match.MedicStats = medicStats

	var ids steamid.Collection
	for _, p := range match.PlayerStats {
		ids = append(ids, p.SteamID)
	}

	players, errPlayers := db.GetPeopleBySteamID(ctx, ids)
	if errPlayers != nil {
		return nil, errors.Wrapf(errPlayers, "Failed to load players")
	}

	match.Players = players

	return &match, nil
}

func (db *Store) GetStats(ctx context.Context, stats *Stats) error {
	const query = `
	SELECT 
		(SELECT COUNT(ban_id) FROM ban) as bans_total,
		(SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 DAY')) as bans_day,
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 DAY')) as bans_week,
		(SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 MONTH')) as bans_month, 
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '3 MONTH')) as bans_3month,
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '6 MONTH')) as bans_6month,
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 YEAR')) as bans_year,
		(SELECT COUNT(net_id) FROM ban_net) as bans_cidr,
		(SELECT COUNT(filter_id) FROM filtered_word) as filtered_words,
		(SELECT COUNT(server_id) FROM server) as servers_total`

	if errQuery := db.QueryRow(ctx, query).
		Scan(&stats.BansTotal, &stats.BansDay, &stats.BansWeek, &stats.BansMonth, &stats.Bans3Month, &stats.Bans6Month, &stats.BansYear, &stats.BansCIDRTotal, &stats.FilteredWords, &stats.ServersTotal); errQuery != nil {
		db.log.Error("Failed to fetch stats", zap.Error(errQuery))

		return Err(errQuery)
	}

	return nil
}

var localStatColumns = []string{ //nolint:gochecknoglobals
	"players", "capacity_full", "capacity_empty", "capacity_partial",
	"map_types", "created_on", "regions", "servers",
}

func (db *Store) SaveLocalTF2Stats(ctx context.Context, duration StatDuration, stats LocalTF2StatsSnapshot) error {
	query, args, errQuery := db.sb.Insert(statDurationTable(Local, duration)).
		Columns(localStatColumns...).
		Values(stats.Players, stats.CapacityFull, stats.CapacityEmpty, stats.CapacityPartial,
			stats.MapTypes, stats.CreatedOn, stats.Regions, stats.Servers).
		ToSql()
	if errQuery != nil {
		return errors.Wrapf(errQuery, "Failed to create query")
	}

	return Err(db.Exec(ctx, query, args...))
}

// var globalStatColumns = []string{"players", "bots", "secure", "servers_community", "servers_total",
//	"capacity_full", "capacity_empty", "capacity_partial", "map_types", "created_on", "regions"}
//
// func (database *pgStore) SaveGlobalTF2Stats(ctx context.Context, duration StatDuration, stats state.GlobalTF2StatsSnapshot) error {
//	query, args, errQuery := sb.Insert(statDurationTable(Global, duration)).
//		Columns(globalStatColumns...).
//		Values(stats.Players, stats.Bots, stats.Secure, stats.ServersCommunity, stats.ServersTotal, stats.CapacityFull, stats.CapacityEmpty, stats.CapacityPartial, stats.trimMapTypes(), stats.CreatedOn, stats.Regions).
//		ToSql()
//	if errQuery != nil {
//		return errQuery
//	}
//	return Err(database.Exec(ctx, query, args...))
// }

type StatLocality int

const (
	Local StatLocality = iota
	Global
)

type StatDuration int

const (
	Live StatDuration = iota
	Hourly
	Daily
)

// func fetchGlobalTF2Snapshots(ctx context.Context, database Store, query string, args []any) ([]state.GlobalTF2StatsSnapshot, error) {
//	rows, errExec := database.Query(ctx, query, args...)
//	if errExec != nil {
//		return nil, Err(errExec)
//	}
//	defer rows.Close()
//	var stats []state.GlobalTF2StatsSnapshot
//	for rows.Next() {
//		var stat state.GlobalTF2StatsSnapshot
//		if errScan := rows.Scan(&stat.StatID, &stat.Players, &stat.Bots, &stat.Secure, &stat.ServersCommunity, &stat.ServersTotal, &stat.CapacityFull, &stat.CapacityEmpty, &stat.CapacityPartial, &stat.MapTypes, &stat.CreatedOn, &stat.Regions); errScan != nil {
//			return nil, Err(errScan)
//		}
//		stats = append(stats, stat)
//	}
//	return stats, nil
//}

func (db *Store) fetchLocalTF2Snapshots(ctx context.Context, query string, args []any) ([]LocalTF2StatsSnapshot, error) {
	rows, errExec := db.Query(ctx, query, args...)
	if errExec != nil {
		return nil, Err(errExec)
	}
	defer rows.Close()

	var stats []LocalTF2StatsSnapshot

	for rows.Next() {
		var stat LocalTF2StatsSnapshot
		if errScan := rows.Scan(&stat.StatID, &stat.Players, &stat.CapacityFull, &stat.CapacityEmpty,
			&stat.CapacityPartial, &stat.MapTypes, &stat.CreatedOn, &stat.Regions, &stat.Servers); errScan != nil {
			return nil, Err(errScan)
		}

		stats = append(stats, stat)
	}

	return stats, nil
}

func HourlyIndex(t time.Time) (time.Time, int) {
	curYear, curMon, curDay := t.Date()
	curHour, _, _ := t.Clock()
	curTime := time.Date(curYear, curMon, curDay, curHour, 0, 0, 0, t.Location())

	return curTime, curHour
}

// func DailyIndex(t time.Time) (time.Time, int) {
// 	curYear, curMon, curDay := t.Date()
// 	curTime := time.Date(curYear, curMon, curDay, 0, 0, 0, 0, t.Location())
//
// 	return curTime, curDay
// }

// currentHourlyTime calculates the absolute start of the current hour.
func currentHourlyTime() time.Time {
	now := time.Now()
	year, mon, day := now.Date()
	hour, _, _ := now.Clock()

	return time.Date(year, mon, day, hour, 0, 0, 0, now.Location())
}

// currentDailyTime calculates the absolute start of the current day
// func currentDailyTime() time.Time {
//	now := config.Now()
//	year, mon, day := now.Date()
//	return time.Date(year, mon, day, 0, 0, 0, 0, now.Location())
// }

// type statIndexFunc = func(t time.Time) (time.Time, int)
//
// func (database *pgStore) BuildGlobalTF2Stats(ctx context.Context) error {
//	maxDate := currentHourlyTime()
//	query, args, errQuery := sb.
//		Select(fp.Prepend(globalStatColumns, "stat_id")...).
//		From(statDurationTable(Global, Live)).
//		Where(sq.Lt{"created_on": maxDate}). // Ignore any results until a full hour has passed
//		OrderBy("created_on").
//		ToSql()
//	if errQuery != nil {
//		return Err(errQuery)
//	}
//	stats, errStats := fetchGlobalTF2Snapshots(ctx, database, query, args)
//	if errStats != nil {
//		return errStats
//	}
//	if len(stats) == 0 {
//		return nil
//	}
//	var (
//		hourlySums           []state.GlobalTF2StatsSnapshot
//		curSums              *state.GlobalTF2StatsSnapshot
//		tempPlayers          []int
//		tempBots             []int
//		tempServersCommunity []int
//		tempServersTotal     []int
//		tempCapacityFull     []int
//		tempCapacityEmpty    []int
//		tempCapacityPartial  []int
//	)
//	tempMapTypes := map[string][]int{}
//	tempRegions := map[string][]int{}
//	curIdx := 0
//	for _, s := range stats {
//		// Group & sum hourly as the minimum granularity
//		curTime, timeIdx := HourlyIndex(s.CreatedOn)
//		if curIdx == 0 {
//			curIdx = timeIdx
//		} else if curIdx != timeIdx && curSums != nil {
//			// If the hour index changed, flush the current results out
//			sumStat := state.NewGlobalTF2Stats()
//			sumStat.CreatedOn = curTime
//			sumStat.Players = fp.Avg(tempPlayers)
//			sumStat.Bots = fp.Avg(tempBots)
//			sumStat.ServersCommunity = fp.Avg(tempServersCommunity)
//			sumStat.ServersTotal = fp.Avg(tempServersTotal)
//			sumStat.CapacityFull = fp.Avg(tempCapacityFull)
//			sumStat.CapacityEmpty = fp.Avg(tempCapacityEmpty)
//			sumStat.CapacityPartial = fp.Avg(tempCapacityPartial)
//			for k := range tempMapTypes {
//				sumStat.MapTypes[k] = fp.Avg(tempMapTypes[k])
//			}
//			for k := range tempRegions {
//				sumStat.Regions[k] = fp.Avg(tempRegions[k])
//			}
//			hourlySums = append(hourlySums, sumStat)
//			curSums = nil
//			curIdx = timeIdx
//			tempPlayers = nil
//			tempBots = nil
//			tempServersCommunity = nil
//			tempServersTotal = nil
//			tempCapacityFull = nil
//			tempCapacityEmpty = nil
//			tempCapacityPartial = nil
//			tempRegions = map[string][]int{}
//			tempMapTypes = map[string][]int{}
//		}
//		if curSums == nil {
//			curSums = &state.GlobalTF2StatsSnapshot{CreatedOn: curTime}
//		}
//		tempPlayers = append(tempPlayers, s.Players)
//		tempBots = append(tempBots, s.Bots)
//		tempServersCommunity = append(tempServersCommunity, s.ServersCommunity)
//		tempServersTotal = append(tempServersTotal, s.ServersTotal)
//		tempCapacityFull = append(tempCapacityFull, s.CapacityFull)
//		tempCapacityEmpty = append(tempCapacityEmpty, s.CapacityEmpty)
//		tempCapacityPartial = append(tempCapacityPartial, s.CapacityPartial)
//		for k, v := range s.MapTypes {
//			_, found := tempMapTypes[k]
//			if !found {
//				tempMapTypes[k] = []int{}
//			}
//			tempMapTypes[k] = append(tempMapTypes[k], v)
//		}
//		for k, v := range s.Regions {
//			_, found := tempRegions[k]
//			if !found {
//				tempRegions[k] = []int{}
//			}
//			tempRegions[k] = append(tempRegions[k], v)
//		}
//	}
//	for _, hourly := range hourlySums {
//		if errSave := database.SaveGlobalTF2Stats(ctx, Hourly, hourly); errSave != nil {
//			if errors.Is(errSave, ErrDuplicate) {
//				continue
//			}
//			return errSave
//		}
//	}
//
//	var statIds []int64
//	for _, s := range stats {
//		statIds = append(statIds, s.StatID)
//	}
//	// Delete old entries
//	delQuery, delArgs, delQueryErr := sb.
//		Delete(statDurationTable(Global, Live)).
//		Where(sq.Eq{"stat_id": statIds}).
//		ToSql()
//	if delQueryErr != nil {
//		return Err(delQueryErr)
//	}
//	return database.Exec(ctx, delQuery, delArgs...)
// }

func statDurationTable(locality StatLocality, duration StatDuration) string {
	switch locality {
	case Global:
		switch duration {
		case Hourly:
			return "global_stats_players_hourly"
		case Daily:
			return "global_stats_players_daily"
		default:
			return "global_stats_players"
		}
	default:
		switch duration {
		case Hourly:
			return "local_stats_players_hourly"
		case Daily:
			return "local_stats_players_daily"
		default:
			return "local_stats_players"
		}
	}
}

//
// func (database *pgStore) GetGlobalTF2Stats(ctx context.Context, duration StatDuration) ([]state.GlobalTF2StatsSnapshot, error) {
//	table := statDurationTable(Global, duration)
//	if table == "" {
//		return nil, errors.New("Unsupported stat duration")
//	}
//	qb := sb.Select(fp.Prepend(globalStatColumns, "stat_id")...).
//		From(table).
//		OrderBy("created_on desc")
//	switch duration {
//	case Hourly:
//		qb = qb.Limit(24 * 7)
//	}
//	query, args, errQuery := qb.ToSql()
//	if errQuery != nil {
//		return nil, Err(errQuery)
//	}
//	return fetchGlobalTF2Snapshots(ctx, database, query, args)
// }

func (db *Store) GetLocalTF2Stats(ctx context.Context, duration StatDuration) ([]LocalTF2StatsSnapshot, error) {
	table := statDurationTable(Local, duration)
	if table == "" {
		return nil, errors.New("Unsupported stat duration")
	}

	builder := db.sb.
		Select(fp.Prepend(localStatColumns, "stat_id")...).
		From(table).
		OrderBy("created_on desc")
	if duration == Hourly {
		builder = builder.Limit(24 * 7)
	}

	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return nil, Err(errQuery)
	}

	return db.fetchLocalTF2Snapshots(ctx, query, args)
}

func (db *Store) BuildLocalTF2Stats(ctx context.Context) error {
	maxDate := currentHourlyTime()

	query, args, errQuery := db.sb.
		Select(fp.Prepend(localStatColumns, "stat_id")...).
		From(statDurationTable(Local, Live)).
		Where(sq.Lt{"created_on": maxDate}). // Ignore any results until a full hour has passed
		OrderBy("created_on").
		ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}

	stats, errStats := db.fetchLocalTF2Snapshots(ctx, query, args)
	if errStats != nil {
		return errStats
	}

	if len(stats) == 0 {
		return nil
	}

	var ( //nolint:prealloc
		hourlySums          []LocalTF2StatsSnapshot
		curSums             *LocalTF2StatsSnapshot
		tempPlayers         []int
		tempCapacityFull    []int
		tempCapacityEmpty   []int
		tempCapacityPartial []int
		tempMapTypes        = map[string][]int{}
		tempRegions         = map[string][]int{}
		tempServers         = map[string][]int{}
		curIdx              = 0
	)

	for _, stat := range stats {
		// Group & sum hourly as the minimum granularity
		curTime, timeIdx := HourlyIndex(stat.CreatedOn)
		if curIdx == 0 {
			curIdx = timeIdx
		} else if curIdx != timeIdx && curSums != nil {
			// If the hour index changed, flush the current results out
			sumStat := NewLocalTF2Stats()
			sumStat.CreatedOn = curTime
			sumStat.Players = fp.Avg(tempPlayers)
			sumStat.CapacityFull = fp.Avg(tempCapacityFull)
			sumStat.CapacityEmpty = fp.Avg(tempCapacityEmpty)
			sumStat.CapacityPartial = fp.Avg(tempCapacityPartial)
			for k := range tempMapTypes {
				sumStat.MapTypes[k] = fp.Avg(tempMapTypes[k])
			}
			for k := range tempRegions {
				sumStat.Regions[k] = fp.Avg(tempRegions[k])
			}
			for k := range tempServers {
				sumStat.Servers[k] = fp.Avg(tempServers[k])
			}
			hourlySums = append(hourlySums, sumStat)
			curSums = nil
			curIdx = timeIdx
			tempPlayers = nil
			tempCapacityFull = nil
			tempCapacityEmpty = nil
			tempCapacityPartial = nil
			tempRegions = map[string][]int{}
			tempMapTypes = map[string][]int{}
			tempServers = map[string][]int{}
		}

		if curSums == nil {
			curSums = &LocalTF2StatsSnapshot{CreatedOn: curTime}
		}

		tempPlayers = append(tempPlayers, stat.Players)
		tempCapacityFull = append(tempCapacityFull, stat.CapacityFull)
		tempCapacityEmpty = append(tempCapacityEmpty, stat.CapacityEmpty)
		tempCapacityPartial = append(tempCapacityPartial, stat.CapacityPartial)

		for mapKey, count := range stat.MapTypes {
			_, found := tempMapTypes[mapKey]
			if !found {
				tempMapTypes[mapKey] = []int{}
			}

			tempMapTypes[mapKey] = append(tempMapTypes[mapKey], count)
		}

		for regionKey, count := range stat.Regions {
			_, found := tempRegions[regionKey]
			if !found {
				tempRegions[regionKey] = []int{}
			}

			tempRegions[regionKey] = append(tempRegions[regionKey], count)
		}

		for serverKey, count := range stat.Servers {
			_, found := tempServers[serverKey]
			if !found {
				tempServers[serverKey] = []int{}
			}

			tempServers[serverKey] = append(tempServers[serverKey], count)
		}
	}

	for _, hourly := range hourlySums {
		if errSave := db.SaveLocalTF2Stats(ctx, Hourly, hourly); errSave != nil {
			if errors.Is(errSave, ErrDuplicate) {
				continue
			}

			return errSave
		}
	}

	statIds := make([]int64, len(stats))
	for index, s := range stats {
		statIds[index] = s.StatID
	}

	// Delete old entries
	delQuery, delArgs, delQueryErr := db.sb.
		Delete(statDurationTable(Local, Live)).
		Where(sq.Eq{"stat_id": statIds}).
		ToSql()
	if delQueryErr != nil {
		return Err(delQueryErr)
	}

	return db.Exec(ctx, delQuery, delArgs...)
}
