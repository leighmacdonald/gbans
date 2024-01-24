package store

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

var ErrInvalidContestID = errors.New("invalid contest id provided")

func (s Stores) ContestByID(ctx context.Context, contestID uuid.UUID, contest *domain.Contest) error {
	if contestID.IsNil() {
		return ErrInvalidContestID
	}

	query := s.
		Builder().
		Select("contest_id", "title", "public", "description", "date_start",
			"date_end", "max_submissions", "media_types", "deleted", "voting", "min_permission_level", "down_votes",
			"created_on", "updated_on", "hide_submissions").
		From("contest").
		Where(sq.And{sq.Eq{"deleted": false}, sq.Eq{"contest_id": contestID.String()}})

	row, errQuery := s.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return errs.DBErr(row.Scan(&contest.ContestID, &contest.Title, &contest.Public, &contest.Description,
		&contest.DateStart, &contest.DateEnd, &contest.MaxSubmissions, &contest.MediaTypes,
		&contest.Deleted, &contest.Voting, &contest.MinPermissionLevel, &contest.DownVotes,
		&contest.CreatedOn, &contest.UpdatedOn, &contest.HideSubmissions))
}

func (s Stores) ContestDelete(ctx context.Context, contestID uuid.UUID) error {
	return errs.DBErr(s.ExecUpdateBuilder(ctx, s.
		Builder().
		Update("contest").
		Set("deleted", true).
		Where(sq.Eq{"contest_id": contestID})))
}

func (s Stores) ContestEntryDelete(ctx context.Context, contestEntryID uuid.UUID) error {
	return errs.DBErr(s.ExecDeleteBuilder(ctx, s.
		Builder().
		Delete("contest_entry").
		Where(sq.Eq{"contest_entry_id": contestEntryID})))
}

func (s Stores) Contests(ctx context.Context, publicOnly bool) ([]domain.Contest, error) {
	var contests []domain.Contest

	builder := s.
		Builder().
		Select("s.contest_id", "s.title", "s.public", "s.description", "s.date_start",
			"s.date_end", "s.max_submissions", "s.media_types", "s.deleted", "s.voting", "s.min_permission_level",
			"s.down_votes", "s.created_on", "s.updated_on", "count(ce.contest_entry_id) as num_entries",
			"s.hide_submissions").
		From("contest s").
		LeftJoin("contest_entry ce USING (contest_id)").
		OrderBy("s.date_end DESC").
		GroupBy("s.contest_id")

	ands := sq.And{sq.Eq{"s.deleted": false}}
	if publicOnly {
		ands = append(ands, sq.Eq{"s.public": true})
	}

	rows, errRows := s.QueryBuilder(ctx, builder.Where(ands))
	if errRows != nil {
		if errors.Is(errRows, errs.ErrNoResult) {
			return []domain.Contest{}, nil
		}

		return nil, errs.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var contest domain.Contest
		if errScan := rows.Scan(&contest.ContestID, &contest.Title, &contest.Public, &contest.Description,
			&contest.DateStart, &contest.DateEnd, &contest.MaxSubmissions, &contest.MediaTypes,
			&contest.Deleted, &contest.Voting, &contest.MinPermissionLevel, &contest.DownVotes,
			&contest.CreatedOn, &contest.UpdatedOn, &contest.NumEntries, &contest.HideSubmissions); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		contests = append(contests, contest)
	}

	return contests, nil
}

func (s Stores) ContestEntrySave(ctx context.Context, entry domain.ContestEntry) error {
	return errs.DBErr(s.ExecInsertBuilder(ctx, s.
		Builder().
		Insert("contest_entry").
		Columns("contest_entry_id", "contest_id", "steam_id", "asset_id", "description",
			"placement", "deleted", "created_on", "updated_on").
		Values(entry.ContestEntryID, entry.ContestID, entry.SteamID, entry.AssetID, entry.Description,
			entry.Placement, entry.Deleted, entry.CreatedOn, entry.UpdatedOn)))
}

func (s Stores) ContestSave(ctx context.Context, contest *domain.Contest) error {
	if contest.ContestID == uuid.FromStringOrNil(EmptyUUID) {
		newID, errID := uuid.NewV4()
		if errID != nil {
			return errors.Join(errID, ErrUUIDGen)
		}

		contest.ContestID = newID

		return s.contestInsert(ctx, contest)
	}

	return s.contestUpdate(ctx, contest)
}

