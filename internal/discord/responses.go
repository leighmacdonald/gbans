package discord

import (
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/olekukonko/tablewriter"
)

const (
	ColourDebug   = 10170623
	ColourSuccess = 302673
	ColourInfo    = 3581519
	ColourWarn    = 14327864
	ColourError   = 13631488
)

func defaultTable(writer io.Writer) *tablewriter.Table {
	tbl := tablewriter.NewWriter(writer)
	tbl.SetAutoFormatHeaders(true)
	tbl.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	tbl.SetCenterSeparator("")
	tbl.SetColumnSeparator("")
	tbl.SetRowSeparator("")
	tbl.SetHeaderLine(false)
	tbl.SetTablePadding("")
	tbl.SetAutoMergeCells(true)
	tbl.SetAlignment(tablewriter.ALIGN_LEFT)

	return tbl
}

func makeClassStatsTable(classes model.PlayerClassStatsCollection) string {
	writer := &strings.Builder{}
	table := defaultTable(writer)
	table.SetHeader([]string{"Class", "K", "A", "D", "KD", "KAD", "DA", "DT", "Dom", "Time"})

	table.Append([]string{
		"total",
		fmt.Sprintf("%d", classes.Kills()),
		fmt.Sprintf("%d", classes.Assists()),
		fmt.Sprintf("%d", classes.Deaths()),
		infString(classes.KDRatio()),
		infString(classes.KDARatio()),
		fmt.Sprintf("%d", classes.Damage()),
		// fmt.Sprintf("%d", classes.DamagePerMin()),
		fmt.Sprintf("%d", classes.DamageTaken()),
		// fmt.Sprintf("%d", classes.Captures()),
		fmt.Sprintf("%d", classes.Dominations()),
		// fmt.Sprintf("%d", classes.Dominated()),
		fmt.Sprintf("%.1fh", (time.Duration(int64(classes.Playtime())) * time.Second).Hours()),
	})

	sort.SliceStable(classes, func(i, j int) bool {
		return classes[i].Playtime > classes[j].Playtime
	})

	for _, player := range classes {
		table.Append([]string{
			player.ClassName,
			fmt.Sprintf("%d", player.Kills),
			fmt.Sprintf("%d", player.Assists),
			fmt.Sprintf("%d", player.Deaths),
			infString(player.KDRatio()),
			infString(player.KDARatio()),
			fmt.Sprintf("%d", player.Damage),
			// fmt.Sprintf("%d", player.DamagePerMin()),
			fmt.Sprintf("%d", player.DamageTaken),
			// fmt.Sprintf("%d", player.Captures),
			fmt.Sprintf("%d", player.Dominations),
			// fmt.Sprintf("%d", player.Dominated),
			fmt.Sprintf("%.1fh", (time.Duration(int64(player.Playtime)) * time.Second).Hours()),
		})
	}

	table.Render()

	return strings.Trim(writer.String(), "\n")
}

func makeWeaponStatsTable(weapons []model.PlayerWeaponStats) string {
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
			fmt.Sprintf("%d", weapon.Kills),
			fmt.Sprintf("%d", weapon.Damage),
			fmt.Sprintf("%d", weapon.Shots),
			fmt.Sprintf("%d", weapon.Hits),
			fmt.Sprintf("%.1f", weapon.Accuracy()),
			fmt.Sprintf("%d", weapon.Backstabs),
			fmt.Sprintf("%d", weapon.Headshots),
			fmt.Sprintf("%d", weapon.Airshots),
		})
	}

	table.Render()

	return writer.String()
}

func makeKillstreakStatsTable(killstreaks []model.PlayerKillstreakStats) string {
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
			fmt.Sprintf("%d", killstreak.Kills),
			killstreak.Class.String(),
			fmt.Sprintf("%d", killstreak.Duration),
			killstreak.CreatedOn.Format(time.DateOnly),
		})
	}

	table.Render()

	return writer.String()
}

