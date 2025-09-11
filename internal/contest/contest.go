package contest

import (
	"context"
	"errors"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrInvalidContestID   = errors.New("invalid contest id")
	ErrInvalidDescription = errors.New("invalid description, cannot be empty")
	ErrTitleEmpty         = errors.New("title cannot be empty")
	ErrDescriptionEmpty   = errors.New("description cannot be empty")
	ErrEndDateBefore      = errors.New("end date comes before start date")
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
	ContestEntryVoteGet(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SteamID, record *ContentVoteRecord) error
	ContestEntryVote(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SteamID, vote bool) error
	ContestEntryVoteDelete(ctx context.Context, contestEntryVoteID int64) error
	ContestEntryVoteUpdate(ctx context.Context, contestEntryVoteID int64, newVote bool) error
}

type ContestUsecase interface {
	ContestSave(ctx context.Context, contest Contest) (Contest, error)
	ContestByID(ctx context.Context, contestID uuid.UUID, contest *Contest) error
	ContestDelete(ctx context.Context, contestID uuid.UUID) error
	ContestEntryDelete(ctx context.Context, contestEntryID uuid.UUID) error
	Contests(ctx context.Context, user domain.PersonInfo) ([]Contest, error)
	ContestEntry(ctx context.Context, contestID uuid.UUID, entry *ContestEntry) error
	ContestEntrySave(ctx context.Context, entry ContestEntry) error
	ContestEntries(ctx context.Context, contestID uuid.UUID) ([]*ContestEntry, error)
	ContestEntryVoteGet(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SteamID, record *ContentVoteRecord) error
	ContestEntryVote(ctx context.Context, contestID uuid.UUID, contestEntryID uuid.UUID, user domain.PersonInfo, vote bool) error
	ContestEntryVoteDelete(ctx context.Context, contestEntryVoteID int64) error
	ContestEntryVoteUpdate(ctx context.Context, contestEntryVoteID int64, newVote bool) error
}

type Contest struct {
	CreatedOn       time.Time `json:"created_on"`
	UpdatedOn       time.Time `json:"updated_on"`
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
	MinPermissionLevel domain.Privilege `json:"min_permission_level"`
	// Allow down voting
	DownVotes bool `json:"down_votes"`
	IsNew     bool
}

type ContestEntry struct {
	CreatedOn      time.Time       `json:"created_on"`
	UpdatedOn      time.Time       `json:"updated_on"`
	ContestEntryID uuid.UUID       `json:"contest_entry_id"`
	ContestID      uuid.UUID       `json:"contest_id"`
	SteamID        steamid.SteamID `json:"steam_id"`
	Personaname    string          `json:"personaname"`
	AvatarHash     string          `json:"avatar_hash"`
	AssetID        uuid.UUID       `json:"asset_id"`
	Description    string          `json:"description"`
	Placement      int             `json:"placement"`
	Deleted        bool            `json:"deleted"`
	VotesUp        int             `json:"votes_up"`
	VotesDown      int             `json:"votes_down"`
	Asset          asset.Asset     `json:"asset"`
}

type ContestEntryVote struct {
	ContestEntryID uuid.UUID       `json:"contest_entry_id"`
	SteamID        steamid.SteamID `json:"steam_id"`
	Vote           int             `json:"vote"`
	CreatedOn      time.Time       `json:"created_on"`
	UpdatedOn      time.Time       `json:"updated_on"`
}

func (c Contest) NewEntry(steamID steamid.SteamID, assetID uuid.UUID, description string) (ContestEntry, error) {
	if c.ContestID.IsNil() {
		return ContestEntry{}, ErrInvalidContestID
	}

	if !steamID.Valid() {
		return ContestEntry{}, domain.ErrInvalidSID
	}

	if description == "" {
		return ContestEntry{}, ErrInvalidDescription
	}

	newID, errID := uuid.NewV4()
	if errID != nil {
		return ContestEntry{}, errors.Join(errID, domain.ErrUUIDCreate)
	}

	return ContestEntry{
		CreatedOn:      time.Now(),
		UpdatedOn:      time.Now(),
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
		return Contest{}, errors.Join(errID, domain.ErrUUIDCreate)
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
		MinPermissionLevel: domain.PUser,
		DownVotes:          false,
		IsNew:              true,
		CreatedOn:          time.Now(),
		UpdatedOn:          time.Now(),
	}

	return contest, nil
}

type ContentVoteRecord struct {
	ContestEntryVoteID int64           `json:"contest_entry_vote_id"`
	ContestEntryID     uuid.UUID       `json:"contest_entry_id"`
	SteamID            steamid.SteamID `json:"steam_id"`
	Vote               bool            `json:"vote"`
	CreatedOn          time.Time       `json:"created_on"`
	UpdatedOn          time.Time       `json:"updated_on"`
}