func (s Stores) contestInsert(ctx context.Context, contest *domain.Contest) error {
	query := s.
		Builder().
		Insert("contest").
		Columns("contest_id", "title", "public", "description", "date_start",
			"date_end", "max_submissions", "media_types", "deleted", "voting", "min_permission_level", "down_votes",
			"created_on", "updated_on", "hide_submissions").
		Values(contest.ContestID, contest.Title, contest.Public, contest.Description, contest.DateStart,
			contest.DateEnd, contest.MaxSubmissions, contest.MediaTypes, contest.Deleted,
			contest.Voting, contest.MinPermissionLevel, contest.DownVotes,
			contest.CreatedOn, contest.UpdatedOn, contest.HideSubmissions)

	if errExec := s.ExecInsertBuilder(ctx, query); errExec != nil {
		return errs.DBErr(errExec)
	}

	contest.IsNew = false

	return nil
}

func (s Stores) contestUpdate(ctx context.Context, contest *domain.Contest) error {
	contest.UpdatedOn = time.Now()

	return errs.DBErr(s.ExecUpdateBuilder(ctx, s.
		Builder().
		Update("contest").
		Set("title", contest.Title).
		Set("public", contest.Public).
		Set("description", contest.Description).
		Set("date_start", contest.DateStart).
		Set("date_end", contest.DateEnd).
		Set("hide_submissions", contest.HideSubmissions).
		Set("max_submissions", contest.MaxSubmissions).
		Set("voting", contest.Voting).
		Set("min_permission_level", contest.MinPermissionLevel).
		Set("down_votes", contest.DownVotes).
		Set("media_types", contest.MediaTypes).
		Set("deleted", contest.Deleted).
		Set("updated_on", contest.UpdatedOn).
		Where(sq.Eq{"contest_id": contest.ContestID})))
}

func (s Stores) ContestEntry(ctx context.Context, contestID uuid.UUID, entry *domain.ContestEntry) error {
	query := `
		SELECT
			s.contest_entry_id,
			s.contest_id,
			s.steam_id,
			s.asset_id,
			s.description,
			s.placement,
			s.deleted,
			s.created_on,
			s.updated_on,
			p.personaname,
			p.avatarhash,
			coalesce(v.votes_up, 0),
			coalesce(v.votes_down, 0),
			a.size,
			a.path,
			a.bucket,
			a.mime_type,
			a.name,
			a.asset_id
		FROM contest_entry s
		LEFT JOIN (
			SELECT 
			    contest_entry_id,
				SUM(CASE WHEN vote THEN 1 ELSE 0 END)     as votes_up,
				SUM(CASE WHEN NOT vote THEN 1 ELSE 0 END) as votes_down
			FROM contest_entry_vote
			GROUP BY contest_entry_id
			) v USING(contest_entry_id)
		LEFT JOIN person p USING(steam_id)
		LEFT JOIN public.asset a USING(asset_id)
		WHERE s.contest_entry_id = $1`

	if errScan := s.
		QueryRow(ctx, query, contestID).
		Scan(&entry.ContestEntryID, &entry.ContestID, &entry.SteamID, &entry.AssetID, &entry.Description,
			&entry.Placement, &entry.Deleted, &entry.CreatedOn, &entry.UpdatedOn,
			&entry.Personaname, &entry.AvatarHash, &entry.VotesUp, &entry.VotesDown,
			&entry.Asset.Size, &entry.Asset.Path, &entry.Asset.Bucket,
			&entry.Asset.MimeType, &entry.Asset.Name, &entry.Asset.AssetID); errScan != nil {
		return errs.DBErr(errScan)
	}

	return nil
}

