package wiki

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWiki(t *testing.T) {
	root := Page{Slug: RootSlug}
	root.Title = "test"
	root.BodyMD = []byte(`
# Title

- list 1
- list 2
`)
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
