package stats

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Repository struct{ database.Database }

func NewRepository(database database.Database) Repository {
	return Repository{Database: database}
}

func (r Repository) Match(ctx context.Context, matchID uuid.UUID) (*Match, error) {
	match, errMatch := r.getMatch(ctx, matchID)
	if errMatch != nil {
		return nil, errMatch
	}

	if errRounds := r.getRounds(ctx, match); errRounds != nil {
		return nil, errRounds
	}

	if errPlayers := r.getPlayers(ctx, match); errPlayers != nil {
		return nil, errPlayers
	}

	if errClasses := r.getRoundPlayersClasses(ctx, match); errClasses != nil {
		return nil, errClasses
	}

	if errWeapons := r.getRoundPlayersWeapons(ctx, match); errWeapons != nil {
		return nil, errWeapons
	}

	return match, nil
}

func (r Repository) getRoundPlayers(ctx context.Context, match *Match) error {
	const query = `
		SELECT
			p.round_id, p.steam_id, p.team, p.mvp, p.tick_start, p.tick_end, p.points, p.connection_count,
			p.bonus_points, p.kills, p.assists, p.deaths, p.postround_kills, p.postround_assists,
			p.preround_healing, p.healing, p.drops, p.near_full_charge_death,
			p.charges_uber, p.charges_kritz, p.charges_vacc, p.charges_quickfix, p.damage, p.damage_taken,
			p.dominations, p.dominated, p.revenges, p.revenged, p.airshots, p.headshots, p.headshot_kills,
			p.backstabs, p.backstab_kills, p.was_headshot, p.was_backstabbed, p.shots, p.hits, p.objects_built, p.objects_destroyed,
			p.scoreboard_kills, p.scoreboard_assists, p.suicides, p.scoreboard_deaths, p.postround_deaths, p.captures, p.captures_blocked,
			p.scoreboard_damage, p.extinguishes, p.ignites, p.buildings_built, p.buildings_destroyed
		FROM
			match_round_player p
		LEFT JOIN
			match_round r ON r.round_id = p.round_id
		WHERE
			r.match_id = $1`
	rows, errRows := r.Query(ctx, query, match.MatchID)
	if errRows != nil {
		return database.Err(errRows)
	}

	for rows.Next() {
		var mrp MatchRoundPlayer
		if err := rows.Scan(&mrp.RoundID, &mrp.SteamID, &mrp.Team, &mrp.MVP, &mrp.TickStart, &mrp.TickEnd, &mrp.Points,
			&mrp.ConnectionCount, &mrp.BonusPoints, &mrp.Kills, &mrp.Assists, &mrp.Deaths, &mrp.PostroundKills,
			&mrp.PostroundAssists, &mrp.PreroundHealing, &mrp.Healing, &mrp.Drops, &mrp.NearFullChargeDeath,
			&mrp.ChargesUber, &mrp.ChargesKritz, &mrp.ChargesVacc, &mrp.ChargesQuickfix, &mrp.Damage, &mrp.DamageTaken,
			&mrp.Dominations, &mrp.Dominated, &mrp.Revenges, &mrp.Revenged, &mrp.Airshots, &mrp.Headshots, &mrp.HeadshotKills,
			&mrp.Backstabs, &mrp.BackstabKills, &mrp.WasHeadshot, &mrp.WasBackstabbed, &mrp.Shots, &mrp.Hits, &mrp.ObjectsBuilt,
			&mrp.ObjectsDestroyed, &mrp.ScoreboardKills, &mrp.ScoreboardAssists, &mrp.Suicides, &mrp.ScoreboardDeaths,
			&mrp.PostroundDeaths, &mrp.Captures, &mrp.CapturesBlocked, &mrp.ScoreboardDamage, &mrp.Extinguishes,
			&mrp.Ignites, &mrp.BuildingsBuilt, &mrp.BUildingsDestroyed,
		); err != nil {
			return database.Err(err)
		}

		match.Players = append(match.Players, mrp)
	}

	return nil
}

