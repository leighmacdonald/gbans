package contest

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type contestRepository struct {
	db database.Database
}

func NewContestRepository(database database.Database) domain.ContestRepository {
	return &contestRepository{db: database}
}

func (c *contestRepository) ContestByID(ctx context.Context, contestID uuid.UUID, contest *domain.Contest) error {
	if contestID.IsNil() {
		return domain.ErrInvalidContestID
	}

	query := c.db.
		Builder().
		Select("contest_id", "title", "public", "description", "date_start",
			"date_end", "max_submissions", "media_types", "deleted", "voting", "min_permission_level", "down_votes",
			"created_on", "updated_on", "hide_submissions").
		From("contest").
		Where(sq.And{sq.Eq{"deleted": false}, sq.Eq{"contest_id": contestID.String()}})

	row, errQuery := c.db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return c.db.DBErr(errQuery)
	}

	return c.db.DBErr(row.Scan(&contest.ContestID, &contest.Title, &contest.Public, &contest.Description,
		&contest.DateStart, &contest.DateEnd, &contest.MaxSubmissions, &contest.MediaTypes,
		&contest.Deleted, &contest.Voting, &contest.MinPermissionLevel, &contest.DownVotes,
		&contest.CreatedOn, &contest.UpdatedOn, &contest.HideSubmissions))
}

func (c *contestRepository) ContestDelete(ctx context.Context, contestID uuid.UUID) error {
	return c.db.DBErr(c.db.ExecUpdateBuilder(ctx, c.db.
		Builder().
		Update("contest").
		Set("deleted", true).
		Where(sq.Eq{"contest_id": contestID})))
}

func (c *contestRepository) ContestEntryDelete(ctx context.Context, contestEntryID uuid.UUID) error {
	return c.db.DBErr(c.db.ExecDeleteBuilder(ctx, c.db.
		Builder().
		Delete("contest_entry").
		Where(sq.Eq{"contest_entry_id": contestEntryID})))
}

func (c *contestRepository) Contests(ctx context.Context, publicOnly bool) ([]domain.Contest, error) {
	var contests []domain.Contest

	builder := c.db.
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

	rows, errRows := c.db.QueryBuilder(ctx, builder.Where(ands))
	if errRows != nil {
		if errors.Is(errRows, domain.ErrNoResult) {
			return []domain.Contest{}, nil
		}

		return nil, c.db.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var contest domain.Contest
		if errScan := rows.Scan(&contest.ContestID, &contest.Title, &contest.Public, &contest.Description,
			&contest.DateStart, &contest.DateEnd, &contest.MaxSubmissions, &contest.MediaTypes,
			&contest.Deleted, &contest.Voting, &contest.MinPermissionLevel, &contest.DownVotes,
			&contest.CreatedOn, &contest.UpdatedOn, &contest.NumEntries, &contest.HideSubmissions); errScan != nil {
			return nil, c.db.DBErr(errScan)
		}

		contests = append(contests, contest)
	}

	return contests, nil
}

func (c *contestRepository) ContestEntrySave(ctx context.Context, entry domain.ContestEntry) error {
	return c.db.DBErr(c.db.ExecInsertBuilder(ctx, c.db.
		Builder().
		Insert("contest_entry").
		Columns("contest_entry_id", "contest_id", "steam_id", "asset_id", "description",
			"placement", "deleted", "created_on", "updated_on").
		Values(entry.ContestEntryID, entry.ContestID, entry.SteamID, entry.AssetID, entry.Description,
			entry.Placement, entry.Deleted, entry.CreatedOn, entry.UpdatedOn)))
}

func (c *contestRepository) ContestSave(ctx context.Context, contest *domain.Contest) error {
	if contest.ContestID == uuid.FromStringOrNil(domain.EmptyUUID) {
		newID, errID := uuid.NewV4()
		if errID != nil {
			return errors.Join(errID, domain.ErrUUIDGen)
		}

		contest.ContestID = newID

		return c.contestInsert(ctx, contest)
	}

	return c.contestUpdate(ctx, contest)
}

func (c *contestRepository) contestInsert(ctx context.Context, contest *domain.Contest) error {
	query := c.db.
		Builder().
		Insert("contest").
		Columns("contest_id", "title", "public", "description", "date_start",
			"date_end", "max_submissions", "media_types", "deleted", "voting", "min_permission_level", "down_votes",
			"created_on", "updated_on", "hide_submissions").
		Values(contest.ContestID, contest.Title, contest.Public, contest.Description, contest.DateStart,
			contest.DateEnd, contest.MaxSubmissions, contest.MediaTypes, contest.Deleted,
			contest.Voting, contest.MinPermissionLevel, contest.DownVotes,
			contest.CreatedOn, contest.UpdatedOn, contest.HideSubmissions)

	if errExec := c.db.ExecInsertBuilder(ctx, query); errExec != nil {
		return c.db.DBErr(errExec)
	}

	contest.IsNew = false

	return nil
}

