package match

import (
	"context"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type matchRepository struct {
	database      database.Database
	persons       domain.PersonUsecase
	notifications domain.NotificationUsecase
	servers       domain.ServersUsecase
	state         domain.StateUsecase
	wm            fp.MutexMap[logparse.Weapon, int]
	events        chan logparse.ServerEvent
	broadcaster   *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]
	matchUUIDMap  fp.MutexMap[int, uuid.UUID]
}

func NewMatchRepository(broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent],
	database database.Database, persons domain.PersonUsecase, servers domain.ServersUsecase, notifications domain.NotificationUsecase,
	state domain.StateUsecase, weaponMap fp.MutexMap[logparse.Weapon, int],
) domain.MatchRepository {
	matchRepo := &matchRepository{
		database:      database,
		persons:       persons,
		servers:       servers,
		notifications: notifications,
		state:         state,
		wm:            weaponMap,
		broadcaster:   broadcaster,
		matchUUIDMap:  fp.NewMutexMap[int, uuid.UUID](),
		events:        make(chan logparse.ServerEvent),
	}

	return matchRepo
}

func (r *matchRepository) GetMatchIDFromServerID(serverID int) (uuid.UUID, bool) {
	return r.matchUUIDMap.Get(serverID)
}

func (r *matchRepository) Matches(ctx context.Context, opts domain.MatchesQueryOpts) ([]domain.MatchSummary, int64, error) {
	countBuilder := r.database.
		Builder().
		Select("count(m.match_id) as count").
		From("match m").
		LeftJoin("public.match_player mp on m.match_id = mp.match_id").
		LeftJoin("public.server s on s.server_id = m.server_id")

	builder := r.database.
		Builder().
		Select(
			"m.match_id",
			"m.server_id",
			"case when mp.team = m.winner then true else false end as winner",
			"s.short_name",
			"m.title",
			"m.map",
			"m.score_blu",
			"m.score_red",
			"m.time_start",
			"m.time_end").
		From("match m").
		LeftJoin("public.match_player mp on m.match_id = mp.match_id AND mp.time_end - mp.time_start > INTERVAL '60' second"). // TODO index?
		LeftJoin("public.server s on s.server_id = m.server_id")

	if opts.Map != "" {
		builder = builder.Where(sq.Eq{"m.map": opts.Map})
		countBuilder = countBuilder.Where(sq.Eq{"m.map": opts.Map})
	}

	if sid, ok := opts.TargetSteamID(); ok {
		builder = builder.Where(sq.Eq{"mp.steam_id": sid})
		countBuilder = countBuilder.Where(sq.Eq{"mp.steam_id": sid})
	}

	builder = opts.ApplySafeOrder(builder, map[string][]string{
		"":   {"winner"},
		"m.": {"match_id", "server_id", "map", "score_blu", "score_red", "time_start", "time_end"},
	}, "match_id")

	builder = opts.ApplyLimitOffsetDefault(builder)

	count, errCount := r.database.GetCount(ctx, nil, countBuilder)
	if errCount != nil {
		return nil, 0, errCount
	}

	rows, errQuery := r.database.QueryBuilder(ctx, nil, builder)
	if errQuery != nil {
		return nil, 0, errors.Join(errQuery, domain.ErrQueryMatch)
	}

	defer rows.Close()

	var matches []domain.MatchSummary

	for rows.Next() {
		var summary domain.MatchSummary
		if errScan := rows.Scan(&summary.MatchID, &summary.ServerID, &summary.IsWinner, &summary.ShortName,
			&summary.Title, &summary.MapName, &summary.ScoreBlu, &summary.ScoreRed, &summary.TimeStart,
			&summary.TimeEnd); errScan != nil {
			return nil, 0, errors.Join(errScan, domain.ErrScanResult)
		}

		matches = append(matches, summary)
	}

	// if rows.DBErr() != nil {
	//	 database.log.Error("Matches rows error", log.ErrAttr(rows.DBErr()))
	// }

	return matches, count, nil
}