func (r Repository) getRoundPlayersWeapons(ctx context.Context, match *Match) error {
	const query = `
		SELECT
			w.weapon, w.round_id, w.steam_id, w.kills, w.assists, w.deaths, w.postround_kills, w.postround_assists,
			w.postround_deaths, w.damage, w.damage_taken, w.dominations, w.dominated, w.revenges,w.revenged, w.airshots,
			w.headshot_kills, w.backstab_kills, w.headshots, w.backstabs, w.was_headshot, w.was_backstabbed,
			w.preround_healing, w.healing, w.postround_healing, w.drops, w.near_full_charge_death, w.charges_uber,
			w.charges_kritz, w.charges_vacc, w.charges_quickfix
		FROM
			match_round_player_weapon w
		LEFT JOIN
			match_round r ON r.round_id = w.round_id
		WHERE
			r.match_id = $1`
	rows, errRows := r.Query(ctx, query, match.MatchID)
	if errRows != nil {
		return database.Err(errRows)
	}

	for rows.Next() {
		var mrws MatchRoundWeaponStats
		if err := rows.Scan(&mrws.Weapon, &mrws.RoundID, &mrws.SteamID, &mrws.Kills, &mrws.Assists, &mrws.Deaths,
			&mrws.PostroundKills, &mrws.PostroundAssists, &mrws.PostroundDeaths, &mrws.Damage, &mrws.DamageTaken,
			&mrws.Dominations, &mrws.Dominated, &mrws.Revenges, &mrws.Revenged, &mrws.Airshots, &mrws.HeadshotKills,
			&mrws.BackstabKills, &mrws.Headshots, &mrws.BackstabKills, &mrws.WasHeadshot, &mrws.WasBackstabbed,
			&mrws.PreroundHeadling, &mrws.Healing, &mrws.PostroundHealing, &mrws.Drops, &mrws.NearFullChargeDeath,
			&mrws.ChargesUber, &mrws.ChargesKritz, &mrws.ChargesVacc, &mrws.ChargesQuickfix,
		); err != nil {
			return database.Err(err)
		}

		match.Weapons = append(match.Weapons, mrws)
	}

	return nil
}

func (r Repository) getRoundPlayersClasses(ctx context.Context, match *Match) error {
	const query = `
		SELECT
			c.class, c.round_id, c.steam_id, c.kills, c.assists, c.deaths, c.postround_kills, c.postround_assists,
			c.postround_deaths, c.damage, c.damage_taken, c.dominations, c.dominated, c.revenges, c.revenged,
			c.airshots, c.headshot_kills, c.backstab_kills, c.headshots, c.backstabs, c.was_headshot, c.was_backstabbed,
			c.preround_healing, c.healing, c.postround_healing, c.drops, c.near_full_charge_death, c.charges_uber,
			c.charges_kritz, c.charges_vacc, c.charges_quickfix
		FROM
			match_round_player_class c
		LEFT JOIN
			match_round r ON r.round_id = c.round_id
		WHERE
			r.match_id = $1`
	rows, errRows := r.Query(ctx, query, match.MatchID)
	if errRows != nil {
		return database.Err(errRows)
	}

	for rows.Next() {
		var mrcs MatchRoundClassStats
		if err := rows.Scan(&mrcs.Class, &mrcs.RoundID, &mrcs.SteamID, &mrcs.Kills, &mrcs.Assists, &mrcs.Deaths, &mrcs.PostroundKills,
			&mrcs.PostroundAssists, &mrcs.PostroundDeaths, &mrcs.Damage, &mrcs.DamageTaken, &mrcs.Dominations, &mrcs.Dominated,
			&mrcs.Revenges, &mrcs.Revenged, &mrcs.Airshots, &mrcs.HeadshotKills, &mrcs.BackstabKills, &mrcs.Headshots, &mrcs.BackstabKills,
			&mrcs.WasHeadshot, &mrcs.WasBackstabbed, &mrcs.PreroundHeadling, &mrcs.Healing, &mrcs.PostroundHealing, &mrcs.Drops,
			&mrcs.NearFullChargeDeath, &mrcs.ChargesUber, &mrcs.ChargesKritz, &mrcs.ChargesVacc, &mrcs.ChargesQuickfix,
		); err != nil {
			return database.Err(err)
		}

		match.Classes = append(match.Classes, mrcs)
	}

	return nil
}

