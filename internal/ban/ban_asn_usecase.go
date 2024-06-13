package ban

import (
	"context"
	"errors"
	"strconv"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type banASN struct {
	repository domain.BanASNRepository
	discord    domain.DiscordUsecase
	networks   domain.NetworkUsecase
	config     domain.ConfigUsecase
	person     domain.PersonUsecase
}

func NewBanASNUsecase(repository domain.BanASNRepository, discord domain.DiscordUsecase,
	network domain.NetworkUsecase, config domain.ConfigUsecase, person domain.PersonUsecase,
) domain.BanASNUsecase {
	return banASN{
		repository: repository,
		discord:    discord,
		networks:   network,
		config:     config,
		person:     person,
	}
}

func (s banASN) Expired(ctx context.Context) ([]domain.BanASN, error) {
	return s.repository.Expired(ctx)
}

func (s banASN) Ban(ctx context.Context, banASN *domain.BanASN) error {
	var existing domain.BanASN
	if errGetExistingBan := s.repository.GetByASN(ctx, banASN.ASNum, &existing); errGetExistingBan != nil {
		if !errors.Is(errGetExistingBan, domain.ErrNoResult) {
			return errors.Join(errGetExistingBan, domain.ErrFailedFetchBan)
		}
	}

	author, errAuthor := s.person.GetPersonBySteamID(ctx, banASN.SourceID)
	if errAuthor != nil {
		return errors.Join(errAuthor, domain.ErrGetPerson)
	}

	if errSave := s.repository.Save(ctx, banASN); errSave != nil {
		return errors.Join(errSave, domain.ErrSaveBan)
	}

	s.discord.SendPayload(domain.ChannelBanLog, discord.BanASNMessage(*banASN, author, s.config.Config()))

	return nil
}

func (s banASN) Unban(ctx context.Context, asnNum string) (bool, error) {
	asNum, errConv := strconv.ParseInt(asnNum, 10, 64)
	if errConv != nil {
		return false, errors.Join(errConv, domain.ErrParseASN)
	}

	var banASN domain.BanASN
	if errGetBanASN := s.repository.GetByASN(ctx, asNum, &banASN); errGetBanASN != nil {
		return false, errors.Join(errGetBanASN, domain.ErrFetchASNBan)
	}

	if errDrop := s.repository.Delete(ctx, &banASN); errDrop != nil {
		return false, errors.Join(errDrop, domain.ErrDropASNBan)
	}

	s.discord.SendPayload(domain.ChannelModLog, discord.UnbanASNMessage(asNum))

	return true, nil
}

func (s banASN) GetByID(ctx context.Context, banID int64, banASN *domain.BanASN) error {
	return s.repository.GetByID(ctx, banID, banASN)
}

func (s banASN) GetByASN(ctx context.Context, asNum int64, banASN *domain.BanASN) error {
	return s.repository.GetByASN(ctx, asNum, banASN)
}

func (s banASN) Get(ctx context.Context, filter domain.ASNBansQueryFilter) ([]domain.BannedASNPerson, error) {
	return s.repository.Get(ctx, filter)
}

func (s banASN) Save(ctx context.Context, banASN *domain.BanASN) error {
	return s.repository.Save(ctx, banASN)
}

func (s banASN) Delete(ctx context.Context, banASN *domain.BanASN) error {
	return s.repository.Delete(ctx, banASN)
}
