package util

import (
	log "github.com/sirupsen/logrus"
	"strconv"
)

func StringToFloat64(s string, def float64) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Warnf("failed to parse float64 value: %s", s)
		return def
	}
	return v
}
