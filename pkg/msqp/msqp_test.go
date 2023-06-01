package msqp

import (
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestClient(t *testing.T) {
	s, errResolve := net.ResolveUDPAddr("udp4", masterBrowserHost)
	require.NoError(t, errResolve)
	conn, errDial := net.DialUDP("udp4", nil, s)
	require.NoError(t, errDial)
	_, errList := List(conn, []Region{AllRegions})
	require.NoError(t, errList)
	require.NoError(t, conn.Close())
}
