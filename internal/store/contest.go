package store

import (
	"context"
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
	MediaTypes     string    `json:"media_types"`
	Deleted        bool      `json:"-"`
	// Allow voting
	Voting bool `json:"voting"`
	// Minimum permission level allowed to vote
	MinPermissionLevel consts.Privilege `json:"min_permission_level"`
	// Allow down voting
	DownVotes bool `json:"down_votes"`
	isNew     bool
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
	isNew          bool
}

type ContestEntryVote struct {
	ContestEntryID uuid.UUID     `json:"contest_entry_id"`
	SteamID        steamid.SID64 `json:"steam_id"`
	Vote           int           `json:"vote"`
	TimeStamped
}

func (c Contest) NewEntry(sid64 steamid.SID64, description string, asset Asset) (*ContestEntry, error) {
	if c.ContestID.IsNil() {
		return nil, errors.New("Invalid contest id")
	}

	newID, errID := uuid.NewV4()
	if errID != nil {
		return nil, errors.Wrap(errID, "Failed to generate uuid")
	}

	if description == "" {
		return nil, errors.New("description cannot be empty")
	}

	entry := ContestEntry{
		TimeStamped:    NewTimeStamped(),
		ContestEntryID: newID,
		ContestID:      c.ContestID,
		SteamID:        sid64,
		AssetID:        asset.AssetID,
		Description:    description,
		Placement:      0,
		Deleted:        false,
		isNew:          true,
	}

	return &entry, nil
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

	query, args, errQuery := db.sb.
		Select("contest_id", "title", "public", "description", "date_start",
			"date_end", "max_submissions", "media_types", "deleted", "voting", "min_permission_level", "down_votes",
			"created_on", "updated_on").
		From("contest").
		Where(sq.Eq{"contest_id": contestID}).
		ToSql()

	if errQuery != nil {
		return Err(errQuery)
	}

	if errScan := db.QueryRow(ctx, query, args...).
		Scan(&contest.ContestID, &contest.Title, &contest.Public, &contest.Description,
			&contest.DateStart, &contest.DateEnd, &contest.MaxSubmissions, &contest.MediaTypes,
			&contest.Deleted, &contest.Voting, &contest.MinPermissionLevel, &contest.DownVotes,
			&contest.ContestID, &contest.UpdatedOn); errScan != nil {
		return Err(errScan)
	}

	return nil
}

func (db *Store) Contests(ctx context.Context, publicOnly bool) ([]Contest, error) {
	contests := []Contest{}

	builder := db.sb.
		Select("contest_id", "title", "public", "description", "date_start",
			"date_end", "max_submissions", "media_types", "deleted", "voting", "min_permission_level", "down_votes",
			"created_on", "updated_on").
		From("contest")

	if publicOnly {
		builder = builder.Where(sq.Eq{"public": true})
	}

	query, args, errQuery := builder.ToSql()
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
			&contest.CreatedOn, &contest.UpdatedOn); errScan != nil {
			return nil, Err(errScan)
		}

		contests = append(contests, contest)
	}

	return contests, nil
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
			"c.placement", "c.deleted", "c.created_on", "c.updated_on", "p.persona_name", "p.avatar_hash").
		From("contest_entry c").
		LeftJoin("person p USING(steam_id)").
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

		if errScan := rows.Scan(&entry.ContestEntryID, &entry.ContestID, &entry.SteamID, &entry.Description,
			&entry.Placement, &entry.Deleted, &entry.CreatedOn, &entry.UpdatedOn,
			&entry.Personaname, &entry.AvatarHash); errScan != nil {
			return nil, Err(errScan)
		}

		entries = append(entries, &entry)
	}

	return entries, nil
}
