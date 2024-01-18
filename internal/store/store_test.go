package store_test

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/model"
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
	t.Run("forum", testForum(database))
}

func testServerTest(database store.Store) func(t *testing.T) {
	return func(t *testing.T) {
		serverA := model.Server{
			ShortName:      fmt.Sprintf("test-%s", golib.RandomString(10)),
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
		require.NoError(t, store.SaveServer(ctx, database, &serverA))
		require.True(t, serverA.ServerID > 0)

		// Fetch saved server
		var s1Get model.Server

		require.NoError(t, store.GetServer(ctx, database, serverA.ServerID, &s1Get))
		require.Equal(t, serverA.ServerID, s1Get.ServerID)
		require.Equal(t, serverA.ShortName, s1Get.ShortName)
		require.Equal(t, serverA.Address, s1Get.Address)
		require.Equal(t, serverA.Port, s1Get.Port)
		require.Equal(t, serverA.RCON, s1Get.RCON)
		require.Equal(t, serverA.Password, s1Get.Password)
		require.Equal(t, serverA.TokenCreatedOn.Second(), s1Get.TokenCreatedOn.Second())
		require.Equal(t, serverA.CreatedOn.Second(), s1Get.CreatedOn.Second())
		require.Equal(t, serverA.UpdatedOn.Second(), s1Get.UpdatedOn.Second())

		// Fetch all enabled servers
		sLenA, count, errGetServers := store.GetServers(ctx, database, store.ServerQueryFilter{})
		require.NoError(t, errGetServers, "Failed to fetch enabled servers")
		require.Equal(t, count, int64(len(sLenA)), "Mismatches counts")
		// Delete a server
		require.NoError(t, store.DropServer(ctx, database, serverA.ServerID))

		var server model.Server

		require.True(t, errors.Is(store.GetServer(ctx, database, serverA.ServerID, &server), store.ErrNoResult))

		sLenB, _, _ := store.GetServers(ctx, database, store.ServerQueryFilter{})
		require.True(t, len(sLenA)-1 == len(sLenB))
	}
}

func randIP() string {
	return fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255)) //nolint:gosec
}

func testReport(database store.Store) func(t *testing.T) {
	return func(t *testing.T) {
		var author model.Person

		require.NoError(t, store.GetOrCreatePersonBySteamID(context.TODO(), database, steamid.New(76561198003911389), &author))

		var target model.Person

		require.NoError(t, store.GetOrCreatePersonBySteamID(context.TODO(), database, steamid.RandSID64(), &target))

		report := model.NewReport()

		report.SourceID = author.SteamID
		report.TargetID = target.SteamID
		report.Description = golib.RandomString(120)

		require.NoError(t, store.SaveReport(context.TODO(), database, &report))

		msg1 := model.NewReportMessage(report.ReportID, author.SteamID, golib.RandomString(100))
		msg2 := model.NewReportMessage(report.ReportID, author.SteamID, golib.RandomString(100))

		require.NoError(t, store.SaveReportMessage(context.Background(), database, &msg1))
		require.NoError(t, store.SaveReportMessage(context.Background(), database, &msg2))

		msgs, msgsErr := store.GetReportMessages(context.Background(), database, report.ReportID)
		require.NoError(t, msgsErr)
		require.Equal(t, 2, len(msgs))
		require.NoError(t, store.DropReport(context.Background(), database, &report))
	}
}

func testBanNet(database *store.Database) func(t *testing.T) {
	return func(t *testing.T) {
		bgCtx := context.Background()
		banNetEqual := func(b1, b2 model.BanCIDR) {
			require.Equal(t, b1.Reason, b2.Reason)
		}

		ctx, cancel := context.WithTimeout(bgCtx, time.Second*10)
		defer cancel()

		rip := randIP()

		var banCidr model.BanCIDR

		require.NoError(t, model.NewBanCIDR(ctx, model.StringSID("76561198003911389"),
			"76561198044052046", time.Minute*10, model.Custom,
			"custom reason", "", model.System, fmt.Sprintf("%s/32", rip), model.Banned, &banCidr))
		require.NoError(t, store.SaveBanNet(ctx, database, &banCidr))
		require.Less(t, int64(0), banCidr.NetID)

		banNet, errGetBanNet := store.GetBanNetByAddress(ctx, database, net.ParseIP(rip))
		require.NoError(t, errGetBanNet)

		banNetEqual(banNet[0], banCidr)
		require.Equal(t, banNet[0].Reason, banCidr.Reason)
	}
}

