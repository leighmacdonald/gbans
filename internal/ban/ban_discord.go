package ban

import (
	"context"
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

type discordHandler struct {
	Bans

	persons person.Provider
	discord person.DiscordPersonProvider
}

func RegisterDiscordCommands(bot discord.Service, bans Bans, persons person.Provider, discordProv person.DiscordPersonProvider) {
	handler := &discordHandler{Bans: bans, persons: persons, discord: discordProv}

	bot.MustRegisterPrefixHandler("ban_unban_button", handler.onUnbanButton)
	bot.MustRegisterPrefixHandler("report_reply_button", handler.onReportReplyButton)
	bot.MustRegisterPrefixHandler("report_reply_submit", handler.onReportReplySubmit)
	bot.MustRegisterPrefixHandler("ban_modal", handler.onBanResponse)
	bot.MustRegisterPrefixHandler("mute_modal", handler.onBanResponse)

	bot.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "ban",
		Description:              "Create Steam / CIDR ban",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: &discord.ModPerms,
	}, handler.createBanMuteModal(true))

	bot.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "mute",
		Description:              "Mute a player",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: &discord.ModPerms,
	}, handler.createBanMuteModal(false))

	bot.MustRegisterPrefixHandler("unban_resp", handler.onUnbanResponse)
	bot.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
		Name:                     "unban",
		Description:              "Unban users",
		Contexts:                 &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild},
		DefaultMemberPermissions: &discord.ModPerms,
	}, handler.onUnban)

	bot.MustRegisterCommandHandler(&discordgo.ApplicationCommand{
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
	}, handler.onCheck)
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

func createBanOpts() []discordgo.SelectMenuOption {
	banOpts := make([]discordgo.SelectMenuOption, len(reason.Reasons))
	for index, op := range reason.Reasons {
		banOpts[index] = discordgo.SelectMenuOption{
			Label: op.String(),
			Value: strconv.Itoa(int(op)),
		}
	}

	return banOpts
}

func createDurationOpts() []discordgo.SelectMenuOption {
	var index int
	durationOpts := make([]discordgo.SelectMenuOption, len(durationMap))
	for label, value := range durationMap {
		durationOpts[index] = discordgo.SelectMenuOption{
			Label: label,
			Value: value,
		}
		index++
	}

	return durationOpts
}

type banModalOpts struct {
	TargetID steamid.SteamID    `id:"1"`
	CIDR     *netip.Prefix      `id:"2"`
	Reason   reason.Reason      `id:"3"`
	Duration *duration.Duration `id:"4"`
	Note     string             `id:"5"`
}

func (h discordHandler) createBanMuteModal(isBan bool) func(_ context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	title := "Ban Player"
	prefix := "ban"
	if !isBan {
		title = "Mute Player"
		prefix = "mute"
	}

	return func(_ context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
		minItems := 1

		return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: prefix + "_modal_" + interaction.Interaction.Member.User.ID,
				Title:    title,
				Flags:    discordgo.MessageFlagsIsComponentsV2 | discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								ID:          discord.IDSteamID,
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
							ID:          discord.IDReason,
							CustomID:    "reason",
							Placeholder: "Select a reason",
							MaxValues:   1,
							MinValues:   &minItems,
							MenuType:    discordgo.StringSelectMenu,
							Options:     createBanOpts(),
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								ID:          discord.IDCIDR,
								CustomID:    "cidr",
								Label:       "IP/CIDR Ban",
								Style:       discordgo.TextInputShort,
								Placeholder: "1.2.3.4/32, 100.100.100.0/24",
								Required:    false,
								MaxLength:   20,
								MinLength:   0,
							},
						},
					},
					discordgo.Label{
						Label: "Duration",
						Component: discordgo.SelectMenu{
							ID:          discord.IDDuration,
							CustomID:    "duration",
							Placeholder: "Select a duration",
							MaxValues:   1,
							MinValues:   &minItems,
							MenuType:    discordgo.StringSelectMenu,
							Options:     createDurationOpts(),
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								ID:          discord.IDNotes,
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
		})
	}
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

	banOpts := Opts{
		Origin:     Bot,
		SourceID:   author.GetSteamID(),
		BanType:    banType,
		ReasonText: "",
	}

	values, errValues := discord.Bind[banModalOpts](ctx, interaction.ModalSubmitData().Components)
	if errValues != nil {
		return errValues
	}

	banOpts.TargetID = values.TargetID
	banOpts.Reason = values.Reason
	banOpts.Duration = values.Duration
	banOpts.Note = values.Note
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

	content := fmt.Sprintf("Ban successful [View](%s)", link.Path(createdBan))
	if !strings.HasPrefix(interaction.ModalSubmitData().CustomID, "ban") {
		content = fmt.Sprintf("Mute successful [View](%s)", link.Path(createdBan))
	}

	return discord.RespondInteraction(session, interaction,
		discordgo.Container{
			AccentColor: ptr.To(discord.ColourSuccess),
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: content},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "🗑️ Unban",
					CustomID: fmt.Sprintf("ban_unban_button_resp_%d", createdBan.BanID),
					Style:    discordgo.SuccessButton,
				},
				discordgo.Button{
					Label:    "🔨 Edit",
					CustomID: fmt.Sprintf("ban_edit_button_resp_%d", createdBan.BanID),
					Style:    discordgo.SecondaryButton,
				},
				discordgo.Button{
					Label: "🔗 Link",
					URL:   link.Path(createdBan),
					Style: discordgo.LinkButton,
				},
			},
		})
}

