package service

import (
	"fmt"
	"github.com/leighmacdonald/gbans/config"
	"github.com/leighmacdonald/gbans/model"
	"github.com/leighmacdonald/golib"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	s1 := model.Server{
		ServerName:     fmt.Sprintf("test-%s", golib.RandomString(10)),
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
	s1Get, err := getServer(s1.ServerID)
	require.NoError(t, err)
	require.Equal(t, s1.ServerID, s1Get.ServerID)
	require.NoError(t, DropServer(s1.ServerID))
}

func TestBanNet(t *testing.T) {
	banNetEqual := func(b1, b2 model.BanNet) {
		require.Equal(t, b1.Reason, b2.Reason)
	}
	n1, _ := model.NewBanNet("172.16.1.0/24", "testing", time.Hour*100, model.System)
	require.NoError(t, saveBanNet(&n1))
	require.Less(t, int64(0), n1.NetID)
	b1, err := getBanNet(net.ParseIP("172.16.1.100"))
	require.NoError(t, err)
	banNetEqual(b1[0], n1)
	require.Equal(t, b1[0].Reason, n1.Reason)
}

func TestBan(t *testing.T) {
	banEqual := func(b1, b2 model.Ban) {
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
	b1, err := model.NewBan(76561198084134025, 76561198003911389, "test", time.Hour*24, model.System)
	require.NoError(t, err)
	require.NoError(t, SaveBan(&b1), "Failed to add ban")

	b1Fetched, err := GetBan(76561198084134025)
	require.NoError(t, err)
	banEqual(b1, b1Fetched)

	b1duplicate, err := model.NewBan(76561198084134025, 76561198003911389, "test", time.Hour*24, model.System)
	require.NoError(t, err)
	require.True(t, errors.Is(SaveBan(&b1duplicate), errDuplicate), "Was able to add duplicate ban")

	b1Fetched.AuthorID = 76561198057999536
	b1Fetched.ReasonText = "test reason"
	b1Fetched.ValidUntil = config.Now().Add(time.Minute * 10)
	b1Fetched.Note = "test note"
	b1Fetched.Source = model.Web
	require.NoError(t, SaveBan(&b1Fetched), "Failed to edit ban")

	b1FetchedUpdated, err := GetBan(76561198084134025)
	require.NoError(t, err)
	banEqual(b1Fetched, b1FetchedUpdated)

	require.NoError(t, DropBan(b1), "Failed to drop ban")
	_, errMissing := GetBan(b1.SteamID)
	require.Error(t, errMissing)
	require.True(t, errors.Is(errMissing, errNoResult))
}

func TestFilteredWords(t *testing.T) {
	//
}

func TestAppeal(t *testing.T) {
	//
}

func TestPerson(t *testing.T) {
	p1 := model.Person{
		SteamID: 76561199093644873,
	}
	p2 := model.Person{
		SteamID: 76561198084134025,
	}
	require.NoError(t, savePerson(&p1))
	p2Fetched, err := getOrCreatePersonBySteamID(p2.SteamID)
	require.NoError(t, err)
	require.Equal(t, p2.SteamID, p2Fetched.SteamID)

	pBadID, err := getPersonBySteamID(0)
	require.Error(t, err)
	require.Equal(t, pBadID.SteamID.Int64(), int64(0))
}
