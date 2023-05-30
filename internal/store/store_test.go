package store

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"
)

func TestMain(testMain *testing.M) {
	logger = zap.NewNop()

	tearDown := func() {
		q := `select 'drop table "' || tablename || '" cascade;' from pg_tables where schemaname = 'public';`
		if errMigrate := Exec(context.Background(), q); errMigrate != nil {
			logger.Error("Failed to migrate database down", zap.Error(errMigrate))
			os.Exit(2)
		}
	}

	_, _ = config.Read()
	config.General.Mode = config.TestMode
	testCtx := context.Background()

	Setup()
	dbStore, dbErr := New(testCtx, logger, config.DB.DSN)
	if dbErr != nil {
		logger.Error("Failed to setup store", zap.Error(dbErr))
		return
	}
	conn = dbStore
	tearDown(dbStore)
	defer util.LogClose(logger, dbStore)
	rc := testMain.Run()
	os.Exit(rc)
}

func TestServer(t *testing.T) {
	serverA := Server{
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
	require.NoError(t, testDatabase.SaveServer(ctx, &serverA))
	require.True(t, serverA.ServerID > 0)
	// Fetch saved server
	var s1Get Server
	require.NoError(t, testDatabase.GetServer(ctx, serverA.ServerID, &s1Get))
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
	sLenA, errGetServers := testDatabase.GetServers(ctx, false)
	require.NoError(t, errGetServers, "Failed to fetch enabled servers")
	require.True(t, len(sLenA) > 0, "Empty server results")
	// Delete a server
	require.NoError(t, testDatabase.DropServer(ctx, serverA.ServerID))
	var server Server
	require.True(t, errors.Is(testDatabase.GetServer(ctx, serverA.ServerID, &server), ErrNoResult))
	sLenB, _ := testDatabase.GetServers(ctx, false)
	require.True(t, len(sLenA)-1 == len(sLenB))
}

func randIP() string {
	return fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255))
}

func TestReport(t *testing.T) {
	var author Person
	require.NoError(t, testDatabase.GetOrCreatePersonBySteamID(context.TODO(), steamid.SID64(76561198003911389), &author))
	var target Person
	require.NoError(t, testDatabase.GetOrCreatePersonBySteamID(context.TODO(), steamid.RandSID64(), &target))
	report := NewReport()
	report.SourceId = author.SteamID
	report.TargetId = target.SteamID
	report.Description = golib.RandomString(120)
	require.NoError(t, testDatabase.SaveReport(context.TODO(), &report))

	msg1 := NewUserMessage(report.ReportId, author.SteamID, golib.RandomString(100))
	msg2 := NewUserMessage(report.ReportId, author.SteamID, golib.RandomString(100))
	require.NoError(t, testDatabase.SaveReportMessage(context.Background(), &msg1))
	require.NoError(t, testDatabase.SaveReportMessage(context.Background(), &msg2))
	msgs, msgsErr := testDatabase.GetReportMessages(context.Background(), report.ReportId)
	require.NoError(t, msgsErr)
	require.Equal(t, 2, len(msgs))
	require.NoError(t, testDatabase.DropReport(context.Background(), &report))
}

