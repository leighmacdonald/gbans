package store

import (
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/golib"
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	dbPath := "./test.sqlite"
	var drop = func() {
		if golib.Exists(dbPath) {
			if err := os.Remove(dbPath); err != nil {
				log.Fatalf("Failed to delete existing database")
			}
		}
	}
	drop()
	Init(dbPath)
	defer func() {
		if err := Close(); err != nil {
			log.Fatalf("Failed to close database")
		}
		drop()
	}()
	os.Exit(m.Run())
}

func TestServer(t *testing.T) {
	s1 := model.Server{
		ServerName:     "test-1",
		Token:          "",
		Address:        "172.16.1.100",
		Port:           27015,
		RCON:           "test",
		Password:       "test",
		TokenCreatedOn: time.Now().Unix(),
		CreatedOn:      time.Now().Unix(),
		UpdatedOn:      time.Now().Unix(),
	}
	require.NoError(t, SaveServer(&s1))
	require.True(t, s1.ServerID > 0)
	s1Get, err := GetServer(s1.ServerID)
	require.NoError(t, err)
	require.Equal(t, s1.ServerID, s1Get.ServerID)
	require.NoError(t, DropServer(s1.ServerID))
}

func TestBanNet(t *testing.T) {
	n1, _ := model.NewBanNet("172.16.1.0/24", "testing", time.Hour*100, model.System)
	require.NoError(t, SaveBanNet(&n1))
	require.Less(t, int64(0), n1.NetID)
	b1, err := GetBanNet("172.16.1.100")
	require.NoError(t, err)
	require.Equal(t, b1.Reason, n1.Reason)
}

func TestBan(t *testing.T) {
	p1 := model.Person{
		SteamID: 76561199093644873,
	}
	require.NoError(t, SavePerson(&p1))
}
