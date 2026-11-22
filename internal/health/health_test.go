//nolint:testpackage // exercises unexported health check helpers directly.
package health

import (
	"errors"
	"testing"
	"time"

	v1 "github.com/leighmacdonald/gbans/internal/health/v1"
	"github.com/stretchr/testify/require"
)

var errTestSSH = errors.New("connection refused")

func TestTrackerSnapshot(t *testing.T) {
	tracker := NewTracker()
	tracker.RecordRCON(7, nil)
	tracker.RecordSSH("10.0.0.7", errTestSSH)
	tracker.Record("stats_refresh", nil)

	snapshot := tracker.Snapshot()
	require.Contains(t, snapshot.RCON, int32(7))
	require.Empty(t, snapshot.RCON[7].Err)
	require.Contains(t, snapshot.SSH, "10.0.0.7")
	require.Equal(t, "connection refused", snapshot.SSH["10.0.0.7"].Err)
	require.Contains(t, snapshot.Events, "stats_refresh")
	require.False(t, snapshot.Started.IsZero())
}

func TestRCONCheck(t *testing.T) {
	service := Service{}

	warn := service.rconCheck(Snapshot{RCON: map[int32]Event{}})
	require.Equal(t, v1.HealthStatus_HEALTH_STATUS_WARN, warn.status)

	ok := service.rconCheck(Snapshot{RCON: map[int32]Event{1: {At: time.Now()}}})
	require.Equal(t, v1.HealthStatus_HEALTH_STATUS_OK, ok.status)

	withFailure := service.rconCheck(Snapshot{RCON: map[int32]Event{
		1: {At: time.Now()},
		2: {At: time.Now(), Err: "timeout"},
	}})
	require.Equal(t, v1.HealthStatus_HEALTH_STATUS_WARN, withFailure.status)
	require.Contains(t, withFailure.detail, "1 failed")
}

func TestEventCheck(t *testing.T) {
	missing := eventCheck("worker", Event{}, time.Hour)
	require.Equal(t, v1.HealthStatus_HEALTH_STATUS_WARN, missing.status)

	failed := eventCheck("worker", Event{At: time.Now(), Err: "boom"}, time.Hour)
	require.Equal(t, v1.HealthStatus_HEALTH_STATUS_ERROR, failed.status)

	stale := eventCheck("worker", Event{At: time.Now().Add(-2 * time.Hour)}, time.Hour)
	require.Equal(t, v1.HealthStatus_HEALTH_STATUS_WARN, stale.status)

	healthy := eventCheck("worker", Event{At: time.Now()}, time.Hour)
	require.Equal(t, v1.HealthStatus_HEALTH_STATUS_OK, healthy.status)
}

func TestDiskCheck(t *testing.T) {
	disabled := diskCheck("Disk", "")
	require.Equal(t, v1.HealthStatus_HEALTH_STATUS_DISABLED, disabled.status)

	missing := diskCheck("Disk", "/this/path/does/not/exist/gbans")
	require.Equal(t, v1.HealthStatus_HEALTH_STATUS_ERROR, missing.status)

	valid := diskCheck("Disk", t.TempDir())
	require.Equal(t, v1.HealthStatus_HEALTH_STATUS_OK, valid.status)
}
