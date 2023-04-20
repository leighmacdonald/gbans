package app

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

var (
	testDatabase store.Store
)

func TestMain(testMain *testing.M) {
	tearDown := func(database store.Store) {
		q := `select 'drop table "' || tablename || '" cascade;' from pg_tables where schemaname = 'public';`
		if errMigrate := database.Exec(context.Background(), q); errMigrate != nil {
			log.Errorf("Failed to migrate database down: %v", errMigrate)
			os.Exit(2)
		}
	}

	config.Read()
	config.General.Mode = config.TestMode
	testCtx := context.Background()

	dbStore, dbErr := store.New(testCtx, config.DB.DSN)
	if dbErr != nil {
		log.Errorf("Failed to setup store: %v", dbErr)
		return
	}
	tearDown(dbStore)
	defer func() {
		if errClose := dbStore.Close(); errClose != nil {
			log.Errorf("Error cleanly closing app: %v", errClose)
		}
	}()
	app := New(testCtx)
	testDatabase = dbStore
	app.store = testDatabase
	webService, errWeb := NewWeb(app)
	if errWeb != nil {
		tearDown(dbStore)
		log.Errorf("Failed to setup web: %v", errWeb)
		return
	}

	// Load in the external network block / ip ban lists to memory if enabled
	if config.Net.Enabled {
		initNetBans()
	} else {
		log.Warnf("External Network ban lists not enabled")
	}

	// Start the discord service
	if config.Discord.Enabled {
		go app.initDiscord(testCtx, dbStore, app.discordSendMsg)
	} else {
		log.Warn("discord bot not enabled")
	}

	// Start the background goroutine workers
	app.initWorkers()

	// Load the filtered word set into memory
	if config.Filter.Enabled {
		initFilters(testCtx, dbStore)
	}

	webCtx, cancel := context.WithCancel(testCtx)
	go func() {
		// Start & block, listening on the HTTP server
		if errHttpListen := webService.ListenAndServe(webCtx); errHttpListen != nil {
			log.Errorf("Error shutting down service: %v", errHttpListen)
		}
	}()
	rc := testMain.Run()
	cancel()
	<-webCtx.Done()
	tearDown(dbStore)
	os.Exit(rc)
}

func TestSteamWebAPI(t *testing.T) {
	if config.General.SteamKey == "" {
		t.Skip("No steamkey set")
		return
	}
	friends, errFetch := thirdparty.FetchFriends(context.Background(), 76561197961279983)
	require.NoError(t, errFetch)
	require.True(t, len(friends) > 100)
	summaries, errFetchSummaries := thirdparty.FetchSummaries(friends)
	require.NoError(t, errFetchSummaries)
	require.Equal(t, len(friends), len(summaries))
}

//func TestFetchPlayerBans(t *testing.T) {
//	reqIds := steamid.Collection{
//		76561198044052046,
//		76561198059958958,
//		76561197999702457,
//		76561198189957966,
//	}
//	bans, errFetch := FetchPlayerBans(reqIds)
//	require.NoError(t, errFetch, "HTTP error fetching Player bans")
//	require.Equal(t, len(bans), len(reqIds))
//}
