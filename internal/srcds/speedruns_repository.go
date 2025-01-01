package srcds

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

func NewSpeedrunRepository(database database.Database, people domain.PersonUsecase) domain.SpeedrunRepository {
	return &speedrunRepository{db: database, people: people}
}

type speedrunRepository struct {
	db     database.Database
	people domain.PersonUsecase
}

func (r *speedrunRepository) LoadOrCreateMap(ctx context.Context, transaction pgx.Tx, mapName string) (domain.MapDetail, error) {
	const query = `
		WITH ins AS (
    		INSERT INTO map (map_id, map_name, updated_on, created_on) VALUES (DEFAULT, lower($1), now(),now())
    		ON CONFLICT (map_name) DO NOTHING RETURNING *
    	)
		SELECT * FROM ins
		UNION 
		SELECT * FROM map
		WHERE map_name = lower($1);
		`
	var mapDetail domain.MapDetail
	if errQuery := r.db.
		QueryRow(ctx, transaction, query, mapName).
		Scan(&mapDetail.MapID, &mapDetail.MapName, &mapDetail.UpdatedOn, &mapDetail.CreatedOn); errQuery != nil {
		return domain.MapDetail{}, r.db.DBErr(errQuery)
	}

	return mapDetail, nil
}

func (r *speedrunRepository) Save(ctx context.Context, details *domain.Speedrun) error {
	return r.db.WrapTx(ctx, func(transaction pgx.Tx) error {
		mapDetail, mapErr := r.LoadOrCreateMap(ctx, transaction, details.MapDetail.MapName)
		if mapErr != nil {
			return mapErr
		}
		details.MapDetail = mapDetail

		if errPlayers := r.insertPlayers(ctx, transaction, details.Players); errPlayers != nil {
			return r.db.DBErr(errPlayers)
		}

		for _, point := range details.PointCaptures {
			if errPlayers := r.insertPlayers(ctx, transaction, point.Players); errPlayers != nil {
				return r.db.DBErr(errPlayers)
			}
		}

		query, args, errQuery := r.db.Builder().
			Insert("speedrun").
			SetMap(map[string]interface{}{
				"server_id":    details.ServerID,
				"map_id":       details.MapDetail.MapID,
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

		if errScan := r.db.QueryRow(ctx, transaction, query, args...).Scan(&details.SpeedrunID); errScan != nil {
			return r.db.DBErr(errScan)
		}

		if errRounds := r.insertCaptures(ctx, transaction, details.SpeedrunID, details.PointCaptures); errRounds != nil {
			return errRounds
		}

		if errPlayers := r.insertRunners(ctx, transaction, details.SpeedrunID, details.Players); errPlayers != nil {
			return errPlayers
		}

		return nil
	})
}

func (r *speedrunRepository) insertPlayers(ctx context.Context, transaction pgx.Tx, players []domain.SpeedrunParticipant) error {
	for _, player := range players {
		if _, errPlayer := r.people.GetOrCreatePersonBySteamID(ctx, transaction, player.SteamID); errPlayer != nil {
			return errPlayer
		}
	}

	return nil
}

func (r *speedrunRepository) insertRunners(ctx context.Context, transaction pgx.Tx, speedrunID int, players []domain.SpeedrunParticipant) error {
	// TODO use pgx.Batch
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

		if _, errExec := transaction.Exec(ctx, query, args...); errExec != nil {
			return r.db.DBErr(errExec)
		}
	}

	return nil
}

func (r *speedrunRepository) insertCaptures(ctx context.Context, transaction pgx.Tx, speedrunID int, rounds []domain.SpeedrunPointCaptures) error {
	// TODO use pgx.Batch
	for roundNum, round := range rounds {
		query, args, errQuery := r.db.Builder().
			Insert("speedrun_capture").
			SetMap(map[string]interface{}{
				"speedrun_id": speedrunID,
				"round_id":    roundNum + 1,
				"duration":    round.Duration,
				"point_name":  round.PointName,
			}).
			Suffix(" RETURNING round_id").
			ToSql()
		if errQuery != nil {
			return r.db.DBErr(errQuery)
		}

		if errExec := r.db.QueryRow(ctx, transaction, query, args...).Scan(&round.RoundID); errExec != nil {
			return r.db.DBErr(errExec)
		}

		if errPlayers := r.insertCapturePlayers(ctx, transaction, speedrunID, roundNum+1, round.Players); errPlayers != nil {
			return errPlayers
		}
	}

	return nil
}

func (r *speedrunRepository) insertCapturePlayers(ctx context.Context, transaction pgx.Tx, speedrunID int, roundID int, players []domain.SpeedrunParticipant) error {
	// TODO use pgx.Batch
	for _, runner := range players {
		query, args, errQuery := r.db.Builder().
			Insert("speedrun_capture_runners").
			SetMap(map[string]interface{}{
				"speedrun_id": speedrunID,
				"round_id":    roundID,
				"steam_id":    runner.SteamID.Int64(),
				"duration":    runner.Duration,
			}).
			ToSql()
		if errQuery != nil {
			return r.db.DBErr(errQuery)
		}

		if errExec := r.db.Exec(ctx, transaction, query, args...); errExec != nil {
			return r.db.DBErr(errExec)
		}
	}

	return nil
}

func (r *speedrunRepository) Query(_ context.Context, _ domain.SpeedrunQuery) ([]domain.Speedrun, error) {
	return []domain.Speedrun{}, nil
}

func (r *speedrunRepository) TopNOverall(ctx context.Context, count int) (map[string][]domain.Speedrun, error) {
	const q = `
		SELECT
			*
		FROM
			(SELECT
				 s.speedrun_id, s.server_id, s.category, s.duration, s.player_count, s.bot_count, s.created_on,
				 rank() OVER (PARTITION BY s.map_id ORDER BY duration ) as rank,
				 m.map_id, m.map_name, m.updated_on, m.created_on
			 FROM speedrun s
					  LEFT JOIN map m ON m.map_id = s.map_id
			) s
		WHERE s.rank <= $1
	`
	rows, errRows := r.db.Query(ctx, nil, q, count)
	if errRows != nil {
		return nil, r.db.DBErr(errRows)
	}
	defer rows.Next()

	runs := map[string][]domain.Speedrun{}
	for rows.Next() {
		var sr domain.Speedrun
		if err := rows.Scan(
			&sr.SpeedrunID, &sr.ServerID, &sr.Category, &sr.Duration, &sr.PlayerCount, &sr.BotCount, &sr.CreatedOn,
			&sr.Rank,
			&sr.MapDetail.MapID, &sr.MapDetail.MapName, &sr.MapDetail.UpdatedOn, &sr.MapDetail.CreatedOn); err != nil {
			return nil, r.db.DBErr(err)
		}
		if _, ok := runs[sr.MapDetail.MapName]; !ok {
			runs[sr.MapDetail.MapName] = []domain.Speedrun{}
		}

		runs[sr.MapDetail.MapName] = append(runs[sr.MapDetail.MapName], sr)
	}

	// TODO this is quite expensive, cache or change to single query
	for _, speedruns := range runs {
		for i := range speedruns {
			runners, errRunners := r.getRunners(ctx, speedruns[i].SpeedrunID)
			if errRunners != nil {
				return nil, errRunners
			}
			speedruns[i].Players = runners

			captures, errCaptures := r.getCaptures(ctx, speedruns[i].SpeedrunID)
			if errCaptures != nil {
				return nil, errCaptures
			}
			speedruns[i].PointCaptures = captures
		}
	}

	return runs, nil
}

func (r *speedrunRepository) ByID(ctx context.Context, speedrunID int) (domain.Speedrun, error) {
	const q = `
		SELECT s.speedrun_id, s.server_id, s.category, s.duration, s.player_count, s.bot_count, s.created_on,
		       m.map_id, m.map_name, m.updated_on, m.created_on
		FROM speedrun s
		LEFT JOIN public.map m on s.map_id = m.map_id
		WHERE speedrun_id = $1`

	var sr domain.Speedrun
	if err := r.db.
		QueryRow(ctx, nil, q, speedrunID).
		Scan(&sr.SpeedrunID, &sr.ServerID, &sr.Category, &sr.Duration, &sr.PlayerCount, &sr.BotCount, &sr.CreatedOn,
			&sr.MapDetail.MapID, &sr.MapDetail.MapName, &sr.MapDetail.UpdatedOn, &sr.MapDetail.CreatedOn); err != nil {
		return domain.Speedrun{}, r.db.DBErr(err)
	}

	runners, errRunners := r.getRunners(ctx, speedrunID)
	if errRunners != nil {
		return sr, errRunners
	}

	captures, errCaptures := r.getCaptures(ctx, speedrunID)
	if errCaptures != nil {
		return sr, errCaptures
	}

	sr.Players = runners
	sr.PointCaptures = captures

	return sr, nil
}

func (r *speedrunRepository) getCapturedPoints(ctx context.Context, speedrunID int) ([]domain.SpeedrunPointCaptures, error) {
	const q = `
		SELECT c.speedrun_id, c.round_id, c.duration, c.point_name
		FROM speedrun_capture c
		WHERE c.speedrun_id = $1
		ORDER BY c.round_id`
	rows, errRows := r.db.Query(ctx, nil, q, speedrunID)
	if errRows != nil {
		return nil, r.db.DBErr(errRows)
	}

	defer rows.Close()

	var captures []domain.SpeedrunPointCaptures
	for rows.Next() {
		var c domain.SpeedrunPointCaptures
		if err := rows.Scan(&c.SpeedrunID, &c.RoundID, &c.Duration, &c.PointName); err != nil {
			return nil, r.db.DBErr(err)
		}

		captures = append(captures, c)
	}

	return captures, nil
}

func (r *speedrunRepository) getCaptures(ctx context.Context, speedrunID int) ([]domain.SpeedrunPointCaptures, error) {
	points, errPoints := r.getCapturedPoints(ctx, speedrunID)
	if errPoints != nil {
		return nil, errPoints
	}

	const q = `
		SELECT r.round_id, r.steam_id, r.duration
		FROM speedrun_capture_runners r
		WHERE r.speedrun_id = $1
		ORDER BY r.round_id`
	rows, errRows := r.db.Query(ctx, nil, q, speedrunID)
	if errRows != nil {
		return nil, r.db.DBErr(errRows)
	}

	defer rows.Close()

	var participants []domain.SpeedrunParticipant

	for rows.Next() {
		var (
			participant domain.SpeedrunParticipant
			sid         int64
		)
		if err := rows.Scan(&participant.RoundID, &sid, &participant.Duration); err != nil {
			return nil, r.db.DBErr(err)
		}

		participant.SteamID = steamid.New(sid)

		participants = append(participants, participant)
	}

	for _, participant := range participants {
		for i := range points {
			if points[i].RoundID == participant.RoundID {
				points[i].Players = append(points[i].Players, participant)
			}
		}
	}

	return points, nil
}

func (r *speedrunRepository) getRunners(ctx context.Context, speedrunID int) ([]domain.SpeedrunParticipant, error) {
	const q = `
		SELECT r.steam_id, r.duration 
		FROM speedrun_runners r
		WHERE speedrun_id = $1`
	rows, errRows := r.db.Query(ctx, nil, q, speedrunID)
	if errRows != nil {
		return nil, r.db.DBErr(errRows)
	}
	defer rows.Close()

	var participants []domain.SpeedrunParticipant

	for rows.Next() {
		var (
			participant domain.SpeedrunParticipant
			sid         int64
		)

		if err := rows.Scan(&sid, &participant.Duration); err != nil {
			return nil, r.db.DBErr(err)
		}
		participant.SteamID = steamid.New(sid)

		participants = append(participants, participant)
	}

	return participants, nil
}
