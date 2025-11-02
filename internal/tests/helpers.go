package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/gin-gonic/gin"
	"github.com/google/go-querystring/query"
	"github.com/leighmacdonald/gbans/internal/auth"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/fs"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
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

type StaticAuthenticator struct {
	Profile personDomain.Core
}

func (s *StaticAuthenticator) Middleware(level permission.Privilege) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if level > s.Profile.PermissionLevel {
			ctx.AbortWithStatus(http.StatusForbidden)

			return
		}
		ctx.Set(auth.CtxKeyUserProfile, s.Profile)
	}
}

func (s *StaticAuthenticator) MiddlewareWS(level permission.Privilege) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if level > s.Profile.PermissionLevel {
			ctx.AbortWithStatus(http.StatusForbidden)

			return
		}
		ctx.Set(auth.CtxKeyUserProfile, s.Profile)
	}
}

// postgresContainer is used instead of the postgres.PostgresContainer one since
// we need to build our custom image with extra extensions.
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

	// Naively look for the docker directory. Assumes the project root directory is named "gbans"
	dockerRoot := fs.FindFile("docker", "gbans")

	fromDockerfile := testcontainers.FromDockerfile{
		Dockerfile: "postgres-ip4r.Dockerfile",

		Context:       dockerRoot,
		PrintBuildLog: false,
		KeepImage:     true,
	}

	cont, errContainer := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: fromDockerfile,

			HostConfigModifier: func(config *container.HostConfig) {
				config.AutoRemove = false
			},
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

// type permTestValues struct {
// 	method string
// 	code   int
// 	path   string
// 	levels []permission.Privilege
// }

type TestConfigRepo struct {
	config config.Config
}

// Read implements config.ConfigRepo.
func (c *TestConfigRepo) Read(_ context.Context) (config.Config, error) {
	return c.config, nil
}

// Write implements config.ConfigRepo.
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

type Fixture struct {
	container *postgresContainer
	Database  database.Database
	Config    *config.Configuration
	Persons   personDomain.Provider
	TFApi     *thirdparty.TFAPI
	DSN       string
	Close     func()
}

func NewFixture() *Fixture {
	testCtx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	testDB, errStore := newDB(testCtx)
	if errStore != nil {
		panic(errStore)
	}

	databaseConn := database.New(testDB.dsn, true, false)
	if err := databaseConn.Connect(testCtx); err != nil {
		panic(err)
	}

	api, _ := thirdparty.NewTFAPI("https://tf-api.roto.lol", &http.Client{Timeout: time.Second * 15})

	conf := TestConfig(testDB.dsn)

	return &Fixture{
		container: testDB,
		Database:  databaseConn,
		TFApi:     api,
		Config:    conf,
		DSN:       testDB.dsn,
		Persons:   person.NewPersons(person.NewRepository(conf.Config(), databaseConn), steamid.New(conf.Config().Owner), nil),
		Close: func() {
			termCtx, termCancel := context.WithTimeout(context.Background(), time.Second*30)
			defer termCancel()

			if errTerm := testDB.Terminate(termCtx); errTerm != nil {
				panic(fmt.Sprintf("Failed to terminate test container: %v", errTerm))
			}
		},
	}
}

func (f *Fixture) CreateRouter() *gin.Engine {
	router, err := httphelper.CreateRouter(httphelper.RouterOpts{LogLevel: log.Error})
	if err != nil {
		panic(err)
	}

	return router
}

func (f *Fixture) Reset(ctx context.Context) {
	const query = `DO
$do$
BEGIN
   EXECUTE
   (SELECT 'TRUNCATE TABLE ' || string_agg(oid::regclass::text, ', ') || ' CASCADE'
    FROM   pg_class
    WHERE  relkind = 'r'
    AND    relnamespace = 'public'::regnamespace
   );
END
$do$;`

	if err := f.Database.Exec(ctx, query); err != nil {
		panic(err)
	}

	if err := f.Database.Migrate(ctx, database.MigrateUp, f.DSN); err != nil {
		panic(err)
	}
}

func EndpointReceiver(t *testing.T, router http.Handler, method string,
	path string, body any, expectedStatus int, tokens *AuthTokens, receiver any,
) {
	t.Helper()

	resp := Endpoint(t, router, method, path, body, expectedStatus, tokens)
	if receiver != nil {
		if err := json.NewDecoder(resp.Body).Decode(&receiver); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
	}
}

type AuthTokens struct {
	user           *auth.UserTokens
	serverPassword string
}

