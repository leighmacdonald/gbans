package ban

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/domain/ban"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/datetime"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	reasonCollection = []ban.Reason{
		ban.External, ban.Cheating, ban.Racism, ban.Harassment, ban.Exploiting,
		ban.WarningsExceeded, ban.Spam, ban.Language, ban.Profile, ban.ItemDescriptions,
		ban.BotHost, ban.Evading, ban.Username, ban.Custom,
	}

	reasons = []*discordgo.ApplicationCommandOptionChoice{
		&discordgo.ApplicationCommandOptionChoice{Name: ban.External.String(), Value: ban.External},
		&discordgo.ApplicationCommandOptionChoice{Name: ban.Cheating.String(), Value: ban.Cheating},
		&discordgo.ApplicationCommandOptionChoice{Name: ban.Racism.String(), Value: ban.Racism},
		&discordgo.ApplicationCommandOptionChoice{Name: ban.Harassment.String(), Value: ban.Harassment},
		&discordgo.ApplicationCommandOptionChoice{Name: ban.Exploiting.String(), Value: ban.Exploiting},
		&discordgo.ApplicationCommandOptionChoice{Name: ban.WarningsExceeded.String(), Value: ban.WarningsExceeded},
		&discordgo.ApplicationCommandOptionChoice{Name: ban.Spam.String(), Value: ban.Spam},
		&discordgo.ApplicationCommandOptionChoice{Name: ban.Language.String(), Value: ban.Language},
		&discordgo.ApplicationCommandOptionChoice{Name: ban.Profile.String(), Value: ban.Profile},
		&discordgo.ApplicationCommandOptionChoice{Name: ban.ItemDescriptions.String(), Value: ban.ItemDescriptions},
		&discordgo.ApplicationCommandOptionChoice{Name: ban.BotHost.String(), Value: ban.BotHost},
		&discordgo.ApplicationCommandOptionChoice{Name: ban.Evading.String(), Value: ban.Evading},
		&discordgo.ApplicationCommandOptionChoice{Name: ban.Username.String(), Value: ban.Username},
		&discordgo.ApplicationCommandOptionChoice{Name: ban.Custom.String(), Value: ban.Custom},
	}

	optBanReason = &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        OptBanReason,
		Description: "Reason for the ban/mute",
		Required:    true,
		Choices:     reasons,
	}
)

func makeOnMute() func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate,
	) (*discordgo.MessageEmbed, error) {
		opts := OptionMap(interaction.ApplicationCommandData().Options)

		playerID, errPlayerID := steamid.Resolve(ctx, opts.String(OptUserIdentifier))
		if errPlayerID != nil || !playerID.Valid() {
			return nil, domain.ErrInvalidSID
		}

		reasonValueOpt, ok := opts[OptBanReason]
		if !ok {
			return nil, domain.ErrReasonInvalid
		}

		author, errAuthor := h.getDiscordAuthor(ctx, interaction)
		if errAuthor != nil {
			return nil, errAuthor
		}

		banSteam, errBan := h.bans.Ban(ctx, author.ToUserProfile(), ban.Bot, ban.BanOpts{
			SourceIDField: domain.SourceIDField{SourceID: author.SteamID.String()},
			TargetIDField: domain.TargetIDField{TargetID: opts.String(OptUserIdentifier)},
			Duration:      opts[OptDuration].StringValue(),
			BanType:       ban.NoComm,
			Reason:        ban.Reason(reasonValueOpt.IntValue()),
			ReasonText:    "",
			Note:          opts[OptNote].StringValue(),
		})
		if errBan != nil {
			return nil, errBan
		}

		return MuteMessage(banSteam), nil
	}
}

// onBanSteam !ban <id> <duration> [reason].
func onBanSteam(ctx context.Context, _ *discordgo.Session,
	interaction *discordgo.InteractionCreate,
) (*discordgo.MessageEmbed, error) {
	opts := OptionMap(interaction.ApplicationCommandData().Options[0].Options)

	author, errAuthor := h.getDiscordAuthor(ctx, interaction)
	if errAuthor != nil {
		return nil, errAuthor
	}

	banSteam, errBan := h.bans.Ban(ctx, author.ToUserProfile(), ban.Bot, ban.BanOpts{
		SourceIDField: domain.SourceIDField{SourceID: author.SteamID.String()},
		TargetIDField: domain.TargetIDField{TargetID: opts[OptUserIdentifier].StringValue()},
		Duration:      opts[OptDuration].StringValue(),
		BanType:       ban.Banned,
		Reason:        ban.Reason(opts[OptBanReason].IntValue()),
		ReasonText:    "",
		Note:          opts[OptNote].StringValue(),
	})
	if errBan != nil {
		if errors.Is(errBan, database.ErrDuplicate) {
			return nil, domain.ErrDuplicateBan
		}

		return nil, ErrCommandFailed
	}

	return message.BanSteamResponse(banSteam), nil
}

