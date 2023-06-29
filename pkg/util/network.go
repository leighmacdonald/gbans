package util

import (
	"encoding/binary"
	"net"
	"net/http"
	"time"
)

// NewHTTPClient allocates a preconfigured *http.Client.
func NewHTTPClient() *http.Client {
	c := &http.Client{
		Timeout: time.Second * 10,
	}

	return c
}

func IP2Int(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}

	return binary.BigEndian.Uint32(ip)
}

func Int2IP(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)

	return ip
}
