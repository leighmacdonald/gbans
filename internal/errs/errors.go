// Package errs contains commonly shared errors
package errs

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInvalidIP          = errors.New("invalid ip, could not parse")
	ErrInvalidCIDR        = errors.New("invalid cidr")
	ErrAuthentication     = errors.New("auth invalid")
	ErrExpired            = errors.New("expired")
	ErrInvalidSID         = errors.New("invalid steamid")
	ErrSourceID           = errors.New("invalid source steam id")
	ErrTargetID           = errors.New("invalid target steam id")
	ErrPlayerNotFound     = errors.New("could not find player")
	ErrInvalidTeam        = errors.New("invalid team")
	ErrUnknownID          = errors.New("could not find matching server/player/steamid")
	ErrInvalidAuthorSID   = errors.New("invalid author steam id")
	ErrInvalidTargetSID   = errors.New("invalid target steam id")
	ErrNotFound           = errors.New("entity not found")
	ErrNoResult           = errors.New("no results found")
	ErrDuplicate          = errors.New("entity already exists")
	ErrUnknownServer      = errors.New("unknown server")
	ErrVoteDeleted        = errors.New("vote deleted")
	ErrCreateRequest      = errors.New("failed to create new request")
	ErrRequestPerform     = errors.New("could not perform http request")
	ErrRequestInvalidCode = errors.New("invalid response code returned from request")
	ErrRequestDecode      = errors.New("failed to decode http response")
	ErrResponseBody       = errors.New("failed to read response body")
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
