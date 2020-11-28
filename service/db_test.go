package service

import (
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/model"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	s1 := model.Server{
		ServerName:     "test-1",
		Token:          "",
		Address:        "172.16.1.100",
		Port:           27015,
		RCON:           "test",
		Password:       "test",
		TokenCreatedOn: config.Now(),
		CreatedOn:      config.Now(),
		UpdatedOn:      config.Now(),
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

func TestPerson(t *testing.T) {
	p1 := model.Person{
		SteamID: 76561199093644873,
	}
	p2 := model.Person{
		SteamID: 76561198084134025,
	}
	require.NoError(t, SavePerson(&p1))
	p2Fetched, err := GetOrCreatePersonBySteamID(p2.SteamID)
	require.NoError(t, err)
	require.Equal(t, p2.SteamID, p2Fetched.SteamID)

	pBadID, err := GetPersonBySteamID(-1)
	require.Error(t, err)
	require.Equal(t, pBadID.SteamID, 0)

}
