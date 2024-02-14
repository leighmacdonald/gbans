package appeal

import (
	"context"
	"time"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/steamid/v3/steamid"
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

func (u *appealUsecase) SaveBanMessage(ctx context.Context, curUser domain.UserProfile, reportMessageID int64, newMsg string) (*domain.BanAppealMessage, error) {
	var existing domain.BanAppealMessage
	if err := u.GetBanMessageByID(ctx, reportMessageID, &existing); err != nil {
		return nil, err
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
		return nil, domain.ErrPermissionDenied
	}

	if newMsg == "" {
		return nil, domain.ErrBadRequest
	}

	if newMsg == existing.MessageMD {
		return nil, domain.ErrDuplicate
	}

	existing.MessageMD = newMsg

	bannedPerson, errReport := u.banUsecase.GetByBanID(ctx, existing.BanID, true)
	if errReport != nil {
		return nil, errReport
	}

	if bannedPerson.AppealState != domain.Open && curUser.PermissionLevel < domain.PModerator {
		return nil, domain.ErrPermissionDenied
	}

	_, errTarget := u.personUsecase.GetOrCreatePersonBySteamID(ctx, bannedPerson.TargetID)
	if errTarget != nil {
		return nil, errTarget
	}

	_, errSource := u.personUsecase.GetOrCreatePersonBySteamID(ctx, bannedPerson.SourceID)
	if errSource != nil {
		return nil, errSource
	}

	msg := domain.NewBanAppealMessage(existing.BanID, curUser.SteamID, existing.MessageMD)
	msg.PermissionLevel = curUser.PermissionLevel
	msg.Personaname = curUser.Name
	msg.Avatarhash = curUser.Avatarhash

	if errSave := u.appealRepository.SaveBanMessage(ctx, &msg); errSave != nil {
		return nil, errSave
	}

	conf := u.configUsecase.Config()

	u.discordUsecase.SendPayload(domain.ChannelModLog, discord.NewAppealMessage(msg.MessageMD,
		conf.ExtURL(bannedPerson.BanSteam), curUser, conf.ExtURL(curUser)))

	bannedPerson.UpdatedOn = time.Now()

	if errUpdate := u.banUsecase.Save(ctx, &bannedPerson.BanSteam); errUpdate != nil {
		return nil, errUpdate
	}

	u.discordUsecase.SendPayload(domain.ChannelModLog, discord.EditAppealMessage(existing, msg.MessageMD, curUser, u.configUsecase.ExtURL(curUser)))

	return &msg, nil
}

func (u *appealUsecase) GetBanMessages(ctx context.Context, userProfile domain.UserProfile, banID int64) ([]domain.BanAppealMessage, error) {
	banPerson, errGetBan := u.banUsecase.GetByBanID(ctx, banID, true)
	if errGetBan != nil {
		return nil, errGetBan
	}

	if !httphelper.HasPrivilege(userProfile, steamid.Collection{banPerson.TargetID, banPerson.SourceID}, domain.PModerator) {
		return nil, domain.ErrPermissionDenied
	}

	return u.appealRepository.GetBanMessages(ctx, banID)
}

func (u *appealUsecase) GetBanMessageByID(ctx context.Context, banMessageID int64, message *domain.BanAppealMessage) error {
	return u.appealRepository.GetBanMessageByID(ctx, banMessageID, message)
}

func (u *appealUsecase) DropBanMessage(ctx context.Context, curUser domain.UserProfile, banMessageID int64) error {
	var existing domain.BanAppealMessage
	if errExist := u.GetBanMessageByID(ctx, banMessageID, &existing); errExist != nil {
		return errExist
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
		return domain.ErrPermissionDenied
	}

	if errDrop := u.appealRepository.DropBanMessage(ctx, &existing); errDrop != nil {
		return errDrop
	}

	u.discordUsecase.SendPayload(domain.ChannelModLog, discord.DeleteAppealMessage(&existing, curUser, u.configUsecase.ExtURL(curUser)))

	return nil
}
