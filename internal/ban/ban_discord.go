package ban

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/discordgo-lipstick/bot"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/datetime"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/sosodev/duration"
)

type discordHandler struct {
	bans    Bans
	persons person.Provider
	discord person.DiscordPersonProvider
	config  *config.Configuration
}

func RegisterDiscordCommands(bot *bot.Bot, bans Bans) {
	var (
		reasons = []*discordgo.ApplicationCommandOptionChoice{
			{Name: External.String(), Value: External},
			{Name: Cheating.String(), Value: Cheating},
			{Name: Racism.String(), Value: Racism},
			{Name: Harassment.String(), Value: Harassment},
			{Name: Exploiting.String(), Value: Exploiting},
			{Name: WarningsExceeded.String(), Value: WarningsExceeded},
			{Name: Spam.String(), Value: Spam},
			{Name: Language.String(), Value: Language},
			{Name: Profile.String(), Value: Profile},
			{Name: ItemDescriptions.String(), Value: ItemDescriptions},
			{Name: BotHost.String(), Value: BotHost},
			{Name: Evading.String(), Value: Evading},
			{Name: Username.String(), Value: Username},
			{Name: Custom.String(), Value: Custom},
		}

		optBanReason = &discordgo.ApplicationCommandOption{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        discord.OptBanReason,
			Description: "Reason for the ban/mute",
			Required:    true,
			Choices:     reasons,
		}
	)

	handler := &discordHandler{bans: bans}

	bot.MustRegisterHandler("ban", &discordgo.ApplicationCommand{
		Name:                     "ban",
		Description:              "Manage steam, ip, group and ASN bans",
		DMPermission:             &discord.DmPerms,
		DefaultMemberPermissions: &discord.ModPerms,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "steam",
				Description: "Ban and kick a user from all servers",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        discord.OptUserIdentifier,
						Description: "SteamID in any format OR profile url",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        discord.OptDuration,
						Description: "Duration [s,m,h,d,w,M,y]N|0",
						Required:    true,
					},
					optBanReason,
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        discord.OptNote,
						Description: "Mod only notes for the ban reason",
						Required:    true,
					},
				},
			},
		},
	}, handler.onBan)

	bot.MustRegisterHandler("unban", &discordgo.ApplicationCommand{
		Name:                     "unban",
		Description:              "Manage steam, ip and ASN bans",
		DMPermission:             &discord.DmPerms,
		DefaultMemberPermissions: &discord.ModPerms,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "steam",
				Description: "Unban a previously banned player",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        discord.OptUserIdentifier,
						Description: "SteamID in any format OR profile url",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        discord.OptUnbanReason,
						Description: "Reason for unbanning",
						Required:    true,
					},
				},
			},
		},
	}, handler.onUnban)

	bot.MustRegisterHandler("mute", &discordgo.ApplicationCommand{
		Name:                     "mute",
		Description:              "Mute a player",
		DMPermission:             &discord.DmPerms,
		DefaultMemberPermissions: &discord.ModPerms,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        discord.OptUserIdentifier,
				Description: "SteamID in any format OR profile url",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        discord.OptDuration,
				Description: "Duration [s,m,h,d,w,M,y]N|0",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        discord.OptBanReason,
				Description: "Reason for the ban/mute",
				Required:    true,
				Choices:     reasons,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        discord.OptNote,
				Description: "Mod only notes for the mute reason",
				Required:    true,
			},
		},
	}, handler.onMute)

	bot.MustRegisterHandler("check", &discordgo.ApplicationCommand{
		Name:                     "check",
		DMPermission:             &discord.DmPerms,
		DefaultMemberPermissions: &discord.ModPerms,
		Description:              "Get ban status for a steam id",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        discord.OptUserIdentifier,
				Description: "SteamID in any format OR profile url",
				Required:    true,
			},
		},
	}, handler.onCheck)
}

func (h discordHandler) onMute(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := bot.OptionMap(interaction.ApplicationCommandData().Options)

	playerID, errPlayerID := steamid.Resolve(ctx, opts.String(discord.OptUserIdentifier))
	if errPlayerID != nil || !playerID.Valid() {
		return nil, steamid.ErrInvalidSID
	}

	reasonValueOpt, ok := opts[discord.OptBanReason]
	if !ok {
		return nil, ErrReasonInvalid
	}

	author, errAuthor := h.discord.GetPersonByDiscordID(ctx, interaction.Member.User.ID)
	if errAuthor != nil {
		return nil, errAuthor
	}
	banOpts := Opts{
		Origin:     Bot,
		SourceID:   author.SteamID,
		TargetID:   steamid.New(opts.String(discord.OptUserIdentifier)),
		BanType:    NoComm,
		Reason:     Reason(reasonValueOpt.IntValue()),
		ReasonText: "",
		Note:       opts[discord.OptNote].StringValue(),
	}

	duration, errDuration := duration.Parse(opts[discord.OptDuration].StringValue())
	if errDuration != nil {
		return nil, errors.Join(errDuration, ErrInvalidBanDuration)
	}

	banOpts.Duration = duration

	banSteam, errBan := h.bans.Create(ctx, banOpts)
	if errBan != nil {
		return nil, errBan
	}

	return MuteMessage(banSteam.TargetID), nil
}