func (r *matchRepository) matchGetPlayerClasses(ctx context.Context, matchID uuid.UUID) (map[steamid.SteamID][]domain.MatchPlayerClass, error) {
	const query = `
		SELECT mp.steam_id, c.match_player_class_id, c.match_player_id, c.player_class_id, c.kills, 
		   c.assists, c.deaths, c.playtime, c.dominations, c.dominated, c.revenges, c.damage, c.damage_taken, c.healing_taken,
		   c.captures, c.captures_blocked, c.buildings_destroyed
		FROM match_player_class c
		LEFT JOIN match_player mp on mp.match_player_id = c.match_player_id
		WHERE mp.match_id = $1`

	rows, errQuery := r.database.Query(ctx, nil, query, matchID)
	if errQuery != nil {
		return nil, r.database.DBErr(errQuery)
	}

	defer rows.Close()

	results := map[steamid.SteamID][]domain.MatchPlayerClass{}

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
			return nil, r.database.DBErr(errScan)
		}

		sid := steamid.New(steamID)

		res, found := results[sid]
		if !found {
			res = []domain.MatchPlayerClass{}
		}

		results[sid] = append(res, stats)
	}

	if errRows := rows.Err(); errRows != nil {
		return nil, r.database.DBErr(errRows)
	}

	return results, nil
}

