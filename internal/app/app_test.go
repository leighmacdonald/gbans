package app

import (
	"context"
	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/store"
	"go.uber.org/zap"
	"os"
	"testing"
)

var (
	testDatabase store.Store
)

func TestMain(testMain *testing.M) {
	logger := zap.NewNop()

	tearDown := func(database store.Store) {
		q := `select 'drop table "' || tablename || '" cascade;' from pg_tables where schemaname = 'public';`
		if errMigrate := database.Exec(context.Background(), q); errMigrate != nil {
			logger.Error("Failed to migrate database down", zap.Error(errMigrate))
			os.Exit(2)
		}
	}

	_, _ = config.Read()
	config.General.Mode = config.TestMode
	testCtx := context.Background()

	dbStore, dbErr := store.New(testCtx, logger, config.DB.DSN)
	if dbErr != nil {
		logger.Error("Failed to setup store", zap.Error(dbErr))
		return
	}
	tearDown(dbStore)
	defer util.LogClose(logger, dbStore)
	app := New(testCtx, logger)
	testDatabase = dbStore
	app.store = testDatabase
	webService, errWeb := NewWeb(app)
	if errWeb != nil {
		tearDown(dbStore)
		logger.Error("Failed to setup web", zap.Error(errWeb))
		return
	}

	// Load in the external network block / ip ban lists to memory if enabled
	if config.Net.Enabled {
		_ = initNetBans()
	} else {
		logger.Warn("External Network ban lists not enabled")
	}

	// Start the discord service
	if config.Discord.Enabled {
		go app.initDiscord(testCtx, app.discordSendMsg)
	} else {
		logger.Warn("discord bot not enabled")
	}

	// Start the background goroutine workers
	app.initWorkers()

	// Load the filtered word set into memory
	if config.Filter.Enabled {
		_ = initFilters(testCtx, dbStore)
	}

	webCtx, cancel := context.WithCancel(testCtx)
	go func() {
		// Start & block, listening on the HTTP server
		if errHttpListen := webService.ListenAndServe(webCtx); errHttpListen != nil {
			logger.Error("Error shutting down service", zap.Error(errHttpListen))
		}
	}()
	rc := testMain.Run()
	cancel()
	<-webCtx.Done()
	tearDown(dbStore)
	os.Exit(rc)
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