func (r Repository) getPlayers(ctx context.Context, match *Match) error {
	const query = `
		SELECT
			p.round_id, p.steam_id, p.team, p.mvp, p.tick_start, p.tick_end, p.points, p.connection_count,
			p.bonus_points, p.kills, p.assists, p.deaths, p.postround_kills, p.postround_assists,
			p.preround_healing, p.healing, p.drops, p.near_full_charge_death, p.charges_uber, p.charges_kritz,
			p.charges_vacc, p.charges_quickfix, p.damage, p.damage_taken, p.dominations, p.dominated,
			p.revenges, p.revenged, p.airshots, p.headshots, p.headshot_kills, p.backstabs, p.backstab_kills,
		 	p.was_headshots, p.was_backstabbed, p.shots, p.hits, p.objects_built, p.objects_destroyed,
			p.scoreboard_kills, p.scoreboard_assists, p.suicides, p.scoreboard_deaths, p.postround_deaths,
			p.captures, p.captures_blocked, p.scoreboard_damage, p.extinguishes, p.ignites, p.buildings_built,
			p.buildings_destroyed
		FROM
			match_round_player p
		LEFT JOIN
			match_round r ON r.round_id = p.round_id
		WHERE
			r.match_id = $1`

	rows, errRows := r.Query(ctx, query, match.MatchID)
	if errRows != nil {
		return database.Err(errRows)
	}

	for rows.Next() {
		var mrp MatchRoundPlayer
		if err := rows.Scan(&mrp.RoundID, &mrp.SteamID, &mrp.Team, &mrp.MVP, &mrp.TickStart, &mrp.TickEnd, &mrp.Points,
			&mrp.ConnectionCount, &mrp.BonusPoints, &mrp.Kills, &mrp.Assists, &mrp.Deaths, &mrp.PostroundKills,
			&mrp.PostroundAssists, &mrp.PostroundHealing, &mrp.Healing, &mrp.Drops, &mrp.NearFullChargeDeath,
			&mrp.ChargesUber, &mrp.ChargesKritz, &mrp.ChargesVacc, &mrp.ChargesQuickfix, &mrp.Damage, &mrp.DamageTaken,
			&mrp.Dominations, &mrp.Dominated, &mrp.Revenges, &mrp.Revenged, &mrp.Airshots, &mrp.Headshots, &mrp.HeadshotKills,
			&mrp.Backstabs, &mrp.BackstabKills, &mrp.WasHeadshot, &mrp.WasBackstabbed, &mrp.Shots, &mrp.Hits, &mrp.ObjectsBuilt,
			&mrp.ObjectsDestroyed, &mrp.ScoreboardKills, &mrp.ScoreboardAssists, &mrp.Suicides, &mrp.ScoreboardDeaths,
			&mrp.PostroundDeaths, &mrp.Captures, &mrp.CapturesBlocked, &mrp.ScoreboardDamage, &mrp.Extinguishes,
			&mrp.Ignites, &mrp.BuildingsBuilt, &mrp.BUildingsDestroyed); err != nil {
			return database.Err(err)
		}

		match.Players = append(match.Players, mrp)
	}

	return nil
}

func (r Repository) getMatch(ctx context.Context, matchID uuid.UUID) (*Match, error) {
	const query = `
		SELECT
			match_id, server_id, map_id, demo_id, stats_bucket_id, hostname, score_red, score_blu,
			start_time, duration_ms, created_on
		FROM
			match
		WHERE
			match_id = $1`

	var match Match
	if err := r.QueryRow(ctx, query, matchID).Scan(&match.MatchID, &match.ServerID, &match.MapID, &match.DemoID,
		&match.StatsBucketID, &match.Hostname, &match.ScoreRed, &match.ScoreBlu, &match.StartTime, &match.DurationMs,
		&match.CreatedOn); err != nil {
		return nil, database.Err(err)
	}

	return &match, nil
}

func (r Repository) getRounds(ctx context.Context, match *Match) error {
	const query = `
		SELECT
			round_id, winner, is_stalemate, is_sudden_death, duration_ms
		FROM
			match_round
		WHERE
			match_id = $1`
	rows, errRows := r.Query(ctx, query, match.MatchID)
	if errRows != nil {
		return database.Err(errRows)
	}

	for rows.Next() {
		var round MatchRound
		if err := rows.Scan(&round.RoundID, &round.Winner, &round.IsStalemate, &round.IsSuddenDeath, &round.DurationMs); err != nil {
			return database.Err(err)
		}

		if round.Winner != "" {
			switch round.Winner {
			case "red":
				match.ScoreRed++
			case "blu":
				match.ScoreBlu++
			}
		}

		match.Rounds = append(match.Rounds, round)
	}

	return nil
}

