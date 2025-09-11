package ban

import (
	"context"
	"log/slog"
	"time"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type appeals struct {
	repository    domain.AppealRepository
	bans          BanUsecase
	persons       domain.PersonUsecase
	notifications domain.NotificationUsecase
	config        domain.ConfigUsecase
}

func NewAppealUsecase(ar domain.AppealRepository, bans BanUsecase, persons domain.PersonUsecase,
	notifications domain.NotificationUsecase, config domain.ConfigUsecase,
) domain.AppealUsecase {
	return &appeals{repository: ar, bans: bans, persons: persons, notifications: notifications, config: config}
}

func (u *appeals) GetAppealsByActivity(ctx context.Context, opts domain.AppealQueryFilter) ([]AppealOverview, error) {
	return u.repository.GetAppealsByActivity(ctx, opts)
}

func (u *appeals) EditBanMessage(ctx context.Context, curUser domain.UserProfile, banMessageID int64, newMsg string) (domain.BanAppealMessage, error) {
	existing, err := u.GetBanMessageByID(ctx, banMessageID)
	if err != nil {
		return domain.BanAppealMessage{}, err
	}

	bannedPerson, errReport := u.bans.Query(ctx, QueryOpts{
		BanID:   existing.BanID,
		Deleted: true,
		EvadeOk: true,
	})
	if errReport != nil {
		return existing, errReport
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
		return existing, domain.ErrPermissionDenied
	}

	if newMsg == "" {
		return existing, domain.ErrInvalidParameter
	}

	if newMsg == existing.MessageMD {
		return existing, database.ErrDuplicate
	}

	existing.MessageMD = newMsg

	if errSave := u.repository.SaveBanMessage(ctx, &existing); errSave != nil {
		return existing, errSave
	}

	conf := u.config.Config()

	u.notifications.Enqueue(ctx, domain.NewDiscordNotification(discord.ChannelModAppealLog, discord.NewAppealMessage(existing.MessageMD,
		conf.ExtURL(bannedPerson.Ban), curUser, conf.ExtURL(curUser))))

	slog.Debug("Appeal message updated", slog.Int64("message_id", banMessageID))

	return existing, nil
}

func (u *appeals) CreateBanMessage(ctx context.Context, curUser domain.UserProfile, banID int64, newMsg string) (domain.BanAppealMessage, error) {
	if banID <= 0 {
		return domain.BanAppealMessage{}, domain.ErrInvalidParameter
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{curUser.GetSteamID()}, domain.PModerator) {
		return domain.BanAppealMessage{}, domain.ErrPermissionDenied
	}

	if newMsg == "" {
		return domain.BanAppealMessage{}, domain.ErrInvalidParameter
	}

	bannedPerson, errReport := u.bans.Query(ctx, QueryOpts{
		BanID:   banID,
		Deleted: true,
		EvadeOk: true,
	})
	if errReport != nil {
		return domain.BanAppealMessage{}, errReport
	}

	if bannedPerson.AppealState != Open && curUser.PermissionLevel < domain.PModerator {
		return domain.BanAppealMessage{}, domain.ErrPermissionDenied
	}

	_, errTarget := u.persons.GetOrCreatePersonBySteamID(ctx, nil, bannedPerson.TargetID)
	if errTarget != nil {
		return domain.BanAppealMessage{}, errTarget
	}

	_, errSource := u.persons.GetOrCreatePersonBySteamID(ctx, nil, bannedPerson.SourceID)
	if errSource != nil {
		return domain.BanAppealMessage{}, errSource
	}

	msg := domain.NewBanAppealMessage(banID, curUser.SteamID, newMsg)
	msg.PermissionLevel = curUser.PermissionLevel
	msg.Personaname = curUser.Name
	msg.Avatarhash = curUser.Avatarhash

	if errSave := u.repository.SaveBanMessage(ctx, &msg); errSave != nil {
		return domain.BanAppealMessage{}, errSave
	}

	bannedPerson.UpdatedOn = time.Now()

	if errUpdate := u.bans.Save(ctx, &bannedPerson.Ban); errUpdate != nil {
		return domain.BanAppealMessage{}, errUpdate
	}

	conf := u.config.Config()

	u.notifications.Enqueue(ctx, domain.NewDiscordNotification(discord.ChannelModAppealLog, discord.NewAppealMessage(msg.MessageMD,
		conf.ExtURL(bannedPerson.Ban), curUser, conf.ExtURL(curUser))))

	u.notifications.Enqueue(ctx, domain.NewSiteGroupNotificationWithAuthor(
		[]domain.Privilege{domain.PModerator, domain.PAdmin},
		domain.SeverityInfo,
		"A new ban appeal message",
		bannedPerson.Path(),
		curUser))

	if curUser.SteamID != bannedPerson.TargetID {
		u.notifications.Enqueue(ctx, domain.NewSiteUserNotification(
			[]steamid.SteamID{bannedPerson.TargetID},
			domain.SeverityInfo,
			"A new ban appeal message",
			bannedPerson.Path()))
	}

	return msg, nil
}

func (u *appeals) GetBanMessages(ctx context.Context, userProfile domain.UserProfile, banID int64) ([]domain.BanAppealMessage, error) {
	banPerson, errGetBan := u.bans.Query(ctx, QueryOpts{
		BanID:   banID,
		Deleted: true,
		EvadeOk: true,
	})
	if errGetBan != nil {
		return nil, errGetBan
	}

	if !httphelper.HasPrivilege(userProfile, steamid.Collection{banPerson.TargetID, banPerson.SourceID}, domain.PModerator) {
		return nil, domain.ErrPermissionDenied
	}

	return u.repository.GetBanMessages(ctx, banID)
}

func (u *appeals) GetBanMessageByID(ctx context.Context, banMessageID int64) (domain.BanAppealMessage, error) {
	return u.repository.GetBanMessageByID(ctx, banMessageID)
}

func (u *appeals) DropBanMessage(ctx context.Context, curUser domain.UserProfile, banMessageID int64) error {
	existing, errExist := u.GetBanMessageByID(ctx, banMessageID)
	if errExist != nil {
		return errExist
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, domain.PModerator) {
		return domain.ErrPermissionDenied
	}

	if errDrop := u.repository.DropBanMessage(ctx, &existing); errDrop != nil {
		return errDrop
	}

	u.notifications.Enqueue(ctx, domain.NewDiscordNotification(
		discord.ChannelModAppealLog,
		discord.DeleteAppealMessage(&existing, curUser, u.config.ExtURL(curUser))))

	slog.Info("Appeal message deleted", slog.Int64("ban_message_id", banMessageID))

	return nil
}