// onBanSteam !ban <id> <duration> [reason].
func (h discordHandler) onBan(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := bot.OptionMap(interaction.ApplicationCommandData().Options[0].Options)

	author, errAuthor := h.discord.GetPersonByDiscordID(ctx, interaction.Member.User.ID)
	if errAuthor != nil {
		return nil, errAuthor
	}

	banOpts := Opts{
		Origin:     Bot,
		SourceID:   author.SteamID,
		TargetID:   steamid.New(opts[discord.OptUserIdentifier].StringValue()),
		BanType:    Banned,
		Reason:     Reason(opts[discord.OptBanReason].IntValue()),
		ReasonText: "",
		Note:       opts[discord.OptNote].StringValue(),
	}

	duration, errDuration := duration.Parse(opts[discord.OptDuration].StringValue())
	if errDuration != nil {
		return nil, errors.Join(errDuration, ErrInvalidBanDuration)
	}
	banOpts.Duration = duration

	banSteam, errBan := h.bans.Create(ctx, banOpts)
	if errBan != nil {
		if errors.Is(errBan, database.ErrDuplicate) {
			return nil, ErrDuplicateBan
		}

		return nil, discord.ErrCommandFailed
	}

	return CreateResponse(banSteam), nil
}

func (h discordHandler) onUnban(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := bot.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	reason := opts[discord.OptUnbanReason].StringValue()

	author, err := h.discord.GetPersonByDiscordID(ctx, interaction.Member.User.ID)
	if err != nil {
		return nil, err
	}

	steamID, errResolveSID := steamid.Resolve(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errResolveSID != nil || !steamID.Valid() {
		return nil, steamid.ErrInvalidSID
	}

	found, errUnban := h.bans.Unban(ctx, steamID, reason, author)
	if errUnban != nil {
		return nil, errUnban
	}

	if !found {
		return nil, ErrBanDoesNotExist
	}

	user, errUser := h.persons.GetOrCreatePersonBySteamID(ctx, steamID)
	if errUser != nil {
		slog.Warn("Could not fetch unbanned Person", slog.String("steam_id", steamID.String()), slog.String("error", errUser.Error()))
	}

	return UnbanMessage("/FIXME", user), nil
}

func (h discordHandler) onCheck(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, //nolint:maintidx
) (*discordgo.MessageEmbed, error) {
	opts := bot.OptionMap(interaction.ApplicationCommandData().Options)
	sid, errResolveSID := steamid.Resolve(ctx, opts[discord.OptUserIdentifier].StringValue())

	if errResolveSID != nil || !sid.Valid() {
		return nil, steamid.ErrInvalidSID
	}

	player, errGetPlayer := h.persons.GetOrCreatePersonBySteamID(ctx, sid)
	if errGetPlayer != nil {
		return nil, discord.ErrCommandFailed
	}

	bans, errGetBanBySID := h.bans.QueryOne(ctx, QueryOpts{EvadeOk: true, TargetID: sid})
	if errGetBanBySID != nil {
		if !errors.Is(errGetBanBySID, database.ErrNoResult) {
			slog.Error("Failed to get ban by steamid", slog.String("error", errGetBanBySID.Error()))

			return nil, discord.ErrCommandFailed
		}
	}

	oldBans, errOld := h.bans.Query(ctx, QueryOpts{})
	if errOld != nil {
		if !errors.Is(errOld, database.ErrNoResult) {
			slog.Error("Failed to fetch old bans", slog.String("error", errOld.Error()))
		}
	}

	// bannedNets, errGetBanNet := h.bans.Query(ctx, player.IPAddr)
	// if errGetBanNet != nil {
	// 	if !errors.Is(errGetBanNet, database.ErrNoResult) {
	// 		slog.Error("Failed to get ban nets by addr", log.ErrAttr(errGetBanNet))

	// 		return nil, helper.ErrCommandFailed
	// 	}
	// }

	var banURL string

	var (
		conf          = h.config.Config()
		authorProfile person.Core
	)

	// TODO Show the longest remaining ban.
	if bans.BanID > 0 {
		if bans.SourceID.Valid() {
			ap, errGetProfile := h.persons.GetOrCreatePersonBySteamID(ctx, bans.SourceID)
			if errGetProfile != nil {
				slog.Error("Failed to load author for ban", slog.String("error", errGetProfile.Error()))
			} else {
				authorProfile = ap
			}
		}

		banURL = conf.ExtURL(bans)
	}

	return CheckMessage(player, bans, banURL, authorProfile, oldBans), nil
}

func UnbanMessage(link string, person person.Info) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("User Unbanned Successfully")
	msgEmbed.Embed().
		SetColor(discord.ColourSuccess).
		SetImage(person.GetAvatar().Full()).
		SetURL(link)
	msgEmbed.AddFieldsSteamID(person.GetSteamID())

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func CheckMessage(player person.Core, banPerson Ban, banURL string, author person.Info, oldBans []Ban) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed()

	title := player.Name

	if banPerson.BanID > 0 {
		switch banPerson.BanType {
		case Banned:
			title += " (BANNED)"
		case NoComm:
			title += " (MUTED)"
		}
	}

	msgEmbed.Embed().SetTitle(title)

	// if player.RealName != "" {
	// 	msgEmbed.Embed().AddField("Real Name", player.RealName)
	// }

	// cd := time.Unix(player.TimeCreated, 0)
	// msgEmbed.Embed().AddField("Age", datetime.FmtDuration(cd))
	// msgEmbed.Embed().AddField("Private", strconv.FormatBool(player.VisibilityState == 1))
	// msgEmbed.AddFieldsSteamID(player.SteamID)

	// if player.VACBans > 0 {
	// 	msgEmbed.Embed().AddField("VAC Bans", fmt.Sprintf("count: %d days: %d", player.VACBans, player.DaysSinceLastBan))
	// }

	// if player.GameBans > 0 {
	// 	msgEmbed.Embed().AddField("Game Bans", fmt.Sprintf("count: %d", player.GameBans))
	// }

	// if player.CommunityBanned {
	// 	msgEmbed.Embed().AddField("Com. Ban", "true")
	// }

	// if player.EconomyBan != "" {
	// 	msgEmbed.Embed().AddField("Econ Ban", string(player.EconomyBan))
	// }

	if len(oldBans) > 0 {
		numMutes, numBans := 0, 0

		for _, oldBan := range oldBans {
			if oldBan.BanType == Banned {
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

	if banPerson.BanID > 0 {
		banned = banPerson.BanType == Banned
		muted = banPerson.BanType == NoComm
		reason := banPerson.ReasonText

		if len(reason) == 0 {
			// in case authorProfile ban without authorProfile reason ever makes its way here, we make sure
			// that Discord doesn't shit itself
			reason = "none"
		}

		expiry = banPerson.ValidUntil
		createdAt = banPerson.CreatedOn.Format(time.RFC3339)

		msgEmbed.Embed().SetURL(banURL)
		msgEmbed.Embed().AddField("Reason", reason)
		msgEmbed.Embed().AddField("Created", datetime.FmtTimeShort(banPerson.CreatedOn)).MakeFieldInline()

		if time.Until(expiry) > time.Hour*24*365*5 {
			msgEmbed.Embed().AddField("Expires", Permanent).MakeFieldInline()
		} else {
			msgEmbed.Embed().AddField("Expires", datetime.FmtDuration(expiry)).MakeFieldInline()
		}

		// msgEmbed.Embed().AddField("Author", fmt.Sprintf("<@%s>", author.GetDiscordID())).MakeFieldInline()

		if banPerson.Note != "" {
			msgEmbed.Embed().AddField("Mod Note", banPerson.Note).MakeFieldInline()
		}

		msgEmbed.AddAuthorPersonInfo(author, "")
	}

	// if len(bannedNets) > 0 {
	// 	// ip = bannedNets[0].CIDR.String()
	// 	netReason := fmt.Sprintf("Banned from %d networks", len(bannedNets))
	// 	netExpiry := bannedNets[0].ValidUntil
	// 	msgEmbed.Embed().AddField("Network Bans", strconv.Itoa(len(bannedNets)))
	// 	msgEmbed.Embed().AddField("Network Reason", netReason)
	// 	msgEmbed.Embed().AddField("Network Expires", datetime.FmtDuration(netExpiry)).MakeFieldInline()
	// }

	banStateStr := "no"

	if banned {
		// #992D22 red
		color = discord.ColourError
		banStateStr = "banned"
	}

	if muted {
		// #E67E22 orange
		color = discord.ColourWarn
		banStateStr = "muted"
	}

	msgEmbed.Embed().AddField("Ban/Muted", banStateStr)

	// if player.IPAddr.IsValid() {
	// 	msgEmbed.Embed().AddField("Last IP", player.IPAddr.String()).MakeFieldInline()
	// }

	// if asn.ASName != "" {
	// 	msgEmbed.Embed().AddField("ASN", fmt.Sprintf("(%d) %s", asn.ASNum, asn.ASName)).MakeFieldInline()
	// }

	// if location.CountryCode != "" {
	// 	msgEmbed.Embed().AddField("City", location.CityName).MakeFieldInline()
	// }

	// if location.CountryName != "" {
	// 	msgEmbed.Embed().AddField("Country", location.CountryName).MakeFieldInline()
	// }

	// if proxy.CountryCode != "" {
	// 	msgEmbed.Embed().AddField("Proxy Type", string(proxy.ProxyType)).MakeFieldInline()
	// 	msgEmbed.Embed().AddField("Proxy", string(proxy.Threat)).MakeFieldInline()
	// }

	// if logData.Logs > 0 {
	// 	msgEmbed.Embed().AddField("Logs.tf", strconv.Itoa(int(logData.Logs))).MakeFieldInline()
	// }

	if createdAt != "" {
		msgEmbed.Embed().AddField("created at", createdAt).MakeFieldInline()
	}

	return msgEmbed.Embed().
		SetURL(player.Path()).
		SetColor(color).
		SetImage(player.GetAvatar().Full()).
		SetThumbnail(player.GetAvatar().Small()).
		Truncate().MessageEmbed
}

func IPMessage() *discordgo.MessageEmbed {
	return discord.NewEmbed("IP ban created successfully").Embed().
		SetColor(discord.ColourSuccess).
		Truncate().
		MessageEmbed
}

func ReportStatusChangeMessage(report ReportWithAuthor, fromStatus ReportStatus, link string) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed(
		"Report status changed",
		fmt.Sprintf("Changed from %s to %s", fromStatus.String(), report.ReportStatus.String()),
		link)

	msgEmbed.Embed().
		AddField("report_id", strconv.FormatInt(report.ReportID, 10))
	msgEmbed.AddAuthorPersonInfo(report.Author, link)
	msgEmbed.AddTargetPerson(report.Subject)

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func MuteMessage(target steamid.SteamID) *discordgo.MessageEmbed {
	embed := discord.NewEmbed("Player muted successfully")
	embed.AddFieldsSteamID(target)

	return embed.Embed().SetColor(discord.ColourSuccess).Truncate().MessageEmbed
}

func KickPlayerEmbed(target person.Info) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("User Kicked Successfully")
	msgEmbed.Embed().SetColor(discord.ColourSuccess)

	return msgEmbed.AddTargetPerson(target).Embed().MessageEmbed
}

func KickPlayerOnConnectEmbed(steamID steamid.SteamID, name string, target person.Info, banSource string) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("User Kicked Successfully")
	msgEmbed.Embed().SetColor(discord.ColourWarn)
	msgEmbed.AddTargetPerson(target)
	msgEmbed.Embed().
		AddField("Connecting As", name).
		AddField("Ban Source", banSource)

	return msgEmbed.AddFieldsSteamID(steamID).Embed().MessageEmbed
}

func SilenceEmbed(target person.Info) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("User Silenced Successfully")
	msgEmbed.Embed().SetColor(discord.ColourSuccess)

	return msgEmbed.AddTargetPerson(target).Embed().MessageEmbed
}

