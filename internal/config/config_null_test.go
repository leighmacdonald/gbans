package config_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/stretchr/testify/require"
)

func TestConfigReadNullSentryDSN(t *testing.T) {
	fixture := tests.NewFixture()
	defer fixture.Close()
	ctx := t.Context()

	repo := config.NewRepository(fixture.Database)
	require.NoError(t, repo.Init(ctx))
	require.NoError(t, fixture.Database.Exec(ctx, `ALTER TABLE config ALTER COLUMN general_sentry_dsn DROP NOT NULL`))
	require.NoError(t, fixture.Database.Exec(ctx, `ALTER TABLE config ALTER COLUMN general_sentry_dsn_web DROP NOT NULL`))
	require.NoError(t, fixture.Database.Exec(ctx, `UPDATE config SET general_sentry_dsn = NULL, general_sentry_dsn_web = NULL`))

	conf, errRead := repo.Read(ctx)
	require.NoError(t, errRead)
	require.Empty(t, conf.General.SentryDSN)
	require.Empty(t, conf.General.SentryDSNWeb)
}
