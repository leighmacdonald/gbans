package match

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/discord/helper"
	"github.com/leighmacdonald/gbans/internal/discord/message"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/match"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var slashCommands = []*discordgo.ApplicationCommand{
	{
		Name:        "log",
		Description: "Show a match log summary",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        helper.OptMatchID,
				Description: "MatchID of any previously uploaded match",
				Required:    true,
			},
		},
	},
	{
		Name:        "logs",
		Description: "Show a list of your recent logs",
		Options:     []*discordgo.ApplicationCommandOption{},
	},
	{
		Name:                     "find",
		DMPermission:             &helper.DmPerms,
		DefaultMemberPermissions: &helper.ModPerms,
		Description:              "Find a user on any of the servers",
		Options: []*discordgo.ApplicationCommandOption{
			&discordgo.ApplicationCommandOption{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        helper.OptUserIdentifier,
				Description: "SteamID in any format OR profile url",
				Required:    true,
			},
		},
	},
}

func makeOnStats() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		name := interaction.ApplicationCommandData().Options[0].Name
		switch name {
		case "player":
			return h.onStatsPlayer(ctx, session, interaction)
		// case string(cmdStatsGlobal):
		//	return discord.onStatsGlobal(ctx, session, interaction, response)
		// case string(cmdStatsServer):
		//	return discord.onStatsServer(ctx, session, interaction, response)
		default:
			return nil, helper.ErrCommandFailed
		}
	}
}

func onStatsPlayer(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := helper.OptionMap(interaction.ApplicationCommandData().Options[0].Options)

	steamID, errResolveSID := steamid.Resolve(ctx, opts[helper.OptUserIdentifier].StringValue())
	if errResolveSID != nil || !steamID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	person, errAuthor := h.persons.GetPersonBySteamID(ctx, nil, steamID)
	if errAuthor != nil {
		return nil, errAuthor
	}

	//
	// Person, errAuthor := getDiscordAuthor(ctx, app.db, interaction)
	// if errAuthor != nil {
	//	return nil, errAuthor
	// }

	classStats, errClassStats := h.matches.StatsPlayerClass(ctx, person.SteamID)
	if errClassStats != nil {
		return nil, errors.Join(errClassStats, domain.ErrFetchClassStats)
	}

	weaponStats, errWeaponStats := h.matches.StatsPlayerWeapons(ctx, person.SteamID)
	if errWeaponStats != nil {
		return nil, errors.Join(errWeaponStats, domain.ErrFetchWeaponStats)
	}

	killstreakStats, errKillstreakStats := h.matches.StatsPlayerKillstreaks(ctx, person.SteamID)
	if errKillstreakStats != nil {
		return nil, errors.Join(errKillstreakStats, domain.ErrFetchKillstreakStats)
	}

	medicStats, errMedicStats := h.matches.StatsPlayerMedic(ctx, person.SteamID)
	if errMedicStats != nil {
		return nil, errors.Join(errMedicStats, domain.ErrFetchMedicStats)
	}

	return StatsPlayerMessage(person, h.config.ExtURL(person), classStats, medicStats, weaponStats, killstreakStats), nil
}

//	func (discord *discord) onStatsServer(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, response *botResponse) error {
//		serverIdStr := interaction.Data.Options[0].Options[0].Value.(string)
//		var (
//			server model.ServerStore
//			stats  model.ServerStats
//		)
//		if errServer := discord.database.GetServerByName(ctx, serverIdStr, &server); errServer != nil {
//			return errServer
//		}
//		if errStats := discord.database.GetServerStats(ctx, server.ServerID, &stats); errStats != nil {
//			return errCommandFailed
//		}
//		acc := 0.0
//		if stats.Hits > 0 && stats.Shots > 0 {
//			acc = float64(stats.Hits) / float64(stats.Shots) * 100
//		}
//		embed := respOk(response, fmt.Sprintf("ServerStore stats for %s ", server.ShortName))
//		addFieldInline(embed, "Kills", fmt.Sprintf("%d", stats.Kills))
//		addFieldInline(embed, "Assists", fmt.Sprintf("%d", stats.Assists))
//		addFieldInline(embed, "Damage", fmt.Sprintf("%d", stats.Damage))
//		addFieldInline(embed, "MedicStats", fmt.Sprintf("%d", stats.MedicStats))
//		addFieldInline(embed, "Shots", fmt.Sprintf("%d", stats.Shots))
//		addFieldInline(embed, "Hits", fmt.Sprintf("%d", stats.Hits))
//		addFieldInline(embed, "Accuracy", fmt.Sprintf("%.2f%%", acc))
//		return nil
//	}
//
//	func (discord *discord) onStatsGlobal(ctx context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate, response *botResponse) error {
//		var stats model.GlobalStats
//		errStats := discord.database.GetGlobalStats(ctx, &stats)
//		if errStats != nil {
//			return errCommandFailed
//		}
//		acc := 0.0
//		if stats.Hits > 0 && stats.Shots > 0 {
//			acc = float64(stats.Hits) / float64(stats.Shots) * 100
//		}
//		embed := respOk(response, "Global stats")
//		addFieldInline(embed, "Kills", fmt.Sprintf("%d", stats.Kills))
//		addFieldInline(embed, "Assists", fmt.Sprintf("%d", stats.Assists))
//		addFieldInline(embed, "Damage", fmt.Sprintf("%d", stats.Damage))
//		addFieldInline(embed, "MedicStats", fmt.Sprintf("%d", stats.MedicStats))
//		addFieldInline(embed, "Shots", fmt.Sprintf("%d", stats.Shots))
//		addFieldInline(embed, "Hits", fmt.Sprintf("%d", stats.Hits))
//		addFieldInline(embed, "Accuracy", fmt.Sprintf("%.2f%%", acc))
//		return nil
//	}

