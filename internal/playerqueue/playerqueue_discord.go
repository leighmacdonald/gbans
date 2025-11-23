package playerqueue

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/ptr"
)

func NewPlayerqueueChatStatus(_ person.Info, target person.Info, status ChatStatus, reason string) *discordgo.MessageSend {
	colour := discord.ColourError
	switch status {
	case Readwrite:
		colour = discord.ColourSuccess
	case Readonly:
		colour = discord.ColourWarn
	}

	sid := target.GetSteamID()

	const foramt = `# Updated chat status
Status: {{.Status}}
Reason: {{.Reason}}
Name: {{.Name}}
SteamID: {{.SteamID}}`

	content, err := discord.Render("queue_chat_status", foramt, struct {
		Status  string
		Reason  string
		Name    string
		SteamID string
	}{
		Status:  string(status),
		Reason:  reason,
		Name:    target.GetName(),
		SteamID: sid.String(),
	})
	if err != nil {
		slog.Error("Failed to render queue_chat_status message", slog.String("error", err.Error()))
	}

	return discord.NewMessageSend(discordgo.Container{
		AccentColor: ptr.To(colour),
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: content},
		},
	})
}

func NewPlayerqueuePurge(_ person.Info, target person.Info, chatLog ChatLog, count int) *discordgo.MessageSend {
	const format = `# Player queue message purge
Message: {{.Message}}
Count: {{.Count}}
Name: {{.Name}}
SteamID: {{.SteamID}}
`
	sid := target.GetSteamID()

	body, errBody := discord.Render("queue_msg_purge", format, struct {
		Message string
		Count   int
		Name    string
		SteamID string
	}{
		Message: chatLog.BodyMD,
		Count:   count,
		Name:    target.GetName(),
		SteamID: sid.String(),
	})
	if errBody != nil {
		slog.Error("Failed to render message", slog.String("error", errBody.Error()))
	}

	return discord.NewMessageSend(discordgo.Container{
		AccentColor: ptr.To(discord.ColourError),
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: body},
		},
	})
}
