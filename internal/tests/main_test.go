//nolint:gochecknoglobals
package tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/go-querystring/query"
	_ "github.com/joho/godotenv/autoload"
	"github.com/leighmacdonald/gbans/internal/anticheat"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/cmd"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	banDomain "github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/internal/forum"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/news"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/patreon"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/internal/votes"
	"github.com/leighmacdonald/gbans/internal/wiki"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/sosodev/duration"
	"github.com/stretchr/testify/require"
)

var (
	dbContainer    *postgresContainer
	tempDB         database.Database
	testServer     servers.Server
	testBan        ban.Ban
	testTarget     person.Person
	blocklistUC    network.Blocklists
	configUC       *config.Configuration
	wikiUC         wiki.Wiki
	personUC       *person.Persons
	authRepo       auth.Repository
	authUC         *auth.Authentication
	networkUC      network.Networks
	bansUC         ban.Bans
	assets         asset.Assets
	chatUC         *chat.Chat
	demoRepository servers.DemoRepository
	demoUC         servers.Demos
	discordUC      *discord.Discord
	forumUC        forum.Forums
	newsUC         news.News
	notificationUC notification.Notifications
	patreonUC      patreon.Manager
	reportUC       ban.Reports
	serversUC      servers.Servers
	speedrunsUC    servers.Speedruns
	srcdsUC        *servers.SRCDS
	stateUC        *servers.State
	votesUC        votes.Votes
	votesRepo      votes.Repository
	wordFilterUC   chat.WordFilters
	appealUC       ban.Appeals
	anticheatUC    anticheat.AntiCheat
)

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

func (c *TestConfigRepo) Init(ctx context.Context) error {
	return nil
}

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.DiscardHandler))
	var dsn string
	testCtx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	internalDB := os.Getenv("TEST_DB_DSN") == ""
	if internalDB {
		testDB, errStore := newDB(testCtx)
		if errStore != nil {
			panic(errStore)
		}

		defer func() {
			termCtx, termCancel := context.WithTimeout(context.Background(), time.Second*30)
			defer termCancel()

			if errTerm := testDB.Terminate(termCtx); errTerm != nil {
				panic(fmt.Sprintf("Failed to terminate test container: %v", errTerm))
			}
		}()
		dsn = testDB.dsn
		dbContainer = testDB
	} else {
		// assumes some data already exists
		dsn = os.Getenv("TEST_DB_DSN")
	}

	databaseConn := database.New(dsn, true, false)
	if err := databaseConn.Connect(testCtx); err != nil {
		panic(err)
	}

	conf := makeTestConfig(dsn)
	eventBroadcaster := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()
	// weaponsMap := fp.NewMutexMap[logparse.Weapon, int]()

	configUC = config.NewConfiguration(conf.StaticConfig, &TestConfigRepo{config: conf})
	if err := configUC.Reload(testCtx); err != nil {
		panic(err)
	}

	// if err := configUC.Write(testCtx, configUC.Config()); err != nil {
	// 	panic(err)
	// }

	// TODO caching client?
	tfapiClient, errClient := thirdparty.NewTFAPI("https://tf-api.roto.lol", &http.Client{Timeout: time.Second * 15})
	if errClient != nil {
		panic(errClient)
	}

	authRepo = auth.NewRepository(databaseConn)

	disc, errDiscord := discord.NewDiscord("dummy", "dummy", "dummy", "dummy")
	if errDiscord != nil {
		panic(errDiscord)
	}
	discordUC = disc

	assets = asset.NewAssets(asset.NewLocalRepository(databaseConn, configUC.Config().LocalStore.PathRoot))
	newsUC = news.NewNews(news.NewRepository(databaseConn))
	serversUC = servers.NewServers(servers.NewRepository(databaseConn))
	wikiUC = wiki.NewWiki(wiki.NewRepository(databaseConn))
	notificationUC = notification.NewNotifications(notification.NewRepository(databaseConn), discordUC)
	patreonUC = patreon.NewPatreonManager(configUC)
	personUC = person.NewPersons(person.NewRepository(conf, databaseConn), configUC, tfapiClient)
	wordFilterUC = chat.NewWordFilter(chat.NewWordFilterRepository(databaseConn))
	forumUC = forum.NewForums(forum.NewRepository(databaseConn))

	stateUC = servers.NewState(eventBroadcaster, servers.NewStateRepository(servers.NewCollector(serversUC)), configUC, serversUC)

	networkUC = network.NewNetworks(eventBroadcaster, network.NewRepository(databaseConn, personUC), configUC)
	demoRepository = servers.NewDemoRepository(databaseConn)
	demoUC = servers.NewDemos("demos", demoRepository, assets, configUC)
	reportUC = ban.NewReports(ban.NewReportRepository(databaseConn), configUC, personUC, demoUC, tfapiClient)
	bansUC = ban.NewBans(ban.NewBanRepository(databaseConn, personUC, networkUC), personUC, configUC, reportUC, stateUC, tfapiClient)
	authUC = auth.NewAuthentication(authRepo, configUC, personUC, bansUC, serversUC, cmd.SentryDSN)
	chatUC = chat.NewChat(configUC, chat.NewChatRepository(databaseConn, personUC, wordFilterUC, eventBroadcaster), wordFilterUC, stateUC, bansUC, personUC)
	votesRepo = votes.NewRepository(databaseConn)
	votesUC = votes.NewVotes(votesRepo, eventBroadcaster)
	appealUC = ban.NewAppeals(ban.NewAppealRepository(databaseConn), bansUC, personUC, configUC)
	speedrunsUC = servers.NewSpeedruns(servers.NewSpeedrunRepository(databaseConn, personUC))
	blocklistUC = network.NewBlocklists(network.NewBlocklistRepository(databaseConn), &bansUC)
	anticheatUC = anticheat.NewAntiCheat(anticheat.NewRepository(databaseConn), bansUC, configUC, personUC)

	if internalDB {
		server, errServer := serversUC.Save(context.Background(), servers.RequestServerUpdate{
			ServerName:      stringutil.SecureRandomString(20),
			ServerNameShort: stringutil.SecureRandomString(5),
			Host:            "1.2.3.4",
			Port:            27015,
			ReservedSlots:   8,
			Password:        stringutil.SecureRandomString(8),
			RCON:            stringutil.SecureRandomString(8),
			Lat:             10,
			Lon:             10,
			CC:              "de",
			Region:          "eu",
			IsEnabled:       true,
			EnableStats:     false,
			LogSecret:       23456789,
		})

		if errServer != nil && !errors.Is(errServer, database.ErrDuplicate) {
			panic(errServer)
		}
		testServer = server
	} else {
		srvs, _, errServer := serversUC.Servers(context.Background(), servers.ServerQueryFilter{})
		if len(srvs) == 0 || errServer != nil {
			panic("no servers exist, please create at least one before testing")
		}
		testServer = srvs[0]
	}

	getOwner()

	mod := getModerator()
	target := getUser()

	// Create a valid ban_id
	bannedPerson, errBan := bansUC.Create(context.Background(), ban.BanOpts{
		SourceID:   mod.SteamID,
		TargetID:   target.SteamID,
		Duration:   duration.FromTimeDuration(time.Hour * 2),
		BanType:    banDomain.Banned,
		Reason:     banDomain.Cheating,
		Origin:     banDomain.System,
		ReasonText: "",
		Note:       "notes",
		ReportID:   0,
		DemoName:   "demo-test.dem",
		DemoTick:   100,
		EvadeOk:    true,
		CIDR:       nil,
		Name:       "",
	})

	if errBan != nil && !errors.Is(errBan, database.ErrDuplicate) {
		panic(errBan)
	}

	testBan = bannedPerson
	testTarget = target
	tempDB = databaseConn

	m.Run()
}

