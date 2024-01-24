package store

import (
	"context"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

var ErrFailedWeapon = errors.New("failed to save weapon")

func (s Stores) LoadWeapons(ctx context.Context, weaponMap fp.MutexMap[logparse.Weapon, int]) error {
	for weapon, name := range logparse.NewWeaponParser().NameMap() {
		var newWeapon domain.Weapon
		if errWeapon := s.GetWeaponByKey(ctx, weapon, &newWeapon); errWeapon != nil {
			if !errors.Is(errWeapon, errs.ErrNoResult) {
				return errWeapon
			}

			newWeapon.Key = weapon
			newWeapon.Name = name

			if errSave := s.SaveWeapon(ctx, &newWeapon); errSave != nil {
				return errs.DBErr(errSave)
			}
		}

		weaponMap.Set(weapon, newWeapon.WeaponID)
	}

	return nil
}

func (s Stores) GetWeaponByKey(ctx context.Context, key logparse.Weapon, weapon *domain.Weapon) error {
	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("weapon_id", "key", "name").
		From("weapon").
		Where(sq.Eq{"key": key}))
	if errRow != nil {
		return errs.DBErr(errRow)
	}

	return errs.DBErr(row.Scan(&weapon.WeaponID, &weapon.Key, &weapon.Name))
}

func (s Stores) GetWeaponByID(ctx context.Context, weaponID int, weapon *domain.Weapon) error {
	row, errRow := s.QueryRowBuilder(ctx, s.
		Builder().
		Select("weapon_id", "key", "name").
		From("weapon").Where(sq.Eq{"weapon_id": weaponID}))
	if errRow != nil {
		return errs.DBErr(errRow)
	}

	return errs.DBErr(row.Scan(&weapon.WeaponID, &weapon.Key, &weapon.Name))
}

func (s Stores) SaveWeapon(ctx context.Context, weapon *domain.Weapon) error {
	if weapon.WeaponID > 0 {
		return errs.DBErr(s.ExecUpdateBuilder(ctx, s.
			Builder().
			Update("weapon").
			Set("key", weapon.Key).
			Set("name", weapon.Name).
			Where(sq.Eq{"weapon_id": weapon.WeaponID})))
	}

	const wq = `INSERT INTO weapon (key, name) VALUES ($1, $2) RETURNING weapon_id`

	if errSave := s.
		QueryRow(ctx, wq, weapon.Key, weapon.Name).
		Scan(&weapon.WeaponID); errSave != nil {
		return errors.Join(errSave, ErrFailedWeapon)
	}

	return nil
}

func (s Stores) Weapons(ctx context.Context, database Database) ([]domain.Weapon, error) {
	rows, errRows := database.QueryBuilder(ctx, database.
		Builder().
		Select("weapon_id", "key", "name").
		From("weapon"))
	if errRows != nil {
		return nil, errs.DBErr(errRows)
	}

	defer rows.Close()

	var weapons []domain.Weapon

	for rows.Next() {
		var weapon domain.Weapon
		if errScan := rows.Scan(&weapon.WeaponID, &weapon.Name); errScan != nil {
			return nil, errors.Join(errScan, ErrScanResult)
		}

		weapons = append(weapons, weapon)
	}

	if errRow := rows.Err(); errRow != nil {
		return nil, errors.Join(errRow, ErrRowResults)
	}

	return weapons, nil
}

func (s Stores) GetStats(ctx context.Context, stats *domain.Stats) error {
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

	if errQuery := s.QueryRow(ctx, query).
		Scan(&stats.BansTotal, &stats.BansDay, &stats.BansWeek, &stats.BansMonth, &stats.Bans3Month, &stats.Bans6Month, &stats.BansYear, &stats.BansCIDRTotal, &stats.FilteredWords, &stats.ServersTotal); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}

func (s Stores) GetMapUsageStats(ctx context.Context) ([]domain.MapUseDetail, error) {
	const query = `SELECT m.map, m.playtime, (m.playtime::float / s.total::float) * 100 percent
		FROM (
			SELECT SUM(extract('epoch' from m.time_end - m.time_start)) as playtime, m.map FROM match m
			    LEFT JOIN public.match_player mp on m.match_id = mp.match_id 
			GROUP BY m.map
		) m CROSS JOIN (
			SELECT SUM(extract('epoch' from mt.time_end - mt.time_start)) total FROM match mt
			LEFT JOIN public.match_player mpt on mt.match_id = mpt.match_id
		) s ORDER BY percent DESC`

	var details []domain.MapUseDetail

	rows, errQuery := s.Query(ctx, query)
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			mud     domain.MapUseDetail
			seconds int64
		)

		if errScan := rows.Scan(&mud.Map, &seconds, &mud.Percent); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		mud.Playtime = seconds

		details = append(details, mud)
	}

	if rows.Err() != nil {
		return nil, errors.Join(rows.Err(), ErrRowResults)
	}

	return details, nil
}

