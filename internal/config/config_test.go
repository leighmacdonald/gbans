package config_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	db := tests.NewFixture()
	defer db.Close()

	configuration := config.NewConfiguration(db.Config.Static, config.NewRepository(db.Database))
	require.NoError(t, configuration.Init(t.Context()))
	require.NoError(t, configuration.Reload(t.Context()))
	conf := configuration.Config()
	conf.General.SiteName += "x"
	require.NoError(t, configuration.Write(t.Context(), conf))
	updated := configuration.Config()
	require.EqualExportedValues(t, conf, updated)
}
