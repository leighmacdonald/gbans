package domain

import (
	"net"

	"github.com/leighmacdonald/steamid/v3/steamid"
)

type GroupBlockerUsecase interface {
	IsMember(steamID steamid.SID64) (steamid.GID, bool)
}

type NetBlockerUsecase interface {
	IsMatch(addr net.IP) (string, bool)
}

type FriendBlockerUsecase interface {
	IsMember(steamID steamid.SID64) (steamid.SID64, bool)
}
