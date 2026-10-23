//nolint:testpackage // exercises the unexported seed role menu helpers directly.
package servers

import (
	"fmt"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/require"
)

func TestSeedRoleMenuOptions(t *testing.T) {
	guildRoles := []*discordgo.Role{
		{ID: "role-1", Name: "seeder-eu"},
		{ID: "role-2", Name: "seeder-us-west"},
		{ID: "role-3", Name: "seeder-us-east"},
	}

	t.Run("maps ids to role names and sorts by label", func(t *testing.T) {
		options := seedRoleMenuOptions([]string{"role-2", "role-1"}, guildRoles)
		require.Equal(t, []discordgo.SelectMenuOption{
			{Label: "seeder-eu", Value: "role-1"},
			{Label: "seeder-us-west", Value: "role-2"},
		}, options)
	})

	t.Run("skips ids that no longer exist in the guild", func(t *testing.T) {
		options := seedRoleMenuOptions([]string{"role-1", "role-missing"}, guildRoles)
		require.Equal(t, []discordgo.SelectMenuOption{{Label: "seeder-eu", Value: "role-1"}}, options)
	})

	t.Run("caps options at the discord limit", func(t *testing.T) {
		var (
			count = maxSelectMenuOptions + 5
			ids   = make([]string, 0, count)
			roles = make([]*discordgo.Role, 0, count)
		)

		for i := range count {
			ids = append(ids, fmt.Sprintf("role-%d", i))
			roles = append(roles, &discordgo.Role{ID: fmt.Sprintf("role-%d", i), Name: fmt.Sprintf("seeder-%02d", i)})
		}

		options := seedRoleMenuOptions(ids, roles)
		require.Len(t, options, maxSelectMenuOptions)
	})

	t.Run("no ids returns no options", func(t *testing.T) {
		require.Empty(t, seedRoleMenuOptions(nil, guildRoles))
	})
}
