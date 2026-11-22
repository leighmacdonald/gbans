package ban_test

import (
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/discord"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

// checkView mirrors the (unexported) view the check template renders against.
type checkView struct {
	Author  personDomain.Info
	Player  personDomain.Info
	SteamID string
	Ban     ban.Ban
	Old     []ban.Ban
}

func TestCheckTemplate_RendersPermissions(t *testing.T) {
	ban.RegisterDiscordCommands(discord.Discard{}, ban.Bans{}, nil, nil)

	player := personDomain.Core{
		SteamID:     steamid.New(76561198000000002),
		Name:        "Cheater",
		Permissions: []rolesv1.Permission{rolesv1.Permission_PERMISSION_BAN_READ},
	}
	author := personDomain.Core{
		SteamID: steamid.New(76561198000000003),
		Name:    "Modder",
	}

	view := checkView{
		Author:  author,
		Player:  player,
		SteamID: player.SteamID.String(),
		// BanID 0 takes the "No Ban Found!" branch, so the active-ban fields
		// are not evaluated by the template.
		Ban: ban.Ban{
			BanID:      0,
			Reason:     reason.Cheating,
			ValidUntil: time.Now().Add(24 * time.Hour),
			CreatedOn:  time.Now(),
		},
	}

	out, err := discord.RenderTemplate("check", view)
	require.NoError(t, err)
	require.Contains(t, out, "PERMISSION_BAN_READ")
	require.NotContains(t, out, "Failed to render")
}

func TestBanSuccessTemplate(t *testing.T) {
	ban.RegisterDiscordCommands(discord.Discard{}, ban.Bans{}, nil, nil)

	view := struct {
		Mute          bool
		Link          string
		AuthorName    string
		AuthorLink    string
		AuthorSteamID string
	}{Mute: true, Link: "http://localhost/ban/1", AuthorName: "Mod", AuthorLink: "http://localhost/profile/1", AuthorSteamID: "76561198000000001"}

	out, err := discord.RenderTemplate("ban_success", view)
	require.NoError(t, err)
	require.Contains(t, out, "Mute successful")
	require.Contains(t, out, "http://localhost/ban/1")
	require.Contains(t, out, "Mod")
}

func TestBanResponseTemplateIncludesAuthor(t *testing.T) {
	ban.RegisterDiscordCommands(discord.Discard{}, ban.Bans{}, nil, nil)

	view := struct {
		Ban           ban.Ban
		Player        personDomain.Info
		Author        personDomain.Info
		SteamIDAuthor string
		SteamID       string
		ExpIn         string
		ExpAt         string
	}{
		Ban:           ban.Ban{BanID: 42, BanType: 2},
		Player:        personDomain.Core{SteamID: steamid.New(76561198000000002), Name: "Cheater"},
		Author:        personDomain.Core{SteamID: steamid.New(76561198000000003), Name: "Modder"},
		SteamIDAuthor: "76561198000000003",
		SteamID:       "76561198000000002",
		ExpIn:         "Permanent",
		ExpAt:         "Permanent",
	}

	out, err := discord.RenderTemplate("ban_response", view)
	require.NoError(t, err)
	require.Contains(t, out, "Cheater")
	require.Contains(t, out, "Modder")
	require.Contains(t, out, "76561198000000003")
	require.NotContains(t, out, "Failed to render")
}

func TestReportNewTemplate(t *testing.T) {
	ban.RegisterDiscordCommands(discord.Discard{}, ban.Bans{}, nil, nil)

	view := struct {
		Report ban.ReportWithAuthor
	}{
		Report: ban.ReportWithAuthor{
			Report:  ban.Report{ReportID: 1, Reason: reason.Cheating, ReasonText: "aimbot"},
			Author:  personDomain.Core{SteamID: steamid.New(76561198000000011), Name: "Mod"},
			Subject: personDomain.Core{SteamID: steamid.New(76561198000000010), Name: "Cheater"},
		},
	}

	out, err := discord.RenderTemplate("report_new", view)
	require.NoError(t, err)
	require.Contains(t, out, "aimbot")
	require.Contains(t, out, "Cheater")
	require.NotContains(t, out, "Failed to render")
}
