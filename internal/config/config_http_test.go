package config_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/tests"
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
		curConf       = fixture.Config.Config()
		authenticator = &tests.UserAuth{Profile: fixture.CreateTestPerson(t.Context(), tests.OwnerSID, permission.Admin)}
		conf          = config.NewConfiguration(curConf.Static, config.NewRepository(fixture.Database))
		router        = fixture.CreateRouter()
	)

	config.NewHandler(router, conf, authenticator, "1.2.3")

	saved := tests.PutGOK[config.Config](t, router, "/api/config", curConf)
	require.Equal(t, curConf, saved)

	info := tests.GetGOK[config.AppInfo](t, router, "/api/info")
	require.NotEmpty(t, info.SiteName)

	release := tests.GetGOK[config.GithubRelease](t, router, "/app/changelog")
	require.NotEmpty(t, release.URL)

	conf1 := tests.GetGOK[config.Config](t, router, "/api/config")
	require.Equal(t, fixture.Config.Config(), conf1)
}
