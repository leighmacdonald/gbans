// Package wiki provides some convenience wrappers around the gomarkdown
// and bluemonday markdown & sanitization libraries
package wiki

import (
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/microcosm-cc/bluemonday"
	"time"
)

// RootSlug is the top-most (index) page of the wiki
const RootSlug = "home"

type Media struct {
	WikiMediaId int           `json:"report_media_id"`
	AuthorId    steamid.SID64 `json:"author_id"`
	MimeType    string        `json:"mime_type"`
	Contents    []byte        `json:"-"`
	Name        string        `json:"name"`
	Size        int64         `json:"size"`
	Deleted     bool          `json:"deleted"`
	CreatedOn   time.Time     `json:"created_on"`
	UpdatedOn   time.Time     `json:"updated_on"`
}

func NewMedia(author steamid.SID64, name string, mime string, content []byte) Media {
	return Media{
		AuthorId:  author,
		MimeType:  mime,
		Name:      name,
		Size:      int64(len(content)),
		Contents:  content,
		Deleted:   false,
		CreatedOn: config.Now(),
		UpdatedOn: config.Now(),
	}
}

type Page struct {
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	BodyMD    string    `json:"body_md"`
	Revision  int       `json:"revision"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

func (page *Page) NewRevision() Page {
	return Page{
		Slug:      page.Slug,
		Title:     page.Title,
		BodyMD:    page.BodyMD,
		Revision:  page.Revision + 1,
		CreatedOn: page.CreatedOn,
		UpdatedOn: config.Now(),
	}
}

func (page *Page) Render() []byte {
	unsafeHTML := markdown.ToHTML([]byte(page.BodyMD), NewParser(), nil)
	return bluemonday.UGCPolicy().SanitizeBytes(unsafeHTML)
}

func NewParser() *parser.Parser {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.Tables
	return parser.NewWithExtensions(extensions)
}
