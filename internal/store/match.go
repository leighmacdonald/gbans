package store

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/pkg/fp"
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
	MatchPlayerID int64 `json:"match_player_id"`
	CommonPlayerStats
	Team      logparse.Team `json:"team"`
	TimeStart time.Time     `json:"time_start"`
	TimeEnd   time.Time     `json:"time_end"`

	MedicStats  *MatchHealer            `json:"medic_stats"`
	Classes     []MatchPlayerClass      `json:"classes"`
	Killstreaks []MatchPlayerKillstreak `json:"killstreaks"`
	Weapons     []MatchPlayerWeapon     `json:"weapons"`
}

func (player MatchPlayer) BiggestKillstreak() *MatchPlayerKillstreak {
	var biggest *MatchPlayerKillstreak

	for _, killstreakVal := range player.Killstreaks {
		killstreak := killstreakVal
		if biggest == nil || killstreak.Killstreak > biggest.Killstreak {
			biggest = &killstreak
		}
	}

	return biggest
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
	return fp.Max[int](int(float64(player.Damage)/player.TimeEnd.Sub(player.TimeStart).Minutes()), 0)
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
	MatchID    uuid.UUID           `json:"match_id"`
	ServerID   int                 `json:"server_id"`
	Title      string              `json:"title"`
	MapName    string              `json:"map_name"`
	TeamScores logparse.TeamScores `json:"team_scores"`
	TimeStart  time.Time           `json:"time_start"`
	TimeEnd    time.Time           `json:"time_end"`
	Winner     logparse.Team       `json:"winner"`
	Players    []*MatchPlayer      `json:"players"`
	Chat       PersonMessages      `json:"chat"`
}

func (match *MatchResult) TopPlayers() []*MatchPlayer {
	players := match.Players

	sort.SliceStable(players, func(i, j int) bool {
		return players[i].Kills > players[j].Kills
	})

	return players
}

func (match *MatchResult) TopKillstreaks(count int) []*MatchPlayer {
	var killStreakPlayers []*MatchPlayer

	for _, player := range match.Players {
		if killStreak := player.BiggestKillstreak(); killStreak != nil {
			killStreakPlayers = append(killStreakPlayers, player)
		}
	}

	sort.SliceStable(killStreakPlayers, func(i, j int) bool {
		return killStreakPlayers[i].BiggestKillstreak().Killstreak > killStreakPlayers[j].BiggestKillstreak().Killstreak
	})

	if len(killStreakPlayers) > count {
		return killStreakPlayers[0:count]
	}

	return killStreakPlayers
}

func (match *MatchResult) Healers() []*MatchPlayer {
	var healers []*MatchPlayer

	for _, player := range match.Players {
		if player.MedicStats != nil {
			healers = append(healers, player)
		}
	}

	sort.SliceStable(healers, func(i, j int) bool {
		return healers[i].MedicStats.Healing > healers[j].MedicStats.Healing
	})

	return healers
}

