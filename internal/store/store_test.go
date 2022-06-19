package store

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/golib"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net"
	"os"
	"regexp"
	"testing"
	"time"
)

var (
	testDatabase Store
)

func TestMain(testMain *testing.M) {
	tearDown := func(database Store) {
		if errMigrate := database.Migrate(MigrateDn); errMigrate != nil {
			log.Errorf("Failed to migrate database down: %v", errMigrate)
			os.Exit(2)
		}
	}
	config.Read()
	config.General.Mode = config.TestMode
	ctx := context.Background()
	database, errNew := New(ctx, config.DB.DSN)
	if errNew != nil {
		log.Errorf("Failed to connect to test database: %v", errNew)
		os.Exit(1)
	}
	defer tearDown(database)
	tearDown(database) // Cleanup any existing tables in case of unclean shutdown
	if errMigrate := database.Migrate(MigrateUp); errMigrate != nil {
		log.Errorf("Failed to migrate database up: %v", errMigrate)
		os.Exit(2)
	}
	testDatabase = database
	os.Exit(testMain.Run())
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
	require.True(t, errors.Is(testDatabase.GetServer(ctx, serverA.ServerID, &server), ErrNoResult))
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
	require.NoError(t, testDatabase.GetOrCreatePersonBySteamID(context.TODO(), steamid.SID64(76561198083950960), &target))
	report := model.NewReport()
	report.AuthorId = author.SteamID
	report.ReportedId = target.SteamID
	report.Title = "test"
	report.Description = "test"
	require.NoError(t, testDatabase.SaveReport(context.TODO(), &report))

	media1 := model.NewReportMedia(report.ReportId)
	val1, errDecode1 := base64.URLEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABAQMAAAAl21bKAAAAA1BMVEUAAACnej3aAAAAAXRSTlMAQObYZgAAAApJREFUCNdjYAAAAAIAAeIhvDMAAAAASUVORK5CYII=")
	require.NoError(t, errDecode1)
	media1.Contents = val1
	media1.MimeType = "image/png"
	media1.Size = 95
	media1.AuthorId = author.SteamID
	require.Equal(t, media1.Size, int64(len(media1.Contents)))

	var2, errDecode2 := base64.URLEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAAAXNSR0IArs4c6QAAAARnQU1BAACxjwv8YQUAAAAJcEhZcwAADsMAAA7DAcdvqGQAAAANSURBVBhXY7D3OPMfAARwAlO7vhiUAAAAAElFTkSuQmCC")
	require.NoError(t, errDecode2)
	media2 := model.NewReportMedia(report.ReportId)
	media2.Contents = var2
	media2.MimeType = "image/png"
	media2.Size = 120
	media2.AuthorId = author.SteamID
	require.Equal(t, media2.Size, int64(len(media2.Contents)))

	require.NoError(t, testDatabase.SaveReportMedia(context.Background(), report.ReportId, &media1))
	require.NoError(t, testDatabase.SaveReportMedia(context.Background(), report.ReportId, &media2))

	msg1 := model.NewReportMessage(report.ReportId, author.SteamID, golib.RandomString(100))
	msg2 := model.NewReportMessage(report.ReportId, author.SteamID, golib.RandomString(100))
	require.NoError(t, testDatabase.SaveReportMessage(context.Background(), report.ReportId, &msg1))
	require.NoError(t, testDatabase.SaveReportMessage(context.Background(), report.ReportId, &msg2))
	msgs, msgsErr := testDatabase.GetReportMessages(context.Background(), report.ReportId)
	require.NoError(t, msgsErr)
	require.Equal(t, 2, len(msgs))
	require.NoError(t, testDatabase.DropReport(context.Background(), &report))
}

