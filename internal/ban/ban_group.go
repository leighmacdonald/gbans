package ban

import (
	"context"
	"encoding/xml"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	errFetchGroupBans          = errors.New("failed to fetch group bans")
	errLoadGroupBanMembersList = errors.New("failed to load group ban members list")
	errSaveGroupBanMembers     = errors.New("failed to save group ban members list")
)

type SteamGroupInfo struct {
	XMLName      xml.Name `xml:"memberList"`
	Text         string   `xml:",chardata"`
	GroupID64    int64    `xml:"groupID64"`
	GroupDetails struct {
		Text          string `xml:",chardata"`
		GroupName     string `xml:"groupName"`
		GroupURL      string `xml:"groupURL"`
		Headline      string `xml:"headline"`
		Summary       string `xml:"summary"`
		AvatarIcon    string `xml:"avatarIcon"`
		AvatarMedium  string `xml:"avatarMedium"`
		AvatarFull    string `xml:"avatarFull"`
		MemberCount   string `xml:"memberCount"`
		MembersInChat string `xml:"membersInChat"`
		MembersInGame string `xml:"membersInGame"`
		MembersOnline string `xml:"membersOnline"`
	} `xml:"groupDetails"`
	MemberCount    string `xml:"memberCount"`
	TotalPages     string `xml:"totalPages"`
	CurrentPage    string `xml:"currentPage"`
	StartingMember string `xml:"startingMember"`
	Members        struct {
		Text      string  `xml:",chardata"`
		SteamID64 []int64 `xml:"steamID64"`
	} `xml:"members"`
}

type MembersList struct {
	MembersID int64
	ParentID  int64
	Members   steamid.Collection
	CreatedOn time.Time
	UpdatedOn time.Time
}

func NewMembersList(parentID int64, members steamid.Collection) MembersList {
	now := time.Now()

	return MembersList{
		ParentID:  parentID,
		Members:   members,
		CreatedOn: now,
		UpdatedOn: now,
	}
}

type Memberships struct {
	*sync.RWMutex

	members    map[steamid.SteamID]steamid.Collection
	store      Repository
	updateFreq time.Duration
	tfAPI      *thirdparty.TFAPI
}

func NewMemberships(db Repository, tfAPI *thirdparty.TFAPI) *Memberships {
	return &Memberships{
		RWMutex:    &sync.RWMutex{},
		store:      db,
		members:    map[steamid.SteamID]steamid.Collection{},
		updateFreq: time.Minute * 60,
		tfAPI:      tfAPI,
	}
}

// IsMember checks membership in the currently known banned group members.
func (g *Memberships) IsMember(steamID steamid.SteamID) (steamid.SteamID, bool) {
	g.RLock()
	defer g.RUnlock()

	for parentID, groupMembers := range g.members {
		if slices.Contains(groupMembers, steamID) {
			return parentID, true
		}
	}

	return steamid.SteamID{}, false
}

func (g *Memberships) Update(ctx context.Context) {
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
func (g *Memberships) updateGroupBanMembers(ctx context.Context) (map[steamid.SteamID]steamid.Collection, error) {
	newMap := map[steamid.SteamID]steamid.Collection{}

	localCtx, cancel := context.WithTimeout(ctx, time.Second*120)
	defer cancel()

	groups, errGroups := g.store.Query(ctx, QueryOpts{GroupsOnly: true})
	if errGroups != nil {
		if errors.Is(errGroups, database.ErrNoResult) {
			return newMap, nil
		}

		return nil, errors.Join(errGroups, errFetchGroupBans)
	}

	for _, bannedGroup := range groups {
		group, errGroup := g.tfAPI.SteamGroup(localCtx, bannedGroup.TargetID)
		if errGroup != nil {
			return nil, errGroup
		}

		if len(group.Members) == 0 {
			continue
		}

		members := make(steamid.Collection, len(group.Members))
		for index, groupMember := range group.Members {
			members[index] = steamid.New(groupMember.SteamId)
		}

		grpID := steamid.New(group.GroupId)
		memberList := NewMembersList(grpID.Int64(), members)
		if errQuery := g.store.GetMembersList(localCtx, grpID.Int64(), &memberList); errQuery != nil {
			if !errors.Is(errQuery, database.ErrNoResult) {
				return nil, errors.Join(errQuery, errLoadGroupBanMembersList)
			}
		}

		if errSave := g.store.SaveMembersList(localCtx, &memberList); errSave != nil {
			return nil, errors.Join(errSave, errSaveGroupBanMembers)
		}

		newMap[grpID] = members

		// Group info doesn't use the steam api so its *heavily* rate limited. Let's try to minimize the ability to
		// get banned incase there is a lot of banned groups. This probably need to be increased if you are blocking a
		// large amount of groups.
		time.Sleep(time.Second * 5)
	}

	return newMap, nil
}
