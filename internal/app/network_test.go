package app_test

import (
	"context"
	"net"
	"testing"

	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/stretchr/testify/require"
)

func TestNetworkBlocker(t *testing.T) {
	testName := "test"
	testSource := "https://raw.githubusercontent.com/X4BNet/lists_vpn/main/output/datacenter/ipv4.txt"
	blocker := app.NewNetworkBlocker()

	count, errAdd := blocker.AddRemoteSource(context.Background(), testName, testSource)
	require.NoError(t, errAdd)
	require.True(t, count > 100)

	matched, name := blocker.IsMatch(net.ParseIP("3.2.2.2"))
	require.True(t, matched)
	require.Equal(t, testName, name)

	noMatch, noName := blocker.IsMatch(net.ParseIP("1.1.1.1"))
	require.False(t, noMatch)
	require.Equal(t, "", noName)

	blocker.RemoveSource(testName)

	noMatch2, _ := blocker.IsMatch(net.ParseIP("3.2.2.2"))
	require.False(t, noMatch2)
}
