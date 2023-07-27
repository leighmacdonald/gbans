package store

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/config"
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
		CreatedOn: config.Now(),
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

func (db *Store) MatchSave(ctx context.Context, match *logparse.Match) error {
	const query = `INSERT INTO match (server_id, map, created_on, title) VALUES ($1, $2, $3, $4) RETURNING match_id`

	for _, p := range match.PlayerSums {
		var player Person
		if errPlayer := db.GetOrCreatePersonBySteamID(ctx, p.SteamID, &player); errPlayer != nil {
			return errors.Wrapf(errPlayer, "Failed to create person")
		}
	}

	if errMatch := db.QueryRow(ctx, query, match.ServerID, match.MapName, match.CreatedOn, match.Title).
		Scan(&match.MatchID); errMatch != nil {
		return errors.Wrapf(errMatch, "Failed to setup match")
	}

	const playerQuery = `INSERT INTO match_player (
			match_id, steam_id, team, 
			time_start, time_end, kills, assists, deaths, dominations, dominated, 
			revenges, damage, damage_taken, healing, healing_taken, health_packs, 
			backstabs, headshots, airshots, captures, shots, extinguishes, 
			hits, buildings, buildings_destroyed) 
		VALUES (
			$1,  $2, $3, 
		    $4,  $5, $6, $7, $8, $9, $10, 
		    $11, $12, $13, $14, $15, $16, 
		    $17, $18, $19, $20, $21, $22, 
		    $23, $24, $25
		) RETURNING match_player_id`

	for _, playerSum := range match.PlayerSums {
		endTime := &match.CreatedOn
		if playerSum.TimeEnd != nil {
			// Use match end time
			endTime = playerSum.TimeEnd
		}

		if errPlayerExec := db.QueryRow(ctx, playerQuery, match.MatchID, playerSum.SteamID.Int64(), playerSum.Team, playerSum.TimeStart, endTime, playerSum.Kills, playerSum.Assists, playerSum.Deaths, playerSum.Dominations, playerSum.Dominated, playerSum.Revenges, playerSum.Damage, playerSum.DamageTaken, playerSum.Healing, playerSum.HealingTaken, playerSum.HealthPacks, playerSum.BackStabs, playerSum.HeadShots, playerSum.AirShots, playerSum.Captures, playerSum.Shots, playerSum.Extinguishes, playerSum.Hits, playerSum.BuildingDestroyed, playerSum.BuildingDestroyed).Scan(&playerSum.MatchPlayerSumID); errPlayerExec != nil {
			return errors.Wrapf(errPlayerExec, "Failed to write player sum")
		}
	}

	const medicQuery = `INSERT INTO match_medic (
            match_id, steam_id, healing, charges, drops, avg_time_to_build, avg_time_before_use, 
            near_full_charge_death, avg_uber_length, death_after_charge, major_adv_lost, biggest_adv_lost) 
            VALUES ($1, $2, $3, $4, $5,$6, $7, $8, $9, $10,$11, $12) RETURNING match_medic_id`

	for _, medicSum := range match.MedicSums {
		charges := 0
		for _, mg := range medicSum.Charges {
			charges += mg
		}

		if errMedExec := db.QueryRow(ctx, medicQuery, match.MatchID, medicSum.SteamID.Int64(), medicSum.Healing, charges, medicSum.Drops, medicSum.AvgTimeToBuild, medicSum.AvgTimeBeforeUse, medicSum.NearFullChargeDeath, medicSum.AvgUberLength, medicSum.DeathAfterCharge, medicSum.MajorAdvLost, medicSum.BiggestAdvLost).Scan(&medicSum.MatchMedicID); errMedExec != nil {
			return errors.Wrapf(errMedExec, "Failed to write medic sum")
		}
	}

	const teamQuery = `INSERT INTO match_team (
		match_id, team, kills, damage, charges, drops, caps, mid_fights
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	RETURNING match_team_id`
	// FIXME team value unset
	for i, s := range match.TeamSums {
		if errTeamExec := db.QueryRow(ctx, teamQuery, match.MatchID, i+1, s.Kills, s.Damage, s.Charges, s.Drops, s.Caps, s.MidFights).Scan(&s.MatchTeamID); errTeamExec != nil {
			return errors.Wrapf(errTeamExec, "Failed to write team sum")
		}
	}

	return nil
}

