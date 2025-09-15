package contest

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
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
	MinPermissionLevel permission.Privilege `json:"min_permission_level"`
	// Allow down voting
	DownVotes bool `json:"down_votes"`
	IsNew     bool
}

type Entry struct {
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

type Vote struct {
	ContestEntryID uuid.UUID       `json:"contest_entry_id"`
	SteamID        steamid.SteamID `json:"steam_id"`
	Vote           int             `json:"vote"`
	CreatedOn      time.Time       `json:"created_on"`
	UpdatedOn      time.Time       `json:"updated_on"`
}

func (c Contest) NewEntry(steamID steamid.SteamID, assetID uuid.UUID, description string) (Entry, error) {
	if c.ContestID.IsNil() {
		return Entry{}, ErrInvalidContestID
	}

	if !steamID.Valid() {
		return Entry{}, domain.ErrInvalidSID
	}

	if description == "" {
		return Entry{}, ErrInvalidDescription
	}

	newID, errID := uuid.NewV4()
	if errID != nil {
		return Entry{}, errors.Join(errID, domain.ErrUUIDCreate)
	}

	return Entry{
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
		MinPermissionLevel: permission.PUser,
		DownVotes:          false,
		IsNew:              true,
		CreatedOn:          time.Now(),
		UpdatedOn:          time.Now(),
	}

	return contest, nil
}

type VoteRecord struct {
	ContestEntryVoteID int64           `json:"contest_entry_vote_id"`
	ContestEntryID     uuid.UUID       `json:"contest_entry_id"`
	SteamID            steamid.SteamID `json:"steam_id"`
	Vote               bool            `json:"vote"`
	CreatedOn          time.Time       `json:"created_on"`
	UpdatedOn          time.Time       `json:"updated_on"`
}

type Contests struct {
	repository Repository
}

func NewContests(repository Repository) Contests {
	return Contests{repository: repository}
}

func (c *Contests) Save(ctx context.Context, contest Contest) (Contest, error) {
	if contest.ContestID.IsNil() {
		newID, errID := uuid.NewV4()
		if errID != nil {
			return contest, errors.Join(errID, domain.ErrUUIDCreate)
		}

		contest.ContestID = newID
	}

	if errSave := c.repository.ContestSave(ctx, &contest); errSave != nil {
		return contest, errSave
	}

	slog.Info("Contest updated",
		slog.String("contest_id", contest.ContestID.String()),
		slog.String("title", contest.Title))

	return contest, nil
}

func (c *Contests) ByID(ctx context.Context, contestID uuid.UUID, contest *Contest) error {
	return c.repository.ContestByID(ctx, contestID, contest)
}

func (c *Contests) ContestDelete(ctx context.Context, contestID uuid.UUID) error {
	if err := c.repository.ContestDelete(ctx, contestID); err != nil {
		return err
	}

	slog.Info("Contest deleted", slog.String("contest_id", contestID.String()))

	return nil
}

func (c *Contests) EntryDelete(ctx context.Context, contestEntryID uuid.UUID) error {
	return c.repository.ContestEntryDelete(ctx, contestEntryID)
}

func (c *Contests) Contests(ctx context.Context, user domain.PersonInfo) ([]Contest, error) {
	return c.repository.Contests(ctx, !user.HasPermission(permission.PModerator))
}

func (c *Contests) Entry(ctx context.Context, contestID uuid.UUID, entry *Entry) error {
	return c.repository.ContestEntry(ctx, contestID, entry)
}

func (c *Contests) EntrySave(ctx context.Context, entry Entry) error {
	return c.repository.ContestEntrySave(ctx, entry)
}

func (c *Contests) Entries(ctx context.Context, contestID uuid.UUID) ([]*Entry, error) {
	return c.repository.ContestEntries(ctx, contestID)
}

func (c *Contests) EntryVoteGet(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SteamID, record *VoteRecord) error {
	return c.repository.ContestEntryVoteGet(ctx, contestEntryID, steamID, record)
}

func (c *Contests) EntryVote(ctx context.Context, contestID uuid.UUID, contestEntryID uuid.UUID, user domain.PersonInfo, vote bool) error {
	var contest Contest
	if errContests := c.ByID(ctx, contestID, &contest); errContests != nil {
		return errContests
	}

	if !contest.Public && !user.HasPermission(permission.PModerator) {
		return permission.ErrPermissionDenied
	}

	if !contest.Voting || !contest.DownVotes && !vote {
		return httphelper.ErrBadRequest // tODO proper error
	}

	if err := c.repository.ContestEntryVote(ctx, contestEntryID, user.GetSteamID(), vote); err != nil {
		return err
	}

	sid := user.GetSteamID()

	slog.Info("Entry vote registered", slog.String("contest_id", contest.ContestID.String()), slog.Bool("vote", vote), slog.String("steam_id", sid.String()))

	return nil
}

func (c *Contests) EntryVoteDelete(ctx context.Context, contestEntryVoteID int64) error {
	return c.repository.ContestEntryVoteDelete(ctx, contestEntryVoteID)
}

func (c *Contests) EntryVoteUpdate(ctx context.Context, contestEntryVoteID int64, newVote bool) error {
	return c.repository.ContestEntryVoteUpdate(ctx, contestEntryVoteID, newVote)
}
