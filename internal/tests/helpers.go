package tests

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/demo"
	"github.com/leighmacdonald/gbans/internal/discord"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/fs"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/internal/network/ip2location"
	"github.com/leighmacdonald/gbans/internal/network/scp"
	"github.com/leighmacdonald/gbans/internal/patreon"
	"github.com/leighmacdonald/gbans/internal/sourcemod"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	OwnerSID = steamid.New(76561198084134025) //nolint:gochecknoglobals
	ModSID   = steamid.New(76561198084134026) //nolint:gochecknoglobals
	UserSID  = steamid.New(76561198084134027) //nolint:gochecknoglobals
	GuestSID = steamid.New(76561198084134028) //nolint:gochecknoglobals

	ErrContainer = errors.New("failed to bring up test container")
)

type ServerAuth struct {
	ServerID int
}

func (s *ServerAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*r = *r.WithContext(context.WithValue(r.Context(), "server_id", s.ServerID))
		next.ServeHTTP(w, r)
	})
}

type UserAuth struct {
	Profile personDomain.Core
}

func (s *UserAuth) Middleware(level permission.Privilege) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if level > s.Profile.PermissionLevel {
				w.WriteHeader(http.StatusForbidden)

				return
			}
			*r = *r.WithContext(context.WithValue(r.Context(), auth.CtxKeyUserProfile, s.Profile))
			next.ServeHTTP(w, r)
		})
	}
}

func (s *UserAuth) MiddlewareWS(level permission.Privilege) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if level > s.Profile.PermissionLevel {
				w.WriteHeader(http.StatusForbidden)

				return
			}
			*r = *r.WithContext(context.WithValue(r.Context(), auth.CtxKeyUserProfile, s.Profile))
			next.ServeHTTP(w, r)
		})
	}
}

type postgresContainer struct {
	testcontainers.Container

	dbName   string
	user     string
	password string
	dsn      string
}

func newDB(ctx context.Context) (*postgresContainer, error) {
	const testInfo = "gbans-test"
	username, password, dbName := testInfo, testInfo, testInfo

	dockerRoot := fs.FindFile("docker", "gbans")

	fromDockerfile := testcontainers.FromDockerfile{
		Dockerfile:    "postgres-ip4r.Dockerfile",
		Context:       dockerRoot,
		KeepImage:     true,
		PrintBuildLog: true,
	}

	cont, errContainer := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: fromDockerfile,

			Env: map[string]string{
				"POSTGRES_DB":       dbName,
				"POSTGRES_USER":     username,
				"POSTGRES_PASSWORD": password,
			},
			AlwaysPullImage: false,
			WaitingFor: wait.
				ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		},
		Started: true,
	})

	if errContainer != nil {
		return nil, errors.Join(errContainer, ErrContainer)
	}

	port, _ := cont.MappedPort(ctx, "5432")
	dsn := fmt.Sprintf("postgresql://%s:%s@localhost:%s/%s", username, password, port.Port(), dbName)

	pgContainer := postgresContainer{
		Container: cont,
		dbName:    dbName,
		user:      username,
		password:  password,
		dsn:       dsn,
	}

	return &pgContainer, nil
}

type TestConfigRepo struct {
	config config.Config
}

func (c *TestConfigRepo) Read(_ context.Context) (config.Config, error) {
	return c.config, nil
}

func (c *TestConfigRepo) Write(_ context.Context, config config.Config) error {
	c.config = config

	return nil
}

func (c *TestConfigRepo) Config() config.Config {
	return c.config
}

func (c *TestConfigRepo) Init(_ context.Context) error {
	return nil
}

func TestConfig(ctx context.Context, dsn string) (*config.Configuration, error) {
	return config.NewConfiguration(ctx, config.Static{}, config.NewMemConfigRepository(config.Config{
		Static: config.Static{
			Owner:               OwnerSID.String(),
			ExternalURL:         "http://example.com",
			HTTPHost:            "localhost",
			HTTPPort:            6006,
			HTTPStaticPath:      "",
			HTTPCookieKey:       stringutil.SecureRandomString(10),
			HTTPClientTimeout:   10,
			HTTPCORSEnabled:     false,
			HTTPCorsOrigins:     nil,
			DatabaseDSN:         dsn,
			DatabaseAutoMigrate: true,
			DatabaseLogQueries:  false,
			PrometheusEnabled:   false,
			PProfEnabled:        false,
		},
		General: &config.General{
			SiteName:           "gbans",
			Mode:               config.TestMode,
			FileServeMode:      "local",
			SrcdsLogAddr:       "",
			AssetURL:           "",
			DefaultRoute:       "",
			NewsEnabled:        true,
			ForumsEnabled:      true,
			ContestsEnabled:    true,
			WikiEnabled:        true,
			StatsEnabled:       true,
			ServersEnabled:     true,
			ReportsEnabled:     true,
			ChatlogsEnabled:    true,
			DemosEnabled:       true,
			PlayerqueueEnabled: true,
			SpeedrunsEnabled:   true,
		},
		Demo: &demo.Config{
			DemoCleanupEnabled:  false,
			DemoCleanupStrategy: "",
			DemoCleanupMinPct:   0,
			DemoCleanupMount:    "",
			DemoCountLimit:      2,
		},
		Filters: &chat.Config{
			Enabled:        true,
			WarningTimeout: 10,
			WarningLimit:   1,
			Dry:            false,
			PingDiscord:    false,
			MaxWeight:      1,
			CheckTimeout:   10,
			MatchTimeout:   10,
		},
		Discord: &discord.Config{
			Enabled: false,
		},
		Clientprefs: &sourcemod.Config{},
		Log: &log.Config{
			HTTPEnabled: false,
			Level:       "error",
		},
		GeoLocation: &ip2location.Config{
			Enabled: false,
		},
		Debug: &config.Debug{},
		Patreon: &patreon.Config{
			Enabled: false,
		},
		SSH: &scp.Config{
			Enabled: false,
		},
		LocalStore: &asset.Config{},
		Exports:    &ban.Config{},
	}))
}
