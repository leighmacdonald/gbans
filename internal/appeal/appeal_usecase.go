package appeal

import (
	"context"
	"time"

	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/steamid/v4/steamid"
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

func (u *appealUsecase) EditBanMessage(ctx context.Context, curUser domain.UserProfile, reportID int64, newMsg string) (domain.BanAppealMessage, error) {
	existing, err := u.GetBanMessageByID(ctx, reportID)
	if err != nil {
		return domain.BanAppealMessage{}, err
	}

	bannedPerson, errReport := u.banUsecase.GetByBanID(ctx, existing.BanID, true)
	if errReport != nil {
		return existing, errReport
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
		return existing, domain.ErrPermissionDenied
	}

	if newMsg == "" {
		return existing, domain.ErrBadRequest
	}

	if newMsg == existing.MessageMD {
		return existing, domain.ErrDuplicate
	}

	existing.MessageMD = newMsg

	if errSave := u.appealRepository.SaveBanMessage(ctx, &existing); errSave != nil {
		return existing, errSave
	}

	conf := u.configUsecase.Config()

	u.discordUsecase.SendPayload(domain.ChannelModLog, discord.NewAppealMessage(existing.MessageMD,
		conf.ExtURL(bannedPerson.BanSteam), curUser, conf.ExtURL(curUser)))

	return existing, nil
}

func (u *appealUsecase) CreateBanMessage(ctx context.Context, curUser domain.UserProfile, banID int64, newMsg string) (domain.BanAppealMessage, error) {
	if !httphelper.HasPrivilege(curUser, steamid.Collection{curUser.GetSteamID()}, domain.PModerator) {
		return domain.BanAppealMessage{}, domain.ErrPermissionDenied
	}

	if newMsg == "" {
		return domain.BanAppealMessage{}, domain.ErrBadRequest
	}

	bannedPerson, errReport := u.banUsecase.GetByBanID(ctx, banID, true)
	if errReport != nil {
		return domain.BanAppealMessage{}, errReport
	}

	if bannedPerson.AppealState != domain.Open && curUser.PermissionLevel < domain.PModerator {
		return domain.BanAppealMessage{}, domain.ErrPermissionDenied
	}

	_, errTarget := u.personUsecase.GetOrCreatePersonBySteamID(ctx, bannedPerson.TargetID)
	if errTarget != nil {
		return domain.BanAppealMessage{}, errTarget
	}

	_, errSource := u.personUsecase.GetOrCreatePersonBySteamID(ctx, bannedPerson.SourceID)
	if errSource != nil {
		return domain.BanAppealMessage{}, errSource
	}

	msg := domain.NewBanAppealMessage(banID, curUser.SteamID, newMsg)
	msg.PermissionLevel = curUser.PermissionLevel
	msg.Personaname = curUser.Name
	msg.Avatarhash = curUser.Avatarhash

	if errSave := u.appealRepository.SaveBanMessage(ctx, &msg); errSave != nil {
		return domain.BanAppealMessage{}, errSave
	}

	conf := u.configUsecase.Config()

	u.discordUsecase.SendPayload(domain.ChannelModLog, discord.NewAppealMessage(msg.MessageMD,
		conf.ExtURL(bannedPerson.BanSteam), curUser, conf.ExtURL(curUser)))

	bannedPerson.UpdatedOn = time.Now()

	if errUpdate := u.banUsecase.Save(ctx, &bannedPerson.BanSteam); errUpdate != nil {
		return domain.BanAppealMessage{}, errUpdate
	}

	u.discordUsecase.SendPayload(domain.ChannelModLog, discord.EditAppealMessage(msg, msg.MessageMD, curUser, u.configUsecase.ExtURL(curUser)))

	return msg, nil
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

func (u *appealUsecase) GetBanMessageByID(ctx context.Context, banMessageID int64) (domain.BanAppealMessage, error) {
	return u.appealRepository.GetBanMessageByID(ctx, banMessageID)
}

func (u *appealUsecase) DropBanMessage(ctx context.Context, curUser domain.UserProfile, banMessageID int64) error {
	existing, errExist := u.GetBanMessageByID(ctx, banMessageID)
	if errExist != nil {
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
