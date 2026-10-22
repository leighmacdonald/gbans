package stats

import (
	_ "embed"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

//go:embed stats_discord.gotmpl
var templateBody []byte

func RegisterDiscordCommands(_ discord.Connection) {
	discord.MustRegisterTemplate(templateBody)
}

type matchDiscordPlayer struct {
	Name     string
	SteamURL string
	Points   uint64
	Kills    uint64
	Assists  uint64
	Deaths   uint64
	Damage   string
}

type matchDiscordView struct {
	MatchID     string
	MapName     string
	ServerName  string
	ScoreRed    uint32
	ScoreBlu    uint32
	Outcome     string
	Duration    string
	StartTime   time.Time
	Rounds      int
	PlayerCount int
	RedPlayers  []matchDiscordPlayer
	BluPlayers  []matchDiscordPlayer
	MatchURL    string
	DemoURL     string
}

type matchPlayerKey struct {
	steamID steamid.SteamID
	team    string
}

type matchPlayerTotals struct {
	steamID steamid.SteamID
	team    string
	name    string
	points  uint64
	kills   uint64
	assists uint64
	deaths  uint64
	damage  uint64
}

// MatchMessage renders a summary of an imported match for the public match log
// channel.
func MatchMessage(match Match) *discordgo.MessageSend {
	view := newMatchDiscordView(match)

	content, errContent := discord.RenderTemplate("match_imported", view)
	if errContent != nil {
		slog.Error("Failed to render match message", slog.String("error", errContent.Error()))
	}

	buttons := []discordgo.MessageComponent{
		discord.Link("View Match", view.MatchURL),
	}
	if view.DemoURL != "" {
		buttons = append(buttons, discord.Link("Download STV", view.DemoURL))
	}

	return discord.NewMessage(
		discord.BodyColouredText(discord.ColourSuccess, content),
		discord.Buttons(buttons...),
	)
}

func newMatchDiscordView(match Match) matchDiscordView {
	validRounds := validMatchRounds(match)
	totals, playerCount := aggregateMatchPlayers(match, validRounds)

	redPlayers := make([]matchDiscordPlayer, 0, len(totals))
	bluPlayers := make([]matchDiscordPlayer, 0, len(totals))

	for _, total := range totals {
		player := matchDiscordPlayer{
			Name:     total.name,
			SteamURL: link.Raw("/profile/" + strconv.FormatInt(total.steamID.Int64(), 10)),
			Points:   total.points,
			Kills:    total.kills,
			Assists:  total.assists,
			Deaths:   total.deaths,
			Damage:   humanize.Comma(int64(total.damage)), //nolint:gosec // Match stats are non-negative and small.
		}

		switch total.team {
		case "red":
			redPlayers = append(redPlayers, player)
		case "blu":
			bluPlayers = append(bluPlayers, player)
		}
	}

	sortMatchPlayers(redPlayers)
	sortMatchPlayers(bluPlayers)

	roundCount := len(validRounds)
	if roundCount == 0 {
		roundCount = len(match.Rounds)
	}

	view := matchDiscordView{
		MatchID:     match.MatchID.String(),
		MapName:     match.MapName,
		ServerName:  matchServerName(match),
		ScoreRed:    match.ScoreRed,
		ScoreBlu:    match.ScoreBlu,
		Outcome:     matchOutcome(match),
		Duration:    matchDuration(match.DurationMs),
		StartTime:   match.StartTime,
		Rounds:      roundCount,
		PlayerCount: playerCount,
		RedPlayers:  topMatchPlayers(redPlayers, 3),
		BluPlayers:  topMatchPlayers(bluPlayers, 3),
		MatchURL:    link.Path(match),
	}

	if !match.AssetID.IsNil() {
		view.DemoURL = link.Raw("/asset/" + match.AssetID.String())
	}

	return view
}

// validMatchRounds mirrors the frontend summary by ignoring zero-duration stub
// rounds. When no round has a duration, all player rows are retained.
func validMatchRounds(match Match) map[uint32]struct{} {
	rounds := make(map[uint32]struct{}, len(match.Rounds))
	for _, round := range match.Rounds {
		if round.DurationMs > 0 {
			rounds[round.RoundID] = struct{}{}
		}
	}

	if len(rounds) == 0 {
		for _, round := range match.Rounds {
			rounds[round.RoundID] = struct{}{}
		}
	}

	return rounds
}

func aggregateMatchPlayers(match Match, validRounds map[uint32]struct{}) ([]matchPlayerTotals, int) {
	totals := make(map[matchPlayerKey]*matchPlayerTotals)
	players := make(map[steamid.SteamID]struct{})

	for _, row := range match.Players {
		if len(validRounds) > 0 {
			if _, ok := validRounds[row.RoundID]; !ok {
				continue
			}
		}

		team := strings.ToLower(row.Team)
		if team != "red" && team != "blu" {
			continue
		}

		key := matchPlayerKey{steamID: row.SteamID, team: team}

		total, ok := totals[key]
		if !ok {
			name := row.Personaname
			if name == "" {
				name = row.SteamID.String()
			}

			total = &matchPlayerTotals{steamID: row.SteamID, team: team, name: name}
			totals[key] = total
		}

		total.points += row.Points
		total.kills += row.Kills
		total.assists += row.Assists
		total.deaths += row.ScoreboardDeaths
		total.damage += row.ScoreboardDamage
		players[row.SteamID] = struct{}{}
	}

	out := make([]matchPlayerTotals, 0, len(totals))
	for _, total := range totals {
		out = append(out, *total)
	}

	return out, len(players)
}

func sortMatchPlayers(players []matchDiscordPlayer) {
	sort.SliceStable(players, func(left, right int) bool {
		if players[left].Points != players[right].Points {
			return players[left].Points > players[right].Points
		}

		if players[left].Kills != players[right].Kills {
			return players[left].Kills > players[right].Kills
		}

		if players[left].Deaths != players[right].Deaths {
			return players[left].Deaths < players[right].Deaths
		}

		return players[left].Name < players[right].Name
	})
}

func topMatchPlayers(players []matchDiscordPlayer, limit int) []matchDiscordPlayer {
	if len(players) <= limit {
		return players
	}

	return players[:limit]
}

func matchServerName(match Match) string {
	if match.ServerNameShort != "" {
		return match.ServerNameShort
	}

	return match.ServerName
}

func matchOutcome(match Match) string {
	switch {
	case match.ScoreRed > match.ScoreBlu:
		return "RED wins"
	case match.ScoreBlu > match.ScoreRed:
		return "BLU wins"
	default:
		return "Draw"
	}
}

func matchDuration(durationMs uint64) string {
	duration := time.Duration(durationMs) * time.Millisecond //nolint:gosec // Match durations are non-negative and small.

	return duration.Round(time.Second).String()
}
