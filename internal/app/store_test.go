package app

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net"
	"strings"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestServer(t *testing.T) {
	serverA := model.Server{
		ServerNameShort: fmt.Sprintf("test-%s", golib.RandomString(10)),
		Token:           "",
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
	var s1Get model.Server
	require.NoError(t, testDatabase.GetServer(ctx, serverA.ServerID, &s1Get))
	require.Equal(t, serverA.ServerID, s1Get.ServerID)
	require.Equal(t, serverA.ServerNameShort, s1Get.ServerNameShort)
	require.Equal(t, serverA.Token, s1Get.Token)
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
	var server model.Server
	require.True(t, errors.Is(testDatabase.GetServer(ctx, serverA.ServerID, &server), store.ErrNoResult))
	sLenB, _ := testDatabase.GetServers(ctx, false)
	require.True(t, len(sLenA)-1 == len(sLenB))
}

func randIP() string {
	return fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255))
}

func TestReport(t *testing.T) {
	var author model.Person
	require.NoError(t, testDatabase.GetOrCreatePersonBySteamID(context.TODO(), steamid.SID64(76561198003911389), &author))
	var target model.Person
	require.NoError(t, testDatabase.GetOrCreatePersonBySteamID(context.TODO(),
		steamid.SID64(steamid.RandSID64().Int64()+int64(rand.Int())), &target))
	report := model.NewReport()
	report.AuthorId = author.SteamID
	report.ReportedId = target.SteamID
	report.Description = golib.RandomString(120)
	require.NoError(t, testDatabase.SaveReport(context.TODO(), &report))

	msg1 := model.NewUserMessage(report.ReportId, author.SteamID, golib.RandomString(100))
	msg2 := model.NewUserMessage(report.ReportId, author.SteamID, golib.RandomString(100))
	require.NoError(t, testDatabase.SaveReportMessage(context.Background(), &msg1))
	require.NoError(t, testDatabase.SaveReportMessage(context.Background(), &msg2))
	msgs, msgsErr := testDatabase.GetReportMessages(context.Background(), report.ReportId)
	require.NoError(t, msgsErr)
	require.Equal(t, 2, len(msgs))
	require.NoError(t, testDatabase.DropReport(context.Background(), &report))
}

