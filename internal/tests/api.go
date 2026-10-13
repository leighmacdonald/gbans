package tests

import (
	"context"
	"time"

	bansv1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/bans/v1"
	bdv1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/bd/v1"
	leaguesv1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/leagues/v1"
	logstfv1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/logstf/v1"
	metav1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/meta/v1"
	statsv1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/stats/v1"
	steamv1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/steam/v1"
	steamrepv1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/steamrep/v1"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const DefaultAvatarHash = "fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb"

type FakeTFAPI struct{}

func (f FakeTFAPI) SteamGroup(_ context.Context, groupID steamid.SteamID) (*steamv1.SteamGroup, error) {
	return &steamv1.SteamGroup{
		GroupId:   groupID.String(),
		GroupName: "test-group",
		CreatedOn: timestamppb.New(time.Now()),
		UpdatedOn: timestamppb.New(time.Now()),
	}, nil
}

func (f FakeTFAPI) LogsTFSummary(_ context.Context, _ steamid.SteamID) (*logstfv1.PlayerSummary, error) {
	return &logstfv1.PlayerSummary{}, nil
}

func (f FakeTFAPI) LogsTFLog(_ context.Context, logID int32) (*logstfv1.LogsTFMatch, error) {
	return &logstfv1.LogsTFMatch{LogId: logID}, nil
}

func (f FakeTFAPI) LogsTFMatchList(_ context.Context, _ steamid.SteamID) ([]*logstfv1.MatchInfo, error) {
	return []*logstfv1.MatchInfo{}, nil
}

func (f FakeTFAPI) LogsTFChatQuery(_ context.Context, _ *logstfv1.ChatQueryRequest) ([]*logstfv1.Chat, error) {
	return []*logstfv1.Chat{}, nil
}

func (f FakeTFAPI) MetaProfile(_ context.Context, steamIDs steamid.Collection) ([]*metav1.MetaProfile, error) {
	resp := make([]*metav1.MetaProfile, len(steamIDs))
	for index, steamID := range steamIDs {
		resp[index] = &metav1.MetaProfile{
			SteamId:    steamID.String(),
			AvatarHash: DefaultAvatarHash,
			EconomyBan: "none",
		}
	}

	return resp, nil
}

func (f FakeTFAPI) MetaSites(_ context.Context) ([]*metav1.SiteInfo, error) {
	return []*metav1.SiteInfo{}, nil
}

func (f FakeTFAPI) SteamBans(_ context.Context, steamIDs steamid.Collection) ([]*steamv1.SteamBan, error) {
	resp := make([]*steamv1.SteamBan, len(steamIDs))

	for index, steamID := range steamIDs {
		resp[index] = &steamv1.SteamBan{
			CommunityBanned:  false,
			DaysSinceLastBan: 0,
			EconomyBan:       "none",
			NumberOfGameBans: 0,
			NumberOfVacBans:  0,
			SteamId:          steamID.String(),
			VacBanned:        false,
		}
	}

	return resp, nil
}

func (f FakeTFAPI) Summaries(_ context.Context, steamIDs steamid.Collection) ([]*steamv1.PlayerSummary, error) {
	resp := make([]*steamv1.PlayerSummary, len(steamIDs))

	for index, steamID := range steamIDs {
		resp[index] = &steamv1.PlayerSummary{
			AvatarHash:      DefaultAvatarHash,
			PersonaName:     "name-" + steamID.String(),
			ProfileState:    1,
			RealName:        "randy marsh",
			SteamId:         steamID.String(),
			TimeCreated:     int32(time.Now().Add(-500 * time.Hour).Unix()), //nolint:gosec
			VisibilityState: 5,
		}
	}

	return resp, nil
}

func (f FakeTFAPI) Friends(_ context.Context, _ steamid.SteamID) ([]*steamv1.SteamFriend, error) {
	return []*steamv1.SteamFriend{}, nil
}

func (f FakeTFAPI) OwnedGames(_ context.Context, _ steamid.Collection) (map[string]*steamv1.OwnedGames, error) {
	return map[string]*steamv1.OwnedGames{}, nil
}

func (f FakeTFAPI) ResolveID(_ context.Context, steamID string) (*steamv1.ResolvedSteamId, error) {
	return &steamv1.ResolvedSteamId{Steam64: steamID}, nil
}

func (f FakeTFAPI) ParseIDs(_ context.Context, _, _, _ string) ([]string, error) {
	return []string{}, nil
}

func (f FakeTFAPI) SchemaItems(_ context.Context) ([]*steamv1.SchemaItem, error) {
	return []*steamv1.SchemaItem{}, nil
}

func (f FakeTFAPI) BansSearch(
	_ context.Context,
	_ steamid.Collection,
	_ string,
	_ bool,
) ([]*bansv1.Ban, error) {
	return []*bansv1.Ban{}, nil
}

func (f FakeTFAPI) LeagueHistory(
	_ context.Context,
	_ steamid.Collection,
	_, _, _ string,
) ([]*leaguesv1.PlayerTeamHistory, error) {
	return []*leaguesv1.PlayerTeamHistory{}, nil
}

func (f FakeTFAPI) LeagueCompetitions(_ context.Context, _, _, _ string) ([]*leaguesv1.League, error) {
	return []*leaguesv1.League{}, nil
}

func (f FakeTFAPI) LeagueTeams(_ context.Context, _ string, _ int64, _ string) ([]*leaguesv1.Team, error) {
	return []*leaguesv1.Team{}, nil
}

func (f FakeTFAPI) LeagueTeamMembers(_ context.Context, _ string, _ int64) ([]*leaguesv1.TeamMember, error) {
	return []*leaguesv1.TeamMember{}, nil
}

func (f FakeTFAPI) BDSearch(
	_ context.Context,
	_ steamid.Collection,
	_ []string,
) ([]*bdv1.BDSearchResult, error) {
	return []*bdv1.BDSearchResult{}, nil
}

func (f FakeTFAPI) BDGenerate(_ context.Context, _ string) (*bdv1.BDSchema, error) {
	return &bdv1.BDSchema{}, nil
}

func (f FakeTFAPI) StatsOverall(_ context.Context) (*statsv1.OverallResponse, error) {
	return &statsv1.OverallResponse{}, nil
}

func (f FakeTFAPI) SteamRepQuery(
	_ context.Context,
	_ steamid.Collection,
) ([]*steamrepv1.SteamRepEntry, error) {
	return []*steamrepv1.SteamRepEntry{}, nil
}

func (f FakeTFAPI) CreateAPIKey(_ context.Context) (string, error) {
	return "test-key", nil
}
