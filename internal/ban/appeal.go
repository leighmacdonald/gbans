package ban

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/database"
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

func (am AppealMessage) Path() string {
	// TODO link to msg direct #.
	return fmt.Sprintf("/ban/%d", am.BanID)
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
	AppealRepository

	bans         Bans
	persons      person.Provider
	notif        notification.Notifier
	logChannelID string
}

func NewAppeals(ar AppealRepository, bans Bans, persons person.Provider, notif notification.Notifier, logChannelID string) Appeals {
	return Appeals{AppealRepository: ar, bans: bans, persons: persons, notif: notif, logChannelID: logChannelID}
}

func (u *Appeals) GetAppealsByActivity(ctx context.Context, opts AppealQueryFilter) ([]AppealOverview, error) {
	return u.ByActivity(ctx, opts)
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
		return existing, httphelper.ErrInvalidParameter
	}

	if newMsg == existing.MessageMD {
		return existing, database.ErrDuplicate
	}

	existing.MessageMD = newMsg

	if errSave := u.SaveMessage(ctx, &existing); errSave != nil {
		return existing, errSave
	}

	if u.notif != nil {
		content := fmt.Sprintf(`# Appeal message edited
%s`, newMsg)
		go u.notif.Send(notification.NewDiscord(u.logChannelID, newAppealMessageResponse(existing, content)))
	}

	slog.Debug("Appeal message updated", slog.Int64("message_id", banMessageID))

	return existing, nil
}

func (u *Appeals) CreateBanMessage(ctx context.Context, curUser person.Info, banID int64, newMsg string) (AppealMessage, error) {
	if banID <= 0 {
		return AppealMessage{}, httphelper.ErrInvalidParameter
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{curUser.GetSteamID()}, permission.Moderator) {
		return AppealMessage{}, permission.ErrDenied
	}

	if newMsg == "" {
		return AppealMessage{}, httphelper.ErrInvalidParameter
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

	if errSave := u.SaveMessage(ctx, &msg); errSave != nil {
		return AppealMessage{}, errSave
	}

	bannedPerson.UpdatedOn = time.Now()

	if errUpdate := u.bans.Save(ctx, &bannedPerson); errUpdate != nil {
		return AppealMessage{}, errUpdate
	}

	go u.notif.Send(notification.NewDiscord(u.logChannelID, newAppealMessageResponse(msg, fmt.Sprintf(`# New report message posted
%s`, msg.MessageMD))))

	go u.notif.Send(notification.NewSiteGroupNotificationWithAuthor(
		[]permission.Privilege{permission.Moderator, permission.Admin},
		notification.Info,
		"A new ban appeal message",
		link.Path(bannedPerson),
		curUser))

	if curUser.GetSteamID() != bannedPerson.TargetID {
		go u.notif.Send(notification.NewSiteUser(
			[]steamid.SteamID{bannedPerson.TargetID},
			notification.Info,
			"A new ban appeal message",
			link.Path(bannedPerson)))
	}

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

	return u.AppealRepository.Messages(ctx, banID)
}

func (u *Appeals) MessageByID(ctx context.Context, banMessageID int64) (AppealMessage, error) {
	return u.AppealRepository.MessageByID(ctx, banMessageID)
}

func (u *Appeals) DropMessage(ctx context.Context, curUser person.Info, banMessageID int64) error {
	existing, errExist := u.MessageByID(ctx, banMessageID)
	if errExist != nil {
		return errExist
	}

	if !httphelper.HasPrivilege(curUser, steamid.Collection{existing.AuthorID}, permission.Moderator) {
		return permission.ErrDenied
	}

	if errDrop := u.AppealRepository.DropMessage(ctx, &existing); errDrop != nil {
		return errDrop
	}

	go u.notif.Send(notification.NewDiscord(u.logChannelID, newAppealMessageDelete(existing)))

	slog.Info("Appeal message deleted", slog.Int64("ban_message_id", banMessageID))

	return nil
}
