package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	expirationYears = 25
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

// Now returns the current time in the configured format of the application runtime
//
// All calls to time.Now() should use this instead to ensure consistency.
func Now() time.Time {
	return time.Now().UTC()
}

// DefaultExpiration returns the default expiration time delta from Now().
func DefaultExpiration() time.Time {
	return Now().AddDate(expirationYears, 0, 0)
}

// FmtTimeShort returns a common format for time display.
func FmtTimeShort(t time.Time) string {
	return t.Format("Mon Jan 2 15:04:05 MST 2006")
}

// FmtDuration calculates and returns a string for duration differences. This handles
// values larger than a day unlike the stdlib in functionalities.
func FmtDuration(t time.Time) string {
	year, month, day, hour, minute, _ := diff(t, Now())

	var pieces []string

	if year > 0 {
		pieces = append(pieces, fmt.Sprintf("%dy", year))
	}

	if month > 0 {
		pieces = append(pieces, fmt.Sprintf("%dM", month))
	}

	if day > 0 {
		pieces = append(pieces, fmt.Sprintf("%dd", day))
	}

	if hour > 0 {
		pieces = append(pieces, fmt.Sprintf("%dh", hour))
	}

	if minute > 0 {
		pieces = append(pieces, fmt.Sprintf("%dm", minute))
	}

	if len(pieces) == 0 {
		return "~now"
	}

	return strings.Join(pieces, " ")
}

func diff(timeFrom time.Time, timeTo time.Time) (int, int, int, int, int, int) {
	var year, month, day, hour, min, sec int

	if timeFrom.Location() != timeTo.Location() {
		timeTo = timeTo.In(timeFrom.Location())
	}

	if timeFrom.After(timeTo) {
		timeFrom, timeTo = timeTo, timeFrom
	}

	year1, Month1, day1 := timeFrom.Date()
	year2, Month2, day2 := timeTo.Date()

	hour1, minute1, second1 := timeFrom.Clock()
	hour2, minute2, second2 := timeTo.Clock()

	year = year2 - year1
	month = int(Month2 - Month1)
	day = day2 - day1
	hour = hour2 - hour1
	min = minute2 - minute1
	sec = second2 - second1

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
		t := time.Date(year1, Month1, 32, 0, 0, 0, 0, time.UTC)
		day += 32 - t.Day()
		month--
	}

	if month < 0 {
		month += 12
		year--
	}

	return year, month, day, hour, min, sec
}