func makeMedicStatsTable(stats []model.PlayerMedicStats) string {
	writer := &strings.Builder{}
	table := defaultTable(writer)
	table.SetHeader([]string{"Healing", "Drop", "NearFull", "AvgLen", "U", "K", "V", "Q"})

	sort.SliceStable(stats, func(i, j int) bool {
		return stats[i].Healing > stats[j].Healing
	})

	for index, medicStats := range stats {
		if index == 3 {
			break
		}

		table.Append([]string{
			fmt.Sprintf("%d", medicStats.Healing),
			fmt.Sprintf("%d", medicStats.Drops),
			fmt.Sprintf("%d", medicStats.NearFullChargeDeath),
			fmt.Sprintf("%.2f", medicStats.AvgUberLength),
			fmt.Sprintf("%d", medicStats.ChargesUber),
			fmt.Sprintf("%d", medicStats.ChargesKritz),
			fmt.Sprintf("%d", medicStats.ChargesVacc),
			fmt.Sprintf("%d", medicStats.ChargesQuickfix),
		})
	}

	table.Render()

	return writer.String()
}

func ErrorMessage(command string, err error) *discordgo.MessageEmbed {
	return NewEmbed("Error Returned").Embed().
		SetColor(ColourError).
		AddField("command", command).
		SetDescription(err.Error()).MessageEmbed
}

func KickPlayerEmbed(target model.PersonInfo) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("User Kicked Successfully")
	msgEmbed.Embed().SetColor(ColourSuccess)

	return msgEmbed.AddTargetPerson(target).Embed().MessageEmbed
}

func SilenceEmbed(target model.PersonInfo) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("User Silenced Successfully")
	msgEmbed.Embed().SetColor(ColourSuccess)

	return msgEmbed.AddTargetPerson(target).Embed().MessageEmbed
}

