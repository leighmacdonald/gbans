package store

import (
	"context"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

var (
	ErrMatchQuery           = errors.New("failed to load match")
	ErrQueryPlayers         = errors.New("failed to query match players")
	ErrQueryMatch           = errors.New("failed to query match")
	ErrChatQuery            = errors.New("failed to query chat history")
	ErrGetPlayerClasses     = errors.New("failed to fetch player class stats")
	ErrGetMedicStats        = errors.New("failed to fetch medic class stats")
	ErrSaveMedicStats       = errors.New("failed to save medic stats")
	ErrSavePlayerStats      = errors.New("failed to save player stats")
	ErrSaveWeaponStats      = errors.New("failed to save weapon stats")
	ErrSaveClassStats       = errors.New("failed to save class stats")
	ErrSaveKillstreakStats  = errors.New("failed to save killstreak stats")
	ErrGetWeaponStats       = errors.New("failed to fetch match weapon stats")
	ErrGetPlayerKillstreaks = errors.New("failed to fetch player killstreak stats")
	ErrGetPerson            = errors.New("failed to fetch person result")
)

func (s Stores) matchGetPlayerClasses(ctx context.Context, matchID uuid.UUID) (map[steamid.SID64][]domain.MatchPlayerClass, error) {
	const query = `
		SELECT mp.steam_id, c.match_player_class_id, c.match_player_id, c.player_class_id, c.kills, 
		   c.assists, c.deaths, c.playtime, c.dominations, c.dominated, c.revenges, c.damage, c.damage_taken, c.healing_taken,
		   c.captures, c.captures_blocked, c.buildings_destroyed
		FROM match_player_class c
		LEFT JOIN match_player mp on mp.match_player_id = c.match_player_id
		WHERE mp.match_id = $1`

	rows, errQuery := s.Query(ctx, query, matchID)
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}

	defer rows.Close()

	results := map[steamid.SID64][]domain.MatchPlayerClass{}

	for rows.Next() {
		var (
			steamID int64
			stats   domain.MatchPlayerClass
		)

		if errScan := rows.
			Scan(&steamID, &stats.MatchPlayerClassID, &stats.MatchPlayerID, &stats.PlayerClass,
				&stats.Kills, &stats.Assists, &stats.Deaths, &stats.Playtime, &stats.Dominations, &stats.Dominated,
				&stats.Revenges, &stats.Damage, &stats.DamageTaken, &stats.HealingTaken, &stats.Captures,
				&stats.CapturesBlocked, &stats.BuildingDestroyed); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		sid := steamid.New(steamID)

		res, found := results[sid]
		if !found {
			res = []domain.MatchPlayerClass{}
		}

		results[sid] = append(res, stats)
	}

	if errRows := rows.Err(); errRows != nil {
		return nil, errs.DBErr(errRows)
	}

	return results, nil
}

func (s Stores) matchGetPlayerWeapons(ctx context.Context, matchID uuid.UUID) (map[steamid.SID64][]domain.MatchPlayerWeapon, error) {
	const query = `
		SELECT mp.steam_id, mw.weapon_id, w.name, w.key,  mw.kills, mw.damage, mw.shots, mw.hits, mw.backstabs, mw.headshots, mw.airshots
		FROM match s
		LEFT JOIN match_player mp on s.match_id = mp.match_id
		LEFT JOIN match_weapon mw on mp.match_player_id = mw.match_player_id
		LEFT JOIN weapon w on w.weapon_id = mw.weapon_id
		WHERE s.match_id = $1 and mw.weapon_id is not null
		ORDER BY mw.kills DESC`

	results := map[steamid.SID64][]domain.MatchPlayerWeapon{}

	rows, errRows := s.Query(ctx, query, matchID)
	if errRows != nil {
		return nil, errs.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			steamID int64
			mpw     domain.MatchPlayerWeapon
		)

		if errScan := rows.
			Scan(&steamID, &mpw.WeaponID, &mpw.Weapon.Name, &mpw.Weapon.Key, &mpw.Kills, &mpw.Damage, &mpw.Shots,
				&mpw.Hits, &mpw.Backstabs, &mpw.Headshots, &mpw.Airshots); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		sid := steamid.New(steamID)

		res, found := results[sid]
		if !found {
			res = []domain.MatchPlayerWeapon{}
		}

		results[sid] = append(res, mpw)
	}

	return results, nil
}

