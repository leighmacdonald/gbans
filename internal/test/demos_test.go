package test_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/leighmacdonald/gbans/internal/demo"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestDemosCleanup(t *testing.T) {
	ctx := context.Background()
	fetcher := demo.NewFetcher(tempDB, configUC, serversUC, assetUC, demoUC)
	go demoUC.Start(ctx)

	for demoNum := range 10 {
		content := make([]byte, 100000)
		_, err := rand.Read(content)
		require.NoError(t, err)

		require.NoError(t, fetcher.OnDemoReceived(ctx, demo.UploadedDemo{
			Name:    fmt.Sprintf("2023111%d-063943-koth_harvest_final.dem", demoNum),
			Server:  testServer,
			Content: content,
		}))
	}

	conf := configUC.Config()
	conf.Demo.DemoCleanupEnabled = true
	conf.Demo.DemoCleanupStrategy = domain.DemoStrategyCount
	conf.Demo.DemoCountLimit = 5

	require.NoError(t, configUC.Write(ctx, conf))

	demoUC.TriggerCleanup()

	allDemos, err := demoUC.GetDemos(ctx)
	require.NoError(t, err)
	require.Len(t, len(allDemos), 5)
}
