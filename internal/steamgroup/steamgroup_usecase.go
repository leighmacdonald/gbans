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
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
)

type banGroupUsecase struct {
	repository domain.BanGroupRepository
	persons    domain.PersonUsecase
	discord    domain.DiscordUsecase
	config     domain.ConfigUsecase
}

func NewBanGroupUsecase(repository domain.BanGroupRepository, persons domain.PersonUsecase,
	discord domain.DiscordUsecase, config domain.ConfigUsecase,
) domain.BanGroupUsecase {
	return &banGroupUsecase{
		repository: repository,
		persons:    persons,
		discord:    discord,
		config:     config,
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
			_, errCreate := s.persons.GetOrCreatePersonBySteamID(ctx, steamID)
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

func (s banGroupUsecase) Save(ctx context.Context, banGroup *domain.BanGroup) error {
	return s.repository.Save(ctx, banGroup)
}

func (s banGroupUsecase) GetByGID(ctx context.Context, groupID steamid.SteamID, banGroup *domain.BanGroup) error {
	return s.repository.GetByGID(ctx, groupID, banGroup)
}

func (s banGroupUsecase) GetByID(ctx context.Context, banGroupID int64, banGroup *domain.BanGroup) error {
	return s.repository.GetByID(ctx, banGroupID, banGroup)
}

func (s banGroupUsecase) Get(ctx context.Context, filter domain.GroupBansQueryFilter) ([]domain.BannedGroupPerson, error) {
	return s.repository.Get(ctx, filter)
}

func (s banGroupUsecase) Delete(ctx context.Context, banGroup *domain.BanGroup) error {
	return s.repository.Delete(ctx, banGroup)
}

func (s banGroupUsecase) GetMembersList(ctx context.Context, parentID int64, list *domain.MembersList) error {
	return s.repository.GetMembersList(ctx, parentID, list)
}

func (s banGroupUsecase) SaveMembersList(ctx context.Context, list *domain.MembersList) error {
	return s.repository.SaveMembersList(ctx, list)
}

func (s banGroupUsecase) Ban(ctx context.Context, banGroup *domain.BanGroup) error {
	members, membersErr := steamweb.GetGroupMembers(ctx, banGroup.GroupID)
	if membersErr != nil || len(members) == 0 {
		return errors.Join(membersErr, domain.ErrGroupValidate)
	}

	author, errAuthor := s.persons.GetPersonBySteamID(ctx, banGroup.SourceID)
	if errAuthor != nil {
		return errors.Join(membersErr, domain.ErrGetPerson)
	}

	if err := s.repository.Ban(ctx, banGroup); err != nil {
		return errors.Join(err, domain.ErrSaveBan)
	}

	s.discord.SendPayload(domain.ChannelBanLog, discord.BanGroupMessage(*banGroup, author, s.config.Config()))

	return nil
}
