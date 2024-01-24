// Package wiki provides some convenience wrappers around the gomarkdown
// and bluemonday markdown & sanitization libraries
package wiki

import (
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/microcosm-cc/bluemonday"
)

// RootSlug is the top-most (index) page of the wiki.
const RootSlug = "home"

type Page struct {
	Slug            string           `json:"slug"`
	BodyMD          string           `json:"body_md"`
	Revision        int              `json:"revision"`
	PermissionLevel domain.Privilege `json:"permission_level"`
	CreatedOn       time.Time        `json:"created_on"`
	UpdatedOn       time.Time        `json:"updated_on"`
}

func (page *Page) NewRevision() Page {
	return Page{
		Slug:      page.Slug,
		BodyMD:    page.BodyMD,
		Revision:  page.Revision + 1,
		CreatedOn: page.CreatedOn,
		UpdatedOn: time.Now(),
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
