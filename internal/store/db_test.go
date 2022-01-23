package store

import (
	"context"
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
	testDb Store
)

func TestMain(m *testing.M) {
	tearDown := func(s Store) {
		if errM := s.Migrate(MigrateDn); errM != nil {
			log.Errorf("Failed to migrate db down: %v", errM)
			os.Exit(2)
		}
	}
	config.Read()
	config.General.Mode = config.TestMode
	db, err := New(config.DB.DSN)
	if err != nil {
		log.Errorf("Failed to connect to test database: %v", err)
		os.Exit(1)
	}
	defer tearDown(db)
	tearDown(db)
	if errM := db.Migrate(MigrateUp); errM != nil {
		log.Errorf("Failed to migrate db up: %v", errM)
		os.Exit(2)
	}
	testDb = db
	os.Exit(m.Run())
}

func TestServer(t *testing.T) {
	s1 := model.Server{
		ServerName:     fmt.Sprintf("test-%s", golib.RandomString(10)),
		Token:          "",
		Address:        "172.16.1.100",
		Port:           27015,
		RCON:           "test",
		Password:       "test",
		IsEnabled:      true,
		TokenCreatedOn: config.Now(),
		CreatedOn:      config.Now(),
		UpdatedOn:      config.Now(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	// Save new server
	require.NoError(t, testDb.SaveServer(ctx, &s1))
	require.True(t, s1.ServerID > 0)
	// Fetch saved server
	var s1Get model.Server
	require.NoError(t, testDb.GetServer(ctx, s1.ServerID, &s1Get))
	require.Equal(t, s1.ServerID, s1Get.ServerID)
	require.Equal(t, s1.ServerName, s1Get.ServerName)
	require.Equal(t, s1.Token, s1Get.Token)
	require.Equal(t, s1.Address, s1Get.Address)
	require.Equal(t, s1.Port, s1Get.Port)
	require.Equal(t, s1.RCON, s1Get.RCON)
	require.Equal(t, s1.Password, s1Get.Password)
	require.Equal(t, s1.TokenCreatedOn.Second(), s1Get.TokenCreatedOn.Second())
	require.Equal(t, s1.CreatedOn.Second(), s1Get.CreatedOn.Second())
	require.Equal(t, s1.UpdatedOn.Second(), s1Get.UpdatedOn.Second())
	// Fetch all enabled servers
	sLenA, eS := testDb.GetServers(ctx, false)
	require.NoError(t, eS, "Failed to fetch enabled servers")
	require.True(t, len(sLenA) > 0, "Empty server results")
	// Delete a server
	require.NoError(t, testDb.DropServer(ctx, s1.ServerID))
	var d model.Server
	require.True(t, errors.Is(testDb.GetServer(ctx, s1.ServerID, &d), ErrNoResult))
	sLenB, _ := testDb.GetServers(ctx, false)
	require.True(t, len(sLenA)-1 == len(sLenB))
}

func randIP() string {
	return fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255))
}

func TestReport(t *testing.T) {
	var author model.Person
	require.NoError(t, testDb.GetOrCreatePersonBySteamID(context.TODO(), steamid.SID64(76561198003911389), &author))
	var target model.Person
	require.NoError(t, testDb.GetOrCreatePersonBySteamID(context.TODO(), steamid.SID64(76561198083950960), &target))
	report := model.NewReport()
	report.AuthorId = author.SteamID
	report.ReportedId = target.SteamID
	report.Title = "test"
	report.Description = "test"
	require.NoError(t, testDb.SaveReport(context.TODO(), &report))

}