func (s Stores) matchGetPlayerKillstreak(ctx context.Context, matchID uuid.UUID) (map[steamid.SID64][]domain.MatchPlayerKillstreak, error) {
	const query = `
		SELECT mp.steam_id, k.match_player_id, k.player_class_id, k.killstreak, k.duration
		FROM match_player_killstreak k
		LEFT JOIN match_player mp on mp.match_player_id = k.match_player_id
		WHERE mp.match_id = $1`

	rows, errRows := s.Query(ctx, query, matchID)
	if errRows != nil {
		return nil, errs.DBErr(errRows)
	}

	defer rows.Close()

	results := map[steamid.SID64][]domain.MatchPlayerKillstreak{}

	for rows.Next() {
		var (
			steamID int64
			stats   domain.MatchPlayerKillstreak
		)

		if errScan := rows.
			Scan(&steamID, &stats.MatchPlayerID, &stats.PlayerClass, &stats.Killstreak, &stats.Duration); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		sid := steamid.New(steamID)

		res, found := results[sid]
		if !found {
			res = []domain.MatchPlayerKillstreak{}
		}

		results[sid] = append(res, stats)
	}

	return results, nil
}

func (s Stores) matchGetPlayers(ctx context.Context, matchID uuid.UUID) ([]*domain.MatchPlayer, error) {
	const queryPlayer = `
		SELECT
			p.match_player_id,
			p.steam_id,
			p.team,
			p.time_start,
			p.time_end,
			coalesce(w.kills, 0) as kills,
			coalesce(w.damage, 0) as damage,
			coalesce(w.shots, 0) as shots,
			coalesce(w.hits, 0) as hits,
			coalesce(w.backstabs, 0) as backstabs,
			coalesce(w.headshots, 0) as headshots,
			coalesce(w.airshots, 0) as airshots,
			coalesce(SUM(c.assists), 0)             as assists,
			coalesce(SUM(c.deaths), 0)              as deaths,
			coalesce(SUM(c.dominations), 0)         as dominations,
			coalesce(SUM(c.dominated), 0)           as dominated,
			coalesce(SUM(c.revenges), 0)            as revenges,
			coalesce(SUM(c.damage_taken), 0)        as damage_taken,
			coalesce(SUM(c.healing_taken), 0)       as healing_taken,
			p.health_packs,
			coalesce(SUM(c.captures), 0)            as captures,
			coalesce(SUM(c.captures_blocked), 0)    as captures_blocked,
			p.extinguishes,
			p.buildings,
			coalesce(SUM(c.buildings_destroyed), 0) as buildings_destroyed,
			pe.personaname,
			pe.avatarhash
		FROM match_player p
		LEFT JOIN match_player_class c on c.match_player_id = p.match_player_id
		LEFT JOIN person pe on p.steam_id = pe.steam_id
		LEFT JOIN (
			SELECT p.match_player_id,
				coalesce(SUM(w.kills), 0)     as kills,
				   coalesce(SUM(w.backstabs), 0) as backstabs,
				   coalesce(SUM(w.headshots), 0) as headshots,
				   coalesce(SUM(w.airshots), 0)  as airshots,
				   coalesce(SUM(w.shots), 0)     as shots,
				   coalesce(SUM(w.hits), 0)      as hits,
				   coalesce(SUM(w.damage), 0)    as damage
			FROM match_weapon w
			LEFT JOIN match_player p on w.match_player_id = p.match_player_id
			GROUP BY p.match_player_id, p.match_id
			ORDER BY kills DESC
		) w ON w.match_player_id = p.match_player_id
		WHERE p.match_id = $1
		GROUP BY 
			p.match_player_id, w.kills, w.damage, w.shots, w.hits, w.backstabs, w.headshots, w.airshots, pe.steam_id
		ORDER BY w.kills DESC`

	var players []*domain.MatchPlayer

	playerRows, errPlayer := s.Query(ctx, queryPlayer, matchID)
	if errPlayer != nil {
		if errors.Is(errPlayer, errs.ErrNoResult) {
			return []*domain.MatchPlayer{}, nil
		}

		return nil, errors.Join(errPlayer, ErrQueryPlayers)
	}

	defer playerRows.Close()

	for playerRows.Next() {
		var (
			mpSum   = &domain.MatchPlayer{}
			steamID int64
		)

		if errRow := playerRows.
			Scan(&mpSum.MatchPlayerID, &steamID, &mpSum.Team, &mpSum.TimeStart, &mpSum.TimeEnd,
				&mpSum.Kills, &mpSum.Damage, &mpSum.Shots, &mpSum.Hits, &mpSum.Backstabs,
				&mpSum.Headshots, &mpSum.Airshots, &mpSum.Assists, &mpSum.Deaths, &mpSum.Dominations, &mpSum.Dominated, &mpSum.Revenges,
				&mpSum.DamageTaken, &mpSum.HealingTaken, &mpSum.HealingPacks, &mpSum.Captures, &mpSum.CapturesBlocked,
				&mpSum.Extinguishes, &mpSum.BuildingBuilt, &mpSum.BuildingDestroyed, &mpSum.Name,
				&mpSum.AvatarHash); errRow != nil {
			return nil, errors.Join(errPlayer, ErrScanResult)
		}

		mpSum.SteamID = steamid.New(steamID)
		players = append(players, mpSum)
	}

	return players, nil
}

