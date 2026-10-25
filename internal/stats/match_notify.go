package stats

import (
	"context"
	"log/slog"

	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type matchPlayerFinder interface {
	FindPlayer(opts servers.FindOpts) (servers.FindResult, bool)
}

type matchSettingsReader interface {
	GetPersonSettings(ctx context.Context, steamID steamid.SteamID) (person.Settings, error)
}

type matchSayFn func(ctx context.Context, server *servers.Server, opts servers.SayOpts) error

type matchPlayerNotifier struct {
	servers        matchPlayerFinder
	persons        matchSettingsReader
	sendPrivateMsg matchSayFn
}

// NewMatchPlayerNotifier creates a notifier that sends an in-game private
// message to match participants who have opted in and are still connected to
// one of the currently tracked servers.
func NewMatchPlayerNotifier(serverCollection *servers.Servers, persons *person.Persons) MatchNotifier {
	return &matchPlayerNotifier{
		servers: serverCollection,
		persons: persons,
		sendPrivateMsg: func(ctx context.Context, server *servers.Server, opts servers.SayOpts) error {
			return server.Say(ctx, opts)
		},
	}
}

func (n *matchPlayerNotifier) NotifyMatch(ctx context.Context, match *Match) {
	if match == nil {
		return
	}

	targets := make(map[*servers.Server][]steamid.SteamID)
	seen := make(map[steamid.SteamID]struct{})

	for _, player := range match.Players {
		steamID := player.SteamID
		if !steamID.Valid() {
			continue
		}

		if _, exists := seen[steamID]; exists {
			continue
		}
		seen[steamID] = struct{}{}

		settings, errSettings := n.persons.GetPersonSettings(ctx, steamID)
		if errSettings != nil {
			slog.Warn("Failed to load person settings for match notification",
				slog.String("steam_id", steamID.String()),
				slog.String("error", errSettings.Error()))

			continue
		}

		if !settings.MatchNotifications {
			continue
		}

		result, found := n.servers.FindPlayer(servers.FindOpts{SteamID: steamID})
		if !found {
			continue
		}

		targets[result.Server] = append(targets[result.Server], steamID)
	}

	if len(targets) == 0 {
		return
	}

	message := "Your match results are ready: " + link.Path(match)

	for server, steamIDs := range targets {
		if errSend := n.sendPrivateMsg(ctx, server, servers.SayOpts{
			Type:    servers.PSay,
			Targets: steamIDs,
			Message: message,
		}); errSend != nil {
			slog.Warn("Failed to send match notification",
				slog.Int("server_id", int(server.ServerID)),
				slog.String("error", errSend.Error()))
		}
	}
}
