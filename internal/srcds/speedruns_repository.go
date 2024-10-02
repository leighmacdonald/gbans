package srcds

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
)

func NewSpeedrunRepository(database database.Database) domain.SpeedrunRepository {
	return &speedrunRepository{db: database}
}

type speedrunRepository struct {
	db database.Database
}

func (r *speedrunRepository) Save(ctx context.Context, details *domain.Speedrun) error {
	return r.db.WrapTx(ctx, func(tx pgx.Tx) error {
		query, args, errQuery := r.db.Builder().
			Insert("speedrun").
			SetMap(map[string]interface{}{
				"map_name":     details.MapName,
				"category":     details.Category,
				"duration":     details.Duration,
				"player_count": details.PlayerCount,
				"bot_count":    details.BotCount,
				"created_on":   details.CreatedOn,
			}).
			Suffix(" RETURNING speedrun_id").
			ToSql()
		if errQuery != nil {
			return r.db.DBErr(errQuery)
		}

		if errScan := tx.QueryRow(ctx, query, args...).Scan(&details.SpeedrunID); errScan != nil {
			return r.db.DBErr(errScan)
		}

		if errRounds := r.insertRounds(ctx, tx, details.SpeedrunID, details.Rounds); errRounds != nil {
			return errRounds
		}

		if errPlayers := r.insertPlayers(ctx, tx, details.SpeedrunID, details.Players); errPlayers != nil {
			return errPlayers
		}

		return nil
	})
}

func (r *speedrunRepository) insertPlayers(ctx context.Context, tx pgx.Tx, speedrunID int, players []domain.SpeedrunRunner) error {
	for _, runner := range players {
		query, args, errQuery := r.db.Builder().
			Insert("speedrun_runners").
			SetMap(map[string]interface{}{
				"speedrun_id": speedrunID,
				"steam_id":    runner.SteamID.Int64(),
				"duration":    runner.Duration,
			}).
			ToSql()
		if errQuery != nil {
			return r.db.DBErr(errQuery)
		}

		if _, errExec := tx.Exec(ctx, query, args...); errExec != nil {
			return r.db.DBErr(errExec)
		}
	}

	return nil
}

func (r *speedrunRepository) insertRounds(ctx context.Context, tx pgx.Tx, speedrunID int, rounds []domain.SpeedrunRound) error {
	for roundNum, round := range rounds {
		query, args, errQuery := r.db.Builder().
			Insert("speedrun_rounds").
			SetMap(map[string]interface{}{
				"speedrun_id":  speedrunID,
				"round_number": roundNum + 1,
				"duration":     round.Duration,
			}).
			Suffix(" RETURNING round_id").
			ToSql()
		if errQuery != nil {
			return r.db.DBErr(errQuery)
		}

		if errExec := tx.QueryRow(ctx, query, args...).Scan(&round.RoundID); errExec != nil {
			return r.db.DBErr(errExec)
		}

		if errPlayers := r.insertRoundPlayers(ctx, tx, round.RoundID, round.Players); errPlayers != nil {
			return errPlayers
		}
	}

	return nil
}

func (r *speedrunRepository) insertRoundPlayers(ctx context.Context, tx pgx.Tx, roundID int, players []domain.SpeedrunRunner) error {
	for _, runner := range players {
		query, args, errQuery := r.db.Builder().
			Insert("speedrun_rounds_runners").
			SetMap(map[string]interface{}{
				"round_id": roundID,
				"steam_id": runner.SteamID.Int64(),
				"duration": runner.Duration,
			}).
			ToSql()
		if errQuery != nil {
			return r.db.DBErr(errQuery)
		}

		if _, errExec := tx.Exec(ctx, query, args...); errExec != nil {
			return r.db.DBErr(errExec)
		}
	}

	return nil
}

func (r *speedrunRepository) Query(ctx context.Context, query domain.SpeedrunQuery) ([]domain.Speedrun, error) {
	return nil, errors.New("error")
}