func (h discordHandler) onUnbanButton(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	banID, errID := discord.CustomIDInt64(interaction.MessageComponentData().CustomID)
	if errID != nil {
		return errID
	}
	ban, errBan := h.Bans.QueryOne(ctx, QueryOpts{BanID: banID})
	if errBan != nil {
		return errBan
	}

	return h.showUnban(ctx, session, interaction, ban.TargetID.String())
}

func (h discordHandler) onUnban(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
	return h.showUnban(ctx, session, interaction, "")
}

func (h discordHandler) showUnban(_ context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate, steamID string) error {
	var component discordgo.MessageComponent
	if steamID == "" {
		component = discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					ID:          discord.IDSteamID,
					CustomID:    "steamid",
					Label:       "SteamID or Profile URL",
					Style:       discordgo.TextInputShort,
					Placeholder: "76561197960542812",
					Required:    true,
					MaxLength:   64,
					MinLength:   0,
				},
			},
		}
	} else {
		component = discordgo.TextDisplay{
			Content: steamID,
		}
	}

	return session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "unban_resp_" + interaction.Interaction.Member.User.ID,
			Title:    "Unban Player",
			Flags:    discordgo.MessageFlagsIsComponentsV2 | discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				component,
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							ID:          discord.IDNotes,
							CustomID:    "unban_reason",
							Label:       "Reason",
							Style:       discordgo.TextInputParagraph,
							Placeholder: "Finally took a shower",
							Required:    true,
							MaxLength:   2000,
							MinLength:   0,
						},
					},
				},
			},
		},
	})
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

	return discord.RespondInteraction(session, interaction,
		discordgo.Container{
			AccentColor: ptr.To(discord.ColourSuccess),
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{
					Content: "Unban successful",
				},
			},
		})
}

const checkTemplate = `# {{ .Player.GetName }}
{{if gt .Ban.BanID 0 }}
### {{if eq .Ban.BanType 2}}Ban{{ else }}Mute{{ end }} #{{ .Ban.BanID }}
Reason: **{{ .Ban.Reason.String }}**
Expires: **{{ .Ban.ValidUntil.Format "2006-01-02 15:04:05" }}**
Remaining: **123**
Evade Ok: {{ .Ban.EvadeOk }}
Author: {{ if ne .Author.GetDiscordID "" }}<@{{ .Author.GetDiscordID }}>{{else}}**{{ .Author.GetName }}**{{ end }}
Name: {{ .Ban.Name }}
Vac Bans: {{ .Player.VacBans }}
Game Bans: {{ .Player.GameBans }}
Notes: {{ .Ban.Note }}
{{else}}
No Ban Found!
{{ end }}
### History
{{ range .Old }}
- {{ . }}
{{ end}}`

type checkContext struct {
	Author  person.Core
	Player  person.Core
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

	activeBan, errGetBanBySID := h.QueryOne(ctx, QueryOpts{EvadeOk: true, TargetID: sid})
	if errGetBanBySID != nil {
		if !errors.Is(errGetBanBySID, database.ErrNoResult) {
			slog.Error("Failed to get ban by steamid", slog.String("error", errGetBanBySID.Error()))

			return discord.ErrCommandFailed
		}
	}

	var author person.Core
	if activeBan.BanID > 0 {
		autheur, errAuthor := h.persons.GetOrCreatePersonBySteamID(ctx, activeBan.SourceID)
		if errAuthor != nil {
			return errAuthor
		}
		author = autheur
	}

	oldBans, errOld := h.Query(ctx, QueryOpts{TargetID: sid})
	if errOld != nil {
		if !errors.Is(errOld, database.ErrNoResult) {
			slog.Error("Failed to fetch old bans", slog.String("error", errOld.Error()))
		}
	}

	content, errContent := discord.Render("check", checkTemplate, checkContext{
		Author:  author,
		Player:  player,
		Ban:     activeBan,
		Old:     oldBans,
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
			discordgo.Button{
				Label:    "🗑️ Unban",
				CustomID: fmt.Sprintf("ban_unban_button_resp_%d", activeBan.BanID),
				Style:    discordgo.SuccessButton,
			},
			discordgo.Button{
				Label: "🔎 View",
				URL:   link.Path(activeBan),
				Style: discordgo.LinkButton,
			})
	} else {
		btn = append(btn, discordgo.Button{
			Label:    "🔨 Ban",
			CustomID: fmt.Sprintf("ban_create_button_resp_%d", player.SteamID.Int64()),
			Style:    discordgo.DangerButton,
		})
	}

	return discord.RespondInteraction(session, interaction,
		discordgo.Container{
			AccentColor: &colour,
			Components: []discordgo.MessageComponent{
				discordgo.MediaGallery{
					Items: []discordgo.MediaGalleryItem{
						{
							Media: discordgo.UnfurledMediaItem{URL: player.GetAvatar().Full()},
						},
					},
				},
				discordgo.TextDisplay{Content: content},
			},
		},
		discordgo.ActionsRow{
			Components: append(btn, discordgo.Button{
				Label: "🔗 Link",
				URL:   link.Path(player),
				Style: discordgo.LinkButton,
			}),
		})
}