func TestBanNet(t *testing.T) {
	banNetEqual := func(b1, b2 model.BanNet) {
		require.Equal(t, b1.Reason, b2.Reason)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	rip := randIP()
	n1, _ := model.NewBanNet(fmt.Sprintf("%s/32", rip), "testing", time.Hour*100, model.System)
	require.NoError(t, testDb.SaveBanNet(ctx, &n1))
	require.Less(t, int64(0), n1.NetID)
	b1, err2 := testDb.GetBanNet(ctx, net.ParseIP(rip))
	require.NoError(t, err2)
	banNetEqual(b1[0], n1)
	require.Equal(t, b1[0].Reason, n1.Reason)
}

func TestBan(t *testing.T) {
	banEqual := func(b1, b2 *model.Ban) {
		require.Equal(t, b1.BanID, b2.BanID)
		require.Equal(t, b1.AuthorID, b2.AuthorID)
		require.Equal(t, b1.Reason, b2.Reason)
		require.Equal(t, b1.ReasonText, b2.ReasonText)
		require.Equal(t, b1.BanType, b2.BanType)
		require.Equal(t, b1.Source, b2.Source)
		require.Equal(t, b1.Note, b2.Note)
		require.True(t, b2.ValidUntil.Unix() > 0)
		require.Equal(t, b1.ValidUntil.Unix(), b2.ValidUntil.Unix())
		require.Equal(t, b1.CreatedOn.Unix(), b2.CreatedOn.Unix())
		require.Equal(t, b1.UpdatedOn.Unix(), b2.UpdatedOn.Unix())
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	b1 := model.NewBan(76561198044052046, 76561198003911389, time.Hour*24)
	require.NoError(t, testDb.SaveBan(ctx, &b1), "Failed to add ban")
	b1Fetched := model.NewBannedPerson()
	require.NoError(t, testDb.GetBanBySteamID(ctx, 76561198044052046, false, &b1Fetched))
	banEqual(&b1, &b1Fetched.Ban)

	b1duplicate := model.NewBan(76561198044052046, 76561198003911389, time.Hour*24)
	require.True(t, errors.Is(testDb.SaveBan(ctx, &b1duplicate), ErrDuplicate), "Was able to add duplicate ban")

	b1Fetched.Ban.AuthorID = 76561198057999536
	b1Fetched.Ban.ReasonText = "test reason"
	b1Fetched.Ban.ValidUntil = config.Now().Add(time.Minute * 10)
	b1Fetched.Ban.Note = "test note"
	b1Fetched.Ban.Source = model.Web
	require.NoError(t, testDb.SaveBan(ctx, &b1Fetched.Ban), "Failed to edit ban")
	b1FetchedUpdated := model.NewBannedPerson()
	require.NoError(t, testDb.GetBanBySteamID(ctx, 76561198044052046, false, &b1FetchedUpdated))
	banEqual(&b1Fetched.Ban, &b1FetchedUpdated.Ban)

	require.NoError(t, testDb.DropBan(ctx, &b1), "Failed to drop ban")
	vb := model.NewBannedPerson()
	errMissing := testDb.GetBanBySteamID(ctx, b1.SteamID, false, &vb)
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
	require.NoError(t, testDb.SaveBan(ctx, &b1), "Failed to add ban")
	appeal := model.Appeal{
		BanID:       b1.BanID,
		AppealText:  "Im a nerd",
		AppealState: model.ASNew,
		Email:       "",
	}
	require.NoError(t, testDb.SaveAppeal(ctx, &appeal), "failed to save appeal")
	require.True(t, appeal.AppealID > 0, "No appeal id set")
	appeal.AppealState = model.ASDenied
	appeal.Email = "test@test.com"
	require.NoError(t, testDb.SaveAppeal(ctx, &appeal), "failed to update appeal")
	var fetched model.Appeal
	require.NoError(t, testDb.GetAppeal(ctx, b1.BanID, &fetched), "failed to get appeal")
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
	require.NoError(t, testDb.SavePerson(ctx, &p1))
	p2Fetched := model.NewPerson(p2.SteamID)
	require.NoError(t, testDb.GetOrCreatePersonBySteamID(ctx, p2.SteamID, &p2Fetched))
	require.Equal(t, p2.SteamID, p2Fetched.SteamID)
	pBadID := model.NewPerson(0)
	require.Error(t, testDb.GetPersonBySteamID(ctx, 0, &pBadID))
	ips, eH := testDb.GetIPHistory(ctx, p1.SteamID)
	require.NoError(t, eH)
	require.NoError(t, testDb.AddPersonIP(ctx, &p1, "10.0.0.2"), "failed to add ip record")
	require.NoError(t, testDb.AddPersonIP(ctx, &p1, "10.0.0.3"), "failed to add 2nd ip record")
	ipsUpdated, eH2 := testDb.GetIPHistory(ctx, p1.SteamID)
	require.NoError(t, eH2)
	require.True(t, len(ipsUpdated)-len(ips) == 2)
	require.NoError(t, testDb.DropPerson(ctx, p1.SteamID))
}

func TestGetChatHistory(t *testing.T) {
	sid := steamid.SID64(76561198083950960)
	ctx := context.Background()
	s := model.NewServer(golib.RandomString(10), "localhost", rand.Intn(65535))
	require.NoError(t, testDb.SaveServer(ctx, &s))
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
			Extra:     "test-1",
			CreatedOn: config.Now().Add(-1 * time.Second),
		},
		{
			Server:    &s,
			Source:    &player,
			EventType: logparse.Say,
			Extra:     "test-2",
			CreatedOn: config.Now(),
		},
	}
	require.NoError(t, testDb.BatchInsertServerLogs(ctx, logs))
	hist, errHist := testDb.GetChatHistory(ctx, sid, 100)
	require.NoError(t, errHist, "Failed to fetch chat history")
	require.True(t, len(hist) >= 2, "History size too small: %d", len(hist))
	require.Equal(t, "test-2", hist[0].Msg)
}

