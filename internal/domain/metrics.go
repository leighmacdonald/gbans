package domain

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

type MetricsUsecase interface {
	Start(ctx context.Context)
}

type MetricCollector struct {
	LogEventCounter     *prometheus.CounterVec
	SayCounter          *prometheus.CounterVec
	DamageCounter       *prometheus.CounterVec
	HealingCounter      *prometheus.CounterVec
	KillCounter         *prometheus.CounterVec
	ShotFiredCounter    *prometheus.CounterVec
	ShotHitCounter      *prometheus.CounterVec
	MapCounter          *prometheus.CounterVec
	RconCounter         *prometheus.CounterVec
	ConnectedCounter    *prometheus.CounterVec
	DisconnectedCounter *prometheus.CounterVec
	ClassCounter        *prometheus.CounterVec
	PlayerCounter       *prometheus.HistogramVec
}
