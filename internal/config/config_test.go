package config_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/stretchr/testify/require"
)

func TestInternalAddr(t *testing.T) {
	static := config.Static{InternalHost: "10.0.0.5", InternalPort: 6605}
	require.Equal(t, "10.0.0.5:6605", static.InternalAddr())

	static.InternalHost = "::1"
	require.Equal(t, "[::1]:6605", static.InternalAddr())
}

func TestConfig(t *testing.T) {
	fixture := tests.NewFixture()
	defer fixture.Close()

	configuration, errConfig := config.NewConfiguration(t.Context(), fixture.Config.Config().Static, config.NewRepository(fixture.Database))
	require.NoError(t, errConfig)
	conf := configuration.Config()
	conf.General.SiteName += "x"
	require.NoError(t, configuration.Write(t.Context(), conf))
	updated := configuration.Config()
	require.EqualExportedValues(t, conf, updated)
}

func TestStatsRefreshScheduleValidation(t *testing.T) {
	fixture := tests.NewFixture()
	defer fixture.Close()

	configuration, errConfig := config.NewConfiguration(t.Context(), fixture.Config.Config().Static, config.NewRepository(fixture.Database))
	require.NoError(t, errConfig)

	t.Run("invalid schedule is rejected", func(t *testing.T) {
		conf := configuration.Config()
		conf.Stats = &config.Stats{RefreshSchedule: "not a cron"}

		require.ErrorIs(t, configuration.Write(t.Context(), conf), config.ErrInvalidConfig)
	})

	t.Run("empty schedule is rejected", func(t *testing.T) {
		conf := configuration.Config()
		conf.Stats = &config.Stats{RefreshSchedule: "  "}

		require.ErrorIs(t, configuration.Write(t.Context(), conf), config.ErrInvalidConfig)
	})

	t.Run("valid schedule is saved", func(t *testing.T) {
		conf := configuration.Config()
		conf.Stats = &config.Stats{RefreshSchedule: "15 3 * * *"}

		require.NoError(t, configuration.Write(t.Context(), conf))
		require.Equal(t, "15 3 * * *", configuration.Config().StatsRefreshSchedule())
	})
}
