package store

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
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
	for weapon, name := range logparse.NewWeaponParser().NameMap() {
		var newWeapon Weapon
		if errWeapon := db.GetWeaponByKey(ctx, weapon, &newWeapon); errWeapon != nil {
			if !errors.Is(errWeapon, ErrNoResult) {
				return errWeapon
			}

			newWeapon.Key = weapon
			newWeapon.Name = name

			if errSave := db.SaveWeapon(ctx, &newWeapon); errSave != nil {
				return Err(errSave)
			}
		}

		db.weaponMap.Set(weapon, newWeapon.WeaponID)
	}

	return nil
}

type Weapon struct {
	WeaponID int             `json:"weapon_id"`
	Key      logparse.Weapon `json:"key"`
	Name     string          `json:"name"`
}

func (db *Store) GetWeaponByKey(ctx context.Context, key logparse.Weapon, weapon *Weapon) error {
	const q = `SELECT weapon_id, key, name FROM weapon WHERE key = $1`

	if errQuery := db.QueryRow(ctx, q, key).Scan(&weapon.WeaponID, &weapon.Key, &weapon.Name); errQuery != nil {
		return Err(errQuery)
	}

	return nil
}

func (db *Store) GetWeaponByID(ctx context.Context, weaponID int, weapon *Weapon) error {
	const q = `SELECT weapon_id, key, name FROM weapon WHERE weapon_id = $1`

	if errQuery := db.QueryRow(ctx, q, weaponID).Scan(&weapon.WeaponID, &weapon.Key, &weapon.Name); errQuery != nil {
		return Err(errQuery)
	}

	return nil
}

