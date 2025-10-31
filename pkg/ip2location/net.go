package ip2location

import (
	"encoding/binary"
	"net"
)

func Int2IP(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)

	return ip
}
