// Package wiki provides some convenience wrappers around the gomarkdown
// and bluemonday markdown & sanitization libraries
package wiki

import (
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/microcosm-cc/bluemonday"
)

func Render(page Page) []byte {
	unsafeHTML := markdown.ToHTML([]byte(page.BodyMD), NewParser(), nil)

	return bluemonday.UGCPolicy().SanitizeBytes(unsafeHTML)
}

func NewParser() *parser.Parser {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.Tables

	return parser.NewWithExtensions(extensions)
}