func makeOnLogs() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		author, errAuthor := h.getDiscordAuthor(ctx, interaction)
		if errAuthor != nil {
			return nil, errAuthor
		}

		matches, count, errMatch := h.matches.Matches(ctx, match.MatchesQueryOpts{
			SteamID:     author.SteamID.String(),
			QueryFilter: domain.QueryFilter{Limit: 5},
		})

		if errMatch != nil {
			return nil, ErrCommandFailed
		}

		matchesWriter := &strings.Builder{}

		for _, match := range matches {
			status := ":x:"
			if match.IsWinner {
				status = ":white_check_mark:"
			}

			if _, err := fmt.Fprintf(matchesWriter, "%s [%s](%s) `%s` `%s`\n",
				status, match.Title, h.config.ExtURL(match), match.MapName, match.TimeStart.Format(time.DateOnly)); err != nil {
				slog.Error("Failed to write match line", log.ErrAttr(err))

				continue
			}
		}

		return message.LogsMessage(count, matchesWriter.String()), nil
	}
}

func makeOnLog() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		opts := OptionMap(interaction.ApplicationCommandData().Options)

		matchIDStr := opts[OptMatchID].StringValue()

		matchID, errMatchID := uuid.FromString(matchIDStr)
		if errMatchID != nil {
			return nil, ErrCommandFailed
		}

		var match match.MatchResult

		if errMatch := h.matches.MatchGetByID(ctx, matchID, &match); errMatch != nil {
			return nil, ErrCommandFailed
		}

		return message.MatchMessage(match, h.config.ExtURLRaw("/log/%s", match.MatchID.String())), nil
	}
}

func LogsMessage(count int64, matches string) *discordgo.MessageEmbed {
	return NewEmbed(fmt.Sprintf("Your most recent matches [%d total]", count)).Embed().
		SetColor(ColourSuccess).
		SetDescription(matches).MessageEmbed
}

func makeWeaponStatsTable(weapons []match.PlayerWeaponStats) string {
	writer := &strings.Builder{}
	table := defaultTable(writer)
	table.SetHeader([]string{"Weapon", "K", "Dmg", "Sh", "Hi", "Acc", "B", "H", "A"})

	sort.SliceStable(weapons, func(i, j int) bool {
		return weapons[i].Kills > weapons[j].Kills
	})

	for i, weapon := range weapons {
		if i == 10 {
			break
		}

		table.Append([]string{
			weapon.WeaponName,
			strconv.Itoa(weapon.Kills),
			strconv.Itoa(weapon.Damage),
			strconv.Itoa(weapon.Shots),
			strconv.Itoa(weapon.Hits),
			fmt.Sprintf("%.1f", weapon.Accuracy()),
			strconv.Itoa(weapon.Backstabs),
			strconv.Itoa(weapon.Headshots),
			strconv.Itoa(weapon.Airshots),
		})
	}

	table.Render()

	return writer.String()
}

func makeKillstreakStatsTable(killstreaks []PlayerKillstreakStats) string {
	writer := &strings.Builder{}
	table := defaultTable(writer)
	table.SetHeader([]string{"Ks", "Class", "Dur", "Date"})

	sort.SliceStable(killstreaks, func(i, j int) bool {
		return killstreaks[i].Kills > killstreaks[j].Kills
	})

	for index, killstreak := range killstreaks {
		if index == 3 {
			break
		}

		table.Append([]string{
			strconv.Itoa(killstreak.Kills),
			killstreak.Class.String(),
			strconv.Itoa(killstreak.Duration),
			killstreak.CreatedOn.Format(time.DateOnly),
		})
	}

	table.Render()

	return writer.String()
}