func BanExpiresMessage(ban model.BanSteam, person model.PersonInfo, banURL string) *discordgo.MessageEmbed {
	banType := "Ban"
	if ban.BanType == model.NoComm {
		banType = "Mute"
	}

	msgEmbed := NewEmbed("Steam Ban Expired")
	msgEmbed.
		Embed().
		SetColor(ColourInfo).
		AddField("Type", banType).
		SetImage(person.GetAvatar().Full()).
		AddField("Name", person.GetName()).
		SetURL(banURL)

	msgEmbed.AddFieldsSteamID(person.GetSteamID())

	if ban.BanType == model.NoComm {
		msgEmbed.Embed().SetColor(ColourWarn)
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func BanSteamResponse(banSteam model.BanSteam) *discordgo.MessageEmbed {
	var (
		title  string
		colour int
	)

	if banSteam.BanType == model.NoComm {
		title = fmt.Sprintf("User Muted (#%d)", banSteam.BanID)
		colour = ColourWarn
	} else {
		title = fmt.Sprintf("User Banned (#%d)", banSteam.BanID)
		colour = ColourError
	}

	msgEmbed := NewEmbed(title)
	msgEmbed.Embed().
		SetColor(colour).
		SetURL(fmt.Sprintf("https://steamcommunity.com/profiles/%s", banSteam.TargetID))

	msgEmbed.AddFieldsSteamID(banSteam.TargetID)

	expIn := "Permanent"
	expAt := "Permanent"

	if banSteam.ValidUntil.Year()-time.Now().Year() < 5 {
		expIn = util.FmtDuration(banSteam.ValidUntil)
		expAt = util.FmtTimeShort(banSteam.ValidUntil)
	}

	msgEmbed.
		Embed().
		AddField("Expires In", expIn).
		AddField("Expires At", expAt)

	return msgEmbed.Embed().MessageEmbed
}

func BanCIDRResponse(cidr *net.IPNet, author model.PersonInfo, authorURL string, target model.PersonInfo, banNet *model.BanCIDR) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("IP Banned Successfully")
	msgEmbed.Embed().
		SetColor(ColourSuccess).
		AddField("cidr", cidr.String()).
		AddField("net_id", fmt.Sprintf("%d", banNet.NetID)).
		AddField("Reason", banNet.Reason.String())

	msgEmbed.AddTargetPerson(target)
	msgEmbed.AddAuthorPersonInfo(author, authorURL)

	return msgEmbed.Embed().MessageEmbed
}

func DeleteReportMessage(existing model.ReportMessage, user model.PersonInfo, userURL string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("User report message deleted")
	msgEmbed.
		Embed().
		SetDescription(existing.MessageMD).
		SetColor(ColourWarn)

	return msgEmbed.AddAuthorPersonInfo(user, userURL).Embed().Truncate().MessageEmbed
}

func NewInGameReportResponse(report model.Report, reportURL string, author model.PersonInfo, authorURL string, demoURL string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("New User Report Created")
	msgEmbed.
		Embed().
		SetDescription(report.Description).
		SetColor(ColourSuccess).
		SetURL(reportURL)

	msgEmbed.AddAuthorPersonInfo(author, authorURL)

	name := author.GetName()

	if name == "" {
		name = report.TargetID.String()
	}

	msgEmbed.
		Embed().
		AddField("Subject", name).
		AddField("Reason", report.Reason.String())

	if report.ReasonText != "" {
		msgEmbed.Embed().AddField("Custom Reason", report.ReasonText)
	}

	if report.DemoName != "" {
		if demoURL != "" {
			msgEmbed.Embed().AddField("Demo", demoURL)
		}

		msgEmbed.Embed().AddField("Demo Tick", fmt.Sprintf("%d", report.DemoTick))
	}

	return msgEmbed.AddFieldsSteamID(report.TargetID).Embed().Truncate().MessageEmbed
}

func NewReportMessageResponse(message string, link string, author model.PersonInfo, authorURL string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("New report message posted")
	msgEmbed.
		Embed().
		SetDescription(message).
		SetColor(ColourSuccess).
		SetURL(link)

	return msgEmbed.AddAuthorPersonInfo(author, authorURL).Embed().Truncate().MessageEmbed
}

func EditReportMessageResponse(body string, oldBody string, link string, author model.PersonInfo, authorURL string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("New report message edited")
	msgEmbed.
		Embed().
		SetDescription(body).
		SetColor(ColourWarn).
		AddField("Old Message", oldBody).
		SetURL(link)

	return msgEmbed.AddAuthorPersonInfo(author, authorURL).Embed().Truncate().MessageEmbed
}

func NewAppealMessage(msg string, link string, author model.PersonInfo, authorURL string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("New ban appeal message posted")
	msgEmbed.
		Embed().
		SetColor(ColourInfo).
		// SetThumbnail(bannedPerson.TargetAvatarhash).
		SetDescription(msg).
		SetURL(link)

	return msgEmbed.AddAuthorPersonInfo(author, authorURL).Embed().Truncate().MessageEmbed
}

func EditAppealMessage(existing model.BanAppealMessage, body string, author model.PersonInfo, authorURL string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("Ban appeal message edited")
	msgEmbed.
		Embed().
		SetDescription(util.DiffString(existing.MessageMD, body)).
		SetColor(ColourWarn)

	return msgEmbed.
		AddAuthorPersonInfo(author, authorURL).
		Embed().
		Truncate().
		MessageEmbed
}

func DeleteAppealMessage(existing model.BanAppealMessage, user model.PersonInfo, userURL string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("User appeal message deleted")
	msgEmbed.
		Embed().
		SetDescription(existing.MessageMD)

	return msgEmbed.
		AddAuthorPersonInfo(user, userURL).
		Embed().
		Truncate().
		MessageEmbed
}

func NewNewsMessage(body string, title string) *discordgo.MessageEmbed {
	return NewEmbed("News Created").
		Embed().
		SetDescription(body).
		AddField("Title", title).MessageEmbed
}

func EditNewsMessages(title string, body string) *discordgo.MessageEmbed {
	return NewEmbed("News Updated").
		Embed().
		AddField("Title", title).
		SetDescription(body).
		MessageEmbed
}

func EditBanAppealStatusMessage() *discordgo.MessageEmbed {
	return NewEmbed("Ban state updates").Embed().MessageEmbed
}

func PingModMessage(author model.PersonInfo, authorURL string, reason string, serverName string, roleID string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("New User In-Game Report")
	msgEmbed.
		Embed().
		SetDescription(fmt.Sprintf("%s | <@&%s>", reason, roleID)).
		AddField("server", serverName)

	msgEmbed.AddAuthorPersonInfo(author, authorURL).Embed().Truncate()

	return msgEmbed.Embed().MessageEmbed
}

func ReportStatsMessage(meta model.ReportMeta, url string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("User Report Stats")
	msgEmbed.
		Embed().
		SetColor(ColourSuccess).
		SetURL(url)

	if meta.OpenWeek > 0 {
		msgEmbed.Embed().SetColor(ColourError)
	} else if meta.Open3Days > 0 {
		msgEmbed.Embed().SetColor(ColourWarn)
	}

	return msgEmbed.
		Embed().
		SetDescription("Current Open Report Counts").
		AddField("New", fmt.Sprintf(" %d", meta.Open1Day)).MakeFieldInline().
		AddField("Total Open", fmt.Sprintf(" %d", meta.TotalOpen)).MakeFieldInline().
		AddField("Total Closed", fmt.Sprintf(" %d", meta.TotalClosed)).MakeFieldInline().
		AddField(">1 Day", fmt.Sprintf(" %d", meta.Open1Day)).MakeFieldInline().
		AddField(">3 Days", fmt.Sprintf(" %d", meta.Open3Days)).MakeFieldInline().
		AddField(">1 Week", fmt.Sprintf(" %d", meta.OpenWeek)).MakeFieldInline().
		Truncate().
		MessageEmbed
}

func WarningMessage(newWarning model.NewUserWarning, banSteam model.BanSteam, person model.PersonInfo) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("Language Warning")
	msgEmbed.Embed().
		SetDescription(newWarning.UserWarning.Message).
		SetColor(ColourWarn).
		AddField("Filter ID", fmt.Sprintf("%d", newWarning.MatchedFilter.FilterID)).
		AddField("Matched", newWarning.Matched).
		AddField("ServerStore", newWarning.UserMessage.ServerName).InlineAllFields().
		AddField("Pattern", newWarning.MatchedFilter.Pattern)

	msgEmbed.
		AddFieldsSteamID(newWarning.UserMessage.SteamID).
		Embed().
		AddField("Name", person.GetName())

	var (
		expIn = "Permanent"
		expAt = expIn
	)

	if banSteam.ValidUntil.Year()-time.Now().Year() < 5 {
		expIn = util.FmtDuration(banSteam.ValidUntil)
		expAt = util.FmtTimeShort(banSteam.ValidUntil)
	}

	return msgEmbed.
		Embed().
		AddField("Expires In", expIn).
		AddField("Expires At", expAt).
		MessageEmbed
}

