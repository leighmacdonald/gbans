package store_test

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

func newTestDB(ctx context.Context) (string, *postgres.PostgresContainer, error) {
	const testInfo = "gbans-test"
	username, password, dbName := testInfo, testInfo, testInfo
	cont, errContainer := postgres.RunContainer(
		ctx,
		testcontainers.WithImage("docker.io/postgis/postgis:15-3.3"),
		postgres.WithDatabase(dbName),
		postgres.WithUsername(username),
		postgres.WithPassword(password),
		testcontainers.WithWaitStrategy(wait.
			ForLog("database system is ready to accept connections").
			WithOccurrence(2)),
	)

	if errContainer != nil {
		return "", nil, errors.Wrap(errContainer, "Failed to bring up test container")
	}

	port, _ := cont.MappedPort(ctx, "5432")
	dsn := fmt.Sprintf("postgresql://%s:%s@localhost:%s/%s", username, password, port.Port(), dbName)

	return dsn, cont, nil
}

func TestStore(t *testing.T) {
	logger := zap.NewNop()
	testCtx := context.Background()

	dsn, databaseContainer, errDB := newTestDB(testCtx)
	if errDB != nil {
		t.Skipf("Failed to bring up testcontainer db: %v", errDB)
	}

	database := store.New(logger, dsn, true, false)
	if dbErr := database.Connect(testCtx); dbErr != nil {
		logger.Fatal("Failed to setup store", zap.Error(dbErr))
	}

	t.Cleanup(func() {
		if errTerm := databaseContainer.Terminate(testCtx); errTerm != nil {
			t.Error("Failed to terminate test container")
		}
	})

	t.Run("server", testServerTest(database))
	t.Run("report", testReport(database))
	t.Run("ban_net", testBanNet(database))
	t.Run("ban_steam", testBanSteam(database))
	t.Run("ban_asn", testBanASN(database))
	t.Run("ban_group", testBanGroup(database))
	t.Run("person", testPerson(database))
	t.Run("chat_hist", testChatHistory(database))
	t.Run("filters", testFilters(database))
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"3s",
			"3s",
		},
		{
			"3m",
			"3m0s",
		},
		{
			"3h",
			"3h0m0s",
		},
		{
			"3d",
			"72h0m0s",
		},
		{
			"3w",
			"504h0m0s",
		},
		{
			"3M",
			"2232h0m0s",
		},
		{
			"3y",
			"26280h0m0s",
		},
	}
	for _, test := range tests {
		d, errParseDuration := store.ParseDuration(test.input)
		require.NoError(t, errParseDuration, "Failed to parse: %s", test.input)
		require.Equal(t, test.expected, d.String(), "Failed to parse: %s", test.input)
	}
}

func testServerTest(database *store.Store) func(t *testing.T) {
	return func(t *testing.T) {
		serverA := store.Server{
			ServerName:     fmt.Sprintf("test-%s", golib.RandomString(10)),
			Address:        "172.16.1.100",
			Port:           27015,
			RCON:           "test",
			Password:       "test",
			IsEnabled:      true,
			TokenCreatedOn: time.Now(),
			CreatedOn:      time.Now(),
			UpdatedOn:      time.Now(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		// Save new server
		require.NoError(t, database.SaveServer(ctx, &serverA))
		require.True(t, serverA.ServerID > 0)

		// Fetch saved server
		var s1Get store.Server

		require.NoError(t, database.GetServer(ctx, serverA.ServerID, &s1Get))
		require.Equal(t, serverA.ServerID, s1Get.ServerID)
		require.Equal(t, serverA.ServerName, s1Get.ServerName)
		require.Equal(t, serverA.Address, s1Get.Address)
		require.Equal(t, serverA.Port, s1Get.Port)
		require.Equal(t, serverA.RCON, s1Get.RCON)
		require.Equal(t, serverA.Password, s1Get.Password)
		require.Equal(t, serverA.TokenCreatedOn.Second(), s1Get.TokenCreatedOn.Second())
		require.Equal(t, serverA.CreatedOn.Second(), s1Get.CreatedOn.Second())
		require.Equal(t, serverA.UpdatedOn.Second(), s1Get.UpdatedOn.Second())

		// Fetch all enabled servers
		sLenA, errGetServers := database.GetServers(ctx, false)
		require.NoError(t, errGetServers, "Failed to fetch enabled servers")
		require.True(t, len(sLenA) > 0, "Empty server results")
		// Delete a server
		require.NoError(t, database.DropServer(ctx, serverA.ServerID))

		var server store.Server

		require.True(t, errors.Is(database.GetServer(ctx, serverA.ServerID, &server), store.ErrNoResult))

		sLenB, _ := database.GetServers(ctx, false)
		require.True(t, len(sLenA)-1 == len(sLenB))
	}
}

func randIP() string {
	return fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255)) //nolint:gosec
}

