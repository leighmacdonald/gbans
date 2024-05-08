package votes

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type voteRepository struct {
	db database.Database
}

func NewVoteRepository(database database.Database) domain.VoteRepository {
	return &voteRepository{db: database}
}

func (r voteRepository) Query(ctx context.Context, filter domain.VoteQueryFilter) ([]domain.VoteResult, int64, error) {
	var constraints sq.And

	if sid, ok := filter.SourceIDField.SourceSteamID(ctx); ok {
		constraints = append(constraints, sq.Eq{"source_id": sid.Int64()})
	}

	if sid, ok := filter.TargetIDField.TargetSteamID(ctx); ok {
		constraints = append(constraints, sq.Eq{"target_id": sid.Int64()})
	}

	if filter.ServerID > 0 {
		constraints = append(constraints, sq.Eq{"server_id": filter.ServerID})
	}

	if filter.Name != "" {
		constraints = append(constraints, sq.Eq{"name": filter.Name})
	}

	if filter.Success >= 0 {
		constraints = append(constraints, sq.Eq{"success": filter.Success == 1})
	}

	builder := r.db.Builder().
		Select("v.vote_id", "v.server_id", "v.source_id",
			"src.personaname", "src.avatarhash", "v.target_id", "tgt.personaname", "tgt.avatarhash",
			"v.name", "v.success", "v.created_on", "s.short_name").
		From("vote_result v").
		LeftJoin("server s USING(server_id)").
		LeftJoin("person src ON v.source_id = src.steam_id").
		LeftJoin("person tgt ON v.target_id = tgt.steam_id")

	builder = builder.Where(constraints)
	builder = filter.ApplyLimitOffsetDefault(builder)
	builder = filter.ApplySafeOrder(builder, map[string][]string{
		"v.":   {"vote_id", "server_id", "source_id", "target_id", "name", "created_on"},
		"tgt.": {"personaname"},
		"src.": {"personaname"},
	}, "vote_id")

	rows, errRows := r.db.QueryBuilder(ctx, builder)
	if errRows != nil {
		return nil, 0, r.db.DBErr(errRows)
	}
	defer rows.Close()

	var results []domain.VoteResult

	for rows.Next() {
		var (
			sourceID *int64
			targetID *int64
			result   domain.VoteResult
		)

		if errScan := rows.Scan(&result.VoteID, &result.ServerID,
			&sourceID, &result.SourceName, &result.SourceAvatarHash,
			&targetID, &result.TargetName, &result.TargetAvatarHash,
			&result.Name, &result.Success, &result.CreatedOn, &result.ServerName); errScan != nil {
			return nil, 0, r.db.DBErr(errScan)
		}

		result.SourceID = steamid.New(*sourceID)
		if targetID != nil {
			result.TargetID = steamid.New(*targetID)
		}

		results = append(results, result)
	}

	count, errCount := r.db.GetCount(ctx, r.db.Builder().
		Select("COUNT(v.vote_id)").
		From("vote_result v").
		Where(constraints))
	if errCount != nil {
		return nil, 0, r.db.DBErr(errCount)
	}

	return results, count, nil
}

func (r voteRepository) AddResult(ctx context.Context, voteResult domain.VoteResult) error {
	return r.db.DBErr(r.db.ExecInsertBuilder(ctx, r.db.Builder().
		Insert("vote_result").
		SetMap(map[string]interface{}{
			"server_id":  voteResult.ServerID,
			"source_id":  voteResult.SourceID,
			"target_id":  voteResult.TargetID,
			"success":    voteResult.Success,
			"name":       voteResult.Name,
			"code":       voteResult.Code,
			"created_on": voteResult.CreatedOn,
		})))
}