func (s Stores) TopChatters(ctx context.Context, count uint64) ([]domain.TopChatterResult, error) {
	rows, errRows := s.QueryBuilder(ctx, s.
		Builder().
		Select("p.personaname", "p.steam_id", "count(person_message_id) as total").
		From("person_messages m").
		LeftJoin("public.person p USING(steam_id)").
		GroupBy("p.steam_id").
		OrderBy("total DESC").
		Limit(count))
	if errRows != nil {
		return nil, errs.DBErr(errRows)
	}

	defer rows.Close()

	var results []domain.TopChatterResult

	for rows.Next() {
		var (
			tcr     domain.TopChatterResult
			steamID int64
		)

		if errScan := rows.Scan(&tcr.Name, &steamID, &tcr.Count); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		tcr.SteamID = steamid.New(steamID)
		results = append(results, tcr)
	}

	return results, nil
}

func (s Stores) WeaponsOverall(ctx context.Context) ([]domain.WeaponsOverallResult, error) {
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

	rows, errQuery := s.Query(ctx, query)
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}
	defer rows.Close()

	var results []domain.WeaponsOverallResult

	for rows.Next() {
		var wor domain.WeaponsOverallResult
		if errScan := rows.
			Scan(&wor.WeaponID, &wor.Name, &wor.Key,
				&wor.Kills, &wor.KillsPct,
				&wor.Headshots, &wor.HeadshotsPct,
				&wor.Airshots, &wor.AirshotsPct,
				&wor.Backstabs, &wor.BackstabsPct,
				&wor.Shots, &wor.ShotsPct,
				&wor.Hits, &wor.HitsPct,
				&wor.Damage, &wor.DamagePct); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		results = append(results, wor)
	}

	return results, nil
}

func (s Stores) WeaponsOverallTopPlayers(ctx context.Context, weaponID int) ([]domain.PlayerWeaponResult, error) {
	rows, errQuery := s.QueryBuilder(ctx, s.
		Builder().
		Select("row_number() over (order by SUM(mw.kills) desc nulls last) as rank",
			"p.steam_id", "p.personaname", "p.avatarhash",
			"SUM(mw.kills) as kills", "sum(mw.damage) as damage",
			"sum(mw.shots) as shots", "sum(mw.hits) as hits",
			"sum(mw.backstabs) as backstabs",
			"sum(mw.headshots) as headshots",
			"sum(mw.airshots) as airshots").
		From("match_weapon mw").
		LeftJoin("weapon w on w.weapon_id = mw.weapon_id").
		LeftJoin("match_player mp on mp.match_player_id = mw.match_player_id").
		LeftJoin("person p on mp.steam_id = p.steam_id").
		Where(sq.Eq{"w.weapon_id": weaponID}).
		GroupBy("p.steam_id", "w.weapon_id").
		OrderBy("kills DESC").
		Limit(250))
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}
	defer rows.Close()

	var results []domain.PlayerWeaponResult

	for rows.Next() {
		var (
			pwr   domain.PlayerWeaponResult
			sid64 int64
		)

		if errScan := rows.
			Scan(&pwr.Rank, &sid64, &pwr.Personaname, &pwr.AvatarHash,
				&pwr.Kills, &pwr.Damage,
				&pwr.Shots, &pwr.Hits,
				&pwr.Backstabs, &pwr.Headshots,
				&pwr.Airshots); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		pwr.SteamID = steamid.New(sid64)
		results = append(results, pwr)
	}

	return results, nil
}

func (s Stores) WeaponsOverallByPlayer(ctx context.Context, steamID steamid.SID64) ([]domain.WeaponsOverallResult, error) {
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

	rows, errQuery := s.Query(ctx, query, steamID.Int64())
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}
	defer rows.Close()

	var results []domain.WeaponsOverallResult

	for rows.Next() {
		var wor domain.WeaponsOverallResult
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
			return nil, errs.DBErr(errScan)
		}

		results = append(results, wor)
	}

	return results, nil
}

