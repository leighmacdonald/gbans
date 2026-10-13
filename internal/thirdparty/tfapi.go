package thirdparty

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"
	authv1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/auth/v1"
	"github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/auth/v1/authv1connect"
	bansv1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/bans/v1"
	"github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/bans/v1/bansv1connect"
	bdv1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/bd/v1"
	"github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/bd/v1/bdv1connect"
	leaguesv1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/leagues/v1"
	"github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/leagues/v1/leaguesv1connect"
	logstfv1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/logstf/v1"
	"github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/logstf/v1/logstfv1connect"
	metav1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/meta/v1"
	"github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/meta/v1/metav1connect"
	statsv1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/stats/v1"
	"github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/stats/v1/statsv1connect"
	steamv1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/steam/v1"
	"github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/steam/v1/steamv1connect"
	steamrepv1 "github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/steamrep/v1"
	"github.com/leighmacdonald/gbans/internal/thirdparty/tfapi/steamrep/v1/steamrepv1connect"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var ErrNoResult = errors.New("no results")

// APIProvider abstracts the tf-api backing service.
type APIProvider interface {
	// Steam
	SteamBans(ctx context.Context, steamIDs steamid.Collection) ([]*steamv1.SteamBan, error)
	Summaries(ctx context.Context, steamIDs steamid.Collection) ([]*steamv1.PlayerSummary, error)
	Friends(ctx context.Context, steamID steamid.SteamID) ([]*steamv1.SteamFriend, error)
	SteamGroup(ctx context.Context, groupID steamid.SteamID) (*steamv1.SteamGroup, error)
	OwnedGames(ctx context.Context, steamIDs steamid.Collection) (map[string]*steamv1.OwnedGames, error)
	ResolveID(ctx context.Context, steamID string) (*steamv1.ResolvedSteamId, error)
	ParseIDs(ctx context.Context, text, format, typ string) ([]string, error)
	SchemaItems(ctx context.Context) ([]*steamv1.SchemaItem, error)
	// LogsTF
	LogsTFSummary(ctx context.Context, steamID steamid.SteamID) (*logstfv1.PlayerSummary, error)
	LogsTFLog(ctx context.Context, logID int32) (*logstfv1.LogsTFMatch, error)
	LogsTFMatchList(ctx context.Context, steamID steamid.SteamID) ([]*logstfv1.MatchInfo, error)
	LogsTFChatQuery(ctx context.Context, req *logstfv1.ChatQueryRequest) ([]*logstfv1.Chat, error)
	// Meta
	MetaProfile(ctx context.Context, steamIDs steamid.Collection) ([]*metav1.MetaProfile, error)
	MetaSites(ctx context.Context) ([]*metav1.SiteInfo, error)
	// Bans
	BansSearch(ctx context.Context, steamIDs steamid.Collection, siteName string, hideUnbanned bool) ([]*bansv1.Ban, error)
	// Leagues
	LeagueHistory(ctx context.Context, steamIDs steamid.Collection, league, typ, format string) ([]*leaguesv1.PlayerTeamHistory, error)
	LeagueCompetitions(ctx context.Context, league, format, typ string) ([]*leaguesv1.League, error)
	LeagueTeams(ctx context.Context, league string, leagueID int64, name string) ([]*leaguesv1.Team, error)
	LeagueTeamMembers(ctx context.Context, league string, leagueID int64) ([]*leaguesv1.TeamMember, error)
	// BD
	BDSearch(ctx context.Context, steamIDs steamid.Collection, attrs []string) ([]*bdv1.BDSearchResult, error)
	BDGenerate(ctx context.Context, site string) (*bdv1.BDSchema, error)
	// Stats
	StatsOverall(ctx context.Context) (*statsv1.OverallResponse, error)
	// SteamRep
	SteamRepQuery(ctx context.Context, steamIDs steamid.Collection) ([]*steamrepv1.SteamRepEntry, error)
	// Auth
	CreateAPIKey(ctx context.Context) (string, error)
}

// TFAPI provides a trivial interface around the generated tf-api Connect clients
// just to make any future api changes a bit easier as its still a bit of a moving target.
type TFAPI struct {
	steam    steamv1connect.SteamServiceClient
	logstf   logstfv1connect.LogsTFServiceClient
	meta     metav1connect.MetaServiceClient
	bans     bansv1connect.BansServiceClient
	leagues  leaguesv1connect.LeaguesServiceClient
	bd       bdv1connect.BDServiceClient
	stats    statsv1connect.StatsServiceClient
	steamRep steamrepv1connect.SteamRepServiceClient
	auth     authv1connect.AuthServiceClient
}

