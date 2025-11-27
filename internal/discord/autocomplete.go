package discord

import (
	"context"
	"errors"
	"fmt"
	"slices"
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

		// TODO come up with a real solution that will work for more than a single option.
		// T stuct with tags with paths? 0.0.1  -> 2 levels deep, second value.
		if len(data.Options) > 0 {
			var val any
			if len(data.Options[0].Options) > 0 {
				if len(data.Options[0].Options[0].Options) > 0 {
					val = data.Options[0].Options[0].Options[0].Value
				} else {
					val = data.Options[0].Options[0].Value
				}
			} else {
				val = data.Options[0].Value
			}
			if val != nil {
				query = strings.ToLower(fmt.Sprintf("%s", val))
			}
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

		slices.SortFunc(choices, func(i, j *discordgo.ApplicationCommandOptionChoice) int {
			return strings.Compare(strings.ToLower(i.Name), strings.ToLower(j.Name))
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