func (s Stores) matchGetMedics(ctx context.Context, matchID uuid.UUID) (map[steamid.SID64]domain.MatchHealer, error) {
	const query = `
		SELECT s.match_medic_id,
			   mp.steam_id,
			   s.healing,
			   s.drops,
			   s.near_full_charge_death,
			   s.avg_uber_length,
			   s.major_adv_lost,
			   s.biggest_adv_lost,
			   s.charge_uber,
			   s.charge_kritz,
			   s.charge_vacc,
			   s.charge_quickfix
		FROM match_medic s
		LEFT JOIN match_player mp on mp.match_player_id = s.match_player_id
		WHERE mp.match_id = $1`

	medics := map[steamid.SID64]domain.MatchHealer{}

	medicRows, errMedics := s.Query(ctx, query, matchID)
	if errMedics != nil {
		if errors.Is(errMedics, errs.ErrNoResult) {
			return medics, nil
		}

		return nil, errors.Join(errMedics, ErrQueryPlayers)
	}

	defer medicRows.Close()

	for medicRows.Next() {
		var (
			mpSum   = domain.MatchHealer{}
			steamID int64
		)

		if errRow := medicRows.
			Scan(&mpSum.MatchMedicID, &steamID, &mpSum.Healing, &mpSum.Drops,
				&mpSum.NearFullChargeDeath, &mpSum.AvgUberLength, &mpSum.MajorAdvLost,
				&mpSum.BiggestAdvLost, &mpSum.ChargesUber, &mpSum.ChargesKritz,
				&mpSum.ChargesVacc, &mpSum.ChargesQuickfix); errRow != nil {
			return nil, errors.Join(errMedics, ErrScanResult)
		}

		sid := steamid.New(steamID)

		medics[sid] = mpSum
	}

	if medicRows.Err() != nil {
		return medics, errors.Join(medicRows.Err(), ErrRowResults)
	}

	return medics, nil
}

func (s Stores) matchGetChat(ctx context.Context, matchID uuid.UUID) (domain.PersonMessages, error) {
	const query = `
		SELECT c.person_message_id, c.steam_id, c.server_id, c.body, c.persona_name, c.team, 
		       c.created_on, c.match_id, COUNT(f.person_message_id)::int::boolean as flagged
		FROM person_messages c
		LEFT JOIN person_messages_filter f on c.person_message_id = f.person_message_id
		WHERE c.match_id = $1
		GROUP BY c.person_message_id
		`

	messages := domain.PersonMessages{}

	chatRows, errQuery := s.Query(ctx, query, matchID)
	if errQuery != nil {
		if errors.Is(errQuery, errs.ErrNoResult) {
			return messages, nil
		}

		return nil, errors.Join(errQuery, ErrChatQuery)
	}

	defer chatRows.Close()

	for chatRows.Next() {
		var (
			msg     domain.PersonMessage
			steamID int64
		)

		if errRow := chatRows.
			Scan(&msg.PersonMessageID, &steamID, &msg.ServerID, &msg.Body,
				&msg.PersonaName, &msg.Team, &msg.CreatedOn,
				&msg.MatchID, &msg.Flagged); errRow != nil {
			return nil, errors.Join(errQuery, ErrScanResult)
		}

		msg.SteamID = steamid.New(steamID)
		messages = append(messages, msg)
	}

	if chatRows.Err() != nil {
		return messages, errors.Join(chatRows.Err(), ErrRowResults)
	}

	return messages, nil
}

