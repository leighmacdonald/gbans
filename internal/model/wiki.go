package model

import (
	"errors"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

const unknownMediaTag = "__unknown__"

func NewMedia(author steamid.SID64, name string, mime string, content []byte) (Media, error) {
	mType := mimetype.Detect(content)
	if !mType.Is(mime) && mime != unknownMediaTag {
		// Should never actually happen unless user is trying nefarious stuff.
		return Media{}, errors.New("Detected mimetype different than provided")
	}

	curTime := time.Now()

	return Media{
		AuthorID:  author,
		MimeType:  mType.String(),
		Name:      strings.ReplaceAll(name, " ", "_"),
		Size:      int64(len(content)),
		Contents:  content,
		Deleted:   false,
		CreatedOn: curTime,
		UpdatedOn: curTime,
		Asset:     Asset{},
	}, nil
}

type Media struct {
	MediaID   int           `json:"media_id"`
	AuthorID  steamid.SID64 `json:"author_id"`
	MimeType  string        `json:"mime_type"`
	Contents  []byte        `json:"-"`
	Name      string        `json:"name"`
	Size      int64         `json:"size"`
	Deleted   bool          `json:"deleted"`
	CreatedOn time.Time     `json:"created_on"`
	UpdatedOn time.Time     `json:"updated_on"`
	Asset     Asset         `json:"asset"`
}