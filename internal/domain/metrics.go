package domain

import (
	"context"

	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type MetricsUsecase interface {
	LogMetricsConsumer(ctx context.Context, collector *MetricCollector, eb *fp.Broadcaster[logparse.EventType, logparse.ServerEvent], logger *zap.Logger)
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
