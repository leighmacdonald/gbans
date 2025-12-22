package ban

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/datetime"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/sosodev/duration"
)

//go:embed ban_discord.tmpl
var templateBody []byte

type discordHandler struct {
	Bans

	persons person.Provider
	discord person.DiscordPersonProvider
}

func RegisterDiscordCommands(bot discord.Service, bans Bans, persons person.Provider, discordProv person.DiscordPersonProvider) {
	discord.MustRegisterTemplate(templateBody)

	handler := &discordHandler{Bans: bans, persons: persons, discord: discordProv}

	bot.MustRegisterPrefixHandler("ban_unban_button", handler.onUnbanButton)
	bot.MustRegisterPrefixHandler("report_reply_button", handler.onReportReplyButton)
	bot.MustRegisterPrefixHandler("report_reply_submit", handler.onReportReplySubmit)
	bot.MustRegisterPrefixHandler("ban_modal", handler.onBanResponse)
	bot.MustRegisterPrefixHandler("ban_create_button", handler.onBanButton)
	bot.MustRegisterPrefixHandler("mute_modal", handler.onBanResponse)

	bot.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "ban",
		Description:              "Create Steam / CIDR ban",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: ptr.To(discord.ModPerms),
	}, handler.createBanModal)

	bot.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "mute",
		Description:              "Mute a player",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: ptr.To(discord.ModPerms),
	}, handler.createMuteModal)

	bot.MustRegisterPrefixHandler("unban_resp", handler.onUnbanResponse)
	bot.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "unban",
		Description:              "Unban users",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: ptr.To(discord.ModPerms),
	}, handler.onUnban)

	bot.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "check",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: ptr.To(discord.ModPerms),
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

func createBanOpts() []discordgo.SelectMenuOption {
	return []discordgo.SelectMenuOption{
		{Label: reason.BotHost.String(), Value: strconv.Itoa(int(reason.BotHost))},
		{Label: reason.Cheating.String(), Value: strconv.Itoa(int(reason.Cheating))},
		{Label: reason.Custom.String(), Value: strconv.Itoa(int(reason.Custom))},
		{Label: reason.Evading.String(), Value: strconv.Itoa(int(reason.Evading))},
		{Label: reason.Exploiting.String(), Value: strconv.Itoa(int(reason.Exploiting))},
		{Label: reason.External.String(), Value: strconv.Itoa(int(reason.External))},
		{Label: reason.Harassment.String(), Value: strconv.Itoa(int(reason.Harassment))},
		{Label: reason.ItemDescriptions.String(), Value: strconv.Itoa(int(reason.ItemDescriptions))},
		{Label: reason.Language.String(), Value: strconv.Itoa(int(reason.Language))},
		{Label: reason.Profile.String(), Value: strconv.Itoa(int(reason.Profile))},
		{Label: reason.Racism.String(), Value: strconv.Itoa(int(reason.Racism))},
		{Label: reason.Spam.String(), Value: strconv.Itoa(int(reason.Spam))},
		{Label: reason.Username.String(), Value: strconv.Itoa(int(reason.Username))},
		{Label: reason.WarningsExceeded.String(), Value: strconv.Itoa(int(reason.WarningsExceeded))},
	}
}

func createDurationOpts() []discordgo.SelectMenuOption {
	return []discordgo.SelectMenuOption{
		{Label: "15 Mins", Value: "PT15M"},
		{Label: "6 Hours", Value: "PT6H"},
		{Label: "12 Hours", Value: "PT12H"},
		{Label: "1 Day", Value: "P1D"},
		{Label: "2 Days", Value: "P2D"},
		{Label: "3 Days", Value: "P3D"},
		{Label: "1 Week", Value: "P1W"},
		{Label: "2 Weeks", Value: "P2W"},
		{Label: "1 Month", Value: "P1M"},
		{Label: "3 Months", Value: "P3M"},
		{Label: "6 Months", Value: "P6M"},
		{Label: "1 Year", Value: "P1Y"},
		{Label: "Permanent", Value: "P0"},
		{Label: "Custom", Value: "custom"},
	}
}

type banModalOpts struct {
	TargetID steamid.SteamID    `id:"1"`
	CIDR     *netip.Prefix      `id:"2"`
	Reason   reason.Reason      `id:"3"`
	Duration *duration.Duration `id:"4"`
	Note     string             `id:"5"`
}