func (r Repository) CreateMatch(ctx context.Context, serverID int32, demoID int32, demo *demoparse.Demo, timeStart time.Time, mapInfo maps.Map, statsBucketID *int32) (uuid.UUID, error) {
	newID, errID := uuid.NewV4()
	if errID != nil {
		return newID, fmt.Errorf("%w: failed to generate UUID", ErrInvalidState)
	}

	scores := demo.Scores()
	duration := time.Duration(demo.Duration) * time.Second

	transaction, errTx := r.Begin(ctx)
	if errTx != nil {
		return newID, database.Err(errTx)
	}

	if statsBucketID == nil {
		bucket, errBucket := r.GetBucket(ctx, 1)
		if errBucket != nil {
			return newID, errBucket
		}
		statsBucketID = &bucket.BucketID
	}

	if _, errMatch := transaction.Exec(ctx, `
		INSERT INTO match (
			match_id, server_id, map_id, demo_id, stats_bucket_id, hostname,
			score_red, score_blu, start_time, duration_ms, created_on)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`,
		newID, serverID, mapInfo.MapID, demoID, statsBucketID, demo.Filename, scores.Red,
		scores.Blu, timeStart, duration.Milliseconds(), time.Now()); errMatch != nil {
		if err := transaction.Rollback(ctx); err != nil {
			slog.Error("Failed to rollback tx", slog.String("error", err.Error()))
		}

		return newID, database.Err(errMatch)
	}
	playerTeams := playerTeamMap(demo)
	for _, round := range demo.Rounds {
		if err := r.insertRound(ctx, transaction, newID, playerTeams, round); err != nil {
			if err := transaction.Rollback(ctx); err != nil {
				slog.Error("Failed to rollback tx", slog.String("error", err.Error()))
			}

			return newID, err
		}
	}

	if err := transaction.Commit(ctx); err != nil {
		if err := transaction.Rollback(ctx); err != nil {
			slog.Error("Failed to rollback tx", slog.String("error", err.Error()))
		}

		return newID, database.Err(err)
	}

	return newID, nil
}

func (r Repository) insertRound(ctx context.Context, transaction pgx.Tx, matchID uuid.UUID, playerTeams map[string]string, round demoparse.RoundSummary) error {
	const query = `
		INSERT INTO match_round (
			match_id, winner, is_stalemate, is_sudden_death, duration_ms
		) VALUES (
			$1, $2, $3, $4, $5
		)
		RETURNING round_id`
	duration := time.Duration(round.Time) * time.Second

	var roundID int64
	if errRound := transaction.
		QueryRow(ctx, query, matchID, toTfTeam(round.Winner), round.IsStalemate, round.IsSuddenDeath, duration.Milliseconds()).
		Scan(&roundID); errRound != nil {
		return database.Err(errRound)
	}

	for _, player := range round.Players {
		steamID := steamid.New(player.SteamID)
		if !steamID.Valid() {
			continue
		}

		player.Team = playerTeams[player.SteamID]

		if err := r.insertRoundPlayer(ctx, transaction, roundID, steamID, round, player); err != nil {
			return err
		}
	}

	return nil
}