func (s Stores) MatchGetByID(ctx context.Context, matchID uuid.UUID, match *domain.MatchResult) error {
	const query = `
		SELECT match_id, server_id, map, title, score_red, score_blu, time_red, time_blu, time_start, time_end, winner
		FROM match WHERE match_id = $1`

	if errMatch := s.
		QueryRow(ctx, query, matchID).
		Scan(&match.MatchID, &match.ServerID, &match.MapName, &match.Title,
			&match.TeamScores.Red, &match.TeamScores.Blu, &match.TeamScores.BluTime, &match.TeamScores.BluTime,
			&match.TimeStart, &match.TimeEnd, &match.Winner); errMatch != nil {
		return errors.Join(errMatch, ErrMatchQuery)
	}

	playerStats, errPlayerStats := s.matchGetPlayers(ctx, matchID)
	if errPlayerStats != nil {
		return errors.Join(errPlayerStats, ErrQueryPlayers)
	}

	match.Players = playerStats

	playerClasses, errPlayerClasses := s.matchGetPlayerClasses(ctx, matchID)
	if errPlayerClasses != nil {
		return errors.Join(errPlayerClasses, ErrGetPlayerClasses)
	}

	for _, player := range playerStats {
		if classes, found := playerClasses[player.SteamID]; found {
			player.Classes = classes
		}
	}

	playerKillstreaks, errPlayerKillstreaks := s.matchGetPlayerKillstreak(ctx, matchID)
	if errPlayerKillstreaks != nil {
		return errors.Join(errPlayerKillstreaks, ErrGetPlayerKillstreaks)
	}

	for _, player := range match.Players {
		if killstreaks, found := playerKillstreaks[player.SteamID]; found {
			player.Killstreaks = killstreaks
		}
	}

	medicStats, errMedics := s.matchGetMedics(ctx, matchID)
	if errMedics != nil {
		return errors.Join(errMedics, ErrGetMedicStats)
	}

	for steamID, stats := range medicStats {
		localStats := stats

		for _, player := range match.Players {
			if player.SteamID == steamID {
				player.MedicStats = &localStats

				break
			}
		}
	}

	weaponStats, errWeapons := s.matchGetPlayerWeapons(ctx, matchID)
	if errWeapons != nil {
		return errors.Join(errMedics, ErrGetWeaponStats)
	}

	for steamID, stats := range weaponStats {
		localStats := stats

		for _, player := range match.Players {
			if player.SteamID == steamID {
				player.Weapons = localStats

				break
			}
		}
	}

	chat, errChat := s.matchGetChat(ctx, matchID)

	if errChat != nil && !errors.Is(errChat, errs.ErrNoResult) {
		return errChat
	}

	match.Chat = chat

	if match.Chat == nil {
		match.Chat = domain.PersonMessages{}
	}

	for _, player := range match.Players {
		if player.Weapons == nil {
			player.Weapons = []domain.MatchPlayerWeapon{}
		}

		if player.Classes == nil {
			player.Classes = []domain.MatchPlayerClass{}
		}

		if player.Killstreaks == nil {
			player.Killstreaks = []domain.MatchPlayerKillstreak{}
		}
	}

	return nil
}

const MinMedicHealing = 500

