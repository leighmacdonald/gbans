package app

import (
	"fmt"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/pkg/errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reDuration         = regexp.MustCompile(`^(\d+)([smhdwMy])$`)
	errInvalidDuration = errors.New("Invalid duration")
)

// FmtTimeShort returns a common format for time display.
func FmtTimeShort(t time.Time) string {
	return t.Format("Mon Jan 2 15:04:05 MST 2006")
}

// FmtDuration calculates and returns a string for duration differences. This handles
// values larger than a day unlike the stdlib in functionalities.
func FmtDuration(t time.Time) string {
	year, month, day, hour, minute, _ := diff(t, time.Now())

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
	var year, month, day, hour, minute, sec int

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
	minute = minute2 - minute1
	sec = second2 - second1

	// Normalize negative values
	if sec < 0 {
		sec += 60
		minute--
	}

	if minute < 0 {
		minute += 60
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

	return year, month, day, hour, minute, sec
}

// ParseUserStringDuration works exactly like time.ParseDuration except that
// it supports durations longer than hours
// Formats: s, m, h, d, w, M, y.
func ParseUserStringDuration(durationString string) (time.Duration, error) {
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

func ParseDuration(value string) (time.Duration, error) {
	duration, errDuration := ParseUserStringDuration(value)
	if errDuration != nil {
		return 0, consts.ErrInvalidDuration
	}

	if duration < 0 {
		return 0, consts.ErrInvalidDuration
	}

	if duration == 0 {
		duration = time.Hour * 24 * 365 * 10
	}

	return duration, nil
}

func calcDuration(durationString string, validUntil time.Time) (time.Duration, error) {
	if durationString == "custom" {
		dur := time.Until(validUntil)
		if dur < 0 {
			return 0, errInvalidDuration
		}

		return dur, nil
	} else {
		dur, errDuration := ParseDuration(durationString)
		if errDuration != nil {
			return 0, errDuration
		}
		return dur, nil
	}
}
