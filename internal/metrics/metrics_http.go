package metrics

import "net/http"

type metricsHandler struct{}

func NewMetricsHandler(_ *http.ServeMux) {
	_ = metricsHandler{}
}