func TestBanNet(t *testing.T) {
	banNetEqual := func(b1, b2 model.BanNet) {
		require.Equal(t, b1.Reason, b2.Reason)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	rip := randIP()
	n1, _ := model.NewBanNet(fmt.Sprintf("%s/32", rip), "testing", time.Hour*100, model.System)
	require.NoError(t, testDatabase.SaveBanNet(ctx, &n1))
	require.Less(t, int64(0), n1.NetID)
	banNet, errGetBanNet := testDatabase.GetBanNet(ctx, net.ParseIP(rip))
	require.NoError(t, errGetBanNet)
	banNetEqual(banNet[0], n1)
	require.Equal(t, banNet[0].Reason, n1.Reason)
}

func TestBan(t *testing.T) {
	banEqual := func(ban1, ban2 *model.Ban) {
		require.Equal(t, ban1.BanID, ban2.BanID)
		require.Equal(t, ban1.AuthorID, ban2.AuthorID)
		require.Equal(t, ban1.Reason, ban2.Reason)
		require.Equal(t, ban1.ReasonText, ban2.ReasonText)
		require.Equal(t, ban1.BanType, ban2.BanType)
		require.Equal(t, ban1.Source, ban2.Source)
		require.Equal(t, ban1.Note, ban2.Note)
		require.True(t, ban2.ValidUntil.Unix() > 0)
		require.Equal(t, ban1.ValidUntil.Unix(), ban2.ValidUntil.Unix())
		require.Equal(t, ban1.CreatedOn.Unix(), ban2.CreatedOn.Unix())
		require.Equal(t, ban1.UpdatedOn.Unix(), ban2.UpdatedOn.Unix())
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	b1 := model.NewBan(76561198044052046, 76561198003911389, time.Hour*24)
	require.NoError(t, testDatabase.SaveBan(ctx, &b1), "Failed to add ban")
	b1Fetched := model.NewBannedPerson()
	require.NoError(t, testDatabase.GetBanBySteamID(ctx, 76561198044052046, false, &b1Fetched))
	banEqual(&b1, &b1Fetched.Ban)

	b1duplicate := model.NewBan(76561198044052046, 76561198003911389, time.Hour*24)
	require.True(t, errors.Is(testDatabase.SaveBan(ctx, &b1duplicate), ErrDuplicate), "Was able to add duplicate ban")

	b1Fetched.Ban.AuthorID = 76561198057999536
	b1Fetched.Ban.ReasonText = "test reason"
	b1Fetched.Ban.ValidUntil = config.Now().Add(time.Minute * 10)
	b1Fetched.Ban.Note = "test note"
	b1Fetched.Ban.Source = model.Web
	require.NoError(t, testDatabase.SaveBan(ctx, &b1Fetched.Ban), "Failed to edit ban")
	b1FetchedUpdated := model.NewBannedPerson()
	require.NoError(t, testDatabase.GetBanBySteamID(ctx, 76561198044052046, false, &b1FetchedUpdated))
	banEqual(&b1Fetched.Ban, &b1FetchedUpdated.Ban)

	require.NoError(t, testDatabase.DropBan(ctx, &b1, false), "Failed to drop ban")
	vb := model.NewBannedPerson()
	errMissing := testDatabase.GetBanBySteamID(ctx, b1.SteamID, false, &vb)
	require.Error(t, errMissing)
	require.True(t, errors.Is(errMissing, ErrNoResult))
}

func TestFilteredWords(t *testing.T) {
	//
}
func randSID() steamid.SID64 {
	return steamid.SID64(76561197960265728 + rand.Int63n(100000000))
}

func TestAppeal(t *testing.T) {
	b1 := model.NewBan(randSID(), 76561198003911389, time.Hour*24)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1000)
	defer cancel()
	require.NoError(t, testDatabase.SaveBan(ctx, &b1), "Failed to add ban")
	appeal := model.Appeal{
		BanID:       b1.BanID,
		AppealText:  "Im a nerd",
		AppealState: model.ASNew,
		Email:       "",
	}
	require.NoError(t, testDatabase.SaveAppeal(ctx, &appeal), "failed to save appeal")
	require.True(t, appeal.AppealID > 0, "No appeal id set")
	appeal.AppealState = model.ASDenied
	appeal.Email = "test@test.com"
	require.NoError(t, testDatabase.SaveAppeal(ctx, &appeal), "failed to update appeal")
	var fetched model.Appeal
	require.NoError(t, testDatabase.GetAppeal(ctx, b1.BanID, &fetched), "failed to get appeal")
	require.Equal(t, appeal.BanID, fetched.BanID)
	require.Equal(t, appeal.Email, fetched.Email)
	require.Equal(t, appeal.AppealState, fetched.AppealState)
	require.Equal(t, appeal.AppealID, fetched.AppealID)
	require.Equal(t, appeal.AppealText, fetched.AppealText)
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
	sid := steamid.SID64(76561198083950960)
	ctx := context.Background()
	s := model.NewServer(golib.RandomString(10), "localhost", rand.Intn(65535))
	require.NoError(t, testDatabase.SaveServer(ctx, &s))
	player := model.Person{
		SteamID: sid,
		PlayerSummary: &steamweb.PlayerSummary{
			PersonaName: "test-name",
		},
	}
	logs := []model.ServerEvent{
		{
			Server:    &s,
			Source:    &player,
			EventType: logparse.Say,
			MetaData:  map[string]any{"msg": "test-1"},
			CreatedOn: config.Now().Add(-1 * time.Second),
		},
		{
			Server:    &s,
			Source:    &player,
			EventType: logparse.Say,
			MetaData:  map[string]any{"msg": "test-2"},
			CreatedOn: config.Now(),
		},
	}
	require.NoError(t, testDatabase.BatchInsertServerLogs(ctx, logs))
	hist, errHist := testDatabase.GetChatHistory(ctx, sid, 100)
	require.NoError(t, errHist, "Failed to fetch chat history")
	require.True(t, len(hist) >= 2, "History size too small: %d", len(hist))
	require.Equal(t, "test-2", hist[0].Msg)
}

func TestFindLogEvents(t *testing.T) {
	sid := steamid.SID64(76561198083950960)
	sid2 := steamid.SID64(76561198083950961)
	ctx := context.Background()
	s := model.NewServer(golib.RandomString(10), "localhost", rand.Intn(65535))
	require.NoError(t, testDatabase.SaveServer(ctx, &s))
	s1 := model.Person{
		SteamID: sid,
		PlayerSummary: &steamweb.PlayerSummary{
			PersonaName: "test-name-1",
		},
	}
	t1 := model.Person{
		SteamID: sid2,
		PlayerSummary: &steamweb.PlayerSummary{
			PersonaName: "test-name-2",
		},
	}
	logs := []model.ServerEvent{
		{
			Server:    &s,
			Source:    &s1,
			EventType: logparse.Say,
			MetaData:  map[string]any{"msg": "test-1"},
		},
		{
			Server:    &s,
			Source:    &s1,
			EventType: logparse.Say,
			MetaData:  map[string]any{"msg": "test-2"},
		},
		{
			Server: &s,
			Source: &s1,
			Target: &t1,
			Weapon: logparse.Scattergun,
			AttackerPOS: logparse.Pos{
				X: 5,
				Y: -5,
				Z: 15,
			},
			VictimPOS: logparse.Pos{
				X: 10,
				Y: -10,
				Z: 100,
			},
			EventType: logparse.Killed,
		},
	}
	require.NoError(t, testDatabase.BatchInsertServerLogs(ctx, logs))
	serverEvents, errLogs := testDatabase.FindLogEvents(ctx, model.LogQueryOpts{
		LogTypes: []logparse.EventType{logparse.Killed},
	})
	require.NoError(t, errLogs, "Failed to fetch logs")
	require.True(t, len(serverEvents) >= 1, "Log size too small: %d", len(serverEvents))
	for _, evt := range serverEvents {
		require.Equal(t, logparse.Killed, evt.EventType)
	}
}

func TestFilters(t *testing.T) {
	existingFilters, errGetFilters := testDatabase.GetFilters(context.Background())
	require.NoError(t, errGetFilters)
	words := []string{golib.RandomString(10), golib.RandomString(10)}
	var savedFilters []model.Filter
	for _, word := range words {
		filter := model.Filter{
			Pattern:   regexp.MustCompile(word),
			CreatedOn: config.Now(),
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
		require.Equal(t, savedFilters[1].Pattern.String(), byId.Pattern.String())
	}
	droppedFilters, errGetDroppedFilters := testDatabase.GetFilters(context.Background())
	require.NoError(t, errGetDroppedFilters)
	require.Equal(t, len(existingFilters)+len(words)-1, len(droppedFilters))

}

func TestBanASN(t *testing.T) {
	var author model.Person
	require.NoError(t, testDatabase.GetOrCreatePersonBySteamID(context.TODO(), steamid.SID64(76561198083950960), &author))
	banASN := model.NewBanASN(1, author.SteamID, "test", time.Minute*10)
	require.NoError(t, testDatabase.SaveBanASN(context.Background(), &banASN))
	require.True(t, banASN.BanASNId > 0)
	var f1 model.BanASN
	require.NoError(t, testDatabase.GetBanASN(context.TODO(), banASN.ASNum, &f1))
	require.NoError(t, testDatabase.DropBanASN(context.TODO(), &f1))
	var d1 model.BanASN
	require.Error(t, testDatabase.GetBanASN(context.TODO(), banASN.ASNum, &d1))
}
