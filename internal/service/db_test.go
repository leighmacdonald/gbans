package service

import (
	"fmt"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
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
	require.NoError(t, dropServer(s1.ServerID))
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
	b1 := model.NewBan(76561198084134025, 76561198003911389, time.Hour*24)
	require.NoError(t, SaveBan(b1), "Failed to add ban")

	b1Fetched, err := getBanBySteamID(76561198084134025, false)
	require.NoError(t, err)
	banEqual(b1, b1Fetched.Ban)

	b1duplicate := model.NewBan(76561198084134025, 76561198003911389, time.Hour*24)
	require.True(t, errors.Is(SaveBan(b1duplicate), errDuplicate), "Was able to add duplicate ban")

	b1Fetched.Ban.AuthorID = 76561198057999536
	b1Fetched.Ban.ReasonText = "test reason"
	b1Fetched.Ban.ValidUntil = config.Now().Add(time.Minute * 10)
	b1Fetched.Ban.Note = "test note"
	b1Fetched.Ban.Source = model.Web
	require.NoError(t, SaveBan(b1Fetched.Ban), "Failed to edit ban")

	b1FetchedUpdated, err := getBanBySteamID(76561198084134025, false)
	require.NoError(t, err)
	banEqual(b1Fetched.Ban, b1FetchedUpdated.Ban)

	require.NoError(t, dropBan(b1), "Failed to drop ban")
	_, errMissing := getBanBySteamID(b1.SteamID, false)
	require.Error(t, errMissing)
	require.True(t, errors.Is(errMissing, errNoResult))
}

func TestFilteredWords(t *testing.T) {
	//
}

func TestAppeal(t *testing.T) {
	b1 := model.NewBan(76561199093644873, 76561198003911389, time.Hour*24)
	require.NoError(t, SaveBan(b1), "Failed to add ban")
	appeal := model.Appeal{
		BanID:       b1.BanID,
		AppealText:  "Im a nerd",
		AppealState: model.ASNew,
		Email:       "",
	}
	require.NoError(t, saveAppeal(&appeal), "failed to save appeal")
	require.True(t, appeal.AppealID > 0, "No appeal id set")
	appeal.AppealState = model.ASDenied
	appeal.Email = "test@test.com"
	require.NoError(t, saveAppeal(&appeal), "failed to update appeal")

	fetched, err := getAppeal(b1.BanID)
	require.NoError(t, err, "failed to get appeal")
	require.Equal(t, appeal.BanID, fetched.BanID)
	require.Equal(t, appeal.Email, fetched.Email)
	require.Equal(t, appeal.AppealState, fetched.AppealState)
	require.Equal(t, appeal.AppealID, fetched.AppealID)
	require.Equal(t, appeal.AppealText, fetched.AppealText)
}

func TestPerson(t *testing.T) {
	p1 := model.NewPerson(76561198083950961)
	p2 := model.NewPerson(76561198084134025)
	require.NoError(t, SavePerson(p1))
	p2Fetched, err := GetOrCreatePersonBySteamID(p2.SteamID)
	require.NoError(t, err)
	require.Equal(t, p2.SteamID, p2Fetched.SteamID)

	pBadID, err := getPersonBySteamID(0)
	require.Error(t, err)
	require.Nil(t, pBadID)

	ips := getIPHistory(p1.SteamID)
	p1.IPAddr = "10.0.0.2"
	require.NoError(t, addPersonIP(p1), "failed to add ip record")
	p1.IPAddr = "10.0.0.3"
	require.NoError(t, addPersonIP(p1), "failed to add 2nd ip record")
	ipsUpdated := getIPHistory(p1.SteamID)
	require.True(t, len(ipsUpdated)-len(ips) == 2)

	require.NoError(t, dropPerson(p1.SteamID))
}
