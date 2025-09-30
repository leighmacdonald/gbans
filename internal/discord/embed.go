package discord

import (
	"github.com/bwmarrin/discordgo"
	embed "github.com/leighmacdonald/discordgo-embed"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/steamid/v4/steamid"
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
		Emb: newEmbed,
	}
}

func (e *Embed) AddFieldsSteamID(steamID steamid.SteamID) *Embed {
	e.Embed().AddField("STEAM", string(steamID.Steam(false))).MakeFieldInline()
	e.Embed().AddField("STEAM3", string(steamID.Steam3())).MakeFieldInline()
	e.Embed().AddField("SID64", steamID.String()).MakeFieldInline()

	return e
}

type Embed struct {
	Emb *embed.Embed
}

func (e *Embed) Embed() *embed.Embed {
	return e.Emb
}

func (e *Embed) Message() *discordgo.MessageEmbed {
	return e.Emb.MessageEmbed
}

func (e *Embed) AddTargetPerson(person person.Info) *Embed {
	name := person.GetName()

	e.Emb.AddField("Name", name)
	e.Embed().SetImage(person.GetAvatar().Full())

	return e
}

func (e *Embed) AddAuthorPersonInfo(person person.Info, url string) *Embed {
	name := person.GetName()

	e.Embed().SetAuthor(name, person.GetAvatar().Full(), url)
	e.AddFieldsSteamID(person.GetSteamID())

	return e
}
