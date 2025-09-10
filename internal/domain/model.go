package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalidBanType     = errors.New("invalid ban type")
	ErrInvalidBanDuration = errors.New("invalid ban duration")
	ErrInvalidBanReason   = errors.New("custom reason cannot be empty")
	ErrInvalidReportID    = errors.New("invalid report ID")
	ErrInvalidASN         = errors.New("invalid asn, out of range")
	ErrInvalidCIDR        = errors.New("failed to parse CIDR address")
)

func NewTimeStamped() TimeStamped {
	now := time.Now()

	return TimeStamped{
		CreatedOn: now,
		UpdatedOn: now,
	}
}

type TimeStamped struct {
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}
