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
	ErrUnknownID        = errors.New("Could not find matching server/player/steamid")
	ErrInvalidAuthorSID = errors.New("Invalid author steam id")
	ErrInvalidTargetSID = errors.New("Invalid target steam id")
	ErrInternal         = errors.New("internal server error")
	ErrBadRequest       = errors.New("invalid request")
	ErrNotFound         = errors.New("entity not found")
	ErrDuplicate        = errors.New("entity already exists")
	ErrInvalidParameter = errors.New("invalid parameter format")
)
