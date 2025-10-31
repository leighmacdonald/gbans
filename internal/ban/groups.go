package ban

import (
	"context"
	"time"

	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type GroupMemberships struct {
	tfAPI *thirdparty.TFAPI
	repo  Repository
}

func NewGroupMemberships(tfAPI *thirdparty.TFAPI, repo Repository) *GroupMemberships {
	return &GroupMemberships{tfAPI: tfAPI, repo: repo}
}

// groupMemberUpdater updates the current members of banned Steam groups in the database.
func (g GroupMemberships) UpdateCache(ctx context.Context) error {
	bans, errBans := g.repo.Query(ctx, QueryOpts{GroupsOnly: true})
	if errBans != nil {
		return errBans
	}

	if err := g.repo.TruncateCache(ctx); err != nil {
		return err
	}

	for idx, ban := range bans {
		if ban.Deleted || ban.ValidUntil.Before(time.Now()) {
			continue
		}

		if idx > 0 {
			// Not sure what the rate limit is, but be generous for groups.
			time.Sleep(time.Second * 5)
		}

		groupInfo, err := g.tfAPI.SteamGroup(ctx, ban.TargetID)
		if err != nil {
			return err
		}

		var list []int64
		for _, member := range groupInfo.Members {
			sid := steamid.New(member.SteamId)
			list = append(list, sid.Int64())
		}

		if err := g.repo.InsertCache(ctx, ban.TargetID, list); err != nil {
			return err
		}
	}

	return nil
}