func NewTFAPI(host string, client *http.Client) (*TFAPI, error) {
	if client == nil {
		client = &http.Client{Timeout: time.Second * 15}
	}

	var httpClient connect.HTTPClient = client

	return &TFAPI{
		steam:    steamv1connect.NewSteamServiceClient(httpClient, host),
		logstf:   logstfv1connect.NewLogsTFServiceClient(httpClient, host),
		meta:     metav1connect.NewMetaServiceClient(httpClient, host),
		bans:     bansv1connect.NewBansServiceClient(httpClient, host),
		leagues:  leaguesv1connect.NewLeaguesServiceClient(httpClient, host),
		bd:       bdv1connect.NewBDServiceClient(httpClient, host),
		stats:    statsv1connect.NewStatsServiceClient(httpClient, host),
		steamRep: steamrepv1connect.NewSteamRepServiceClient(httpClient, host),
		auth:     authv1connect.NewAuthServiceClient(httpClient, host),
	}, nil
}

func (c *TFAPI) SteamBans(ctx context.Context, steamIDs steamid.Collection) ([]*steamv1.SteamBan, error) {
	resp, err := c.steam.GetBans(ctx, &steamv1.GetBansRequest{SteamIds: steamIDs.ToStringSlice()})
	if err != nil {
		return nil, err
	}

	if resp.GetBans() == nil {
		return nil, ErrNoResult
	}

	return resp.GetBans(), nil
}

func (c *TFAPI) Summaries(ctx context.Context, steamIDs steamid.Collection) ([]*steamv1.PlayerSummary, error) {
	resp, err := c.steam.GetSummary(ctx, &steamv1.GetSummaryRequest{SteamIds: steamIDs.ToStringSlice()})
	if err != nil {
		return nil, err
	}

	if resp.GetSummaries() == nil {
		return nil, ErrNoResult
	}

	return resp.GetSummaries(), nil
}

func (c *TFAPI) Friends(ctx context.Context, steamID steamid.SteamID) ([]*steamv1.SteamFriend, error) {
	resp, err := c.steam.GetFriends(ctx, &steamv1.GetFriendsRequest{SteamId: steamID.String()})
	if err != nil {
		return nil, err
	}

	if resp.GetFriends() == nil {
		return nil, ErrNoResult
	}

	return resp.GetFriends(), nil
}

func (c *TFAPI) SteamGroup(ctx context.Context, groupID steamid.SteamID) (*steamv1.SteamGroup, error) {
	resp, err := c.steam.GetGroup(ctx, &steamv1.GetGroupRequest{GroupId: groupID.String()})
	if err != nil {
		return nil, err
	}

	if resp.GetGroup() == nil {
		return nil, ErrNoResult
	}

	return resp.GetGroup(), nil
}

func (c *TFAPI) OwnedGames(ctx context.Context, steamIDs steamid.Collection) (map[string]*steamv1.OwnedGames, error) {
	resp, err := c.steam.GetOwnedGames(ctx, &steamv1.GetOwnedGamesRequest{SteamIds: steamIDs.ToStringSlice()})
	if err != nil {
		return nil, err
	}

	return resp.GetGames(), nil
}

func (c *TFAPI) ResolveID(ctx context.Context, steamID string) (*steamv1.ResolvedSteamId, error) {
	resp, err := c.steam.ResolveId(ctx, &steamv1.ResolveIdRequest{SteamId: steamID})
	if err != nil {
		return nil, err
	}

	if resp.GetId() == nil {
		return nil, ErrNoResult
	}

	return resp.GetId(), nil
}

func (c *TFAPI) ParseIDs(ctx context.Context, text, format, typ string) ([]string, error) {
	resp, err := c.steam.Parse(ctx, &steamv1.ParseRequest{Text: text, Format: format, Type: typ})
	if err != nil {
		return nil, err
	}

	return resp.GetSteamIds(), nil
}

func (c *TFAPI) SchemaItems(ctx context.Context) ([]*steamv1.SchemaItem, error) {
	resp, err := c.steam.GetSchemaItems(ctx, &steamv1.GetSchemaItemsRequest{})
	if err != nil {
		return nil, err
	}

	return resp.GetItems(), nil
}

func (c *TFAPI) LogsTFSummary(ctx context.Context, steamID steamid.SteamID) (*logstfv1.PlayerSummary, error) {
	resp, err := c.logstf.PlayerSummary(ctx, &logstfv1.PlayerSummaryRequest{SteamId: steamID.String()})
	if err != nil {
		return nil, err
	}

	if resp.GetSummary() == nil {
		return nil, ErrNoResult
	}

	return resp.GetSummary(), nil
}

func (c *TFAPI) LogsTFLog(ctx context.Context, logID int32) (*logstfv1.LogsTFMatch, error) {
	resp, err := c.logstf.Log(ctx, &logstfv1.LogRequest{LogId: logID})
	if err != nil {
		return nil, err
	}

	if resp.GetMatch() == nil {
		return nil, ErrNoResult
	}

	return resp.GetMatch(), nil
}

func (c *TFAPI) LogsTFMatchList(ctx context.Context, steamID steamid.SteamID) ([]*logstfv1.MatchInfo, error) {
	resp, err := c.logstf.MatchList(ctx, &logstfv1.MatchListRequest{SteamId: steamID.String()})
	if err != nil {
		return nil, err
	}

	return resp.GetMatches(), nil
}