func makeMedicStatsTable(stats []match.PlayerMedicStats) string {
	writer := &strings.Builder{}
	table := message.DefaultTable(writer)
	table.SetHeader([]string{"Healing", "Drop", "NearFull", "AvgLen", "U", "K", "V", "Q"})

	sort.SliceStable(stats, func(i, j int) bool {
		return stats[i].Healing > stats[j].Healing
	})

	for index, medicStats := range stats {
		if index == 3 {
			break
		}

		table.Append([]string{
			strconv.Itoa(medicStats.Healing),
			strconv.Itoa(medicStats.Drops),
			strconv.Itoa(medicStats.NearFullChargeDeath),
			fmt.Sprintf("%.2f", medicStats.AvgUberLength),
			strconv.Itoa(medicStats.ChargesUber),
			strconv.Itoa(medicStats.ChargesKritz),
			strconv.Itoa(medicStats.ChargesVacc),
			strconv.Itoa(medicStats.ChargesQuickfix),
		})
	}

	table.Render()

	return writer.String()
}

func makeClassStatsTable(classes match.PlayerClassStatsCollection) string {
	writer := &strings.Builder{}
	table := DefaultTable(writer)
	table.SetHeader([]string{"Class", "K", "A", "D", "KD", "KAD", "DA", "DT", "Dom", "Time"})

	table.Append([]string{
		"total",
		strconv.FormatInt(int64(classes.Kills()), 10),
		strconv.Itoa(classes.Assists()),
		strconv.Itoa(classes.Deaths()),
		infString(classes.KDRatio()),
		infString(classes.KDARatio()),
		strconv.Itoa(classes.Damage()),
		// fmt.Sprintf("%d", classes.DamagePerMin()),
		strconv.Itoa(classes.DamageTaken()),
		// fmt.Sprintf("%d", classes.Captures()),
		strconv.Itoa(classes.Dominations()),
		// fmt.Sprintf("%d", classes.Dominated()),
		fmt.Sprintf("%.1fh", (time.Duration(int64(classes.Playtime())) * time.Second).Hours()),
	})

	sort.SliceStable(classes, func(i, j int) bool {
		return classes[i].Playtime > classes[j].Playtime
	})

	for _, player := range classes {
		table.Append([]string{
			player.ClassName,
			strconv.Itoa(player.Kills),
			strconv.Itoa(player.Assists),
			strconv.Itoa(player.Deaths),
			infString(player.KDRatio()),
			infString(player.KDARatio()),
			strconv.Itoa(player.Damage),
			// fmt.Sprintf("%d", player.DamagePerMin()),
			strconv.Itoa(player.DamageTaken),
			// fmt.Sprintf("%d", player.Captures),
			strconv.Itoa(player.Dominations),
			// fmt.Sprintf("%d", player.Dominated),
			fmt.Sprintf("%.1fh", (time.Duration(int64(player.Playtime)) * time.Second).Hours()),
		})
	}

	table.Render()

	return strings.Trim(writer.String(), "\n")
}

func StatsPlayerMessage(person domain.PersonInfo, url string, classStats match.PlayerClassStatsCollection,
	medicStats []match.PlayerMedicStats, weaponStats []match.PlayerWeaponStats, killstreakStats []match.PlayerKillstreakStats,
) *discordgo.MessageEmbed {
	emb := NewEmbed()
	emb.Embed().
		SetTitle("Overall Player Stats").
		SetColor(ColourSuccess)

	emb.Embed().SetDescription(fmt.Sprintf("Class Totals```%s```Healing```%s```Top Weapons```%s```Top Killstreaks```%s```",
		makeClassStatsTable(classStats),
		makeMedicStatsTable(medicStats),
		makeWeaponStatsTable(weaponStats),
		makeKillstreakStatsTable(killstreakStats),
	))

	return emb.AddAuthorPersonInfo(person, url).Embed().MessageEmbed
}

