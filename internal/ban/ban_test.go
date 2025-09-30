package ban_test

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/tests"
)

func TestBan(t *testing.T) {
	testDB := tests.NewFixture()
	defer testDB.Close()

	// bans := ban.NewBans(ban.NewRepository(testDB.Database), nil, nil, nil, nil, nil, nil)

	// opts := []ban.Opts{
	// 	{SourceID: steamid.New(o), TargetID: "", Duration: "P1M", BanType: banDomain.Banned, Reason: banDomain.Cheating, Origin: banDomain.System, Name: "", Note: "", ReasonText: "", CIDR: nil},
	// }

	// for _, opt := range opts {
	// 	ban, err := bans.Create(t.Context(), opt)
	// 	require.NoError(t, err)
	// 	require.Greater(t, 0, ban.BanID)
	// }
}