func UnbanMessage(person person.Info) *discordgo.MessageSend {
	return discord.NewMessageSend(discordgo.Container{
		AccentColor: ptr.To(discord.ColourSuccess),
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: "# User Unbanned Successfully"},
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label: "🔗 Link",
						URL:   link.Path(person),
						Style: discordgo.LinkButton,
					},
				},
			},
		},
	})
}

func createBanResponse(ban Ban, player person.Core) *discordgo.MessageSend {
	var title string
	if ban.BanType == bantype.NoComm {
		title = fmt.Sprintf("User Muted (#%d)", ban.BanID)
	} else {
		title = fmt.Sprintf("User Banned (#%d)", ban.BanID)
	}

	expIn := Permanent
	expAt := Permanent

	if ban.ValidUntil.Year()-time.Now().Year() < 5 {
		expIn = datetime.FmtDuration(ban.ValidUntil)
		expAt = datetime.FmtTimeShort(ban.ValidUntil)
	}
	// TODO use template
	msgContent := fmt.Sprintf(`# %s

Name: %s
Steam ID: %s
Expires In: %s
Expires At: %s
`, title, player.GetName(), ban.TargetID.String(), expIn, expAt)

	msg := &discordgo.MessageSend{
		Flags: discordgo.MessageFlagsIsComponentsV2,
		Components: []discordgo.MessageComponent{
			discordgo.Container{
				AccentColor: ptr.To(discord.ColourError),
				Components: []discordgo.MessageComponent{
					discordgo.Section{
						Components: []discordgo.MessageComponent{
							discordgo.TextDisplay{Content: msgContent},
						},
						Accessory: discordgo.Thumbnail{
							Media:       discordgo.UnfurledMediaItem{URL: player.GetAvatar().Full()},
							Description: ptr.To(fmt.Sprintf("Profile Picure [%s]", player.Avatarhash)),
						},
					},
				},
			},

			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "🗑️ Unban",
						CustomID: fmt.Sprintf("ban_unban_button_resp_%d", ban.BanID),
						Style:    discordgo.SuccessButton,
					},
					discordgo.Button{
						Label:    "🔨 Edit",
						CustomID: fmt.Sprintf("ban_edit_button_resp_%d", ban.BanID),
						Style:    discordgo.SecondaryButton,
					},
					discordgo.Button{
						Label: "🔎 View",
						URL:   link.Path(ban),
						Style: discordgo.LinkButton,
					},
					discordgo.Button{
						Label: "🌐 Steam",
						URL:   "https://steamcommunity.com/profiles/" + ban.TargetID.String(),
						Style: discordgo.LinkButton,
					},
				},
			},
		},
	}

	return msg
}

func DeleteReportMessage(existing ReportMessage, _ person.Info) *discordgo.MessageSend {
	return discord.NewMessageSend(
		discordgo.Container{
			AccentColor: ptr.To(discord.ColourWarn),
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: "#User report message deleted"},
				discordgo.TextDisplay{Content: existing.MessageMD},
			},
		},
	)
}

func EditReportMessageResponse(body string, oldBody string, link string, author person.Info, authorURL string) *discordgo.MessageSend {
	return discord.NewMessageSend(
		discordgo.TextDisplay{Content: "New report message edited"},
		discordgo.Container{
			AccentColor: ptr.To(discord.ColourWarn),
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: oldBody},
			},
		},
		discordgo.Container{
			AccentColor: ptr.To(discord.ColourSuccess),
			Components: []discordgo.MessageComponent{
				discordgo.TextDisplay{Content: body},
			},
		})
}

func ReportStatsMessage(meta ReportMeta, url string) *discordgo.MessageSend {
	const format = `# Current Open Report Counts
New: {{ .New }}
Total Open: {{ .TotalOpen }}
Total Closed: {{ .TotalClosed }}
Open >1 Day: {{ .Open1Day }}
Open >3 Days: {{ .Open3Days }}
Open >7 Days: {{ .Open1Week }}
`
	colour := discord.ColourSuccess
	if meta.OpenWeek > 0 {
		colour = discord.ColourError
	} else if meta.Open3Days > 0 {
		colour = discord.ColourWarn
	}

	body, errBody := discord.Render("report_stats", format, struct {
		New         int
		TotalOpen   int
		TotalClosed int
		Open1Day    int
		Open3Days   int
		Open1Week   int
	}{
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
	return discord.NewMessageSend(discordgo.Container{
		AccentColor: ptr.To(colour),
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{
				Content: body,
			},
		},
	})
}
