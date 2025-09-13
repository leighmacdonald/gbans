package ban

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/discord/helper"
	"github.com/leighmacdonald/gbans/internal/discord/message"
	"github.com/leighmacdonald/gbans/internal/domain"
	banDomain "github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/datetime"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	slashCommands = []*discordgo.ApplicationCommand{
		{
			Name:                     "ban",
			Description:              "Manage steam, ip, group and ASN bans",
			DMPermission:             &helper.DmPerms,
			DefaultMemberPermissions: &helper.ModPerms,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "steam",
					Description: "Ban and kick a user from all servers",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						&discordgo.ApplicationCommandOption{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        helper.OptUserIdentifier,
							Description: "SteamID in any format OR profile url",
							Required:    true,
						},
						&discordgo.ApplicationCommandOption{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        helper.OptDuration,
							Description: "Duration [s,m,h,d,w,M,y]N|0",
							Required:    true,
						},
						optBanReason,
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        helper.OptNote,
							Description: "Mod only notes for the ban reason",
							Required:    true,
						},
					},
				},
			},
		},
		{
			Name:                     "unban",
			Description:              "Manage steam, ip and ASN bans",
			DMPermission:             &helper.DmPerms,
			DefaultMemberPermissions: &helper.ModPerms,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "steam",
					Description: "Unban a previously banned player",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        helper.OptUserIdentifier,
							Description: "SteamID in any format OR profile url",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        helper.OptUnbanReason,
							Description: "Reason for unbanning",
							Required:    true,
						},
					},
				},
			},
		},
		{
			Name:                     "mute",
			Description:              "Mute a player",
			DMPermission:             &helper.DmPerms,
			DefaultMemberPermissions: &helper.ModPerms,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        helper.OptUserIdentifier,
					Description: "SteamID in any format OR profile url",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        helper.OptDuration,
					Description: "Duration [s,m,h,d,w,M,y]N|0",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        helper.OptBanReason,
					Description: "Reason for the ban/mute",
					Required:    true,
					Choices:     reasons,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        helper.OptNote,
					Description: "Mod only notes for the mute reason",
					Required:    true,
				},
			},
		},
		{
			Name:                     "check",
			DMPermission:             &helper.DmPerms,
			DefaultMemberPermissions: &helper.ModPerms,
			Description:              "Get ban status for a steam id",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        helper.OptUserIdentifier,
					Description: "SteamID in any format OR profile url",
					Required:    true,
				},
			},
		},
		{
			Name:                     "checkip",
			DMPermission:             &helper.DmPerms,
			DefaultMemberPermissions: &helper.ModPerms,
			Description:              "Check if a ip is banned",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        helper.OptIP,
					Description: "IP address to check",
					Required:    true,
				},
			},
		},
	}
	reasonCollection = []banDomain.Reason{
		banDomain.External, banDomain.Cheating, banDomain.Racism, banDomain.Harassment, banDomain.Exploiting,
		banDomain.WarningsExceeded, banDomain.Spam, banDomain.Language, banDomain.Profile, banDomain.ItemDescriptions,
		banDomain.BotHost, banDomain.Evading, banDomain.Username, banDomain.Custom,
	}

	reasons = []*discordgo.ApplicationCommandOptionChoice{
		&discordgo.ApplicationCommandOptionChoice{Name: banDomain.External.String(), Value: banDomain.External},
		&discordgo.ApplicationCommandOptionChoice{Name: banDomain.Cheating.String(), Value: banDomain.Cheating},
		&discordgo.ApplicationCommandOptionChoice{Name: banDomain.Racism.String(), Value: banDomain.Racism},
		&discordgo.ApplicationCommandOptionChoice{Name: banDomain.Harassment.String(), Value: banDomain.Harassment},
		&discordgo.ApplicationCommandOptionChoice{Name: banDomain.Exploiting.String(), Value: banDomain.Exploiting},
		&discordgo.ApplicationCommandOptionChoice{Name: banDomain.WarningsExceeded.String(), Value: banDomain.WarningsExceeded},
		&discordgo.ApplicationCommandOptionChoice{Name: banDomain.Spam.String(), Value: banDomain.Spam},
		&discordgo.ApplicationCommandOptionChoice{Name: banDomain.Language.String(), Value: banDomain.Language},
		&discordgo.ApplicationCommandOptionChoice{Name: banDomain.Profile.String(), Value: banDomain.Profile},
		&discordgo.ApplicationCommandOptionChoice{Name: banDomain.ItemDescriptions.String(), Value: banDomain.ItemDescriptions},
		&discordgo.ApplicationCommandOptionChoice{Name: banDomain.BotHost.String(), Value: banDomain.BotHost},
		&discordgo.ApplicationCommandOptionChoice{Name: banDomain.Evading.String(), Value: banDomain.Evading},
		&discordgo.ApplicationCommandOptionChoice{Name: banDomain.Username.String(), Value: banDomain.Username},
		&discordgo.ApplicationCommandOptionChoice{Name: banDomain.Custom.String(), Value: banDomain.Custom},
	}

	optBanReason = &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        helper.OptBanReason,
		Description: "Reason for the ban/mute",
		Required:    true,
		Choices:     reasons,
	}
)

