// Package consts contains common errors and constants
package consts

import "github.com/pkg/errors"

var (
	ErrAuthentication   = errors.New("Auth invalid")
	ErrExpired          = errors.New("expired")
	ErrInvalidDuration  = errors.New("Invalid duration")
	ErrInvalidSID       = errors.New("Invalid steamid")
	ErrPlayerNotFound   = errors.New("Could not find player")
	ErrInvalidTeam      = errors.New("Invalid team")
	ErrPermissionDenied = errors.New("Permission denied")
	ErrInternal         = errors.New("Internal error :(")
	ErrUnknownID        = errors.New("Could not find matching server/player/steamid")
	ErrInvalidAuthorSID = errors.New("Invalid author steam id")
	ErrInvalidTargetSID = errors.New("Invalid target steam id")
)
