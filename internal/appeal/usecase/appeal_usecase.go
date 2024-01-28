package usecase

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"time"
)

type appealUsecase struct {
	appealRepository domain.AppealRepository
	banUsecase       domain.BanUsecase
}

func (s *appealUsecase) SaveBanMessage(ctx context.Context, message *domain.BanAppealMessage) error {
	if err := s.appealRepository.SaveBanMessage(ctx, message); err != nil {
		return err
	}

	bannedPerson := domain.NewBannedPerson()
	if errBan := s.banUsecase.GetBanByBanID(ctx, message.BanID, &bannedPerson, true); errBan != nil {
		return errs.ErrNoResult
	}

	bannedPerson.UpdatedOn = time.Now()

	if errUpdate := s.banUsecase.SaveBan(ctx, &bannedPerson.BanSteam); errUpdate != nil {
		return errUpdate
	}

	return nil
}

func (s *appealUsecase) GetBanMessages(ctx context.Context, banID int64) ([]domain.BanAppealMessage, error) {
	return s.appealRepository.GetBanMessages(ctx, banID)
}

func (s *appealUsecase) GetBanMessageByID(ctx context.Context, banMessageID int, message *domain.BanAppealMessage) error {
	return s.appealRepository.GetBanMessageByID(ctx, banMessageID, message)
}

func (s *appealUsecase) DropBanMessage(ctx context.Context, message *domain.BanAppealMessage) error {
	return s.appealRepository.DropBanMessage(ctx, message)
}
