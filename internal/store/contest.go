package store

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

func ContestByID(ctx context.Context, database Store, contestID uuid.UUID, contest *model.Contest) error {
	if contestID.IsNil() {
		return errors.New("Invalid contest id")
	}

	query := database.
		Builder().
		Select("contest_id", "title", "public", "description", "date_start",
			"date_end", "max_submissions", "media_types", "deleted", "voting", "min_permission_level", "down_votes",
			"created_on", "updated_on", "hide_submissions").
		From("contest").
		Where(sq.And{sq.Eq{"deleted": false}, sq.Eq{"contest_id": contestID.String()}})

	row, errQuery := database.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return DBErr(errQuery)
	}

	return DBErr(row.Scan(&contest.ContestID, &contest.Title, &contest.Public, &contest.Description,
		&contest.DateStart, &contest.DateEnd, &contest.MaxSubmissions, &contest.MediaTypes,
		&contest.Deleted, &contest.Voting, &contest.MinPermissionLevel, &contest.DownVotes,
		&contest.CreatedOn, &contest.UpdatedOn, &contest.HideSubmissions))
}

func ContestDelete(ctx context.Context, database Store, contestID uuid.UUID) error {
	return DBErr(database.ExecUpdateBuilder(ctx, database.
		Builder().
		Update("contest").
		Set("deleted", true).
		Where(sq.Eq{"contest_id": contestID})))
}

func ContestEntryDelete(ctx context.Context, database Store, contestEntryID uuid.UUID) error {
	return DBErr(database.ExecDeleteBuilder(ctx, database.
		Builder().
		Delete("contest_entry").
		Where(sq.Eq{"contest_entry_id": contestEntryID})))
}

func Contests(ctx context.Context, database Store, publicOnly bool) ([]model.Contest, error) {
	var contests []model.Contest

	builder := database.
		Builder().
		Select("c.contest_id", "c.title", "c.public", "c.description", "c.date_start",
			"c.date_end", "c.max_submissions", "c.media_types", "c.deleted", "c.voting", "c.min_permission_level",
			"c.down_votes", "c.created_on", "c.updated_on", "count(ce.contest_entry_id) as num_entries",
			"c.hide_submissions").
		From("contest c").
		LeftJoin("contest_entry ce USING (contest_id)").
		OrderBy("c.date_end DESC").
		GroupBy("c.contest_id")

	ands := sq.And{sq.Eq{"c.deleted": false}}
	if publicOnly {
		ands = append(ands, sq.Eq{"c.public": true})
	}

	rows, errRows := database.QueryBuilder(ctx, builder.Where(ands))
	if errRows != nil {
		if errors.Is(errRows, ErrNoResult) {
			return []model.Contest{}, nil
		}

		return nil, DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var contest model.Contest
		if errScan := rows.Scan(&contest.ContestID, &contest.Title, &contest.Public, &contest.Description,
			&contest.DateStart, &contest.DateEnd, &contest.MaxSubmissions, &contest.MediaTypes,
			&contest.Deleted, &contest.Voting, &contest.MinPermissionLevel, &contest.DownVotes,
			&contest.CreatedOn, &contest.UpdatedOn, &contest.NumEntries, &contest.HideSubmissions); errScan != nil {
			return nil, DBErr(errScan)
		}

		contests = append(contests, contest)
	}

	return contests, nil
}

func ContestEntrySave(ctx context.Context, database Store, entry model.ContestEntry) error {
	return DBErr(database.ExecInsertBuilder(ctx, database.
		Builder().
		Insert("contest_entry").
		Columns("contest_entry_id", "contest_id", "steam_id", "asset_id", "description",
			"placement", "deleted", "created_on", "updated_on").
		Values(entry.ContestEntryID, entry.ContestID, entry.SteamID, entry.AssetID, entry.Description,
			entry.Placement, entry.Deleted, entry.CreatedOn, entry.UpdatedOn)))
}

func ContestSave(ctx context.Context, database Store, contest *model.Contest) error {
	if contest.ContestID == uuid.FromStringOrNil(EmptyUUID) {
		newID, errID := uuid.NewV4()
		if errID != nil {
			return errors.Wrap(errID, "Failed to generate new uuidv4")
		}

		contest.ContestID = newID

		return contestInsert(ctx, database, contest)
	}

	return contestUpdate(ctx, database, contest)
}

func contestInsert(ctx context.Context, database Store, contest *model.Contest) error {
	query := database.
		Builder().
		Insert("contest").
		Columns("contest_id", "title", "public", "description", "date_start",
			"date_end", "max_submissions", "media_types", "deleted", "voting", "min_permission_level", "down_votes",
			"created_on", "updated_on", "hide_submissions").
		Values(contest.ContestID, contest.Title, contest.Public, contest.Description, contest.DateStart,
			contest.DateEnd, contest.MaxSubmissions, contest.MediaTypes, contest.Deleted,
			contest.Voting, contest.MinPermissionLevel, contest.DownVotes,
			contest.CreatedOn, contest.UpdatedOn, contest.HideSubmissions)

	if errExec := database.ExecInsertBuilder(ctx, query); errExec != nil {
		return DBErr(errExec)
	}

	contest.IsNew = false

	return nil
}