type DiscordHandler struct {
	bans *BanUsecase
}

func RegisterDiscordHandler(bans *BanUsecase) {
	_ = &DiscordHandler{
		bans: bans,
	}
}

func (h DiscordHandler) onMute(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := helper.OptionMap(interaction.ApplicationCommandData().Options)

	playerID, errPlayerID := steamid.Resolve(ctx, opts.String(helper.OptUserIdentifier))
	if errPlayerID != nil || !playerID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	reasonValueOpt, ok := opts[helper.OptBanReason]
	if !ok {
		return nil, domain.ErrReasonInvalid
	}

	author, errAuthor := h.getDiscordAuthor(ctx, interaction)
	if errAuthor != nil {
		return nil, errAuthor
	}
	banOpts := BanOpts{
		Origin:     banDomain.Bot,
		SourceID:   author.SteamID,
		TargetID:   steamid.New(opts.String(helper.OptUserIdentifier)),
		BanType:    banDomain.NoComm,
		Reason:     banDomain.Reason(reasonValueOpt.IntValue()),
		ReasonText: "",
		Note:       opts[helper.OptNote].StringValue(),
	}
	banOpts.SetDuration(opts[helper.OptDuration].StringValue())

	banSteam, errBan := h.bans.Ban(ctx, banOpts)
	if errBan != nil {
		return nil, errBan
	}

	return MuteMessage(banSteam), nil
}

// onBanSteam !ban <id> <duration> [reason].
func (h DiscordHandler) onBan(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := helper.OptionMap(interaction.ApplicationCommandData().Options[0].Options)

	author, errAuthor := h.getDiscordAuthor(ctx, interaction)
	if errAuthor != nil {
		return nil, errAuthor
	}

	banOpts := BanOpts{
		Origin:     banDomain.Bot,
		SourceID:   author.SteamID,
		TargetID:   steamid.New(opts[helper.OptUserIdentifier].StringValue()),
		BanType:    banDomain.Banned,
		Reason:     banDomain.Reason(opts[helper.OptBanReason].IntValue()),
		ReasonText: "",
		Note:       opts[helper.OptNote].StringValue(),
	}
	banOpts.SetDuration(opts[helper.OptDuration].StringValue())

	banSteam, errBan := h.bans.Ban(ctx, banOpts)
	if errBan != nil {
		if errors.Is(errBan, database.ErrDuplicate) {
			return nil, domain.ErrDuplicateBan
		}

		return nil, helper.ErrCommandFailed
	}

	return BanSteamResponse(banSteam), nil
}

