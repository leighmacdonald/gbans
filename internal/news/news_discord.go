package news

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
)

func NewNewsMessage(body string, title string) *discordgo.MessageSend {
	const format = `# %s
%s`

	return discord.NewMessageSend(discordgo.Container{
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: fmt.Sprintf(format, title, body)},
		},
	})
}

func EditNewsMessages(title string, body string) *discordgo.MessageSend {
	const format = `# %s
%s`

	return discord.NewMessageSend(discordgo.Container{
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: fmt.Sprintf(format, title, body)},
		},
	})
}