func CheckMessage(player model.Person, ban model.BannedSteamPerson, banURL string, author model.Person,
	oldBans []model.BannedSteamPerson, bannedNets []model.BanCIDR, asn ip2location.ASNRecord,
	location ip2location.LocationRecord,
	proxy ip2location.ProxyRecord, logData *thirdparty.LogsTFResult,
) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed()

	title := player.PersonaName

	if ban.BanID > 0 {
		if ban.BanType == model.Banned {
			title = fmt.Sprintf("%s (BANNED)", title)
		} else if ban.BanType == model.NoComm {
			title = fmt.Sprintf("%s (MUTED)", title)
		}
	}

	msgEmbed.Embed().SetTitle(title)

	if player.RealName != "" {
		msgEmbed.Embed().AddField("Real Name", player.RealName)
	}

	cd := time.Unix(int64(player.TimeCreated), 0)
	msgEmbed.Embed().AddField("Age", util.FmtDuration(cd))
	msgEmbed.Embed().AddField("Private", fmt.Sprintf("%v", player.CommunityVisibilityState == 1))
	msgEmbed.AddFieldsSteamID(player.SteamID)

	if player.VACBans > 0 {
		msgEmbed.Embed().AddField("VAC Bans", fmt.Sprintf("count: %d days: %d", player.VACBans, player.DaysSinceLastBan))
	}

	if player.GameBans > 0 {
		msgEmbed.Embed().AddField("Game Bans", fmt.Sprintf("count: %d", player.GameBans))
	}

	if player.CommunityBanned {
		msgEmbed.Embed().AddField("Com. Ban", "true")
	}

	if player.EconomyBan != "" {
		msgEmbed.Embed().AddField("Econ Ban", string(player.EconomyBan))
	}

	if len(oldBans) > 0 {
		numMutes, numBans := 0, 0

		for _, oldBan := range oldBans {
			if oldBan.BanType == model.Banned {
				numBans++
			} else {
				numMutes++
			}
		}

		msgEmbed.Embed().AddField("Total Mutes", fmt.Sprintf("%d", numMutes))
		msgEmbed.Embed().AddField("Total Bans", fmt.Sprintf("%d", numBans))
	}

	msgEmbed.Embed().InlineAllFields()

	var (
		banned    = false
		muted     = false
		reason    = ""
		color     = 0
		createdAt = ""
		expiry    time.Time
	)

	if ban.BanID > 0 {
		banned = ban.BanType == model.Banned
		muted = ban.BanType == model.NoComm
		reason = ban.ReasonText

		if len(reason) == 0 {
			// in case authorProfile ban without authorProfile reason ever makes its way here, we make sure
			// that Discord doesn't shit itself
			reason = "none"
		}

		expiry = ban.ValidUntil
		createdAt = ban.CreatedOn.Format(time.RFC3339)

		msgEmbed.Embed().SetURL(banURL)
		msgEmbed.Embed().AddField("Reason", reason)
		msgEmbed.Embed().AddField("Created", util.FmtTimeShort(ban.CreatedOn)).MakeFieldInline()

		if time.Until(expiry) > time.Hour*24*365*5 {
			msgEmbed.Embed().AddField("Expires", "Permanent").MakeFieldInline()
		} else {
			msgEmbed.Embed().AddField("Expires", util.FmtDuration(expiry)).MakeFieldInline()
		}

		msgEmbed.Embed().AddField("Author", fmt.Sprintf("<@%s>", author.DiscordID)).MakeFieldInline()

		if ban.Note != "" {
			msgEmbed.Embed().AddField("Mod Note", ban.Note).MakeFieldInline()
		}

		msgEmbed.AddAuthorPersonInfo(author, "")
	}

	if len(bannedNets) > 0 {
		// ip = bannedNets[0].CIDR.String()
		reason = fmt.Sprintf("Banned from %d networks", len(bannedNets))
		expiry = bannedNets[0].ValidUntil
		msgEmbed.Embed().AddField("Network Bans", fmt.Sprintf("%d", len(bannedNets)))
	}

	banStateStr := "no"

	if banned {
		// #992D22 red
		color = ColourError
		banStateStr = "banned"
	}

	if muted {
		// #E67E22 orange
		color = ColourWarn
		banStateStr = "muted"
	}

	msgEmbed.Embed().AddField("Ban/Muted", banStateStr)

	if player.IPAddr != nil {
		msgEmbed.Embed().AddField("Last IP", player.IPAddr.String()).MakeFieldInline()
	}

	if asn.ASName != "" {
		msgEmbed.Embed().AddField("ASN", fmt.Sprintf("(%d) %s", asn.ASNum, asn.ASName)).MakeFieldInline()
	}

	if location.CountryCode != "" {
		msgEmbed.Embed().AddField("City", location.CityName).MakeFieldInline()
	}

	if location.CountryName != "" {
		msgEmbed.Embed().AddField("Country", location.CountryName).MakeFieldInline()
	}

	if proxy.CountryCode != "" {
		msgEmbed.Embed().AddField("Proxy Type", string(proxy.ProxyType)).MakeFieldInline()
		msgEmbed.Embed().AddField("Proxy", string(proxy.Threat)).MakeFieldInline()
	}

	if logData != nil && logData.Total > 0 {
		msgEmbed.Embed().AddField("Logs.tf", fmt.Sprintf("%d", logData.Total)).MakeFieldInline()
	}

	if createdAt != "" {
		msgEmbed.Embed().AddField("created at", createdAt).MakeFieldInline()
	}

	return msgEmbed.Embed().
		SetURL(player.ProfileURL).
		SetColor(color).
		SetImage(player.AvatarFull).
		SetThumbnail(player.Avatar).
		Truncate().MessageEmbed
}

