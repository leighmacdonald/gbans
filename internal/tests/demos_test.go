package tests_test

import (
	"crypto/rand"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/leighmacdonald/gbans/internal/demo"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/demoparse"
	"github.com/leighmacdonald/gbans/pkg/fs"
	"github.com/stretchr/testify/require"
)

func TestDemosCleanup(t *testing.T) {
	tempDir := os.TempDir()

	conf := configUC.Config()
	conf.LocalStore.PathRoot = tempDir
	conf.Demo.DemoCleanupEnabled = true
	conf.Demo.DemoCleanupStrategy = domain.DemoStrategyCount
	conf.Demo.DemoCountLimit = 5

	require.NoError(t, configUC.Write(t.Context(), conf))

	fetcher := demo.NewFetcher(tempDB, configUC, serversUC, assetUC, demoUC, anticheatUC)

	for demoNum := range 10 {
		content := make([]byte, 100000)
		_, err := rand.Read(content)
		require.NoError(t, err)
		if configUC.Config().Demo.DemoParserURL != "" {
			require.NoError(t, fetcher.OnDemoReceived(t.Context(), demo.UploadedDemo{
				Name:    fmt.Sprintf("2023111%d-063943-koth_harvest_final.dem", demoNum),
				Server:  testServer,
				Content: content,
			}))
		}
	}

	expired, errExpired := demoRepository.ExpiredDemos(t.Context(), 5)
	require.NoError(t, errExpired)
	for _, expiredDemo := range expired {
		require.Less(t, expiredDemo.DemoID, int64(6))
	}

	demoUC.Cleanup(t.Context())

	allDemos, err := demoUC.GetDemos(t.Context())
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
	detail, err := demoparse.Submit(t.Context(), configUC.Config().Demo.DemoParserURL, demoPath)
	require.NoError(t, err)
	require.Len(t, detail.State.Users, 46)
}