func testBanSteam(database *store.Database) func(t *testing.T) {
	return func(t *testing.T) {
		bgCtx := context.Background()
		banEqual := func(ban1, ban2 *model.BanSteam) {
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

		var banSteam model.BanSteam

		require.NoError(t, model.NewBanSteam(
			ctx,
			model.StringSID("76561198003911389"),
			"76561198044052046",
			time.Hour*24*31,
			model.Cheating,
			model.Cheating.String(),
			"Mod Note",
			model.System, 0, model.Banned, false, &banSteam), "Failed to create ban opts")

		require.NoError(t, store.SaveBan(ctx, database, &banSteam), "Failed to add ban")

		b1Fetched := model.NewBannedPerson()
		require.NoError(t, store.GetBanBySteamID(ctx, database, steamid.New(76561198044052046), &b1Fetched, false))
		banEqual(&banSteam, &b1Fetched.BanSteam)

		b1duplicate := banSteam
		b1duplicate.BanID = 0
		require.True(t, errors.Is(store.SaveBan(ctx, database, &b1duplicate), store.ErrDuplicate), "Was able to add duplicate ban")

		b1Fetched.SourceID = steamid.New(76561198057999536)
		b1Fetched.ReasonText = "test reason"
		b1Fetched.ValidUntil = time.Now().Add(time.Minute * 10)
		b1Fetched.Note = "test note"
		b1Fetched.Origin = model.Web
		require.NoError(t, store.SaveBan(ctx, database, &b1Fetched.BanSteam), "Failed to edit ban")

		b1FetchedUpdated := model.NewBannedPerson()
		require.NoError(t, store.GetBanBySteamID(ctx, database, steamid.New(76561198044052046), &b1FetchedUpdated, false))
		banEqual(&b1Fetched.BanSteam, &b1FetchedUpdated.BanSteam)

		require.NoError(t, store.DropBan(ctx, database, &banSteam, false), "Failed to drop ban")

		vb := model.NewBannedPerson()
		errMissing := store.GetBanBySteamID(ctx, database, banSteam.TargetID, &vb, false)

		require.Error(t, errMissing)
		require.True(t, errors.Is(errMissing, store.ErrNoResult))
	}
}

func randSID() steamid.SID64 {
	return steamid.New(76561197960265728 + rand.Int63n(100000000)) //nolint:gosec
}

func testPerson(database store.Store) func(t *testing.T) {
	return func(t *testing.T) {
		var (
			person1 = model.NewPerson(randSID())
			person2 = model.NewPerson(randSID())
		)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
		defer cancel()

		require.NoError(t, store.SavePerson(ctx, database, &person1))

		p2Fetched := model.NewPerson(person2.SteamID)
		require.NoError(t, store.GetOrCreatePersonBySteamID(ctx, database, person2.SteamID, &p2Fetched))
		require.Equal(t, person2.SteamID, p2Fetched.SteamID)

		pBadID := model.NewPerson("")
		require.Error(t, store.GetPersonBySteamID(ctx, database, "", &pBadID))

		_, errHistory := store.GetPersonIPHistory(ctx, database, person1.SteamID, 1000)
		require.NoError(t, errHistory)
		require.NoError(t, store.DropPerson(ctx, database, person1.SteamID))
	}
}

func testChatHistory(database *store.Database) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		newServer := model.NewServer(golib.RandomString(10), "localhost", rand.Intn(65535)) //nolint:gosec
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
		require.NoError(t, store.SaveServer(ctx, database, &newServer))
	}
}

