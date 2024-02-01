package appeal

import (
	"context"
	"time"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type appealUsecase struct {
	appealRepository domain.AppealRepository
	banUsecase       domain.BanSteamUsecase
	personUsecase    domain.PersonUsecase
	discordUsecase   domain.DiscordUsecase
	configUsecase    domain.ConfigUsecase
}

func NewAppealUsecase(ar domain.AppealRepository, banUsecase domain.BanSteamUsecase, personUsecase domain.PersonUsecase,
	discordUsecase domain.DiscordUsecase, configUsecase domain.ConfigUsecase,
) domain.AppealUsecase {
	return &appealUsecase{appealRepository: ar, banUsecase: banUsecase, personUsecase: personUsecase, discordUsecase: discordUsecase, configUsecase: configUsecase}
}

func (u *appealUsecase) GetAppealsByActivity(ctx context.Context, opts domain.AppealQueryFilter) ([]domain.AppealOverview, int64, error) {
	return u.appealRepository.GetAppealsByActivity(ctx, opts)
}

func (u *appealUsecase) SaveBanMessage(ctx context.Context, curUserProfile domain.UserProfile, req domain.BanAppealMessage) (*domain.BanAppealMessage, error) {
	if req.MessageMD == "" {
		return nil, domain.ErrBadRequest
	}

	bannedPerson := domain.NewBannedPerson()
	if errReport := u.banUsecase.GetByBanID(ctx, req.BanID, &bannedPerson, true); errReport != nil {
		return nil, errReport
	}

	if bannedPerson.AppealState != domain.Open && curUserProfile.PermissionLevel < domain.PModerator {
		return nil, domain.ErrPermissionDenied
	}

	var target domain.Person
	if errTarget := u.personUsecase.GetOrCreatePersonBySteamID(ctx, bannedPerson.TargetID, &target); errTarget != nil {
		return nil, errTarget
	}

	var source domain.Person
	if errSource := u.personUsecase.GetOrCreatePersonBySteamID(ctx, bannedPerson.SourceID, &source); errSource != nil {
		return nil, errSource
	}

	msg := domain.NewBanAppealMessage(req.BanID, curUserProfile.SteamID, req.MessageMD)
	msg.PermissionLevel = curUserProfile.PermissionLevel
	msg.Personaname = curUserProfile.Name
	msg.Avatarhash = curUserProfile.Avatarhash

	if errSave := u.appealRepository.SaveBanMessage(ctx, &msg); errSave != nil {
		return nil, errSave
	}

	conf := u.configUsecase.Config()

	u.discordUsecase.SendPayload(domain.ChannelModLog, discord.NewAppealMessage(msg.MessageMD,
		conf.ExtURL(bannedPerson.BanSteam), curUserProfile, conf.ExtURL(curUserProfile)))

	bannedPerson.UpdatedOn = time.Now()

	if errUpdate := u.banUsecase.Save(ctx, &bannedPerson.BanSteam); errUpdate != nil {
		return nil, errUpdate
	}

	return &msg, nil
}

func (u *appealUsecase) GetBanMessages(ctx context.Context, banID int64) ([]domain.BanAppealMessage, error) {
	return u.appealRepository.GetBanMessages(ctx, banID)
}

func (u *appealUsecase) GetBanMessageByID(ctx context.Context, banMessageID int, message *domain.BanAppealMessage) error {
	return u.appealRepository.GetBanMessageByID(ctx, banMessageID, message)
}

func (u *appealUsecase) DropBanMessage(ctx context.Context, curUser domain.UserProfile, message *domain.BanAppealMessage) error {
	if errDrop := u.appealRepository.DropBanMessage(ctx, message); errDrop != nil {
		return errDrop
	}

	u.discordUsecase.SendPayload(domain.ChannelModLog, discord.DeleteAppealMessage(message, curUser, u.configUsecase.ExtURL(curUser)))

	return nil
}