func (r *matchRepository) matchGetPlayerWeapons(ctx context.Context, matchID uuid.UUID) (map[steamid.SteamID][]domain.MatchPlayerWeapon, error) {
	const query = `
		SELECT mp.steam_id, mw.weapon_id, w.name, w.key,  mw.kills, mw.damage, mw.shots, mw.hits, mw.backstabs, mw.headshots, mw.airshots
		FROM match m
		LEFT JOIN match_player mp on m.match_id = mp.match_id
		LEFT JOIN match_weapon mw on mp.match_player_id = mw.match_player_id
		LEFT JOIN weapon w on w.weapon_id = mw.weapon_id
		WHERE m.match_id = $1 and mw.weapon_id is not null
		ORDER BY mw.kills DESC`

	results := map[steamid.SteamID][]domain.MatchPlayerWeapon{}

	rows, errRows := r.database.Query(ctx, nil, query, matchID)
	if errRows != nil {
		return nil, r.database.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			steamID int64
			mpw     domain.MatchPlayerWeapon
		)

		if errScan := rows.
			Scan(&steamID, &mpw.WeaponID, &mpw.Name, &mpw.Key, &mpw.Kills, &mpw.Damage, &mpw.Shots,
				&mpw.Hits, &mpw.Backstabs, &mpw.Headshots, &mpw.Airshots); errScan != nil {
			return nil, r.database.DBErr(errScan)
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

func (r *matchRepository) matchGetPlayerKillstreak(ctx context.Context, matchID uuid.UUID) (map[steamid.SteamID][]domain.MatchPlayerKillstreak, error) {
	const query = `
		SELECT mp.steam_id, k.match_player_id, k.player_class_id, k.killstreak, k.duration
		FROM match_player_killstreak k
		LEFT JOIN match_player mp on mp.match_player_id = k.match_player_id
		WHERE mp.match_id = $1`

	rows, errRows := r.database.Query(ctx, nil, query, matchID)
	if errRows != nil {
		return nil, r.database.DBErr(errRows)
	}

	defer rows.Close()

	results := map[steamid.SteamID][]domain.MatchPlayerKillstreak{}

	for rows.Next() {
		var (
			steamID int64
			stats   domain.MatchPlayerKillstreak
		)

		if errScan := rows.
			Scan(&steamID, &stats.MatchPlayerID, &stats.PlayerClass, &stats.Killstreak, &stats.Duration); errScan != nil {
			return nil, r.database.DBErr(errScan)
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

func (r *matchRepository) matchGetPlayers(ctx context.Context, matchID uuid.UUID) ([]*domain.MatchPlayer, error) {
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

	playerRows, errPlayer := r.database.Query(ctx, nil, queryPlayer, matchID)
	if errPlayer != nil {
		if errors.Is(errPlayer, domain.ErrNoResult) {
			return []*domain.MatchPlayer{}, nil
		}

		return nil, errors.Join(errPlayer, domain.ErrQueryPlayers)
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
			return nil, errors.Join(errPlayer, domain.ErrScanResult)
		}

		mpSum.SteamID = steamid.New(steamID)
		players = append(players, mpSum)
	}

	return players, nil
}

func (r *matchRepository) matchGetMedics(ctx context.Context, matchID uuid.UUID) (map[steamid.SteamID]domain.MatchHealer, error) {
	const query = `
		SELECT m.match_medic_id,
			   mp.steam_id,
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

	medics := map[steamid.SteamID]domain.MatchHealer{}

	medicRows, errMedics := r.database.Query(ctx, nil, query, matchID)
	if errMedics != nil {
		if errors.Is(errMedics, domain.ErrNoResult) {
			return medics, nil
		}

		return nil, errors.Join(errMedics, domain.ErrQueryPlayers)
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
			return nil, errors.Join(errMedics, domain.ErrScanResult)
		}

		sid := steamid.New(steamID)

		medics[sid] = mpSum
	}

	if medicRows.Err() != nil {
		return medics, errors.Join(medicRows.Err(), domain.ErrRowResults)
	}

	return medics, nil
}

func (r *matchRepository) matchGetChat(ctx context.Context, matchID uuid.UUID) (domain.PersonMessages, error) {
	const query = `
		SELECT x.*, coalesce(f.person_message_filter_id, 0)
		FROM (SELECT c.person_message_id,
					 c.steam_id,
					 c.server_id,
					 c.body,
					 c.persona_name,
					 c.team,
					 c.created_on,
					 c.match_id
			  FROM person_messages c
		
			  WHERE c.match_id = $1
			  GROUP BY c.person_message_id) x
         LEFT JOIN person_messages_filter f on x.person_message_id = f.person_message_id
		`

	messages := domain.PersonMessages{}

	chatRows, errQuery := r.database.Query(ctx, nil, query, matchID)
	if errQuery != nil {
		if errors.Is(errQuery, domain.ErrNoResult) {
			return messages, nil
		}

		return nil, errors.Join(errQuery, domain.ErrChatQuery)
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
				&msg.MatchID, &msg.AutoFilterFlagged); errRow != nil {
			return nil, errors.Join(errQuery, domain.ErrScanResult)
		}

		msg.SteamID = steamid.New(steamID)
		messages = append(messages, msg)
	}

	if chatRows.Err() != nil {
		return messages, errors.Join(chatRows.Err(), domain.ErrRowResults)
	}

	return messages, nil
}

func (r *matchRepository) MatchGetByID(ctx context.Context, matchID uuid.UUID, match *domain.MatchResult) error {
	const query = `
		SELECT match_id, server_id, map, title, score_red, score_blu, time_red, time_blu, time_start, time_end, winner
		FROM match WHERE match_id = $1`

	if errMatch := r.database.
		QueryRow(ctx, nil, query, matchID).
		Scan(&match.MatchID, &match.ServerID, &match.MapName, &match.Title,
			&match.TeamScores.Red, &match.TeamScores.Blu, &match.TeamScores.BluTime, &match.TeamScores.BluTime,
			&match.TimeStart, &match.TimeEnd, &match.Winner); errMatch != nil {
		return errors.Join(errMatch, domain.ErrMatchQuery)
	}

	playerStats, errPlayerStats := r.matchGetPlayers(ctx, matchID)
	if errPlayerStats != nil {
		return errors.Join(errPlayerStats, domain.ErrQueryPlayers)
	}

	match.Players = playerStats

	playerClasses, errPlayerClasses := r.matchGetPlayerClasses(ctx, matchID)
	if errPlayerClasses != nil {
		return errors.Join(errPlayerClasses, domain.ErrGetPlayerClasses)
	}

	for _, player := range playerStats {
		if classes, found := playerClasses[player.SteamID]; found {
			player.Classes = classes
		}
	}

	playerKillstreaks, errPlayerKillstreaks := r.matchGetPlayerKillstreak(ctx, matchID)
	if errPlayerKillstreaks != nil {
		return errors.Join(errPlayerKillstreaks, domain.ErrGetPlayerKillstreaks)
	}

	for _, player := range match.Players {
		if killstreaks, found := playerKillstreaks[player.SteamID]; found {
			player.Killstreaks = killstreaks
		}
	}

	medicStats, errMedics := r.matchGetMedics(ctx, matchID)
	if errMedics != nil {
		return errors.Join(errMedics, domain.ErrGetMedicStats)
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

	weaponStats, errWeapons := r.matchGetPlayerWeapons(ctx, matchID)
	if errWeapons != nil {
		return errors.Join(errMedics, domain.ErrGetWeaponStats)
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

	chat, errChat := r.matchGetChat(ctx, matchID)

	if errChat != nil && !errors.Is(errChat, domain.ErrNoResult) {
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

func (r *matchRepository) MatchSave(ctx context.Context, match *logparse.Match, weaponMap fp.MutexMap[logparse.Weapon, int]) error {
	const (
		query = `
		INSERT INTO match (match_id, server_id, map, title, score_red, score_blu, time_red, time_blu, time_start, time_end, winner) 
		VALUES ($1, $2, $3, $4, $5, $6,$7, $8, $9, $10, $11) 
		RETURNING match_id`
	)

	transaction, errTx := r.database.Begin(ctx)
	if errTx != nil {
		return errors.Join(errTx, domain.ErrTxStart)
	}

	if errQuery := transaction.
		QueryRow(ctx, query, match.MatchID, match.ServerID, match.MapName, match.Title,
			match.TeamScores.Red, match.TeamScores.Blu, match.TeamScores.RedTime, match.TeamScores.BluTime,
			match.TimeStart, match.TimeEnd, match.Winner()).
		Scan(&match.MatchID); errQuery != nil {
		if errRollback := transaction.Rollback(ctx); errRollback != nil {
			return errors.Join(errRollback, domain.ErrTxRollback)
		}

		return errors.Join(errQuery, domain.ErrMatchQuery)
	}

	for _, player := range match.PlayerSums {
		if !player.SteamID.Valid() {
			// TODO Why can this happen? stv host?
			continue
		}

		_, errPlayer := r.persons.GetOrCreatePersonBySteamID(ctx, transaction, player.SteamID)
		if errPlayer != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				return errors.Join(errRollback, domain.ErrTxRollback)
			}

			return errors.Join(errPlayer, domain.ErrGetPerson)
		}

		if errSave := r.saveMatchPlayerStats(ctx, transaction, match, player); errSave != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				return errors.Join(errRollback, domain.ErrTxRollback)
			}

			return errSave
		}

		if errSave := r.saveMatchWeaponStats(ctx, transaction, player, weaponMap); errSave != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				return errors.Join(errRollback, domain.ErrTxRollback)
			}

			return errSave
		}

		if errSave := r.saveMatchPlayerClassStats(ctx, transaction, player); errSave != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				return errors.Join(errRollback, domain.ErrTxRollback)
			}

			return errSave
		}

		if errSave := r.saveMatchKillstreakStats(ctx, transaction, player); errSave != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				return errors.Join(errRollback, domain.ErrTxRollback)
			}

			return errSave
		}

		if player.HealingStats != nil && player.HealingStats.Healing >= domain.MinMedicHealing {
			if errSave := r.saveMatchMedicStats(ctx, transaction, player.MatchPlayerID, player.HealingStats); errSave != nil {
				if errRollback := transaction.Rollback(ctx); errRollback != nil {
					return errors.Join(errRollback, domain.ErrTxRollback)
				}

				return errSave
			}
		}
	}

	if errCommit := transaction.Commit(ctx); errCommit != nil {
		return errors.Join(errCommit, domain.ErrTxCommit)
	}

	return nil
}

func (r *matchRepository) saveMatchPlayerStats(ctx context.Context, transaction pgx.Tx, match *logparse.Match, stats *logparse.PlayerStats) error {
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
		return errors.Join(errPlayerExec, domain.ErrSavePlayerStats)
	}

	return nil
}

func (r *matchRepository) saveMatchMedicStats(ctx context.Context, transaction pgx.Tx, matchPlayerID int64, stats *logparse.HealingStats) error {
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
		return errors.Join(errMedExec, domain.ErrSaveMedicStats)
	}

	return nil
}

func (r *matchRepository) saveMatchWeaponStats(ctx context.Context, transaction pgx.Tx, player *logparse.PlayerStats, weaponMap fp.MutexMap[logparse.Weapon, int]) error {
	const query = `
		INSERT INTO match_weapon (match_player_id, weapon_id, kills, damage, shots, hits, backstabs, headshots, airshots) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) 
		RETURNING player_weapon_id`

	for weapon, info := range player.WeaponInfo {
		weaponID, found := weaponMap.Get(weapon)
		if !found {
			// database.log.Error("Unknown weapon", slog.String("weapon", string(weapon)))
			continue
		}

		if _, errWeapon := transaction.
			Exec(ctx, query, player.MatchPlayerID, weaponID, info.Kills, info.Damage, info.Shots, info.Hits,
				info.BackStabs, info.Headshots, info.Airshots); errWeapon != nil {
			return errors.Join(errWeapon, domain.ErrSaveWeaponStats)
		}
	}

	return nil
}

func (r *matchRepository) saveMatchPlayerClassStats(ctx context.Context, transaction pgx.Tx, player *logparse.PlayerStats) error {
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
			return errors.Join(errWeapon, domain.ErrSaveClassStats)
		}
	}

	return nil
}

func (r *matchRepository) saveMatchKillstreakStats(ctx context.Context, transaction pgx.Tx, player *logparse.PlayerStats) error {
	const query = `
		INSERT INTO match_player_killstreak (match_player_id, player_class_id, killstreak, duration) 
		VALUES ($1, $2, $3, $4)`

	for class, stats := range player.KillStreaks {
		if _, errWeapon := transaction.
			Exec(ctx, query, player.MatchPlayerID, class, stats.Killstreak, stats.Duration); errWeapon != nil {
			return errors.Join(errWeapon, domain.ErrSaveKillstreakStats)
		}
	}

	return nil
}

func (r *matchRepository) StatsPlayerClass(ctx context.Context, sid64 steamid.SteamID) (domain.PlayerClassStatsCollection, error) {
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

	rows, errQuery := r.database.Query(ctx, nil, query, sid64.Int64())
	if errQuery != nil {
		return nil, r.database.DBErr(errQuery)
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
			return nil, r.database.DBErr(errScan)
		}

		class.ClassName = class.Class.String()
		stats = append(stats, class)
	}

	return stats, nil
}

func (r *matchRepository) StatsPlayerWeapons(ctx context.Context, sid64 steamid.SteamID) ([]domain.PlayerWeaponStats, error) {
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

	rows, errQuery := r.database.Query(ctx, nil, query, sid64.Int64())
	if errQuery != nil {
		return nil, r.database.DBErr(errQuery)
	}

	defer rows.Close()

	var stats []domain.PlayerWeaponStats

	for rows.Next() {
		var class domain.PlayerWeaponStats
		if errScan := rows.
			Scan(&class.Weapon, &class.WeaponName, &class.Kills, &class.Damage, &class.Shots, &class.Hits,
				&class.Backstabs, &class.Headshots, &class.Airshots); errScan != nil {
			return nil, r.database.DBErr(errScan)
		}

		stats = append(stats, class)
	}

	return stats, nil
}

func (r *matchRepository) StatsPlayerKillstreaks(ctx context.Context, sid64 steamid.SteamID) ([]domain.PlayerKillstreakStats, error) {
	const query = `
		SELECT k.player_class_id,
			   SUM(k.killstreak) as killstreak,
			   SUM(k.duration)   as duration,
			   m.time_start
		FROM match_player p
				 LEFT JOIN match_player_killstreak k on p.match_player_id = k.match_player_id
				 LEFT JOIN match_player mp on mp.match_player_id = k.match_player_id
				 LEFT JOIN match m on p.match_id = m.match_id
		WHERE p.steam_id = $1
		  AND k.player_class_id IS NOT NULL
		GROUP BY k.match_killstreak_id, m.time_start, k.player_class_id
		ORDER BY killstreak DESC
		LIMIT 10;`

	rows, errQuery := r.database.Query(ctx, nil, query, sid64.Int64())
	if errQuery != nil {
		return nil, r.database.DBErr(errQuery)
	}

	defer rows.Close()

	var stats []domain.PlayerKillstreakStats

	for rows.Next() {
		var class domain.PlayerKillstreakStats
		if errScan := rows.
			Scan(&class.Class, &class.Kills, &class.Duration, &class.CreatedOn); errScan != nil {
			return nil, r.database.DBErr(errScan)
		}

		class.ClassName = class.Class.String()
		stats = append(stats, class)
	}

	return stats, nil
}

func (r *matchRepository) StatsPlayerMedic(ctx context.Context, sid64 steamid.SteamID) ([]domain.PlayerMedicStats, error) {
	const query = `
		SELECT coalesce(SUM(m.healing), 0)                as healing,
			   coalesce(SUM(m.drops), 0)                  as drops,
			   coalesce(SUM(m.near_full_charge_death), 0) as near_full_charge_death,
			   coalesce(AVG(m.avg_uber_length), 0)        as avg_uber_length,
			   coalesce(SUM(m.charge_uber), 0)            as charge_uber,
			   coalesce(SUM(m.charge_kritz), 0)           as charge_kritz,
			   coalesce(SUM(m.charge_vacc), 0)            as charge_vacc,
			   coalesce(SUM(m.charge_quickfix), 0)        as charge_quickfix
		FROM match_player p
		LEFT JOIN match_medic m on p.match_player_id = m.match_player_id
		WHERE p.steam_id = $1
		GROUP BY p.steam_id`

	rows, errQuery := r.database.Query(ctx, nil, query, sid64.Int64())
	if errQuery != nil {
		return nil, r.database.DBErr(errQuery)
	}

	defer rows.Close()

	var stats []domain.PlayerMedicStats

	for rows.Next() {
		var class domain.PlayerMedicStats
		if errScan := rows.
			Scan(&class.Healing, &class.Drops, &class.NearFullChargeDeath, &class.AvgUberLength,
				&class.ChargesUber, &class.ChargesKritz, &class.ChargesVacc, &class.ChargesQuickfix); errScan != nil {
			return nil, r.database.DBErr(errScan)
		}

		stats = append(stats, class)
	}

	return stats, nil
}

func (r *matchRepository) PlayerStats(ctx context.Context, steamID steamid.SteamID, stats *domain.PlayerStats) error {
	const query = `
		SELECT count(m.match_id)            as                     matches,
			   sum(case when mp.team = m.winner then 1 else 0 end) wins,
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
				 LEFT JOIN match m on m.match_id = mp.match_id
				 LEFT JOIN match_player_class mpc on mp.match_player_id = mpc.match_player_id
				 LEFT JOIN match_medic mm on mp.match_player_id = mm.match_player_id
		
		WHERE mp.steam_id = $1 AND
			  m.time_start BETWEEN LOCALTIMESTAMP - INTERVAL '1 DAY' and LOCALTIMESTAMP`

	if errQuery := r.database.
		QueryRow(ctx, nil, query, steamID).
		Scan(&stats.MatchesWon, &stats.MatchesWon, &stats.HealthPacks,
			&stats.Extinguishes, &stats.BuildingBuilt, &stats.Kills, &stats.Assists, &stats.Damage, &stats.DamageTaken,
			&stats.PlayTime, &stats.Captures, &stats.CapturesBlocked, &stats.Dominated, &stats.Dominations, &stats.Revenges,
			&stats.Deaths, &stats.BuildingDestroyed, &stats.HealingTaken, &stats.Healing, &stats.Drops, &stats.ChargesUber,
			&stats.ChargesKritz, &stats.ChargesQuickfix, &stats.ChargesVacc); errQuery != nil {
		return r.database.DBErr(errQuery)
	}

	stats.SteamID = steamID

	return nil
}

func (r *matchRepository) WeaponsOverall(ctx context.Context) ([]domain.WeaponsOverallResult, error) {
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

	rows, errQuery := r.database.Query(ctx, nil, query)
	if errQuery != nil {
		return nil, r.database.DBErr(errQuery)
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
			return nil, r.database.DBErr(errScan)
		}

		results = append(results, wor)
	}

	return results, nil
}

func (r *matchRepository) WeaponsOverallTopPlayers(ctx context.Context, weaponID int) ([]domain.PlayerWeaponResult, error) {
	rows, errQuery := r.database.QueryBuilder(ctx, nil, r.database.
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
		return nil, r.database.DBErr(errQuery)
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
			return nil, r.database.DBErr(errScan)
		}

		pwr.SteamID = steamid.New(sid64)
		results = append(results, pwr)
	}

	return results, nil
}

func (r *matchRepository) WeaponsOverallByPlayer(ctx context.Context, steamID steamid.SteamID) ([]domain.WeaponsOverallResult, error) {
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

	rows, errQuery := r.database.Query(ctx, nil, query, steamID.Int64())
	if errQuery != nil {
		return nil, r.database.DBErr(errQuery)
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
			return nil, r.database.DBErr(errScan)
		}

		results = append(results, wor)
	}

	return results, nil
}

func (r *matchRepository) PlayersOverallByKills(ctx context.Context, count int) ([]domain.PlayerWeaponResult, error) {
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

	rows, errQuery := r.database.Query(ctx, nil, query, count)
	if errQuery != nil {
		return nil, r.database.DBErr(errQuery)
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
			return nil, r.database.DBErr(errScan)
		}

		wor.SteamID = steamid.New(sid64)
		results = append(results, wor)
	}

	return results, nil
}

func (r *matchRepository) HealersOverallByHealing(ctx context.Context, count int) ([]domain.HealingOverallResult, error) {
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
            case c.playtime WHEN 0 THEN 0 ELSE coalesce(h.healing::float / (c.playtime::float / 60), 0) END as hpm,
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

	rows, errQuery := r.database.Query(ctx, nil, query, count)
	if errQuery != nil {
		return nil, r.database.DBErr(errQuery)
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
			return nil, r.database.DBErr(errScan)
		}

		wor.SteamID = steamid.New(sid64)
		results = append(results, wor)
	}

	return results, nil
}

func (r *matchRepository) PlayerOverallClassStats(ctx context.Context, steamID steamid.SteamID) ([]domain.PlayerClassOverallResult, error) {
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

	rows, errQuery := r.database.Query(ctx, nil, query, steamID.Int64())
	if errQuery != nil {
		return nil, r.database.DBErr(errQuery)
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
			return nil, r.database.DBErr(errScan)
		}

		results = append(results, wor)
	}

	return results, nil
}

func (r *matchRepository) PlayerOverallStats(ctx context.Context, steamID steamid.SteamID, por *domain.PlayerOverallResult) error {
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

	if errQuery := r.database.
		QueryRow(ctx, nil, query, steamID.Int64()).Scan(
		&por.Healing, &por.Drops, &por.NearFullChargeDeath, &por.AvgUberLen, &por.MajorAdvLost, &por.BiggestAdvLost,
		&por.ChargesUber, &por.ChargesKritz, &por.ChargesVacc, &por.ChargesQuickfix, &por.Buildings, &por.Extinguishes,
		&por.HealthPacks, &por.KA, &por.Kills, &por.Assists, &por.Deaths, &por.KD, &por.KAD, &por.DPM, &por.Shots, &por.Hits, &por.Accuracy, &por.Airshots, &por.Backstabs,
		&por.Headshots, &por.Playtime, &por.Dominations, &por.Dominated, &por.Revenges, &por.Damage, &por.DamageTaken,
		&por.Captures, &por.CapturesBlocked, &por.BuildingsDestroyed, &por.HealingTaken, &por.Wins, &por.Matches, &por.WinRate,
	); errQuery != nil {
		return r.database.DBErr(errQuery)
	}

	return nil
}

func (r *matchRepository) GetMapUsageStats(ctx context.Context) ([]domain.MapUseDetail, error) {
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

	rows, errQuery := r.database.Query(ctx, nil, query)
	if errQuery != nil {
		return nil, r.database.DBErr(errQuery)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			mud     domain.MapUseDetail
			seconds int64
		)

		if errScan := rows.Scan(&mud.Map, &seconds, &mud.Percent); errScan != nil {
			return nil, r.database.DBErr(errScan)
		}

		mud.Playtime = seconds

		details = append(details, mud)
	}

	if rows.Err() != nil {
		return nil, errors.Join(rows.Err(), domain.ErrRowResults)
	}

	return details, nil
}

func (r *matchRepository) LoadWeapons(ctx context.Context, weaponMap fp.MutexMap[logparse.Weapon, int]) error {
	for weapon, name := range logparse.NewWeaponParser().NameMap() {
		var newWeapon domain.Weapon
		if errWeapon := r.GetWeaponByKey(ctx, weapon, &newWeapon); errWeapon != nil {
			if !errors.Is(errWeapon, domain.ErrNoResult) {
				return errWeapon
			}

			newWeapon.Key = weapon
			newWeapon.Name = name

			if errSave := r.SaveWeapon(ctx, &newWeapon); errSave != nil {
				return r.database.DBErr(errSave)
			}
		}

		weaponMap.Set(weapon, newWeapon.WeaponID)
	}

	return nil
}

func (r *matchRepository) GetWeaponByKey(ctx context.Context, key logparse.Weapon, weapon *domain.Weapon) error {
	row, errRow := r.database.QueryRowBuilder(ctx, nil, r.database.
		Builder().
		Select("weapon_id", "key", "name").
		From("weapon").
		Where(sq.Eq{"key": key}))
	if errRow != nil {
		return r.database.DBErr(errRow)
	}

	return r.database.DBErr(row.Scan(&weapon.WeaponID, &weapon.Key, &weapon.Name))
}

func (r *matchRepository) GetWeaponByID(ctx context.Context, weaponID int, weapon *domain.Weapon) error {
	row, errRow := r.database.QueryRowBuilder(ctx, nil, r.database.
		Builder().
		Select("weapon_id", "key", "name").
		From("weapon").Where(sq.Eq{"weapon_id": weaponID}))
	if errRow != nil {
		return r.database.DBErr(errRow)
	}

	return r.database.DBErr(row.Scan(&weapon.WeaponID, &weapon.Key, &weapon.Name))
}

func (r *matchRepository) SaveWeapon(ctx context.Context, weapon *domain.Weapon) error {
	if weapon.WeaponID > 0 {
		return r.database.DBErr(r.database.ExecUpdateBuilder(ctx, nil, r.database.
			Builder().
			Update("weapon").
			Set("key", weapon.Key).
			Set("name", weapon.Name).
			Where(sq.Eq{"weapon_id": weapon.WeaponID})))
	}

	const wq = `INSERT INTO weapon (key, name) VALUES ($1, $2) RETURNING weapon_id`

	if errSave := r.database.
		QueryRow(ctx, nil, wq, weapon.Key, weapon.Name).
		Scan(&weapon.WeaponID); errSave != nil {
		return errors.Join(errSave, domain.ErrFailedWeapon)
	}

	return nil
}

func (r *matchRepository) Weapons(ctx context.Context) ([]domain.Weapon, error) {
	rows, errRows := r.database.QueryBuilder(ctx, nil, r.database.
		Builder().
		Select("weapon_id", "key", "name").
		From("weapon"))
	if errRows != nil {
		return nil, r.database.DBErr(errRows)
	}

	defer rows.Close()

	var weapons []domain.Weapon

	for rows.Next() {
		var weapon domain.Weapon
		if errScan := rows.Scan(&weapon.WeaponID, &weapon.Name); errScan != nil {
			return nil, errors.Join(errScan, domain.ErrScanResult)
		}

		weapons = append(weapons, weapon)
	}

	if errRow := rows.Err(); errRow != nil {
		return nil, errors.Join(errRow, domain.ErrRowResults)
	}

	return weapons, nil
}