func ExpiredMessage(inBan Ban, person person.Info, banURL string) *discordgo.MessageEmbed {
	banType := "Ban"
	if inBan.BanType == NoComm {
		banType = "Mute"
	}

	msgEmbed := discord.NewEmbed("Steam Ban Expired")
	msgEmbed.
		Embed().
		SetColor(discord.ColourInfo).
		AddField("Type", banType).
		SetImage(person.GetAvatar().Full()).
		AddField("Name", person.GetName()).
		SetURL(banURL)

	msgEmbed.AddFieldsSteamID(person.GetSteamID())

	if inBan.BanType == NoComm {
		msgEmbed.Embed().SetColor(discord.ColourWarn)
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func CreateResponse(banSteam Ban) *discordgo.MessageEmbed {
	var (
		title  string
		colour int
	)

	if banSteam.BanType == NoComm {
		title = fmt.Sprintf("User Muted (#%d)", banSteam.BanID)
		colour = discord.ColourWarn
	} else {
		title = fmt.Sprintf("User Banned (#%d)", banSteam.BanID)
		colour = discord.ColourError
	}

	msgEmbed := discord.NewEmbed(title)
	msgEmbed.Embed().
		SetColor(colour).
		SetURL("https://steamcommunity.com/profiles/" + banSteam.TargetID.String())

	msgEmbed.AddFieldsSteamID(banSteam.TargetID)

	//	msgEmbed.Embed().SetAuthor(banSteam.SourcePersonaname, domain.NewAvatar(banSteam.SourceAvatarhash).Full(), "https://steamcommunity.com/profiles/"+banSteam.SourceID.String())

	expIn := Permanent
	expAt := Permanent

	if banSteam.ValidUntil.Year()-time.Now().Year() < 5 {
		expIn = datetime.FmtDuration(banSteam.ValidUntil)
		expAt = datetime.FmtTimeShort(banSteam.ValidUntil)
	}

	msgEmbed.
		Embed().
		AddField("Expires In", expIn).
		AddField("Expires At", expAt)

	if banSteam.Note != "" {
		msgEmbed.Emb.Description = banSteam.Note
	}

	msgEmbed.Emb.URL = "https://steamcommunity.com/profiles/" + banSteam.TargetID.String()

	return msgEmbed.Embed().MessageEmbed
}

func DeleteReportMessage(existing ReportMessage, user person.Info, userURL string) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("User report message deleted")
	msgEmbed.
		Embed().
		SetDescription(existing.MessageMD).
		SetColor(discord.ColourWarn)

	return msgEmbed.AddAuthorPersonInfo(user, userURL).Embed().Truncate().MessageEmbed
}

func NewReportMessageResponse(msg string, link string, author person.Info, authorURL string) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("New report message posted")
	msgEmbed.
		Embed().
		SetDescription(msg).
		SetColor(discord.ColourSuccess).
		SetURL(link)

	return msgEmbed.AddAuthorPersonInfo(author, authorURL).Embed().Truncate().MessageEmbed
}