func (r Repository) insertRoundPlayer(ctx context.Context, transaction pgx.Tx, roundID int64, steamID steamid.SteamID, round demoparse.RoundSummary, player demoparse.PlayerSummary) error {
	const query = `
		INSERT INTO match_round_player (
			round_id, steam_id, team, mvp, tick_start, tick_end, points, connection_count, bonus_points,
			kills, assists, deaths, postround_kills, postround_assists, preround_healing, healing, drops,
			near_full_charge_death, charges_uber, charges_kritz, charges_vacc, charges_quickfix, damage,
			damage_taken, dominations, dominated, revenges, revenged, airshots, headshots, headshot_kills,
			backstabs, backstab_kills, was_headshots, was_backstabbed, shots, hits, objects_built, objects_destroyed,
			scoreboard_kills, scoreboard_assists, suicides, scoreboard_deaths, postround_deaths, captures,
			captures_blocked, scoreboard_damage, extinguishes, ignites, buildings_built, buildings_destroyed)
		VALUES(
			$1,  $2,  $3,  $4,  $5,  $6,  $7,  $8,  $9,  $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34, $35, $36, $37, $38, $39, $40,
			$41, $42, $43, $44, $45, $46, $47, $48, $49, $50,
			$51
		)`

	isMvp := slices.Contains(round.Mvps, string(steamID.Steam3()))

	if _, err := transaction.Exec(ctx, query,
		roundID, steamID.Int64(), toTfTeam(player.Team), isMvp, player.TickStart, player.TickEnd,
		player.Points, player.ConnectionCount, player.BonusPoints, player.Kills, player.Assists,
		player.Deaths, player.PostroundKills, player.PostroundAssists, player.PreroundHealing,
		player.Healing, player.Drops, player.NearFullChargeDeath, player.ChargesUber, player.ChargesKritz,
		player.ChargesVacc, player.ChargesQuickfix, player.Damage, player.DamageTaken, player.Dominations,
		player.Dominated, player.Revenges, player.Revenged, player.Airshots, player.Headshots,
		player.HeadshotKills, player.Backstabs, player.BackstabKills, player.WasHeadshot, player.WasBackstabbed,
		player.Shots, player.Hits, player.ObjectBuilt, player.ObjectDestroyed, player.ScoreboardKills,
		player.ScoreboardAssists, player.Suicides, player.ScoreboardDeaths, player.PostroundDeaths,
		player.Captures, player.CapturesBlocked, player.ScoreboardDamage, player.Extinguishes,
		player.Ignites, player.BuildingBuilt, player.BuildingDestroyed); err != nil {
		return database.Err(err)
	}

	for weapon, weaponStats := range player.Weapons {
		if err := r.insertRoundPlayerWeapon(ctx, transaction, steamID, roundID, weaponStats, weapon); err != nil {
			return database.Err(err)
		}
	}

	for class, weaponStats := range player.Classes {
		if err := r.insertRoundPlayerClass(ctx, transaction, steamID, roundID, weaponStats, class); err != nil {
			return database.Err(err)
		}
	}

	return nil
}

func (r Repository) insertRoundPlayerWeapon(ctx context.Context, transaction pgx.Tx, steamID steamid.SteamID, roundID int64, stats demoparse.Stats, weapon string) error {
	const query = `
		INSERT INTO match_round_player_weapon (
			weapon, round_id, steam_id, kills, assists, deaths, postround_kills, postround_assists,  postround_deaths, damage,
			damage_taken, dominations, dominated, revenges, revenged, airshots, headshot_kills, backstabs, backstab_kills,
			headshots, was_headshot, was_backstabbed, preround_healing, healing, postround_healing, drops, near_full_charge_death,
			charges_uber, charges_kritz, charges_vacc, charges_quickfix
		) VALUES (
			LOWER($1),$2,  $3,  $4,  $5,  $6,  $7,  $8,  $9,  $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31
		)`

	if _, err := transaction.Exec(ctx, query, weapon, roundID, steamID.Int64(),
		stats.Kills, stats.Assists, stats.Deaths, stats.PostroundKills, stats.PostroundAssists, stats.PostroundDeaths, stats.Damage,
		stats.DamageTaken, stats.Dominations, stats.Dominated, stats.Revenges, stats.Revenged, stats.Airshots, stats.HeadshotKills,
		stats.Backstabs, stats.BackstabKills, stats.Headshots, stats.WasHeadshot, stats.WasBackstabbed, stats.PreroundHealing, stats.Healing,
		stats.PostroundHealing, stats.Drops, stats.NearFullChargeDeath, stats.ChargesUber, stats.ChargesKritz,
		stats.ChargesVacc, stats.ChargesQuickfix); err != nil {
		return database.Err(err)
	}

	return nil
}

