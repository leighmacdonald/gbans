package discord

import (
	"fmt"
	"io"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/olekukonko/tablewriter"
)

const (
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

func makeClassStatsTable(classes domain.PlayerClassStatsCollection) string {
	writer := &strings.Builder{}
	table := defaultTable(writer)
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

func makeWeaponStatsTable(weapons []domain.PlayerWeaponStats) string {
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

func makeKillstreakStatsTable(killstreaks []domain.PlayerKillstreakStats) string {
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

func makeMedicStatsTable(stats []domain.PlayerMedicStats) string {
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

func ErrorMessage(command string, err error) *discordgo.MessageEmbed {
	return NewEmbed("Error Returned").Embed().
		SetColor(ColourError).
		AddField("command", command).
		SetDescription(err.Error()).MessageEmbed
}

func KickPlayerEmbed(target domain.PersonInfo) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("User Kicked Successfully")
	msgEmbed.Embed().SetColor(ColourSuccess)

	return msgEmbed.AddTargetPerson(target).Embed().MessageEmbed
}

func SilenceEmbed(target domain.PersonInfo) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("User Silenced Successfully")
	msgEmbed.Embed().SetColor(ColourSuccess)

	return msgEmbed.AddTargetPerson(target).Embed().MessageEmbed
}

func BanExpiresMessage(ban domain.BanSteam, person domain.PersonInfo, banURL string) *discordgo.MessageEmbed {
	banType := "Ban"
	if ban.BanType == domain.NoComm {
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

	if ban.BanType == domain.NoComm {
		msgEmbed.Embed().SetColor(ColourWarn)
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func BanSteamResponse(banSteam domain.BanSteam) *discordgo.MessageEmbed {
	var (
		title  string
		colour int
	)

	if banSteam.BanType == domain.NoComm {
		title = fmt.Sprintf("User Muted (#%d)", banSteam.BanID)
		colour = ColourWarn
	} else {
		title = fmt.Sprintf("User Banned (#%d)", banSteam.BanID)
		colour = ColourError
	}

	msgEmbed := NewEmbed(title)
	msgEmbed.Embed().
		SetColor(colour).
		SetURL("https://steamcommunity.com/profiles/" + banSteam.TargetID.String())

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

func BanCIDRResponse(cidr *net.IPNet, author domain.PersonInfo, authorURL string, target domain.PersonInfo, banNet *domain.BanCIDR) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("IP Banned Successfully")
	msgEmbed.Embed().
		SetColor(ColourSuccess).
		AddField("cidr", cidr.String()).
		AddField("net_id", strconv.FormatInt(banNet.NetID, 10)).
		AddField("Reason", banNet.Reason.String())

	msgEmbed.AddTargetPerson(target)
	msgEmbed.AddAuthorPersonInfo(author, authorURL)

	return msgEmbed.Embed().MessageEmbed
}

func DeleteReportMessage(existing domain.ReportMessage, user domain.PersonInfo, userURL string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("User report message deleted")
	msgEmbed.
		Embed().
		SetDescription(existing.MessageMD).
		SetColor(ColourWarn)

	return msgEmbed.AddAuthorPersonInfo(user, userURL).Embed().Truncate().MessageEmbed
}

func NewInGameReportResponse(report domain.Report, reportURL string, author domain.PersonInfo, authorURL string, _ string) *discordgo.MessageEmbed {
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

	return msgEmbed.AddFieldsSteamID(report.TargetID).Embed().Truncate().MessageEmbed
}

func NewReportMessageResponse(message string, link string, author domain.PersonInfo, authorURL string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("New report message posted")
	msgEmbed.
		Embed().
		SetDescription(message).
		SetColor(ColourSuccess).
		SetURL(link)

	return msgEmbed.AddAuthorPersonInfo(author, authorURL).Embed().Truncate().MessageEmbed
}

func EditReportMessageResponse(body string, oldBody string, link string, author domain.PersonInfo, authorURL string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("New report message edited")
	msgEmbed.
		Embed().
		SetDescription(body).
		SetColor(ColourWarn).
		AddField("Old Message", oldBody).
		SetURL(link)

	return msgEmbed.AddAuthorPersonInfo(author, authorURL).Embed().Truncate().MessageEmbed
}

func NewAppealMessage(msg string, link string, author domain.PersonInfo, authorURL string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("New ban appeal message posted")
	msgEmbed.
		Embed().
		SetColor(ColourInfo).
		// SetThumbnail(bannedPerson.TargetAvatarhash).
		SetDescription(msg).
		SetURL(link)

	return msgEmbed.AddAuthorPersonInfo(author, authorURL).Embed().Truncate().MessageEmbed
}

func EditAppealMessage(existing domain.BanAppealMessage, body string, author domain.PersonInfo, authorURL string) *discordgo.MessageEmbed {
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

func DeleteAppealMessage(existing *domain.BanAppealMessage, user domain.PersonInfo, userURL string) *discordgo.MessageEmbed {
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

func PingModMessage(author domain.PersonInfo, authorURL string, reason string, server domain.Server, roleID string, connect string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("New User In-Game Report")
	msgEmbed.
		Embed().
		SetDescription(fmt.Sprintf("%s | <@&%s>", reason, roleID)).
		AddField("server", server.Name)

	if connect != "" {
		msgEmbed.Embed().AddField("connect", connect)
	}

	msgEmbed.AddAuthorPersonInfo(author, authorURL).Embed().Truncate()

	return msgEmbed.Embed().MessageEmbed
}

func ReportStatsMessage(meta domain.ReportMeta, url string) *discordgo.MessageEmbed {
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

func WarningMessage(newWarning domain.NewUserWarning, banSteam domain.BanSteam, person domain.PersonInfo) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("Language Warning")
	msgEmbed.Embed().
		SetDescription(newWarning.UserWarning.Message).
		SetColor(ColourWarn).
		AddField("Filter ID", strconv.FormatInt(newWarning.MatchedFilter.FilterID, 10)).
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

func CheckMessage(player domain.Person, ban domain.BannedSteamPerson, banURL string, author domain.Person,
	oldBans []domain.BannedSteamPerson, bannedNets []domain.BanCIDR, asn domain.NetworkASN,
	location domain.NetworkLocation,
	proxy domain.NetworkProxy, logData *thirdparty.LogsTFResult,
) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed()

	title := player.PersonaName

	if ban.BanID > 0 {
		if ban.BanType == domain.Banned {
			title += " (BANNED)"
		} else if ban.BanType == domain.NoComm {
			title += " (MUTED)"
		}
	}

	msgEmbed.Embed().SetTitle(title)

	if player.RealName != "" {
		msgEmbed.Embed().AddField("Real Name", player.RealName)
	}

	cd := time.Unix(int64(player.TimeCreated), 0)
	msgEmbed.Embed().AddField("Age", util.FmtDuration(cd))
	msgEmbed.Embed().AddField("Private", strconv.FormatBool(player.CommunityVisibilityState == 1))
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
			if oldBan.BanType == domain.Banned {
				numBans++
			} else {
				numMutes++
			}
		}

		msgEmbed.Embed().AddField("Total Mutes", strconv.Itoa(numMutes))
		msgEmbed.Embed().AddField("Total Bans", strconv.Itoa(numBans))
	}

	msgEmbed.Embed().InlineAllFields()

	var (
		banned    = false
		muted     = false
		color     = 0
		createdAt = ""
		expiry    time.Time
	)

	if ban.BanID > 0 {
		banned = ban.BanType == domain.Banned
		muted = ban.BanType == domain.NoComm
		reason := ban.ReasonText

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
		netReason := fmt.Sprintf("Banned from %d networks", len(bannedNets))
		netExpiry := bannedNets[0].ValidUntil
		msgEmbed.Embed().AddField("Network Bans", strconv.Itoa(len(bannedNets)))
		msgEmbed.Embed().AddField("Network Reason", netReason)
		msgEmbed.Embed().AddField("Network Expires", util.FmtDuration(netExpiry)).MakeFieldInline()
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

	if player.IPAddr.IsValid() {
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
		msgEmbed.Embed().AddField("Logs.tf", strconv.Itoa(logData.Total)).MakeFieldInline()
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

func HistoryMessage(person domain.PersonInfo) *discordgo.MessageEmbed {
	return NewEmbed("IP History of: " + person.GetName()).Embed().
		SetDescription("IP history (20 max)").
		Truncate().MessageEmbed
}

// func BanMessage(ban model.BannedSteamPerson, link string, target model.PersonInfo, source model.PersonInfo) *discordgo.MessageEmbed {
//	msgEmbed := NewEmbed()
//	msgEmbed.Embed().
//		SetTitle(fmt.Sprintf("Ban created successfully (#%d)", ban.BanID)).
//		SetDescription(ban.Note).
//		SetURL(link).
//		SetColor(ColourSuccess)
//
//	if ban.ReasonText != "" {
//		msgEmbed.Embed().AddField("Reason", ban.ReasonText)
//	}
//
//	msgEmbed.Embed().SetImage(target.GetAvatar().Full())
//
//	msgEmbed.AddAuthorPersonInfo(source, "")
//
//	if ban.ValidUntil.Year()-time.Now().Year() > 5 {
//		msgEmbed.Embed().AddField("Expires In", "Permanent")
//		msgEmbed.Embed().AddField("Expires At", "Permanent")
//	} else {
//		msgEmbed.Embed().AddField("Expires In", util.FmtDuration(ban.ValidUntil))
//		msgEmbed.Embed().AddField("Expires At", util.FmtTimeShort(ban.ValidUntil))
//	}
//
//	msgEmbed.AddFieldsSteamID(ban.TargetID)
//
//	return msgEmbed.Embed().Truncate().MessageEmbed
// }

func UnbanMessage(cu domain.ConfigUsecase, person domain.PersonInfo) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("User Unbanned Successfully")
	msgEmbed.Embed().
		SetColor(ColourSuccess).
		SetImage(person.GetAvatar().Full()).
		SetURL(cu.ExtURL(person))
	msgEmbed.AddFieldsSteamID(person.GetSteamID())

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func UnbanASNMessage(asn int64) *discordgo.MessageEmbed {
	return NewEmbed("ASN Networks Unbanned Successfully").
		Embed().
		SetColor(ColourSuccess).
		AddField("ASN", strconv.FormatInt(asn, 10)).
		Truncate().MessageEmbed
}

func KickMessage(players []domain.PlayerServerInfo) *discordgo.MessageEmbed {
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

func PSayMessage(player steamid.SteamID, msg string) *discordgo.MessageEmbed {
	return NewEmbed("Sent private message successfully").Embed().
		SetColor(ColourSuccess).
		AddField("Player", string(player.Steam(false))).
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

func ServersMessage(currentStateRegion map[string][]domain.ServerState, serversURL string) *discordgo.MessageEmbed {
	var (
		stats       = map[string]float64{}
		used, total = 0, 0
		regionNames = make([]string, 9)
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

func FilterAddMessage(filter domain.Filter) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("Filter Created Successfully").Embed().
		SetColor(ColourSuccess).
		AddField("pattern", filter.Pattern).
		Truncate()

	return msgEmbed.MessageEmbed
}

func FilterDelMessage(filter domain.Filter) *discordgo.MessageEmbed {
	return NewEmbed("Filter Deleted Successfully").
		Embed().
		SetColor(ColourSuccess).
		AddField("filter", filter.Pattern).
		Truncate().MessageEmbed
}

func FilterCheckMessage(matches []domain.Filter) *discordgo.MessageEmbed {
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

func StatsPlayerMessage(person domain.PersonInfo, url string, classStats domain.PlayerClassStatsCollection,
	medicStats []domain.PlayerMedicStats, weaponStats []domain.PlayerWeaponStats, killstreakStats []domain.PlayerKillstreakStats,
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

func FindMessage(found []domain.FoundPlayer) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("Player(s) Found")
	for _, info := range found {
		msgEmbed.Embed().
			AddField("Name", info.Player.Player.Name).
			AddField("ServerStore", info.Server.ShortName).MakeFieldInline().
			AddField("steam", fmt.Sprintf("https://steamcommunity.com/profiles/%d", info.Player.Player.SID.Int64())).
			AddField("connect", "connect "+info.Server.Addr())
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func MatchMessage(match domain.MatchResult, link string) *discordgo.MessageEmbed {
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

func infString(f float64) string {
	if f == -1 {
		return "∞"
	}

	return fmt.Sprintf("%.1f", f)
}

const tableNameLen = 13

func matchASCIITable(match domain.MatchResult) string {
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
	tableHealers := defaultTable(writerPlayers)
	tableHealers.SetHeader([]string{" ", "Name", "A", "D", "Heal", "H/M", "Dr", "U/K/Q/V", "AUL"})

	for _, player := range match.Healers() {
		if player.MedicStats.Healing < domain.MinMedicHealing {
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

func MuteMessage(banSteam domain.BanSteam) *discordgo.MessageEmbed {
	return NewEmbed("Player muted successfully").
		AddFieldsSteamID(banSteam.TargetID).
		Embed().Truncate().MessageEmbed
}

func BanASNMessage(asNum int64) *discordgo.MessageEmbed {
	return NewEmbed("ASN BanSteam Created Successfully").Embed().
		SetColor(ColourSuccess).
		AddField("ASNum", strconv.FormatInt(asNum, 10)).
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

func NotificationMessage(message string, link string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("Notification", message)
	if link != "" {
		msgEmbed.Embed().SetURL(link)
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func VoteResultMessage(result domain.VoteResult) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("Vote Result")
	msgEmbed.Embed().
		AddField("Initiator", result.SourceID.String()).
		AddField("Target", result.TargetID.String()).
		AddField("Code", fmt.Sprintf("%d", result.Code)).
		AddField("Success", strconv.FormatBool(result.Success)).
		AddField("Server", strconv.FormatInt(int64(result.ServerID), 10))

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func ForumCategorySave(category domain.ForumCategory) *discordgo.MessageEmbed {
	embed := NewEmbed("Forum Category Saved")
	embed.Embed().AddField("Category", category.Title)
	embed.Embed().AddField("ID", strconv.Itoa(category.ForumCategoryID))

	if category.Description != "" {
		embed.Embed().AddField("Description", category.Description)
	}

	return embed.Embed().MessageEmbed
}

func ForumCategoryDelete(category domain.ForumCategory) *discordgo.MessageEmbed {
	embed := NewEmbed("Forum Category Deleted")
	embed.Embed().AddField("Category", category.Title)
	embed.Embed().AddField("ID", strconv.Itoa(category.ForumCategoryID))

	if category.Description != "" {
		embed.Embed().AddField("Description", category.Description)
	}

	return embed.Embed().MessageEmbed
}

func ForumMessageSaved(message domain.ForumMessage) *discordgo.MessageEmbed {
	embed := NewEmbed("Forum Message Created/Edited", message.BodyMD)
	embed.Embed().
		AddField("Category", message.Title)

	embed.Embed().Author.Name = message.Personaname
	embed.Embed().Author.IconURL = domain.NewAvatarLinks(message.Avatarhash).Medium()

	return embed.Embed().MessageEmbed
}

func ForumSaved(message domain.Forum) *discordgo.MessageEmbed {
	embed := NewEmbed("Forum Created/Edited")
	embed.Embed().
		AddField("Forum", message.Title)

	if message.Description != "" {
		embed.Embed().AddField("Description", message.Description)
	}

	return embed.Embed().MessageEmbed
}