func testFilters(database store.Store) func(t *testing.T) {
	return func(t *testing.T) {
		player1 := model.NewPerson(randSID())

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		require.NoError(t, store.SavePerson(ctx, database, &player1))

		existingFilters, _, errGetFilters := store.GetFilters(context.Background(), database, store.FiltersQueryFilter{})
		require.NoError(t, errGetFilters)

		var (
			words        = []string{golib.RandomString(10), golib.RandomString(20)}
			savedFilters = make([]model.Filter, len(words))
		)

		for index, word := range words {
			filter := model.Filter{
				IsEnabled: true,
				IsRegex:   false,
				AuthorID:  player1.SteamID,
				Pattern:   word,
				UpdatedOn: time.Now(),
				CreatedOn: time.Now(),
			}

			require.NoError(t, store.SaveFilter(ctx, database, &filter), "Failed to insert filter: %s", word)
			require.True(t, filter.FilterID > 0)
			savedFilters[index] = filter
		}

		currentFilters, _, errGetCurrentFilters := store.GetFilters(ctx, database, store.FiltersQueryFilter{})
		require.NoError(t, errGetCurrentFilters)
		require.Equal(t, len(existingFilters)+len(words), len(currentFilters))
		require.NoError(t, store.DropFilter(ctx, database, &savedFilters[0]))

		var byID model.Filter

		require.NoError(t, store.GetFilterByID(ctx, database, savedFilters[1].FilterID, &byID))
		require.Equal(t, savedFilters[1].FilterID, byID.FilterID)
		require.Equal(t, savedFilters[1].Pattern, byID.Pattern)

		droppedFilters, _, errGetDroppedFilters := store.GetFilters(ctx, database, store.FiltersQueryFilter{})
		require.NoError(t, errGetDroppedFilters)
		require.Equal(t, len(existingFilters)+len(words)-1, len(droppedFilters))
	}
}

func testBanASN(database store.Store) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		var author model.Person

		require.NoError(t, store.GetOrCreatePersonBySteamID(ctx, database, steamid.New(76561198083950960), &author))

		var banASN model.BanASN

		require.NoError(t, model.NewBanASN(ctx,
			model.StringSID(author.SteamID.String()), "0",
			time.Minute*10, model.Cheating, "", "", model.System, rand.Int63n(23455), model.Banned, &banASN)) //nolint:gosec

		require.NoError(t, store.SaveBanASN(context.Background(), database, &banASN))
		require.True(t, banASN.BanASNId > 0)

		var banASN2 model.BanASN

		require.NoError(t, store.GetBanASN(context.TODO(), database, banASN.ASNum, &banASN2))
		require.NoError(t, store.DropBanASN(context.TODO(), database, &banASN2))

		var banASN3 model.BanASN

		require.Error(t, store.GetBanASN(context.TODO(), database, banASN.ASNum, &banASN3))
	}
}

func testBanGroup(database store.Store) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()

		var banGroup model.BanGroup

		require.NoError(t, model.NewBanSteamGroup(
			ctx,
			model.StringSID("76561198083950960"),
			"0",
			time.Minute*10,
			"",
			model.System,
			steamid.NewGID(103000000000000000+int64(rand.Int())), //nolint:gosec
			golib.RandomString(10),
			model.Banned,
			&banGroup))

		require.NoError(t, store.SaveBanGroup(context.TODO(), database, &banGroup))
		require.True(t, banGroup.BanGroupID > 0)

		var bgB model.BanGroup

		require.NoError(t, store.GetBanGroup(context.TODO(), database, banGroup.GroupID, &bgB))
		require.EqualValues(t, banGroup.BanGroupID, bgB.BanGroupID)
		require.NoError(t, store.DropBanGroup(context.TODO(), database, &banGroup))

		var bgDeleted model.BanGroup

		getErr := store.GetBanGroup(context.TODO(), database, banGroup.GroupID, &bgDeleted)
		require.EqualError(t, store.ErrNoResult, getErr.Error())
	}
}