func (db *Store) matchGetPlayerClasses(ctx context.Context, matchID uuid.UUID) (map[steamid.SID64][]MatchPlayerClass, error) {
	const query = `
		SELECT mp.steam_id, c.match_player_class_id, c.match_player_id, c.player_class_id, c.kills, 
		   c.assists, c.deaths, c.playtime, c.dominations, c.dominated, c.revenges, c.damage, c.damage_taken, c.healing_taken,
		   c.captures, c.captures_blocked, c.buildings_destroyed
		FROM match_player_class c
		LEFT JOIN match_player mp on mp.match_player_id = c.match_player_id
		WHERE mp.match_id = $1`

	rows, errQuery := db.Query(ctx, query, matchID)
	if errQuery != nil {
		return nil, Err(errQuery)
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
				&stats.Revenges, &stats.Damage, &stats.DamageTaken, &stats.HealingTaken, &stats.Captures,
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

	if errRows := rows.Err(); errRows != nil {
		return nil, Err(errRows)
	}

	return results, nil
}

type MatchPlayerWeapon struct {
	Weapon
	Kills     int     `json:"kills"`
	Damage    int     `json:"damage"`
	Shots     int     `json:"shots"`
	Hits      int     `json:"hits"`
	Accuracy  float64 `json:"accuracy"`
	Backstabs int     `json:"backstabs"`
	Headshots int     `json:"headshots"`
	Airshots  int     `json:"airshots"`
}

func (db *Store) matchGetPlayerWeapons(ctx context.Context, matchID uuid.UUID) (map[steamid.SID64][]MatchPlayerWeapon, error) {
	const query = `
		SELECT mp.steam_id, mw.weapon_id, w.name, w.key,  mw.kills, mw.damage, mw.shots, mw.hits, mw.backstabs, mw.headshots, mw.airshots
		FROM match m
		LEFT JOIN match_player mp on m.match_id = mp.match_id
		LEFT JOIN match_weapon mw on mp.match_player_id = mw.match_player_id
		LEFT JOIN weapon w on w.weapon_id = mw.weapon_id
		WHERE m.match_id = $1 and mw.weapon_id is not null
		ORDER BY mw.kills DESC`

	results := map[steamid.SID64][]MatchPlayerWeapon{}

	rows, errRows := db.Query(ctx, query, matchID)
	if errRows != nil {
		return nil, Err(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var (
			steamID int64
			mpw     MatchPlayerWeapon
		)

		if errScan := rows.
			Scan(&steamID, &mpw.WeaponID, &mpw.Weapon.Name, &mpw.Weapon.Key, &mpw.Kills, &mpw.Damage, &mpw.Shots,
				&mpw.Hits, &mpw.Backstabs, &mpw.Headshots, &mpw.Airshots); errScan != nil {
			return nil, Err(errScan)
		}

		sid := steamid.New(steamID)

		res, found := results[sid]
		if !found {
			res = []MatchPlayerWeapon{}
		}

		results[sid] = append(res, mpw)
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

func (db *Store) matchGetPlayers(ctx context.Context, matchID uuid.UUID) ([]*MatchPlayer, error) {
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

	var players []*MatchPlayer

	playerRows, errPlayer := db.Query(ctx, queryPlayer, matchID)
	if errPlayer != nil {
		if errors.Is(errPlayer, ErrNoResult) {
			return []*MatchPlayer{}, nil
		}

		return nil, errors.Wrapf(errPlayer, "Failed to query match players")
	}

	defer playerRows.Close()

	for playerRows.Next() {
		var (
			mpSum   = &MatchPlayer{}
			steamID int64
		)

		if errRow := playerRows.
			Scan(&mpSum.MatchPlayerID, &steamID, &mpSum.Team, &mpSum.TimeStart, &mpSum.TimeEnd,
				&mpSum.Kills, &mpSum.Damage, &mpSum.Shots, &mpSum.Hits, &mpSum.Backstabs,
				&mpSum.Headshots, &mpSum.Airshots, &mpSum.Assists, &mpSum.Deaths, &mpSum.Dominations, &mpSum.Dominated, &mpSum.Revenges,
				&mpSum.DamageTaken, &mpSum.HealingTaken, &mpSum.HealingPacks, &mpSum.Captures, &mpSum.CapturesBlocked,
				&mpSum.Extinguishes, &mpSum.BuildingBuilt, &mpSum.BuildingDestroyed, &mpSum.Name,
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

func (db *Store) matchGetChat(ctx context.Context, matchID uuid.UUID) (PersonMessages, error) {
	const query = `
		SELECT c.person_message_id, c.steam_id, c.server_id, c.body, c.persona_name, c.team, 
		       c.created_on, c.match_id, COUNT(f.person_message_id)::int::boolean as flagged
		FROM person_messages c
		LEFT JOIN person_messages_filter f on c.person_message_id = f.person_message_id
		WHERE c.match_id = $1
		GROUP BY c.person_message_id
		`

	messages := PersonMessages{}

	medicRows, errMedics := db.Query(ctx, query, matchID)
	if errMedics != nil {
		if errors.Is(errMedics, ErrNoResult) {
			return messages, nil
		}

		return nil, errors.Wrapf(errMedics, "Failed to query match healers")
	}

	defer medicRows.Close()

	for medicRows.Next() {
		var (
			msg     PersonMessage
			steamID int64
		)

		if errRow := medicRows.
			Scan(&msg.PersonMessageID, &steamID, &msg.ServerID, &msg.Body,
				&msg.PersonaName, &msg.Team, &msg.CreatedOn,
				&msg.MatchID, &msg.Flagged); errRow != nil {
			return nil, errors.Wrapf(errMedics, "Failed to scan match healer")
		}

		msg.SteamID = steamid.New(steamID)
		messages = append(messages, msg)
	}

	if medicRows.Err() != nil {
		return messages, errors.Wrap(medicRows.Err(), "medicRows error returned")
	}

	return messages, nil
}

func (db *Store) MatchGetByID(ctx context.Context, matchID uuid.UUID, match *MatchResult) error {
	const query = `
		SELECT match_id, server_id, map, title, score_red, score_blu, time_red, time_blu, time_start, time_end, winner
		FROM match WHERE match_id = $1`

	if errMatch := db.
		QueryRow(ctx, query, matchID).
		Scan(&match.MatchID, &match.ServerID, &match.MapName, &match.Title,
			&match.TeamScores.Red, &match.TeamScores.Blu, &match.TeamScores.BluTime, &match.TeamScores.BluTime,
			&match.TimeStart, &match.TimeEnd, &match.Winner); errMatch != nil {
		return errors.Wrapf(errMatch, "Failed to load root match")
	}

	playerStats, errPlayerStats := db.matchGetPlayers(ctx, matchID)
	if errPlayerStats != nil {
		return errors.Wrap(errPlayerStats, "Failed to fetch match players")
	}

	match.Players = playerStats

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

	for _, player := range match.Players {
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

		for _, player := range match.Players {
			if player.SteamID == steamID {
				player.MedicStats = &localStats

				break
			}
		}
	}

	weaponStats, errWeapons := db.matchGetPlayerWeapons(ctx, matchID)
	if errWeapons != nil {
		return errors.Wrap(errMedics, "Failed to fetch match weapon stats")
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

	chat, errChat := db.matchGetChat(ctx, matchID)

	if errChat != nil && !errors.Is(errChat, ErrNoResult) {
		return errors.Wrap(errMedics, "Failed to fetch match chat history")
	}

	match.Chat = chat

	if match.Chat == nil {
		match.Chat = PersonMessages{}
	}

	for _, player := range match.Players {
		if player.Weapons == nil {
			player.Weapons = []MatchPlayerWeapon{}
		}

		if player.Classes == nil {
			player.Classes = []MatchPlayerClass{}
		}

		if player.Killstreaks == nil {
			player.Killstreaks = []MatchPlayerKillstreak{}
		}
	}

	return nil
}

var (
	ErrIncompleteMatch     = errors.New("Insufficient match data")
	ErrInsufficientPlayers = errors.New("Insufficient match players")
)

const MinMedicHealing = 500

func (db *Store) MatchSave(ctx context.Context, match *logparse.Match) error {
	const (
		minPlayers = 6
		query      = `
		INSERT INTO match (match_id, server_id, map, title, score_red, score_blu, time_red, time_blu, time_start, time_end, winner) 
		VALUES ($1, $2, $3, $4, $5, $6,$7, $8, $9, $10, $11) 
		RETURNING match_id`
	)

	if match.TimeStart == nil || match.MapName == "" {
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
		QueryRow(ctx, query, match.MatchID, match.ServerID, match.MapName, match.Title,
			match.TeamScores.Red, match.TeamScores.Blu, match.TeamScores.RedTime, match.TeamScores.BluTime,
			match.TimeStart, match.TimeEnd, match.Winner()).
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

		if errSave := db.saveMatchPlayerStats(ctx, transaction, match, player); errSave != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				db.log.Error("Failed to rollback tx", zap.Error(errRollback))
			}

			return errSave
		}

		if errSave := db.saveMatchWeaponStats(ctx, transaction, player); errSave != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				db.log.Error("Failed to rollback tx", zap.Error(errRollback))
			}

			return errSave
		}

		if errSave := db.saveMatchPlayerClassStats(ctx, transaction, player); errSave != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				db.log.Error("Failed to rollback tx", zap.Error(errRollback))
			}

			return errSave
		}

		if errSave := db.saveMatchKillstreakStats(ctx, transaction, player); errSave != nil {
			if errRollback := transaction.Rollback(ctx); errRollback != nil {
				db.log.Error("Failed to rollback tx", zap.Error(errRollback))
			}

			return errSave
		}

		if player.HealingStats != nil && player.HealingStats.Healing >= MinMedicHealing {
			if errSave := db.saveMatchMedicStats(ctx, transaction, player.MatchPlayerID, player.HealingStats); errSave != nil {
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

func (db *Store) saveMatchPlayerStats(ctx context.Context, transaction pgx.Tx, match *logparse.Match, stats *logparse.PlayerStats) error {
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

func (db *Store) saveMatchMedicStats(ctx context.Context, transaction pgx.Tx, matchPlayerID int64, stats *logparse.HealingStats) error {
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

func (db *Store) saveMatchWeaponStats(ctx context.Context, transaction pgx.Tx, player *logparse.PlayerStats) error {
	const query = `
		INSERT INTO match_weapon (match_player_id, weapon_id, kills, damage, shots, hits, backstabs, headshots, airshots) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) 
		RETURNING player_weapon_id`

	for weapon, info := range player.WeaponInfo {
		weaponID, found := db.weaponMap.Get(weapon)
		if !found {
			db.log.Error("Unknown weapon", zap.String("weapon", string(weapon)))

			continue
		}

		if _, errWeapon := transaction.
			Exec(ctx, query, player.MatchPlayerID, weaponID, info.Kills, info.Damage, info.Shots, info.Hits,
				info.BackStabs, info.Headshots, info.Airshots); errWeapon != nil {
			return errors.Wrapf(errWeapon, "Failed to write weapon stats")
		}
	}

	return nil
}

func (db *Store) saveMatchPlayerClassStats(ctx context.Context, transaction pgx.Tx, player *logparse.PlayerStats) error {
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

func (db *Store) saveMatchKillstreakStats(ctx context.Context, transaction pgx.Tx, player *logparse.PlayerStats) error {
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

type PlayerClassStats struct {
	Class              logparse.PlayerClass
	ClassName          string
	Kills              int
	Assists            int
	Deaths             int
	Damage             int
	Dominations        int
	Dominated          int
	Revenges           int
	DamageTaken        int
	HealingTaken       int
	HealthPacks        int
	Captures           int
	CapturesBlocked    int
	Extinguishes       int
	BuildingsBuilt     int
	BuildingsDestroyed int
	Playtime           float64 // seconds
}

func (player PlayerClassStats) KDRatio() float64 {
	if player.Deaths <= 0 {
		return -1
	}

	return math.Ceil((float64(player.Kills)/float64(player.Deaths))*100) / 100
}

func (player PlayerClassStats) KDARatio() float64 {
	if player.Deaths <= 0 {
		return -1
	}

	return math.Ceil((float64(player.Kills+player.Assists)/float64(player.Deaths))*100) / 100
}

func (player PlayerClassStats) DamagePerMin() int {
	return int(float64(player.Damage) / (player.Playtime / 60))
}

type PlayerClassStatsCollection []PlayerClassStats

func (ps PlayerClassStatsCollection) Kills() int {
	var total int
	for _, class := range ps {
		total += class.Kills
	}

	return total
}

func (ps PlayerClassStatsCollection) Assists() int {
	var total int
	for _, class := range ps {
		total += class.Assists
	}

	return total
}

func (ps PlayerClassStatsCollection) Deaths() int {
	var total int
	for _, class := range ps {
		total += class.Deaths
	}

	return total
}

func (ps PlayerClassStatsCollection) Damage() int {
	var total int
	for _, class := range ps {
		total += class.Damage
	}

	return total
}

func (ps PlayerClassStatsCollection) DamageTaken() int {
	var total int
	for _, class := range ps {
		total += class.DamageTaken
	}

	return total
}

func (ps PlayerClassStatsCollection) Captures() int {
	var total int
	for _, class := range ps {
		total += class.Captures
	}

	return total
}

func (ps PlayerClassStatsCollection) Dominations() int {
	var total int
	for _, class := range ps {
		total += class.Dominations
	}

	return total
}

func (ps PlayerClassStatsCollection) Dominated() int {
	var total int
	for _, class := range ps {
		total += class.Dominated
	}

	return total
}

func (ps PlayerClassStatsCollection) Playtime() float64 {
	var total float64
	for _, class := range ps {
		total += class.Playtime
	}

	return total
}

func (ps PlayerClassStatsCollection) DamagePerMin() int {
	return int(float64(ps.Damage()) / (ps.Playtime() / 60))
}

func (ps PlayerClassStatsCollection) KDRatio() float64 {
	if ps.Deaths() <= 0 {
		return -1
	}

	return math.Ceil((float64(ps.Kills())/float64(ps.Deaths()))*100) / 100
}

func (ps PlayerClassStatsCollection) KDARatio() float64 {
	if ps.Deaths() <= 0 {
		return -1
	}

	return math.Ceil((float64(ps.Kills()+ps.Assists())/float64(ps.Deaths()))*100) / 100
}

func (db *Store) StatsPlayerClass(ctx context.Context, sid64 steamid.SID64) (PlayerClassStatsCollection, error) {
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

	rows, errQuery := db.Query(ctx, query, sid64.Int64())
	if errQuery != nil {
		return nil, Err(errQuery)
	}

	defer rows.Close()

	var stats PlayerClassStatsCollection

	for rows.Next() {
		var class PlayerClassStats
		if errScan := rows.
			Scan(&class.Class, &class.Kills, &class.Damage, &class.Assists, &class.Deaths, &class.Dominations,
				&class.Dominated, &class.Revenges, &class.DamageTaken, &class.HealingTaken, &class.HealthPacks,
				&class.Captures, &class.CapturesBlocked, &class.Extinguishes, &class.BuildingsBuilt,
				&class.BuildingsDestroyed, &class.Playtime); errScan != nil {
			return nil, Err(errScan)
		}

		class.ClassName = class.Class.String()
		stats = append(stats, class)
	}

	return stats, nil
}

type PlayerWeaponStats struct {
	Weapon     logparse.Weapon
	WeaponName string
	Kills      int
	Damage     int
	Shots      int
	Hits       int
	Backstabs  int
	Headshots  int
	Airshots   int
}

func (ws PlayerWeaponStats) Accuracy() float64 {
	if ws.Shots == 0 {
		return 0
	}

	return math.Ceil(float64(ws.Hits)/float64(ws.Shots)*10000) / 100
}

func (db *Store) StatsPlayerWeapons(ctx context.Context, sid64 steamid.SID64) ([]PlayerWeaponStats, error) {
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

	rows, errQuery := db.Query(ctx, query, sid64.Int64())
	if errQuery != nil {
		return nil, Err(errQuery)
	}

	defer rows.Close()

	var stats []PlayerWeaponStats

	for rows.Next() {
		var class PlayerWeaponStats
		if errScan := rows.
			Scan(&class.Weapon, &class.WeaponName, &class.Kills, &class.Damage, &class.Shots, &class.Hits,
				&class.Backstabs, &class.Headshots, &class.Airshots); errScan != nil {
			return nil, Err(errScan)
		}

		stats = append(stats, class)
	}

	return stats, nil
}

type PlayerKillstreakStats struct {
	Class     logparse.PlayerClass `json:"class"`
	ClassName string               `json:"class_name"`
	Kills     int                  `json:"kills"`
	Duration  int                  `json:"duration"`
	CreatedOn time.Time            `json:"created_on"`
}

func (db *Store) StatsPlayerKillstreaks(ctx context.Context, sid64 steamid.SID64) ([]PlayerKillstreakStats, error) {
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

	rows, errQuery := db.Query(ctx, query, sid64.Int64())
	if errQuery != nil {
		return nil, Err(errQuery)
	}

	defer rows.Close()

	var stats []PlayerKillstreakStats

	for rows.Next() {
		var class PlayerKillstreakStats
		if errScan := rows.
			Scan(&class.Class, &class.Kills, &class.Duration, &class.CreatedOn); errScan != nil {
			return nil, Err(errScan)
		}

		class.ClassName = class.Class.String()
		stats = append(stats, class)
	}

	return stats, nil
}

type PlayerMedicStats struct {
	Healing             int
	Drops               int
	NearFullChargeDeath int
	AvgUberLength       float64
	ChargesUber         int
	ChargesKritz        int
	ChargesVacc         int
	ChargesQuickfix     int
}

func (db *Store) StatsPlayerMedic(ctx context.Context, sid64 steamid.SID64) ([]PlayerMedicStats, error) {
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

	rows, errQuery := db.Query(ctx, query, sid64.Int64())
	if errQuery != nil {
		return nil, Err(errQuery)
	}

	defer rows.Close()

	var stats []PlayerMedicStats

	for rows.Next() {
		var class PlayerMedicStats
		if errScan := rows.
			Scan(&class.Healing, &class.Drops, &class.NearFullChargeDeath, &class.AvgUberLength,
				&class.ChargesUber, &class.ChargesKritz, &class.ChargesVacc, &class.ChargesQuickfix); errScan != nil {
			return nil, Err(errScan)
		}

		stats = append(stats, class)
	}

	return stats, nil
}

type CommonPlayerStats struct {
	SteamID           steamid.SID64 `json:"steam_id"`
	Name              string        `json:"name"`
	AvatarHash        string        `json:"avatar_hash"`
	Kills             int           `json:"kills"`
	Assists           int           `json:"assists"`
	Deaths            int           `json:"deaths"`
	Suicides          int           `json:"suicides"`
	Dominations       int           `json:"dominations"`
	Dominated         int           `json:"dominated"`
	Revenges          int           `json:"revenges"`
	Damage            int           `json:"damage"`
	DamageTaken       int           `json:"damage_taken"`
	HealingTaken      int           `json:"healing_taken"`
	HealthPacks       int           `json:"health_packs"`
	HealingPacks      int           `json:"healing_packs"` // Healing from packs
	Captures          int           `json:"captures"`
	CapturesBlocked   int           `json:"captures_blocked"`
	Extinguishes      int           `json:"extinguishes"`
	BuildingBuilt     int           `json:"building_built"`
	BuildingDestroyed int           `json:"building_destroyed"` // Opposing team buildings
	Backstabs         int           `json:"backstabs"`
	Airshots          int           `json:"airshots"`
	Headshots         int           `json:"headshots"`
	Shots             int           `json:"shots"`
	Hits              int           `json:"hits"`
}
type PlayerStats struct {
	CommonPlayerStats
	PlayerMedicStats
	MatchesTotal int           `json:"matches_total"`
	MatchesWon   int           `json:"matches_won"`
	PlayTime     time.Duration `json:"play_time"`
}

func (db *Store) PlayerStats(ctx context.Context, steamID steamid.SID64, stats *PlayerStats) error {
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

	if errQuery := db.
		QueryRow(ctx, query, steamID).
		Scan(&stats.MatchesWon, &stats.MatchesWon, &stats.HealthPacks,
			&stats.Extinguishes, &stats.BuildingBuilt, &stats.Kills, &stats.Assists, &stats.Damage, &stats.DamageTaken,
			&stats.PlayTime, &stats.Captures, &stats.CapturesBlocked, &stats.Dominated, &stats.Dominations, &stats.Revenges,
			&stats.Deaths, &stats.BuildingDestroyed, &stats.HealingTaken, &stats.Healing, &stats.Drops, &stats.ChargesUber,
			&stats.ChargesKritz, &stats.ChargesQuickfix, &stats.ChargesVacc); errQuery != nil {
		return Err(errQuery)
	}

	stats.SteamID = steamID

	return nil
}

type MatchSummary struct {
	MatchID   uuid.UUID `json:"match_id"`
	ServerID  int       `json:"server_id"`
	IsWinner  bool      `json:"is_winner"`
	ShortName string    `json:"short_name"`
	Title     string    `json:"title"`
	MapName   string    `json:"map_name"`
	ScoreBlu  int       `json:"score_blu"`
	ScoreRed  int       `json:"score_red"`
	TimeStart time.Time `json:"time_start"`
	TimeEnd   time.Time `json:"time_end"`
}

func (m MatchSummary) Path() string {
	return fmt.Sprintf("/log/%s", m.MatchID.String())
}

func (db *Store) Matches(ctx context.Context, opts MatchesQueryOpts) ([]MatchSummary, int64, error) {
	countBuilder := db.sb.Select("count(m.match_id) as count").
		From("match m").
		LeftJoin("public.match_player mp on m.match_id = mp.match_id").
		LeftJoin("public.server s on s.server_id = m.server_id")

	builder := db.sb.
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

	if opts.SteamID.Valid() {
		builder = builder.Where(sq.Eq{"mp.steam_id": opts.SteamID.Int64()})
		countBuilder = countBuilder.Where(sq.Eq{"mp.steam_id": opts.SteamID.Int64()})
	}

	builder = applySafeOrder(builder, opts.QueryFilter, map[string][]string{
		"":   {"winner"},
		"m.": {"match_id", "server_id", "map", "score_blu", "score_red", "time_start", "time_end"},
	}, "match_id")

	if opts.Limit > 0 {
		builder = builder.Limit(opts.Limit)
	}

	if opts.Offset > 0 {
		builder = builder.Offset(opts.Offset)
	}

	countQuery, countArgs, errCountArgs := countBuilder.ToSql()
	if errCountArgs != nil {
		return nil, 0, errors.Wrapf(errCountArgs, "Failed to build count query")
	}

	var count int64

	if errCount := db.
		QueryRow(ctx, countQuery, countArgs...).
		Scan(&count); errCount != nil {
		return nil, 0, Err(errCount)
	}

	query, args, errQueryArgs := builder.ToSql()
	if errQueryArgs != nil {
		return nil, 0, errors.Wrapf(errQueryArgs, "Failed to build query")
	}

	rows, errQuery := db.Query(ctx, query, args...)
	if errQuery != nil {
		return nil, 0, errors.Wrapf(errQuery, "Failed to query matches")
	}

	defer rows.Close()

	var matches []MatchSummary

	for rows.Next() {
		var summary MatchSummary
		if errScan := rows.Scan(&summary.MatchID, &summary.ServerID, &summary.IsWinner, &summary.ShortName,
			&summary.Title, &summary.MapName, &summary.ScoreBlu, &summary.ScoreRed, &summary.TimeStart,
			&summary.TimeEnd); errScan != nil {
			return nil, 0, errors.Wrapf(errScan, "Failed to scan match row")
		}

		matches = append(matches, summary)
	}

	if rows.Err() != nil {
		db.log.Error("Matches rows error", zap.Error(rows.Err()))
	}

	return matches, count, nil
}
