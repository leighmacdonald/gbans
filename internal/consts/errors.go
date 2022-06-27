// Package consts contains common errors and constants
package consts

import "github.com/pkg/errors"

var (
	ErrDuplicateBan     = errors.New("Duplicate ban")
	ErrAuthentication   = errors.New("Auth invalid")
	ErrInvalidDuration  = errors.New("Invalid duration")
	ErrInvalidSID       = errors.New("Invalid steamid")
	ErrInvalidTeam      = errors.New("Invalid team")
	ErrMalformedRequest = errors.New("Malformed request")
	ErrInternal         = errors.New("Internal error :(")
	ErrUnknownID        = errors.New("Could not find matching server/player/steamid")
	ErrUnlinkedAccount  = errors.New("You must link your steam and discord accounts, see: /set_steam")
)