func (s Stores) PlayersOverallByKills(ctx context.Context, count int) ([]domain.PlayerWeaponResult, error) {
	const query = `SELECT row_number() over (order by c.assists + w.kills desc nulls last) as rank,
			   p.personaname,
			   p.steam_id,
			   p.avatarhash,
			   coalesce(w.kills, 0) + coalesce(c.assists, 0) as ka,
			   coalesce(w.kills, 0),
			   coalesce(c.assists, 0),
			   coalesce(c.deaths, 0),
			   case coalesce(c.deaths, 0) WHEN 0 THEN -1 ELSE (coalesce(w.kills, 0)::float / c.deaths::float) END kd,
			   case coalesce(c.deaths, 0) WHEN 0 THEN -1 ELSE ((coalesce(c.assists, 0)::float + coalesce(w.kills,0)::float) / c.deaths::float) END kad,
			   case coalesce(c.playtime, 0) WHEN 0 THEN 0 ELSE coalesce(c.damage, 0)::float / (c.playtime::float / 60) END as dpm,
			   coalesce(w.shots, 0),
			   coalesce(w.hits, 0),
			   case coalesce(w.shots, 0) WHEN 0 THEN -1 ELSE (w.hits::float / w.shots::float) * 100 END acc,
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
			   coalesce( c.captures_blocked, 0),
			   coalesce(c.buildings_destroyed, 0)
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

	rows, errQuery := s.Query(ctx, query, count)
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}
	defer rows.Close()

	var results []domain.PlayerWeaponResult

	for rows.Next() {
		var (
			wor   domain.PlayerWeaponResult
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
			return nil, errs.DBErr(errScan)
		}

		wor.SteamID = steamid.New(sid64)
		results = append(results, wor)
	}

	return results, nil
}

func (s Stores) HealersOverallByHealing(ctx context.Context, count int) ([]domain.HealingOverallResult, error) {
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

	rows, errQuery := s.Query(ctx, query, count)
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}
	defer rows.Close()

	var results []domain.HealingOverallResult

	for rows.Next() {
		var (
			wor   domain.HealingOverallResult
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
			return nil, errs.DBErr(errScan)
		}

		wor.SteamID = steamid.New(sid64)
		results = append(results, wor)
	}

	return results, nil
}

func (s Stores) PlayerOverallClassStats(ctx context.Context, steamID steamid.SID64) ([]domain.PlayerClassOverallResult, error) {
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

	rows, errQuery := s.Query(ctx, query, steamID.Int64())
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}
	defer rows.Close()

	var results []domain.PlayerClassOverallResult

	for rows.Next() {
		var wor domain.PlayerClassOverallResult

		if errScan := rows.
			Scan(&wor.PlayerClassID, &wor.ClassName, &wor.ClassKey,
				&wor.Kills, &wor.KA, &wor.Assists, &wor.Deaths, &wor.Playtime,
				&wor.Dominations, &wor.Dominated, &wor.Revenges, &wor.Damage, &wor.DamageTaken,
				&wor.HealingTaken, &wor.Captures, &wor.CapturesBlocked, &wor.BuildingsDestroyed,
				&wor.KD, &wor.KAD, &wor.DPM,
			); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		results = append(results, wor)
	}

	return results, nil
}

func (s Stores) PlayerOverallStats(ctx context.Context, steamID steamid.SID64, por *domain.PlayerOverallResult) error {
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
			   coalesce(w.kills, 0) + coalesce(c.assists, 0)                                                   as        ka,
			   coalesce(w.kills, 0),
			   coalesce(c.assists, 0),
			   coalesce(c.deaths, 0),
			   coalesce(case c.deaths WHEN 0 THEN 0 ELSE (w.kills::float / c.deaths::float) END, 0)                      kd,
			   coalesce(case c.deaths WHEN 0 THEN 0 ELSE ((c.assists::float + w.kills::float) / c.deaths::float) END, 0) kad,
			   coalesce(case c.playtime WHEN 0 THEN 0 ELSE c.damage::float / (c.playtime::float / 60) END, 0)  as        dpm,
			   coalesce(w.shots, 0),
			   coalesce(w.hits, 0),
			   coalesce(case w.shots WHEN 0 THEN -1 ELSE (w.hits::float / w.shots::float) * 100 END, 0)                  acc,
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
			   coalesce(case mx.matches WHEN 0 THEN -1 ELSE (mx.wins::float / mx.matches::float) * 100 END, 0) as        win_rate
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

	if errQuery := s.
		QueryRow(ctx, query, steamID.Int64()).Scan(
		&por.Healing, &por.Drops, &por.NearFullChargeDeath, &por.AvgUberLen, &por.MajorAdvLost, &por.BiggestAdvLost,
		&por.ChargesUber, &por.ChargesKritz, &por.ChargesVacc, &por.ChargesQuickfix, &por.Buildings, &por.Extinguishes,
		&por.HealthPacks, &por.KA, &por.Kills, &por.Assists, &por.Deaths, &por.KD, &por.KAD, &por.DPM, &por.Shots, &por.Hits, &por.Accuracy, &por.Airshots, &por.Backstabs,
		&por.Headshots, &por.Playtime, &por.Dominations, &por.Dominated, &por.Revenges, &por.Damage, &por.DamageTaken,
		&por.Captures, &por.CapturesBlocked, &por.BuildingsDestroyed, &por.HealingTaken, &por.Wins, &por.Matches, &por.WinRate,
	); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}
