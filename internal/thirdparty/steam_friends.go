package thirdparty

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"go.uber.org/zap"
)

var (
	ErrSteamBanLoad        = errors.New("failed to load existing bans with friends")
	ErrSteamBanFriendsLoad = errors.New("failed to load friends of banned user")
	ErrSteamFriendsFetch   = errors.New("failed to load steam friends")
	ErrSteamFriendsSave    = errors.New("failed to save steam friends")
)

type SteamFriends struct {
	log        *zap.Logger
	updateFreq time.Duration
	store      store.Stores
	members    map[steamid.SID64]steamid.Collection
	*sync.RWMutex
}

func NewSteamFriends(logger *zap.Logger, database store.Stores) *SteamFriends {
	return &SteamFriends{
		RWMutex:    &sync.RWMutex{},
		log:        logger.Named("SteamFriends"),
		updateFreq: time.Hour * 6,
		store:      database,
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
				sf.log.Error("failed to update steam ban friends")

				continue
			}

			sf.Lock()
			sf.members = updated
			sf.Unlock()

			sf.log.Debug("Updated friend list member bans",
				zap.Int("friends", len(updated)))
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

	opts := model.SteamBansQueryFilter{
		BansQueryFilter:    model.BansQueryFilter{QueryFilter: model.QueryFilter{Deleted: false}},
		IncludeFriendsOnly: true,
	}

	steamBans, _, errSteam := sf.store.GetBansSteam(ctx, opts)
	if errSteam != nil {
		if errors.Is(errSteam, errs.ErrNoResult) {
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

		memberList := model.NewMembersList(steamBan.TargetID.Int64(), sids)
		if errQuery := sf.store.GetMembersList(ctx, steamBan.TargetID.Int64(), &memberList); errQuery != nil {
			if !errors.Is(errQuery, errs.ErrNoResult) {
				return nil, errors.Join(errQuery, ErrSteamFriendsFetch)
			}
		}

		if errSave := sf.store.SaveMembersList(ctx, &memberList); errSave != nil {
			return nil, errors.Join(errSave, ErrSteamFriendsSave)
		}

		newMap[steamBan.TargetID] = memberList.Members
	}

	return newMap, nil
}
