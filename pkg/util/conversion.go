package util

import (
	"math"
	"strconv"
)

// StringToFloat64 converts a string to a float64, returning a default values on
// conversion error.
func StringToFloat64(numericString string, defaultValue float64) float64 {
	value, errParseFloat := strconv.ParseFloat(numericString, 64)
	if errParseFloat != nil {
		return defaultValue
	}

	return value
}

const DefaultIntAllocate int = 0

func StringToInt(desired string) int {
	parsed, err := strconv.Atoi(desired)
	if err != nil {
		return DefaultIntAllocate
	}

	if parsed > 0 && parsed <= math.MaxInt32 {
		return parsed
	}

	return DefaultIntAllocate
}
