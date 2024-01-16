package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	embed "github.com/leighmacdonald/discordgo-embed"
	"github.com/leighmacdonald/gbans/internal/common"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

// NewEmbed construct a new discord embed message. This must not be chained if using gbans helper functions.
func NewEmbed(config config.Config, args ...string) *Embed {
	newEmbed := embed.
		NewEmbed().
		SetFooter(providerName, iconURL)

	if len(args) == 2 {
		newEmbed = newEmbed.SetTitle(args[0]).
			SetDescription(args[1])
	} else if len(args) == 1 {
		newEmbed = newEmbed.SetTitle(args[0])
	}

	return &Embed{
		emb:    newEmbed,
		config: config,
	}
}

func (e *Embed) AddFieldsSteamID(steamID steamid.SID64) *Embed {
	e.Embed().AddField("STEAM", string(steamid.SID64ToSID(steamID))).MakeFieldInline()
	e.Embed().AddField("STEAM3", string(steamid.SID64ToSID3(steamID))).MakeFieldInline()
	e.Embed().AddField("SID64", steamID.String()).MakeFieldInline()

	return e
}

type Embed struct {
	emb    *embed.Embed
	config config.Config
}

func (e *Embed) Embed() *embed.Embed {
	return e.emb
}

func (e *Embed) Message() *discordgo.MessageEmbed {
	return e.emb.MessageEmbed
}

func (e *Embed) AddTargetPerson(person common.PersonInfo) *Embed {
	name := person.GetName()
	if person.GetDiscordID() != "" {
		name = fmt.Sprintf("<@%s> | ", person.GetDiscordID()) + name
	}

	e.emb.AddField("Name", name)
	e.Embed().SetImage(person.GetAvatar().Full())

	return e
}

func (e *Embed) AddAuthorPersonInfo(person common.PersonInfo) *Embed {
	name := person.GetName()
	if person.GetDiscordID() != "" {
		name = fmt.Sprintf("<@%s> | ", person.GetDiscordID()) + name
	}

	e.Embed().SetAuthor(name, person.GetAvatar().Full(), e.config.ExtURL(person))

	return e
}