func TestFindLogEvents(t *testing.T) {
	sid := steamid.SID64(76561198083950960)
	sid2 := steamid.SID64(76561198083950961)
	ctx := context.Background()
	s := model.NewServer(golib.RandomString(10), "localhost", rand.Intn(65535))
	require.NoError(t, testDb.SaveServer(ctx, &s))
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
			Extra:     "test-1",
		},
		{
			Server:    &s,
			Source:    &s1,
			EventType: logparse.Say,
			Extra:     "test-2",
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
	require.NoError(t, testDb.BatchInsertServerLogs(ctx, logs))
	logEvents, errLogs := testDb.FindLogEvents(ctx, model.LogQueryOpts{
		LogTypes: []logparse.MsgType{logparse.Killed},
	})
	require.NoError(t, errLogs, "Failed to fetch logs")
	require.True(t, len(logEvents) >= 1, "Log size too small: %d", len(logEvents))
	for _, evt := range logEvents {
		require.Equal(t, logparse.Killed, evt.EventType)
	}
}

func TestFilters(t *testing.T) {
	existingFilters, err := testDb.GetFilters(context.Background())
	require.NoError(t, err)
	words := []string{golib.RandomString(10), golib.RandomString(10)}
	var savedFilters []model.Filter
	for _, word := range words {
		f := model.Filter{
			Pattern:   regexp.MustCompile(word),
			CreatedOn: config.Now(),
		}
		require.NoError(t, testDb.SaveFilter(context.Background(), &f), "Failed to insert filter: %s", word)
		require.True(t, f.WordID > 0)
		savedFilters = append(savedFilters, f)
	}
	currentFilters, err2 := testDb.GetFilters(context.Background())
	require.NoError(t, err2)
	require.Equal(t, len(existingFilters)+len(words), len(currentFilters))
	if savedFilters != nil {
		require.NoError(t, testDb.DropFilter(context.Background(), &savedFilters[0]))
		var byId model.Filter
		require.NoError(t, testDb.GetFilterByID(context.Background(), savedFilters[1].WordID, &byId))
		require.Equal(t, savedFilters[1].WordID, byId.WordID)
		require.Equal(t, savedFilters[1].Pattern.String(), byId.Pattern.String())
	}
	droppedFilters, err3 := testDb.GetFilters(context.Background())
	require.NoError(t, err3)
	require.Equal(t, len(existingFilters)+len(words)-1, len(droppedFilters))

}

func TestBanASN(t *testing.T) {
	var author model.Person
	require.NoError(t, testDb.GetOrCreatePersonBySteamID(context.TODO(), steamid.SID64(76561198083950960), &author))
	b1 := model.NewBanASN(1, author.SteamID, "test", time.Minute*10)
	require.NoError(t, testDb.SaveBanASN(context.Background(), &b1))
	require.True(t, b1.BanASNId > 0)
	var f1 model.BanASN
	require.NoError(t, testDb.GetBanASN(context.TODO(), b1.ASNum, &f1))
	require.NoError(t, testDb.DropBanASN(context.TODO(), &f1))
	var d1 model.BanASN
	require.Error(t, testDb.GetBanASN(context.TODO(), b1.ASNum, &d1))
}
