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
	banGroupRepository domain.BanGroupRepository
	groupMemberships   *SteamGroupMemberships
	log                *zap.Logger
}

func NewBanGroupUsecase(logger *zap.Logger, banGroupRepository domain.BanGroupRepository) domain.BanGroupUsecase {
	sg := NewSteamGroupMemberships(logger, banGroupRepository)

	return &banGroupUsecase{
		banGroupRepository: banGroupRepository,
		groupMemberships:   sg,
		log:                logger.Named("bangroup"),
	}
}

func (s *banGroupUsecase) Save(ctx context.Context, banGroup *domain.BanGroup) error {
	return s.banGroupRepository.Save(ctx, banGroup)
}

func (s *banGroupUsecase) Start(ctx context.Context) {
	s.groupMemberships.Start(ctx)
}

func (s *banGroupUsecase) IsMember(steamID steamid.SID64) (steamid.GID, bool) {
	return s.groupMemberships.IsMember(steamID)
}

func (s *banGroupUsecase) GetByGID(ctx context.Context, groupID steamid.GID, banGroup *domain.BanGroup) error {
	return s.banGroupRepository.GetByGID(ctx, groupID, banGroup)
}

func (s *banGroupUsecase) GetByID(ctx context.Context, banGroupID int64, banGroup *domain.BanGroup) error {
	return s.banGroupRepository.GetByID(ctx, banGroupID, banGroup)
}

func (s *banGroupUsecase) Get(ctx context.Context, filter domain.GroupBansQueryFilter) ([]domain.BannedGroupPerson, int64, error) {
	return s.banGroupRepository.Get(ctx, filter)
}

func (s *banGroupUsecase) Delete(ctx context.Context, banGroup *domain.BanGroup) error {
	return s.banGroupRepository.Delete(ctx, banGroup)
}

func (s *banGroupUsecase) GetMembersList(ctx context.Context, parentID int64, list *domain.MembersList) error {
	return s.banGroupRepository.GetMembersList(ctx, parentID, list)
}

func (s *banGroupUsecase) SaveMembersList(ctx context.Context, list *domain.MembersList) error {
	return s.banGroupRepository.SaveMembersList(ctx, list)
}

func (s *banGroupUsecase) Ban(ctx context.Context, banGroup *domain.BanGroup) error {
	members, membersErr := steamweb.GetGroupMembers(ctx, banGroup.GroupID)
	if membersErr != nil || len(members) == 0 {
		return errors.Join(membersErr, domain.ErrGroupValidate)
	}

	return s.banGroupRepository.Ban(ctx, banGroup)
}
