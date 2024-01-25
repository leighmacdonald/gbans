package domain

import "errors"

var (
	ErrRowResults       = errors.New("resulting rows contain error")
	ErrTxStart          = errors.New("could not start transaction")
	ErrTxCommit         = errors.New("failed to commit tx changes")
	ErrTxRollback       = errors.New("could not rollback transaction")
	ErrPoolFailed       = errors.New("could not create store pool")
	ErrUUIDGen          = errors.New("could not generate uuid")
	ErrCreateQuery      = errors.New("failed to generate query")
	ErrCountQuery       = errors.New("failed to get count result")
	ErrTooShort         = errors.New("value too short")
	ErrInvalidParameter = errors.New("invalid parameter format")
	ErrPermissionDenied = errors.New("permission denied")
	ErrBadRequest       = errors.New("invalid request")
	ErrInternal         = errors.New("internal server error")
	ErrParamKeyMissing  = errors.New("param key not found")
	ErrParamParse       = errors.New("failed to parse param value")
	ErrParamInvalid     = errors.New("param value invalid")
	ErrScanResult       = errors.New("failed to scan result")
)