func (db *Store) SaveWeapon(ctx context.Context, weapon *Weapon) error {
	if weapon.WeaponID > 0 {
		updateQuery, updateArgs, errUpdateQuery := db.sb.
			Update("weapon").
			Set("key", weapon.Key).
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

	const wq = `INSERT INTO weapon (key, name) VALUES ($1, $2) RETURNING weapon_id`

	if errSave := db.
		QueryRow(ctx, wq, weapon.Key, weapon.Name).
		Scan(&weapon.WeaponID); errSave != nil {
		return errors.Wrap(errSave, "Failed to insert weapon")
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

type MapUseDetail struct {
	Map      string  `json:"map"`
	Playtime int64   `json:"playtime"`
	Percent  float64 `json:"percent"`
}

func (db *Store) GetMapUsageStats(ctx context.Context) ([]MapUseDetail, error) {
	const query = `SELECT m.map, m.playtime, (m.playtime::float / s.total::float) * 100 percent
		FROM (
			SELECT SUM(extract('epoch' from m.time_end - m.time_start)) as playtime, m.map FROM match m
			    LEFT JOIN public.match_player mp on m.match_id = mp.match_id 
			GROUP BY m.map
		) m CROSS JOIN (
			SELECT SUM(extract('epoch' from mt.time_end - mt.time_start)) total FROM match mt
			LEFT JOIN public.match_player mpt on mt.match_id = mpt.match_id
		) s ORDER BY percent DESC`

	var details []MapUseDetail

	rows, errQuery := db.Query(ctx, query)
	if errQuery != nil {
		return nil, Err(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			mud     MapUseDetail
			seconds int64
		)

		if errScan := rows.Scan(&mud.Map, &seconds, &mud.Percent); errScan != nil {
			return nil, Err(errScan)
		}

		mud.Playtime = seconds

		details = append(details, mud)
	}

	if rows.Err() != nil {
		return nil, errors.Wrap(rows.Err(), "rows returned error")
	}

	return details, nil
}

type TopChatterResult struct {
	Name    string
	SteamID steamid.SID64
	Count   int
}

func (db *Store) TopChatters(ctx context.Context, count int) ([]TopChatterResult, error) {
	const query = `SELECT
    	p.personaname, p.steam_id, count(person_message_id) as total
		FROM person_messages m
		LEFT JOIN public.person p on m.steam_id = p.steam_id
		GROUP BY p.steam_id
		ORDER BY total DESC
		LIMIT $1`

	rows, errRows := db.Query(ctx, query, count)
	if errRows != nil {
		return nil, Err(errRows)
	}

	defer rows.Close()

	var results []TopChatterResult

	for rows.Next() {
		var (
			tcr     TopChatterResult
			steamID int64
		)

		if errScan := rows.Scan(&tcr.Name, &steamID, &tcr.Count); errScan != nil {
			return nil, Err(errScan)
		}

		tcr.SteamID = steamid.New(steamID)
		results = append(results, tcr)
	}

	return results, nil
}

type RankedResult struct {
	Rank int `json:"rank"`
}

type WeaponsOverallResult struct {
	Weapon
	RankedResult
	Kills        int64   `json:"kills"`
	KillsPct     float64 `json:"kills_pct"`
	Damage       int64   `json:"damage"`
	DamagePct    float64 `json:"damage_pct"`
	Headshots    int64   `json:"headshots"`
	HeadshotsPct float64 `json:"headshots_pct"`
	Airshots     int64   `json:"airshots"`
	AirshotsPct  float64 `json:"airshots_pct"`
	Backstabs    int64   `json:"backstabs"`
	BackstabsPct float64 `json:"backstabs_pct"`
	Shots        int64   `json:"shots"`
	ShotsPct     float64 `json:"shots_pct"`
	Hits         int64   `json:"hits"`
	HitsPct      float64 `json:"hits_pct"`
}

func (db *Store) WeaponsOverall(ctx context.Context) ([]WeaponsOverallResult, error) {
	const query = `
		SELECT 
		    s.weapon_id, s.name, s.key, 
		    s.kills, case t.kills_total WHEN 0 THEN 0 ELSE (s.kills::float / t.kills_total::float) * 100 END kills_pct,
		    s.hs,  case t.headshots_total WHEN 0 THEN 0 ELSE (s.hs::float / t.headshots_total::float) * 100 END hs_pct,
		    s.airshots, case t.airshots_total WHEN 0 THEN 0 ELSE (s.airshots::float / t.airshots_total::float) * 100 END airshots_pct,
		    s.bs, case t.backstabs_total WHEN 0 THEN 0 ELSE (s.bs::float / t.backstabs_total::float) * 100 END  bs_pct,
			s.shots,  case t.shots_total WHEN 0 THEN 0 ELSE (s.shots::float / t.shots_total::float) * 100 END shots_pct,
			s.hits, case t.hits_total WHEN 0 THEN 0 ELSE (s.hits::float / t.hits_total::float) * 100 END hits_pct,
			s.damage, case t.damage_total WHEN 0 THEN 0 ELSE (s.damage::float / t.damage_total::float) * 100 END damage_pct
		FROM (
    		SELECT
    		    w.weapon_id, w.key, w.name,
             	SUM(mw.kills)  as kills,
             	SUM(mw.damage)  as damage,
             	SUM(mw.shots) as shots,
             	SUM(mw.hits) as hits,
             	SUM(headshots) as hs,
             	SUM(airshots)  as airshots,
             	SUM(backstabs) as bs
      		FROM match_weapon mw
    		LEFT JOIN public.weapon w on w.weapon_id = mw.weapon_id
      		GROUP BY w.weapon_id
		) s CROSS JOIN (
			SELECT 
			    SUM(mw.kills) as kills_total, 
			    SUM(mw.damage) as damage_total,
			    SUM(mw.shots) as shots_total,
			    SUM(mw.hits) as hits_total,
			    SUM(mw.airshots) as airshots_total,
			    SUM(mw.backstabs) as backstabs_total,
			    SUM(mw.headshots) as headshots_total
            FROM match_weapon mw
        ) t ;`

	rows, errQuery := db.Query(ctx, query)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()

	var results []WeaponsOverallResult

	for rows.Next() {
		var wor WeaponsOverallResult
		if errScan := rows.
			Scan(&wor.WeaponID, &wor.Name, &wor.Key,
				&wor.Kills, &wor.KillsPct,
				&wor.Headshots, &wor.HeadshotsPct,
				&wor.Airshots, &wor.AirshotsPct,
				&wor.Backstabs, &wor.BackstabsPct,
				&wor.Shots, &wor.ShotsPct,
				&wor.Hits, &wor.HitsPct,
				&wor.Damage, &wor.DamagePct); errScan != nil {
			return nil, Err(errScan)
		}

		results = append(results, wor)
	}

	return results, nil
}

type PlayerWeaponResult struct {
	Rank               int           `json:"rank"`
	SteamID            steamid.SID64 `json:"steam_id"`
	Personaname        string        `json:"personaname"`
	AvatarHash         string        `json:"avatar_hash"`
	KA                 int64         `json:"ka"`
	Kills              int64         `json:"kills"`
	Assists            int64         `json:"assists"`
	Deaths             int64         `json:"deaths"`
	KD                 float64       `json:"kd"`
	KAD                float64       `json:"kad"`
	DPM                float64       `json:"dpm"`
	Shots              int64         `json:"shots"`
	Hits               int64         `json:"hits"`
	Accuracy           float64       `json:"accuracy"`
	Airshots           int64         `json:"airshots"`
	Backstabs          int64         `json:"backstabs"`
	Headshots          int64         `json:"headshots"`
	Playtime           int64         `json:"playtime"`
	Dominations        int64         `json:"dominations"`
	Dominated          int64         `json:"dominated"`
	Revenges           int64         `json:"revenges"`
	Damage             int64         `json:"damage"`
	DamageTaken        int64         `json:"damage_taken"`
	Captures           int64         `json:"captures"`
	CapturesBlocked    int64         `json:"captures_blocked"`
	BuildingsDestroyed int64         `json:"buildings_destroyed"`
}

func (db *Store) WeaponsOverallTopPlayers(ctx context.Context, weaponID int) ([]PlayerWeaponResult, error) {
	const query = `
		SELECT row_number() over (order by SUM(mw.kills) desc nulls last) as rank,
		       p.steam_id, p.personaname, p.avatarhash,
			   SUM(mw.kills) as kills, sum(mw.damage) as damage,
			   sum(mw.shots) as shots, sum(mw.hits) as hits,
			   sum(mw.backstabs) as backstabs,
			   sum(mw.headshots) as headshots,
			   sum(mw.airshots) as airshots
		FROM match_weapon mw
		LEFT JOIN weapon w on w.weapon_id = mw.weapon_id
		LEFT JOIN match_player mp on mp.match_player_id = mw.match_player_id
		LEFT JOIN person p on mp.steam_id = p.steam_id
		WHERE w.weapon_id = $1
		GROUP BY p.steam_id, w.weapon_id ORDER BY kills DESC
		LIMIT 250`

	rows, errQuery := db.Query(ctx, query, weaponID)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()

	var results []PlayerWeaponResult

	for rows.Next() {
		var (
			pwr   PlayerWeaponResult
			sid64 int64
		)

		if errScan := rows.
			Scan(&pwr.Rank, &sid64, &pwr.Personaname, &pwr.AvatarHash,
				&pwr.Kills, &pwr.Damage,
				&pwr.Shots, &pwr.Hits,
				&pwr.Backstabs, &pwr.Headshots,
				&pwr.Airshots); errScan != nil {
			return nil, Err(errScan)
		}

		pwr.SteamID = steamid.New(sid64)
		results = append(results, pwr)
	}

	return results, nil
}

func (db *Store) WeaponsOverallByPlayer(ctx context.Context, steamID steamid.SID64) ([]WeaponsOverallResult, error) {
	const query = `
		SELECT
			row_number() over (order by s.kills desc nulls last) as rank,
			s.weapon_id, s.name, s.key,
			s.kills,    case t.kills_total WHEN 0 THEN 0 ELSE (s.kills::float / t.kills_total::float) * 100 END kills_pct,
			s.hs,       case t.headshots_total WHEN 0 THEN 0 ELSE (s.hs::float / t.headshots_total::float) * 100 END hs_pct,
			s.airshots, case t.airshots_total WHEN 0 THEN 0 ELSE (s.airshots::float / t.airshots_total::float) * 100 END airshots_pct,
			s.bs,	    case t.backstabs_total WHEN 0 THEN 0 ELSE (s.bs::float / t.backstabs_total::float) * 100 END  bs_pct,
			s.shots,    case t.shots_total WHEN 0 THEN 0 ELSE (s.shots::float / t.shots_total::float) * 100 END shots_pct,
			s.hits,     case t.hits_total WHEN 0 THEN 0 ELSE (s.hits::float / t.hits_total::float) * 100 END hits_pct,
			s.damage,   case t.damage_total WHEN 0 THEN 0 ELSE (s.damage::float / t.damage_total::float) * 100 END damage_pct
		FROM (
			 SELECT
				 w.weapon_id, w.key, w.name,
				 SUM(mw.kills)  as kills,
				 SUM(mw.damage)  as damage,
				 SUM(mw.shots) as shots,
				 SUM(mw.hits) as hits,
				 SUM(headshots) as hs,
				 SUM(airshots)  as airshots,
				 SUM(backstabs) as bs
			 FROM match_weapon mw
			 LEFT JOIN weapon w on w.weapon_id = mw.weapon_id
			 LEFT JOIN match_player mp on mw.match_player_id = mp.match_player_id
			 WHERE mp.steam_id = $1
			 GROUP BY w.weapon_id
			 ORDER BY kills DESC
		) s
		CROSS JOIN (
			SELECT
				SUM(mw.kills) as kills_total,
				SUM(mw.damage) as damage_total,
				SUM(mw.shots) as shots_total,
				SUM(mw.hits) as hits_total,
				SUM(mw.airshots) as airshots_total,
				SUM(mw.backstabs) as backstabs_total,
				SUM(mw.headshots) as headshots_total
			FROM match_weapon mw
			LEFT JOIN match_player mp on mw.match_player_id = mp.match_player_id
			WHERE mp.steam_id = $1
		) t`

	rows, errQuery := db.Query(ctx, query, steamID.Int64())
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()

	var results []WeaponsOverallResult

	for rows.Next() {
		var wor WeaponsOverallResult
		if errScan := rows.
			Scan(&wor.Rank,
				&wor.WeaponID, &wor.Name, &wor.Key,
				&wor.Kills, &wor.KillsPct,
				&wor.Headshots, &wor.HeadshotsPct,
				&wor.Airshots, &wor.AirshotsPct,
				&wor.Backstabs, &wor.BackstabsPct,
				&wor.Shots, &wor.ShotsPct,
				&wor.Hits, &wor.HitsPct,
				&wor.Damage, &wor.DamagePct); errScan != nil {
			return nil, Err(errScan)
		}

		results = append(results, wor)
	}

	return results, nil
}

func (db *Store) PlayersOverallByKills(ctx context.Context, count int) ([]PlayerWeaponResult, error) {
	const query = `
		SELECT row_number() over (order by c.assists + w.kills desc nulls last) as rank,
			   p.personaname,
			   p.steam_id,
			   p.avatarhash,
			   w.kills + c.assists as ka,
			   w.kills,
			   c.assists,
			   c.deaths,
			   case c.deaths WHEN 0 THEN -1 ELSE (w.kills::float / c.deaths::float) END kd,
			   case c.deaths WHEN 0 THEN -1 ELSE ((c.assists::float + w.kills::float) / c.deaths::float) END kad,
			   c.damage::float / (c.playtime::float / 60) as dpm,
			   w.shots,
			   w.hits,
			   case w.shots WHEN 0 THEN -1 ELSE (w.hits::float / w.shots::float) * 100 END acc,
			   w.airshots,
			   w.backstabs,
			   w.headshots,
			   c.playtime,
			   c.dominations,
			   c.dominated,
			   c.revenges,
			   c.damage,
			   c.damage_taken,
			   c.captures,
			   c.captures_blocked,
			   c.buildings_destroyed
		FROM person p
		LEFT JOIN (
			SELECT mp.steam_id,
				   sum(mw.kills)     as kills,
				   sum(mw.shots)     as shots,
				   sum(mw.hits)      as hits,
				   sum(mw.airshots)  as airshots,
				   sum(mw.backstabs) as backstabs,
				   sum(mw.headshots) as headshots
			FROM  match_player mp
			LEFT JOIN match_weapon mw on mp.match_player_id = mw.match_player_id
			GROUP BY mp.steam_id
		) w ON w.steam_id = p.steam_id
		LEFT JOIN (
			SELECT mp.steam_id,
				   SUM(mpc.assists) as assists,
				   sum(mpc.deaths)              as deaths,
				   sum(mpc.playtime)            as playtime,
				   sum(mpc.dominations)         as dominations,
				   sum(mpc.dominated)           as dominated,
				   sum(mpc.revenges)            as revenges,
				   sum(mpc.damage)        		as damage,
				   sum(mpc.damage_taken)        as damage_taken,
				   sum(mpc.healing_taken)       as healing_taken,
				   sum(mpc.captures)            as captures,
				   sum(mpc.captures_blocked)    as captures_blocked,
				   sum(mpc.buildings_destroyed) as buildings_destroyed
			FROM match_player mp
			LEFT JOIN match_player_class mpc on mp.match_player_id = mpc.match_player_id
			GROUP BY mp.steam_id
		) c ON c.steam_id = p.steam_id
		ORDER BY rank
		LIMIT $1`

	rows, errQuery := db.Query(ctx, query, count)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()

	var results []PlayerWeaponResult

	for rows.Next() {
		var (
			wor   PlayerWeaponResult
			sid64 int64
		)

		if errScan := rows.
			Scan(&wor.Rank,
				&wor.Personaname, &sid64, &wor.AvatarHash,
				&wor.KA, &wor.Kills, &wor.Assists, &wor.Deaths, &wor.KD,
				&wor.KAD, &wor.DPM, &wor.Shots, &wor.Hits, &wor.Accuracy,
				&wor.Airshots, &wor.Backstabs, &wor.Headshots, &wor.Playtime, &wor.Dominations,
				&wor.Dominated, &wor.Revenges, &wor.Damage, &wor.DamageTaken, &wor.Captures,
				&wor.CapturesBlocked, &wor.BuildingsDestroyed,
			); errScan != nil {
			return nil, Err(errScan)
		}

		wor.SteamID = steamid.New(sid64)
		results = append(results, wor)
	}

	return results, nil
}

type HealingOverallResult struct {
	RankedResult
	SteamID             steamid.SID64 `json:"steam_id"`
	Personaname         string        `json:"personaname"`
	AvatarHash          string        `json:"avatar_hash"`
	Healing             int           `json:"healing"`
	Drops               int           `json:"drops"`
	NearFullChargeDeath int           `json:"near_full_charge_death"`
	ChargesUber         int           `json:"charges_uber"`
	ChargesKritz        int           `json:"charges_kritz"`
	ChargesVacc         int           `json:"charges_vacc"`
	ChargesQuickfix     int           `json:"charges_quickfix"`
	AvgUberLength       float32       `json:"avg_uber_length"`
	MajorAdvLost        int           `json:"major_adv_lost"`
	BiggestAdvLost      int           `json:"biggest_adv_lost"`
	Extinguishes        int64         `json:"extinguishes"`
	HealthPacks         int64         `json:"health_packs"`
	Assists             int64         `json:"assists"`
	Deaths              int64         `json:"deaths"`
	HPM                 float64       `json:"hpm"`
	KA                  float64       `json:"ka"`
	KAD                 float64       `json:"kad"`
	Playtime            int64         `json:"playtime"`
	Dominations         int64         `json:"dominations"`
	Dominated           int64         `json:"dominated"`
	Revenges            int64         `json:"revenges"`
	DamageTaken         int64         `json:"damage_taken"`
	DTM                 float64       `json:"dtm"`
	Wins                int64         `json:"wins"`
	Matches             int64         `json:"matches"`
	WinRate             float64       `json:"win_rate"`
}

func (db *Store) HealersOverallByHealing(ctx context.Context, count int) ([]HealingOverallResult, error) {
	const query = `
		SELECT
            row_number() over (order by h.healing desc nulls last) as rank,
            p.steam_id,
            p.personaname,
            p.avatarhash,
            coalesce(h.healing, 0) as healing,
            coalesce(h.drops, 0) as drops,
            coalesce(h.near_full_charge_death, 0) as near_full_charge_death,
            coalesce(h.avg_uber_length, 0) as avg_uber_length,
            coalesce(h.major_adv_lost, 0) as major_adv_lost,
            coalesce(h.biggest_adv_lost, 0) as biggest_adv_lost,
            coalesce(h.charge_uber, 0) as charge_uber,
            coalesce(h.charge_kritz, 0) as charge_kritz,
            coalesce(h.charge_vacc, 0) as charge_vacc,
            coalesce(h.charge_quickfix, 0) as charge_quickfix,
            coalesce(h.extinguishes, 0) as extinguishes,
            coalesce(h.health_packs, 0) as health_packs,
            coalesce(c.assists, 0) as assists,
            coalesce(c.kills, 0) + coalesce(c.assists, 0)  as ka,
            coalesce(c.deaths, 0) as deaths,
            case c.playtime WHEN 0 THEN 0 ELSE h.healing::float / (c.playtime::float / 60) END as hpm,
            case c.deaths WHEN 0 THEN -1 ELSE ((c.assists::float + c.kills::float) / c.deaths::float) END kad,
            coalesce(c.playtime, 0) as playtime,
            coalesce(c.dominations, 0) as dominations,
            coalesce(c.dominated, 0) as dominated,
            coalesce(c.revenges, 0) as revenges,
            coalesce(c.damage_taken, 0) as damage_taken,
            case c.playtime WHEN 0 THEN 0 ELSE c.damage_taken::float / (c.playtime::float / 60) END as dtm,
            coalesce(mx.wins, 0) as wins,
            coalesce(mx.matches, 0) as matches,
            case mx.matches WHEN 0 THEN -1 ELSE (mx.wins::float / mx.matches::float) * 100 END as win_rate
		FROM person p
				 LEFT JOIN (SELECT mp.steam_id,
								   sum(mm.healing)                as healing,
								   sum(mm.drops)                  as drops,
								   sum(mm.near_full_charge_death) as near_full_charge_death,
								   sum(mm.avg_uber_length)        as avg_uber_length,
								   sum(mm.major_adv_lost)         as major_adv_lost,
								   sum(mm.biggest_adv_lost)       as biggest_adv_lost,
								   sum(mm.charge_uber)            as charge_uber,
								   sum(mm.charge_kritz)           as charge_kritz,
								   sum(mm.charge_vacc)            as charge_vacc,
								   sum(mm.charge_quickfix)        as charge_quickfix,
								   sum(mp.buildings)              as buildings,
								   sum(mp.health_packs)           as health_packs,
								   sum(mp.extinguishes)           as extinguishes
							FROM match_player mp
									 LEFT JOIN match_medic mm on mp.match_player_id = mm.match_player_id
							GROUP BY mp.steam_id) h ON h.steam_id = p.steam_id
				 LEFT JOIN (SELECT mp.steam_id,
								   sum(case when m.winner = mp.team then 1 else 0 end) as wins,
								   count(m.match_id)                                   as matches
							FROM match m
									 LEFT JOIN match_player mp on m.match_id = mp.match_id
							GROUP BY mp.steam_id) mx ON mx.steam_id = p.steam_id
		
				 LEFT JOIN (SELECT mp.steam_id,
								   mpc.player_class_id,
								   SUM(mpc.assists)             as assists,
								   SUM(mpc.kills)               as kills,
								   sum(mpc.deaths)              as deaths,
								   sum(mpc.playtime)            as playtime,
								   sum(mpc.dominations)         as dominations,
								   sum(mpc.dominated)           as dominated,
								   sum(mpc.revenges)            as revenges,
								   sum(mpc.damage)              as damage,
								   sum(mpc.damage_taken)        as damage_taken,
								   sum(mpc.healing_taken)       as healing_taken,
								   sum(mpc.captures)            as captures,
								   sum(mpc.captures_blocked)    as captures_blocked,
								   sum(mpc.buildings_destroyed) as buildings_destroyed
							FROM match_player mp
									 LEFT JOIN match_player_class mpc on mp.match_player_id = mpc.match_player_id
							GROUP BY mp.steam_id, mpc.player_class_id) c ON c.steam_id = p.steam_id and c.player_class_id = 7
		ORDER BY rank
		LIMIT $1`

	rows, errQuery := db.Query(ctx, query, count)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()

	var results []HealingOverallResult

	for rows.Next() {
		var (
			wor   HealingOverallResult
			sid64 int64
		)

		if errScan := rows.
			Scan(&wor.Rank,
				&sid64, &wor.Personaname, &wor.AvatarHash,
				&wor.Healing, &wor.Drops, &wor.NearFullChargeDeath, &wor.AvgUberLength, &wor.MajorAdvLost,
				&wor.BiggestAdvLost, &wor.ChargesUber, &wor.ChargesKritz, &wor.ChargesVacc, &wor.ChargesQuickfix,
				&wor.Extinguishes, &wor.HealthPacks, &wor.Assists, &wor.KA, &wor.Deaths, &wor.HPM, &wor.KAD,
				&wor.Playtime, &wor.Dominations, &wor.Dominated, &wor.Revenges,
				&wor.DamageTaken, &wor.DTM, &wor.Wins, &wor.Matches, &wor.WinRate,
			); errScan != nil {
			return nil, Err(errScan)
		}

		wor.SteamID = steamid.New(sid64)
		results = append(results, wor)
	}

	return results, nil
}

type PlayerClass struct {
	PlayerClassID int    `json:"player_class_id"`
	ClassName     string `json:"class_name"`
	ClassKey      string `json:"class_key"`
}

type PlayerClassOverallResult struct {
	PlayerClass
	Kills              int64   `json:"kills"`
	KA                 int64   `json:"ka"`
	Assists            int64   `json:"assists"`
	Deaths             int64   `json:"deaths"`
	KD                 float64 `json:"kd"`
	KAD                float64 `json:"kad"`
	DPM                float64 `json:"dpm"`
	Playtime           int64   `json:"playtime"`
	Dominations        int64   `json:"dominations"`
	Dominated          int64   `json:"dominated"`
	Revenges           int64   `json:"revenges"`
	Damage             int64   `json:"damage"`
	DamageTaken        int64   `json:"damage_taken"`
	HealingTaken       int64   `json:"healing_taken"`
	Captures           int64   `json:"captures"`
	CapturesBlocked    int64   `json:"captures_blocked"`
	BuildingsDestroyed int64   `json:"buildings_destroyed"`
}

func (db *Store) PlayerOverallClassStats(ctx context.Context, steamID steamid.SID64) ([]PlayerClassOverallResult, error) {
	const query = `
		SELECT
			c.player_class_id,
			c.class_name,
			c.class_key,
			sum(pc.kills) as kills,
			sum(pc.kills + pc.assists) as ka,
			sum(pc.assists) as assists,
			sum(pc.deaths) as deaths,
			sum(pc.playtime) as playtime,
			sum(pc.dominations) as dominations,
			sum(pc.dominated) as dominated,
			sum(pc.revenges) as revenges,
			sum(pc.damage) as damage,
			sum(pc.damage_taken) as damage_taken,
			sum(pc.healing_taken) as healing_taken,
			sum(pc.captures) as captures,
			sum(pc.captures_blocked) as captures_blocked,
			sum(pc.buildings_destroyed) as buildings_destroyed,
			case sum(pc.deaths) WHEN 0 THEN 0 ELSE ( sum(pc.kills)::float / sum(pc.deaths)::float) END kd,
			case sum(pc.deaths) WHEN 0 THEN 0 ELSE ((sum(pc.assists)::float +  sum(pc.kills)::float) / sum(pc.deaths)::float) END kad,
			sum(pc.damage)::float / (sum(pc.playtime)::float / 60) as dpm
		FROM match_player mp
		INNER JOIN match_player_class pc on mp.match_player_id = pc.match_player_id
		LEFT JOIN player_class c on pc.player_class_id = c.player_class_id
		WHERE mp.steam_id = $1
		GROUP BY c.player_class_id`

	rows, errQuery := db.Query(ctx, query, steamID.Int64())
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()

	var results []PlayerClassOverallResult

	for rows.Next() {
		var wor PlayerClassOverallResult

		if errScan := rows.
			Scan(&wor.PlayerClassID, &wor.ClassName, &wor.ClassKey,
				&wor.Kills, &wor.KA, &wor.Assists, &wor.Deaths, &wor.Playtime,
				&wor.Dominations, &wor.Dominated, &wor.Revenges, &wor.Damage, &wor.DamageTaken,
				&wor.HealingTaken, &wor.Captures, &wor.CapturesBlocked, &wor.BuildingsDestroyed,
				&wor.KD, &wor.KAD, &wor.DPM,
			); errScan != nil {
			return nil, Err(errScan)
		}

		results = append(results, wor)
	}

	return results, nil
}

type PlayerOverallResult struct {
	Healing             int64   `json:"healing"`
	Drops               int64   `json:"drops"`
	NearFullChargeDeath int64   `json:"near_full_charge_death"`
	AvgUberLen          float64 `json:"avg_uber_len"`
	BiggestAdvLost      float64 `json:"biggest_adv_lost"`
	MajorAdvLost        float64 `json:"major_adv_lost"`
	ChargesUber         int64   `json:"charges_uber"`
	ChargesKritz        int64   `json:"charges_kritz"`
	ChargesVacc         int64   `json:"charges_vacc"`
	ChargesQuickfix     int64   `json:"charges_quickfix"`
	Buildings           int64   `json:"buildings"`
	Extinguishes        int64   `json:"extinguishes"`
	HealthPacks         int64   `json:"health_packs"`
	KA                  int64   `json:"ka"`
	Kills               int64   `json:"kills"`
	Assists             int64   `json:"assists"`
	Deaths              int64   `json:"deaths"`
	KD                  float64 `json:"kd"`
	KAD                 float64 `json:"kad"`
	DPM                 float64 `json:"dpm"`
	Shots               int64   `json:"shots"`
	Hits                int64   `json:"hits"`
	Accuracy            float64 `json:"accuracy"`
	Airshots            int64   `json:"airshots"`
	Backstabs           int64   `json:"backstabs"`
	Headshots           int64   `json:"headshots"`
	Playtime            int64   `json:"playtime"`
	Dominations         int64   `json:"dominations"`
	Dominated           int64   `json:"dominated"`
	Revenges            int64   `json:"revenges"`
	Damage              int64   `json:"damage"`
	DamageTaken         int64   `json:"damage_taken"`
	Captures            int64   `json:"captures"`
	CapturesBlocked     int64   `json:"captures_blocked"`
	BuildingsDestroyed  int64   `json:"buildings_destroyed"`
	HealingTaken        int64   `json:"healing_taken"`
	Wins                int64   `json:"wins"`
	Matches             int64   `json:"matches"`
	WinRate             float64 `json:"win_rate"`
}

func (db *Store) PlayerOverallStats(ctx context.Context, steamID steamid.SID64, por *PlayerOverallResult) error {
	const query = `
		SELECT coalesce(h.healing, 0),
			   coalesce(h.drops, 0),
			   coalesce(h.near_full_charge_death, 0),
			   coalesce(h.avg_uber_length, 0),
			   coalesce(h.major_adv_lost, 0),
			   coalesce(h.biggest_adv_lost, 0),
			   coalesce(h.charge_uber, 0),
			   coalesce(h.charge_kritz, 0),
			   coalesce(h.charge_vacc, 0),
			   coalesce(h.charge_quickfix, 0),
			   coalesce(h.buildings, 0),
			   coalesce(h.extinguishes, 0),
			   coalesce(h.health_packs, 0),
			   coalesce(w.kills, 0) + coalesce(c.assists, 0)                                      as         ka,
			   coalesce(w.kills, 0),
			   coalesce(c.assists, 0),
			   coalesce(c.deaths, 0),
			   case c.deaths WHEN 0 THEN -1 ELSE (w.kills::float / c.deaths::float) END                      kd,
			   case c.deaths WHEN 0 THEN -1 ELSE ((c.assists::float + w.kills::float) / c.deaths::float) END kad,
			   case c.damage WHEN 0 THEN 0 ELSE c.damage::float / (c.playtime::float / 60) END    as         dpm,
			   coalesce(w.shots, 0),
			   coalesce(w.hits, 0),
			   case w.shots WHEN 0 THEN -1 ELSE (w.hits::float / w.shots::float) * 100 END                   acc,
			   coalesce(w.airshots, 0),
			   coalesce(w.backstabs, 0),
			   coalesce(w.headshots, 0),
			   coalesce(c.playtime, 0),
			   coalesce(c.dominations, 0),
			   coalesce(c.dominated, 0),
			   coalesce(c.revenges, 0),
			   coalesce(c.damage, 0),
			   coalesce(c.damage_taken, 0),
			   coalesce(c.captures, 0),
			   coalesce(c.captures_blocked, 0),
			   coalesce(c.buildings_destroyed, 0),
			   coalesce(c.healing_taken, 0),
			   coalesce(mx.wins, 0),
			   coalesce(mx.matches, 0),
			   case mx.matches WHEN 0 THEN -1 ELSE (mx.wins::float / mx.matches::float) * 100 END as         win_rate
		FROM person p
				 LEFT JOIN (SELECT mp.steam_id,
								   sum(mm.healing)                as healing,
								   sum(mm.drops)                  as drops,
								   sum(mm.near_full_charge_death) as near_full_charge_death,
								   sum(mm.avg_uber_length)        as avg_uber_length,
								   sum(mm.major_adv_lost)         as major_adv_lost,
								   sum(mm.biggest_adv_lost)       as biggest_adv_lost,
								   sum(mm.charge_uber)            as charge_uber,
								   sum(mm.charge_kritz)           as charge_kritz,
								   sum(mm.charge_vacc)            as charge_vacc,
								   sum(mm.charge_quickfix)        as charge_quickfix,
								   sum(mp.buildings)              as buildings,
								   sum(mp.health_packs)           as health_packs,
								   sum(mp.extinguishes)           as extinguishes
							FROM match_player mp
									 LEFT JOIN match_medic mm on mp.match_player_id = mm.match_player_id
							GROUP BY mp.steam_id) h ON h.steam_id = p.steam_id
				 LEFT JOIN (SELECT mp.steam_id,
								   sum(case when m.winner = mp.team then 1 else 0 end) as wins,
								   count(m.match_id)                                   as matches
							FROM match m
									 LEFT JOIN match_player mp on m.match_id = mp.match_id
							GROUP BY mp.steam_id) mx ON mx.steam_id = p.steam_id
				 LEFT JOIN (SELECT mp.steam_id,
								   sum(mw.kills)     as kills,
								   sum(mw.shots)     as shots,
								   sum(mw.hits)      as hits,
								   sum(mw.airshots)  as airshots,
								   sum(mw.backstabs) as backstabs,
								   sum(mw.headshots) as headshots
							FROM match_player mp
									 LEFT JOIN match_weapon mw on mp.match_player_id = mw.match_player_id
							GROUP BY mp.steam_id) w ON w.steam_id = p.steam_id
				 LEFT JOIN (SELECT mp.steam_id,
								   SUM(mpc.assists)             as assists,
								   sum(mpc.deaths)              as deaths,
								   sum(mpc.playtime)            as playtime,
								   sum(mpc.dominations)         as dominations,
								   sum(mpc.dominated)           as dominated,
								   sum(mpc.revenges)            as revenges,
								   sum(mpc.damage)              as damage,
								   sum(mpc.damage_taken)        as damage_taken,
								   sum(mpc.healing_taken)       as healing_taken,
								   sum(mpc.captures)            as captures,
								   sum(mpc.captures_blocked)    as captures_blocked,
								   sum(mpc.buildings_destroyed) as buildings_destroyed
							FROM match_player mp
									 LEFT JOIN match_player_class mpc on mp.match_player_id = mpc.match_player_id
							GROUP BY mp.steam_id) c ON c.steam_id = p.steam_id
		WHERE p.steam_id = $1`

	if errQuery := db.
		QueryRow(ctx, query, steamID.Int64()).Scan(
		&por.Healing, &por.Drops, &por.NearFullChargeDeath, &por.AvgUberLen, &por.MajorAdvLost, &por.BiggestAdvLost,
		&por.ChargesUber, &por.ChargesKritz, &por.ChargesVacc, &por.ChargesQuickfix, &por.Buildings, &por.Extinguishes,
		&por.HealthPacks, &por.KA, &por.Kills, &por.Assists, &por.Deaths, &por.KD, &por.KAD, &por.DPM, &por.Shots, &por.Hits, &por.Accuracy, &por.Airshots, &por.Backstabs,
		&por.Headshots, &por.Playtime, &por.Dominations, &por.Dominated, &por.Revenges, &por.Damage, &por.DamageTaken,
		&por.Captures, &por.CapturesBlocked, &por.BuildingsDestroyed, &por.HealingTaken, &por.Wins, &por.Matches, &por.WinRate,
	); errQuery != nil {
		return Err(errQuery)
	}

	return nil
}
