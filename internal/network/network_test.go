package network_test

import (
	"context"
	"net/netip"
	"testing"

	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/stretchr/testify/require"
)

func TestNetworkBlocker(t *testing.T) {
	testName := "test"
	testSource := "https://raw.githubusercontent.com/X4BNet/lists_vpn/main/output/datacenter/ipv4.txt"
	blocker := network.NewBlocker()

	count, errAdd := blocker.AddRemoteSource(context.Background(), testName, testSource)
	require.NoError(t, errAdd)
	require.Greater(t, count, int64(100))

	name, matched := blocker.IsMatch(netip.MustParseAddr("3.2.2.2"))
	require.True(t, matched)
	require.Equal(t, testName, name)

	noName, noMatch := blocker.IsMatch(netip.MustParseAddr("1.1.1.1"))
	require.False(t, noMatch)
	require.Equal(t, "", noName)

	blocker.RemoveSource(testName)

	_, noMatch2 := blocker.IsMatch(netip.MustParseAddr("3.2.2.2"))
	require.False(t, noMatch2)
}
