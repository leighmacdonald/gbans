// Package consts contains common errors and constants
package consts

import "github.com/pkg/errors"

var (
	ErrAuthentication   = errors.New("Auth invalid")
	ErrInvalidDuration  = errors.New("Invalid duration")
	ErrInvalidSID       = errors.New("Invalid steamid")
	ErrInvalidTeam      = errors.New("Invalid team")
	ErrPermissionDenied = errors.New("Permission denied")
	ErrInternal         = errors.New("Internal error :(")
	ErrUnknownID        = errors.New("Could not find matching server/player/steamid")
)
