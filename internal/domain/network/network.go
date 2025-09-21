package network

import "errors"

var (
	ErrOpenClient     = errors.New("failed to open client")
	ErrFailedToList   = errors.New("failed to list files")
	ErrFailedOpenFile = errors.New("failed to open file")
	ErrFailedReadFile = errors.New("failed to read file")
	ErrCloseReader    = errors.New("failed to close file reader")
)
