package model

import (
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

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
		return ContestEntry{}, errors.New("Invalid contest id")
	}

	if !steamID.Valid() {
		return ContestEntry{}, errs.ErrInvalidSID
	}

	if description == "" {
		return ContestEntry{}, errors.New("Description cannot be empty")
	}

	newID, errID := uuid.NewV4()
	if errID != nil {
		return ContestEntry{}, errors.Join(errID, errors.New("Failed to generate new uuidv4"))
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
		return Contest{}, errors.Join(errID, errors.New("Failed to generate uuid"))
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
