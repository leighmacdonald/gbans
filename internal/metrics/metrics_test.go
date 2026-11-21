package metrics_test

import (
	"context"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/metrics"
	"github.com/leighmacdonald/gbans/pkg/broadcaster"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func newTestMetrics(t *testing.T) metrics.Metrics {
	t.Helper()

	metricsSvc, errMetrics := metrics.New(
		broadcaster.New[logparse.EventType, logparse.ServerEvent](), "test-version",
	)
	require.NoError(t, errMetrics)

	return metricsSvc
}

func family(t *testing.T, registry *prometheus.Registry, name string) *dto.MetricFamily {
	t.Helper()

	families, errGather := registry.Gather()
	require.NoError(t, errGather)

	for _, candidate := range families {
		if candidate.GetName() == name {
			return candidate
		}
	}

	t.Fatalf("metric family %q not found", name)

	return nil
}

func TestNewRegistersGbansMetrics(t *testing.T) {
	metricsSvc := newTestMetrics(t)

	build := family(t, metricsSvc.Registry(), "gbans_build_info")
	require.Len(t, build.GetMetric(), 1)
	require.InDelta(t, 1, build.GetMetric()[0].GetGauge().GetValue(), 0.0001)
	require.Equal(t, "test-version", build.GetMetric()[0].GetLabel()[0].GetValue())

	for _, name := range []string{
		"gbans_game_log_events_total",
		"gbans_servers_total",
		"gbans_players_total",
	} {
		require.NotNil(t, family(t, metricsSvc.Registry(), name), name)
	}

	// No provider yet, but the gauges must still be present at zero.
	servers := family(t, metricsSvc.Registry(), "gbans_servers_total")
	require.InDelta(t, 0, servers.GetMetric()[0].GetGauge().GetValue(), 0.0001)
}

func TestServerProviderGauges(t *testing.T) {
	metricsSvc := newTestMetrics(t)
	metricsSvc.SetServerProvider(func() []metrics.ServerSnapshot {
		return []metrics.ServerSnapshot{
			{Name: "us-1", Humans: 3},
			{Name: "eu-1", Humans: 5},
		}
	})

	servers := family(t, metricsSvc.Registry(), "gbans_servers_total")
	require.InDelta(t, 2, servers.GetMetric()[0].GetGauge().GetValue(), 0.0001)

	players := family(t, metricsSvc.Registry(), "gbans_players_total")
	require.InDelta(t, 8, players.GetMetric()[0].GetGauge().GetValue(), 0.0001)

	perServer := family(t, metricsSvc.Registry(), "gbans_server_players")
	require.Len(t, perServer.GetMetric(), 2)
}

func TestObserveRequest(t *testing.T) {
	metricsSvc := newTestMetrics(t)
	metricsSvc.ObserveRequest("GET", "GET /health", 200, 10*time.Millisecond)

	requests := family(t, metricsSvc.Registry(), "gbans_http_requests_total")
	require.Len(t, requests.GetMetric(), 1)
	require.InDelta(t, 1, requests.GetMetric()[0].GetCounter().GetValue(), 0.0001)

	durations := family(t, metricsSvc.Registry(), "gbans_http_request_duration_seconds")
	require.EqualValues(t, 1, durations.GetMetric()[0].GetHistogram().GetSampleCount())
}

func TestStartUpdatesEventCounters(t *testing.T) {
	eventBus := broadcaster.New[logparse.EventType, logparse.ServerEvent]()
	metricsSvc, errMetrics := metrics.New(eventBus, "test-version")
	require.NoError(t, errMetrics)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go metricsSvc.Start(ctx)

	require.Eventually(t, func() bool {
		// Retry in case the event is emitted before Start registers its consumer.
		eventBus.Emit(logparse.Say, logparse.ServerEvent{ServerName: "us-1", EventType: logparse.Say})

		events := family(t, metricsSvc.Registry(), "gbans_game_log_events_total")

		return events.GetMetric()[0].GetCounter().GetValue() >= 1
	}, 2*time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		chat := family(t, metricsSvc.Registry(), "gbans_game_chat_total")

		return len(chat.GetMetric()) > 0 && chat.GetMetric()[0].GetCounter().GetValue() >= 1
	}, 2*time.Second, 10*time.Millisecond)
}
