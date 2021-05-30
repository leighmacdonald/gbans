package consts

import "github.com/pkg/errors"

var (
	ErrDuplicateBan    = errors.New("Duplicate ban")
	ErrAuthhentication = errors.New("Auth invalid")
	ErrInvalidDuration = errors.New("Invalid duration")
	ErrInvalidSID      = errors.New("Invalid steamid")
	ErrInternal        = errors.New("Internal error :(")
	ErrUnknownID       = errors.New("Could not find matching player/steamid")
	ErrUnlinkedAccount = errors.New("You must link your steam and discord accounts, see: /set_steam")
)
