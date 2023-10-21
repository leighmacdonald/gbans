package store

import (
	"context"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

type Contest struct {
	TimeStamped
	ContestID      uuid.UUID `json:"contest_id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Public         bool      `json:"public"`
	DateStart      time.Time `json:"date_start"`
	DateEnd        time.Time `json:"date_end"`
	MaxSubmissions int       `json:"max_submissions"`
	OwnSubmissions int       `json:"own_submissions"`
	MediaTypes     string    `json:"media_types"`
	NumEntries     int       `json:"num_entries"`
	Deleted        bool      `json:"-"`
	// Allow voting
	Voting bool `json:"voting"`
	// Minimum permission level allowed to vote
	MinPermissionLevel consts.Privilege `json:"min_permission_level"`
	// Allow down voting
	DownVotes bool `json:"down_votes"`
	isNew     bool
}

func (c Contest) MimeTypeAcceptable(mediaType string) bool {
	if c.MediaTypes == "" {
		return true
	}

	for _, validType := range strings.Split(c.MediaTypes, ",") {
		if strings.EqualFold(validType, mediaType) {
			return true
		}
	}

	return false
}

type ContestEntry struct {
	TimeStamped
	ContestEntryID uuid.UUID     `json:"contest_entry_id"`
	ContestID      uuid.UUID     `json:"contest_id"`
	SteamID        steamid.SID64 `json:"steam_id"`
	Personaname    string        `json:"personaname"`
	AvatarHash     string        `json:"avatar_hash"`
	AssetID        uuid.UUID     `json:"asset_id"`
	Description    string        `json:"description"`
	Placement      int           `json:"placement"`
	Deleted        bool          `json:"deleted"`
}

type ContestEntryVote struct {
	ContestEntryID uuid.UUID     `json:"contest_entry_id"`
	SteamID        steamid.SID64 `json:"steam_id"`
	Vote           int           `json:"vote"`
	TimeStamped
}

func (c Contest) NewEntry(steamID steamid.SID64, assetID uuid.UUID, description string) (ContestEntry, error) {
	if c.ContestID.IsNil() {
		return ContestEntry{}, errors.New("Invalid contest id")
	}

	if !steamID.Valid() {
		return ContestEntry{}, consts.ErrInvalidSID
	}

	if description == "" {
		return ContestEntry{}, errors.New("Description cannot be empty")
	}

	newID, errID := uuid.NewV4()
	if errID != nil {
		return ContestEntry{}, errors.Wrap(errID, "Failed to generate new uuidv4")
	}

	return ContestEntry{
		TimeStamped:    NewTimeStamped(),
		ContestEntryID: newID,
		ContestID:      c.ContestID,
		SteamID:        steamID,
		Personaname:    "",
		AvatarHash:     "",
		AssetID:        assetID,
		Description:    description,
		Placement:      0,
		Deleted:        false,
	}, nil
}

func NewContest(title string, description string, dateStart time.Time, dateEnd time.Time, public bool) (Contest, error) {
	newID, errID := uuid.NewV4()
	if errID != nil {
		return Contest{}, errors.Wrap(errID, "Failed to generate uuid")
	}

	if title == "" {
		return Contest{}, errors.New("Title cannot be empty")
	}

	if description == "" {
		return Contest{}, errors.New("Title cannot be empty")
	}

	if dateEnd.Before(dateStart) {
		return Contest{}, errors.New("End date cannot come before start date")
	}

	contest := Contest{
		TimeStamped:        NewTimeStamped(),
		ContestID:          newID,
		Title:              title,
		Description:        description,
		Public:             public,
		DateStart:          dateStart,
		DateEnd:            dateEnd,
		MaxSubmissions:     0,
		MediaTypes:         "",
		Deleted:            false,
		Voting:             false,
		MinPermissionLevel: consts.PUser,
		DownVotes:          false,
		isNew:              true,
	}

	return contest, nil
}

func (db *Store) ContestByID(ctx context.Context, contestID uuid.UUID, contest *Contest) error {
	if contestID.IsNil() {
		return errors.New("Invalid contest id")
	}

	query, args := db.sb.
		Select("contest_id", "title", "public", "description", "date_start",
			"date_end", "max_submissions", "media_types", "deleted", "voting", "min_permission_level", "down_votes",
			"created_on", "updated_on").
		From("contest").
		Where(sq.And{sq.Eq{"deleted": false}, sq.Eq{"contest_id": contestID.String()}}).
		MustSql()

	if errScan := db.QueryRow(ctx, query, args...).
		Scan(&contest.ContestID, &contest.Title, &contest.Public, &contest.Description,
			&contest.DateStart, &contest.DateEnd, &contest.MaxSubmissions, &contest.MediaTypes,
			&contest.Deleted, &contest.Voting, &contest.MinPermissionLevel, &contest.DownVotes,
			&contest.CreatedOn, &contest.UpdatedOn); errScan != nil {
		return Err(errScan)
	}

	return nil
}

func (db *Store) ContestDelete(ctx context.Context, contestID uuid.UUID) error {
	const query = `
		UPDATE contest SET deleted = true 
    	WHERE contest_id = $1`

	if errExec := db.Exec(ctx, query, contestID); errExec != nil {
		return Err(errExec)
	}

	return nil
}

func (db *Store) Contests(ctx context.Context, publicOnly bool) ([]Contest, error) {
	contests := []Contest{}

	builder := db.sb.
		Select("c.contest_id", "c.title", "c.public", "c.description", "c.date_start",
			"c.date_end", "c.max_submissions", "c.media_types", "c.deleted", "c.voting", "c.min_permission_level",
			"c.down_votes", "c.created_on", "c.updated_on", "count(ce.contest_entry_id) as num_entries").
		From("contest c").
		LeftJoin("contest_entry ce USING (contest_id)").
		GroupBy("c.contest_id")

	ands := sq.And{sq.Eq{"c.deleted": false}}
	if publicOnly {
		ands = append(ands, sq.Eq{"c.public": true})
	}

	query, args, errQuery := builder.Where(ands).ToSql()
	if errQuery != nil {
		return nil, Err(errQuery)
	}

	rows, errRows := db.Query(ctx, query, args...)
	if errRows != nil {
		if errors.Is(errRows, ErrNoResult) {
			return contests, nil
		}

		return nil, Err(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var contest Contest
		if errScan := rows.Scan(&contest.ContestID, &contest.Title, &contest.Public, &contest.Description,
			&contest.DateStart, &contest.DateEnd, &contest.MaxSubmissions, &contest.MediaTypes,
			&contest.Deleted, &contest.Voting, &contest.MinPermissionLevel, &contest.DownVotes,
			&contest.CreatedOn, &contest.UpdatedOn, &contest.NumEntries); errScan != nil {
			return nil, Err(errScan)
		}

		contests = append(contests, contest)
	}

	return contests, nil
}

func (db *Store) ContestEntrySave(ctx context.Context, entry ContestEntry) error {
	query, args, errQuery := db.sb.
		Insert("contest_entry").
		Columns("contest_entry_id", "contest_id", "steam_id", "asset_id", "description",
			"placement", "deleted", "created_on", "updated_on").
		Values(entry.ContestEntryID, entry.ContestID, entry.SteamID, entry.AssetID, entry.Description,
			entry.Placement, entry.Deleted, entry.CreatedOn, entry.UpdatedOn).
		ToSql()

	if errQuery != nil {
		return Err(errQuery)
	}

	if errExec := db.Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}

	return nil
}

func (db *Store) ContestSave(ctx context.Context, contest *Contest) error {
	if contest.ContestID == uuid.FromStringOrNil(EmptyUUID) {
		newID, errID := uuid.NewV4()
		if errID != nil {
			return errors.Wrap(errID, "Failed to generate new uuidv4")
		}

		contest.ContestID = newID

		return db.contestInsert(ctx, contest)
	}

	return db.contestUpdate(ctx, contest)
}

func (db *Store) contestInsert(ctx context.Context, contest *Contest) error {
	query, args, errQuery := db.sb.
		Insert("contest").
		Columns("contest_id", "title", "public", "description", "date_start",
			"date_end", "max_submissions", "media_types", "deleted", "voting", "min_permission_level", "down_votes",
			"created_on", "updated_on").
		Values(contest.ContestID, contest.Title, contest.Public, contest.Description, contest.DateStart,
			contest.DateEnd, contest.MaxSubmissions, contest.MediaTypes, contest.Deleted,
			contest.Voting, contest.MinPermissionLevel, contest.DownVotes,
			contest.CreatedOn, contest.UpdatedOn).
		ToSql()

	if errQuery != nil {
		return Err(errQuery)
	}

	if errExec := db.Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}

	contest.isNew = false

	return nil
}

func (db *Store) contestUpdate(ctx context.Context, contest *Contest) error {
	contest.UpdatedOn = time.Now()

	query, args, errQuery := db.sb.
		Update("contest").
		Set("title", contest.Title).
		Set("public", contest.Public).
		Set("description", contest.Description).
		Set("date_start", contest.DateStart).
		Set("date_end", contest.DateEnd).
		Set("max_submissions", contest.MaxSubmissions).
		Set("voting", contest.Voting).
		Set("min_permission_level", contest.MinPermissionLevel).
		Set("down_votes", contest.DownVotes).
		Set("media_types", contest.MediaTypes).
		Set("deleted", contest.Deleted).
		Set("updated_on", contest.UpdatedOn).
		Where(sq.Eq{"contest_id": contest.ContestID}).
		ToSql()

	if errQuery != nil {
		return Err(errQuery)
	}

	if errExec := db.Exec(ctx, query, args...); errExec != nil {
		return Err(errExec)
	}

	return nil
}

func (db *Store) ContestEntries(ctx context.Context, contestID uuid.UUID) ([]*ContestEntry, error) {
	query, args, errQuery := db.sb.
		Select("c.contest_entry_id", "c.contest_id", "c.steam_id", "c.asset_id", "c.description",
			"c.placement", "c.deleted", "c.created_on", "c.updated_on", "p.personaname", "p.avatarhash").
		From("contest_entry c").
		LeftJoin("person p USING(steam_id)").
		Where(sq.Eq{"contest_id": contestID}).
		ToSql()

	if errQuery != nil {
		return nil, Err(errQuery)
	}

	entries := []*ContestEntry{}

	rows, errRows := db.Query(ctx, query, args...)
	if errRows != nil {
		if errors.Is(errRows, ErrNoResult) {
			return entries, nil
		}

		return nil, Err(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var entry ContestEntry

		if errScan := rows.Scan(&entry.ContestEntryID, &entry.ContestID, &entry.SteamID, &entry.AssetID, &entry.Description,
			&entry.Placement, &entry.Deleted, &entry.CreatedOn, &entry.UpdatedOn,
			&entry.Personaname, &entry.AvatarHash); errScan != nil {
			return nil, Err(errScan)
		}

		entries = append(entries, &entry)
	}

	return entries, nil
}

type ContentVoteRecord struct {
	ContestEntryVoteID int64         `json:"contest_entry_vote_id"`
	ContestEntryID     uuid.UUID     `json:"contest_entry_id"`
	SteamID            steamid.SID64 `json:"steam_id"`
	Vote               bool          `json:"vote"`
	TimeStamped
}

func (db *Store) ContestEntryVoteGet(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SID64, record *ContentVoteRecord) error {
	query, args, errQuery := db.sb.
		Select("contest_entry_vote_id", "contest_entry_id", "steam_id",
			"vote", "created_on", "updated_on").
		Where(sq.And{sq.Eq{"contest_entry_id": contestEntryID}, sq.Eq{"steam_id": steamID}}).
		ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}

	if errExec := db.
		QueryRow(ctx, query, args...).
		Scan(&record.ContestEntryVoteID, &record.ContestEntryID,
			&record.SteamID, &record.Vote, &record.CreatedOn, &record.UpdatedOn); errExec != nil {
		return Err(errExec)
	}

	return nil
}

func (db *Store) ContestEntryVote(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SID64, upvote bool) error {
	var record ContentVoteRecord
	if errRecord := db.ContestEntryVoteGet(ctx, contestEntryID, steamID, &record); errRecord != nil {
		if !errors.Is(errRecord, ErrNoResult) {
			return Err(errRecord)
		}

		record = ContentVoteRecord{
			ContestEntryID: contestEntryID,
			SteamID:        steamID,
			Vote:           upvote,
			TimeStamped:    NewTimeStamped(),
		}

		now := time.Now()

		query, args, errQuery := db.sb.
			Insert("contest_entry_vote").
			Columns("contest_entry_id", "steam_id", "vote", "created_on", "updated_on").
			Values(contestEntryID, steamID, upvote, now, now).ToSql()
		if errQuery != nil {
			return Err(errQuery)
		}

		if errExec := db.Exec(ctx, query, args...); errExec != nil {
			return Err(errExec)
		}

		return nil
	}

	if record.Vote == upvote {
		// No changes
		return nil
	}

	return nil
}
