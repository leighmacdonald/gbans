package service

import (
	"context"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	require.NoError(t, SaveServer(ctx, &s1))
	require.True(t, s1.ServerID > 0)
	s1Get, err := getServer(ctx, s1.ServerID)
	require.NoError(t, err)
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
	sLenA, eS := getServers(ctx)
	require.NoError(t, eS, "Failed to fetch servers")
	require.True(t, len(sLenA) > 0, "Empty server results")
	require.NoError(t, dropServer(ctx, s1.ServerID))
	_, errDel := getServer(ctx, s1.ServerID)
	require.True(t, errors.Is(errDel, errNoResult))
	sLenB, _ := getServers(ctx)
	require.True(t, len(sLenA)-1 == len(sLenB))
}

func TestBanNet(t *testing.T) {
	banNetEqual := func(b1, b2 model.BanNet) {
		require.Equal(t, b1.Reason, b2.Reason)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	n1, _ := model.NewBanNet("172.16.1.0/24", "testing", time.Hour*100, model.System)
	require.NoError(t, saveBanNet(ctx, &n1))
	require.Less(t, int64(0), n1.NetID)
	b1, err := getBanNet(ctx, net.ParseIP("172.16.1.100"))
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	b1 := model.NewBan(76561198084134025, 76561198003911389, time.Hour*24)
	require.NoError(t, saveBan(ctx, b1), "Failed to add ban")

	b1Fetched, err := getBanBySteamID(ctx, 76561198084134025, false)
	require.NoError(t, err)
	banEqual(b1, b1Fetched.Ban)

	b1duplicate := model.NewBan(76561198084134025, 76561198003911389, time.Hour*24)
	require.True(t, errors.Is(saveBan(ctx, b1duplicate), errDuplicate), "Was able to add duplicate ban")

	b1Fetched.Ban.AuthorID = 76561198057999536
	b1Fetched.Ban.ReasonText = "test reason"
	b1Fetched.Ban.ValidUntil = config.Now().Add(time.Minute * 10)
	b1Fetched.Ban.Note = "test note"
	b1Fetched.Ban.Source = model.Web
	require.NoError(t, saveBan(ctx, b1Fetched.Ban), "Failed to edit ban")

	b1FetchedUpdated, err := getBanBySteamID(ctx, 76561198084134025, false)
	require.NoError(t, err)
	banEqual(b1Fetched.Ban, b1FetchedUpdated.Ban)

	require.NoError(t, dropBan(ctx, b1), "Failed to drop ban")
	_, errMissing := getBanBySteamID(ctx, b1.SteamID, false)
	require.Error(t, errMissing)
	require.True(t, errors.Is(errMissing, errNoResult))
}

func TestFilteredWords(t *testing.T) {
	//
}

func TestAppeal(t *testing.T) {
	b1 := model.NewBan(76561199093644873, 76561198003911389, time.Hour*24)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	require.NoError(t, saveBan(ctx, b1), "Failed to add ban")
	appeal := model.Appeal{
		BanID:       b1.BanID,
		AppealText:  "Im a nerd",
		AppealState: model.ASNew,
		Email:       "",
	}
	require.NoError(t, saveAppeal(ctx, &appeal), "failed to save appeal")
	require.True(t, appeal.AppealID > 0, "No appeal id set")
	appeal.AppealState = model.ASDenied
	appeal.Email = "test@test.com"
	require.NoError(t, saveAppeal(ctx, &appeal), "failed to update appeal")
	fetched, err := getAppeal(ctx, b1.BanID)
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	require.NoError(t, SavePerson(ctx, p1))
	p2Fetched, err := GetOrCreatePersonBySteamID(ctx, p2.SteamID)
	require.NoError(t, err)
	require.Equal(t, p2.SteamID, p2Fetched.SteamID)
	pBadID, err := getPersonBySteamID(ctx, 0)
	require.Error(t, err)
	require.Nil(t, pBadID)
	ips := getIPHistory(ctx, p1.SteamID)
	require.NoError(t, addPersonIP(ctx, p1, "10.0.0.2"), "failed to add ip record")
	require.NoError(t, addPersonIP(ctx, p1, "10.0.0.3"), "failed to add 2nd ip record")
	ipsUpdated := getIPHistory(ctx, p1.SteamID)
	require.True(t, len(ipsUpdated)-len(ips) == 2)
	require.NoError(t, dropPerson(ctx, p1.SteamID))
}

func TestFilters(t *testing.T) {
	existingFilters, err := getFilters(context.Background())
	require.NoError(t, err)
	words := []string{golib.RandomString(10), golib.RandomString(10)}
	var savedFilters []*model.Filter
	for _, word := range words {
		f, e := insertFilter(context.Background(), word)
		require.NoError(t, e, "Failed to insert filter: %s", word)
		require.True(t, f.WordID > 0)
		savedFilters = append(savedFilters, f)
	}
	currentFilters, err := getFilters(context.Background())
	require.NoError(t, err)
	require.Equal(t, len(existingFilters)+len(words), len(currentFilters))
	if savedFilters != nil {
		require.NoError(t, dropFilter(context.Background(), savedFilters[0]))
		byId, errId := getFilterByID(context.Background(), savedFilters[1].WordID)
		require.NoError(t, errId)
		require.Equal(t, savedFilters[1].WordID, byId.WordID)
		require.Equal(t, savedFilters[1].Word.String(), byId.Word.String())
	}
	droppedFilters, err := getFilters(context.Background())
	require.NoError(t, err)
	require.Equal(t, len(existingFilters)+len(words)-1, len(droppedFilters))

}
