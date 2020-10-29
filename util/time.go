package util

import (
	"github.com/pkg/errors"
	"regexp"
	"strconv"
	"time"
)

var (
	reDuration *regexp.Regexp

	errInvalidDuration = errors.New("Invalid duration")
)

func init() {
	reDuration = regexp.MustCompile(`^(\d+)([smhdwMy])$`)
}

// ParseDuration works exactly like time.ParseDuration except that
// it supports durations longer than hours
// Formats: s, m, h, d, w, M, y
func ParseDuration(s string) (time.Duration, error) {
	if s == "0" {
		return 0, nil
	}
	m := reDuration.FindStringSubmatch(s)
	if m == nil {
		return 0, errInvalidDuration
	}
	valueInt, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0, errInvalidDuration
	}
	value := time.Duration(valueInt)
	day := time.Hour * 24
	switch m[2] {
	case "s":
		return time.Second * value, nil
	case "m":
		return time.Minute * value, nil
	case "h":
		return time.Hour * value, nil
	case "d":
		return day * value, nil
	case "w":
		return day * 7 * value, nil
	case "M":
		return day * 31 * value, nil
	case "y":
		return day * 365 * value, nil
	}
	return 0, errInvalidDuration
}