func (s Stores) MatchSave(ctx context.Context, match *logparse.Match, weaponMap fp.MutexMap[logparse.Weapon, int]) error {
	const (
		query = `
		INSERT INTO match (match_id, server_id, map, title, score_red, score_blu, time_red, time_blu, time_start, time_end, winner) 
		VALUES ($1, $2, $3, $4, $5, $6,$7, $8, $9, $10, $11) 
		RETURNING match_id`
	)

	transaction, errTx := s.Begin(ctx)
	if errTx != nil {
		return errors.Join(errTx, ErrTxStart)
	}

	if errQuery := transaction.
		QueryRow(ctx, query, match.MatchID, match.ServerID, match.MapName, match.Title,
			match.TeamScores.Red, match.TeamScores.Blu, match.TeamScores.RedTime, match.TeamScores.BluTime,
			match.TimeStart, match.TimeEnd, match.Winner()).
		Scan(&match.MatchID); errQuery != nil {
		if errRollback := transaction.Rollback(ctx); errRollback != nil {
			return errors.Join(errRollback, ErrTxRollback)
		}

		return errors.Join(errQuery, ErrMatchQuery)
	}

	for _, player := range match.PlayerSums {
		if !player.SteamID.Valid() {
			// TODO Why can this happen? stv host?
			continue
		}

		var loadPlayerTest domain.Person
		if errPlayer := s.GetOrCreatePersonBySteamID(ctx, player.SteamID, &loadPlayerTest); errPlayer != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				return errors.Join(errRollback, ErrTxRollback)
			}

			return errors.Join(errPlayer, ErrGetPerson)
		}

		if errSave := s.saveMatchPlayerStats(ctx, transaction, match, player); errSave != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				return errors.Join(errRollback, ErrTxRollback)
			}

			return errSave
		}

		if errSave := s.saveMatchWeaponStats(ctx, transaction, player, weaponMap); errSave != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				return errors.Join(errRollback, ErrTxRollback)
			}

			return errSave
		}

		if errSave := s.saveMatchPlayerClassStats(ctx, transaction, player); errSave != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				return errors.Join(errRollback, ErrTxRollback)
			}

			return errSave
		}

		if errSave := s.saveMatchKillstreakStats(ctx, transaction, player); errSave != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				return errors.Join(errRollback, ErrTxRollback)
			}

			return errSave
		}

		if player.HealingStats != nil && player.HealingStats.Healing >= MinMedicHealing {
			if errSave := s.saveMatchMedicStats(ctx, transaction, player.MatchPlayerID, player.HealingStats); errSave != nil {
				if errRollback := transaction.Rollback(ctx); errRollback != nil {
					return errors.Join(errRollback, ErrTxRollback)
				}

				return errSave
			}
		}
	}

	if errCommit := transaction.Commit(ctx); errCommit != nil {
		return errors.Join(errCommit, ErrTxCommit)
	}

	return nil
}

func (s Stores) saveMatchPlayerStats(ctx context.Context, database pgx.Tx, match *logparse.Match, stats *logparse.PlayerStats) error {
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

	if errPlayerExec := database.
		QueryRow(ctx, playerQuery, match.MatchID, stats.SteamID.Int64(), stats.Team, stats.TimeStart,
			endTime, stats.HealthPacks(), stats.Extinguishes(), stats.BuildingBuilt).
		Scan(&stats.MatchPlayerID); errPlayerExec != nil {
		return errors.Join(errPlayerExec, ErrSavePlayerStats)
	}

	return nil
}

func (s Stores) saveMatchMedicStats(ctx context.Context, transaction pgx.Tx, matchPlayerID int64, stats *logparse.HealingStats) error {
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
		return errors.Join(errMedExec, ErrSaveMedicStats)
	}

	return nil
}

func (s Stores) saveMatchWeaponStats(ctx context.Context, transaction pgx.Tx, player *logparse.PlayerStats, weaponMap fp.MutexMap[logparse.Weapon, int]) error {
	const query = `
		INSERT INTO match_weapon (match_player_id, weapon_id, kills, damage, shots, hits, backstabs, headshots, airshots) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) 
		RETURNING player_weapon_id`

	for weapon, info := range player.WeaponInfo {
		weaponID, found := weaponMap.Get(weapon)
		if !found {
			// db.log.Error("Unknown weapon", zap.String("weapon", string(weapon)))
			continue
		}

		if _, errWeapon := transaction.
			Exec(ctx, query, player.MatchPlayerID, weaponID, info.Kills, info.Damage, info.Shots, info.Hits,
				info.BackStabs, info.Headshots, info.Airshots); errWeapon != nil {
			return errors.Join(errWeapon, ErrSaveWeaponStats)
		}
	}

	return nil
}