func (h discordHandler) createBanModal(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	return h.showBan(ctx, session, interaction, "Ban Player", "ban_modal_"+interaction.Member.User.ID, "")
}

func (h discordHandler) createMuteModal(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	return h.showBan(ctx, session, interaction, "Mute Player", "mute_modal_"+interaction.Member.User.ID, "")
}

func (h discordHandler) onBanButton(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	steamID, errID := discord.CustomIDInt64(interaction.MessageComponentData().CustomID)
	if errID != nil {
		return errID
	}

	sid := steamid.New(steamID)

	return h.showBan(ctx, session, interaction, "Ban Player", "ban_modal_"+sid.String(), sid.String())
}

func (h discordHandler) showBan(_ context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, title string, prefix string, sid string) error {
	minItems := 1

	var sidComp discordgo.MessageComponent
	if sid == "" {
		sidComp = discord.ModalInputRowRequired(discord.IDSteamID, "steamid",
			"SteamID or Profile URL", "76561197960542812", sid, 0, 64)
	} else {
		sidComp = discordgo.TextDisplay{
			Content: sid,
		}
	}

	return discord.RespondModal(session, interaction, prefix, title,
		sidComp,
		discord.SelectOption(discord.IDReason, "Reason", "reason", "Select a reason", minItems, 1, createBanOpts()),
		discord.ModalInputRow(discord.IDCIDR, "cidr", "IP/CIDR Ban", "1.2.3.4/32, 100.100.100.0/24", "", 0, 20),
		discord.SelectOption(discord.IDDuration, "Duration", "duration", "Select a duration", minItems, 1, createDurationOpts()),
		discord.ModalInputRows(discord.IDNotes, "notes", "Notes", "", "", 0, 0),
	)
}

func (h discordHandler) onBanResponse(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	if err := discord.AckInteraction(session, interaction); err != nil {
		return err
	}

	author, errAuthor := h.discord.GetPersonByDiscordID(ctx, interaction.Member.User.ID)
	if errAuthor != nil {
		return errAuthor
	}

	banType := bantype.Banned
	if !strings.HasPrefix(interaction.ModalSubmitData().CustomID, "ban") {
		banType = bantype.NoComm
	}

	values, errValues := discord.Bind[banModalOpts](ctx, interaction.ModalSubmitData().Components)
	if errValues != nil {
		return errValues
	}

	banOpts := Opts{
		Origin:     Bot,
		SourceID:   author.GetSteamID(),
		BanType:    banType,
		ReasonText: "",
		TargetID:   values.TargetID,
		Reason:     values.Reason,
		Duration:   values.Duration,
		Note:       values.Note,
	}

	if values.CIDR != nil {
		prefix := values.CIDR.String()
		banOpts.CIDR = &prefix
	}

	createdBan, errBan := h.Create(ctx, banOpts)
	if errBan != nil {
		if errors.Is(errBan, database.ErrDuplicate) {
			return ErrDuplicateBan
		}
		slog.Error("Failed to create ban", slog.String("error", errBan.Error()))

		return discord.ErrCommandFailed
	}

	content, errContent := discord.RenderTemplate("ban_success", templateBody)
	if errContent != nil {
		return errContent
	}

	return discord.RespondUpdate(session, interaction,
		discord.BodyColouredText(discord.ColourSuccess, content),
		discord.Buttons(
			discord.Button(discordgo.SuccessButton, "🗑️ Unban", fmt.Sprintf("ban_unban_button_resp_%d", createdBan.BanID)),
			discord.Button(discordgo.SecondaryButton, "🔨 Edit", fmt.Sprintf("ban_edit_button_resp_%d", createdBan.BanID)),
			discord.Link("🔗 Link", link.Path(createdBan))))
}

func (h discordHandler) onUnbanButton(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	banID, errID := discord.CustomIDInt64(interaction.MessageComponentData().CustomID)
	if errID != nil {
		return errID
	}
	ban, errBan := h.QueryOne(ctx, QueryOpts{BanID: banID})
	if errBan != nil {
		return errBan
	}

	return h.showUnban(ctx, session, interaction, ban.TargetID.String())
}

func (h discordHandler) onUnban(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	return h.showUnban(ctx, session, interaction, "")
}

