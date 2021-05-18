package consts

import "github.com/pkg/errors"

var (
	ErrDuplicateBan    = errors.New("Duplicate ban")
	ErrInvalidDuration = errors.New("Invalid duration")
	ErrInvalidSID      = errors.New("Invalid steamid")
	ErrInternal        = errors.New("Internal error :(")
	ErrUnknownID       = errors.New("Could not find matching player/steamid")
)
