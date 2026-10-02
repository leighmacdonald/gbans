package ban_test

import (
	"encoding/xml"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestSteamGroupInfo_XML(t *testing.T) {
	const sample = `<memberList>
		<groupID64>103582791451024762</groupID64>
		<groupDetails>
			<groupName>Team Fortress 2 Cheaters</groupName>
			<groupURL>https://steamcommunity.com/groups/tf2cheaters</groupURL>
			<memberCount>2</memberCount>
		</groupDetails>
		<memberCount>2</memberCount>
		<totalPages>1</totalPages>
		<currentPage>1</currentPage>
		<startingMember>1</startingMember>
		<members>
			<steamID64>76561198000000001</steamID64>
			<steamID64>76561198000000002</steamID64>
		</members>
	</memberList>`

	var info ban.SteamGroupInfo
	require.NoError(t, xml.Unmarshal([]byte(sample), &info))

	require.Equal(t, int64(103582791451024762), info.GroupID64)
	require.Equal(t, "Team Fortress 2 Cheaters", info.GroupDetails.GroupName)
	require.Equal(t, "https://steamcommunity.com/groups/tf2cheaters", info.GroupDetails.GroupURL)
	require.Equal(t, "2", info.GroupDetails.MemberCount)
	require.Equal(t, "2", info.MemberCount)
	require.Equal(t, "1", info.TotalPages)
	require.Equal(t, "1", info.CurrentPage)
	require.Equal(t, "1", info.StartingMember)
	require.Equal(t, []int64{76561198000000001, 76561198000000002}, info.Members.SteamID64)
}

func TestNewMembersList(t *testing.T) {
	memberA, memberB := steamid.RandSID64(), steamid.RandSID64()

	list := ban.NewMembersList(42, steamid.Collection{memberA, memberB})

	require.Zero(t, list.MembersID)
	require.Equal(t, int64(42), list.ParentID)
	require.Equal(t, steamid.Collection{memberA, memberB}, list.Members)
	require.True(t, list.CreatedOn.Before(time.Now().Add(time.Second)))
	require.True(t, list.UpdatedOn.Before(time.Now().Add(time.Second)))
}

func TestMemberships_IsMember(t *testing.T) {
	t.Parallel()

	e := newEnv(t)

	memberships := ban.NewMemberships(e.repo, fixture.TFApi)

	member := steamid.RandSID64()
	parent, found := memberships.IsMember(member)
	require.False(t, found)
	require.Zero(t, parent.Int64())
}
