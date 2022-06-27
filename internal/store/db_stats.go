package store

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (database *pgStore) MatchSave(ctx context.Context, match *model.Match) error {
	for _, p := range match.PlayerSums {
		var player model.Person
		if errPlayer := database.GetOrCreatePersonBySteamID(ctx, p.SteamId, &player); errPlayer != nil {
			return errors.Wrapf(errPlayer, "Failed to create person")
		}
	}
	const q = `INSERT INTO match (server_id, map, created_on, title) VALUES ($1, $2, $3, $4) RETURNING match_id`
	if errMatch := database.QueryRow(ctx, q, match.ServerId, match.MapName, match.CreatedOn, match.Title).
		Scan(&match.MatchID); errMatch != nil {
		return errors.Wrapf(errMatch, "Failed to setup match")
	}
	const pq = `INSERT INTO match_player (
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
	for _, s := range match.PlayerSums {
		endTime := &match.CreatedOn
		if s.TimeEnd != nil {
			// Use match end time
			endTime = s.TimeEnd
		}
		if errPlayerExec := database.QueryRow(ctx, pq, match.MatchID, s.SteamId, s.Team,
			s.TimeStart, endTime, s.Kills, s.Assists, s.Deaths, s.Dominations, s.Dominated,
			s.Revenges, s.Damage, s.DamageTaken, s.Healing, s.HealingTaken, s.HealthPacks,
			s.BackStabs, s.HeadShots, s.Airshots, s.Captures, s.Shots, s.Extinguishes,
			s.Hits, s.BuildingDestroyed, s.BuildingDestroyed,
		).Scan(&s.MatchPlayerSumID); errPlayerExec != nil {
			return errors.Wrapf(errPlayerExec, "Failed to write player sum")
		}
	}

	const mq = `INSERT INTO match_medic (
            match_id, steam_id, healing, charges, drops, avg_time_to_build, avg_time_before_use, 
            near_full_charge_death, avg_uber_length, death_after_charge, major_adv_lost, biggest_adv_lost) 
            VALUES ($1, $2, $3, $4, $5,$6, $7, $8, $9, $10,$11, $12) RETURNING match_medic_id`

	for _, s := range match.MedicSums {
		charges := 0
		for _, mg := range s.Charges {
			charges += mg
		}
		if errMedExec := database.QueryRow(ctx, mq, match.MatchID, s.SteamId, s.Healing, charges, s.Drops, s.AvgTimeToBuild,
			s.AvgTimeBeforeUse, s.NearFullChargeDeath, s.AvgUberLength, s.DeathAfterCharge, s.MajorAdvLost,
			s.BiggestAdvLost).Scan(&s.MatchMedicId); errMedExec != nil {
			return errors.Wrapf(errMedExec, "Failed to write medic sum")
		}
	}

	const tq = `INSERT INTO match_team (
		match_id, team, kills, damage, charges, drops, caps, mid_fights
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	RETURNING match_team_id`
	// FIXME team value unset
	for i, s := range match.TeamSums {
		if errTeamExec := database.QueryRow(ctx, tq, match.MatchID, i+1, s.Kills, s.Damage, s.Charges, s.Drops,
			s.Caps, s.MidFights).Scan(&s.MatchTeamId); errTeamExec != nil {
			return errors.Wrapf(errTeamExec, "Failed to write team sum")
		}
	}
	return nil
}

func (database *pgStore) MatchGetById(ctx context.Context, matchId int) (*model.Match, error) {
	m := model.NewMatch()
	m.MatchID = matchId
	const qm = `SELECT server_id, map, title, created_on  FROM match WHERE match_id = $1`
	if errMatch := database.QueryRow(ctx, qm, matchId).Scan(&m.ServerId,
		&m.MapName, &m.Title, &m.CreatedOn); errMatch != nil {
		return nil, errors.Wrapf(errMatch, "Failed to load root match")
	}
	const qp = `
		SELECT 
		    match_player_id, steam_id, team, time_start, time_end, kills, assists,
       		deaths, dominations, dominated, revenges, damage, damage_taken, healing, healing_taken, health_packs, 
       		backstabs, headshots, airshots, captures, shots, extinguishes, hits, buildings, 
       		buildings_destroyed 
		FROM 
		    match_player 
		WHERE 
		    match_id = $1`
	playerRows, errPlayer := database.Query(ctx, qp, matchId)
	if errPlayer != nil {
		return nil, errors.Wrapf(errPlayer, "Failed to query match players")
	}
	defer playerRows.Close()
	for playerRows.Next() {
		s := model.MatchPlayerSum{MatchPlayerSumID: matchId}
		if errRow := playerRows.Scan(&s.MatchPlayerSumID, &s.SteamId, &s.Team, &s.TimeStart, &s.TimeEnd, &s.Kills, &s.Assists,
			&s.Deaths, &s.Dominations, &s.Dominated, &s.Revenges, &s.Damage, &s.DamageTaken, &s.Healing, &s.HealingTaken,
			&s.HealthPacks, &s.BackStabs, &s.HeadShots, &s.Airshots, &s.Captures, &s.Shots, &s.Extinguishes, &s.Hits,
			&s.BuildingBuilt, &s.BuildingDestroyed); errRow != nil {
			return nil, errors.Wrapf(errPlayer, "Failed to scan match players")
		}
		m.PlayerSums = append(m.PlayerSums, &s)
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
	medicRows, errMedQuery := database.Query(ctx, qMed, matchId)
	if errMedQuery != nil && !errors.Is(errMedQuery, ErrNoResult) {
		return nil, errors.Wrapf(errMedQuery, "Failed to query match medics")
	}
	defer medicRows.Close()
	for medicRows.Next() {
		ms := model.MatchMedicSum{MatchId: matchId, Charges: map[logparse.Medigun]int{
			logparse.Uber:       0,
			logparse.Kritzkrieg: 0,
			logparse.Vaccinator: 0,
			logparse.QuickFix:   0,
		}}
		charges := 0
		if errRow := medicRows.Scan(&ms.MatchMedicId, &ms.SteamId, &ms.Healing, &charges, &ms.Drops,
			&ms.AvgTimeToBuild, &ms.AvgTimeBeforeUse, &ms.NearFullChargeDeath, &ms.AvgUberLength, &ms.DeathAfterCharge,
			&ms.MajorAdvLost, &ms.BiggestAdvLost); errRow != nil {
			return nil, errors.Wrapf(errMedQuery, "Failed to scan match medics")
		}
		// FIXME all charges are counted as uber for now
		ms.Charges[logparse.Uber] = charges
		m.MedicSums = append(m.MedicSums, &ms)
	}

	const qTeam = `
		SELECT 
		    match_team_id, team, kills, damage, charges, drops, caps, mid_fights 
		FROM 
		    match_team 
		WHERE 
		    match_id = $1`
	teamRows, errTeamQuery := database.Query(ctx, qTeam, matchId)
	if errTeamQuery != nil && !errors.Is(errTeamQuery, ErrNoResult) {
		return nil, errors.Wrapf(errMedQuery, "Failed to query match medics")
	}
	defer teamRows.Close()
	for teamRows.Next() {
		ts := model.MatchTeamSum{MatchId: matchId}
		if errRow := teamRows.Scan(&ts.MatchTeamId, &ts.Team, &ts.Kills, &ts.Damage, &ts.Charges,
			&ts.Drops, &ts.Caps, &ts.MidFights); errRow != nil {
			return nil, errors.Wrapf(errRow, "Failed to scan match medics")
		}
		m.TeamSums = append(m.TeamSums, &ts)
	}
	return &m, nil
}

func (database *pgStore) GetStats(ctx context.Context, stats *model.Stats) error {
	const q = `
	SELECT 
		(SELECT COUNT(ban_id) FROM ban) as bans_total,
		(SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 DAY')) as bans_day,
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 DAY')) as bans_week,
		(SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 MONTH')) as bans_month, 
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '3 MONTH')) as bans_3month,
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '6 MONTH')) as bans_6month,
	    (SELECT COUNT(ban_id) FROM ban WHERE created_on >= (now() - INTERVAL '1 YEAR')) as bans_year,
		(SELECT COUNT(net_id) FROM ban_net) as bans_cidr, 
		(SELECT COUNT(appeal_id) FROM ban_appeal WHERE appeal_state = 0) as appeals_open,
		(SELECT COUNT(appeal_id) FROM ban_appeal WHERE appeal_state = 1 OR appeal_state = 2) as appeals_closed,
		(SELECT COUNT(word_id) FROM filtered_word) as filtered_words,
		(SELECT COUNT(server_id) FROM server) as servers_total`
	if errQuery := database.conn.QueryRow(ctx, q).
		Scan(&stats.BansTotal, &stats.BansDay, &stats.BansWeek, &stats.BansMonth,
			&stats.Bans3Month, &stats.Bans6Month, &stats.BansYear, &stats.BansCIDRTotal,
			&stats.AppealsOpen, &stats.AppealsClosed, &stats.FilteredWords, &stats.ServersTotal,
		); errQuery != nil {
		log.Errorf("Failed to fetch stats: %v", errQuery)
		return Err(errQuery)
	}
	return nil

}
