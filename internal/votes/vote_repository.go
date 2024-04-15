package votes

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type voteRepository struct {
	db            database.Database
	personUsecase domain.PersonUsecase
	matchUsecase  domain.MatchUsecase
	broadcaster   *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]
}

func NewVoteRepository(database database.Database, personUsecase domain.PersonUsecase, matchUsecase domain.MatchUsecase, broadcaster *fp.Broadcaster[logparse.EventType, logparse.ServerEvent]) domain.VoteRepository {
	return &voteRepository{
		db:            database,
		personUsecase: personUsecase,
		matchUsecase:  matchUsecase,
		broadcaster:   broadcaster,
	}
}

func (r voteRepository) Query(ctx context.Context, filter domain.VoteQueryFilter) ([]domain.VoteResult, error) {
	var constraints sq.And

	if filter.SourceID.Valid() {
		constraints = append(constraints, sq.Eq{"source_id": filter.SourceID.Int64()})
	}
	if filter.TargetID.Valid() {
		constraints = append(constraints, sq.Eq{"target_id": filter.TargetID.Int64()})
	}
	if !filter.MatchID.IsNil() {
		constraints = append(constraints, sq.Eq{"match_id": filter.MatchID.String()})
	}
	if filter.ServerID > 0 {
		constraints = append(constraints, sq.Eq{"server_id": filter.ServerID})
	}
	if filter.Name != "" {
		constraints = append(constraints, sq.Eq{"name": filter.Name})
	}

	builder := r.db.Builder().
		Select("v.vote_id", "v.server_id", "v.match_id", "v.source_id", "v.target_id", "v.valid", "v.name", "v.created_on").
		From("vote_result v")

	builder = builder.Where(constraints)
	builder = filter.ApplyLimitOffsetDefault(builder)
	builder = filter.ApplySafeOrder(builder, map[string][]string{
		"v.": {"vote_id", "server_id", "match_id", "source_id", "target_id", "valid", "name", "created_on"},
	}, "vote_id")

	rows, errRows := r.db.QueryBuilder(ctx, builder)
	if errRows != nil {
		return nil, r.db.DBErr(errRows)
	}
	defer rows.Close()

	var results []domain.VoteResult

	for rows.Next() {
		var (
			sourceID *int64
			targetID *int64
			result   domain.VoteResult
		)
		if errScan := rows.Scan(&result.ServerID, &result.MatchID, &sourceID, &targetID, &result.Valid, &result.Name, &result.CreatedOn); errScan != nil {
			return nil, r.db.DBErr(errScan)
		}

		result.SourceID = steamid.New(*sourceID)
		if targetID != nil {
			result.TargetID = steamid.New(*targetID)
		}

		results = append(results, result)
	}

	return results, nil
}

func (r voteRepository) AddResult(ctx context.Context, voteResult domain.VoteResult) error {
	return r.db.DBErr(r.db.ExecInsertBuilder(ctx, r.db.Builder().
		Insert("vote_result").
		SetMap(map[string]interface{}{
			"server_id": voteResult.ServerID,
			"match_id":  voteResult.MatchID,
			"source_id": voteResult.SourceID,
			"target_id": voteResult.TargetID,
			"valid":     voteResult.Valid,
			"name":      voteResult.Name,
		})))
}
