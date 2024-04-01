package ban

import (
	"context"
	"errors"
	"strconv"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type banASNUsecase struct {
	repository     domain.BanASNRepository
	discordUsecase domain.DiscordUsecase
	networkUsecase domain.NetworkUsecase
}

func NewBanASNUsecase(repository domain.BanASNRepository, discordUsecase domain.DiscordUsecase, networkUsecase domain.NetworkUsecase) domain.BanASNUsecase {
	return banASNUsecase{
		repository:     repository,
		discordUsecase: discordUsecase,
		networkUsecase: networkUsecase,
	}
}

func (s banASNUsecase) Expired(ctx context.Context) ([]domain.BanASN, error) {
	return s.repository.Expired(ctx)
}

func (s banASNUsecase) Ban(ctx context.Context, banASN *domain.BanASN) error {
	var existing domain.BanASN
	if errGetExistingBan := s.repository.GetByASN(ctx, banASN.ASNum, &existing); errGetExistingBan != nil {
		if !errors.Is(errGetExistingBan, domain.ErrNoResult) {
			return errors.Join(errGetExistingBan, domain.ErrFailedFetchBan)
		}
	}

	if errSave := s.repository.Save(ctx, banASN); errSave != nil {
		return errors.Join(errSave, domain.ErrSaveBan)
	}
	// TODO Kick all Current players matching
	return nil
}

func (s banASNUsecase) Unban(ctx context.Context, asnNum string) (bool, error) {
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

	s.discordUsecase.SendPayload(domain.ChannelModLog, discord.UnbanASNMessage(asNum))

	return true, nil
}

func (s banASNUsecase) GetByASN(ctx context.Context, asNum int64, banASN *domain.BanASN) error {
	return s.repository.GetByASN(ctx, asNum, banASN)
}

func (s banASNUsecase) Get(ctx context.Context, filter domain.ASNBansQueryFilter) ([]domain.BannedASNPerson, int64, error) {
	return s.repository.Get(ctx, filter)
}

func (s banASNUsecase) Save(ctx context.Context, banASN *domain.BanASN) error {
	return s.repository.Save(ctx, banASN)
}

func (s banASNUsecase) Delete(ctx context.Context, banASN *domain.BanASN) error {
	return s.repository.Delete(ctx, banASN)
}
