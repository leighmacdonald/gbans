package store_test

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/store"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var logger *zap.Logger

func TestMain(testMain *testing.M) {
	logger = zap.NewNop()

	tearDown := func() {
		defer func() { _ = store.Close() }()
		q := `select 'drop table "' || tablename || '" cascade;' from pg_tables where schemaname = 'public';`
		if errMigrate := store.Exec(context.Background(), q); errMigrate != nil {
			logger.Error("Failed to migrate database down", zap.Error(errMigrate))
			os.Exit(2)
		}
	}

	_, errConfig := config.Read()
	if errConfig != nil {
		return
	}
	config.General.Mode = config.TestMode
	testCtx := context.Background()
	if dbErr := store.Init(testCtx, logger); dbErr != nil {
		logger.Fatal("Failed to setup store", zap.Error(dbErr))
	}
	defer tearDown()
	rc := testMain.Run()

	os.Exit(rc)
}

func TestServer(t *testing.T) {
	serverA := store.Server{
		ServerNameShort: fmt.Sprintf("test-%s", golib.RandomString(10)),
		Address:         "172.16.1.100",
		Port:            27015,
		RCON:            "test",
		Password:        "test",
		IsEnabled:       true,
		TokenCreatedOn:  config.Now(),
		CreatedOn:       config.Now(),
		UpdatedOn:       config.Now(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	// Save new server
	require.NoError(t, store.SaveServer(ctx, &serverA))
	require.True(t, serverA.ServerID > 0)
	// Fetch saved server
	var s1Get store.Server
	require.NoError(t, store.GetServer(ctx, serverA.ServerID, &s1Get))
	require.Equal(t, serverA.ServerID, s1Get.ServerID)
	require.Equal(t, serverA.ServerNameShort, s1Get.ServerNameShort)
	require.Equal(t, serverA.Address, s1Get.Address)
	require.Equal(t, serverA.Port, s1Get.Port)
	require.Equal(t, serverA.RCON, s1Get.RCON)
	require.Equal(t, serverA.Password, s1Get.Password)
	require.Equal(t, serverA.TokenCreatedOn.Second(), s1Get.TokenCreatedOn.Second())
	require.Equal(t, serverA.CreatedOn.Second(), s1Get.CreatedOn.Second())
	require.Equal(t, serverA.UpdatedOn.Second(), s1Get.UpdatedOn.Second())
	// Fetch all enabled servers
	sLenA, errGetServers := store.GetServers(ctx, false)
	require.NoError(t, errGetServers, "Failed to fetch enabled servers")
	require.True(t, len(sLenA) > 0, "Empty server results")
	// Delete a server
	require.NoError(t, store.DropServer(ctx, serverA.ServerID))
	var server store.Server
	require.True(t, errors.Is(store.GetServer(ctx, serverA.ServerID, &server), store.ErrNoResult))
	sLenB, _ := store.GetServers(ctx, false)
	require.True(t, len(sLenA)-1 == len(sLenB))
}

func randIP() string {
	return fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255)) //nolint:gosec
}

func TestReport(t *testing.T) {
	var author store.Person
	require.NoError(t, store.GetOrCreatePersonBySteamID(context.TODO(), steamid.SID64(76561198003911389), &author))
	var target store.Person
	require.NoError(t, store.GetOrCreatePersonBySteamID(context.TODO(), steamid.RandSID64(), &target))
	report := store.NewReport()
	report.SourceId = author.SteamID
	report.TargetId = target.SteamID
	report.Description = golib.RandomString(120)
	require.NoError(t, store.SaveReport(context.TODO(), &report))

	msg1 := store.NewUserMessage(report.ReportId, author.SteamID, golib.RandomString(100))
	msg2 := store.NewUserMessage(report.ReportId, author.SteamID, golib.RandomString(100))
	require.NoError(t, store.SaveReportMessage(context.Background(), &msg1))
	require.NoError(t, store.SaveReportMessage(context.Background(), &msg2))
	msgs, msgsErr := store.GetReportMessages(context.Background(), report.ReportId)
	require.NoError(t, msgsErr)
	require.Equal(t, 2, len(msgs))
	require.NoError(t, store.DropReport(context.Background(), &report))
}

