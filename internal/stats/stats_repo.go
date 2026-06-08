package stats

import (
	"context"
	"fmt"
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

func (r Repository) CreateMatch(ctx context.Context, serverID int32, demoID int32, demo *demoparse.Demo, timeStart time.Time, mapInfo maps.Map, statsBucketID *int32) (uuid.UUID, error) {
	newID, errID := uuid.NewV4()
	if errID != nil {
		return newID, fmt.Errorf("%w: failed to generate UUID", ErrInvalidState)
	}

	scores := demo.Scores()
	duration := time.Duration(demo.Duration) * time.Second

	tx, errTx := r.Begin(ctx)
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

	if _, errMatch := tx.Exec(ctx, `
		INSERT INTO match (match_id, server_id, map_id, demo_id, stats_bucket_id, hostname, score_red, score_blu, start_time, duration_ms, created_on)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		newID, serverID, mapInfo.MapID, demoID, statsBucketID, demo.Filename, scores.Red,
		scores.Blu, timeStart, duration.Milliseconds(), time.Now()); errMatch != nil {
		tx.Rollback(ctx)

		return newID, database.Err(errMatch)
	}

	for _, round := range demo.Rounds {
		if err := r.insertRound(ctx, tx, newID, round); err != nil {
			tx.Rollback(ctx)

			return newID, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		tx.Rollback(ctx)

		return newID, database.Err(err)
	}

	return newID, nil
}

func (r Repository) insertRound(ctx context.Context, tx pgx.Tx, matchID uuid.UUID, round demoparse.RoundSummary) error {
	const query = `
		INSERT INTO match_round (match_id, winner, is_stalemate, is_sudden_death, duration_ms)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING round_id`
	duration := time.Duration(round.Duration) * time.Second

	var roundID int64
	if errRound := tx.
		QueryRow(ctx, query, matchID, toTfTeam(round.Winner), round.IsStalemate, round.IsSuddenDeath, duration).
		Scan(&roundID); errRound != nil {
		return database.Err(errRound)
	}

	for _, player := range round.Players {
		steamID := steamid.New(player.SteamID)
		if !steamID.Valid() {
			continue
		}

		if err := r.insertRoundPlayer(ctx, tx, roundID, steamID, round, player); err != nil {
			return err
		}
	}

	return nil
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

func (r Repository) insertRoundPlayer(ctx context.Context, tx pgx.Tx, roundID int64, steamID steamid.SteamID, round demoparse.RoundSummary, p demoparse.PlayerSummary) error {
	const query = `
		INSERT INTO match_round_player (
			round_id, steam_id, team, mvp, tick_start, tick_end, points, connection_count, bonus_points,
			kills, assists, deaths, postround_kills, postround_assists, preround_healing, healing, drops, near_full_charge_death,
			charges_uber, charges_kritz, charges_vacc, charges_quickfix, damage, damage_taken, dominations, dominated, revenges, revenged,
			airshots, headshots, headshot_kills, backstabs, backstab_kills, was_headshots, was_backstabbed, shots, hits,
			objects_built, objects_destroyed, scoreboard_kills, scoreboard_assists, suicides, scoreboard_deaths, postround_deaths,
			captures, captures_blocked, scoreboard_damage, extinguishes, ignites, buildings_built, buildings_destroyed)
		VALUES($1,  $2,  $3,  $4,  $5,  $6,  $7,  $8,  $9,  $10,
		       $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			   $21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
			   $31, $32, $33, $34, $35, $36, $37, $38, $39, $40,
			   $41, $42, $43, $44, $45, $46, $47, $48, $49, $50,
			   $51)
	`

	isMvp := slices.Contains(round.Mvps, string(steamID.Steam3()))

	if _, err := tx.Exec(ctx, query,
		roundID, steamID.Int64(), toTfTeam(p.Team), isMvp, p.TickStart, p.TickEnd, p.Points, p.ConnectionCount, p.BonusPoints,
		p.Kills, p.Assists, p.Deaths, p.PostroundKills, p.PostroundAssists, p.PreroundHealing, p.Healing, p.Drops, p.NearFullChargeDeath,
		p.ChargesUber, p.ChargesKritz, p.ChargesVacc, p.ChargesQuickfix, p.Damage, p.DamageTaken, p.Dominations, p.Dominated, p.Revenges, p.Revenged,
		p.Airshots, p.Headshots, p.HeadshotKills, p.Backstabs, p.BackstabKills, p.WasHeadshot, p.WasBackstabbed, p.Shots, p.Hits,
		p.ObjectBuilt, p.ObjectDestroyed, p.ScoreboardKills, p.ScoreboardAssists, p.Suicides, p.ScoreboardDeaths, p.PostroundDeaths,
		p.Captures, p.CapturesBlocked, p.ScoreboardDamage, p.Extinguishes, p.Ignites, p.BuildingBuilt, p.BuildingDestroyed); err != nil {
		return database.Err(err)
	}

	// for weapon, weaponStats := range p.Weapons {
	// 	r.insertRoundPlayerWeapon(ctx, tx, roundID, weaponStats)
	// }

	return nil
}

func (r Repository) insertRoundPlayerWeapon(ctx context.Context, tx pgx.Tx, roundID int64, stats demoparse.Stats, weapon demoparse.WeaponID) error {
	return nil
}

func (r Repository) insertRoundPlayerClass(ctx context.Context, tx pgx.Tx, roundID int64, stats demoparse.Stats, class demoparse.PlayerClass) error {
	return nil
}

func (r Repository) AddPlayerStatsAlltime(_ context.Context, _ demoparse.Stats) error {
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
