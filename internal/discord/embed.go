package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	embed "github.com/leighmacdonald/discordgo-embed"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

const (
	iconURL      = "https://cdn.discordapp.com/avatars/758536119397646370/6a371d1a481a72c512244ba9853f7eff.webp?size=128"
	providerName = "gbans"
)

// NewEmbed construct a new discord embed message. This must not be chained if using gbans helper functions.
func NewEmbed(args ...string) *Embed {
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
		emb: newEmbed,
	}
}

func (e *Embed) AddFieldsSteamID(steamID steamid.SID64) *Embed {
	e.Embed().AddField("STEAM", string(steamid.SID64ToSID(steamID))).MakeFieldInline()
	e.Embed().AddField("STEAM3", string(steamid.SID64ToSID3(steamID))).MakeFieldInline()
	e.Embed().AddField("SID64", steamID.String()).MakeFieldInline()

	return e
}

type Embed struct {
	emb *embed.Embed
}

func (e *Embed) Embed() *embed.Embed {
	return e.emb
}

func (e *Embed) Message() *discordgo.MessageEmbed {
	return e.emb.MessageEmbed
}

func (e *Embed) AddTargetPerson(person model.PersonInfo) *Embed {
	name := person.GetName()
	if person.GetDiscordID() != "" {
		name = fmt.Sprintf("<@%s> | ", person.GetDiscordID()) + name
	}

	e.emb.AddField("Name", name)
	e.Embed().SetImage(person.GetAvatar().Full())

	return e
}

func (e *Embed) AddAuthorPersonInfo(person model.PersonInfo, url string) *Embed {
	name := person.GetName()
	if person.GetDiscordID() != "" {
		name = fmt.Sprintf("<@%s> | ", person.GetDiscordID()) + name
	}

	e.Embed().SetAuthor(name, person.GetAvatar().Full(), url)

	return e
}
