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
	"github.com/leighmacdonald/gbans/internal/cmd"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/pkg/fs"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	OwnerSID = steamid.New(76561198084134025)
	ModSID   = steamid.New(76561198084134026)
	UserSID  = steamid.New(76561198084134027)
	GuestSID = steamid.New(76561198084134028)

	ErrContainer = errors.New("failed to bring up test container")

	authed     = []permission.Privilege{permission.Guest}                                        //nolint:gochecknoglobals
	moderators = []permission.Privilege{permission.Guest, permission.User}                       //nolint:gochecknoglobals
	admin      = []permission.Privilege{permission.Guest, permission.User, permission.Moderator} //nolint:gochecknoglobals
)

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
		Dockerfile:    "postgres-ip4r.Dockerfile",
		Context:       dockerRoot,
		PrintBuildLog: false,
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

type permTestValues struct {
	method string
	code   int
	path   string
	levels []permission.Privilege
}

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
	Config    config.Config
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

	return &Fixture{
		container: testDB,
		Database:  databaseConn,
		Config:    TestConfig(testDB.dsn),
		Close: func() {
			termCtx, termCancel := context.WithTimeout(context.Background(), time.Second*30)
			defer termCancel()

			if errTerm := testDB.Terminate(termCtx); errTerm != nil {
				panic(fmt.Sprintf("Failed to terminate test container: %v", errTerm))
			}
		},
	}
}

func testRouter(conf config.Config) *gin.Engine {
	conf.General.Mode = config.TestMode
	router, errRouter := cmd.CreateRouter(conf, cmd.BuildInfo{
		BuildVersion: "master",
		Commit:       "",
		Date:         time.Now().Format(time.DateTime),
	})

	if errRouter != nil {
		panic(errRouter)
	}

	return router
}

func testEndpointWithReceiver(t *testing.T, router *gin.Engine, method string,
	path string, body any, expectedStatus int, tokens *authTokens, receiver any,
) {
	t.Helper()

	resp := testEndpoint(t, router, method, path, body, expectedStatus, tokens)
	if receiver != nil {
		if err := json.NewDecoder(resp.Body).Decode(&receiver); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
	}
}

type authTokens struct {
	user           *auth.UserTokens
	serverPassword string
}

