package config_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/anticheat"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}

func TestConfigHTTP(t *testing.T) {
	var (
		authenticator = &tests.UserAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)}
		router        = fixture.CreateRouter()
		conf          = config.NewConfiguration(fixture.Config.Config().Static, config.NewRepository(fixture.Database))
	)

	require.NoError(t, conf.Init(t.Context()))
	config.NewHandler(router, authenticator, conf, "1.2.3")

	curConf := conf.Config()
	curConf.General.SiteName = stringutil.SecureRandomString(10)
	curConf.Anticheat.Action = anticheat.ActionBan
	curConf.Demo.DemoCleanupStrategy = servers.DemoStrategyCount
	curConf.General.FileServeMode = config.LocalMode
	curConf.General.Mode = config.TestMode
	curConf.Log.Level = log.Warn

	saved := tests.PutGOK[config.Config](t, router, "/api/config", curConf)
	require.Equal(t, curConf, saved)

	info := tests.GetGOK[config.AppInfo](t, router, "/api/info")
	require.NotEmpty(t, info.SiteName)

	release := tests.GetGOK[[]config.GithubRelease](t, router, "/api/changelog")
	require.NotEmpty(t, release)

	conf1 := tests.GetGOK[config.Config](t, router, "/api/config")
	require.Equal(t, curConf, conf1)
}