func (h DiscordHandler) onUnban(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := helper.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	reason := opts[helper.OptUnbanReason].StringValue()

	author, err := h.getDiscordAuthor(ctx, interaction)
	if err != nil {
		return nil, err
	}

	steamID, errResolveSID := steamid.Resolve(ctx, opts[helper.OptUserIdentifier].StringValue())
	if errResolveSID != nil || !steamID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	found, errUnban := h.bans.Unban(ctx, steamID, reason, author.ToUserProfile())
	if errUnban != nil {
		return nil, errUnban
	}

	if !found {
		return nil, domain.ErrBanDoesNotExist
	}

	user, errUser := h.persons.GetPersonBySteamID(ctx, nil, steamID)
	if errUser != nil {
		slog.Warn("Could not fetch unbanned Person", slog.String("steam_id", steamID.String()), log.ErrAttr(errUser))
	}

	return UnbanMessage(h.config, user), nil
}

func (h DiscordHandler) onCheck(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, //nolint:maintidx
) (*discordgo.MessageEmbed, error) {
	opts := helper.OptionMap(interaction.ApplicationCommandData().Options)
	sid, errResolveSID := steamid.Resolve(ctx, opts[helper.OptUserIdentifier].StringValue())

	if errResolveSID != nil || !sid.Valid() {
		return nil, domain.ErrInvalidSID
	}

	player, errGetPlayer := h.persons.GetOrCreatePersonBySteamID(ctx, nil, sid)
	if errGetPlayer != nil {
		return nil, helper.ErrCommandFailed
	}

	bans, errGetBanBySID := h.bans.Query(ctx, QueryOpts{EvadeOk: true, TargetID: sid})
	if errGetBanBySID != nil {
		if !errors.Is(errGetBanBySID, database.ErrNoResult) {
			slog.Error("Failed to get ban by steamid", log.ErrAttr(errGetBanBySID))

			return nil, helper.ErrCommandFailed
		}
	}

	oldBans, errOld := h.bans.Query(ctx, QueryOpts{})
	if errOld != nil {
		if !errors.Is(errOld, database.ErrNoResult) {
			slog.Error("Failed to fetch old bans", log.ErrAttr(errOld))
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
		conf = h.config.Config()

		authorProfile person.Person
	)

	// TODO Show the longest remaining ban.
	if bans.BanID > 0 {
		if ban.SourceID.Valid() {
			ap, errGetProfile := h.persons.GetPersonBySteamID(ctx, nil, bans.SourceID)
			if errGetProfile != nil {
				slog.Error("Failed to load author for ban", log.ErrAttr(errGetProfile))
			} else {
				authorProfile = ap
			}
		}

		banURL = conf.ExtURL(bans.Ban)
	}

	logData, errLogs := h.tfAPI.LogsTFSummary(ctx, sid)
	if errLogs != nil {
		slog.Info("Failed to query logstf summary", slog.String("error", errLogs.Error()))
	}

	network, errNetwork := h.network.QueryNetwork(ctx, player.IPAddr)
	if errNetwork != nil {
		slog.Error("Failed to query network details")
	}

	return CheckMessage(player, ban, banURL, authorProfile, oldBans, network.Location, network.Proxy, logData), nil
}

func UnbanMessage(link string, person domain.PersonInfo) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("User Unbanned Successfully")
	msgEmbed.Embed().
		SetColor(message.ColourSuccess).
		SetImage(person.GetAvatar().Full()).
		SetURL(link)
	msgEmbed.AddFieldsSteamID(person.GetSteamID())

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func CheckMessage(player person.Person, banPerson Ban, banURL string, author domain.PersonInfo,
	oldBans []Ban, location network.NetworkLocation,
	proxy network.NetworkProxy, logData thirdparty.LogsTFPlayerSummary,
) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed()

	title := player.PersonaName

	if banPerson.BanID > 0 {
		switch banPerson.BanType {
		case banDomain.Banned:
			title += " (BANNED)"
		case banDomain.NoComm:
			title += " (MUTED)"
		}
	}

	msgEmbed.Embed().SetTitle(title)

	if player.RealName != "" {
		msgEmbed.Embed().AddField("Real Name", player.RealName)
	}

	cd := time.Unix(player.TimeCreated, 0)
	msgEmbed.Embed().AddField("Age", datetime.FmtDuration(cd))
	msgEmbed.Embed().AddField("Private", strconv.FormatBool(player.VisibilityState == 1))
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
			if oldBan.BanType == banDomain.Banned {
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
		banned = banPerson.BanType == banDomain.Banned
		muted = banPerson.BanType == banDomain.NoComm
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
			msgEmbed.Embed().AddField("Expires", "Permanent").MakeFieldInline()
		} else {
			msgEmbed.Embed().AddField("Expires", datetime.FmtDuration(expiry)).MakeFieldInline()
		}

		msgEmbed.Embed().AddField("Author", fmt.Sprintf("<@%s>", author.GetDiscordID())).MakeFieldInline()

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
		color = message.ColourError
		banStateStr = "banned"
	}

	if muted {
		// #E67E22 orange
		color = message.ColourWarn
		banStateStr = "muted"
	}

	msgEmbed.Embed().AddField("Ban/Muted", banStateStr)

	if player.IPAddr.IsValid() {
		msgEmbed.Embed().AddField("Last IP", player.IPAddr.String()).MakeFieldInline()
	}

	// if asn.ASName != "" {
	// 	msgEmbed.Embed().AddField("ASN", fmt.Sprintf("(%d) %s", asn.ASNum, asn.ASName)).MakeFieldInline()
	// }

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

	if logData.Logs > 0 {
		msgEmbed.Embed().AddField("Logs.tf", strconv.Itoa(int(logData.Logs))).MakeFieldInline()
	}

	if createdAt != "" {
		msgEmbed.Embed().AddField("created at", createdAt).MakeFieldInline()
	}

	return msgEmbed.Embed().
		SetURL(player.Profile()).
		SetColor(color).
		SetImage(player.AvatarFull()).
		SetThumbnail(player.Avatar()).
		Truncate().MessageEmbed
}

