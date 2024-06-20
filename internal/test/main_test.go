//nolint:gochecknoglobals
package test_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/auth"
	"github.com/leighmacdonald/gbans/internal/ban"
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
	container      *postgresContainer
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
	demoUC         domain.DemoUsecase
	discordUC      domain.DiscordUsecase
	forumUC        domain.ForumUsecase
	matchUC        domain.MatchUsecase
	newsUC         domain.NewsUsecase
	notificationUC domain.NotificationUsecase
	patreonUC      domain.PatreonUsecase
	reportUC       domain.ReportUsecase
	serversUC      domain.ServersUsecase
	stateUC        domain.StateUsecase
	votesUC        domain.VoteUsecase
	wordFilterUC   domain.WordFilterUsecase
)

func TestMain(m *testing.M) {
	testCtx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	dbContainer, errStore := newDB(testCtx)
	if errStore != nil {
		panic(errStore)
	}

	defer func() {
		termCtx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		if errTerm := container.Terminate(termCtx); errTerm != nil {
			panic(fmt.Sprintf("Failed to terminate test container: %v", errTerm))
		}
	}()

	databaseConn := database.New(dbContainer.dsn, true, false)
	if err := databaseConn.Connect(testCtx); err != nil {
		panic(err)
	}

	conf := makeTestConfig(dbContainer.dsn)
	eventBroadcaster := fp.NewBroadcaster[logparse.EventType, logparse.ServerEvent]()
	weaponsMap := fp.NewMutexMap[logparse.Weapon, int]()

	configUC = config.NewConfigUsecase(conf.StaticConfig, newConfigRepo(conf))
	if err := configUC.Reload(testCtx); err != nil {
		panic(err)
	}

	authRepo = auth.NewAuthRepository(databaseConn)

	discordUC = discord.NewDiscordUsecase(discord.NewNullDiscordRepository(), configUC)

	assetUC = asset.NewAssetUsecase(asset.NewLocalRepository(databaseConn, configUC))
	newsUC = news.NewNewsUsecase(news.NewNewsRepository(databaseConn))
	serversUC = servers.NewServersUsecase(servers.NewServersRepository(databaseConn))
	wikiUC = wiki.NewWikiUsecase(wiki.NewWikiRepository(databaseConn))

	patreonUC = patreon.NewPatreonUsecase(patreon.NewPatreonRepository(databaseConn), configUC)
	personUC = person.NewPersonUsecase(person.NewPersonRepository(conf, databaseConn), configUC)
	wordFilterUC = wordfilter.NewWordFilterUsecase(wordfilter.NewWordFilterRepository(databaseConn), discordUC)
	forumUC = forum.NewForumUsecase(forum.NewForumRepository(databaseConn), discordUC)

	notificationUC = notification.NewNotificationUsecase(notification.NewNotificationRepository(databaseConn), personUC)
	stateUC = state.NewStateUsecase(eventBroadcaster, state.NewStateRepository(state.NewCollector(serversUC)), configUC, serversUC)

	networkUC = network.NewNetworkUsecase(eventBroadcaster, network.NewNetworkRepository(databaseConn), personUC, configUC)
	demoUC = demo.NewDemoUsecase("demos", demo.NewDemoRepository(databaseConn), assetUC, configUC, serversUC)
	reportUC = report.NewReportUsecase(report.NewReportRepository(databaseConn), discordUC, configUC, personUC, demoUC)
	banSteamUC = ban.NewBanSteamUsecase(ban.NewBanSteamRepository(databaseConn, personUC, networkUC), personUC, configUC, discordUC, reportUC, stateUC)
	authUC = auth.NewAuthUsecase(authRepo, configUC, personUC, banSteamUC, serversUC)
	banASNUC = ban.NewBanASNUsecase(ban.NewBanASNRepository(databaseConn), discordUC, networkUC, configUC, personUC)
	banGroupUC = steamgroup.NewBanGroupUsecase(steamgroup.NewSteamGroupRepository(databaseConn), personUC, discordUC, configUC)
	banNetUC = ban.NewBanNetUsecase(ban.NewBanNetRepository(databaseConn), personUC, configUC, discordUC, stateUC)

	matchUC = match.NewMatchUsecase(match.NewMatchRepository(eventBroadcaster, databaseConn, personUC, serversUC, discordUC, stateUC, weaponsMap), stateUC, serversUC, discordUC)
	chatUC = chat.NewChatUsecase(configUC, chat.NewChatRepository(databaseConn, personUC, wordFilterUC, matchUC, eventBroadcaster), wordFilterUC, stateUC, banSteamUC, personUC, discordUC)
	votesUC = votes.NewVoteUsecase(votes.NewVoteRepository(databaseConn), personUC, matchUC, discordUC, configUC, eventBroadcaster)

	container = dbContainer

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
	ban.NewBanHandler(router, banSteamUC, discordUC, personUC, configUC, authUC)
	ban.NewBanNetHandler(router, banNetUC, authUC)
	ban.NewBanASNHandler(router, banASNUC, authUC)
	news.NewNewsHandler(router, newsUC, discordUC, authUC)
	wiki.NewWIkiHandler(router, wikiUC, authUC)

	return router
}

func testEndpointWithReceiver(t *testing.T, router *gin.Engine, method string,
	path string, body any, expectedStatus int, tokens *domain.UserTokens, receiver any,
) {
	t.Helper()

	resp := testEndpoint(t, router, method, path, body, expectedStatus, tokens)
	if receiver != nil {
		if err := json.NewDecoder(resp.Body).Decode(&receiver); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
	}
}

func testEndpoint(t *testing.T, router *gin.Engine, method string, path string, body any, expectedStatus int, tokens *domain.UserTokens) *httptest.ResponseRecorder {
	t.Helper()

	reqCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
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

	request, errRequest := http.NewRequestWithContext(reqCtx, method, path, bodyReader)
	if errRequest != nil {
		t.Fatalf("Failed to make request: %v", errRequest)
	}

	if tokens != nil {
		request.AddCookie(&http.Cookie{
			Name:     domain.FingerprintCookieName,
			Value:    tokens.Fingerprint,
			Path:     "/api",
			Domain:   "example.com",
			Expires:  time.Now().AddDate(0, 0, 1),
			MaxAge:   0,
			Secure:   false,
			HttpOnly: false,
			SameSite: http.SameSiteStrictMode,
		})
		request.Header.Add("Authorization", "Bearer "+tokens.Access)
	}

	router.ServeHTTP(recorder, request)

	require.Equal(t, expectedStatus, recorder.Code, "Received invalid response code")

	return recorder
}

func createTestPerson(sid steamid.SteamID, level domain.Privilege) domain.Person {
	player, err := personUC.GetOrCreatePersonBySteamID(context.Background(), sid)
	if err != nil {
		panic(err)
	}

	player.PermissionLevel = level

	if errSave := personUC.SavePerson(context.Background(), &player); errSave != nil {
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