func onUnbanSteam(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	opts := OptionMap(interaction.ApplicationCommandData().Options[0].Options)
	reason := opts[OptUnbanReason].StringValue()

	author, err := h.getDiscordAuthor(ctx, interaction)
	if err != nil {
		return nil, err
	}

	steamID, errResolveSID := steamid.Resolve(ctx, opts[OptUserIdentifier].StringValue())
	if errResolveSID != nil || !steamID.Valid() {
		return nil, domain.ErrInvalidSID
	}

	found, errUnban := h.bans.Unban(ctx, steamID, reason, author.ToUserProfile())
	if errUnban != nil {
		return nil, errUnban
	}

	if !found {
		return nil, ban.ErrBanDoesNotExist
	}

	user, errUser := h.persons.GetPersonBySteamID(ctx, nil, steamID)
	if errUser != nil {
		slog.Warn("Could not fetch unbanned Person", slog.String("steam_id", steamID.String()), log.ErrAttr(errUser))
	}

	return UnbanMessage(h.config, user), nil
}

func makeOnBan() func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		name := interaction.ApplicationCommandData().Options[0].Name
		switch name {
		case "steam":
			return onBanSteam(ctx, session, interaction)
		default:
			return nil, ErrCommandFailed
		}
	}
}

func makeOnUnban() func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) { //nolint:maintidx
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) {
		switch interaction.ApplicationCommandData().Options[0].Name {
		case "steam":
			return onUnbanSteam(ctx, session, interaction)
		default:
			return nil, ErrCommandFailed
		}
	}
}

func makeOnCheck() func(_ context.Context, _ *discordgo.Session, _ *discordgo.InteractionCreate) (*discordgo.MessageEmbed, error) { //nolint:maintidx
	return func(ctx context.Context, _ *discordgo.Session, interaction *discordgo.InteractionCreate, //nolint:maintidx
	) (*discordgo.MessageEmbed, error) {
		opts := OptionMap(interaction.ApplicationCommandData().Options)
		sid, errResolveSID := steamid.Resolve(ctx, opts[OptUserIdentifier].StringValue())

		if errResolveSID != nil || !sid.Valid() {
			return nil, domain.ErrInvalidSID
		}

		player, errGetPlayer := h.persons.GetOrCreatePersonBySteamID(ctx, nil, sid)
		if errGetPlayer != nil {
			return nil, ErrCommandFailed
		}

		bans, errGetBanBySID := h.bans.Query(ctx, ban.QueryOpts{EvadeOk: true, TargetID: sid})
		if errGetBanBySID != nil {
			if !errors.Is(errGetBanBySID, database.ErrNoResult) {
				slog.Error("Failed to get ban by steamid", log.ErrAttr(errGetBanBySID))

				return nil, ErrCommandFailed
			}
		}

		oldBans, errOld := h.bans.Query(ctx, ban.QueryOpts{})
		if errOld != nil {
			if !errors.Is(errOld, database.ErrNoResult) {
				slog.Error("Failed to fetch old bans", log.ErrAttr(errOld))
			}
		}

		bannedNets, errGetBanNet := h.bans.GetByAddress(ctx, player.IPAddr)
		if errGetBanNet != nil {
			if !errors.Is(errGetBanNet, database.ErrNoResult) {
				slog.Error("Failed to get ban nets by addr", log.ErrAttr(errGetBanNet))

				return nil, ErrCommandFailed
			}
		}

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

		return message.CheckMessage(player, ban, banURL, authorProfile, oldBans, network.Location, network.Proxy, logData), nil
	}
}