func testReport(database *store.Store) func(t *testing.T) {
	return func(t *testing.T) {
		var author store.Person

		require.NoError(t, database.GetOrCreatePersonBySteamID(context.TODO(), steamid.New(76561198003911389), &author))

		var target store.Person

		require.NoError(t, database.GetOrCreatePersonBySteamID(context.TODO(), steamid.RandSID64(), &target))

		report := store.NewReport()

		report.SourceID = author.SteamID
		report.TargetID = target.SteamID
		report.Description = golib.RandomString(120)

		require.NoError(t, database.SaveReport(context.TODO(), &report))

		msg1 := store.NewUserMessage(report.ReportID, author.SteamID, golib.RandomString(100))
		msg2 := store.NewUserMessage(report.ReportID, author.SteamID, golib.RandomString(100))

		require.NoError(t, database.SaveReportMessage(context.Background(), &msg1))
		require.NoError(t, database.SaveReportMessage(context.Background(), &msg2))

		msgs, msgsErr := database.GetReportMessages(context.Background(), report.ReportID)
		require.NoError(t, msgsErr)
		require.Equal(t, 2, len(msgs))
		require.NoError(t, database.DropReport(context.Background(), &report))
	}
}

func testBanNet(database *store.Store) func(t *testing.T) {
	return func(t *testing.T) {
		bgCtx := context.Background()
		banNetEqual := func(b1, b2 store.BanCIDR) {
			require.Equal(t, b1.Reason, b2.Reason)
		}

		ctx, cancel := context.WithTimeout(bgCtx, time.Second*10)
		defer cancel()

		rip := randIP()

		var banCidr store.BanCIDR

		require.NoError(t, store.NewBanCIDR(ctx, store.StringSID("76561198003911389"),
			"76561198044052046", "10m", store.Custom,
			"custom reason", "", store.System, fmt.Sprintf("%s/32", rip), store.Banned, &banCidr))
		require.NoError(t, database.SaveBanNet(ctx, &banCidr))
		require.Less(t, int64(0), banCidr.NetID)

		banNet, errGetBanNet := database.GetBanNetByAddress(ctx, net.ParseIP(rip))
		require.NoError(t, errGetBanNet)

		banNetEqual(banNet[0], banCidr)
		require.Equal(t, banNet[0].Reason, banCidr.Reason)
	}
}

