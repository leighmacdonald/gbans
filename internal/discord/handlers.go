package discord

import (
	"github.com/bwmarrin/discordgo"
)

type CommandOptions map[optionKey]*discordgo.ApplicationCommandInteractionDataOption

// OptionMap will take the recursive discord slash commands and flatten them into a simple
// map.
func OptionMap(options []*discordgo.ApplicationCommandInteractionDataOption) CommandOptions {
	optionM := make(CommandOptions, len(options))
	for _, opt := range options {
		optionM[optionKey(opt.Name)] = opt
	}

	return optionM
}

func (opts CommandOptions) String(key optionKey) string {
	root, found := opts[key]
	if !found {
		return ""
	}
	val, ok := root.Value.(string)
	if !ok {
		return ""
	}

	return val
}