func HistoryMessage(person model.PersonInfo) *discordgo.MessageEmbed {
	return NewEmbed(fmt.Sprintf("IP History of: %s", person.GetName())).Embed().
		SetDescription("IP history (20 max)").
		Truncate().MessageEmbed
}

func BanMessage(ban model.BannedSteamPerson, link string, target model.PersonInfo, source model.PersonInfo) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed()
	msgEmbed.Embed().
		SetTitle(fmt.Sprintf("Ban created successfully (#%d)", ban.BanID)).
		SetDescription(ban.Note).
		SetURL(link).
		SetColor(ColourSuccess)

	if ban.ReasonText != "" {
		msgEmbed.Embed().AddField("Reason", ban.ReasonText)
	}

	msgEmbed.Embed().SetImage(target.GetAvatar().Full())

	msgEmbed.AddAuthorPersonInfo(source, "")

	if ban.ValidUntil.Year()-time.Now().Year() > 5 {
		msgEmbed.Embed().AddField("Expires In", "Permanent")
		msgEmbed.Embed().AddField("Expires At", "Permanent")
	} else {
		msgEmbed.Embed().AddField("Expires In", util.FmtDuration(ban.ValidUntil))
		msgEmbed.Embed().AddField("Expires At", util.FmtTimeShort(ban.ValidUntil))
	}

	msgEmbed.AddFieldsSteamID(ban.TargetID)

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func UnbanMessage(person model.PersonInfo) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("User Unbanned Successfully")
	msgEmbed.Embed().
		SetColor(ColourSuccess).
		SetImage(person.GetAvatar().Full()).
		SetURL(person.Path())
	msgEmbed.AddFieldsSteamID(person.GetSteamID())

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func UnbanASNMessage(asn int64, asnNetworks ip2location.ASNRecords) *discordgo.MessageEmbed {
	return NewEmbed("ASN Networks Unbanned Successfully").
		Embed().
		SetColor(ColourSuccess).
		AddField("ASN", fmt.Sprintf("%d", asn)).
		AddField("Hosts", fmt.Sprintf("%d", asnNetworks.Hosts())).
		Truncate().MessageEmbed
}

