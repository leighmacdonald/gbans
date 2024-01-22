// Package errs contains commonly shared errors
package errs

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrAuthentication   = errors.New("Auth invalid")
	ErrExpired          = errors.New("expired")
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
	ErrNoResult         = errors.New("No results found")
	ErrDuplicate        = errors.New("entity already exists")
	ErrInvalidParameter = errors.New("invalid parameter format")
	ErrUnknownServer    = errors.New("Unknown server")
	ErrVoteDeleted      = errors.New("Vote deleted")
)

// DBErr is used to wrap common database errors in owr own error types.
func DBErr(rootError error) error {
	if rootError == nil {
		return nil
	}

	if errors.Is(rootError, pgx.ErrNoRows) {
		return ErrNoResult
	}

	var pgErr *pgconn.PgError

	if errors.As(rootError, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return ErrDuplicate
		default:
			return rootError
		}
	}

	return rootError
}
