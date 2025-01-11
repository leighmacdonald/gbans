package tests_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/leighmacdonald/gbans/pkg/fs"
	"os"
	"path"
	"testing"

	"github.com/leighmacdonald/gbans/internal/demo"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/demostats"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/stretchr/testify/require"
)

func TestDemosCleanup(t *testing.T) {
	ctx := context.Background()

	tempDir, errDir := os.MkdirTemp("", "test-assets")
	require.NoError(t, errDir)

	conf := configUC.Config()
	conf.LocalStore.PathRoot = tempDir
	conf.Demo.DemoCleanupEnabled = true
	conf.Demo.DemoCleanupStrategy = domain.DemoStrategyCount
	conf.Demo.DemoCountLimit = 5

	require.NoError(t, configUC.Write(ctx, conf))

	fetcher := demo.NewFetcher(tempDB, configUC, serversUC, assetUC, demoUC)

	for demoNum := range 10 {
		content := make([]byte, 100000)
		_, err := rand.Read(content)
		require.NoError(t, err)
		if configUC.Config().Demo.DemoParserURL != "" {
			require.NoError(t, fetcher.OnDemoReceived(ctx, demo.UploadedDemo{
				Name:    fmt.Sprintf("2023111%d-063943-koth_harvest_final.dem", demoNum),
				Server:  testServer,
				Content: content,
			}))
		}
	}

	expired, errExpired := demoRepository.ExpiredDemos(ctx, 5)
	require.NoError(t, errExpired)
	for _, expiredDemo := range expired {
		require.Less(t, expiredDemo.DemoID, int64(6))
	}

	demoUC.Cleanup(ctx)

	allDemos, err := demoUC.GetDemos(ctx)
	require.NoError(t, err)
	if configUC.Config().Demo.DemoParserURL != "" {
		require.Len(t, allDemos, 5)
	}
}

func TestDemoUpload(t *testing.T) {
	inputResponse, errOpen := os.Open(fs.FindFile(path.Join("testdata", "demostats-good.json"), "gbans"))
	require.NoError(t, errOpen)
	details, errGood := demostats.ParseReader(inputResponse)
	require.NoError(t, errGood)
	match, errSave := matchUC.MatchSaveFromDemo(context.Background(), details, fp.MutexMap[logparse.Weapon, int]{})
	require.NoError(t, errSave)
	require.False(t, match.MatchID.IsNil())
}