func MatchMessage(match match.MatchResult, link string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed(strings.Join([]string{match.Title, match.MapName}, " | "))
	msgEmbed.Embed().
		SetColor(ColourSuccess).
		SetURL(link)

	msgEmbed.Embed().SetDescription(matchASCIITable(match))

	msgEmbed.Embed().AddField("Red Score", strconv.Itoa(match.TeamScores.Red)).MakeFieldInline()
	msgEmbed.Embed().AddField("Blu Score", strconv.Itoa(match.TeamScores.Blu)).MakeFieldInline()
	msgEmbed.Embed().AddField("Map", match.MapName).MakeFieldInline()
	msgEmbed.Embed().AddField("Chat Messages", strconv.Itoa(len(match.Chat))).MakeFieldInline()

	msgCounts := map[steamid.SteamID]int{}

	for _, msg := range match.Chat {
		_, found := msgCounts[msg.SteamID]
		if !found {
			msgCounts[msg.SteamID] = 0
		}

		msgCounts[msg.SteamID]++
	}

	var (
		chatSid   steamid.SteamID
		count     int
		kathyName string
	)

	for sid, cnt := range msgCounts {
		if cnt > count {
			count = cnt
			chatSid = sid
		}
	}

	for _, player := range match.Players {
		if player.SteamID == chatSid {
			kathyName = player.Name

			break
		}
	}

	msgEmbed.Embed().AddField("Top Chatter", fmt.Sprintf("%s (count: %d)", kathyName, count)).MakeFieldInline()
	msgEmbed.Embed().AddField("Players", strconv.Itoa(len(match.Players))).MakeFieldInline()
	msgEmbed.Embed().AddField("Duration", match.TimeEnd.Sub(match.TimeStart).String()).MakeFieldInline()

	return msgEmbed.Embed().Truncate().MessageEmbed
}

const tableNameLen = 13

func matchASCIITable(matchResult match.MatchResult) string {
	writerPlayers := &strings.Builder{}
	tablePlayers := DefaultTable(writerPlayers)
	tablePlayers.SetHeader([]string{"T", "Name", "K", "A", "D", "KD", "KAD", "DA", "B/H/A/C"})

	players := matchResult.TopPlayers()

	for i, player := range players {
		if i == tableNameLen {
			break
		}

		name := player.SteamID.String()
		if player.Name != "" {
			name = player.Name
		}

		if len(name) > tableNameLen {
			name = name[0:tableNameLen]
		}

		tablePlayers.Append([]string{
			player.Team.String()[0:1],
			name,
			strconv.Itoa(player.Kills),
			strconv.Itoa(player.Assists),
			strconv.Itoa(player.Deaths),
			infString(player.KDRatio()),
			infString(player.KDARatio()),
			strconv.Itoa(player.Damage),
			fmt.Sprintf("%d/%d/%d/%d", player.Backstabs, player.Headshots, player.Airshots, player.Captures),
		})
	}

	tablePlayers.Render()

	writerHealers := &strings.Builder{}
	tableHealers := DefaultTable(writerPlayers)
	tableHealers.SetHeader([]string{" ", "Name", "A", "D", "Heal", "H/M", "Dr", "U/K/Q/V", "AUL"})

	for _, player := range matchResult.Healers() {
		if player.MedicStats.Healing < match.MinMedicHealing {
			continue
		}

		name := player.SteamID.String()
		if player.Name != "" {
			name = player.Name
		}

		if len(name) > tableNameLen {
			name = name[0:tableNameLen]
		}

		tableHealers.Append([]string{
			player.Team.String()[0:1],
			name,
			strconv.Itoa(player.Assists),
			strconv.Itoa(player.Deaths),
			strconv.Itoa(player.MedicStats.Healing),
			strconv.Itoa(player.MedicStats.HealingPerMin(player.TimeEnd.Sub(player.TimeStart))),
			strconv.Itoa(player.MedicStats.Drops),
			fmt.Sprintf("%d/%d/%d/%d", player.MedicStats.ChargesUber, player.MedicStats.ChargesKritz,
				player.MedicStats.ChargesQuickfix, player.MedicStats.ChargesVacc),

			fmt.Sprintf("%.1f", player.MedicStats.AvgUberLength),
		})
	}

	tableHealers.Render()

	var topKs string

	topKillstreaks := matchResult.TopKillstreaks(3)

	if len(topKillstreaks) > 0 {
		writerKillstreak := &strings.Builder{}
		tableKillstreaks := message.DefaultTable(writerKillstreak)
		tableKillstreaks.SetHeader([]string{" ", "Name", "Killstreak", "Class", "Duration"})

		for _, player := range topKillstreaks {
			killstreak := player.BiggestKillstreak()

			name := player.SteamID.String()
			if player.Name != "" {
				name = player.Name
			}

			if len(name) > 17 {
				name = name[0:17]
			}

			tableKillstreaks.Append([]string{
				player.Team.String()[0:1],
				name,
				strconv.Itoa(killstreak.Killstreak),
				killstreak.PlayerClass.String(),
				(time.Duration(killstreak.Duration) * time.Second).String(),
			})
		}

		tableKillstreaks.Render()

		topKs = writerKillstreak.String()
	}

	resp := fmt.Sprintf("```%s\n%s\n%s```",
		strings.Trim(writerPlayers.String(), "\n"),
		strings.Trim(writerHealers.String(), "\n"),
		strings.Trim(topKs, "\n"))

	return resp
}
