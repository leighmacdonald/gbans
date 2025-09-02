package steamgroup

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/datetime"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type banGroupUsecase struct {
	repository    domain.BanGroupRepository
	persons       domain.PersonUsecase
	notifications domain.NotificationUsecase
	config        domain.ConfigUsecase
	tfAPI         *thirdparty.TFAPI
}

func NewBanGroupUsecase(repository domain.BanGroupRepository, persons domain.PersonUsecase,
	notifications domain.NotificationUsecase, config domain.ConfigUsecase, tfAPI *thirdparty.TFAPI,
) domain.BanGroupUsecase {
	return &banGroupUsecase{
		repository:    repository,
		persons:       persons,
		notifications: notifications,
		tfAPI:         tfAPI,
		config:        config,
	}
}

func (s banGroupUsecase) UpdateCache(ctx context.Context) error {
	groups, errGroups := s.Get(ctx, domain.GroupBansQueryFilter{Deleted: false})
	if errGroups != nil {
		return errGroups
	}

	if err := s.repository.TruncateCache(ctx); err != nil {
		return err
	}

	client := httphelper.NewHTTPClient()

	for _, group := range groups {
		listURL := fmt.Sprintf("https://steamcommunity.com/gid/%d/memberslistxml/?xml=1", group.GroupID.Int64())

		req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
		if errReq != nil {
			return errors.Join(errReq, domain.ErrRequestCreate)
		}

		resp, errResp := client.Do(req)
		if errResp != nil {
			return errors.Join(errResp, domain.ErrRequestPerform)
		}

		var list domain.SteamGroupInfo

		decoder := xml.NewDecoder(resp.Body)
		if err := decoder.Decode(&list); err != nil {
			_ = resp.Body.Close()

			return errors.Join(err, domain.ErrRequestDecode)
		}

		_ = resp.Body.Close()

		groupID := steamid.New(list.GroupID64)
		if !groupID.Valid() {
			return domain.ErrInvalidSID
		}

		for _, member := range list.Members.SteamID64 {
			steamID := steamid.New(member)
			if !steamID.Valid() {
				continue
			}

			// Statisfy FK
			_, errCreate := s.persons.GetOrCreatePersonBySteamID(ctx, nil, steamID)
			if errCreate != nil {
				return errCreate
			}
		}

		if err := s.repository.InsertCache(ctx, groupID, list.Members.SteamID64); err != nil {
			return err
		}
	}

	return nil
}

func (s banGroupUsecase) Save(ctx context.Context, banID int64, req domain.RequestBanGroupUpdate) (domain.BannedGroupPerson, error) {
	targetSID, sidValid := req.TargetSteamID(ctx)
	if !sidValid {
		return domain.BannedGroupPerson{}, domain.ErrInvalidParameter
	}

	ban, errBan := s.GetByID(ctx, banID)
	if errBan != nil {
		return domain.BannedGroupPerson{}, errBan
	}

	ban.Note = req.Note
	ban.ValidUntil = req.ValidUntil
	ban.TargetID = targetSID

	if err := s.repository.Save(ctx, &ban.BanGroup); err != nil {
		return domain.BannedGroupPerson{}, err
	}

	return ban, nil
}

func (s banGroupUsecase) GetByGID(ctx context.Context, groupID steamid.SteamID, banGroup *domain.BanGroup) error {
	return s.repository.GetByGID(ctx, groupID, banGroup)
}

func (s banGroupUsecase) GetByID(ctx context.Context, banGroupID int64) (domain.BannedGroupPerson, error) {
	return s.repository.GetByID(ctx, banGroupID)
}

func (s banGroupUsecase) Get(ctx context.Context, filter domain.GroupBansQueryFilter) ([]domain.BannedGroupPerson, error) {
	return s.repository.Get(ctx, filter)
}

func (s banGroupUsecase) Delete(ctx context.Context, banID int64, req domain.RequestUnban) error {
	ban, errFetch := s.GetByID(ctx, banID)
	if errFetch != nil {
		return errFetch
	}

	ban.UnbanReasonText = req.UnbanReasonText
	ban.Deleted = true

	return s.repository.Delete(ctx, &ban.BanGroup)
}

func (s banGroupUsecase) GetMembersList(ctx context.Context, parentID int64, list *domain.MembersList) error {
	return s.repository.GetMembersList(ctx, parentID, list)
}

func (s banGroupUsecase) SaveMembersList(ctx context.Context, list *domain.MembersList) error {
	return s.repository.SaveMembersList(ctx, list)
}

func (s banGroupUsecase) Ban(ctx context.Context, req domain.RequestBanGroupCreate) (domain.BannedGroupPerson, error) {
	gid, valid := req.TargetGroupID(ctx)
	if !valid {
		return domain.BannedGroupPerson{}, domain.ErrInvalidParameter
	}

	sid, validSID := req.SourceSteamID(ctx)
	if !validSID {
		return domain.BannedGroupPerson{}, domain.ErrInvalidParameter
	}

	targetID, validTargetID := req.TargetSteamID(ctx)
	if !validTargetID {
		return domain.BannedGroupPerson{}, domain.ErrInvalidParameter
	}

	duration, errDuration := datetime.CalcDuration(req.Duration, req.ValidUntil)
	if errDuration != nil {
		return domain.BannedGroupPerson{}, errDuration
	}

	_, errGroup := s.tfAPI.SteamGroup(ctx, gid)
	if errGroup != nil {
		return domain.BannedGroupPerson{}, errGroup
	}

	author, errAuthor := s.persons.GetPersonBySteamID(ctx, nil, sid)
	if errAuthor != nil {
		return domain.BannedGroupPerson{}, errors.Join(errAuthor, domain.ErrGetPerson)
	}

	_, errTarget := s.persons.GetPersonBySteamID(ctx, nil, targetID)
	if errTarget != nil {
		return domain.BannedGroupPerson{}, errors.Join(errTarget, domain.ErrGetPerson)
	}

	var banGroup domain.BanGroup
	if err := domain.NewBanSteamGroup(sid, targetID, duration, req.Note, domain.System, gid, "",
		domain.Banned, &banGroup); err != nil {
		return domain.BannedGroupPerson{}, err
	}

	if err := s.repository.Ban(ctx, &banGroup); err != nil {
		return domain.BannedGroupPerson{}, errors.Join(err, domain.ErrSaveBan)
	}

	s.notifications.Enqueue(ctx, domain.NewDiscordNotification(
		domain.ChannelBanLog,
		discord.BanGroupMessage(banGroup, author, s.config.Config())))

	return s.GetByID(ctx, banGroup.BanGroupID)
}
