package util

import (
	log "github.com/sirupsen/logrus"
	"strconv"
)

// StringToFloat64 converts a string to a float64, returning a default values on
// conversion error
func StringToFloat64(numericString string, defaultValue float64) float64 {
	value, errParseFloat := strconv.ParseFloat(numericString, 64)
	if errParseFloat != nil {
		log.Warnf("failed to parse float64 value: %s", numericString)
		return defaultValue
	}
	return value
}