type MatchesQueryOpts struct {
	QueryFilter
	SteamID   steamid.SID64 `json:"steam_id"`
	ServerID  int           `json:"server_id"`
	Map       string        `json:"map"`
	TimeStart *time.Time    `json:"time_start,omitempty"`
	TimeEnd   *time.Time    `json:"time_end,omitempty"`
}

func (db *Store) Matches(ctx context.Context, opts MatchesQueryOpts) (logparse.MatchSummaryCollection, error) {
	builder := db.sb.
		Select("m.match_id", "m.server_id", "m.map", "m.created_on", "COALESCE(sum(mp.kills), 0)", "COALESCE(sum(mp.assists), 0)", "COALESCE(sum(mp.damage), 0)", "COALESCE(sum(mp.healing), 0)", "COALESCE(sum(mp.airshots), 0)").
		From("match m").
		LeftJoin("match_player mp on m.match_id = mp.match_id").
		GroupBy("m.match_id")
	if opts.Map != "" {
		builder = builder.Where(sq.Eq{"m.map_name": opts.Map})
	}

	if opts.SteamID.Valid() {
		builder = builder.Where(sq.Eq{"mp.steam_id": opts.SteamID.Int64()})
	}

	if opts.Desc {
		builder = builder.OrderBy("m.match_id DESC")
	} else {
		builder = builder.OrderBy("m.match_id ASC")
	}

	if opts.Limit > 0 {
		builder = builder.Limit(opts.Limit)
	}

	query, args, errQueryArgs := builder.ToSql()
	if errQueryArgs != nil {
		return nil, errors.Wrapf(errQueryArgs, "Failed to build query")
	}

	rows, errQuery := db.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, errors.Wrapf(errQuery, "Failed to query matches")
	}

	defer rows.Close()

	var matches logparse.MatchSummaryCollection

	for rows.Next() {
		var m logparse.MatchSummary
		if errScan := rows.Scan(&m.MatchID, &m.ServerID, &m.MapName, &m.CreatedOn /*&m.PlayerCount,*/, &m.Kills, &m.Assists, &m.Damage, &m.Healing, &m.Airshots); errScan != nil {
			return nil, errors.Wrapf(errScan, "Failed to scan match row")
		}

		matches = append(matches, &m)
	}

	return matches, nil
}

