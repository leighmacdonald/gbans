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
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/datetime"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/sosodev/duration"
)

type discordHandler struct {
	Bans

	persons person.Provider
	discord person.DiscordPersonProvider
}

func RegisterDiscordCommands(bot discord.Service, bans Bans, persons person.Provider, discordProv person.DiscordPersonProvider) {
	var (
		handler = &discordHandler{Bans: bans, persons: persons, discord: discordProv}

		reasons = []*discordgo.ApplicationCommandOptionChoice{
			{Name: reason.External.String(), Value: reason.External},
			{Name: reason.Cheating.String(), Value: reason.Cheating},
			{Name: reason.Racism.String(), Value: reason.Racism},
			{Name: reason.Harassment.String(), Value: reason.Harassment},
			{Name: reason.Exploiting.String(), Value: reason.Exploiting},
			{Name: reason.WarningsExceeded.String(), Value: reason.WarningsExceeded},
			{Name: reason.Spam.String(), Value: reason.Spam},
			{Name: reason.Language.String(), Value: reason.Language},
			{Name: reason.Profile.String(), Value: reason.Profile},
			{Name: reason.ItemDescriptions.String(), Value: reason.ItemDescriptions},
			{Name: reason.BotHost.String(), Value: reason.BotHost},
			{Name: reason.Evading.String(), Value: reason.Evading},
			{Name: reason.Username.String(), Value: reason.Username},
			{Name: reason.Custom.String(), Value: reason.Custom},
		}
	)

	bot.MustRegisterHandler("ban", &discordgo.ApplicationCommand{
		Name:                     "ban",
		Description:              "Manage steam, ip, group and ASN bans",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: &discord.ModPerms,
	}, handler.onBan, discord.CommandTypeModal, handler.onBanResponse)

	bot.MustRegisterHandler("unban", &discordgo.ApplicationCommand{
		Name:                     "unban",
		Description:              "Manage steam, ip and ASN bans",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
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
	}, handler.onUnban, discord.CommandTypeCLI)

	bot.MustRegisterHandler("mute", &discordgo.ApplicationCommand{
		Name:                     "mute",
		Description:              "Mute a player",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
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
	}, handler.onMute, discord.CommandTypeCLI)

	bot.MustRegisterHandler("check", &discordgo.ApplicationCommand{
		Name:                     "check",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
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
	}, handler.onCheck, discord.CommandTypeCLI)
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
		SourceID:   author.GetSteamID(),
		TargetID:   steamid.New(opts.String(discord.OptUserIdentifier)),
		BanType:    bantype.NoComm,
		Reason:     reason.Reason(reasonValueOpt.IntValue()),
		ReasonText: "",
		Note:       opts[discord.OptNote].StringValue(),
	}

	parsedDuration, errDuration := duration.Parse(opts[discord.OptDuration].StringValue())
	if errDuration != nil {
		return nil, errors.Join(errDuration, ErrInvalidBanDuration)
	}

	banOpts.Duration = parsedDuration

	banSteam, errBan := h.Create(ctx, banOpts)
	if errBan != nil {
		return nil, errBan
	}

	return MuteMessage(banSteam.TargetID), nil
}

var durationMap = map[string]string{
	"15 Mins":   "PT15M",
	"6 Hours":   "PT6H",
	"12 Hours":  "PT12H",
	"1 Day":     "P1D",
	"2 Days":    "P2D",
	"3 Days":    "P3D",
	"1 Week":    "P1W",
	"2 Weeks":   "P2W",
	"1 Month":   "P1M",
	"3 Months":  "P3M",
	"6 Months":  "P6M",
	"1 Year":    "P1Y",
	"Permanent": "P0",
	"Custom":    "custom",
}

func (h discordHandler) onBanResponse(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	author, errAuthor := h.discord.GetPersonByDiscordID(ctx, interaction.Member.User.ID)
	if errAuthor != nil {
		return nil, errAuthor
	}

	banOpts := Opts{
		Origin:     Bot,
		SourceID:   author.GetSteamID(),
		BanType:    bantype.Banned,
		ReasonText: "",
	}

	data := interaction.ModalSubmitData()

	for _, component := range data.Components {
		switch component.Type() {
		case discordgo.ActionsRowComponent:
			row := component.(*discordgo.ActionsRow)
			for _, comp := range row.Components {
				switch comp.Type() {
				case discordgo.TextInputComponent:
					choice, ok := comp.(*discordgo.TextInput)
					if !ok {
						slog.Error("Failed to cast to textinput")

						return nil, nil
					}

					switch choice.ID {
					case idSteamID:
						sid, errSID := steamid.Resolve(ctx, choice.Value)
						if errSID != nil {
							return nil, errSID
						}
						if !sid.Valid() {
							return nil, steamid.ErrInvalidSID
						}
						banOpts.TargetID = sid
					case idCIDR:
						banOpts.CIDR = &choice.Value
					case idNotes:
						banOpts.Note = choice.Value
					default:
						continue
					}
				}
			}
		case discordgo.LabelComponent:
			row := component.(*discordgo.Label)
			comp := row.Component.(*discordgo.SelectMenu)
			switch comp.ID {
			case idReason:
				reasonValue, errReason := strconv.Atoi(comp.Values[0])
				if errReason != nil {
					return nil, errReason
				}
				banOpts.Reason = reason.Reason(reasonValue)
			case idDuration:
				durationValue, errDuration := duration.Parse(comp.Values[0])
				if errDuration != nil {
					return nil, ErrInvalidBanDuration
				}
				banOpts.Duration = durationValue
			default:
				continue
			}
		}
	}

	banSteam, errBan := h.Create(ctx, banOpts)
	if errBan != nil {
		if errors.Is(errBan, database.ErrDuplicate) {
			return nil, ErrDuplicateBan
		}

		return nil, discord.ErrCommandFailed
	}

	return CreateResponse(banSteam), nil
}

