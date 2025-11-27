package wiki_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/internal/wiki"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestRender(t *testing.T) {
	root := wiki.Page{Slug: wiki.RootSlug}
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

func TestWiki(t *testing.T) {
	testDB := tests.NewFixture()
	defer testDB.Close()

	wikiCase := wiki.NewWiki(wiki.NewRepository(testDB.Database), notification.NewDiscard(), "", "")
	page := wiki.NewPage(stringutil.SecureRandomString(10), stringutil.SecureRandomString(500))
	saved, errSave := wikiCase.Save(t.Context(), page)
	require.NoError(t, errSave)
	require.Equal(t, page.BodyMD, saved.BodyMD)
	require.Equal(t, page.Slug, saved.Slug)

	saved2, errSave2 := wikiCase.Save(t.Context(), saved)
	require.NoError(t, errSave2)

	require.Equal(t, saved.Revision+1, saved2.Revision)
}
