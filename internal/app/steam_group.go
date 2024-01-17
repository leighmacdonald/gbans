package app

import (
	"context"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type groupMemberStore interface {
	GetBanGroups(ctx context.Context, filter store.GroupBansQueryFilter) ([]model.BannedGroupPerson, int64, error)
	GetMembersList(ctx context.Context, parentID int64, list *model.MembersList) error
	SaveMembersList(ctx context.Context, list *model.MembersList) error
}

type steamGroupMemberships struct {
	members    map[int64]steamid.Collection
	membersMu  *sync.RWMutex
	log        *zap.Logger
	store      groupMemberStore
	updateFreq time.Duration
}

func newSteamGroupMemberships(log *zap.Logger, store groupMemberStore) *steamGroupMemberships {
	return &steamGroupMemberships{
		store:      store,
		log:        log.Named("steamGroupMemberships"),
		members:    map[int64]steamid.Collection{},
		membersMu:  &sync.RWMutex{},
		updateFreq: time.Minute * 60,
	}
}

// isMember checks membership in the currently known banned group members.
func (g *steamGroupMemberships) isMember(steamID steamid.SID64) (int64, bool) {
	g.membersMu.RLock()
	defer g.membersMu.RUnlock()

	for parentID, groupMembers := range g.members {
		for _, member := range groupMembers {
			if steamID == member {
				return parentID, true
			}
		}
	}

	return 0, false
}

func (g *steamGroupMemberships) update(ctx context.Context) {
	newMap := map[int64]steamid.Collection{}

	var total int

	groupEntries, errGroupEntries := g.updateGroupBanMembers(ctx)
	if errGroupEntries == nil {
		for k, v := range groupEntries {
			total += len(v)
			newMap[k] = v
		}
	}

	g.membersMu.Lock()
	g.members = newMap
	g.membersMu.Unlock()

	g.log.Info("Updated group memberships", zap.Int("count", total))
}

// updateGroupBanMembers handles fetching and updating the member lists of steam groups. This does
// NOT use the steam API, so be careful to not call too often.
//
// Group IDs can be found using https://steamcommunity.com/groups/<GROUP_NAME>/memberslistxml/?xml=1
func (g *steamGroupMemberships) updateGroupBanMembers(ctx context.Context) (map[int64]steamid.Collection, error) {
	newMap := map[int64]steamid.Collection{}

	localCtx, cancel := context.WithTimeout(ctx, time.Second*120)
	defer cancel()

	groups, _, errGroups := g.store.GetBanGroups(ctx, store.GroupBansQueryFilter{})
	if errGroups != nil {
		if errors.Is(errGroups, store.ErrNoResult) {
			return newMap, nil
		}

		return nil, errors.Wrap(errGroups, "Failed to fetch banned groups")
	}

	for _, group := range groups {
		members, errMembers := steamweb.GetGroupMembers(localCtx, group.GroupID)
		if errMembers != nil {
			return nil, errors.Wrapf(errMembers, "Failed to fetch group members")
		}

		if len(members) == 0 {
			continue
		}

		memberList := model.NewMembersList(group.GroupID.Int64(), members)
		if errQuery := g.store.GetMembersList(ctx, group.GroupID.Int64(), &memberList); errQuery != nil {
			if !errors.Is(errQuery, store.ErrNoResult) {
				return nil, errors.Wrap(errQuery, "Failed to fetch members list")
			}
		}

		if errSave := g.store.SaveMembersList(ctx, &memberList); errSave != nil {
			return nil, errors.Wrap(errSave, "Failed to save banned groups member list")
		}

		newMap[group.GroupID.Int64()] = members

		// Group info doesn't use the steam api so its *heavily* rate limited. Let's try to minimize the ability to
		// get banned incase there is a lot of banned groups. This probably need to be increased if you are blocked a
		// large amount of groups.
		time.Sleep(time.Second * 5)
	}

	return newMap, nil
}

func (g *steamGroupMemberships) start(ctx context.Context) {
	ticker := time.NewTicker(g.updateFreq)
	updateChan := make(chan any)

	go func() {
		updateChan <- true
	}()

	for {
		select {
		case <-ticker.C:
			updateChan <- true
		case <-updateChan:
			g.update(ctx)
			ticker.Reset(g.updateFreq)
		case <-ctx.Done():
			g.log.Debug("steamGroupMemberships shutting down")

			return
		}
	}
}
