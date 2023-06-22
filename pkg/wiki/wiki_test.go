package wiki_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/pkg/wiki"

	"github.com/stretchr/testify/require"
)

func TestWiki(t *testing.T) {
	root := wiki.Page{Slug: wiki.RootSlug}
	root.BodyMD = `
# Title

- list 1
- list 2
`
	v := root.Render()
	require.Equal(t,
		[]byte(`<h1 id="title">Title</h1>

<ul>
<li>list 1</li>
<li>list 2</li>
</ul>
`),
		v,
		"Invalid markdown output")
}
