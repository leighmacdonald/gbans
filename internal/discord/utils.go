package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"go.uber.org/zap"
)

const (
	maxEmbedFields = 25
	// TODO ensure these

	//maxUsernameChars    = 32
	//maxAuthorChars      = 256.
	maxFieldNameChars   = 256
	maxFieldValueChars  = 1024
	maxDescriptionChars = 2048
)

type Colour int

const (
	Green  Colour = 3066993
	Orange Colour = 15105570
	Red    Colour = 10038562
)

var (
	defaultProvider = discordgo.MessageEmbedProvider{
		URL:  "https://github.com/leighmacdonald/gbans",
		Name: "gbans",
	}
	defaultFooter = discordgo.MessageEmbedFooter{
		Text:         "gbans",
		IconURL:      "https://cdn.discordapp.com/avatars/758536119397646370/6a371d1a481a72c512244ba9853f7eff.webp?size=128",
		ProxyIconURL: "",
	}
)

type ResponseMsgType int

const (
	MtString ResponseMsgType = iota
	MtEmbed
)

type Response struct {
	MsgType ResponseMsgType
	Value   any
}

type Payload struct {
	ChannelId string
	Embed     *discordgo.MessageEmbed
}

// RespErr creates a common error message embed.
func RespErr(response *Response, message string) {
	response.Value = &discordgo.MessageEmbed{
		URL:      "",
		Type:     discordgo.EmbedTypeRich,
		Title:    "Command Error",
		Color:    int(Red),
		Provider: &defaultProvider,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Message",
				Value:  message,
				Inline: false,
			},
		},
		Footer: &defaultFooter,
	}
	response.MsgType = MtEmbed
}

// RespOk will set up and allocate a base successful response embed that can be
// further customized.
func RespOk(response *Response, title string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Type:        discordgo.EmbedTypeRich,
		Title:       title,
		Description: "",
		Color:       int(Green),
		Footer:      &defaultFooter,
		Image:       nil,
		Thumbnail:   nil,
		Video:       nil,
		Provider:    &defaultProvider,
		Author:      nil,
		Fields:      nil,
	}
	if response != nil {
		response.MsgType = MtEmbed
		response.Value = embed
	}
	return embed
}

func AddFieldInline(embed *discordgo.MessageEmbed, title string, value string) {
	AddFieldRaw(embed, title, value, true)
}

func AddField(embed *discordgo.MessageEmbed, title string, value string) {
	AddFieldRaw(embed, title, value, false)
}

func AddFieldInt64Inline(embed *discordgo.MessageEmbed, title string, value int64) {
	AddField(embed, title, fmt.Sprintf("%d", value))
}

func AddAuthorProfile(embed *discordgo.MessageEmbed, sid steamid.SID64, name string, url string) {
	if name == "" {
		name = sid.String()
	}
	if name == "" {
		logger.Warn("Value cannot be empty, dropping field", zap.String("field", "name"))
		return
	}
	embed.Author = &discordgo.MessageEmbedAuthor{URL: url, Name: name}
}

// func addFieldInt64(embed *discordgo.MessageEmbed, logger *zap.Logger, title string, value int64) {
//	AddField(embed, logger, title, fmt.Sprintf("%d", value))
//}

// func addAuthor(embed *discordgo.MessageEmbed, person model.Person) {
//	name := person.PersonaName
//	if name == "" {
//		name = person.SteamID.String()
//	}
//	embed.Author = &discordgo.MessageEmbedAuthor{URL: person.ToURL(), Name: name}
//}

type Linkable interface {
	ToURL() string
}

func AddFieldsSteamID(embed *discordgo.MessageEmbed, steamId steamid.SID64) {
	AddFieldInline(embed, "STEAM", string(steamid.SID64ToSID(steamId)))
	AddFieldInline(embed, "STEAM3", string(steamid.SID64ToSID3(steamId)))
	AddFieldInline(embed, "SID64", steamId.String())
}

func AddLink(embed *discordgo.MessageEmbed, value Linkable) {
	url := value.ToURL()
	if len(url) > 0 {
		AddFieldRaw(embed, "Link", url, false)
	}
}

func AddFieldRaw(embed *discordgo.MessageEmbed, title string, value string, inline bool) {
	if len(embed.Fields) >= maxEmbedFields {
		logger.Warn("Dropping embed fields. Already at max count", zap.Int("max", maxEmbedFields))
		return
	}
	if len(title) == 0 {
		logger.Warn("Title cannot be empty, dropping field")
		return
	}
	if len(value) == 0 {
		logger.Warn("Value cannot be empty, dropping field", zap.String("field", title))
		return
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   truncate(title, maxFieldNameChars),
		Value:  truncate(value, maxFieldValueChars),
		Inline: inline,
	})
}

func truncate(str string, maxLen int) string {
	if len(str) > maxLen {
		return str[:maxLen]
	}
	return str
}
