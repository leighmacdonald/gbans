package config

import (
	"fmt"
	"github.com/pkg/errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	expirationYears = 25
)

var (
	reDuration         *regexp.Regexp
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

// Now returns the current time in the configured format of the application runtime
//
// All calls to time.Now() should use this instead to ensure consistency
func Now() time.Time {
	if General.UseUTC {
		return time.Now().UTC()
	}
	return time.Now()
}

// DefaultExpiration returns the default expiration time delta from Now()
func DefaultExpiration() time.Time {
	return Now().AddDate(expirationYears, 0, 0)
}

// FmtTimeShort returns a common format for time display
func FmtTimeShort(t time.Time) string {
	return t.Format("Mon Jan 2 15:04:05 MST 2006")
}

// FmtDuration calculates and returns a string for duration differences. This handles
// values larger than a day unlike the stdlib in functionalities
func FmtDuration(t time.Time) string {
	year, month, day, hour, min, _ := diff(t, Now())
	var pcs []string
	if year > 0 {
		pcs = append(pcs, fmt.Sprintf("%dy", year))
	}
	if month > 0 {
		pcs = append(pcs, fmt.Sprintf("%dM", month))
	}
	if day > 0 {
		pcs = append(pcs, fmt.Sprintf("%dd", day))
	}
	if hour > 0 {
		pcs = append(pcs, fmt.Sprintf("%dh", hour))
	}
	if min > 0 {
		pcs = append(pcs, fmt.Sprintf("%dm", min))
	}
	return strings.Join(pcs, " ")
}

func diff(a, b time.Time) (year, month, day, hour, min, sec int) {
	if a.Location() != b.Location() {
		b = b.In(a.Location())
	}
	if a.After(b) {
		a, b = b, a
	}
	y1, M1, d1 := a.Date()
	y2, M2, d2 := b.Date()

	h1, m1, s1 := a.Clock()
	h2, m2, s2 := b.Clock()

	year = y2 - y1
	month = int(M2 - M1)
	day = d2 - d1
	hour = h2 - h1
	min = m2 - m1
	sec = s2 - s1

	// Normalize negative values
	if sec < 0 {
		sec += 60
		min--
	}
	if min < 0 {
		min += 60
		hour--
	}
	if hour < 0 {
		hour += 24
		day--
	}
	if day < 0 {
		// days in month:
		t := time.Date(y1, M1, 32, 0, 0, 0, 0, time.UTC)
		day += 32 - t.Day()
		month--
	}
	if month < 0 {
		month += 12
		year--
	}

	return
}
