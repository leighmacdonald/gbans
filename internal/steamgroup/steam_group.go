package steamgroup

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

var (
	errFetchGroupBans          = errors.New("failed to fetch group bans")
	errFetchGroupBanMembersAPI = errors.New("failed to fetch group ban members from steam api")
	errLoadGroupBanMembersList = errors.New("failed to load group ban members list")
	errSaveGroupBanMembers     = errors.New("failed to save group ban members list")
)

type SteamGroupMemberships struct {
	members map[steamid.SteamID]steamid.Collection
	*sync.RWMutex
	store      domain.BanGroupRepository
	updateFreq time.Duration
}

func NewSteamGroupMemberships(db domain.BanGroupRepository) *SteamGroupMemberships {
	return &SteamGroupMemberships{
		RWMutex:    &sync.RWMutex{},
		store:      db,
		members:    map[steamid.SteamID]steamid.Collection{},
		updateFreq: time.Minute * 60,
	}
}

func (g *SteamGroupMemberships) Start(ctx context.Context) {
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
			slog.Debug("SteamGroupMemberships shutting down")

			return
		}
	}
}

// IsMember checks membership in the currently known banned group members.
func (g *SteamGroupMemberships) IsMember(steamID steamid.SteamID) (steamid.SteamID, bool) {
	g.RLock()
	defer g.RUnlock()

	for parentID, groupMembers := range g.members {
		for _, member := range groupMembers {
			if steamID == member {
				return parentID, true
			}
		}
	}

	return steamid.SteamID{}, false
}

func (g *SteamGroupMemberships) update(ctx context.Context) {
	newMap := map[steamid.SteamID]steamid.Collection{}

	var total int

	groupEntries, errGroupEntries := g.updateGroupBanMembers(ctx)
	if errGroupEntries == nil {
		for k, v := range groupEntries {
			total += len(v)
			newMap[k] = v
		}
	}

	g.Lock()
	g.members = newMap
	g.Unlock()

	slog.Info("Updated group memberships", slog.Int("count", total))
}

// updateGroupBanMembers handles fetching and updating the member lists of steam groups. This does
// NOT use the steam API, so be careful to not call too often.
//
// Group IDs can be found using https://steamcommunity.com/groups/<GROUP_NAME>/memberslistxml/?xml=1
func (g *SteamGroupMemberships) updateGroupBanMembers(ctx context.Context) (map[steamid.SteamID]steamid.Collection, error) {
	newMap := map[steamid.SteamID]steamid.Collection{}

	localCtx, cancel := context.WithTimeout(ctx, time.Second*120)
	defer cancel()

	groups, _, errGroups := g.store.Get(ctx, domain.GroupBansQueryFilter{})
	if errGroups != nil {
		if errors.Is(errGroups, domain.ErrNoResult) {
			return newMap, nil
		}

		return nil, errors.Join(errGroups, errFetchGroupBans)
	}

	for _, group := range groups {
		members, errMembers := steamweb.GetGroupMembers(localCtx, group.GroupID)
		if errMembers != nil {
			return nil, errors.Join(errMembers, errFetchGroupBanMembersAPI)
		}

		if len(members) == 0 {
			continue
		}

		memberList := domain.NewMembersList(group.GroupID.Int64(), members)
		if errQuery := g.store.GetMembersList(ctx, group.GroupID.Int64(), &memberList); errQuery != nil {
			if !errors.Is(errQuery, domain.ErrNoResult) {
				return nil, errors.Join(errQuery, errLoadGroupBanMembersList)
			}
		}

		if errSave := g.store.SaveMembersList(ctx, &memberList); errSave != nil {
			return nil, errors.Join(errSave, errSaveGroupBanMembers)
		}

		newMap[group.GroupID] = members

		// Group info doesn't use the steam api so its *heavily* rate limited. Let's try to minimize the ability to
		// get banned incase there is a lot of banned groups. This probably need to be increased if you are blocking a
		// large amount of groups.
		time.Sleep(time.Second * 5)
	}

	return newMap, nil
}
