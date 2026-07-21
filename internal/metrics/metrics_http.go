package metrics

import "net/http"

type metricsHandler struct{}

func NewMetricsHandler(mux *http.ServeMux) {
	_ = metricsHandler{}
}