func EditReportMessageResponse(body string, oldBody string, link string, author person.Info, authorURL string) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("New report message edited")
	msgEmbed.
		Embed().
		SetDescription(body).
		SetColor(discord.ColourWarn).
		AddField("Old Message", oldBody).
		SetURL(link)

	return msgEmbed.AddAuthorPersonInfo(author, authorURL).Embed().Truncate().MessageEmbed
}

func NewAppealMessage(msg string, link string, author person.Info, authorURL string) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("New ban appeal message posted")
	msgEmbed.
		Embed().
		SetColor(discord.ColourInfo).
		// SetThumbnail(bannedPerson.TargetAvatarhash).
		SetDescription(msg).
		SetURL(link)

	return msgEmbed.AddAuthorPersonInfo(author, authorURL).Embed().Truncate().MessageEmbed
}

func EditAppealMessage(existing AppealMessage, body string, author person.Info, authorURL string) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("Ban appeal message edited")
	msgEmbed.
		Embed().
		SetDescription(stringutil.DiffString(existing.MessageMD, body)).
		SetColor(discord.ColourWarn)

	return msgEmbed.
		AddAuthorPersonInfo(author, authorURL).
		Embed().
		Truncate().
		MessageEmbed
}

func DeleteAppealMessage(existing *AppealMessage, user person.Info, userURL string) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("User appeal message deleted")
	msgEmbed.
		Embed().
		SetDescription(existing.MessageMD)

	return msgEmbed.
		AddAuthorPersonInfo(user, userURL).
		Embed().
		Truncate().
		MessageEmbed
}

func EditBanAppealStatusMessage() *discordgo.MessageEmbed {
	return discord.NewEmbed("Ban state updates").Embed().MessageEmbed
}

func ReportStatsMessage(meta ReportMeta, url string) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("User Report Stats")
	msgEmbed.
		Embed().
		SetColor(discord.ColourSuccess).
		SetURL(url)

	if meta.OpenWeek > 0 {
		msgEmbed.Embed().SetColor(discord.ColourError)
	} else if meta.Open3Days > 0 {
		msgEmbed.Embed().SetColor(discord.ColourWarn)
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

func NewInGameReportResponse(report ReportWithAuthor, reportURL string, author person.Info, authorURL string, _ string) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("New User Report Created")
	msgEmbed.
		Embed().
		SetDescription(report.Description).
		SetColor(discord.ColourSuccess).
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