func (c *contestRepository) contestUpdate(ctx context.Context, contest *domain.Contest) error {
	contest.UpdatedOn = time.Now()

	return c.db.DBErr(c.db.ExecUpdateBuilder(ctx, c.db.
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

func (c *contestRepository) ContestEntry(ctx context.Context, contestID uuid.UUID, entry *domain.ContestEntry) error {
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

	if errScan := c.db.
		QueryRow(ctx, query, contestID).
		Scan(&entry.ContestEntryID, &entry.ContestID, &entry.SteamID, &entry.AssetID, &entry.Description,
			&entry.Placement, &entry.Deleted, &entry.CreatedOn, &entry.UpdatedOn,
			&entry.Personaname, &entry.AvatarHash, &entry.VotesUp, &entry.VotesDown,
			&entry.Asset.Size, &entry.Asset.Bucket,
			&entry.Asset.MimeType, &entry.Asset.Name, &entry.Asset.AssetID); errScan != nil {
		return c.db.DBErr(errScan)
	}

	return nil
}

func (c *contestRepository) ContestEntries(ctx context.Context, contestID uuid.UUID) ([]*domain.ContestEntry, error) {
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

	var entries []*domain.ContestEntry

	rows, errRows := c.db.Query(ctx, query, contestID)
	if errRows != nil {
		if errors.Is(errRows, domain.ErrNoResult) {
			return []*domain.ContestEntry{}, nil
		}

		return nil, c.db.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var entry domain.ContestEntry

		if errScan := rows.Scan(&entry.ContestEntryID, &entry.ContestID, &entry.SteamID, &entry.AssetID, &entry.Description,
			&entry.Placement, &entry.Deleted, &entry.CreatedOn, &entry.UpdatedOn,
			&entry.Personaname, &entry.AvatarHash, &entry.VotesUp, &entry.VotesDown,
			&entry.Asset.Size, &entry.Asset.Bucket,
			&entry.Asset.MimeType, &entry.Asset.Name, &entry.Asset.AssetID); errScan != nil {
			return nil, c.db.DBErr(errScan)
		}

		entries = append(entries, &entry)
	}

	return entries, nil
}

func (c *contestRepository) ContestEntryVoteGet(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SteamID, record *domain.ContentVoteRecord) error {
	query := c.db.
		Builder().
		Select("contest_entry_vote_id", "contest_entry_id", "steam_id",
			"vote", "created_on", "updated_on").
		From("contest_entry_vote").
		Where(sq.And{sq.Eq{"contest_entry_id": contestEntryID}, sq.Eq{"steam_id": steamID}})

	row, errQuery := c.db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return c.db.DBErr(errQuery)
	}

	if errScan := row.
		Scan(&record.ContestEntryVoteID, &record.ContestEntryID,
			&record.SteamID, &record.Vote, &record.CreatedOn, &record.UpdatedOn); errScan != nil {
		return c.db.DBErr(errScan)
	}

	return nil
}

func (c *contestRepository) ContestEntryVote(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SteamID, vote bool) error {
	var record domain.ContentVoteRecord
	if errRecord := c.ContestEntryVoteGet(ctx, contestEntryID, steamID, &record); errRecord != nil {
		if !errors.Is(errRecord, domain.ErrNoResult) {
			return c.db.DBErr(errRecord)
		}

		record = domain.ContentVoteRecord{
			ContestEntryID: contestEntryID,
			SteamID:        steamID,
			Vote:           vote,
			TimeStamped:    domain.NewTimeStamped(),
		}

		now := time.Now()

		return c.db.DBErr(c.db.ExecInsertBuilder(ctx, c.db.
			Builder().
			Insert("contest_entry_vote").
			Columns("contest_entry_id", "steam_id", "vote", "created_on", "updated_on").
			Values(contestEntryID, steamID, vote, now, now)))
	}

	if record.Vote == vote {
		// Delete the vote when user presses vote button again once already voted
		if errDelete := c.ContestEntryVoteDelete(ctx, record.ContestEntryVoteID); errDelete != nil {
			return errDelete
		}

		return domain.ErrVoteDeleted
	} else {
		if errSave := c.ContestEntryVoteUpdate(ctx, record.ContestEntryVoteID, vote); errSave != nil {
			return errSave
		}
	}

	return nil
}

func (c *contestRepository) ContestEntryVoteDelete(ctx context.Context, contestEntryVoteID int64) error {
	return c.db.DBErr(c.db.ExecDeleteBuilder(ctx, c.db.
		Builder().
		Delete("contest_entry_vote").
		Where(sq.Eq{"contest_entry_vote_id": contestEntryVoteID})))
}

func (c *contestRepository) ContestEntryVoteUpdate(ctx context.Context, contestEntryVoteID int64, newVote bool) error {
	return c.db.DBErr(c.db.ExecUpdateBuilder(ctx, c.db.
		Builder().
		Update("contest_entry_vote").
		Set("vote", newVote).
		Set("updated_on", time.Now()).
		Where(sq.Eq{"contest_entry_vote_id": contestEntryVoteID})))
}
