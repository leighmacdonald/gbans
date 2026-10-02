package ban

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/leighmacdonald/gbans/internal/config/link"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type AppealMessage struct {
	BanID        int32
	BanMessageID int64
	AuthorID     steamid.SteamID
	MessageMD    string
	Deleted      bool
	CreatedOn    time.Time
	UpdatedOn    time.Time
	Avatarhash   string
	Personaname  string
}

func (am AppealMessage) Path() string {
	return fmt.Sprintf("/ban/%d#msg-%d", am.BanID, am.BanMessageID)
}

func NewBanAppealMessage(banID int32, authorID steamid.SteamID, message string) AppealMessage {
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

	SourcePersonaname string
	SourceAvatarhash  string
	TargetPersonaname string
	TargetAvatarhash  string
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
	Deleted bool
}

type Appeals struct {
	AppealRepository

	bans         Bans
	persons      person.Provider
	notif        notification.Notifier
	logChannelID string
	roleAuth     *rpc.RoleAuth
}

func NewAppeals(ar AppealRepository, bans Bans, persons person.Provider, notif notification.Notifier, logChannelID string, roleAuth *rpc.RoleAuth) Appeals {
	return Appeals{AppealRepository: ar, bans: bans, persons: persons, notif: notif, logChannelID: logChannelID, roleAuth: roleAuth}
}

func (u *Appeals) GetAppealsByActivity(ctx context.Context, opts AppealQueryFilter) ([]AppealOverview, error) {
	return u.ByActivity(ctx, opts)
}

func (u *Appeals) EditBanMessage(ctx context.Context, curUser person.BaseUser, banMessageID int64, newMsg string) (AppealMessage, error) {
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

	if !u.roleAuth.HasPermissionForSteamID(ctx, curUser.GetSteamID(), rolesv1.Permission_PERMISSION_APPEAL_ADMIN) && !existing.AuthorID.Equal(curUser.GetSteamID()) {
		return existing, rpc.ErrPermission
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
		go u.notif.Send(notification.NewDiscord(u.logChannelID, newAppealMessageResponse(existing)))
	}

	slog.Debug("Appeal message updated", slog.Int64("message_id", banMessageID))

	return existing, nil
}

func (u *Appeals) CreateBanMessage(ctx context.Context, curUser person.BaseUser, banID int32, newMsg string) (AppealMessage, error) {
	if banID <= 0 {
		return AppealMessage{}, httphelper.ErrInvalidParameter
	}

	ban, errBan := u.bans.QueryOne(ctx, QueryOpts{BanID: banID})
	if errBan != nil {
		return AppealMessage{}, httphelper.ErrInvalidParameter
	}

	if !u.roleAuth.HasPermissionForSteamID(ctx, curUser.GetSteamID(), rolesv1.Permission_PERMISSION_BAN_CREATE) && !ban.TargetID.Equal(curUser.GetSteamID()) {
		return AppealMessage{}, rpc.ErrPermission
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

	if bannedPerson.AppealState != Open && !u.roleAuth.HasPermissionForSteamID(ctx, curUser.GetSteamID(), rolesv1.Permission_PERMISSION_APPEAL_ADMIN) {
		return AppealMessage{}, rpc.ErrPermission
	}

	if errTarget := u.persons.EnsurePerson(ctx, bannedPerson.TargetID); errTarget != nil {
		return AppealMessage{}, errTarget
	}

	if errSource := u.persons.EnsurePerson(ctx, bannedPerson.SourceID); errSource != nil {
		return AppealMessage{}, errSource
	}

	msg := NewBanAppealMessage(banID, curUser.GetSteamID(), newMsg)
	msg.Personaname = curUser.GetName()
	msg.Avatarhash = curUser.GetAvatar().Hash()

	if errSave := u.SaveMessage(ctx, &msg); errSave != nil {
		return AppealMessage{}, errSave
	}

	bannedPerson.UpdatedOn = time.Now()

	if errUpdate := u.bans.Save(ctx, &bannedPerson); errUpdate != nil {
		return AppealMessage{}, errUpdate
	}

	go u.notif.Send(notification.NewDiscord(u.logChannelID, newAppealMessageResponse(msg)))

	go u.notif.Send(notification.NewSiteGroupNotificationWithAuthor(
		[]rolesv1.Permission{rolesv1.Permission_PERMISSION_BAN_READ},
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

func (u *Appeals) Messages(ctx context.Context, userProfile person.BaseUser, banID int32) ([]AppealMessage, error) {
	banPerson, errGetBan := u.bans.QueryOne(ctx, QueryOpts{
		BanID:   banID,
		Deleted: true,
		EvadeOk: true,
	})
	if errGetBan != nil {
		return nil, errGetBan
	}

	if !u.roleAuth.HasPermissionForSteamID(ctx, userProfile.GetSteamID(), rolesv1.Permission_PERMISSION_APPEAL_READ) && !banPerson.TargetID.Equal(userProfile.GetSteamID()) {
		return nil, rpc.ErrPermission
	}

	return u.AppealRepository.Messages(ctx, banID)
}

func (u *Appeals) MessageByID(ctx context.Context, banMessageID int64) (AppealMessage, error) {
	return u.AppealRepository.MessageByID(ctx, banMessageID)
}

func (u *Appeals) DropMessage(ctx context.Context, curUser person.BaseUser, banMessageID int64) error {
	existing, errExist := u.MessageByID(ctx, banMessageID)
	if errExist != nil {
		return errExist
	}

	if !u.roleAuth.HasPermissionForSteamID(ctx, curUser.GetSteamID(), rolesv1.Permission_PERMISSION_APPEAL_ADMIN) && !existing.AuthorID.Equal(curUser.GetSteamID()) {
		return rpc.ErrPermission
	}

	if errDrop := u.AppealRepository.DropMessage(ctx, &existing); errDrop != nil {
		return errDrop
	}

	go u.notif.Send(notification.NewDiscord(u.logChannelID, newAppealMessageDelete(existing)))

	slog.Info("Appeal message deleted", slog.Int64("ban_message_id", banMessageID))

	return nil
}