func (h discordHandler) showUnban(_ context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, steamID string) error {
	return discord.RespondModal(session, interaction,
		"unban_resp", "Unban Player",
		discord.ModalInputRowRequired(discord.IDSteamID, "steamid", "SteamID or Profile URL",
			"76561197960542812", steamID, 0, 64),
		discord.ModalInputRowRequired(discord.IDNotes, "unban_reason", "Reason",
			"Finally took a shower", "", 0, 2000),
	)
}

type UnbanRequestModal struct {
	TargetID    steamid.SteamID `id:"1"`
	UnbanReason string          `id:"6"`
}

func (h discordHandler) onUnbanResponse(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	if err := discord.AckInteraction(session, interaction); err != nil {
		return err
	}

	req, errReq := discord.Bind[UnbanRequestModal](ctx, interaction.ModalSubmitData().Components)
	if errReq != nil {
		return errReq
	}

	author, errAuthor := h.discord.GetPersonByDiscordID(ctx, interaction.Member.User.ID)
	if errAuthor != nil {
		return errAuthor
	}

	exists, errUnban := h.Unban(ctx, req.TargetID, req.UnbanReason, author)
	if errUnban != nil {
		return errUnban
	}
	if !exists {
		return ErrBanDoesNotExist
	}

	return discord.RespondUpdate(session, interaction,
		discord.BodyColouredText(discord.ColourSuccess, "Unban successful"))
}

type checkView struct {
	Author  person.Info
	Player  person.Info
	SteamID string
	Ban     Ban
	Old     []Ban
}

func (h discordHandler) onCheck(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, //nolint:maintidx
) error {
	if err := discord.AckInteraction(session, interaction); err != nil {
		return err
	}

	opts := discord.OptionMap(interaction.ApplicationCommandData().Options)

	sid, errResolveSID := steamid.Resolve(ctx, opts[discord.OptUserIdentifier].StringValue())
	if errResolveSID != nil || !sid.Valid() {
		return steamid.ErrInvalidSID
	}

	player, errGetPlayer := h.persons.GetOrCreatePersonBySteamID(ctx, sid)
	if errGetPlayer != nil {
		return discord.ErrCommandFailed
	}

	bans, errOld := h.Query(ctx, QueryOpts{TargetID: sid, Deleted: true})
	if errOld != nil {
		if !errors.Is(errOld, database.ErrNoResult) {
			slog.Error("Failed to fetch old bans", slog.String("error", errOld.Error()))
		}
	}
	var (
		author    person.Info
		activeBan Ban
		expired   []Ban
	)
	for _, ban := range bans {
		if !ban.Expired() {
			activeBan = ban
			autheur, errAuthor := h.persons.GetOrCreatePersonBySteamID(ctx, activeBan.SourceID)
			if errAuthor != nil {
				return errAuthor
			}
			author = autheur
		} else {
			expired = append(expired, ban)
		}
	}

	content, errContent := discord.RenderTemplate("check", checkView{
		Author:  author,
		Player:  player,
		Ban:     activeBan,
		Old:     expired,
		SteamID: player.SteamID.String(),
	})
	if errContent != nil {
		slog.Error("Failed to render check body", slog.String("error", errContent.Error()))
		content = "Error rendering response :("
	}

	colour := discord.ColourSuccess
	if activeBan.BanID > 0 {
		colour = discord.ColourError
	}

	var btn []discordgo.MessageComponent
	if activeBan.BanID > 0 {
		btn = append(btn,
			discord.Button(discordgo.SuccessButton, "🗑️ Unban", fmt.Sprintf("ban_unban_button_resp_%d", activeBan.BanID)),
			discord.Link("🔎 View", link.Path(activeBan)))
	} else {
		btn = append(btn,
			discord.Button(discordgo.DangerButton, "🔨 Ban", fmt.Sprintf("ban_create_button_resp_%d", player.SteamID.Int64())),
		)
	}

	return discord.RespondUpdate(session, interaction,
		discord.Heading("Player Check: %s", player.GetName()),
		discord.BodyColour(
			colour,
			discordgo.MediaGallery{
				Items: []discordgo.MediaGalleryItem{
					{
						Media: discordgo.UnfurledMediaItem{URL: player.GetAvatar().Full()},
					},
				},
			},
			discordgo.TextDisplay{Content: content}),
		discord.Buttons(append(btn,
			discord.Link("🔗 Link", link.Path(player)),
			discord.Link("🔧 Steam", "https://steamcommunity.com/profiles/"+player.SteamID.String()))...))
}