func testEndpoint(t *testing.T, router *gin.Engine, method string, path string, body any, expectedStatus int, tokens *authTokens) *httptest.ResponseRecorder {
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

// func createTestPerson(sid steamid.SteamID, level permission.Privilege) person.Person {
// 	_, _ = personUC.GetOrCreatePersonBySteamID(context.Background(), nil, sid)

// 	player, err := personUC.GetPersonBySteamID(context.Background(), nil, sid)
// 	if err != nil {
// 		panic(err)
// 	}

// 	player.PermissionLevel = level

// 	if errSave := personUC.SavePerson(context.Background(), nil, &player); errSave != nil {
// 		panic(errSave)
// 	}

// 	return player
// }

// func getOwner() person.Person {
// 	return createTestPerson(steamid.New(configUC.Config().Owner), permission.Admin)
// }

// var curUserID atomic.Int32

// func getUser() person.Person {
// 	return createTestPerson(steamid.New(76561198004429398+int64(curUserID.Add(1))), permission.User)
// }

// func getModerator() person.Person {
// 	return createTestPerson(steamid.New(76561198057999536), permission.Moderator)
// }

// func loginUser(person person.Person) *auth.UserTokens {
// 	conf := configUC.Config()
// 	fingerprint := stringutil.SecureRandomString(40)

// 	accessToken, errAccess := authUC.NewUserToken(person.SteamID, conf.HTTPCookieKey, fingerprint, auth.AuthTokenDuration)
// 	if errAccess != nil {
// 		panic(errAccess)
// 	}

// 	ipAddr := net.ParseIP("127.0.0.1")
// 	if ipAddr == nil {
// 		panic(domain.ErrClientIP)
// 	}

// 	personAuth := auth.NewPersonAuth(person.SteamID, ipAddr, accessToken)
// 	if saveErr := authRepo.SavePersonAuth(context.Background(), &personAuth); saveErr != nil {
// 		panic(saveErr)
// 	}

//		return &auth.UserTokens{Access: accessToken, Fingerprint: fingerprint}
//	}
func (f Fixture) CreateTestPerson(ctx context.Context, steamID steamid.SteamID) domain.PersonCore {
	p := person.NewPersons(person.NewRepository(f.Config, f.Database), OwnerSID, nil)
	person, errPerson := p.GetOrCreatePersonBySteamID(ctx, nil, steamID)
	if errPerson != nil {
		panic(errPerson)
	}
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

func TestConfig(dsn string) config.Config {
	return config.Config{
		Static: config.Static{
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
	}
}

// func TestMain(m *testing.M) {
// 	slog.SetDefault(slog.New(slog.DiscardHandler))

// 	conf := makeTestConfig(dsn)
// 	eventBroadcaster := broadcaster.New[logparse.EventType, logparse.ServerEvent]()
// 	// weaponsMap := fp.NewMutexMap[logparse.Weapon, int]()

// 	configUC = config.NewConfiguration(conf.Static, &TestConfigRepo{config: conf})
// 	if err := configUC.Reload(testCtx); err != nil {
// 		panic(err)
// 	}

// 	// if err := configUC.Write(testCtx, configUC.Config()); err != nil {
// 	// 	panic(err)
// 	// }

// 	// TODO caching client?
// 	tfapiClient, errClient := thirdparty.NewTFAPI("https://tf-api.roto.lol", &http.Client{Timeout: time.Second * 15})
// 	if errClient != nil {
// 		panic(errClient)
// 	}

// 	authRepo = auth.NewRepository(databaseConn)

// 	disc, errDiscord := discord.NewDiscord("dummy", "dummy", "dummy", "dummy")
// 	if errDiscord != nil {
// 		panic(errDiscord)
// 	}
// 	discordUC = disc

// assets = asset.NewAssets(asset.NewLocalRepository(databaseConn, configUC.Config().LocalStore.PathRoot))
// newsUC = news.NewNews(news.NewRepository(databaseConn))
// serversUC = servers.NewServers(servers.NewRepository(databaseConn))
// wikiUC = wiki.NewWiki(wiki.NewRepository(databaseConn))
// notificationUC = notification.NewNotifications(notification.NewRepository(databaseConn), discordUC)
// patreonUC = patreon.NewPatreonManager(configUC)
// personUC = person.NewPersons(person.NewRepository(conf, databaseConn), configUC, tfapiClient)
// wordFilterUC = chat.NewWordFilters(chat.NewWordFilterRepository(databaseConn), notificationUC, configUC)
// forumUC = forum.NewForums(forum.NewRepository(databaseConn), configUC, notificationUC)

// stateUC = servers.NewState(eventBroadcaster, servers.NewStateRepository(servers.NewCollector(serversUC)), configUC, serversUC)

// networkUC = network.NewNetworks(eventBroadcaster, network.NewRepository(databaseConn, personUC), configUC)
// demoRepository = servers.NewDemoRepository(databaseConn)
// demoUC = servers.NewDemos("demos", demoRepository, assets, configUC)
// reportUC = ban.NewReports(ban.NewReportRepository(databaseConn), configUC, personUC, demoUC, tfapiClient, notificationUC)
// bansUC = ban.NewBans(ban.NewRepository(databaseConn, personUC, networkUC), personUC, configUC, reportUC, stateUC, tfapiClient, notificationUC)
// authUC = auth.NewAuthentication(authRepo, configUC, personUC, bansUC, serversUC, cmd.SentryDSN)
// chatUC = chat.NewChat(chat.NewRepository(databaseConn), configUC, wordFilterUC, bansUC, personUC)
// votesRepo = votes.NewRepository(databaseConn)
// votesUC = votes.NewVotes(votesRepo, eventBroadcaster, notificationUC, configUC, personUC)
// appealUC = ban.NewAppeals(ban.NewAppealRepository(databaseConn), bansUC, personUC, configUC, notificationUC)
// speedrunsUC = servers.NewSpeedruns(servers.NewSpeedrunRepository(databaseConn, personUC))
// blocklistUC = network.NewBlocklists(network.NewBlocklistRepository(databaseConn), &bansUC)
// anticheatUC = anticheat.NewAntiCheat(anticheat.NewRepository(databaseConn), bansUC, configUC, personUC, notificationUC)

// if internalDB {
// 	server, errServer := serversUC.Save(context.Background(), servers.RequestServerUpdate{
// 		ServerName:      stringutil.SecureRandomString(20),
// 		ServerNameShort: stringutil.SecureRandomString(5),
// 		Host:            "1.2.3.4",
// 		Port:            27015,
// 		ReservedSlots:   8,
// 		Password:        stringutil.SecureRandomString(8),
// 		RCON:            stringutil.SecureRandomString(8),
// 		Lat:             10,
// 		Lon:             10,
// 		CC:              "de",
// 		Region:          "eu",
// 		IsEnabled:       true,
// 		EnableStats:     false,
// 		LogSecret:       23456789,
// 	})

// 	if errServer != nil && !errors.Is(errServer, database.ErrDuplicate) {
// 		panic(errServer)
// 	}
// 	testServer = server
// } else {
// 	srvs, _, errServer := serversUC.Servers(context.Background(), servers.ServerQueryFilter{})
// 	if len(srvs) == 0 || errServer != nil {
// 		panic("no servers exist, please create at least one before testing")
// 	}
// 	testServer = srvs[0]
// }

// getOwner()

// mod := getModerator()
// target := getUser()

// // Create a valid ban_id
// bannedPerson, errBan := bansUC.Create(context.Background(), ban.Opts{
// 	SourceID:   mod.SteamID,
// 	TargetID:   target.SteamID,
// 	Duration:   duration.FromTimeDuration(time.Hour * 2),
// 	BanType:    banDomain.Banned,
// 	Reason:     banDomain.Cheating,
// 	Origin:     banDomain.System,
// 	ReasonText: "",
// 	Note:       "notes",
// 	ReportID:   0,
// 	DemoName:   "demo-test.dem",
// 	DemoTick:   100,
// 	EvadeOk:    true,
// 	CIDR:       nil,
// 	Name:       "",
// })

// if errBan != nil && !errors.Is(errBan, database.ErrDuplicate) {
// 	panic(errBan)
// }

// testBan = bannedPerson
// testTarget = target
// tempDB = databaseConn

// m.Run()
// }