func (r Repository) insertRoundPlayerClass(ctx context.Context, transaction pgx.Tx, steamID steamid.SteamID, roundID int64, stats demoparse.Stats, class string) error {
	const query = `
		INSERT INTO match_round_player_class (
			class, round_id, steam_id, kills, assists, deaths, postround_kills, postround_assists,  postround_deaths, damage,
			damage_taken, dominations, dominated, revenges, revenged, airshots, headshot_kills, backstabs, backstab_kills, headshots,
			was_headshot, was_backstabbed, preround_healing, healing, postround_healing, drops, near_full_charge_death, charges_uber,
			charges_kritz, charges_vacc, charges_quickfix
		) VALUES (
			LOWER($1)::player_class, $2,  $3,  $4,  $5,  $6,  $7,  $8,  $9,  $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31
		)`

	if _, err := transaction.Exec(ctx, query, toTFClass(class), roundID, steamID.Int64(),
		stats.Kills, stats.Assists, stats.Deaths, stats.PostroundKills, stats.PostroundAssists, stats.PostroundDeaths, stats.Damage,
		stats.DamageTaken, stats.Dominations, stats.Dominated, stats.Revenges, stats.Revenged, stats.Airshots, stats.HeadshotKills,
		stats.Backstabs, stats.BackstabKills, stats.Headshots, stats.WasHeadshot, stats.WasBackstabbed, stats.PreroundHealing, stats.Healing,
		stats.PostroundHealing, stats.Drops, stats.NearFullChargeDeath, stats.ChargesUber, stats.ChargesKritz, stats.ChargesVacc,

		stats.ChargesQuickfix); err != nil {
		return database.Err(err)
	}

	return nil
}

func (r Repository) GetBucket(ctx context.Context, statsBucketID int32) (*Bucket, error) {
	const query = "SELECT stats_bucket_id, bucket_name FROM stats_bucket WHERE stats_bucket_id = $1"
	var bucket Bucket
	if err := r.QueryRow(ctx, query, statsBucketID).Scan(&bucket.BucketID, &bucket.BucketName); err != nil {
		return nil, database.Err(err)
	}

	return &bucket, nil
}

func (r Repository) CreateBucket(ctx context.Context, bucketName string) (*Bucket, error) {
	const query = "INSERT INTO stats_bucket (bucket_name) VALUES ($1) RETURNING stats_bucket_id"
	bucket := Bucket{BucketName: bucketName}
	if err := r.QueryRow(ctx, query, bucketName).Scan(&bucket.BucketID); err != nil {
		return nil, database.Err(err)
	}

	return &bucket, nil
}

func (r Repository) SaveBucket(ctx context.Context, bucket Bucket) error {
	const query = "UPDATE stats_bucket SET bucket_name = $1 WHERE stats_bucket_id = $2"
	if err := r.Exec(ctx, query, bucket.BucketName, bucket.BucketID); err != nil {
		return database.Err(err)
	}

	return nil
}

func playerTeamMap(demo *demoparse.Demo) map[string]string {
	// Find the last team the player played on and use that as their final team.
	playerTeams := map[string]string{}
	for _, steamID := range demo.SteamIDs() {
		steam3 := string(steamID.Steam3())
		found := false
		for roundIdx := range slices.Backward(demo.Rounds) {
			for _, player := range demo.Rounds[roundIdx].Players {
				if player.SteamID == steam3 {
					if demo.Rounds[roundIdx].Winner == "" {
						continue
					}
					redWinner := toTfTeam(demo.Rounds[roundIdx].Winner) == "red"
					if redWinner {
						if slices.Contains(demo.Rounds[roundIdx].Winners, steam3) {
							playerTeams[steam3] = "red"
						} else {
							playerTeams[steam3] = "blu"
						}
					} else {
						if slices.Contains(demo.Rounds[roundIdx].Winners, steam3) {
							playerTeams[steam3] = "blu"
						} else {
							playerTeams[steam3] = "red"
						}
					}

					found = true

					break
				}
			}
			if found {
				break
			}
		}
	}

	return playerTeams
}

func toTfTeam(team string) string {
	switch strings.ToLower(team) {
	case "blue":
		fallthrough
	case "blu":
		return "blu"
	case "red":
		return "red"
	case "spec":
		return "spec"
	case "unassigned":
		fallthrough
	default:
		return "unassigned"
	}
}

func toTFClass(class string) string {
	switch strings.ToLower(class) {
	case "scout":
		return "scout"
	case "soldier":
		return "soldier"
	case "pyro":
		return "pyro"
	case "demoman":
		fallthrough
	case "demo":
		return "demo"
	case "Heavyweapons":
		fallthrough
	case "heavy":
		return "heavy"
	case "engy":
		fallthrough
	case "engineer":
		return "engineer"
	case "medic":
		return "medic"
	case "sniper":
		return "sniper"
	case "spy":
		return "spy"
	default:
		return ""
	}
}
