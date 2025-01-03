package tests_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/leighmacdonald/gbans/internal/demo"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/fs"
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
	if configUC.Config().Demo.DemoParserURL == "" {
		t.Skip("Parser url undefined")
	}
	demoPath := fs.FindFile(path.Join("testdata", "test.dem"), "gbans")
	detail, err := demoUC.SendAndParseDemo(context.Background(), demoPath)
	require.NoError(t, err)
	require.Len(t, detail.State.Users, 46)
}