func (c *TFAPI) LogsTFChatQuery(ctx context.Context, req *logstfv1.ChatQueryRequest) ([]*logstfv1.Chat, error) {
	resp, err := c.logstf.ChatQuery(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.GetMessages(), nil
}

func (c *TFAPI) MetaProfile(ctx context.Context, steamIDs steamid.Collection) ([]*metav1.MetaProfile, error) {
	resp, err := c.meta.Profile(ctx, &metav1.ProfileRequest{SteamIds: steamIDs.ToStringSlice()})
	if err != nil {
		return nil, err
	}

	if resp.GetProfiles() == nil {
		return nil, ErrNoResult
	}

	return resp.GetProfiles(), nil
}

func (c *TFAPI) MetaSites(ctx context.Context) ([]*metav1.SiteInfo, error) {
	resp, err := c.meta.Sites(ctx, &metav1.SitesRequest{})
	if err != nil {
		return nil, err
	}

	return resp.GetSites(), nil
}

func (c *TFAPI) BansSearch(
	ctx context.Context,
	steamIDs steamid.Collection,
	siteName string,
	hideUnbanned bool,
) ([]*bansv1.Ban, error) {
	resp, err := c.bans.Search(ctx, &bansv1.SearchRequest{
		SteamIds:     steamIDs.ToStringSlice(),
		SiteName:     siteName,
		HideUnbanned: hideUnbanned,
	})
	if err != nil {
		return nil, err
	}

	return resp.GetBans(), nil
}

func (c *TFAPI) LeagueHistory(
	ctx context.Context,
	steamIDs steamid.Collection,
	league, typ, format string,
) ([]*leaguesv1.PlayerTeamHistory, error) {
	resp, err := c.leagues.History(ctx, &leaguesv1.HistoryRequest{
		SteamIds: steamIDs.ToStringSlice(),
		League:   league,
		Type:     typ,
		Format:   format,
	})
	if err != nil {
		return nil, err
	}

	return resp.GetHistory(), nil
}

func (c *TFAPI) LeagueCompetitions(ctx context.Context, league, format, typ string) ([]*leaguesv1.League, error) {
	resp, err := c.leagues.Competitions(ctx, &leaguesv1.CompetitionsRequest{
		League: league,
		Format: format,
		Type:   typ,
	})
	if err != nil {
		return nil, err
	}

	return resp.GetCompetitions(), nil
}

func (c *TFAPI) LeagueTeams(ctx context.Context, league string, leagueID int64, name string) ([]*leaguesv1.Team, error) {
	resp, err := c.leagues.Teams(ctx, &leaguesv1.TeamsRequest{
		League:   league,
		LeagueId: leagueID,
		Name:     name,
	})
	if err != nil {
		return nil, err
	}

	return resp.GetTeams(), nil
}

func (c *TFAPI) LeagueTeamMembers(ctx context.Context, league string, leagueID int64) ([]*leaguesv1.TeamMember, error) {
	resp, err := c.leagues.TeamMembers(ctx, &leaguesv1.TeamMembersRequest{
		League:   league,
		LeagueId: leagueID,
	})
	if err != nil {
		return nil, err
	}

	return resp.GetMembers(), nil
}

func (c *TFAPI) BDSearch(
	ctx context.Context,
	steamIDs steamid.Collection,
	attrs []string,
) ([]*bdv1.BDSearchResult, error) {
	resp, err := c.bd.SearchLists(ctx, &bdv1.SearchListsRequest{
		SteamIds: steamIDs.ToStringSlice(),
		Attrs:    attrs,
	})
	if err != nil {
		return nil, err
	}

	return resp.GetResults(), nil
}

func (c *TFAPI) BDGenerate(ctx context.Context, site string) (*bdv1.BDSchema, error) {
	resp, err := c.bd.Generate(ctx, &bdv1.GenerateRequest{Site: site})
	if err != nil {
		return nil, err
	}

	if resp.GetSchema() == nil {
		return nil, ErrNoResult
	}

	return resp.GetSchema(), nil
}

func (c *TFAPI) StatsOverall(ctx context.Context) (*statsv1.OverallResponse, error) {
	return c.stats.Overall(ctx, &statsv1.OverallRequest{})
}

func (c *TFAPI) SteamRepQuery(
	ctx context.Context,
	steamIDs steamid.Collection,
) ([]*steamrepv1.SteamRepEntry, error) {
	resp, err := c.steamRep.Query(ctx, &steamrepv1.QueryRequest{SteamIds: steamIDs.ToStringSlice()})
	if err != nil {
		return nil, err
	}

	return resp.GetEntries(), nil
}

func (c *TFAPI) CreateAPIKey(ctx context.Context) (string, error) {
	resp, err := c.auth.CreateAPIKey(ctx, &authv1.CreateAPIKeyRequest{})
	if err != nil {
		return "", err
	}

	return resp.GetKey(), nil
}
