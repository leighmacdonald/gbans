package ban

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

var (
	ErrSteamBanLoad        = errors.New("failed to load existing bans with friends")
	ErrSteamBanFriendsLoad = errors.New("failed to load friends of banned user")
	ErrSteamFriendsFetch   = errors.New("failed to load steam friends")
	ErrSteamFriendsSave    = errors.New("failed to save steam friends")
)

type SteamFriends struct {
	updateFreq time.Duration
	bu         domain.BanSteamUsecase
	bgu        domain.BanGroupUsecase
	members    map[steamid.SID64]steamid.Collection
	*sync.RWMutex
}

func NewSteamFriends(bu domain.BanSteamUsecase, bgu domain.BanGroupUsecase) *SteamFriends {
	return &SteamFriends{
		RWMutex:    &sync.RWMutex{},
		updateFreq: time.Hour * 6,
		bu:         bu,
		bgu:        bgu,
		members:    map[steamid.SID64]steamid.Collection{},
	}
}

func (sf *SteamFriends) IsMember(steamID steamid.SID64) (steamid.SID64, bool) {
	sf.RLock()
	defer sf.RUnlock()

	for parentID, groupMembers := range sf.members {
		for _, member := range groupMembers {
			if steamID == member {
				return parentID, true
			}
		}
	}

	return "", false
}

func (sf *SteamFriends) Start(ctx context.Context) {
	timer := time.NewTicker(time.Minute * 120)
	update := make(chan bool)

	go func() {
		update <- true
	}()

	for {
		select {
		case <-update:
			updated, errUpdate := sf.updateSteamBanMembers(ctx)
			if errUpdate != nil {
				slog.Error("failed to update steam ban friends")

				continue
			}

			sf.Lock()
			sf.members = updated
			sf.Unlock()

			slog.Debug("Updated friend list member bans",
				slog.Int("friends", len(updated)))
		case <-timer.C:
			update <- true
		case <-ctx.Done():
			return
		}
	}
}

func (sf *SteamFriends) updateSteamBanMembers(ctx context.Context) (map[steamid.SID64]steamid.Collection, error) {
	newMap := map[steamid.SID64]steamid.Collection{}

	localCtx, cancel := context.WithTimeout(ctx, time.Second*120)
	defer cancel()

	opts := domain.SteamBansQueryFilter{
		BansQueryFilter:    domain.BansQueryFilter{QueryFilter: domain.QueryFilter{Deleted: false}},
		IncludeFriendsOnly: true,
	}

	steamBans, _, errSteam := sf.bu.Get(ctx, opts)
	if errSteam != nil {
		if errors.Is(errSteam, domain.ErrNoResult) {
			return newMap, nil
		}

		return nil, errors.Join(errSteam, ErrSteamBanLoad)
	}

	for _, steamBan := range steamBans {
		friends, errFriends := steamweb.GetFriendList(localCtx, steamBan.TargetID)
		if errFriends != nil {
			return nil, errors.Join(errFriends, ErrSteamBanFriendsLoad)
		}

		if len(friends) == 0 {
			continue
		}

		var sids steamid.Collection

		for _, friend := range friends {
			sids = append(sids, friend.SteamID)
		}

		memberList := domain.NewMembersList(steamBan.TargetID.Int64(), sids)
		if errQuery := sf.bgu.GetMembersList(ctx, steamBan.TargetID.Int64(), &memberList); errQuery != nil {
			if !errors.Is(errQuery, domain.ErrNoResult) {
				return nil, errors.Join(errQuery, ErrSteamFriendsFetch)
			}
		}

		if errSave := sf.bgu.SaveMembersList(ctx, &memberList); errSave != nil {
			return nil, errors.Join(errSave, ErrSteamFriendsSave)
		}

		newMap[steamBan.TargetID] = memberList.Members
	}

	return newMap, nil
}