func contestUpdate(ctx context.Context, database Store, contest *model.Contest) error {
	contest.UpdatedOn = time.Now()

	return DBErr(database.ExecUpdateBuilder(ctx, database.
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

func ContestEntry(ctx context.Context, database Store, contestID uuid.UUID, entry *model.ContestEntry) error {
	query := `
		SELECT
			c.contest_entry_id,
			c.contest_id,
			c.steam_id,
			c.asset_id,
			c.description,
			c.placement,
			c.deleted,
			c.created_on,
			c.updated_on,
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
		FROM contest_entry c
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
		WHERE c.contest_entry_id = $1`

	if errScan := database.QueryRow(ctx, query, contestID).Scan(&entry.ContestEntryID, &entry.ContestID, &entry.SteamID, &entry.AssetID, &entry.Description,
		&entry.Placement, &entry.Deleted, &entry.CreatedOn, &entry.UpdatedOn,
		&entry.Personaname, &entry.AvatarHash, &entry.VotesUp, &entry.VotesDown,
		&entry.Asset.Size, &entry.Asset.Path, &entry.Asset.Bucket,
		&entry.Asset.MimeType, &entry.Asset.Name, &entry.Asset.AssetID); errScan != nil {
		return DBErr(errScan)
	}

	return nil
}

func ContestEntries(ctx context.Context, database Store, contestID uuid.UUID) ([]*model.ContestEntry, error) {
	query := `
		SELECT
			c.contest_entry_id,
			c.contest_id,
			c.steam_id,
			c.asset_id,
			c.description,
			c.placement,
			c.deleted,
			c.created_on,
			c.updated_on,
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
		FROM contest_entry c
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
		WHERE c.contest_id = $1
		ORDER BY c.created_on DESC`

	var entries []*model.ContestEntry

	rows, errRows := database.Query(ctx, query, contestID)
	if errRows != nil {
		if errors.Is(errRows, ErrNoResult) {
			return []*model.ContestEntry{}, nil
		}

		return nil, DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var entry model.ContestEntry

		if errScan := rows.Scan(&entry.ContestEntryID, &entry.ContestID, &entry.SteamID, &entry.AssetID, &entry.Description,
			&entry.Placement, &entry.Deleted, &entry.CreatedOn, &entry.UpdatedOn,
			&entry.Personaname, &entry.AvatarHash, &entry.VotesUp, &entry.VotesDown,
			&entry.Asset.Size, &entry.Asset.Path, &entry.Asset.Bucket,
			&entry.Asset.MimeType, &entry.Asset.Name, &entry.Asset.AssetID); errScan != nil {
			return nil, DBErr(errScan)
		}

		entries = append(entries, &entry)
	}

	return entries, nil
}

func ContestEntryVoteGet(ctx context.Context, database Store, contestEntryID uuid.UUID, steamID steamid.SID64, record *model.ContentVoteRecord) error {
	query := database.
		Builder().
		Select("contest_entry_vote_id", "contest_entry_id", "steam_id",
			"vote", "created_on", "updated_on").
		From("contest_entry_vote").
		Where(sq.And{sq.Eq{"contest_entry_id": contestEntryID}, sq.Eq{"steam_id": steamID}})

	row, errQuery := database.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return DBErr(errQuery)
	}

	if errScan := row.
		Scan(&record.ContestEntryVoteID, &record.ContestEntryID,
			&record.SteamID, &record.Vote, &record.CreatedOn, &record.UpdatedOn); errScan != nil {
		return DBErr(errScan)
	}

	return nil
}

var ErrVoteDeleted = errors.New("Vote deleted")

func ContestEntryVote(ctx context.Context, database Store, contestEntryID uuid.UUID, steamID steamid.SID64, vote bool) error {
	var record model.ContentVoteRecord
	if errRecord := ContestEntryVoteGet(ctx, database, contestEntryID, steamID, &record); errRecord != nil {
		if !errors.Is(errRecord, ErrNoResult) {
			return DBErr(errRecord)
		}

		record = model.ContentVoteRecord{
			ContestEntryID: contestEntryID,
			SteamID:        steamID,
			Vote:           vote,
			TimeStamped:    model.NewTimeStamped(),
		}

		now := time.Now()

		return DBErr(database.ExecInsertBuilder(ctx, database.
			Builder().
			Insert("contest_entry_vote").
			Columns("contest_entry_id", "steam_id", "vote", "created_on", "updated_on").
			Values(contestEntryID, steamID, vote, now, now)))
	}

	if record.Vote == vote {
		// Delete the vote when user presses vote button again once already voted
		if errDelete := ContestEntryVoteDelete(ctx, database, record.ContestEntryVoteID); errDelete != nil {
			return errDelete
		}

		return ErrVoteDeleted
	} else {
		if errSave := ContestEntryVoteUpdate(ctx, database, record.ContestEntryVoteID, vote); errSave != nil {
			return errSave
		}
	}

	return nil
}

func ContestEntryVoteDelete(ctx context.Context, database Store, contestEntryVoteID int64) error {
	return DBErr(database.ExecDeleteBuilder(ctx, database.
		Builder().
		Delete("contest_entry_vote").
		Where(sq.Eq{"contest_entry_vote_id": contestEntryVoteID})))
}

func ContestEntryVoteUpdate(ctx context.Context, database Store, contestEntryVoteID int64, newVote bool) error {
	return DBErr(database.ExecUpdateBuilder(ctx, database.
		Builder().
		Update("contest_entry_vote").
		Set("vote", newVote).
		Set("updated_on", time.Now()).
		Where(sq.Eq{"contest_entry_vote_id": contestEntryVoteID})))
}