func KickMessage(players []model.PlayerServerInfo) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("Users Kicked")
	for _, player := range players {
		msgEmbed.Embed().AddField("Name", player.Player.Name)
		msgEmbed.AddFieldsSteamID(player.Player.SID)
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func SayMessage(server string, msg string) *discordgo.MessageEmbed {
	return NewEmbed("Sent chat message successfully").Embed().
		SetColor(ColourSuccess).
		AddField("ServerStore", server).
		AddField("Message", msg).
		Truncate().MessageEmbed
}

func CSayMessage(server string, msg string) *discordgo.MessageEmbed {
	return NewEmbed("Sent console message successfully").Embed().
		SetColor(ColourSuccess).
		AddField("ServerStore", server).
		AddField("Message", msg).
		Truncate().MessageEmbed
}

func PSayMessage(player string, msg string) *discordgo.MessageEmbed {
	return NewEmbed("Sent private message successfully").Embed().
		SetColor(ColourSuccess).
		AddField("Player", player).
		AddField("Message", msg).
		Truncate().MessageEmbed
}

// TODO dont hardcode.
func mapRegion(region string) string {
	switch region {
	case "asia":
		return "Asia Pacific"
	case "au":
		return "Australia"
	case "eu":
		return "Europe (+UK)"
	case "sa":
		return "South America"
	case "us-east":
		return "NA East"
	case "us-west":
		return "NA West"
	case "us-central":
		return "NA Central"
	case "global":
		return "Global"
	default:
		return "Unknown"
	}
}

func ServersMessage(currentStateRegion map[string][]state.ServerState, serversURL string) *discordgo.MessageEmbed {
	var (
		stats       = map[string]float64{}
		used, total = 0, 0
		regionNames []string //nolint:realloc
	)

	msgEmbed := NewEmbed("Current ServerStore Populations")
	msgEmbed.Embed().SetURL(serversURL)

	for k := range currentStateRegion {
		regionNames = append(regionNames, k)
	}

	sort.Strings(regionNames)

	for _, region := range regionNames {
		var counts []string

		for _, curState := range currentStateRegion[region] {
			_, ok := stats[region]
			if !ok {
				stats[region] = 0
				stats[region+"total"] = 0
			}

			maxPlayers := curState.MaxPlayers - curState.Reserved
			if maxPlayers <= 0 {
				maxPlayers = 32 - curState.Reserved
			}

			stats[region] += float64(curState.PlayerCount)
			stats[region+"total"] += float64(maxPlayers)
			used += curState.PlayerCount
			total += maxPlayers
			counts = append(counts, fmt.Sprintf("%s:   %2d/%2d  ", curState.NameShort, curState.PlayerCount, maxPlayers))
		}

		msg := strings.Join(counts, "    ")
		if msg != "" {
			msgEmbed.Embed().AddField(mapRegion(region), fmt.Sprintf("```%s```", msg))
		}
	}

	for statName := range stats {
		if strings.HasSuffix(statName, "total") {
			continue
		}

		msgEmbed.Embed().AddField(mapRegion(statName), fmt.Sprintf("%.2f%%", (stats[statName]/stats[statName+"total"])*100)).MakeFieldInline()
	}

	msgEmbed.Embed().AddField("Global", fmt.Sprintf("%d/%d %.2f%%", used, total, float64(used)/float64(total)*100)).MakeFieldInline()

	if total == 0 {
		msgEmbed.Embed().SetColor(ColourError)
		msgEmbed.Embed().SetDescription("No server states available")
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func PlayersMessage(rows []string, maxPlayers int, serverName string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed(fmt.Sprintf("%s Current Players: %d / %d", serverName, len(rows), maxPlayers))
	if len(rows) > 0 {
		msgEmbed.Embed().SetDescription(strings.Join(rows, "\n"))
		msgEmbed.Embed().SetColor(ColourSuccess)
	} else {
		msgEmbed.Embed().SetDescription("No players :(")
		msgEmbed.Embed().SetColor(ColourError)
	}

	return msgEmbed.Embed().MessageEmbed
}

func FilterAddMessage(filter model.Filter) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("Filter Created Successfully").Embed().
		SetColor(ColourSuccess).
		AddField("pattern", filter.Pattern).
		Truncate()

	return msgEmbed.MessageEmbed
}

func FilterDelMessage(filter model.Filter) *discordgo.MessageEmbed {
	return NewEmbed("Filter Deleted Successfully").
		Embed().
		SetColor(ColourSuccess).
		AddField("filter", filter.Pattern).
		Truncate().MessageEmbed
}

func FilterCheckMessage(matches []model.Filter) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed()
	if len(matches) == 0 {
		msgEmbed.Embed().SetTitle("No Matches Found")
		msgEmbed.Embed().SetColor(ColourSuccess)
	} else {
		msgEmbed.Embed().SetTitle("Matched Found")
		msgEmbed.Embed().SetColor(ColourWarn)
		for _, match := range matches {
			msgEmbed.Embed().AddField(fmt.Sprintf("Matched ID: %d", match.FilterID), match.Pattern)
		}
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func StatsPlayerMessage(person model.PersonInfo, url string, classStats model.PlayerClassStatsCollection,
	medicStats []model.PlayerMedicStats, weaponStats []model.PlayerWeaponStats, killstreakStats []model.PlayerKillstreakStats,
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

func LogsMessage(count int64, matches string) *discordgo.MessageEmbed {
	return NewEmbed(fmt.Sprintf("Your most recent matches [%d total]", count)).Embed().
		SetColor(ColourSuccess).
		SetDescription(matches).MessageEmbed
}

type FoundPlayer struct {
	Player model.PlayerServerInfo
	Server model.Server
}

func FindMessage(found []FoundPlayer) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("Player(s) Found")
	for _, info := range found {
		msgEmbed.Embed().
			AddField("Name", info.Player.Player.Name).
			AddField("ServerStore", info.Server.ShortName).MakeFieldInline().
			AddField("steam", fmt.Sprintf("https://steamcommunity.com/profiles/%d", info.Player.Player.SID.Int64())).
			AddField("connect", fmt.Sprintf("connect %s", info.Server.Addr()))
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func MatchMessage(match model.MatchResult, link string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed(strings.Join([]string{match.Title, match.MapName}, " | "))
	msgEmbed.Embed().
		SetColor(ColourSuccess).
		SetURL(link)

	msgEmbed.Embed().SetDescription(matchASCIITable(match))

	msgEmbed.Embed().AddField("Red Score", fmt.Sprintf("%d", match.TeamScores.Red)).MakeFieldInline()
	msgEmbed.Embed().AddField("Blu Score", fmt.Sprintf("%d", match.TeamScores.Blu)).MakeFieldInline()
	msgEmbed.Embed().AddField("Map", match.MapName).MakeFieldInline()
	msgEmbed.Embed().AddField("Chat Messages", fmt.Sprintf("%d", len(match.Chat))).MakeFieldInline()

	msgCounts := map[steamid.SID64]int{}

	for _, msg := range match.Chat {
		_, found := msgCounts[msg.SteamID]
		if !found {
			msgCounts[msg.SteamID] = 0
		}
		msgCounts[msg.SteamID]++
	}

	var (
		chatSid   steamid.SID64
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
	msgEmbed.Embed().AddField("Players", fmt.Sprintf("%d", len(match.Players))).MakeFieldInline()
	msgEmbed.Embed().AddField("Duration", match.TimeEnd.Sub(match.TimeStart).String()).MakeFieldInline()

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func infString(f float64) string {
	if f == -1 {
		return "∞"
	}

	return fmt.Sprintf("%.1f", f)
}

const tableNameLen = 13

func matchASCIITable(match model.MatchResult) string {
	writerPlayers := &strings.Builder{}
	tablePlayers := defaultTable(writerPlayers)
	tablePlayers.SetHeader([]string{"T", "Name", "K", "A", "D", "KD", "KAD", "DA", "B/H/A/C"})

	players := match.TopPlayers()

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
			fmt.Sprintf("%d", player.Kills),
			fmt.Sprintf("%d", player.Assists),
			fmt.Sprintf("%d", player.Deaths),
			infString(player.KDRatio()),
			infString(player.KDARatio()),
			fmt.Sprintf("%d", player.Damage),
			fmt.Sprintf("%d/%d/%d/%d", player.Backstabs, player.Headshots, player.Airshots, player.Captures),
		})
	}

	tablePlayers.Render()

	writerHealers := &strings.Builder{}
	tableHealers := defaultTable(writerPlayers)
	tableHealers.SetHeader([]string{" ", "Name", "A", "D", "Heal", "H/M", "Dr", "U/K/Q/V", "AUL"})

	for _, player := range match.Healers() {
		if player.MedicStats.Healing < store.MinMedicHealing {
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
			fmt.Sprintf("%d", player.Assists),
			fmt.Sprintf("%d", player.Deaths),
			fmt.Sprintf("%d", player.MedicStats.Healing),
			fmt.Sprintf("%d", player.MedicStats.HealingPerMin(player.TimeEnd.Sub(player.TimeStart))),
			fmt.Sprintf("%d", player.MedicStats.Drops),
			fmt.Sprintf("%d/%d/%d/%d", player.MedicStats.ChargesUber, player.MedicStats.ChargesKritz,
				player.MedicStats.ChargesQuickfix, player.MedicStats.ChargesVacc),

			fmt.Sprintf("%.1f", player.MedicStats.AvgUberLength),
		})
	}

	tableHealers.Render()

	var topKs string

	topKillstreaks := match.TopKillstreaks(3)

	if len(topKillstreaks) > 0 {
		writerKillstreak := &strings.Builder{}
		tableKillstreaks := defaultTable(writerKillstreak)
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
				fmt.Sprintf("%d", killstreak.Killstreak),
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

func MuteMessage(banSteam model.BanSteam) *discordgo.MessageEmbed {
	return NewEmbed("Player muted successfully").
		AddFieldsSteamID(banSteam.TargetID).
		Embed().Truncate().MessageEmbed
}

func BanASNMessage(asNum int64, asnRecords ip2location.ASNRecords) *discordgo.MessageEmbed {
	return NewEmbed("ASN BanSteam Created Successfully").Embed().
		SetColor(ColourSuccess).
		AddField("ASNum", fmt.Sprintf("%d", asNum)).
		AddField("Total IPs Blocked", fmt.Sprintf("%d", asnRecords.Hosts())).
		Truncate().
		MessageEmbed
}

func BanIPMessage() *discordgo.MessageEmbed {
	return NewEmbed("IP ban created successfully").Embed().
		SetColor(ColourSuccess).
		Truncate().
		MessageEmbed
}

func SetSteamMessage() *discordgo.MessageEmbed {
	return NewEmbed().Embed().
		SetTitle("Steam Account Linked").
		SetDescription("Your steam and discord accounts are now linked").
		SetColor(ColourSuccess).
		Truncate().
		MessageEmbed
}