func (s Stores) saveMatchPlayerClassStats(ctx context.Context, transaction pgx.Tx, player *logparse.PlayerStats) error {
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
			return errors.Join(errWeapon, ErrSaveClassStats)
		}
	}

	return nil
}

func (s Stores) saveMatchKillstreakStats(ctx context.Context, transaction pgx.Tx, player *logparse.PlayerStats) error {
	const query = `
		INSERT INTO match_player_killstreak (match_player_id, player_class_id, killstreak, duration) 
		VALUES ($1, $2, $3, $4)`

	for class, stats := range player.KillStreaks {
		if _, errWeapon := transaction.
			Exec(ctx, query, player.MatchPlayerID, class, stats.Killstreak, stats.Duration); errWeapon != nil {
			return errors.Join(errWeapon, ErrSaveKillstreakStats)
		}
	}

	return nil
}

func (s Stores) StatsPlayerClass(ctx context.Context, sid64 steamid.SID64) (domain.PlayerClassStatsCollection, error) {
	const query = `
		SELECT c.player_class_id,
			   coalesce(SUM(c.kills), 0)               as kill,
			   coalesce(SUM(c.damage), 0)              as damage,
			   coalesce(SUM(c.assists), 0)             as assists,
			   coalesce(SUM(c.deaths), 0)              as deaths,
			   coalesce(SUM(c.dominations), 0)         as dominations,
			   coalesce(SUM(c.dominated), 0)           as dominated,
			   coalesce(SUM(c.revenges), 0)            as revenges,
			   coalesce(SUM(c.damage_taken), 0)        as damage_taken,
			   coalesce(SUM(c.healing_taken), 0)       as healing_taken,
			   coalesce(SUM(p.health_packs), 0)        as health_packs,
			   coalesce(SUM(c.captures), 0)            as captures,
			   coalesce(SUM(c.captures_blocked), 0)    as captures_blocked,
			   coalesce(SUM(p.extinguishes), 0)        as extinguishes,
			   coalesce(SUM(p.buildings), 0)           as buildings_built,
			   coalesce(SUM(c.buildings_destroyed), 0) as buildings_destroyed,
			   coalesce(SUM(c.playtime), 0)            as playtime
		FROM match_player p
		LEFT JOIN match_player_class c on c.match_player_id = p.match_player_id
		WHERE p.steam_id = $1 AND c.player_class_id != 0
		GROUP BY p.steam_id, c.player_class_id
		ORDER BY c.player_class_id`

	rows, errQuery := s.Query(ctx, query, sid64.Int64())
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}

	defer rows.Close()

	var stats domain.PlayerClassStatsCollection

	for rows.Next() {
		var class domain.PlayerClassStats
		if errScan := rows.
			Scan(&class.Class, &class.Kills, &class.Damage, &class.Assists, &class.Deaths, &class.Dominations,
				&class.Dominated, &class.Revenges, &class.DamageTaken, &class.HealingTaken, &class.HealthPacks,
				&class.Captures, &class.CapturesBlocked, &class.Extinguishes, &class.BuildingsBuilt,
				&class.BuildingsDestroyed, &class.Playtime); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		class.ClassName = class.Class.String()
		stats = append(stats, class)
	}

	return stats, nil
}

func (s Stores) StatsPlayerWeapons(ctx context.Context, sid64 steamid.SID64) ([]domain.PlayerWeaponStats, error) {
	const query = `
		SELECT n.key,
			   n.name,
			   SUM(w.kills)     as kills,
			   SUM(w.damage)     as damage,
			   SUM(w.shots)     as shots,
			   SUM(w.hits)      as hits,
			   SUM(w.backstabs) as backstabs,
			   SUM(w.headshots) as headshots,
			   SUM(w.airshots)  as airshots
		FROM match_player p
		LEFT JOIN match_weapon w on p.match_player_id = w.match_player_id
		LEFT JOIN weapon n on n.weapon_id = w.weapon_id
		WHERE p.steam_id = $1
		  AND w.weapon_id IS NOT NULL
		GROUP BY w.weapon_id, n.weapon_id;`

	rows, errQuery := s.Query(ctx, query, sid64.Int64())
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}

	defer rows.Close()

	var stats []domain.PlayerWeaponStats

	for rows.Next() {
		var class domain.PlayerWeaponStats
		if errScan := rows.
			Scan(&class.Weapon, &class.WeaponName, &class.Kills, &class.Damage, &class.Shots, &class.Hits,
				&class.Backstabs, &class.Headshots, &class.Airshots); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		stats = append(stats, class)
	}

	return stats, nil
}

