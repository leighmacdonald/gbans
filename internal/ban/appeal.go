package ban

import (
	"context"
	"log/slog"
	"time"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type AppealMessage struct {
	BanID           int64                `json:"ban_id"`
	BanMessageID    int64                `json:"ban_message_id"`
	AuthorID        steamid.SteamID      `json:"author_id"`
	MessageMD       string               `json:"message_md"`
	Deleted         bool                 `json:"deleted"`
	CreatedOn       time.Time            `json:"created_on"`
	UpdatedOn       time.Time            `json:"updated_on"`
	Avatarhash      string               `json:"avatarhash"`
	Personaname     string               `json:"personaname"`
	PermissionLevel permission.Privilege `json:"permission_level"`
}

func NewBanAppealMessage(banID int64, authorID steamid.SteamID, message string) AppealMessage {
	return AppealMessage{
		BanID:     banID,
		AuthorID:  authorID,
		MessageMD: message,
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	}
}

type AppealOverview struct {
	Ban

	SourcePersonaname string `json:"source_personaname"`
	SourceAvatarhash  string `json:"source_avatarhash"`
	TargetPersonaname string `json:"target_personaname"`
	TargetAvatarhash  string `json:"target_avatarhash"`
}

type AppealState int

const (
	AnyState AppealState = iota - 1
	Open
	Denied
	Accepted
	Reduced
	NoAppeal
)

func (as AppealState) String() string {
	switch as {
	case Denied:
		return "Denied"
	case Accepted:
		return "Accepted"
	case Reduced:
		return "Reduced"
	case NoAppeal:
		return "No Appeal"
	case AnyState:
		fallthrough
	case Open:
		fallthrough
	default:
		return "Open"
	}
}

type AppealQueryFilter struct {
	Deleted bool `json:"deleted"`
}

type Appeals struct {
	repository AppealRepository
	bans       Bans
	persons    person.Provider
	config     *config.Configuration
	notif      notification.Notifier
}

func NewAppeals(ar AppealRepository, bans Bans, persons person.Provider, config *config.Configuration, notif notification.Notifier) Appeals {
	return Appeals{repository: ar, bans: bans, persons: persons, config: config, notif: notif}
}

func (u *Appeals) GetAppealsByActivity(ctx context.Context, opts AppealQueryFilter) ([]AppealOverview, error) {
	return u.repository.ByActivity(ctx, opts)
}

func (u *Appeals) EditBanMessage(ctx context.Context, curUser person.Info, banMessageID int64, newMsg string) (AppealMessage, error) {
	existing, err := u.MessageByID(ctx, banMessageID)
	if err != nil {
		return AppealMessage{}, err
	}

	_, errReport := u.bans.QueryOne(ctx, QueryOpts{
		BanID:   existing.BanID,
		Deleted: true,
		EvadeOk: true,
	})
	if errReport != nil {
		return existing, errReport
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, permission.Moderator) {
		return existing, permission.ErrDenied
	}

	if newMsg == "" {
		return existing, domain.ErrInvalidParameter
	}

	if newMsg == existing.MessageMD {
		return existing, database.ErrDuplicate
	}

	existing.MessageMD = newMsg

	if errSave := u.repository.SaveMessage(ctx, &existing); errSave != nil {
		return existing, errSave
	}

	conf := u.config.Config()

	u.notif.Send(notification.NewDiscord(conf.Discord.LogChannelID, NewAppealMessage(existing.MessageMD,
		conf.ExtURLRaw("/ban/%d", existing.BanID), curUser, conf.ExtURL(curUser))))

	slog.Debug("Appeal message updated", slog.Int64("message_id", banMessageID))

	return existing, nil
}

func (u *Appeals) CreateBanMessage(ctx context.Context, curUser person.Info, banID int64, newMsg string) (AppealMessage, error) {
	if banID <= 0 {
		return AppealMessage{}, domain.ErrInvalidParameter
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{curUser.GetSteamID()}, permission.Moderator) {
		return AppealMessage{}, permission.ErrDenied
	}

	if newMsg == "" {
		return AppealMessage{}, domain.ErrInvalidParameter
	}

	bannedPerson, errReport := u.bans.QueryOne(ctx, QueryOpts{
		BanID:   banID,
		Deleted: true,
		EvadeOk: true,
	})
	if errReport != nil {
		return AppealMessage{}, errReport
	}

	if bannedPerson.AppealState != Open && !curUser.HasPermission(permission.Moderator) {
		return AppealMessage{}, permission.ErrDenied
	}

	_, errTarget := u.persons.GetOrCreatePersonBySteamID(ctx, bannedPerson.TargetID)
	if errTarget != nil {
		return AppealMessage{}, errTarget
	}

	_, errSource := u.persons.GetOrCreatePersonBySteamID(ctx, bannedPerson.SourceID)
	if errSource != nil {
		return AppealMessage{}, errSource
	}

	msg := NewBanAppealMessage(banID, curUser.GetSteamID(), newMsg)
	msg.PermissionLevel = curUser.Permissions()
	msg.Personaname = curUser.GetName()
	msg.Avatarhash = curUser.GetAvatar().Hash()

	if errSave := u.repository.SaveMessage(ctx, &msg); errSave != nil {
		return AppealMessage{}, errSave
	}

	bannedPerson.UpdatedOn = time.Now()

	if errUpdate := u.bans.Save(ctx, &bannedPerson); errUpdate != nil {
		return AppealMessage{}, errUpdate
	}

	// conf := u.config.Config()

	// u.notifications.Enqueue(ctx, notification.NewDiscordNotification(discord.ChannelModAppealLog, discord.NewAppealMessage(msg.MessageMD,
	// 	conf.ExtURL(bannedPerson.Ban), curUser, conf.ExtURL(curUser))))

	// u.notifications.Enqueue(ctx, notification.NewSiteGroupNotificationWithAuthor(
	// 	[]permission.Privilege{permission.PModerator, permission.PAdmin},
	// 	notification.SeverityInfo,
	// 	"A new ban appeal message",
	// 	bannedPerson.Path(),
	// 	curUser))

	// if curUser.SteamID != bannedPerson.TargetID {
	// 	u.notifications.Enqueue(ctx, notification.NewSiteUserNotification(
	// 		[]steamid.SteamID{bannedPerson.TargetID},
	// 		notification.SeverityInfo,
	// 		"A new ban appeal message",
	// 		bannedPerson.Path()))
	// }

	return msg, nil
}

func (u *Appeals) Messages(ctx context.Context, userProfile person.Info, banID int64) ([]AppealMessage, error) {
	banPerson, errGetBan := u.bans.QueryOne(ctx, QueryOpts{
		BanID:   banID,
		Deleted: true,
		EvadeOk: true,
	})
	if errGetBan != nil {
		return nil, errGetBan
	}

	if !httphelper.HasPrivilege(userProfile, steamid.Collection{banPerson.TargetID, banPerson.SourceID}, permission.Moderator) {
		return nil, permission.ErrDenied
	}

	return u.repository.Messages(ctx, banID)
}

func (u *Appeals) MessageByID(ctx context.Context, banMessageID int64) (AppealMessage, error) {
	return u.repository.MessageByID(ctx, banMessageID)
}

func (u *Appeals) DropMessage(ctx context.Context, curUser person.Info, banMessageID int64) error {
	existing, errExist := u.MessageByID(ctx, banMessageID)
	if errExist != nil {
		return errExist
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, permission.Moderator) {
		return permission.ErrDenied
	}

	if errDrop := u.repository.DropMessage(ctx, &existing); errDrop != nil {
		return errDrop
	}

	// u.notifications.Enqueue(ctx, notification.NewDiscordNotification(
	// 	discord.ChannelModAppealLog,
	// 	discord.DeleteAppealMessage(&existing, curUser, u.config.ExtURL(curUser))))

	slog.Info("Appeal message deleted", slog.Int64("ban_message_id", banMessageID))

	return nil
}