func testRouter() *gin.Engine {
	conf := configUC.Config()
	conf.General.Mode = config.TestMode
	router, errRouter := cmd.CreateRouter(conf, cmd.BuildInfo{
		BuildVersion: "master",
		Commit:       "",
		Date:         time.Now().Format(time.DateTime),
	})

	if errRouter != nil {
		panic(errRouter)
	}

	ban.NewHandlerSteam(router, bansUC, configUC, authUC)
	servers.NewServersHandler(router, serversUC, stateUC, authUC)
	news.NewNewsHandler(router, newsUC, notificationUC, authUC)
	wiki.NewWikiHandler(router, wikiUC, authUC)
	votes.NewVotesHandler(router, votesUC, authUC)
	config.NewConfigHandler(router, configUC, authUC, cmd.BuildVersion)
	ban.NewReportHandler(router, reportUC, authUC)
	ban.NewAppealHandler(router, appealUC, authUC)
	chat.NewChatHandler(router, chatUC, authUC)
	chat.NewWordFilterHandler(router, configUC, wordFilterUC, chatUC, authUC)
	person.NewPersonHandler(router, configUC, personUC, authUC)
	servers.NewSRCDSHandler(router, srcdsUC, serversUC, personUC, assets, bansUC, networkUC, authUC, configUC, stateUC, blocklistUC, cmd.SentryDSN)
	network.NewBlocklistHandler(router, blocklistUC, networkUC, authUC)
	servers.NewSpeedrunsHandler(router, speedrunsUC, authUC, configUC, serversUC, cmd.SentryDSN)

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

func createTestPerson(sid steamid.SteamID, level permission.Privilege) person.Person {
	_, _ = personUC.GetOrCreatePersonBySteamID(context.Background(), nil, sid)

	player, err := personUC.GetPersonBySteamID(context.Background(), nil, sid)
	if err != nil {
		panic(err)
	}

	player.PermissionLevel = level

	if errSave := personUC.SavePerson(context.Background(), nil, &player); errSave != nil {
		panic(errSave)
	}

	return player
}

func getOwner() person.Person {
	return createTestPerson(steamid.New(configUC.Config().Owner), permission.PAdmin)
}

var curUserID atomic.Int32

func getUser() person.Person {
	return createTestPerson(steamid.New(76561198004429398+int64(curUserID.Add(1))), permission.PUser)
}

func getModerator() person.Person {
	return createTestPerson(steamid.New(76561198057999536), permission.PModerator)
}

func loginUser(person person.Person) *auth.UserTokens {
	conf := configUC.Config()
	fingerprint := stringutil.SecureRandomString(40)

	accessToken, errAccess := authUC.NewUserToken(person.SteamID, conf.HTTPCookieKey, fingerprint, auth.AuthTokenDuration)
	if errAccess != nil {
		panic(errAccess)
	}

	ipAddr := net.ParseIP("127.0.0.1")
	if ipAddr == nil {
		panic(domain.ErrClientIP)
	}

	personAuth := auth.NewPersonAuth(person.SteamID, ipAddr, accessToken)
	if saveErr := authRepo.SavePersonAuth(context.Background(), &personAuth); saveErr != nil {
		panic(saveErr)
	}

	return &auth.UserTokens{Access: accessToken, Fingerprint: fingerprint}
}

func makeTestConfig(dsn string) config.Config {
	return config.Config{
		StaticConfig: config.StaticConfig{
			Owner: "76561198084134025",
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