func (s Stores) ContestEntries(ctx context.Context, contestID uuid.UUID) ([]*domain.ContestEntry, error) {
	query := `
		SELECT
			s.contest_entry_id,
			s.contest_id,
			s.steam_id,
			s.asset_id,
			s.description,
			s.placement,
			s.deleted,
			s.created_on,
			s.updated_on,
			p.personaname,
			p.avatarhash,
			coalesce(v.votes_up, 0),
			coalesce(v.votes_down, 0),
			a.size,
			a.path,
			a.bucket,
			a.mime_type,
			a.name,
			a.asset_id
		FROM contest_entry s
		LEFT JOIN (
			SELECT 
			    contest_entry_id,
				SUM(CASE WHEN vote THEN 1 ELSE 0 END)     as votes_up,
				SUM(CASE WHEN NOT vote THEN 1 ELSE 0 END) as votes_down
			FROM contest_entry_vote
			GROUP BY contest_entry_id
			) v USING(contest_entry_id)
		LEFT JOIN person p USING(steam_id)
		LEFT JOIN public.asset a USING(asset_id)
		WHERE s.contest_id = $1
		ORDER BY s.created_on DESC`

	var entries []*domain.ContestEntry

	rows, errRows := s.Query(ctx, query, contestID)
	if errRows != nil {
		if errors.Is(errRows, errs.ErrNoResult) {
			return []*domain.ContestEntry{}, nil
		}

		return nil, errs.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var entry domain.ContestEntry

		if errScan := rows.Scan(&entry.ContestEntryID, &entry.ContestID, &entry.SteamID, &entry.AssetID, &entry.Description,
			&entry.Placement, &entry.Deleted, &entry.CreatedOn, &entry.UpdatedOn,
			&entry.Personaname, &entry.AvatarHash, &entry.VotesUp, &entry.VotesDown,
			&entry.Asset.Size, &entry.Asset.Path, &entry.Asset.Bucket,
			&entry.Asset.MimeType, &entry.Asset.Name, &entry.Asset.AssetID); errScan != nil {
			return nil, errs.DBErr(errScan)
		}

		entries = append(entries, &entry)
	}

	return entries, nil
}

func (s Stores) ContestEntryVoteGet(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SID64, record *domain.ContentVoteRecord) error {
	query := s.
		Builder().
		Select("contest_entry_vote_id", "contest_entry_id", "steam_id",
			"vote", "created_on", "updated_on").
		From("contest_entry_vote").
		Where(sq.And{sq.Eq{"contest_entry_id": contestEntryID}, sq.Eq{"steam_id": steamID}})

	row, errQuery := s.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errs.DBErr(errQuery)
	}

	if errScan := row.
		Scan(&record.ContestEntryVoteID, &record.ContestEntryID,
			&record.SteamID, &record.Vote, &record.CreatedOn, &record.UpdatedOn); errScan != nil {
		return errs.DBErr(errScan)
	}

	return nil
}

func (s Stores) ContestEntryVote(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SID64, vote bool) error {
	var record domain.ContentVoteRecord
	if errRecord := s.ContestEntryVoteGet(ctx, contestEntryID, steamID, &record); errRecord != nil {
		if !errors.Is(errRecord, errs.ErrNoResult) {
			return errs.DBErr(errRecord)
		}

		record = domain.ContentVoteRecord{
			ContestEntryID: contestEntryID,
			SteamID:        steamID,
			Vote:           vote,
			TimeStamped:    domain.NewTimeStamped(),
		}

		now := time.Now()

		return errs.DBErr(s.ExecInsertBuilder(ctx, s.
			Builder().
			Insert("contest_entry_vote").
			Columns("contest_entry_id", "steam_id", "vote", "created_on", "updated_on").
			Values(contestEntryID, steamID, vote, now, now)))
	}

	if record.Vote == vote {
		// Delete the vote when user presses vote button again once already voted
		if errDelete := s.ContestEntryVoteDelete(ctx, record.ContestEntryVoteID); errDelete != nil {
			return errDelete
		}

		return errs.ErrVoteDeleted
	} else {
		if errSave := s.ContestEntryVoteUpdate(ctx, record.ContestEntryVoteID, vote); errSave != nil {
			return errSave
		}
	}

	return nil
}

func (s Stores) ContestEntryVoteDelete(ctx context.Context, contestEntryVoteID int64) error {
	return errs.DBErr(s.ExecDeleteBuilder(ctx, s.
		Builder().
		Delete("contest_entry_vote").
		Where(sq.Eq{"contest_entry_vote_id": contestEntryVoteID})))
}

func (s Stores) ContestEntryVoteUpdate(ctx context.Context, contestEntryVoteID int64, newVote bool) error {
	return errs.DBErr(s.ExecUpdateBuilder(ctx, s.
		Builder().
		Update("contest_entry_vote").
		Set("vote", newVote).
		Set("updated_on", time.Now()).
		Where(sq.Eq{"contest_entry_vote_id": contestEntryVoteID})))
}
