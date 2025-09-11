package ban

import (
	"context"
	"log/slog"
	"time"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/person/permission"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type AppealsUsecase struct {
	repository    appealRepository
	bans          BanUsecase
	persons       *person.PersonUsecase
	notifications notification.NotificationUsecase
	config        *config.ConfigUsecase
}

func NewAppealUsecase(ar appealRepository, bans BanUsecase, persons *person.PersonUsecase,
	notifications notification.NotificationUsecase, config *config.ConfigUsecase,
) *AppealsUsecase {
	return &AppealsUsecase{repository: ar, bans: bans, persons: persons, notifications: notifications, config: config}
}

func (u *AppealsUsecase) GetAppealsByActivity(ctx context.Context, opts AppealQueryFilter) ([]AppealOverview, error) {
	return u.repository.GetAppealsByActivity(ctx, opts)
}

func (u *AppealsUsecase) EditBanMessage(ctx context.Context, curUser person.UserProfile, banMessageID int64, newMsg string) (BanAppealMessage, error) {
	existing, err := u.GetBanMessageByID(ctx, banMessageID)
	if err != nil {
		return BanAppealMessage{}, err
	}

	bannedPerson, errReport := u.bans.Query(ctx, QueryOpts{
		BanID:   existing.BanID,
		Deleted: true,
		EvadeOk: true,
	})
	if errReport != nil {
		return existing, errReport
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, permission.PModerator) {
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

	u.notifications.Enqueue(ctx, notification.NewDiscordNotification(discord.ChannelModAppealLog, discord.NewAppealMessage(existing.MessageMD,
		conf.ExtURL(bannedPerson.Ban), curUser, conf.ExtURL(curUser))))

	slog.Debug("Appeal message updated", slog.Int64("message_id", banMessageID))

	return existing, nil
}

func (u *AppealsUsecase) CreateBanMessage(ctx context.Context, curUser person.UserProfile, banID int64, newMsg string) (BanAppealMessage, error) {
	if banID <= 0 {
		return BanAppealMessage{}, domain.ErrInvalidParameter
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{curUser.GetSteamID()}, permission.PModerator) {
		return BanAppealMessage{}, domain.ErrPermissionDenied
	}

	if newMsg == "" {
		return BanAppealMessage{}, domain.ErrInvalidParameter
	}

	bannedPerson, errReport := u.bans.Query(ctx, QueryOpts{
		BanID:   banID,
		Deleted: true,
		EvadeOk: true,
	})
	if errReport != nil {
		return BanAppealMessage{}, errReport
	}

	if bannedPerson.AppealState != Open && curUser.PermissionLevel < permission.PModerator {
		return BanAppealMessage{}, domain.ErrPermissionDenied
	}

	_, errTarget := u.persons.GetOrCreatePersonBySteamID(ctx, nil, bannedPerson.TargetID)
	if errTarget != nil {
		return BanAppealMessage{}, errTarget
	}

	_, errSource := u.persons.GetOrCreatePersonBySteamID(ctx, nil, bannedPerson.SourceID)
	if errSource != nil {
		return BanAppealMessage{}, errSource
	}

	msg := NewBanAppealMessage(banID, curUser.SteamID, newMsg)
	msg.PermissionLevel = curUser.PermissionLevel
	msg.Personaname = curUser.Name
	msg.Avatarhash = curUser.Avatarhash

	if errSave := u.repository.SaveBanMessage(ctx, &msg); errSave != nil {
		return BanAppealMessage{}, errSave
	}

	bannedPerson.UpdatedOn = time.Now()

	if errUpdate := u.bans.Save(ctx, &bannedPerson.Ban); errUpdate != nil {
		return BanAppealMessage{}, errUpdate
	}

	conf := u.config.Config()

	u.notifications.Enqueue(ctx, notification.NewDiscordNotification(discord.ChannelModAppealLog, discord.NewAppealMessage(msg.MessageMD,
		conf.ExtURL(bannedPerson.Ban), curUser, conf.ExtURL(curUser))))

	u.notifications.Enqueue(ctx, notification.NewSiteGroupNotificationWithAuthor(
		[]permission.Privilege{permission.PModerator, permission.PAdmin},
		notification.SeverityInfo,
		"A new ban appeal message",
		bannedPerson.Path(),
		curUser))

	if curUser.SteamID != bannedPerson.TargetID {
		u.notifications.Enqueue(ctx, notification.NewSiteUserNotification(
			[]steamid.SteamID{bannedPerson.TargetID},
			notification.SeverityInfo,
			"A new ban appeal message",
			bannedPerson.Path()))
	}

	return msg, nil
}

func (u *AppealsUsecase) GetBanMessages(ctx context.Context, userProfile person.UserProfile, banID int64) ([]BanAppealMessage, error) {
	banPerson, errGetBan := u.bans.Query(ctx, QueryOpts{
		BanID:   banID,
		Deleted: true,
		EvadeOk: true,
	})
	if errGetBan != nil {
		return nil, errGetBan
	}

	if !httphelper.HasPrivilege(userProfile, steamid.Collection{banPerson.TargetID, banPerson.SourceID}, permission.PModerator) {
		return nil, domain.ErrPermissionDenied
	}

	return u.repository.GetBanMessages(ctx, banID)
}

func (u *AppealsUsecase) GetBanMessageByID(ctx context.Context, banMessageID int64) (BanAppealMessage, error) {
	return u.repository.GetBanMessageByID(ctx, banMessageID)
}

func (u *AppealsUsecase) DropBanMessage(ctx context.Context, curUser person.UserProfile, banMessageID int64) error {
	existing, errExist := u.GetBanMessageByID(ctx, banMessageID)
	if errExist != nil {
		return errExist
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, permission.PModerator) {
		return domain.ErrPermissionDenied
	}

	if errDrop := u.repository.DropBanMessage(ctx, &existing); errDrop != nil {
		return errDrop
	}

	u.notifications.Enqueue(ctx, notification.NewDiscordNotification(
		discord.ChannelModAppealLog,
		discord.DeleteAppealMessage(&existing, curUser, u.config.ExtURL(curUser))))

	slog.Info("Appeal message deleted", slog.Int64("ban_message_id", banMessageID))

	return nil
}