func testBanSteam(database *store.Store) func(t *testing.T) {
	return func(t *testing.T) {
		bgCtx := context.Background()
		banEqual := func(ban1, ban2 *store.BanSteam) {
			require.Equal(t, ban1.BanID, ban2.BanID)
			require.Equal(t, ban1.SourceID, ban2.SourceID)
			require.Equal(t, ban1.Reason, ban2.Reason)
			require.Equal(t, ban1.ReasonText, ban2.ReasonText)
			require.Equal(t, ban1.BanType, ban2.BanType)
			require.Equal(t, ban1.Origin, ban2.Origin)
			require.Equal(t, ban1.Note, ban2.Note)
			require.True(t, ban2.ValidUntil.Unix() > 0)
			require.Equal(t, ban1.ValidUntil.Unix(), ban2.ValidUntil.Unix())
			require.Equal(t, ban1.CreatedOn.Unix(), ban2.CreatedOn.Unix())
			require.Equal(t, ban1.UpdatedOn.Unix(), ban2.UpdatedOn.Unix())
		}

		ctx, cancel := context.WithTimeout(bgCtx, time.Second*20)
		defer cancel()

		var banSteam store.BanSteam

		require.NoError(t, store.NewBanSteam(
			ctx,
			store.StringSID("76561198003911389"),
			"76561198044052046",
			"1M",
			store.Cheating,
			store.ReasonString(store.Cheating),
			"Mod Note",
			store.System, 0, store.Banned, &banSteam), "Failed to create ban opts")

		require.NoError(t, database.SaveBan(ctx, &banSteam), "Failed to add ban")

		b1Fetched := store.NewBannedPerson()
		require.NoError(t, database.GetBanBySteamID(ctx, steamid.New(76561198044052046), &b1Fetched, false))
		banEqual(&banSteam, &b1Fetched.Ban)

		b1duplicate := banSteam
		b1duplicate.BanID = 0
		require.True(t, errors.Is(database.SaveBan(ctx, &b1duplicate), store.ErrDuplicate), "Was able to add duplicate ban")

		b1Fetched.Ban.SourceID = steamid.New(76561198057999536)
		b1Fetched.Ban.ReasonText = "test reason"
		b1Fetched.Ban.ValidUntil = time.Now().Add(time.Minute * 10)
		b1Fetched.Ban.Note = "test note"
		b1Fetched.Ban.Origin = store.Web
		require.NoError(t, database.SaveBan(ctx, &b1Fetched.Ban), "Failed to edit ban")

		b1FetchedUpdated := store.NewBannedPerson()
		require.NoError(t, database.GetBanBySteamID(ctx, steamid.New(76561198044052046), &b1FetchedUpdated, false))
		banEqual(&b1Fetched.Ban, &b1FetchedUpdated.Ban)

		require.NoError(t, database.DropBan(ctx, &banSteam, false), "Failed to drop ban")

		vb := store.NewBannedPerson()
		errMissing := database.GetBanBySteamID(ctx, banSteam.TargetID, &vb, false)

		require.Error(t, errMissing)
		require.True(t, errors.Is(errMissing, store.ErrNoResult))
	}
}

func randSID() steamid.SID64 {
	return steamid.New(76561197960265728 + rand.Int63n(100000000)) //nolint:gosec
}

func testPerson(database *store.Store) func(t *testing.T) {
	return func(t *testing.T) {
		var (
			person1 = store.NewPerson(randSID())
			person2 = store.NewPerson(randSID())
		)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
		defer cancel()

		require.NoError(t, database.SavePerson(ctx, &person1))

		p2Fetched := store.NewPerson(person2.SteamID)
		require.NoError(t, database.GetOrCreatePersonBySteamID(ctx, person2.SteamID, &p2Fetched))
		require.Equal(t, person2.SteamID, p2Fetched.SteamID)

		pBadID := store.NewPerson("")
		require.Error(t, database.GetPersonBySteamID(ctx, "", &pBadID))

		_, errHistory := database.GetPersonIPHistory(ctx, person1.SteamID, 1000)
		require.NoError(t, errHistory)
		require.NoError(t, database.DropPerson(ctx, person1.SteamID))
	}
}

func testChatHistory(database *store.Store) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		newServer := store.NewServer(golib.RandomString(10), "localhost", rand.Intn(65535)) //nolint:gosec
		// player := model.Person{
		//	SteamID: sid,
		//	PlayerSummary: &steamweb.PlayerSummary{
		//		PersonaName: "test-name",
		//	},
		// }
		// logs := []model.ServerEvent{
		//	{
		//		Server:    &newServer,
		//		Source:    &player,
		//		EventType: logparse.Say,
		//		MetaData:  map[string]any{"msg": "test-1"},
		//		CreatedOn: config.Now().Add(-1 * time.Second),
		//	},
		//	{
		//		Server:    &newServer,
		//		Source:    &player,
		//		EventType: logparse.Say,
		//		MetaData:  map[string]any{"msg": "test-2"},
		//		CreatedOn: config.Now(),
		//	},
		// }
		// require.NoError(t, testDatabase.BatchInsertServerLogs(ctx, logs))
		// hist, errHist := testDatabase.GetChatHistory(ctx, sid, 100)
		// require.NoError(t, errHist, "Failed to fetch chat history")
		// require.True(t, len(hist) >= 2, "History size too small: %d", len(hist))
		// require.Equal(t, "test-2", hist[0].Msg)
		require.NoError(t, database.SaveServer(ctx, &newServer))
	}
}