func (s Stores) StatsPlayerKillstreaks(ctx context.Context, sid64 steamid.SID64) ([]domain.PlayerKillstreakStats, error) {
	const query = `
		SELECT k.player_class_id,
			   SUM(k.killstreak) as killstreak,
			   SUM(k.duration)   as duration,
			   s.time_start
		FROM match_player p
				 LEFT JOIN match_player_killstreak k on p.match_player_id = k.match_player_id
				 LEFT JOIN match_player mp on mp.match_player_id = k.match_player_id
				 LEFT JOIN match s on p.match_id = s.match_id
		WHERE p.steam_id = $1
		  AND k.player_class_id IS NOT NULL
		GROUP BY k.match_killstreak_id, s.time_start, k.player_class_id
		ORDER BY killstreak DESC
		LIMIT 10;`

	rows, errQuery := s.Query(ctx, query, sid64.Int64())
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}

	defer rows.Close()

	var stats []domain.PlayerKillstreakStats

	for rows.Next() {
		var class domain.PlayerKillstreakStats
		if errScan := rows.
			Scan(&class.Class, &class.Kills, &class.Duration, &class.CreatedOn); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		class.ClassName = class.Class.String()
		stats = append(stats, class)
	}

	return stats, nil
}

func (s Stores) StatsPlayerMedic(ctx context.Context, sid64 steamid.SID64) ([]domain.PlayerMedicStats, error) {
	const query = `
		SELECT coalesce(SUM(s.healing), 0)                as healing,
			   coalesce(SUM(s.drops), 0)                  as drops,
			   coalesce(SUM(s.near_full_charge_death), 0) as near_full_charge_death,
			   coalesce(AVG(s.avg_uber_length), 0)        as avg_uber_length,
			   coalesce(SUM(s.charge_uber), 0)            as charge_uber,
			   coalesce(SUM(s.charge_kritz), 0)           as charge_kritz,
			   coalesce(SUM(s.charge_vacc), 0)            as charge_vacc,
			   coalesce(SUM(s.charge_quickfix), 0)        as charge_quickfix
		FROM match_player p
		LEFT JOIN match_medic s on p.match_player_id = s.match_player_id
		WHERE p.steam_id = $1
		GROUP BY p.steam_id`

	rows, errQuery := s.Query(ctx, query, sid64.Int64())
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}

	defer rows.Close()

	var stats []domain.PlayerMedicStats

	for rows.Next() {
		var class domain.PlayerMedicStats
		if errScan := rows.
			Scan(&class.Healing, &class.Drops, &class.NearFullChargeDeath, &class.AvgUberLength,
				&class.ChargesUber, &class.ChargesKritz, &class.ChargesVacc, &class.ChargesQuickfix); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		stats = append(stats, class)
	}

	return stats, nil
}