func BanIPMessage() *discordgo.MessageEmbed {
	return message.NewEmbed("IP ban created successfully").Embed().
		SetColor(message.ColourSuccess).
		Truncate().
		MessageEmbed
}

func ReportStatusChangeMessage(report ReportWithAuthor, fromStatus ReportStatus, link string) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed(
		"Report status changed",
		fmt.Sprintf("Changed from %s to %s", fromStatus.String(), report.ReportStatus.String()),
		link)

	msgEmbed.Embed().
		AddField("report_id", strconv.FormatInt(report.ReportID, 10))
	msgEmbed.AddAuthorPersonInfo(report.Author, link)
	msgEmbed.AddTargetPerson(report.Subject)

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func MuteMessage(banSteam Ban) *discordgo.MessageEmbed {
	embed := message.NewEmbed("Player muted successfully")
	embed.AddFieldsSteamID(banSteam.TargetID)

	return embed.Embed().SetColor(message.ColourSuccess).Truncate().MessageEmbed
}

func KickPlayerEmbed(target domain.PersonInfo) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("User Kicked Successfully")
	msgEmbed.Embed().SetColor(message.ColourSuccess)

	return msgEmbed.AddTargetPerson(target).Embed().MessageEmbed
}

func KickPlayerOnConnectEmbed(steamID steamid.SteamID, name string, target domain.PersonInfo, banSource string) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("User Kicked Successfully")
	msgEmbed.Embed().SetColor(message.ColourWarn)
	msgEmbed.AddTargetPerson(target)
	msgEmbed.Embed().
		AddField("Connecting As", name).
		AddField("Ban Source", banSource)

	return msgEmbed.AddFieldsSteamID(steamID).Embed().MessageEmbed
}

func SilenceEmbed(target domain.PersonInfo) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("User Silenced Successfully")
	msgEmbed.Embed().SetColor(message.ColourSuccess)

	return msgEmbed.AddTargetPerson(target).Embed().MessageEmbed
}

