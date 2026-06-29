package config_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/stretchr/testify/require"
)

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