func testForum(database store.Store) func(t *testing.T) {
	ctx := context.Background()

	return func(t *testing.T) {
		t.Run("category", func(t *testing.T) {
			forumCategory1 := model.ForumCategory{
				Title:       "test category",
				Description: "test description",
				Ordering:    2,
				TimeStamped: model.NewTimeStamped(),
			}
			require.NoError(t, store.ForumCategorySave(ctx, database, &forumCategory1))
			require.Greater(t, forumCategory1.ForumCategoryID, 0)

			var forumCategory2 model.ForumCategory

			require.NoError(t, store.ForumCategory(ctx, database, forumCategory1.ForumCategoryID, &forumCategory2))
			require.Equal(t, forumCategory1.Title, forumCategory2.Title)
			require.Equal(t, forumCategory1.Description, forumCategory2.Description)
			require.Equal(t, forumCategory1.Ordering, forumCategory2.Ordering)

			forumCategory2.Title += forumCategory2.Title
			forumCategory2.Description += forumCategory2.Description
			forumCategory2.Ordering = 3

			require.NoError(t, store.ForumCategorySave(ctx, database, &forumCategory2))

			var forumCategory3 model.ForumCategory

			require.NoError(t, store.ForumCategory(ctx, database, forumCategory1.ForumCategoryID, &forumCategory3))

			require.Equal(t, forumCategory2.Title, forumCategory3.Title)
			require.Equal(t, forumCategory2.Description, forumCategory3.Description)
			require.Equal(t, forumCategory2.Ordering, forumCategory3.Ordering)

			require.NoError(t, store.ForumCategoryDelete(ctx, database, forumCategory3.ForumCategoryID))

			var forumCategory4 model.ForumCategory
			require.ErrorIs(t, store.ErrNoResult, store.ForumCategory(ctx, database, forumCategory3.ForumCategoryID, &forumCategory4))
		})
		t.Run("forum", func(t *testing.T) {
			validCategory := model.ForumCategory{
				Title:       "valid category",
				Description: "test description",
				Ordering:    1,
				TimeStamped: model.NewTimeStamped(),
			}
			require.NoError(t, store.ForumCategorySave(ctx, database, &validCategory))

			forum := validCategory.NewForum("Forum Title", "Forum Description")

			require.NoError(t, store.ForumSave(ctx, database, &forum))
			require.True(t, forum.ForumID > 0)

			forum.Title = "new title"

			require.NoError(t, store.ForumSave(ctx, database, &forum))

			var updatedForum model.Forum

			require.NoError(t, store.Forum(ctx, database, forum.ForumID, &updatedForum))
			require.Equal(t, forum.Title, updatedForum.Title)
			require.NoError(t, store.ForumDelete(ctx, database, updatedForum.ForumID))
			require.ErrorIs(t, store.ErrNoResult, store.Forum(ctx, database, forum.ForumID, &updatedForum))
		})
		t.Run("thread", func(t *testing.T) {
			validCategory := model.ForumCategory{
				Title: "new valid category", Description: "test description", TimeStamped: model.NewTimeStamped(),
			}
			require.NoError(t, store.ForumCategorySave(ctx, database, &validCategory))
			validForum := validCategory.NewForum("forum", "")
			require.NoError(t, store.ForumSave(ctx, database, &validForum))
			require.True(t, validForum.ForumID > 0)
			var person model.Person
			require.NoError(t, store.GetOrCreatePersonBySteamID(ctx, database,
				steamid.New(76561198057999536), &person))
			thread := validForum.NewThread("thread title", person.SteamID)
			require.NoError(t, store.ForumThreadSave(ctx, database, &thread))

			thread.Title = "new title"

			require.NoError(t, store.ForumThreadSave(ctx, database, &thread))
			var updatedThread model.ForumThread
			require.NoError(t, store.ForumThread(ctx, database, thread.ForumThreadID, &updatedThread))
			require.Equal(t, thread.Title, updatedThread.Title)
			require.NoError(t, store.ForumThreadDelete(ctx, database, updatedThread.ForumThreadID))
			require.ErrorIs(t, store.ErrNoResult, store.ForumThread(ctx, database, thread.ForumThreadID, &updatedThread))
		})

		t.Run("message", func(t *testing.T) {
			validCategory := model.ForumCategory{
				Title: "new valid category 2", Description: "test description", TimeStamped: model.NewTimeStamped(),
			}
			require.NoError(t, store.ForumCategorySave(ctx, database, &validCategory))
			validForum := validCategory.NewForum("forum messages", "")
			require.NoError(t, store.ForumSave(ctx, database, &validForum))
			var person model.Person
			require.NoError(t, store.GetOrCreatePersonBySteamID(ctx, database,
				steamid.New(76561198057999536), &person))
			validThread := validForum.NewThread("thread title", person.SteamID)
			require.NoError(t, store.ForumThreadSave(ctx, database, &validThread))

			message := validThread.NewMessage(person.SteamID, "test *body*")
			require.NoError(t, store.ForumMessageSave(ctx, database, &message))
			require.True(t, message.ForumMessageID > 0)
			message.BodyMD += "blah"
			require.NoError(t, store.ForumMessageSave(ctx, database, &message))

			var updatedMessage model.ForumMessage
			require.NoError(t, store.ForumMessage(ctx, database, message.ForumMessageID, &updatedMessage))
			require.Equal(t, message.BodyMD, updatedMessage.BodyMD)

			require.NoError(t, store.ForumMessageDelete(ctx, database, message.ForumMessageID))

			require.ErrorIs(t, store.ErrNoResult, store.ForumMessage(ctx, database, updatedMessage.ForumMessageID, &updatedMessage))
		})

		t.Run("vote", func(t *testing.T) {
			validCategory := model.ForumCategory{
				Title: "new valid category 3", Description: "test description", TimeStamped: model.NewTimeStamped(),
			}
			require.NoError(t, store.ForumCategorySave(ctx, database, &validCategory))
			validForum := validCategory.NewForum("forum messages 2", "")
			require.NoError(t, store.ForumSave(ctx, database, &validForum))
			var person model.Person
			require.NoError(t, store.GetOrCreatePersonBySteamID(ctx, database,
				steamid.New(76561198057999536), &person))
			validThread := validForum.NewThread("thread title ", person.SteamID)
			require.NoError(t, store.ForumThreadSave(ctx, database, &validThread))

			message := validThread.NewMessage(person.SteamID, "test *body*")
			require.NoError(t, store.ForumMessageSave(ctx, database, &message))

			upVote := message.NewVote(person.SteamID, model.VoteUp)

			require.NoError(t, store.ForumMessageVoteApply(ctx, database, &upVote))

			var fetchedUpVote model.ForumMessageVote
			require.NoError(t, store.ForumMessageVoteByID(ctx, database, upVote.ForumMessageVoteID, &fetchedUpVote))
			require.Equal(t, upVote.Vote, fetchedUpVote.Vote)

			downVote := message.NewVote(person.SteamID, model.VoteDown)
			require.NoError(t, store.ForumMessageVoteApply(ctx, database, &downVote))
			var fetchedDownVote model.ForumMessageVote
			require.NoError(t, store.ForumMessageVoteByID(ctx, database, upVote.ForumMessageVoteID, &fetchedDownVote))
			require.Equal(t, downVote.Vote, fetchedDownVote.Vote)

			downVote2 := message.NewVote(person.SteamID, model.VoteDown)
			require.NoError(t, store.ForumMessageVoteApply(ctx, database, &downVote2))
			var fetchedDownVote2 model.ForumMessageVote
			require.ErrorIs(t, store.ErrNoResult, store.ForumMessageVoteByID(ctx, database, upVote.ForumMessageVoteID, &fetchedDownVote2))
		})
	}
}
