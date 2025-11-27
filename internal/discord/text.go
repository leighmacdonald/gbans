package discord

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"text/template"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/ptr"
)

type TextProcessor func(text string) string

func Render(name string, templ []byte, context any, textProcessor ...TextProcessor) (string, error) {
	var buffer bytes.Buffer
	tmpl, err := template.New(name).
		Funcs(template.FuncMap{
			"linkPath": link.Path,
			"linkRaw":  link.Raw,
		}).
		Parse(string(templ))
	if err != nil {
		return "", errors.Join(err, ErrTemplate)
	}
	if err = tmpl.Execute(&buffer, context); err != nil {
		return "", errors.Join(err, ErrTemplate)
	}

	body := buffer.String()
	for _, processor := range textProcessor {
		body = processor(body)
	}

	return body, nil
}

// HydrateLinks will transform relative markdown links into full urls, eg:
// [Settings](/wiki/Settings) -> [Settings](http://example.com/wiki/Settings),
func HydrateLinks() TextProcessor {
	extURLRegex := regexp.MustCompile(`\[(.+?)]\((/.+?)\)`)

	return func(text string) string {
		return extURLRegex.ReplaceAllString(text, fmt.Sprintf(`[$1](%s$2)`, link.Raw("")))
	}
}

func Heading(text string) discordgo.TextDisplay {
	return discordgo.TextDisplay{
		Content: "### " + text,
	}
}

func BodyColouredText(colour int, text string) discordgo.Container {
	return discordgo.Container{
		AccentColor: ptr.To(colour),
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: text},
		},
	}
}

func BodyText(text string) discordgo.Container {
	return discordgo.Container{
		Components: []discordgo.MessageComponent{
			discordgo.TextDisplay{Content: text},
		},
	}
}

func BodyColour(colour int, components ...discordgo.MessageComponent) discordgo.Container {
	return discordgo.Container{
		AccentColor: ptr.To(colour),
		Components:  components,
	}
}

func Body(components ...discordgo.MessageComponent) discordgo.Container {
	return discordgo.Container{
		Components: components,
	}
}

func Buttons(buttons ...discordgo.MessageComponent) discordgo.ActionsRow {
	return discordgo.ActionsRow{
		Components: buttons,
	}
}
