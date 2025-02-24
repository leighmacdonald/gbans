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
	"github.com/leighmacdonald/gbans/internal/anticheat"
	"github.com/leighmacdonald/gbans/internal/app"
	"github.com/leighmacdonald/gbans/internal/appeal"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/blocklist"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/demo"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/forum"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/match"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/news"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/patreon"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/report"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/srcds"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/steamgroup"
	"github.com/leighmacdonald/gbans/internal/votes"
	"github.com/leighmacdonald/gbans/internal/wiki"
	"github.com/leighmacdonald/gbans/internal/wordfilter"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

var (
	dbContainer    *postgresContainer
	tempDB         database.Database
	testServer     domain.Server
	testBan        domain.BannedSteamPerson
	testTarget     domain.Person
	blocklistUC    domain.BlocklistUsecase
	configUC       domain.ConfigUsecase
	wikiUC         domain.WikiUsecase
	personUC       domain.PersonUsecase
	authRepo       domain.AuthRepository
	authUC         domain.AuthUsecase
	networkUC      domain.NetworkUsecase
	banSteamUC     domain.BanSteamUsecase
	banASNUC       domain.BanASNUsecase
	banGroupUC     domain.BanGroupUsecase
	banNetUC       domain.BanNetUsecase
	assetUC        domain.AssetUsecase
	chatUC         domain.ChatUsecase
	demoRepository domain.DemoRepository
	demoUC         domain.DemoUsecase
	discordUC      domain.DiscordUsecase
	forumUC        domain.ForumUsecase
	matchUC        domain.MatchUsecase
	newsUC         domain.NewsUsecase
	notificationUC domain.NotificationUsecase
	patreonUC      domain.PatreonUsecase
	reportUC       domain.ReportUsecase
	serversUC      domain.ServersUsecase
	speedrunsUC    domain.SpeedrunUsecase
	srcdsUC        domain.SRCDSUsecase
	stateUC        domain.StateUsecase
	votesUC        domain.VoteUsecase
	votesRepo      domain.VoteRepository
	wordFilterUC   domain.WordFilterUsecase
	appealUC       domain.AppealUsecase
	anticheatUC    domain.AntiCheatUsecase
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
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
	weaponsMap := fp.NewMutexMap[logparse.Weapon, int]()

	configUC = config.NewConfigUsecase(conf.StaticConfig, newConfigRepo(conf))
	if err := configUC.Reload(testCtx); err != nil {
		panic(err)
	}

	if err := configUC.Write(testCtx, configUC.Config()); err != nil {
		panic(err)
	}

	authRepo = auth.NewAuthRepository(databaseConn)

	discordUC = discord.NewDiscordUsecase(discord.NewNullDiscordRepository(), configUC)

	assetUC = asset.NewAssetUsecase(asset.NewLocalRepository(databaseConn, configUC))
	newsUC = news.NewNewsUsecase(news.NewNewsRepository(databaseConn))
	serversUC = servers.NewServersUsecase(servers.NewServersRepository(databaseConn))
	wikiUC = wiki.NewWikiUsecase(wiki.NewWikiRepository(databaseConn))
	notificationUC = notification.NewNotificationUsecase(notification.NewNotificationRepository(databaseConn), discordUC)
	patreonUC = patreon.NewPatreonUsecase(patreon.NewPatreonRepository(databaseConn), configUC)
	personUC = person.NewPersonUsecase(person.NewPersonRepository(conf, databaseConn), configUC)
	wordFilterUC = wordfilter.NewWordFilterUsecase(wordfilter.NewWordFilterRepository(databaseConn), notificationUC)
	forumUC = forum.NewForumUsecase(forum.NewForumRepository(databaseConn), notificationUC)

	stateUC = state.NewStateUsecase(eventBroadcaster, state.NewStateRepository(state.NewCollector(serversUC)), configUC, serversUC)

	networkUC = network.NewNetworkUsecase(eventBroadcaster, network.NewNetworkRepository(databaseConn), personUC, configUC)
	demoRepository = demo.NewDemoRepository(databaseConn)
	demoUC = demo.NewDemoUsecase("demos", demoRepository, assetUC, configUC, serversUC)
	reportUC = report.NewReportUsecase(report.NewReportRepository(databaseConn), notificationUC, configUC, personUC, demoUC)
	banSteamUC = ban.NewBanSteamUsecase(ban.NewBanSteamRepository(databaseConn, personUC, networkUC), personUC, configUC, notificationUC, reportUC, stateUC)
	authUC = auth.NewAuthUsecase(authRepo, configUC, personUC, banSteamUC, serversUC)
	banASNUC = ban.NewBanASNUsecase(ban.NewBanASNRepository(databaseConn), notificationUC, networkUC, configUC, personUC)
	banGroupUC = steamgroup.NewBanGroupUsecase(steamgroup.NewSteamGroupRepository(databaseConn), personUC, notificationUC, configUC)
	banNetUC = ban.NewBanNetUsecase(ban.NewBanNetRepository(databaseConn), personUC, configUC, notificationUC, stateUC)

	matchUC = match.NewMatchUsecase(match.NewMatchRepository(eventBroadcaster, databaseConn, personUC, serversUC, notificationUC, stateUC, weaponsMap), stateUC, serversUC, notificationUC)
	chatUC = chat.NewChatUsecase(configUC, chat.NewChatRepository(databaseConn, personUC, wordFilterUC, matchUC, eventBroadcaster), wordFilterUC, stateUC, banSteamUC, personUC, notificationUC)
	votesRepo = votes.NewVoteRepository(databaseConn)
	votesUC = votes.NewVoteUsecase(votesRepo, personUC, matchUC, notificationUC, configUC, eventBroadcaster)
	appealUC = appeal.NewAppealUsecase(appeal.NewAppealRepository(databaseConn), banSteamUC, personUC, notificationUC, configUC)
	speedrunsUC = srcds.NewSpeedrunUsecase(srcds.NewSpeedrunRepository(databaseConn, personUC))
	blocklistUC = blocklist.NewBlocklistUsecase(blocklist.NewBlocklistRepository(databaseConn), banSteamUC, banGroupUC)
	anticheatUC = anticheat.NewAntiCheatUsecase(anticheat.NewAntiCheatRepository(databaseConn), personUC, banSteamUC, configUC)

	if internalDB {
		server, errServer := serversUC.Save(context.Background(), domain.RequestServerUpdate{
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

		if errServer != nil && !errors.Is(errServer, domain.ErrDuplicate) {
			panic(errServer)
		}
		testServer = server
	} else {
		srvs, _, errServer := serversUC.Servers(context.Background(), domain.ServerQueryFilter{})
		if len(srvs) == 0 || errServer != nil {
			panic("no servers exist, please create at least one before testing")
		}
		testServer = srvs[0]
	}

	getOwner()

	mod := getModerator()
	target := getUser()

	// Create a valid ban_id
	bannedPerson, errBan := banSteamUC.Ban(context.Background(), mod.ToUserProfile(), domain.System, domain.RequestBanSteamCreate{
		SourceIDField:  domain.SourceIDField{SourceID: mod.SteamID.String()},
		TargetIDField:  domain.TargetIDField{TargetID: target.SteamID.String()},
		Duration:       "1d",
		BanType:        domain.Banned,
		Reason:         domain.Cheating,
		ReasonText:     "",
		Note:           "notes",
		ReportID:       0,
		DemoName:       "demo-test.dem",
		DemoTick:       100,
		IncludeFriends: true,
		EvadeOk:        true,
	})

	if errBan != nil && !errors.Is(errBan, domain.ErrDuplicate) {
		panic(errBan)
	}

	testBan = bannedPerson
	testTarget = target
	tempDB = databaseConn

	m.Run()
}

func testRouter() *gin.Engine {
	router, errRouter := httphelper.CreateRouter(configUC.Config(), domain.BuildInfo{
		BuildVersion: "master",
		Commit:       "",
		Date:         time.Now().Format(time.DateTime),
	})

	if errRouter != nil {
		panic(errRouter)
	}

	ban.NewHandlerSteam(router, banSteamUC, configUC, authUC)
	ban.NewHandlerNet(router, banNetUC, authUC)
	ban.NewASNHandlerASN(router, banASNUC, authUC)
	servers.NewHandler(router, serversUC, stateUC, authUC)
	steamgroup.NewHandler(router, banGroupUC, authUC)
	news.NewHandler(router, newsUC, notificationUC, authUC)
	wiki.NewHandler(router, wikiUC, authUC)
	votes.NewHandler(router, votesUC, authUC)
	config.NewHandler(router, configUC, authUC, app.Version())
	report.NewHandler(router, reportUC, authUC, notificationUC)
	appeal.NewHandler(router, appealUC, authUC)
	wordfilter.NewHandler(router, configUC, wordFilterUC, chatUC, authUC)
	person.NewHandler(router, configUC, personUC, authUC)
	srcds.NewHandlerSRCDS(router, srcdsUC, serversUC, personUC, assetUC, reportUC, banSteamUC, networkUC, banGroupUC,
		authUC, banASNUC, banNetUC, configUC, notificationUC, stateUC, blocklistUC)
	blocklist.NewHandler(router, blocklistUC, networkUC, authUC)
	srcds.NewHandler(router, speedrunsUC, authUC, configUC)

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
	user           *domain.UserTokens
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
				Name:     domain.FingerprintCookieName,
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

func createTestPerson(sid steamid.SteamID, level domain.Privilege) domain.Person {
	player, err := personUC.GetOrCreatePersonBySteamID(context.Background(), nil, sid)
	if err != nil {
		panic(err)
	}

	player.PermissionLevel = level

	if errSave := personUC.SavePerson(context.Background(), nil, &player); errSave != nil {
		panic(errSave)
	}

	return player
}

func getOwner() domain.Person {
	return createTestPerson(steamid.New(configUC.Config().Owner), domain.PAdmin)
}

var curUserID atomic.Int32

func getUser() domain.Person {
	return createTestPerson(steamid.New(76561198004429398+int64(curUserID.Add(1))), domain.PUser)
}

func getModerator() domain.Person {
	return createTestPerson(steamid.New(76561198057999536), domain.PModerator)
}

func loginUser(person domain.Person) *domain.UserTokens {
	conf := configUC.Config()
	fingerprint := stringutil.SecureRandomString(40)

	accessToken, errAccess := authUC.NewUserToken(person.SteamID, conf.HTTPCookieKey, fingerprint, domain.AuthTokenDuration)
	if errAccess != nil {
		panic(errAccess)
	}

	ipAddr := net.ParseIP("127.0.0.1")
	if ipAddr == nil {
		panic(domain.ErrClientIP)
	}

	personAuth := domain.NewPersonAuth(person.SteamID, ipAddr, accessToken)
	if saveErr := authRepo.SavePersonAuth(context.Background(), &personAuth); saveErr != nil {
		panic(saveErr)
	}

	return &domain.UserTokens{Access: accessToken, Fingerprint: fingerprint}
}

func makeTestConfig(dsn string) domain.Config {
	steamKey, found := os.LookupEnv("GBANS_GENERAL_STEAM_KEY")
	if !found || len(steamKey) != 32 {
		panic("GBANS_GENERAL_STEAM_KEY is not set, or is invalid")
	}

	return domain.Config{
		StaticConfig: domain.StaticConfig{
			Owner:               "76561198084134025",
			SteamKey:            steamKey,
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
		General: domain.ConfigGeneral{
			SiteName:        "gbans",
			Mode:            domain.TestMode,
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
		Demo: domain.ConfigDemo{
			DemoCleanupEnabled:  false,
			DemoCleanupStrategy: "",
			DemoCleanupMinPct:   0,
			DemoCleanupMount:    "",
			DemoCountLimit:      2,
		},
		Filters: domain.ConfigFilter{
			Enabled:        true,
			WarningTimeout: 10,
			WarningLimit:   1,
			Dry:            false,
			PingDiscord:    false,
			MaxWeight:      1,
			CheckTimeout:   10,
			MatchTimeout:   10,
		},
		Discord: domain.ConfigDiscord{
			Enabled: false,
		},
		Clientprefs: domain.ConfigClientprefs{},
		Log: domain.ConfigLog{
			HTTPEnabled: false,
			Level:       "error",
		},
		GeoLocation: domain.ConfigIP2Location{
			Enabled: false,
		},
		Debug: domain.ConfigDebug{},
		Patreon: domain.ConfigPatreon{
			Enabled: false,
		},
		SSH: domain.ConfigSSH{
			Enabled: false,
		},
		LocalStore: domain.ConfigLocalStore{},
		Exports:    domain.ConfigExports{},
		Sentry:     domain.ConfigSentry{},
	}
}
