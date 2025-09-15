package servers

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type SpeedrunInterval int

const (
	Daily   SpeedrunInterval = 86400
	Weekly                   = Daily * 7
	Monthly                  = Weekly * 7
	Yearly                   = Monthly * 12
	AllTime                  = -1
)

type SpeedrunCategory string

const (
	Mode24v40 SpeedrunCategory = "24_40"
)

type MapDetail struct {
	MapID     int       `json:"map_id"`
	MapName   string    `json:"map_name"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

type SpeedrunQuery struct {
	Map      string           `json:"map"`
	Interval SpeedrunInterval `json:"interval"`
	Count    int              `json:"count"`
}

type SpeedrunPointCaptures struct {
	SpeedrunID int                   `json:"speedrun_id"`
	RoundID    int                   `json:"round_id"`
	Players    []SpeedrunParticipant `json:"players"`
	Duration   time.Duration         `json:"duration"`
	PointName  string                `json:"point_name"`
}

type Speedrun struct {
	SpeedrunID    int                     `json:"speedrun_id"`
	ServerID      int                     `json:"server_id"`
	Rank          int                     `json:"rank,omitempty"`
	InitialRank   int                     `json:"initial_rank,omitempty"`
	MapDetail     MapDetail               `json:"map_detail"`
	PointCaptures []SpeedrunPointCaptures `json:"point_captures"`
	Players       []SpeedrunParticipant   `json:"players"`
	Duration      time.Duration           `json:"duration"`
	PlayerCount   int                     `json:"player_count"`
	BotCount      int                     `json:"bot_count"`
	CreatedOn     time.Time               `json:"created_on"`
	Category      SpeedrunCategory        `json:"category"`
	TotalPlayers  int                     `json:"total_players"`
}

type SpeedrunParticipant struct {
	RoundID     int             `json:"round_id"`
	SteamID     steamid.SteamID `json:"steam_id"`
	Duration    time.Duration   `json:"duration"`
	AvatarHash  string          `json:"avatar_hash"`
	PersonaName string          `json:"persona_name"`
}

type SpeedrunMapOverview struct {
	SpeedrunID   int              `json:"speedrun_id"`
	ServerID     int              `json:"server_id"`
	Rank         int              `json:"rank"`
	InitialRank  int              `json:"initial_rank"`
	MapDetail    MapDetail        `json:"map_detail"`
	Duration     time.Duration    `json:"duration"`
	PlayerCount  int              `json:"player_count"`
	BotCount     int              `json:"bot_count"`
	CreatedOn    time.Time        `json:"created_on"`
	Category     SpeedrunCategory `json:"category"`
	TotalPlayers int              `json:"total_players"`
}

func NewSpeedrunRepository(database database.Database, people PersonProvider) SpeedrunRepository {
	return SpeedrunRepository{db: database, people: people}
}

type SpeedrunRepository struct {
	db     database.Database
	people PersonProvider
}

func (r *SpeedrunRepository) LoadOrCreateMap(ctx context.Context, transaction pgx.Tx, mapName string) (MapDetail, error) {
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
	var mapDetail MapDetail
	if errQuery := r.db.
		QueryRow(ctx, transaction, query, mapName).
		Scan(&mapDetail.MapID, &mapDetail.MapName, &mapDetail.UpdatedOn, &mapDetail.CreatedOn); errQuery != nil {
		return MapDetail{}, database.DBErr(errQuery)
	}

	return mapDetail, nil
}

func (r *SpeedrunRepository) Save(ctx context.Context, details *Speedrun) error {
	return r.db.WrapTx(ctx, func(transaction pgx.Tx) error {
		mapDetail, mapErr := r.LoadOrCreateMap(ctx, transaction, details.MapDetail.MapName)
		if mapErr != nil {
			return mapErr
		}
		details.MapDetail = mapDetail

		if errPlayers := r.insertPlayers(ctx, transaction, details.Players); errPlayers != nil {
			return database.DBErr(errPlayers)
		}

		for _, point := range details.PointCaptures {
			if errPlayers := r.insertPlayers(ctx, transaction, point.Players); errPlayers != nil {
				return database.DBErr(errPlayers)
			}
		}

		query, args, errQuery := r.db.Builder().
			Insert("speedrun").
			SetMap(map[string]any{
				"server_id":    details.ServerID,
				"map_id":       details.MapDetail.MapID,
				"category":     details.Category,
				"duration":     details.Duration,
				"initial_rank": details.InitialRank,
				"player_count": details.PlayerCount,
				"bot_count":    details.BotCount,
				"created_on":   details.CreatedOn,
			}).
			Suffix(" RETURNING speedrun_id").
			ToSql()
		if errQuery != nil {
			return database.DBErr(errQuery)
		}

		if errScan := r.db.QueryRow(ctx, transaction, query, args...).Scan(&details.SpeedrunID); errScan != nil {
			return database.DBErr(errScan)
		}

		if errRounds := r.insertCaptures(ctx, transaction, details.SpeedrunID, details.PointCaptures); errRounds != nil {
			return errRounds
		}

		if errPlayers := r.insertRunners(ctx, transaction, details.SpeedrunID, details.Players); errPlayers != nil {
			return errPlayers
		}

		rank, errRank := r.updateSpeedrunRank(ctx, transaction, details.SpeedrunID)
		if errRank != nil {
			return errRank
		}

		details.Rank = rank

		return nil
	})
}

func (r *SpeedrunRepository) updateSpeedrunRank(ctx context.Context, transaction pgx.Tx, speedrunID int) (int, error) {
	const query = `
		SELECT rank
		FROM (
			 SELECT speedrun_id, rank() OVER (PARTITION BY s.map_id ORDER BY duration ) as rank
			 FROM speedrun s
			 LEFT JOIN map m on s.map_id = m.map_id
		 ) s
		WHERE speedrun_id = $1;`

	var rank int
	if err := transaction.QueryRow(ctx, query, speedrunID).Scan(&rank); err != nil {
		return 0, database.DBErr(err)
	}

	const queryUpdate = `UPDATE speedrun SET initial_rank = $1 WHERE speedrun_id = $2`
	if _, err := transaction.Exec(ctx, queryUpdate, rank, speedrunID); err != nil {
		return 0, database.DBErr(err)
	}

	return rank, nil
}

func (r *SpeedrunRepository) insertPlayers(ctx context.Context, transaction pgx.Tx, players []SpeedrunParticipant) error {
	for _, player := range players {
		if _, errPlayer := r.people.GetOrCreatePersonBySteamID(ctx, transaction, player.SteamID); errPlayer != nil {
			return errPlayer
		}
	}

	return nil
}

func (r *SpeedrunRepository) insertRunners(ctx context.Context, transaction pgx.Tx, speedrunID int, players []SpeedrunParticipant) error {
	// TODO use pgx.Batch
	for _, runner := range players {
		query, args, errQuery := r.db.Builder().
			Insert("speedrun_runners").
			SetMap(map[string]any{
				"speedrun_id": speedrunID,
				"steam_id":    runner.SteamID.Int64(),
				"duration":    runner.Duration,
			}).
			ToSql()
		if errQuery != nil {
			return database.DBErr(errQuery)
		}

		if _, errExec := transaction.Exec(ctx, query, args...); errExec != nil {
			return database.DBErr(errExec)
		}
	}

	return nil
}

func (r *SpeedrunRepository) insertCaptures(ctx context.Context, transaction pgx.Tx, speedrunID int, rounds []SpeedrunPointCaptures) error {
	// TODO use pgx.Batch
	for roundNum, round := range rounds {
		query, args, errQuery := r.db.Builder().
			Insert("speedrun_capture").
			SetMap(map[string]any{
				"speedrun_id": speedrunID,
				"round_id":    roundNum + 1,
				"duration":    round.Duration,
				"point_name":  round.PointName,
			}).
			Suffix(" RETURNING round_id").
			ToSql()
		if errQuery != nil {
			return database.DBErr(errQuery)
		}

		if errExec := r.db.QueryRow(ctx, transaction, query, args...).Scan(&round.RoundID); errExec != nil {
			return database.DBErr(errExec)
		}

		if errPlayers := r.insertCapturePlayers(ctx, transaction, speedrunID, roundNum+1, round.Players); errPlayers != nil {
			return errPlayers
		}
	}

	return nil
}

func (r *SpeedrunRepository) insertCapturePlayers(ctx context.Context, transaction pgx.Tx, speedrunID int, roundID int, players []SpeedrunParticipant) error {
	// TODO use pgx.Batch
	for _, runner := range players {
		query, args, errQuery := r.db.Builder().
			Insert("speedrun_capture_runners").
			SetMap(map[string]any{
				"speedrun_id": speedrunID,
				"round_id":    roundID,
				"steam_id":    runner.SteamID.Int64(),
				"duration":    runner.Duration,
			}).
			ToSql()
		if errQuery != nil {
			return database.DBErr(errQuery)
		}

		if errExec := r.db.Exec(ctx, transaction, query, args...); errExec != nil {
			return database.DBErr(errExec)
		}
	}

	return nil
}

func (r *SpeedrunRepository) Query(_ context.Context, _ SpeedrunQuery) ([]Speedrun, error) {
	return []Speedrun{}, nil
}

func (r *SpeedrunRepository) TopNOverall(ctx context.Context, count int) (map[string][]Speedrun, error) {
	const query = `
		SELECT
			*
		FROM
			(SELECT
				 s.speedrun_id, s.server_id, s.category, s.duration, s.player_count, s.bot_count, s.created_on, s.initial_rank,
				 rank() OVER (PARTITION BY s.map_id ORDER BY duration ) as rank,
				 m.map_id, m.map_name, m.updated_on, m.created_on
			 FROM speedrun s
					  LEFT JOIN map m ON m.map_id = s.map_id
			) s
		WHERE s.rank <= $1
	`
	rows, errRows := r.db.Query(ctx, nil, query, count)
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}
	defer rows.Next()

	runs := map[string][]Speedrun{}
	for rows.Next() {
		var run Speedrun
		if err := rows.Scan(
			&run.SpeedrunID, &run.ServerID, &run.Category, &run.Duration, &run.PlayerCount, &run.BotCount, &run.CreatedOn,
			&run.InitialRank, &run.Rank,
			&run.MapDetail.MapID, &run.MapDetail.MapName, &run.MapDetail.UpdatedOn, &run.MapDetail.CreatedOn); err != nil {
			return nil, database.DBErr(err)
		}
		if _, ok := runs[run.MapDetail.MapName]; !ok {
			runs[run.MapDetail.MapName] = []Speedrun{}
		}

		runs[run.MapDetail.MapName] = append(runs[run.MapDetail.MapName], run)
	}

	// TODO this is quite expensive, cache or change to single query
	for _, speedruns := range runs {
		for runnerIdx := range speedruns {
			runners, errRunners := r.getRunners(ctx, speedruns[runnerIdx].SpeedrunID)
			if errRunners != nil {
				return nil, errRunners
			}
			speedruns[runnerIdx].Players = runners

			captures, errCaptures := r.getCaptures(ctx, speedruns[runnerIdx].SpeedrunID)
			if errCaptures != nil {
				return nil, errCaptures
			}
			speedruns[runnerIdx].PointCaptures = captures
		}
	}

	return runs, nil
}

func (r *SpeedrunRepository) ByID(ctx context.Context, speedrunID int) (Speedrun, error) {
	const query = `
		SELECT *
		FROM (
			SELECT s.speedrun_id, s.server_id, s.category, s.duration, s.player_count, s.bot_count, s.created_on, s.initial_rank,
				m.map_id, m.map_name, m.updated_on, m.created_on,
				rank() OVER (PARTITION BY s.map_id ORDER BY duration ) as rank
			FROM speedrun s
			LEFT JOIN public.map m on s.map_id = m.map_id
		) s
		WHERE speedrun_id =  $1`

	var run Speedrun
	if err := r.db.
		QueryRow(ctx, nil, query, speedrunID).
		Scan(&run.SpeedrunID, &run.ServerID, &run.Category, &run.Duration, &run.PlayerCount, &run.BotCount, &run.CreatedOn, &run.InitialRank,
			&run.MapDetail.MapID, &run.MapDetail.MapName, &run.MapDetail.UpdatedOn, &run.MapDetail.CreatedOn, &run.Rank); err != nil {
		return Speedrun{}, database.DBErr(err)
	}

	runners, errRunners := r.getRunners(ctx, speedrunID)
	if errRunners != nil {
		return run, errRunners
	}

	captures, errCaptures := r.getCaptures(ctx, speedrunID)
	if errCaptures != nil {
		return run, errCaptures
	}

	run.Players = runners
	run.PointCaptures = captures

	return run, nil
}

func (r *SpeedrunRepository) Recent(ctx context.Context, limit int) ([]SpeedrunMapOverview, error) {
	const query = `
		SELECT s.*,
			   r.count,
			   m.map_name
		FROM (SELECT s.speedrun_id,
					 s.map_id,
					 s.server_id,
					 s.category,
					 s.duration,
					 s.player_count,
					 s.bot_count,
					 s.created_on,
					 s.initial_rank,
					 rank() OVER (PARTITION BY s.map_id ORDER BY s.duration ) as rank
			  FROM speedrun s) s
				 LEFT JOIN (SELECT speedrun_id, COUNT(r.steam_id) as count
							FROM speedrun_runners r
							GROUP BY speedrun_id) r ON s.speedrun_id = r.speedrun_id
		LEFT JOIN map m ON m.map_id = s.map_id
		ORDER BY s.created_on DESC
		LIMIT $1`
	rows, errRows := r.db.Query(ctx, nil, query, limit)
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}
	defer rows.Close()

	var smo []SpeedrunMapOverview
	for rows.Next() {
		var run SpeedrunMapOverview
		if err := rows.Scan(&run.SpeedrunID, &run.MapDetail.MapID, &run.ServerID, &run.Category,
			&run.Duration, &run.PlayerCount, &run.BotCount, &run.CreatedOn, &run.InitialRank,
			&run.Rank, &run.PlayerCount, &run.MapDetail.MapName); err != nil {
			return []SpeedrunMapOverview{}, database.DBErr(err)
		}
		smo = append(smo, run)
	}

	return smo, nil
}

func (r *SpeedrunRepository) ByMap(ctx context.Context, mapName string) ([]SpeedrunMapOverview, error) {
	const query = `
		SELECT s.*,
			   r.count,
			   m.map_name
		FROM (SELECT s.speedrun_id,
					 s.map_id,
					 s.server_id,
					 s.category,
					 s.duration,
					 s.player_count,
					 s.bot_count,
					 s.created_on,
					 s.initial_rank,
					 rank() OVER (PARTITION BY s.map_id ORDER BY s.duration ) as rank
			  FROM speedrun s) s
				 LEFT JOIN (SELECT speedrun_id, COUNT(r.steam_id) as count
							FROM speedrun_runners r
							GROUP BY speedrun_id) r ON s.speedrun_id = r.speedrun_id
		LEFT JOIN map m ON m.map_id = s.map_id
		WHERE m.map_name = lower($1)
		ORDER BY rank`
	rows, errRows := r.db.Query(ctx, nil, query, mapName)
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}
	defer rows.Close()

	var smo []SpeedrunMapOverview
	for rows.Next() {
		var run SpeedrunMapOverview
		if err := rows.Scan(&run.SpeedrunID, &run.MapDetail.MapID, &run.ServerID, &run.Category,
			&run.Duration, &run.PlayerCount, &run.BotCount, &run.CreatedOn, &run.InitialRank,
			&run.Rank, &run.PlayerCount, &run.MapDetail.MapName); err != nil {
			return []SpeedrunMapOverview{}, database.DBErr(err)
		}
		smo = append(smo, run)
	}

	return smo, nil
}

func (r *SpeedrunRepository) getCapturedPoints(ctx context.Context, speedrunID int) ([]SpeedrunPointCaptures, error) {
	const q = `
		SELECT c.speedrun_id, c.round_id, c.duration, c.point_name
		FROM speedrun_capture c
		WHERE c.speedrun_id = $1
		ORDER BY c.round_id`
	rows, errRows := r.db.Query(ctx, nil, q, speedrunID)
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}

	defer rows.Close()

	var captures []SpeedrunPointCaptures
	for rows.Next() {
		var c SpeedrunPointCaptures
		if err := rows.Scan(&c.SpeedrunID, &c.RoundID, &c.Duration, &c.PointName); err != nil {
			return nil, database.DBErr(err)
		}

		captures = append(captures, c)
	}

	return captures, nil
}

func (r *SpeedrunRepository) getCaptures(ctx context.Context, speedrunID int) ([]SpeedrunPointCaptures, error) {
	points, errPoints := r.getCapturedPoints(ctx, speedrunID)
	if errPoints != nil {
		return nil, errPoints
	}

	const query = `
		SELECT r.round_id, r.steam_id, r.duration, p.avatarhash, p.personaname
		FROM speedrun_capture_runners r
		LEFT JOIN person p USING (steam_id)
		WHERE r.speedrun_id = $1
		ORDER BY r.round_id`
	rows, errRows := r.db.Query(ctx, nil, query, speedrunID)
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}

	defer rows.Close()

	var participants []SpeedrunParticipant

	for rows.Next() {
		var (
			participant SpeedrunParticipant
			sid         int64
		)
		if err := rows.Scan(&participant.RoundID, &sid, &participant.Duration, &participant.AvatarHash, &participant.PersonaName); err != nil {
			return nil, database.DBErr(err)
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

func (r *SpeedrunRepository) getRunners(ctx context.Context, speedrunID int) ([]SpeedrunParticipant, error) {
	const q = `
		SELECT r.steam_id, r.duration, p.avatarhash, p.personaname
		FROM speedrun_runners r
		LEFT OUTER JOIN person p USING(steam_id)
		WHERE speedrun_id = $1`
	rows, errRows := r.db.Query(ctx, nil, q, speedrunID)
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}
	defer rows.Close()

	var participants []SpeedrunParticipant

	for rows.Next() {
		var (
			participant SpeedrunParticipant
			sid         int64
		)

		if err := rows.Scan(&sid, &participant.Duration, &participant.AvatarHash, &participant.PersonaName); err != nil {
			return nil, database.DBErr(err)
		}
		participant.SteamID = steamid.New(sid)

		participants = append(participants, participant)
	}

	return participants, nil
}