func unbanMessage(person person.Info, reason string) *discordgo.MessageSend {
	return discord.NewMessage(
		discord.Heading("User Unbanned Successfully: %s", person.GetName()),
		discord.BodyColour(discord.ColourSuccess, discordgo.MediaGallery{
			Items: []discordgo.MediaGalleryItem{
				{
					Media: discordgo.UnfurledMediaItem{URL: person.GetAvatar().Full()},
				},
			},
		},
			discordgo.TextDisplay{Content: reason}),
		discord.Buttons(discord.Link("🔗 Link", link.Path(person))))
}

type banResponseView struct {
	Ban           Ban
	Player        person.Info
	Author        person.Info
	SteamIDAuthor string
	SteamID       string
	ExpIn         string
	ExpAt         string
}

func createBanResponse(ban Ban, author person.Info, player person.Info) *discordgo.MessageSend {
	expIn := Permanent
	expAt := Permanent
	if ban.ValidUntil.Year()-time.Now().Year() < 5 {
		expIn = datetime.FmtDuration(ban.ValidUntil)
		expAt = datetime.FmtTimeShort(ban.ValidUntil)
	}

	return discord.NewMessage(
		discord.BodyTextWithThumbnail(discord.ColourError,
			discord.PlayerThumbnail(player),
			"ban_response",
			banResponseView{
				Ban:           ban,
				Player:        player,
				Author:        author,
				SteamIDAuthor: author.GetSteamIDString(),
				SteamID:       player.GetSteamIDString(),
				ExpIn:         expIn,
				ExpAt:         expAt,
			}),
		discord.Buttons(
			discord.Button(discordgo.SuccessButton, "🗑️ Unban", fmt.Sprintf("ban_unban_button_resp_%d", ban.BanID)),
			discord.Button(discordgo.SecondaryButton, "🔨 Edit", fmt.Sprintf("ban_edit_button_resp_%d", ban.BanID)),
			discord.Link("🔎 View", link.Path(ban)),
			discord.Link("🌐 Steam", "https://steamcommunity.com/profiles/"+ban.TargetID.String())))
}

type deleteReportMessageView struct {
	Existing ReportMessage
	Person   person.Info
}

func DeleteReportMessage(existing ReportMessage, person person.Info) *discordgo.MessageSend {
	content, errContent := discord.RenderTemplate("report_message_deleted", deleteReportMessageView{
		Existing: existing,
		Person:   person,
	})
	if errContent != nil {
		return nil
	}

	return discord.NewMessage(
		discord.Heading("Report Message Deleted"),
		discord.BodyColouredText(discord.ColourWarn, content),
		discord.Buttons(discord.Link("🔎 View", link.Path(existing))),
	)
}

func EditReportMessageResponse(body string, oldBody string, link string, _ person.Info, _ string) *discordgo.MessageSend {
	return discord.NewMessage(
		discord.Heading("Report Message Edited"),
		discord.BodyColouredText(discord.ColourWarn, oldBody),
		discord.BodyColouredText(discord.ColourSuccess, body),
		discord.Buttons(discord.Link("🔎 View", link)))
}

type reportStatsView struct {
	New         int
	TotalOpen   int
	TotalClosed int
	Open1Day    int
	Open3Days   int
	Open1Week   int
}

func ReportStatsMessage(meta ReportMeta, _ string) *discordgo.MessageSend {
	colour := discord.ColourSuccess
	if meta.OpenWeek > 0 {
		colour = discord.ColourError
	} else if meta.Open3Days > 0 {
		colour = discord.ColourWarn
	}

	body, errBody := discord.RenderTemplate("report_stats", reportStatsView{
		New:         meta.Open1Day,
		TotalOpen:   meta.TotalOpen,
		TotalClosed: meta.TotalClosed,
		Open1Day:    meta.Open1Day,
		Open3Days:   meta.Open3Days,
		Open1Week:   meta.OpenWeek,
	})
	if errBody != nil {
		slog.Error("Failed to render report stats", slog.String("error", errBody.Error()))
	}

	return discord.NewMessage(
		discord.Heading("Report Stats"),
		discord.BodyColouredText(colour, body),
		discord.Buttons(discord.Link("Reports", link.Raw("/admin/reports"))))
}
