package steamgroup

import (
	"context"
	"errors"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"go.uber.org/zap"
)

type banGroupUsecase struct {
	bgr domain.BanGroupRepository
	sg  *SteamGroupMemberships
	log *zap.Logger
}

func NewBanGroupUsecase(logger *zap.Logger, bgr domain.BanGroupRepository) domain.BanGroupUsecase {
	sg := NewSteamGroupMemberships(logger, bgr)
	return &banGroupUsecase{
		bgr: bgr,
		sg:  sg,
		log: logger.Named("bangroup"),
	}
}

func (s *banGroupUsecase) SaveBanGroup(ctx context.Context, banGroup *domain.BanGroup) error {
	return s.bgr.SaveBanGroup(ctx, banGroup)
}

func (s *banGroupUsecase) Start(ctx context.Context) {
	s.sg.Start(ctx)
}

func (s *banGroupUsecase) IsMember(steamID steamid.SID64) (steamid.GID, bool) {
	return s.sg.IsMember(steamID)
}

func (s *banGroupUsecase) GetBanGroup(ctx context.Context, groupID steamid.GID, banGroup *domain.BanGroup) error {
	return s.bgr.GetBanGroup(ctx, groupID, banGroup)
}

func (s *banGroupUsecase) GetBanGroupByID(ctx context.Context, banGroupID int64, banGroup *domain.BanGroup) error {
	return s.bgr.GetBanGroupByID(ctx, banGroupID, banGroup)
}

func (s *banGroupUsecase) GetBanGroups(ctx context.Context, filter domain.GroupBansQueryFilter) ([]domain.BannedGroupPerson, int64, error) {
	return s.bgr.GetBanGroups(ctx, filter)
}

func (s *banGroupUsecase) DropBanGroup(ctx context.Context, banGroup *domain.BanGroup) error {
	return s.bgr.DropBanGroup(ctx, banGroup)
}

func (s *banGroupUsecase) GetMembersList(ctx context.Context, parentID int64, list *domain.MembersList) error {
	return s.bgr.GetMembersList(ctx, parentID, list)
}

func (s *banGroupUsecase) SaveMembersList(ctx context.Context, list *domain.MembersList) error {
	return s.bgr.SaveMembersList(ctx, list)
}

func (s *banGroupUsecase) BanSteamGroup(ctx context.Context, banGroup *domain.BanGroup) error {
	members, membersErr := steamweb.GetGroupMembers(ctx, banGroup.GroupID)
	if membersErr != nil || len(members) == 0 {
		return errors.Join(membersErr, domain.ErrGroupValidate)
	}

	return s.bgr.BanSteamGroup(ctx, banGroup)
}
