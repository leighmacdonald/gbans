package stringutil

import (
	"strconv"
)

// StringToFloat64Default converts a string to a float64, returning a default values on
// conversion error.
func StringToFloat64Default(numericString string, defaultValue float64) float64 {
	value, errParseFloat := strconv.ParseFloat(numericString, 64)
	if errParseFloat != nil {
		return defaultValue
	}

	return value
}

const defaultIntAllocate int = 0

// StringToIntOrZero handles convering a string to a integer that is within 32bit bounds.
// Returns 0 on a out of bounds value.
func StringToIntOrZero(desired string) int {
	parsed, err := strconv.ParseInt(desired, 10, 32)
	if err != nil {
		return defaultIntAllocate
	}

	return int(parsed)
}