func TestBanNet(t *testing.T) {
	banNetEqual := func(b1, b2 BanCIDR) {
		require.Equal(t, b1.Reason, b2.Reason)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	rip := randIP()
	var banCidr BanCIDR
	require.NoError(t, NewBanCIDR(StringSID("76561198003911389"),
		"76561198044052046", "10m", Custom,
		"custom reason", "", System, fmt.Sprintf("%s/32", rip), Banned, &banCidr))
	require.NoError(t, testDatabase.SaveBanNet(ctx, &banCidr))
	require.Less(t, int64(0), banCidr.NetID)
	banNet, errGetBanNet := testDatabase.GetBanNetByAddress(ctx, net.ParseIP(rip))
	require.NoError(t, errGetBanNet)
	banNetEqual(banNet[0], banCidr)
	require.Equal(t, banNet[0].Reason, banCidr.Reason)
}

func TestBan(t *testing.T) {
	banEqual := func(ban1, ban2 *BanSteam) {
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

	var banSteam BanSteam
	require.NoError(t, NewBanSteam(
		StringSID("76561198003911389"),
		"76561198044052046",
		"1M",
		Cheating,
		Cheating.String(),
		"Mod Note",
		System, 0, Banned, &banSteam), "Failed to create ban opts")

	require.NoError(t, testDatabase.SaveBan(ctx, &banSteam), "Failed to add ban")
	b1Fetched := NewBannedPerson()
	require.NoError(t, testDatabase.GetBanBySteamID(ctx, 76561198044052046, &b1Fetched, false))
	banEqual(&banSteam, &b1Fetched.Ban)

	b1duplicate := banSteam
	b1duplicate.BanID = 0
	require.True(t, errors.Is(testDatabase.SaveBan(ctx, &b1duplicate), ErrDuplicate), "Was able to add duplicate ban")

	b1Fetched.Ban.SourceId = 76561198057999536
	b1Fetched.Ban.ReasonText = "test reason"
	b1Fetched.Ban.ValidUntil = config.Now().Add(time.Minute * 10)
	b1Fetched.Ban.Note = "test note"
	b1Fetched.Ban.Origin = Web
	require.NoError(t, testDatabase.SaveBan(ctx, &b1Fetched.Ban), "Failed to edit ban")
	b1FetchedUpdated := NewBannedPerson()
	require.NoError(t, testDatabase.GetBanBySteamID(ctx, 76561198044052046, &b1FetchedUpdated, false))
	banEqual(&b1Fetched.Ban, &b1FetchedUpdated.Ban)

	require.NoError(t, testDatabase.DropBan(ctx, &banSteam, false), "Failed to drop ban")
	vb := NewBannedPerson()
	errMissing := testDatabase.GetBanBySteamID(ctx, banSteam.TargetId, &vb, false)
	require.Error(t, errMissing)
	require.True(t, errors.Is(errMissing, ErrNoResult))
}

func TestFilteredWords(t *testing.T) {
	//
}
func randSID() steamid.SID64 {
	return steamid.SID64(76561197960265728 + rand.Int63n(100000000))
}

func TestPerson(t *testing.T) {
	p1 := NewPerson(randSID())
	p2 := NewPerson(randSID())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	require.NoError(t, testDatabase.SavePerson(ctx, &p1))
	p2Fetched := NewPerson(p2.SteamID)
	require.NoError(t, testDatabase.GetOrCreatePersonBySteamID(ctx, p2.SteamID, &p2Fetched))
	require.Equal(t, p2.SteamID, p2Fetched.SteamID)
	pBadID := NewPerson(0)
	require.Error(t, testDatabase.GetPersonBySteamID(ctx, 0, &pBadID))
	_, eH := testDatabase.GetPersonIPHistory(ctx, p1.SteamID, 1000)
	require.NoError(t, eH)
	require.NoError(t, testDatabase.DropPerson(ctx, p1.SteamID))
}

func TestGetChatHistory(t *testing.T) {
	//sid := steamid.SID64(76561198083950960)
	ctx := context.Background()
	s := NewServer(golib.RandomString(10), "localhost", rand.Intn(65535))
	require.NoError(t, testDatabase.SaveServer(ctx, &s))
	//player := model.Person{
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
	p1 := NewPerson(randSID())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	require.NoError(t, testDatabase.SavePerson(ctx, &p1))
	existingFilters, errGetFilters := testDatabase.GetFilters(context.Background())
	require.NoError(t, errGetFilters)
	words := []string{golib.RandomString(10), golib.RandomString(20)}
	var savedFilters []Filter
	for _, word := range words {
		filter := Filter{
			IsEnabled: true,
			IsRegex:   false,
			AuthorId:  p1.SteamID,
			Pattern:   word,
			UpdatedOn: config.Now(),
			CreatedOn: config.Now(),
		}
		require.NoError(t, testDatabase.SaveFilter(ctx, &filter), "Failed to insert filter: %s", word)
		require.True(t, filter.FilterID > 0)
		savedFilters = append(savedFilters, filter)
	}
	currentFilters, errGetCurrentFilters := testDatabase.GetFilters(ctx)
	require.NoError(t, errGetCurrentFilters)
	require.Equal(t, len(existingFilters)+len(words), len(currentFilters))
	if savedFilters != nil {
		require.NoError(t, testDatabase.DropFilter(ctx, &savedFilters[0]))
		var byId Filter
		require.NoError(t, testDatabase.GetFilterByID(ctx, savedFilters[1].FilterID, &byId))
		require.Equal(t, savedFilters[1].FilterID, byId.FilterID)
		require.Equal(t, savedFilters[1].Pattern, byId.Pattern)
	}
	droppedFilters, errGetDroppedFilters := testDatabase.GetFilters(ctx)
	require.NoError(t, errGetDroppedFilters)
	require.Equal(t, len(existingFilters)+len(words)-1, len(droppedFilters))

}

func TestBanASN(t *testing.T) {
	var author Person
	require.NoError(t, testDatabase.GetOrCreatePersonBySteamID(context.TODO(), steamid.SID64(76561198083950960), &author))
	var banASN BanASN
	require.NoError(t, NewBanASN(
		StringSID(author.SteamID.String()), "0",
		"10m", Cheating, "", "", System, rand.Int63n(23455), Banned, &banASN))

	require.NoError(t, testDatabase.SaveBanASN(context.Background(), &banASN))
	require.True(t, banASN.BanASNId > 0)

	var f1 BanASN
	require.NoError(t, testDatabase.GetBanASN(context.TODO(), banASN.ASNum, &f1))
	require.NoError(t, testDatabase.DropBanASN(context.TODO(), &f1))
	var d1 BanASN
	require.Error(t, testDatabase.GetBanASN(context.TODO(), banASN.ASNum, &d1))
}

func TestBanGroup(t *testing.T) {
	var banGroup BanGroup
	require.NoError(t, NewBanSteamGroup(
		StringSID("76561198083950960"),
		"0",
		"10m",
		Cheating,
		"",
		"",
		System,
		steamid.GID(int64(103000000000000000)+int64(rand.Int())),
		golib.RandomString(10),
		Banned,
		&banGroup))
	require.NoError(t, testDatabase.SaveBanGroup(context.TODO(), &banGroup))
	require.True(t, banGroup.BanGroupId > 0)
	var bgB BanGroup
	require.NoError(t, testDatabase.GetBanGroup(context.TODO(), banGroup.GroupId, &bgB))
	require.EqualValues(t, banGroup.BanGroupId, bgB.BanGroupId)
	require.NoError(t, testDatabase.DropBanGroup(context.TODO(), &banGroup))
	var bgDeleted BanGroup
	require.EqualError(t, ErrNoResult, testDatabase.GetBanGroup(context.TODO(), banGroup.GroupId, &bgDeleted).Error())
}