func TestBanNet(t *testing.T) {
	banNetEqual := func(b1, b2 model.BanCIDR) {
		require.Equal(t, b1.Reason, b2.Reason)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	rip := randIP()
	var banCidr model.BanCIDR
	require.NoError(t, NewBanCIDR(model.StringSID("76561198003911389"),
		"76561198044052046", "10m", model.Custom,
		"", "", model.System, fmt.Sprintf("%s/32", rip), model.Banned, &banCidr))
	require.NoError(t, testDatabase.SaveBanNet(ctx, &banCidr))
	require.Less(t, int64(0), banCidr.NetID)
	banNet, errGetBanNet := testDatabase.GetBanNetByAddress(ctx, net.ParseIP(rip))
	require.NoError(t, errGetBanNet)
	banNetEqual(banNet[0], banCidr)
	require.Equal(t, banNet[0].Reason, banCidr.Reason)
}

func TestBan(t *testing.T) {
	banEqual := func(ban1, ban2 *model.BanSteam) {
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

	var banSteam model.BanSteam
	require.NoError(t, NewBanSteam(
		model.StringSID("76561198003911389"),
		"76561198044052046",
		"",
		model.Cheating,
		model.Cheating.String(),
		"Mod Note",
		model.System, 0, model.Banned, &banSteam), "Failed to create ban opts")

	require.NoError(t, testDatabase.SaveBan(ctx, &banSteam), "Failed to add ban")
	b1Fetched := model.NewBannedPerson()
	require.NoError(t, testDatabase.GetBanBySteamID(ctx, 76561198044052046, &b1Fetched, false))
	banEqual(&banSteam, &b1Fetched.Ban)

	b1duplicate := banSteam
	b1duplicate.BanID = 0
	require.True(t, errors.Is(testDatabase.SaveBan(ctx, &b1duplicate), store.ErrDuplicate), "Was able to add duplicate ban")

	b1Fetched.Ban.SourceId = 76561198057999536
	b1Fetched.Ban.ReasonText = "test reason"
	b1Fetched.Ban.ValidUntil = config.Now().Add(time.Minute * 10)
	b1Fetched.Ban.Note = "test note"
	b1Fetched.Ban.Origin = model.Web
	require.NoError(t, testDatabase.SaveBan(ctx, &b1Fetched.Ban), "Failed to edit ban")
	b1FetchedUpdated := model.NewBannedPerson()
	require.NoError(t, testDatabase.GetBanBySteamID(ctx, 76561198044052046, &b1FetchedUpdated, false))
	banEqual(&b1Fetched.Ban, &b1FetchedUpdated.Ban)

	require.NoError(t, testDatabase.DropBan(ctx, &banSteam, false), "Failed to drop ban")
	vb := model.NewBannedPerson()
	errMissing := testDatabase.GetBanBySteamID(ctx, banSteam.TargetId, &vb, false)
	require.Error(t, errMissing)
	require.True(t, errors.Is(errMissing, store.ErrNoResult))
}

func TestFilteredWords(t *testing.T) {
	//
}
func randSID() steamid.SID64 {
	return steamid.SID64(76561197960265728 + rand.Int63n(100000000))
}

func TestPerson(t *testing.T) {
	p1 := model.NewPerson(randSID())
	p2 := model.NewPerson(randSID())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	require.NoError(t, testDatabase.SavePerson(ctx, &p1))
	p2Fetched := model.NewPerson(p2.SteamID)
	require.NoError(t, testDatabase.GetOrCreatePersonBySteamID(ctx, p2.SteamID, &p2Fetched))
	require.Equal(t, p2.SteamID, p2Fetched.SteamID)
	pBadID := model.NewPerson(0)
	require.Error(t, testDatabase.GetPersonBySteamID(ctx, 0, &pBadID))
	_, eH := testDatabase.GetPersonIPHistory(ctx, p1.SteamID, 1000)
	require.NoError(t, eH)
	require.NoError(t, testDatabase.DropPerson(ctx, p1.SteamID))
}

func TestGetChatHistory(t *testing.T) {
	//sid := steamid.SID64(76561198083950960)
	ctx := context.Background()
	s := model.NewServer(golib.RandomString(10), "localhost", rand.Intn(65535))
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

func TestFindLogEvents(t *testing.T) {
	//sid := steamid.SID64(76561198083950960)
	//sid2 := steamid.SID64(76561198083950961)
	ctx := context.Background()
	s := model.NewServer(golib.RandomString(10), "localhost", rand.Intn(65535))
	require.NoError(t, testDatabase.SaveServer(ctx, &s))
	//s1 := model.Person{
	//	SteamID: sid,
	//	PlayerSummary: &steamweb.PlayerSummary{
	//		PersonaName: "test-name-1",
	//	},
	//}
	//t1 := model.Person{
	//	SteamID: sid2,
	//	PlayerSummary: &steamweb.PlayerSummary{
	//		PersonaName: "test-name-2",
	//	},
	//}
	//logs := []model.ServerEvent{
	//	{
	//		Server:    &s,
	//		Source:    &s1,
	//		EventType: logparse.Say,
	//		MetaData:  map[string]any{"msg": "test-1"},
	//	},
	//	{
	//		Server:    &s,
	//		Source:    &s1,
	//		EventType: logparse.Say,
	//		MetaData:  map[string]any{"msg": "test-2"},
	//	},
	//	{
	//		Server: &s,
	//		Source: &s1,
	//		Target: &t1,
	//		Weapon: logparse.Scattergun,
	//		AttackerPOS: logparse.Pos{
	//			X: 5,
	//			Y: -5,
	//			Z: 15,
	//		},
	//		VictimPOS: logparse.Pos{
	//			X: 10,
	//			Y: -10,
	//			Z: 100,
	//		},
	//		EventType: logparse.Killed,
	//	},
	//}
	//require.NoError(t, testDatabase.BatchInsertServerLogs(ctx, logs))
	//serverEvents, errLogs := testDatabase.FindLogEvents(ctx, model.LogQueryOpts{
	//	LogTypes: []logparse.EventType{logparse.Killed},
	//})
	//require.NoError(t, errLogs, "Failed to fetch logs")
	//require.True(t, len(serverEvents) >= 1, "Log size too small: %d", len(serverEvents))
	//for _, evt := range serverEvents {
	//	require.Equal(t, logparse.Killed, evt.EventType)
	//}
}

func TestFilters(t *testing.T) {
	existingFilters, errGetFilters := testDatabase.GetFilters(context.Background())
	require.NoError(t, errGetFilters)
	words := []string{golib.RandomString(10), golib.RandomString(20)}
	var savedFilters []model.Filter
	for wordIdx, word := range words {
		filter := model.Filter{
			FilterName: fmt.Sprintf("%d-%s", wordIdx, word),
			Patterns:   strings.Split(word, "||"),
			CreatedOn:  config.Now(),
		}
		require.NoError(t, testDatabase.SaveFilter(context.Background(), &filter), "Failed to insert filter: %s", word)
		require.True(t, filter.WordID > 0)
		savedFilters = append(savedFilters, filter)
	}
	currentFilters, errGetCurrentFilters := testDatabase.GetFilters(context.Background())
	require.NoError(t, errGetCurrentFilters)
	require.Equal(t, len(existingFilters)+len(words), len(currentFilters))
	if savedFilters != nil {
		require.NoError(t, testDatabase.DropFilter(context.Background(), &savedFilters[0]))
		var byId model.Filter
		require.NoError(t, testDatabase.GetFilterByID(context.Background(), savedFilters[1].WordID, &byId))
		require.Equal(t, savedFilters[1].WordID, byId.WordID)
		require.Equal(t, savedFilters[1].Patterns, byId.Patterns)
	}
	droppedFilters, errGetDroppedFilters := testDatabase.GetFilters(context.Background())
	require.NoError(t, errGetDroppedFilters)
	require.Equal(t, len(existingFilters)+len(words)-1, len(droppedFilters))

}

func TestBanASN(t *testing.T) {
	var author model.Person
	require.NoError(t, testDatabase.GetOrCreatePersonBySteamID(context.TODO(), steamid.SID64(76561198083950960), &author))
	var banASN model.BanASN
	require.NoError(t, NewBanASN(
		model.StringSID(author.SteamID.String()), "0",
		"10m", model.Cheating, "", "", model.System, 200, model.Banned, &banASN))

	require.NoError(t, testDatabase.SaveBanASN(context.Background(), &banASN))
	require.True(t, banASN.BanASNId > 0)

	var f1 model.BanASN
	require.NoError(t, testDatabase.GetBanASN(context.TODO(), banASN.ASNum, &f1))
	require.NoError(t, testDatabase.DropBanASN(context.TODO(), &f1))
	var d1 model.BanASN
	require.Error(t, testDatabase.GetBanASN(context.TODO(), banASN.ASNum, &d1))
}

func TestBanGroup(t *testing.T) {
	var banGroup model.BanGroup
	require.NoError(t, NewBanSteamGroup(
		model.StringSID("76561198083950960"),
		"",
		"10m",
		model.Cheating,
		"",
		"",
		model.System,
		steamid.GID(int64(103000000000000000)+int64(rand.Int())),
		golib.RandomString(10),
		model.Banned,
		&banGroup))
	require.NoError(t, testDatabase.SaveBanGroup(context.TODO(), &banGroup))
	require.True(t, banGroup.BanGroupId > 0)
	var bgB model.BanGroup
	require.NoError(t, testDatabase.GetBanGroup(context.TODO(), banGroup.GroupId, &bgB))
	require.EqualValues(t, banGroup.BanGroupId, bgB.BanGroupId)
	require.NoError(t, testDatabase.DropBanGroup(context.TODO(), &banGroup))
	var bgDeleted model.BanGroup
	require.EqualError(t, store.ErrNoResult, testDatabase.GetBanGroup(context.TODO(), banGroup.GroupId, &bgDeleted).Error())
}
