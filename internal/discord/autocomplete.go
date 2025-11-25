package discord

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type AutoCompleteValuer interface {
	Name() string
	Value() string
}

func NewAutoCompleteValue(name string, value string) AutoCompleValue {
	return AutoCompleValue{name: name, value: value}
}

type AutoCompleValue struct {
	name  string
	value string
}

func (v AutoCompleValue) Name() string { return v.name }

func (v AutoCompleValue) Value() string { return v.value }

type AutoCompleter func(ctx context.Context, query string) ([]AutoCompleteValuer, error)

// Autocomplete returns a function that will return results for a discord option autocompketion response.
func Autocomplete(completer AutoCompleter) func(context.Context, *discordgo.Session, *discordgo.InteractionCreate) error {
	return func(ctx context.Context, session *discordgo.Session, interaction *discordgo.InteractionCreate) error {
		var (
			data    = interaction.ApplicationCommandData()
			choices []*discordgo.ApplicationCommandOptionChoice
			query   string
		)

		if len(data.Options) > 0 {
			query = strings.ToLower(fmt.Sprintf("%s", data.Options[0].Value))
		}

		values, errValues := completer(ctx, query)
		if errValues != nil {
			return errValues
		}

		for _, autoValue := range values {
			if query == "" || strings.Contains(autoValue.Name(), query) || strings.Contains(autoValue.Value(), query) {
				choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
					Name:  autoValue.Name(),
					Value: autoValue.Value(),
				})
			}

			if len(choices) == 25 {
				break
			}
		}

		sort.Slice(choices, func(i, j int) bool {
			return choices[i].Name < choices[j].Name
		})

		if err := session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{Choices: choices},
		}); err != nil {
			return errors.Join(err, ErrCommandSend)
		}

		return nil
	}
}
