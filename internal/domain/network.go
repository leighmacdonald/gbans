package domain

import "errors"

var (
	ErrFailedToList   = errors.New("failed to list files")
	ErrFailedOpenFile = errors.New("failed to open file")
	ErrFailedReadFile = errors.New("failed to read file")
	ErrCloseReader    = errors.New("failed to close file reader")
)
