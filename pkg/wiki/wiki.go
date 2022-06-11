// Package wiki provides some convenience wrappers around the gomarkdown
// and bluemonday markdown & sanitization libraries
package wiki

import (
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/microcosm-cc/bluemonday"
	"time"
)

// RootSlug is the top-most (index) page of the wiki
const RootSlug = "home"

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