func testFilters(database *store.Store) func(t *testing.T) {
	return func(t *testing.T) {
		player1 := store.NewPerson(randSID())

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		require.NoError(t, database.SavePerson(ctx, &player1))

		existingFilters, errGetFilters := database.GetFilters(context.Background())
		require.NoError(t, errGetFilters)

		var (
			words        = []string{golib.RandomString(10), golib.RandomString(20)}
			savedFilters = make([]store.Filter, len(words))
		)

		for index, word := range words {
			filter := store.Filter{
				IsEnabled: true,
				IsRegex:   false,
				AuthorID:  player1.SteamID,
				Pattern:   word,
				UpdatedOn: time.Now(),
				CreatedOn: time.Now(),
			}

			require.NoError(t, database.SaveFilter(ctx, &filter), "Failed to insert filter: %s", word)
			require.True(t, filter.FilterID > 0)
			savedFilters[index] = filter
		}

		currentFilters, errGetCurrentFilters := database.GetFilters(ctx)
		require.NoError(t, errGetCurrentFilters)
		require.Equal(t, len(existingFilters)+len(words), len(currentFilters))
		require.NoError(t, database.DropFilter(ctx, &savedFilters[0]))

		var byID store.Filter

		require.NoError(t, database.GetFilterByID(ctx, savedFilters[1].FilterID, &byID))
		require.Equal(t, savedFilters[1].FilterID, byID.FilterID)
		require.Equal(t, savedFilters[1].Pattern, byID.Pattern)

		droppedFilters, errGetDroppedFilters := database.GetFilters(ctx)
		require.NoError(t, errGetDroppedFilters)
		require.Equal(t, len(existingFilters)+len(words)-1, len(droppedFilters))
	}
}

func testBanASN(database *store.Store) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		var author store.Person

		require.NoError(t, database.GetOrCreatePersonBySteamID(ctx, steamid.New(76561198083950960), &author))

		var banASN store.BanASN

		require.NoError(t, store.NewBanASN(ctx,
			store.StringSID(author.SteamID.String()), "0",
			"10m", store.Cheating, "", "", store.System, rand.Int63n(23455), store.Banned, &banASN)) //nolint:gosec

		require.NoError(t, database.SaveBanASN(context.Background(), &banASN))
		require.True(t, banASN.BanASNId > 0)

		var banASN2 store.BanASN

		require.NoError(t, database.GetBanASN(context.TODO(), banASN.ASNum, &banASN2))
		require.NoError(t, database.DropBanASN(context.TODO(), &banASN2))

		var banASN3 store.BanASN

		require.Error(t, database.GetBanASN(context.TODO(), banASN.ASNum, &banASN3))
	}
}

func testBanGroup(database *store.Store) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		var banGroup store.BanGroup

		require.NoError(t, store.NewBanSteamGroup(
			ctx,
			store.StringSID("76561198083950960"),
			"0",
			"10m",
			store.Cheating,
			"",
			"",
			store.System,
			steamid.NewGID(103000000000000000+int64(rand.Int())), //nolint:gosec
			golib.RandomString(10),
			store.Banned,
			&banGroup))

		require.NoError(t, database.SaveBanGroup(context.TODO(), &banGroup))
		require.True(t, banGroup.BanGroupID > 0)

		var bgB store.BanGroup

		require.NoError(t, database.GetBanGroup(context.TODO(), banGroup.GroupID, &bgB))
		require.EqualValues(t, banGroup.BanGroupID, bgB.BanGroupID)
		require.NoError(t, database.DropBanGroup(context.TODO(), &banGroup))

		var bgDeleted store.BanGroup

		require.EqualError(t, store.ErrNoResult, database.GetBanGroup(context.TODO(), banGroup.GroupID, &bgDeleted).Error())
	}
}
