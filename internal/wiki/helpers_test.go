package wiki_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/wiki"
	"github.com/stretchr/testify/require"
)

func TestWiki(t *testing.T) {
	root := domain.Page{Slug: domain.RootSlug}
	root.BodyMD = `
# Title

- list 1
- list 2
`
	rendered := wiki.Render(root)
	require.Equal(t,
		[]byte(`<h1 id="title">Title</h1>

<ul>
<li>list 1</li>
<li>list 2</li>
</ul>
`),
		rendered,
		"Invalid markdown output")
}