func (db *Store) MatchGetByID(ctx context.Context, matchID int) (*logparse.Match, error) {
	const query = `SELECT server_id, map, title, created_on  FROM match WHERE match_id = $1`

	match := logparse.NewMatch(db.log, -1, "")

	match.MatchID = matchID

	if errMatch := db.
		QueryRow(ctx, query, matchID).
		Scan(&match.ServerID, &match.MapName, &match.Title, &match.CreatedOn); errMatch != nil {
		return nil, errors.Wrapf(errMatch, "Failed to load root match")
	}

	const queryPlayer = `
		SELECT 
		    match_player_id, steam_id, team, time_start, time_end, kills, assists,
       		deaths, dominations, dominated, revenges, damage, damage_taken, healing, healing_taken, health_packs, 
       		backstabs, headshots, airshots, captures, shots, extinguishes, hits, buildings, 
       		buildings_destroyed, (kills::real/deaths::real), ((kills::real+assists::real)/deaths::real)
		FROM 
		    match_player
		WHERE 
		    match_id = $1`

	playerRows, errPlayer := db.Query(ctx, queryPlayer, matchID)
	if errPlayer != nil {
		return nil, errors.Wrapf(errPlayer, "Failed to query match players")
	}

	defer playerRows.Close()

	for playerRows.Next() {
		var (
			mpSum   = logparse.MatchPlayerSum{MatchPlayerSumID: matchID}
			steamID int64
		)

		if errRow := playerRows.Scan(&mpSum.MatchPlayerSumID, &steamID, &mpSum.Team, &mpSum.TimeStart, &mpSum.TimeEnd, &mpSum.Kills, &mpSum.Assists, &mpSum.Deaths, &mpSum.Dominations, &mpSum.Dominated, &mpSum.Revenges, &mpSum.Damage, &mpSum.DamageTaken, &mpSum.Healing, &mpSum.HealingTaken, &mpSum.HealthPacks, &mpSum.BackStabs, &mpSum.HeadShots, &mpSum.AirShots, &mpSum.Captures, &mpSum.Shots, &mpSum.Extinguishes, &mpSum.Hits, &mpSum.BuildingBuilt, &mpSum.BuildingDestroyed, &mpSum.KDRatio, &mpSum.KADRatio); errRow != nil {
			return nil, errors.Wrapf(errPlayer, "Failed to scan match players")
		}

		mpSum.SteamID = steamid.New(steamID)

		match.PlayerSums = append(match.PlayerSums, &mpSum)
	}

	const qMed = `
		SELECT 
		    match_medic_id, steam_id, healing, charges, drops, avg_time_to_build, 
       		avg_time_before_use, near_full_charge_death, avg_uber_length, death_after_charge, 
       		major_adv_lost, biggest_adv_lost 
		FROM
		    match_medic
		WHERE 
		    match_id = $1`

	medicRows, errMedQuery := db.Query(ctx, qMed, matchID)
	if errMedQuery != nil && !errors.Is(errMedQuery, ErrNoResult) {
		return nil, errors.Wrapf(errMedQuery, "Failed to query match medics")
	}

	defer medicRows.Close()

	for medicRows.Next() {
		var steamID int64

		medicSum := logparse.MatchMedicSum{MatchID: matchID, Charges: map[logparse.MedigunType]int{
			logparse.Uber:       0,
			logparse.Kritzkrieg: 0,
			logparse.Vaccinator: 0,
			logparse.QuickFix:   0,
		}}

		charges := 0

		if errRow := medicRows.Scan(&medicSum.MatchMedicID, &steamID, &medicSum.Healing, &charges, &medicSum.Drops, &medicSum.AvgTimeToBuild, &medicSum.AvgTimeBeforeUse, &medicSum.NearFullChargeDeath, &medicSum.AvgUberLength, &medicSum.DeathAfterCharge, &medicSum.MajorAdvLost, &medicSum.BiggestAdvLost); errRow != nil {
			return nil, errors.Wrapf(errMedQuery, "Failed to scan match medics")
		}

		medicSum.SteamID = steamid.New(steamID)

		// FIXME all charges are counted as uber for now
		medicSum.Charges[logparse.Uber] = charges
		match.MedicSums = append(match.MedicSums, &medicSum)
	}

	const qTeam = `
		SELECT 
		    match_team_id, team, kills, damage, charges, drops, caps, mid_fights 
		FROM 
		    match_team 
		WHERE 
		    match_id = $1`

	teamRows, errTeamQuery := db.Query(ctx, qTeam, matchID)
	if errTeamQuery != nil && !errors.Is(errTeamQuery, ErrNoResult) {
		return nil, errors.Wrapf(errMedQuery, "Failed to query match medics")
	}

	defer teamRows.Close()

	for teamRows.Next() {
		ts := logparse.MatchTeamSum{MatchID: matchID}
		if errRow := teamRows.Scan(&ts.MatchTeamID, &ts.Team, &ts.Kills, &ts.Damage, &ts.Charges, &ts.Drops, &ts.Caps, &ts.MidFights); errRow != nil {
			return nil, errors.Wrapf(errRow, "Failed to scan match medics")
		}

		match.TeamSums = append(match.TeamSums, &ts)
	}

	// var ids steamid.Collection
	// for _, p := range match.PlayerSums {
	// 	ids = append(ids, p.SteamID)
	// }
	// players, errPlayers := database.GetPeopleBySteamID(ctx, ids)
	// if errPlayers != nil {
	// 	return nil, errors.Wrapf(errPlayers, "Failed to load players")
	// }
	// match.Players = players
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
	now := config.Now()
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
