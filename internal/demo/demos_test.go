package demo_test

import (
	"context"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/demo"
	demostatsv1 "github.com/leighmacdonald/gbans/internal/demostats/v1"
	"github.com/leighmacdonald/gbans/internal/demostats/v1/demostatsv1connect"
	"github.com/leighmacdonald/gbans/internal/json"
	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/stats"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

// fixture provides a shared set of common dependencies that can be used for integration testing.
var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}

// fakeParser serves fixture parse results over ConnectRPC like a
// tf2_demostats v0.3.x server.
type fakeParser struct {
	demo *demoparse.Demo
}

func (f fakeParser) ParseDemo(_ context.Context, req *demostatsv1.ParseDemoRequest) (*demostatsv1.ParseDemoResponse, error) {
	resp := demoparse.ProtoFromDemo(f.demo)
	if req.GetFilename() != "" {
		resp.GetDemo().Filename = req.GetFilename()
	}

	return resp, nil
}

// TestDemoUploadDeduplication ensures that uploading the same demo twice does not create
// duplicate asset, demo, or stats entries.
func TestDemoUploadDeduplication(t *testing.T) {
	demoFile, err := os.Open(filepath.Join("..", "stats", "testdata", "demo-1427611.json"))
	require.NoError(t, err)

	defer demoFile.Close()

	parsed, err := json.Decode[demoparse.Demo](demoFile)
	require.NoError(t, err)

	_, handler := demostatsv1connect.NewDemoServiceHandler(fakeParser{demo: &parsed})
	parser := httptest.NewServer(handler)

	defer parser.Close()

	fixture.Config.Config().Demo.DemoParserURL = parser.URL
	ownerSID := steamid.New(fixture.Config.Config().Owner)
	require.NoError(t, fixture.Persons.EnsurePerson(t.Context(), ownerSID))

	var (
		server  = fixture.CreateTestServer(t.Context())
		assets  = asset.NewAssets(asset.NewLocalRepository(fixture.Database, t.TempDir()))
		filters = chat.NewWordFilters(chat.NewWordFilterRepository(fixture.Database),
			notification.NewDiscard(), fixture.Config.Config().Filters)
		chatUC = chat.New(chat.NewRepository(fixture.Database), fixture.Config.Config().Filters,
			filters, fixture.Persons, notification.NewDiscard(), nil, "")
		statsUC = stats.New(stats.NewRepository(fixture.Database), maps.New(maps.NewRepository(fixture.Database)))
		demos   = demo.NewDemos(asset.BucketDemo, demo.NewRepository(fixture.Database),
			assets, statsUC, chatUC, fixture.Persons, fixture.Config.Config().Demo, ownerSID)
	)

	// The demo filename must match the YYYYMMDD-HHMMSS-map.dem format used to derive the map name.
	src, err := os.Open(filepath.Join("testdata", "test.dem"))
	require.NoError(t, err)
	defer src.Close()

	demoPath := filepath.Join(t.TempDir(), "20240101-120000-koth_lakeside_f5.dem")
	dst, err := os.Create(demoPath)
	require.NoError(t, err)

	_, err = io.Copy(dst, src)
	require.NoError(t, err)
	require.NoError(t, dst.Close())

	countRows := func(table string) int {
		var count int
		errCount := fixture.Database.QueryRow(t.Context(),
			"SELECT count(*) FROM "+table).Scan(&count)
		require.NoError(t, errCount)

		return count
	}

	first, err := demos.ImportFile(t.Context(), server.ServerID, demoPath, true, false)
	require.NoError(t, err)

	counts := map[string]int{
		"asset":                       countRows("asset"),
		"demo":                        countRows("demo"),
		"match":                       countRows("match"),
		"match_round":                 countRows("match_round"),
		"match_round_player":          countRows("match_round_player"),
		"match_round_player_variants": countRows("match_round_player_variants"),
		"person_messages":             countRows("person_messages"),
	}

	second, err := demos.ImportFile(t.Context(), server.ServerID, demoPath, true, false)
	require.NoError(t, err)

	for table, expected := range counts {
		require.Equalf(t, expected, countRows(table), "unexpected duplicate rows in %s", table)
	}
	require.Equal(t, first.AssetID, second.AssetID)
	require.Equal(t, first.DemoID, second.DemoID)
}

func TestUploadDemo(t *testing.T) {
	demoFile, err := os.Open(filepath.Join("..", "stats", "testdata", "demo-1427611.json"))
	require.NoError(t, err)

	defer demoFile.Close()

	parsed, err := json.Decode[demoparse.Demo](demoFile)
	require.NoError(t, err)

	_, handler := demostatsv1connect.NewDemoServiceHandler(fakeParser{demo: &parsed})
	parser := httptest.NewServer(handler)

	defer parser.Close()

	fixture.Config.Config().Demo.DemoParserURL = parser.URL
	ownerSID := steamid.New(fixture.Config.Config().Owner)
	require.NoError(t, fixture.Persons.EnsurePerson(t.Context(), ownerSID))

	var (
		server  = fixture.CreateTestServer(t.Context())
		assets  = asset.NewAssets(asset.NewLocalRepository(fixture.Database, t.TempDir()))
		filters = chat.NewWordFilters(chat.NewWordFilterRepository(fixture.Database),
			notification.NewDiscard(), fixture.Config.Config().Filters)
		chatUC = chat.New(chat.NewRepository(fixture.Database), fixture.Config.Config().Filters,
			filters, fixture.Persons, notification.NewDiscard(), nil, "")
		statsUC = stats.New(stats.NewRepository(fixture.Database), maps.New(maps.NewRepository(fixture.Database)))
		demos   = demo.NewDemos(asset.BucketDemo, demo.NewRepository(fixture.Database),
			assets, statsUC, chatUC, fixture.Persons, fixture.Config.Config().Demo, ownerSID)
	)

	upload := demo.UploadedDemo{
		Name:     "manual_upload.dem",
		ServerID: server.ServerID,
		Content:  []byte("manual demo upload"),
	}
	first, firstMatchID, err := demos.Upload(t.Context(), upload, false)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.Regexp(t, `^\d{8}-\d{6}-manual_upload\.dem\.zstd$`, func() string {
		uploadedAsset, errAsset := assets.Get(t.Context(), first.AssetID)
		require.NoError(t, errAsset)

		return uploadedAsset.Name
	}())

	match, errMatch := statsUC.Match(t.Context(), firstMatchID)
	require.NoError(t, errMatch)
	require.Equal(t, first.DemoID, match.DemoID)

	second, secondMatchID, err := demos.Upload(t.Context(), upload, false)
	require.NoError(t, err)
	require.Equal(t, first.AssetID, second.AssetID)
	require.Equal(t, first.DemoID, second.DemoID)
	require.Equal(t, firstMatchID, secondMatchID)

	_, _, invalidNameErr := demos.Upload(t.Context(), demo.UploadedDemo{
		Name:     "manual_upload.txt",
		ServerID: server.ServerID,
		Content:  []byte("manual demo upload"),
	}, false)
	require.ErrorIs(t, invalidNameErr, demo.ErrDemoFilename)

	_, _, invalidServerErr := demos.Upload(t.Context(), demo.UploadedDemo{
		Name:     "20240101-120000-koth_lakeside_f5.dem",
		ServerID: -1,
		Content:  []byte("manual demo upload"),
	}, false)
	require.ErrorIs(t, invalidServerErr, demo.ErrServerValidate)
}