func TestBanNet(t *testing.T) {
	banNetEqual := func(b1, b2 store.BanCIDR) {
		require.Equal(t, b1.Reason, b2.Reason)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	rip := randIP()
	var banCidr store.BanCIDR
	require.NoError(t, store.NewBanCIDR(store.StringSID("76561198003911389"),
		"76561198044052046", "10m", store.Custom,
		"custom reason", "", store.System, fmt.Sprintf("%s/32", rip), store.Banned, &banCidr))
	require.NoError(t, store.SaveBanNet(ctx, &banCidr))
	require.Less(t, int64(0), banCidr.NetID)
	banNet, errGetBanNet := store.GetBanNetByAddress(ctx, net.ParseIP(rip))
	require.NoError(t, errGetBanNet)
	banNetEqual(banNet[0], banCidr)
	require.Equal(t, banNet[0].Reason, banCidr.Reason)
}

func TestBan(t *testing.T) {
	banEqual := func(ban1, ban2 *store.BanSteam) {
		require.Equal(t, ban1.BanID, ban2.BanID)
		require.Equal(t, ban1.SourceId, ban2.SourceId)
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	var banSteam store.BanSteam
	require.NoError(t, store.NewBanSteam(
		store.StringSID("76561198003911389"),
		"76561198044052046",
		"1M",
		store.Cheating,
		store.Cheating.String(),
		"Mod Note",
		store.System, 0, store.Banned, &banSteam), "Failed to create ban opts")

	require.NoError(t, store.SaveBan(ctx, &banSteam), "Failed to add ban")
	b1Fetched := store.NewBannedPerson()
	require.NoError(t, store.GetBanBySteamID(ctx, 76561198044052046, &b1Fetched, false))
	banEqual(&banSteam, &b1Fetched.Ban)

	b1duplicate := banSteam
	b1duplicate.BanID = 0
	require.True(t, errors.Is(store.SaveBan(ctx, &b1duplicate), store.ErrDuplicate), "Was able to add duplicate ban")

	b1Fetched.Ban.SourceId = 76561198057999536
	b1Fetched.Ban.ReasonText = "test reason"
	b1Fetched.Ban.ValidUntil = config.Now().Add(time.Minute * 10)
	b1Fetched.Ban.Note = "test note"
	b1Fetched.Ban.Origin = store.Web
	require.NoError(t, store.SaveBan(ctx, &b1Fetched.Ban), "Failed to edit ban")
	b1FetchedUpdated := store.NewBannedPerson()
	require.NoError(t, store.GetBanBySteamID(ctx, 76561198044052046, &b1FetchedUpdated, false))
	banEqual(&b1Fetched.Ban, &b1FetchedUpdated.Ban)

	require.NoError(t, store.DropBan(ctx, &banSteam, false), "Failed to drop ban")
	vb := store.NewBannedPerson()
	errMissing := store.GetBanBySteamID(ctx, banSteam.TargetId, &vb, false)
	require.Error(t, errMissing)
	require.True(t, errors.Is(errMissing, store.ErrNoResult))
}

func randSID() steamid.SID64 {
	return steamid.SID64(76561197960265728 + rand.Int63n(100000000)) //nolint:gosec
}

func TestPerson(t *testing.T) {
	p1 := store.NewPerson(randSID())
	p2 := store.NewPerson(randSID())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	require.NoError(t, store.SavePerson(ctx, &p1))
	p2Fetched := store.NewPerson(p2.SteamID)
	require.NoError(t, store.GetOrCreatePersonBySteamID(ctx, p2.SteamID, &p2Fetched))
	require.Equal(t, p2.SteamID, p2Fetched.SteamID)
	pBadID := store.NewPerson(0)
	require.Error(t, store.GetPersonBySteamID(ctx, 0, &pBadID))
	_, eH := store.GetPersonIPHistory(ctx, p1.SteamID, 1000)
	require.NoError(t, eH)
	require.NoError(t, store.DropPerson(ctx, p1.SteamID))
}

func TestGetChatHistory(t *testing.T) {
	ctx := context.Background()
	s := store.NewServer(golib.RandomString(10), "localhost", rand.Intn(65535))
	require.NoError(t, store.SaveServer(ctx, &s))
	// player := model.Person{
	//	SteamID: sid,
	//	PlayerSummary: &steamweb.PlayerSummary{
	//		PersonaName: "test-name",
	//	},
	//}
	//logs := []model.ServerEvent{
	//	{
	//		Server:    &s,
	//		Source:    &player,
	//		EventType: logparse.Say,
	//		MetaData:  map[string]any{"msg": "test-1"},
	//		CreatedOn: config.Now().Add(-1 * time.Second),
	//	},
	//	{
	//		Server:    &s,
	//		Source:    &player,
	//		EventType: logparse.Say,
	//		MetaData:  map[string]any{"msg": "test-2"},
	//		CreatedOn: config.Now(),
	//	},
	//}
	//require.NoError(t, testDatabase.BatchInsertServerLogs(ctx, logs))
	//hist, errHist := testDatabase.GetChatHistory(ctx, sid, 100)
	//require.NoError(t, errHist, "Failed to fetch chat history")
	//require.True(t, len(hist) >= 2, "History size too small: %d", len(hist))
	//require.Equal(t, "test-2", hist[0].Msg)
}

func TestFilters(t *testing.T) {
	p1 := store.NewPerson(randSID())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	require.NoError(t, store.SavePerson(ctx, &p1))
	existingFilters, errGetFilters := store.GetFilters(context.Background())
	require.NoError(t, errGetFilters)
	words := []string{golib.RandomString(10), golib.RandomString(20)}
	var savedFilters []store.Filter
	for _, word := range words {
		filter := store.Filter{
			IsEnabled: true,
			IsRegex:   false,
			AuthorId:  p1.SteamID,
			Pattern:   word,
			UpdatedOn: config.Now(),
			CreatedOn: config.Now(),
		}
		require.NoError(t, store.SaveFilter(ctx, &filter), "Failed to insert filter: %s", word)
		require.True(t, filter.FilterID > 0)
		savedFilters = append(savedFilters, filter)
	}
	currentFilters, errGetCurrentFilters := store.GetFilters(ctx)
	require.NoError(t, errGetCurrentFilters)
	require.Equal(t, len(existingFilters)+len(words), len(currentFilters))
	if savedFilters != nil {
		require.NoError(t, store.DropFilter(ctx, &savedFilters[0]))
		var byId store.Filter
		require.NoError(t, store.GetFilterByID(ctx, savedFilters[1].FilterID, &byId))
		require.Equal(t, savedFilters[1].FilterID, byId.FilterID)
		require.Equal(t, savedFilters[1].Pattern, byId.Pattern)
	}
	droppedFilters, errGetDroppedFilters := store.GetFilters(ctx)
	require.NoError(t, errGetDroppedFilters)
	require.Equal(t, len(existingFilters)+len(words)-1, len(droppedFilters))
}

func TestBanASN(t *testing.T) {
	var author store.Person
	require.NoError(t, store.GetOrCreatePersonBySteamID(context.TODO(), steamid.SID64(76561198083950960), &author))
	var banASN store.BanASN
	require.NoError(t, store.NewBanASN(
		store.StringSID(author.SteamID.String()), "0",
		"10m", store.Cheating, "", "", store.System, rand.Int63n(23455), store.Banned, &banASN))

	require.NoError(t, store.SaveBanASN(context.Background(), &banASN))
	require.True(t, banASN.BanASNId > 0)

	var f1 store.BanASN
	require.NoError(t, store.GetBanASN(context.TODO(), banASN.ASNum, &f1))
	require.NoError(t, store.DropBanASN(context.TODO(), &f1))
	var d1 store.BanASN
	require.Error(t, store.GetBanASN(context.TODO(), banASN.ASNum, &d1))
}

func TestBanGroup(t *testing.T) {
	var banGroup store.BanGroup
	require.NoError(t, store.NewBanSteamGroup(
		store.StringSID("76561198083950960"),
		"0",
		"10m",
		store.Cheating,
		"",
		"",
		store.System,
		steamid.GID(int64(103000000000000000)+int64(rand.Int())),
		golib.RandomString(10),
		store.Banned,
		&banGroup))
	require.NoError(t, store.SaveBanGroup(context.TODO(), &banGroup))
	require.True(t, banGroup.BanGroupId > 0)
	var bgB store.BanGroup
	require.NoError(t, store.GetBanGroup(context.TODO(), banGroup.GroupId, &bgB))
	require.EqualValues(t, banGroup.BanGroupId, bgB.BanGroupId)
	require.NoError(t, store.DropBanGroup(context.TODO(), &banGroup))
	var bgDeleted store.BanGroup
	require.EqualError(t, store.ErrNoResult, store.GetBanGroup(context.TODO(), banGroup.GroupId, &bgDeleted).Error())
}
