package thirdparty

import (
	"net"
	"strings"

	"github.com/leighmacdonald/steamid/v3/steamid"
)

// parseValveSID parses the format: banid 0 STEAM_0:1:16683555.
func parseValveSID(src []byte) (steamid.Collection, error) {
	var steamIds steamid.Collection
	for _, line := range strings.Split(string(src), "\r\n") {
		pieces := strings.SplitN(line, " ", 3)
		if len(pieces) != 3 {
			continue
		}
		sid64 := steamid.SIDToSID64(steamid.SID(pieces[2]))
		if !sid64.Valid() {
			continue
		}
		steamIds = append(steamIds, sid64)
	}

	return steamIds, nil
}

// parseValveNet parses the format: addip 0 89.229.79.121.
func parseValveNet(src []byte) ([]*net.IPNet, error) {
	var valveNetworks []*net.IPNet //nolint:prealloc
	for _, line := range strings.Split(string(src), "\r\n") {
		pieces := strings.SplitN(line, " ", 3)
		if len(pieces) != 3 {
			continue
		}
		_, network, errParseCIDR := net.ParseCIDR(pieces[2] + `\32`)
		if errParseCIDR != nil {
			continue
		}
		valveNetworks = append(valveNetworks, network)
	}

	return valveNetworks, nil
}