func Endpoint(t *testing.T, router http.Handler, method string, path string, body any, expectedStatus int, tokens *AuthTokens) *httptest.ResponseRecorder {
	t.Helper()

	reqCtx, cancel := context.WithTimeout(t.Context(), time.Second*10)
	defer cancel()

	recorder := httptest.NewRecorder()

	var bodyReader io.Reader
	if body != nil {
		bodyJSON, errJSON := json.Marshal(body)
		if errJSON != nil {
			t.Fatalf("Failed to encode request: %v", errJSON)
		}

		bodyReader = bytes.NewReader(bodyJSON)
	}

	if body != nil && method == http.MethodGet {
		values, err := query.Values(body)
		if err != nil {
			t.Fatalf("failed to encode values: %v", err)
		}

		path += "?" + values.Encode()
	}

	request, errRequest := http.NewRequestWithContext(reqCtx, method, path, bodyReader)
	if errRequest != nil {
		t.Fatalf("Failed to make request: %v", errRequest)
	}

	if tokens != nil {
		if tokens.serverPassword != "" {
			request.Header.Add("Authorization", tokens.serverPassword)
		} else if tokens.user != nil {
			request.AddCookie(&http.Cookie{
				Name:     auth.FingerprintCookieName,
				Value:    tokens.user.Fingerprint,
				Path:     "/api",
				Domain:   "example.com",
				Expires:  time.Now().AddDate(0, 0, 1),
				MaxAge:   0,
				Secure:   false,
				HttpOnly: false,
				SameSite: http.SameSiteStrictMode,
			})
			request.Header.Add("Authorization", "Bearer "+tokens.user.Access)
		}
	}

	router.ServeHTTP(recorder, request)

	require.Equal(t, expectedStatus, recorder.Code, "Received invalid response code. method: %s path: %s", method, path)

	return recorder
}

func (f Fixture) CreateTestPerson(ctx context.Context, steamID steamid.SteamID, perm permission.Privilege) personDomain.Core {
	people := person.NewPersons(person.NewRepository(f.Config.Config(), f.Database), OwnerSID, nil)
	person, errPerson := people.GetOrCreatePersonBySteamID(ctx, steamID)
	if errPerson != nil {
		panic(errPerson)
	}
	full, _ := people.BySteamID(ctx, steamID)
	full.PermissionLevel = perm
	person.PermissionLevel = perm
	_ = people.Save(ctx, &full)

	return person
}

func (f Fixture) CreateTestServer(ctx context.Context) servers.Server {
	serverCase := servers.NewServers(servers.NewRepository(f.Database))
	server, errServer := serverCase.Save(ctx, servers.Server{
		Name:      stringutil.SecureRandomString(10),
		ShortName: stringutil.SecureRandomString(3),
		Address:   stringutil.SecureRandomString(10),
		Password:  stringutil.SecureRandomString(10),
		LogSecret: 12345678,
		Port:      27015,
		RCON:      stringutil.SecureRandomString(10),
		IsEnabled: true,
		Region:    "eu",
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	})
	if errServer != nil {
		panic(errServer)
	}

	return server
}

func TestConfig(dsn string) *config.Configuration {
	return config.NewConfiguration(config.Static{}, config.NewMemConfigRepository(config.Config{
		Static: config.
			Static{
			Owner: OwnerSID.String(),
			//	SteamKey:            steamKey,
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
		General: config.General{
			SiteName:        "gbans",
			Mode:            config.TestMode,
			FileServeMode:   "local",
			SrcdsLogAddr:    "",
			AssetURL:        "",
			DefaultRoute:    "",
			NewsEnabled:     true,
			ForumsEnabled:   true,
			ContestsEnabled: true,
			WikiEnabled:     true,
			StatsEnabled:    true,
			ServersEnabled:  true,
			ReportsEnabled:  true,
			ChatlogsEnabled: true,
			DemosEnabled:    true,
		},
		Demo: config.Demo{
			DemoCleanupEnabled:  false,
			DemoCleanupStrategy: "",
			DemoCleanupMinPct:   0,
			DemoCleanupMount:    "",
			DemoCountLimit:      2,
		},
		Filters: config.Filter{
			Enabled:        true,
			WarningTimeout: 10,
			WarningLimit:   1,
			Dry:            false,
			PingDiscord:    false,
			MaxWeight:      1,
			CheckTimeout:   10,
			MatchTimeout:   10,
		},
		Discord: config.Discord{
			Enabled: false,
		},
		Clientprefs: config.Clientprefs{},
		Log: config.Log{
			HTTPEnabled: false,
			Level:       "error",
		},
		GeoLocation: config.IP2Location{
			Enabled: false,
		},
		Debug: config.Debug{},
		Patreon: config.Patreon{
			Enabled: false,
		},
		SSH: config.SSH{
			Enabled: false,
		},
		LocalStore: config.LocalStore{},
		Exports:    config.Exports{},
	}))
}