func UnbanMessage(cu *config.ConfigUsecase, person domain.PersonInfo) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("User Unbanned Successfully")
	msgEmbed.Embed().
		SetColor(ColourSuccess).
		SetImage(person.GetAvatar().Full()).
		SetURL(cu.ExtURL(person))
	msgEmbed.AddFieldsSteamID(person.GetSteamID())

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func CheckMessage(player person.Person, banPerson ban.BannedPerson, banURL string, author domain.PersonInfo,
	oldBans []ban.BannedPerson, location network.NetworkLocation,
	proxy network.NetworkProxy, logData thirdparty.LogsTFPlayerSummary,
) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed()

	title := player.PersonaName

	if banPerson.BanID > 0 {
		switch banPerson.BanType {
		case ban.Banned:
			title += " (BANNED)"
		case ban.NoComm:
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
			if oldBan.BanType == ban.Banned {
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
		banned = banPerson.BanType == ban.Banned
		muted = banPerson.BanType == ban.NoComm
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
	return NewEmbed("IP ban created successfully").Embed().
		SetColor(ColourSuccess).
		Truncate().
		MessageEmbed
}

func ReportStatusChangeMessage(report ban.ReportWithAuthor, fromStatus ban.ReportStatus, link string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed(
		"Report status changed",
		fmt.Sprintf("Changed from %s to %s", fromStatus.String(), report.ReportStatus.String()),
		link)

	msgEmbed.Embed().
		AddField("report_id", strconv.FormatInt(report.ReportID, 10))
	msgEmbed.AddAuthorPersonInfo(report.Author, link)
	msgEmbed.AddTargetPerson(report.Subject)

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func MuteMessage(banSteam ban.BannedPerson) *discordgo.MessageEmbed {
	embed := NewEmbed("Player muted successfully")
	embed.AddFieldsSteamID(banSteam.TargetID)

	return embed.Embed().SetColor(ColourSuccess).Truncate().MessageEmbed
}

func KickPlayerEmbed(target domain.PersonInfo) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("User Kicked Successfully")
	msgEmbed.Embed().SetColor(ColourSuccess)

	return msgEmbed.AddTargetPerson(target).Embed().MessageEmbed
}

func KickPlayerOnConnectEmbed(steamID steamid.SteamID, name string, target domain.PersonInfo, banSource string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("User Kicked Successfully")
	msgEmbed.Embed().SetColor(ColourWarn)
	msgEmbed.AddTargetPerson(target)
	msgEmbed.Embed().
		AddField("Connecting As", name).
		AddField("Ban Source", banSource)

	return msgEmbed.AddFieldsSteamID(steamID).Embed().MessageEmbed
}

func SilenceEmbed(target domain.PersonInfo) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("User Silenced Successfully")
	msgEmbed.Embed().SetColor(ColourSuccess)

	return msgEmbed.AddTargetPerson(target).Embed().MessageEmbed
}

func BanExpiresMessage(inBan ban.Ban, person domain.PersonInfo, banURL string) *discordgo.MessageEmbed {
	banType := "Ban"
	if inBan.BanType == ban.NoComm {
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

	if inBan.BanType == ban.NoComm {
		msgEmbed.Embed().SetColor(ColourWarn)
	}

	return msgEmbed.Embed().Truncate().MessageEmbed
}

func BanSteamResponse(banSteam ban.BannedPerson) *discordgo.MessageEmbed {
	var (
		title  string
		colour int
	)

	if banSteam.BanType == ban.NoComm {
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

	msgEmbed.Embed().SetAuthor(banSteam.SourcePersonaname, domain.NewAvatar(banSteam.SourceAvatarhash).Full(), "https://steamcommunity.com/profiles/"+banSteam.SourceID.String())

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
		msgEmbed.emb.Description = banSteam.Note
	}

	msgEmbed.emb.URL = "https://steamcommunity.com/profiles/" + banSteam.TargetID.String()

	return msgEmbed.Embed().MessageEmbed
}

func DeleteReportMessage(existing ban.ReportMessage, user domain.PersonInfo, userURL string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("User report message deleted")
	msgEmbed.
		Embed().
		SetDescription(existing.MessageMD).
		SetColor(ColourWarn)

	return msgEmbed.AddAuthorPersonInfo(user, userURL).Embed().Truncate().MessageEmbed
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

func EditAppealMessage(existing ban.BanAppealMessage, body string, author domain.PersonInfo, authorURL string) *discordgo.MessageEmbed {
	msgEmbed := NewEmbed("Ban appeal message edited")
	msgEmbed.
		Embed().
		SetDescription(stringutil.DiffString(existing.MessageMD, body)).
		SetColor(ColourWarn)

	return msgEmbed.
		AddAuthorPersonInfo(author, authorURL).
		Embed().
		Truncate().
		MessageEmbed
}

func DeleteAppealMessage(existing *ban.BanAppealMessage, user domain.PersonInfo, userURL string) *discordgo.MessageEmbed {
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

func EditBanAppealStatusMessage() *discordgo.MessageEmbed {
	return NewEmbed("Ban state updates").Embed().MessageEmbed
}

func ReportStatsMessage(meta ban.ReportMeta, url string) *discordgo.MessageEmbed {
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
