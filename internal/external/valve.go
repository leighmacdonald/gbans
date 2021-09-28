package external

import (
	"github.com/leighmacdonald/steamid/v2/steamid"
	"net"
	"strings"
)

// parseValveSID parses the format: banid 0 STEAM_0:1:16683555
func parseValveSID(src []byte) (steamid.Collection, error) {
	var ids steamid.Collection
	for _, line := range strings.Split(string(src), "\r\n") {
		pcs := strings.SplitN(line, " ", 3)
		if len(pcs) != 3 {
			continue
		}
		sid := steamid.SIDToSID64(steamid.SID(pcs[2]))
		if !sid.Valid() {
			continue
		}
		ids = append(ids, sid)
	}
	return ids, nil
}

// parseValveNet parses the format: addip 0 89.229.79.121
func parseValveNet(src []byte) ([]*net.IPNet, error) {
	var nets []*net.IPNet
	for _, line := range strings.Split(string(src), "\r\n") {
		pcs := strings.SplitN(line, " ", 3)
		if len(pcs) != 3 {
			continue
		}
		_, cidr, err := net.ParseCIDR(pcs[2] + `\32`)
		if err != nil {
			continue
		}
		nets = append(nets, cidr)
	}
	return nets, nil
}
