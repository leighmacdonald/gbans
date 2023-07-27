package app

import (
	"fmt"
	"strings"
	"time"
)

const (
	expirationYears = 25
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