func BanExpiresMessage(inBan Ban, person domain.PersonInfo, banURL string) *discordgo.MessageEmbed {
	banType := "Ban"
	if inBan.BanType == banDomain.NoComm {
		banType = "Mute"
	}

	msgEmbed := message.NewEmbed("Steam Ban Expired")
	msgEmbed.
		Embed().
		SetColor(message.ColourInfo).
		AddField("Type", banType).
		SetImage(person.GetAvatar().Full()).
		AddField("Name", person.GetName()).
		SetURL(banURL)

	msgEmbed.AddFieldsSteamID(person.GetSteamID())

	if inBan.BanType == banDomain.NoComm {
		msgEmbed.Embed().SetColor(message.ColourWarn)
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func BanSteamResponse(banSteam Ban) *discordgo.MessageEmbed {
	var (
		title  string
		colour int
	)

	if banSteam.BanType == banDomain.NoComm {
		title = fmt.Sprintf("User Muted (#%d)", banSteam.BanID)
		colour = message.ColourWarn
	} else {
		title = fmt.Sprintf("User Banned (#%d)", banSteam.BanID)
		colour = message.ColourError
	}

	msgEmbed := message.NewEmbed(title)
	msgEmbed.Embed().
		SetColor(colour).
		SetURL("https://steamcommunity.com/profiles/" + banSteam.TargetID.String())

	msgEmbed.AddFieldsSteamID(banSteam.TargetID)

	//	msgEmbed.Embed().SetAuthor(banSteam.SourcePersonaname, domain.NewAvatar(banSteam.SourceAvatarhash).Full(), "https://steamcommunity.com/profiles/"+banSteam.SourceID.String())

	expIn := "Permanent"
	expAt := "Permanent"

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

func DeleteReportMessage(existing ReportMessage, user domain.PersonInfo, userURL string) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("User report message deleted")
	msgEmbed.
		Embed().
		SetDescription(existing.MessageMD).
		SetColor(message.ColourWarn)

	return msgEmbed.AddAuthorPersonInfo(user, userURL).Embed().Truncate().MessageEmbed
}

func NewReportMessageResponse(msg string, link string, author domain.PersonInfo, authorURL string) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("New report message posted")
	msgEmbed.
		Embed().
		SetDescription(msg).
		SetColor(message.ColourSuccess).
		SetURL(link)

	return msgEmbed.AddAuthorPersonInfo(author, authorURL).Embed().Truncate().MessageEmbed
}

func EditReportMessageResponse(body string, oldBody string, link string, author domain.PersonInfo, authorURL string) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("New report message edited")
	msgEmbed.
		Embed().
		SetDescription(body).
		SetColor(message.ColourWarn).
		AddField("Old Message", oldBody).
		SetURL(link)

	return msgEmbed.AddAuthorPersonInfo(author, authorURL).Embed().Truncate().MessageEmbed
}

func NewAppealMessage(msg string, link string, author domain.PersonInfo, authorURL string) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("New ban appeal message posted")
	msgEmbed.
		Embed().
		SetColor(message.ColourInfo).
		// SetThumbnail(bannedPerson.TargetAvatarhash).
		SetDescription(msg).
		SetURL(link)

	return msgEmbed.AddAuthorPersonInfo(author, authorURL).Embed().Truncate().MessageEmbed
}

func EditAppealMessage(existing BanAppealMessage, body string, author domain.PersonInfo, authorURL string) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("Ban appeal message edited")
	msgEmbed.
		Embed().
		SetDescription(stringutil.DiffString(existing.MessageMD, body)).
		SetColor(message.ColourWarn)

	return msgEmbed.
		AddAuthorPersonInfo(author, authorURL).
		Embed().
		Truncate().
		MessageEmbed
}

func DeleteAppealMessage(existing *BanAppealMessage, user domain.PersonInfo, userURL string) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("User appeal message deleted")
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
	return message.NewEmbed("Ban state updates").Embed().MessageEmbed
}

func ReportStatsMessage(meta ReportMeta, url string) *discordgo.MessageEmbed {
	msgEmbed := message.NewEmbed("User Report Stats")
	msgEmbed.
		Embed().
		SetColor(message.ColourSuccess).
		SetURL(url)

	if meta.OpenWeek > 0 {
		msgEmbed.Embed().SetColor(message.ColourError)
	} else if meta.Open3Days > 0 {
		msgEmbed.Embed().SetColor(message.ColourWarn)
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
