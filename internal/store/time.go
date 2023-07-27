package store

import (
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

var (
	reDuration         = regexp.MustCompile(`^(\d+)([smhdwMy])$`)
	errInvalidDuration = errors.New("Invalid duration")
)

// ParseDuration works exactly like time.ParseDuration except that
// it supports durations longer than hours
// Formats: s, m, h, d, w, M, y.
func ParseDuration(durationString string) (time.Duration, error) {
	if durationString == "0" {
		return 0, nil
	}

	matchDuration := reDuration.FindStringSubmatch(durationString)
	if matchDuration == nil {
		return 0, errInvalidDuration
	}

	valueInt, errParseInt := strconv.ParseInt(matchDuration[1], 10, 64)
	if errParseInt != nil {
		return 0, errInvalidDuration
	}

	var (
		value = time.Duration(valueInt)
		day   = time.Hour * 24
	)

	switch matchDuration[2] {
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
