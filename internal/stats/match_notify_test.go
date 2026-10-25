package stats //nolint:testpackage // Tests exercise the unexported notifier implementation directly.

import (
	"context"
	"strings"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

type fakeMatchPlayerFinder struct {
	players map[steamid.SteamID]servers.FindResult
}

func (f fakeMatchPlayerFinder) FindPlayer(opts servers.FindOpts) (servers.FindResult, bool) {
	result, found := f.players[opts.SteamID]

	return result, found
}

type fakeMatchSettingsReader struct {
	settings map[steamid.SteamID]person.Settings
}

func (f fakeMatchSettingsReader) GetPersonSettings(_ context.Context, steamID steamid.SteamID) (person.Settings, error) {
	return f.settings[steamID], nil
}

func TestMatchPlayerNotifier(t *testing.T) {
	t.Parallel()

	serverA := &servers.Server{ServerID: 1}
	serverB := &servers.Server{ServerID: 2}

	optedIn := steamid.New(76561197960265729)
	optedOut := steamid.New(76561197960265730)
	optedInOtherServer := steamid.New(76561197960265731)
	optedInOffline := steamid.New(76561197960265732)

	match := &Match{
		MatchID: uuid.Must(uuid.NewV4()),
		Players: []MatchOverallStatsRound{
			{SteamID: optedIn},
			// Duplicate rows for the same player must only produce one message.
			{SteamID: optedIn},
			{SteamID: optedOut},
			{SteamID: optedInOtherServer},
			{SteamID: optedInOffline},
		},
	}

	notifier := &matchPlayerNotifier{
		servers: fakeMatchPlayerFinder{players: map[steamid.SteamID]servers.FindResult{
			optedIn:            {Server: serverA},
			optedOut:           {Server: serverA},
			optedInOtherServer: {Server: serverB},
		}},
		persons: fakeMatchSettingsReader{settings: map[steamid.SteamID]person.Settings{
			optedIn:            {MatchNotifications: true},
			optedOut:           {MatchNotifications: false},
			optedInOtherServer: {MatchNotifications: true},
			optedInOffline:     {MatchNotifications: true},
		}},
	}

	type sentMessage struct {
		server *servers.Server
		opts   servers.SayOpts
	}

	var sent []sentMessage
	notifier.sendPrivateMsg = func(_ context.Context, server *servers.Server, opts servers.SayOpts) error {
		sent = append(sent, sentMessage{server: server, opts: opts})

		return nil
	}

	notifier.NotifyMatch(t.Context(), match)

	require.Len(t, sent, 2)
	require.NotEmpty(t, sent)
	require.Contains(t, sent[0].opts.Message, match.Path())
	require.Equal(t, servers.PSay, sent[0].opts.Type)

	targets := map[steamid.SteamID]int{}
	for _, message := range sent {
		for _, target := range message.opts.Targets {
			targets[target]++
		}

		require.True(t, strings.HasPrefix(message.opts.Message, "Your match results are ready:"))
	}

	require.Equal(t, map[steamid.SteamID]int{
		optedIn:            1,
		optedInOtherServer: 1,
	}, targets)
}