const (
	idSteamID = iota + 1
	idCIDR
	idReason
	idDuration
	idNotes
)

func (h discordHandler) onBan(_ context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	banOpts := make([]discordgo.SelectMenuOption, len(reason.Reasons))
	for index, op := range reason.Reasons {
		banOpts[index] = discordgo.SelectMenuOption{
			Label: op.String(),
			Value: strconv.Itoa(int(op)),
		}
	}

	var index int
	durationOpts := make([]discordgo.SelectMenuOption, len(durationMap))
	for label, value := range durationMap {
		durationOpts[index] = discordgo.SelectMenuOption{
			Label: label,
			Value: value,
		}
		index++
	}

	minItems := 1
	if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "ban_resp_" + interaction.Interaction.Member.User.ID,
			Title:    "Ban Player",
			Flags:    discordgo.MessageFlagsIsComponentsV2 | discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							ID:          idSteamID,
							CustomID:    "steamid",
							Label:       "SteamID or Profile URL",
							Style:       discordgo.TextInputShort,
							Placeholder: "76561197960542812",
							Required:    true,
							MaxLength:   64,
							MinLength:   0,
						},
					},
				},
				discordgo.Label{
					Label: "Reason",
					Component: discordgo.SelectMenu{
						ID:          idReason,
						CustomID:    "reason",
						Placeholder: "Select a reason",
						MaxValues:   1,
						MinValues:   &minItems,
						MenuType:    discordgo.StringSelectMenu,
						Options:     banOpts,
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							ID:          idCIDR,
							CustomID:    "cidr",
							Label:       "IP/CIDR Ban",
							Style:       discordgo.TextInputShort,
							Placeholder: "",
							Required:    false,
							MaxLength:   64,
							MinLength:   0,
						},
					},
				},
				discordgo.Label{
					Label: "Duration",
					Component: discordgo.SelectMenu{
						ID:          idDuration,
						CustomID:    "duration",
						Placeholder: "Select a duration",
						MaxValues:   1,
						MinValues:   &minItems,
						MenuType:    discordgo.StringSelectMenu,
						Options:     durationOpts,
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							ID:          idNotes,
							CustomID:    "notes",
							Label:       "Extended moderator only notes",
							Style:       discordgo.TextInputParagraph,
							Placeholder: "",
							Required:    false,
							MaxLength:   2000,
							MinLength:   0,
						},
					},
				},
			},
		},
	}); err != nil {
		slog.Error(err.Error())

		return nil, err
	}

	return nil, nil
}

func (h discordHandler) onUnban(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := bot.OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	unbanReason := opts[discord.OptUnbanReason].StringValue()

	author, err := h.discord.GetPersonByDiscordID(ctx, interaction.Member.User.ID)
	if err != nil {
		return nil, err
	}

	steamID, errResolveSID := steamid.Resolve(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errResolveSID != nil || !steamID.Valid() {
		return nil, steamid.ErrInvalidSID
	}

	found, errUnban := h.Unban(ctx, steamID, unbanReason, author)
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

	return UnbanMessage(user), nil
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

	bans, errGetBanBySID := h.QueryOne(ctx, QueryOpts{EvadeOk: true, TargetID: sid})
	if errGetBanBySID != nil {
		if !errors.Is(errGetBanBySID, database.ErrNoResult) {
			slog.Error("Failed to get ban by steamid", slog.String("error", errGetBanBySID.Error()))

			return nil, discord.ErrCommandFailed
		}
	}

	oldBans, errOld := h.Query(ctx, QueryOpts{})
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

	var authorProfile person.Core

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

		banURL = link.Path(bans)
	}

	return CheckMessage(player, bans, banURL, authorProfile, oldBans), nil
}

func UnbanMessage(person person.Info) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed("User Unbanned Successfully")
	msgEmbed.Embed().
		SetColor(discord.ColourSuccess).
		SetImage(person.GetAvatar().Full()).
		SetURL(link.Path(person))
	msgEmbed.AddFieldsSteamID(person.GetSteamID())

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func CheckMessage(player person.Core, banPerson Ban, banURL string, author person.Info, oldBans []Ban) *discordgo.MessageEmbed {
	msgEmbed := discord.NewEmbed()

	title := player.Name

	if banPerson.BanID > 0 {
		if banPerson.BanType == bantype.Banned {
			title += " (BANNED)"
		} else if banPerson.BanType == bantype.NoComm {
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
			if oldBan.BanType == bantype.Banned {
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
		banned = banPerson.BanType == bantype.Banned
		muted = banPerson.BanType == bantype.NoComm
		banReason := banPerson.ReasonText

		if len(banReason) == 0 {
			// in case authorProfile ban without authorProfile reason ever makes its way here, we make sure
			// that Discord doesn't shit itself
			banReason = "none"
		}

		expiry = banPerson.ValidUntil
		createdAt = banPerson.CreatedOn.Format(time.RFC3339)

		msgEmbed.Embed().SetURL(banURL)
		msgEmbed.Embed().AddField("Reason", banReason)
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
		SetURL(link.Path(player)).
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
	if inBan.BanType == bantype.NoComm {
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

	if inBan.BanType == bantype.NoComm {
		msgEmbed.Embed().SetColor(discord.ColourWarn)
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func CreateResponse(banSteam Ban) *discordgo.MessageEmbed {
	var (
		title  string
		colour int
	)

	if banSteam.BanType == bantype.NoComm {
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