func (s Stores) PlayerStats(ctx context.Context, steamID steamid.SID64, stats *domain.PlayerStats) error {
	const query = `
		SELECT count(s.match_id)            as                     matches,
			   sum(case when mp.team = s.winner then 1 else 0 end) wins,
			   sum(mp.health_packs)         as                     health_packs,
			   sum(mp.extinguishes)         as                     extinguishes,
			   sum(mp.buildings)            as                     buildings,
			   sum(mpc.kills)               as                     kill,
			   sum(mpc.assists)             as                     assists,
			   sum(mpc.damage)              as                     damage,
			   sum(mpc.damage_taken)        as                     damage_taken,
			   sum(mpc.playtime)            as                     playtime,
			   sum(mpc.captures)            as                     captures,
			   sum(mpc.captures_blocked)    as                     captures_blocked,
			   sum(mpc.dominated)           as                     dominated,
			   sum(mpc.dominations)         as                     dominations,
			   sum(mpc.revenges)            as                     revenges,
			   sum(mpc.deaths)              as                     deaths,
			   sum(mpc.buildings_destroyed) as                     buildings_destroyed,
			   sum(mpc.healing_taken)       as                     healing_taken,
			   sum(mm.healing)              as                     healing,
			   sum(mm.drops)                as                     drops,
			   sum(mm.charge_uber)          as                     charge_uber,
			   sum(mm.charge_kritz)         as                     charge_kritz,
			   sum(mm.charge_quickfix)      as                     charge_quickfix,
			   sum(mm.charge_vacc)          as                     charge_vacc
		
		FROM match_player mp
				 LEFT JOIN match s on s.match_id = mp.match_id
				 LEFT JOIN match_player_class mpc on mp.match_player_id = mpc.match_player_id
				 LEFT JOIN match_medic mm on mp.match_player_id = mm.match_player_id
		
		WHERE mp.steam_id = $1 AND
			  s.time_start BETWEEN LOCALTIMESTAMP - INTERVAL '1 DAY' and LOCALTIMESTAMP`

	if errQuery := s.
		QueryRow(ctx, query, steamID).
		Scan(&stats.MatchesWon, &stats.MatchesWon, &stats.HealthPacks,
			&stats.Extinguishes, &stats.BuildingBuilt, &stats.Kills, &stats.Assists, &stats.Damage, &stats.DamageTaken,
			&stats.PlayTime, &stats.Captures, &stats.CapturesBlocked, &stats.Dominated, &stats.Dominations, &stats.Revenges,
			&stats.Deaths, &stats.BuildingDestroyed, &stats.HealingTaken, &stats.Healing, &stats.Drops, &stats.ChargesUber,
			&stats.ChargesKritz, &stats.ChargesQuickfix, &stats.ChargesVacc); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	stats.SteamID = steamID

	return nil
}

func (s Stores) Matches(ctx context.Context, opts domain.MatchesQueryOpts) ([]domain.MatchSummary, int64, error) {
	countBuilder := s.
		Builder().
		Select("count(s.match_id) as count").
		From("match s").
		LeftJoin("public.match_player mp on s.match_id = mp.match_id").
		LeftJoin("public.server s on s.server_id = s.server_id")

	builder := s.
		Builder().
		Select(
			"s.match_id",
			"s.server_id",
			"case when mp.team = s.winner then true else false end as winner",
			"s.short_name",
			"s.title",
			"s.map",
			"s.score_blu",
			"s.score_red",
			"s.time_start",
			"s.time_end").
		From("match s").
		LeftJoin("public.match_player mp on s.match_id = mp.match_id AND mp.time_end - mp.time_start > INTERVAL '60' second"). // TODO index?
		LeftJoin("public.server s on s.server_id = s.server_id")

	if opts.Map != "" {
		builder = builder.Where(sq.Eq{"s.map": opts.Map})
		countBuilder = countBuilder.Where(sq.Eq{"s.map": opts.Map})
	}

	if opts.SteamID.Valid() {
		builder = builder.Where(sq.Eq{"mp.steam_id": opts.SteamID.Int64()})
		countBuilder = countBuilder.Where(sq.Eq{"mp.steam_id": opts.SteamID.Int64()})
	}

	builder = opts.QueryFilter.ApplySafeOrder(builder, map[string][]string{
		"":   {"winner"},
		"s.": {"match_id", "server_id", "map", "score_blu", "score_red", "time_start", "time_end"},
	}, "match_id")

	builder = opts.ApplyLimitOffsetDefault(builder)

	count, errCount := getCount(ctx, s, countBuilder)
	if errCount != nil {
		return nil, 0, errCount
	}

	rows, errQuery := s.QueryBuilder(ctx, builder)
	if errQuery != nil {
		return nil, 0, errors.Join(errQuery, ErrQueryMatch)
	}

	defer rows.Close()

	var matches []domain.MatchSummary

	for rows.Next() {
		var summary domain.MatchSummary
		if errScan := rows.Scan(&summary.MatchID, &summary.ServerID, &summary.IsWinner, &summary.ShortName,
			&summary.Title, &summary.MapName, &summary.ScoreBlu, &summary.ScoreRed, &summary.TimeStart,
			&summary.TimeEnd); errScan != nil {
			return nil, 0, errors.Join(errScan, ErrScanResult)
		}

		matches = append(matches, summary)
	}

	// if rows.DBErr() != nil {
	//	 database.log.Error("Matches rows error", zap.Error(rows.DBErr()))
	// }

	return matches, count, nil
}
