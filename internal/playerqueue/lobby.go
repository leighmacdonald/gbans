package playerqueue

import (
	"errors"
	"net"
)

type Lobby struct {
	ServerID    int
	PlayerCount int
	MaxPlayers  int
	Title       string
	ShortName   string
	Hostname    string
	Port        uint16
	CC          string
	Members     []ClientQueueState
}

func (l *Lobby) IP() (net.IP, error) {
	ipAddr, errIP := net.ResolveIPAddr("ip4", l.Hostname)
	if errIP != nil {
		return nil, errors.Join(errIP, ErrHostname)
	}

	return ipAddr.IP, nil
}
