package domain

import (
	"context"
	"errors"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

// EmptyUUID is used as a placeholder value for signaling the entity is new.
const EmptyUUID = "feb4bf16-7f55-4cb4-923c-4de69a093b79"

type ContestRepository interface {
	ContestSave(ctx context.Context, contest *Contest) error
	ContestByID(ctx context.Context, contestID uuid.UUID, contest *Contest) error
	ContestDelete(ctx context.Context, contestID uuid.UUID) error
	ContestEntryDelete(ctx context.Context, contestEntryID uuid.UUID) error
	Contests(ctx context.Context, publicOnly bool) ([]Contest, error)
	ContestEntry(ctx context.Context, contestID uuid.UUID, entry *ContestEntry) error
	ContestEntrySave(ctx context.Context, entry ContestEntry) error
	ContestEntries(ctx context.Context, contestID uuid.UUID) ([]*ContestEntry, error)
	ContestEntryVoteGet(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SID64, record *ContentVoteRecord) error
	ContestEntryVote(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SID64, vote bool) error
	ContestEntryVoteDelete(ctx context.Context, contestEntryVoteID int64) error
	ContestEntryVoteUpdate(ctx context.Context, contestEntryVoteID int64, newVote bool) error
}

type ContestUsecase interface {
	ContestSave(ctx context.Context, contest Contest) (Contest, error)
	ContestByID(ctx context.Context, contestID uuid.UUID, contest *Contest) error
	ContestDelete(ctx context.Context, contestID uuid.UUID) error
	ContestEntryDelete(ctx context.Context, contestEntryID uuid.UUID) error
	Contests(ctx context.Context, user PersonInfo) ([]Contest, error)
	ContestEntry(ctx context.Context, contestID uuid.UUID, entry *ContestEntry) error
	ContestEntrySave(ctx context.Context, entry ContestEntry) error
	ContestEntries(ctx context.Context, contestID uuid.UUID) ([]*ContestEntry, error)
	ContestEntryVoteGet(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SID64, record *ContentVoteRecord) error
	ContestEntryVote(ctx context.Context, contestID uuid.UUID, contestEntryID uuid.UUID, user PersonInfo, vote bool) error
	ContestEntryVoteDelete(ctx context.Context, contestEntryVoteID int64) error
	ContestEntryVoteUpdate(ctx context.Context, contestEntryVoteID int64, newVote bool) error
}

type Contest struct {
	TimeStamped
	ContestID       uuid.UUID `json:"contest_id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Public          bool      `json:"public"`
	HideSubmissions bool      `json:"hide_submissions"` // Are user submissions visible for the public
	DateStart       time.Time `json:"date_start"`
	DateEnd         time.Time `json:"date_end"`
	MaxSubmissions  int       `json:"max_submissions"`
	OwnSubmissions  int       `json:"own_submissions"`
	MediaTypes      string    `json:"media_types"`
	NumEntries      int       `json:"num_entries"`
	Deleted         bool      `json:"-"`
	// Allow voting
	Voting bool `json:"voting"`
	// Minimum permission level allowed to vote
	MinPermissionLevel Privilege `json:"min_permission_level"`
	// Allow down voting
	DownVotes bool `json:"down_votes"`
	IsNew     bool
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
	VotesUp        int           `json:"votes_up"`
	VotesDown      int           `json:"votes_down"`
	Asset          Asset         `json:"asset"`
}

type ContestEntryVote struct {
	ContestEntryID uuid.UUID     `json:"contest_entry_id"`
	SteamID        steamid.SID64 `json:"steam_id"`
	Vote           int           `json:"vote"`
	TimeStamped
}

func (c Contest) NewEntry(steamID steamid.SID64, assetID uuid.UUID, description string) (ContestEntry, error) {
	if c.ContestID.IsNil() {
		return ContestEntry{}, ErrInvalidContestID
	}

	if !steamID.Valid() {
		return ContestEntry{}, ErrInvalidSID
	}

	if description == "" {
		return ContestEntry{}, ErrInvalidDescription
	}

	newID, errID := uuid.NewV4()
	if errID != nil {
		return ContestEntry{}, errors.Join(errID, ErrUUIDCreate)
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
		return Contest{}, errors.Join(errID, ErrUUIDCreate)
	}

	if title == "" {
		return Contest{}, ErrTitleEmpty
	}

	if description == "" {
		return Contest{}, ErrDescriptionEmpty
	}

	if dateEnd.Before(dateStart) {
		return Contest{}, ErrEndDateBefore
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
		HideSubmissions:    false,
		MediaTypes:         "",
		Deleted:            false,
		Voting:             false,
		MinPermissionLevel: PUser,
		DownVotes:          false,
		IsNew:              true,
	}

	return contest, nil
}

type ContentVoteRecord struct {
	ContestEntryVoteID int64         `json:"contest_entry_vote_id"`
	ContestEntryID     uuid.UUID     `json:"contest_entry_id"`
	SteamID            steamid.SID64 `json:"steam_id"`
	Vote               bool          `json:"vote"`
	TimeStamped
}
