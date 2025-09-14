package playerqueue

import (
	"context"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

func NewPlayerqueueUsecase(repo PlayerqueueRepository, persons person.PersonUsecase, serversUC servers.ServersUsecase,
	state *servers.StateUsecase, chatLogs []ChatLog,
) *PlayerqueueUsecase {
	return &PlayerqueueUsecase{
		repo:    repo,
		persons: persons,
		queue: New(100, 2, chatLogs, func() ([]Lobby, error) {
			currentState := state.Current()

			srvs, _, errServers := serversUC.Servers(context.Background(), servers.ServerQueryFilter{
				QueryFilter:     domain.QueryFilter{},
				IncludeDisabled: false,
			})

			if errServers != nil {
				return nil, errServers
			}

			var lobbies []Lobby
			for _, srv := range srvs {
				lobby := Lobby{ServerID: srv.ServerID}
				for _, serverState := range currentState {
					if serverState.ServerID == lobby.ServerID {
						lobby.Hostname = serverState.Host
						lobby.Port = serverState.Port
						lobby.ShortName = serverState.NameShort
						lobby.Title = serverState.Name
						lobby.CC = serverState.CC
						lobby.MaxPlayers = serverState.MaxPlayers
						lobby.PlayerCount = serverState.PlayerCount
					}
				}

				lobbies = append(lobbies, lobby)
			}

			return lobbies, nil
		}),
	}
}

type PlayerqueueUsecase struct {
	repo     PlayerqueueRepository
	perssons person.PersonUsecase
	notif    notification.NotificationUsecase
	queue    *Coordinator
}

func (p PlayerqueueUsecase) Start(ctx context.Context) {
	refreshState := time.NewTicker(time.Second * 2)

	p.queue.updateState()

	for {
		select {
		case <-refreshState.C:
			p.queue.updateState()
		case <-ctx.Done():
			p.queue.broadcast(Response{Op: Bye, Payload: ByePayload{Message: "Server shutting down... run!!!"}})

			return
		}
	}
}

func (p PlayerqueueUsecase) JoinLobbies(client QueueClient, servers []int) error {
	return p.queue.Join(client, servers)
}

func (p PlayerqueueUsecase) LeaveLobbies(client QueueClient, servers []int) error {
	return p.queue.Leave(client, servers)
}

func (p PlayerqueueUsecase) Connect(ctx context.Context, user domain.PersonInfo, conn *websocket.Conn) QueueClient {
	return p.queue.Connect(ctx, user.GetSteamID(), user.GetName(), user.GetAvatar().Hash(), conn)
}

func (p PlayerqueueUsecase) Disconnect(client QueueClient) {
	p.queue.Disconnect(client)
}

func (p PlayerqueueUsecase) Purge(ctx context.Context, authorID steamid.SteamID, messageID int64, count int) error {
	message, errMessage := p.repo.Message(ctx, messageID)
	if errMessage != nil {
		return errMessage
	}

	var messageIDs []int64 //nolint:prealloc
	for _, msg := range p.queue.FindMessages(message.SteamID, count) {
		messageIDs = append(messageIDs, msg.MessageID)
	}

	if errDelete := p.repo.Delete(ctx, messageIDs...); errDelete != nil {
		return errDelete
	}

	p.queue.PurgeMessages(messageIDs...)

	// author, errGetProfile := p.persons.GetOrCreatePersonBySteamID(ctx, nil, authorID)
	// if errGetProfile != nil {
	// 	return errGetProfile
	// }

	// target, errGetTarget := p.persons.GetOrCreatePersonBySteamID(ctx, nil, message.SteamID)
	// if errGetTarget != nil {
	// 	return errGetTarget
	// }

	// p.notif.Enqueue(ctx, notification.NewDiscordNotification(
	// 	discord.ChannelPlayerqueue, discord.NewPlayerqueuePurge(author.ToUserProfile(), target.ToUserProfile(), message, count)))

	return nil
}

func (p PlayerqueueUsecase) Message(ctx context.Context, messageID int64) (ChatLog, error) {
	return p.repo.Message(ctx, messageID)
}

func (p PlayerqueueUsecase) Delete(ctx context.Context, messageID ...int64) error {
	if len(messageID) == 0 {
		return nil
	}

	return p.repo.Delete(ctx, messageID...)
}

func (p PlayerqueueUsecase) SetChatStatus(ctx context.Context, authorID steamid.SteamID, steamID steamid.SteamID, status ChatStatus, reason string) error {
	if !steamID.Valid() {
		return domain.ErrInvalidSID
	}

	person, errPerson := p.persons.GetOrCreatePersonBySteamID(ctx, nil, steamID)
	if errPerson != nil {
		return errPerson
	}

	allowed, errAlter := p.persons.CanAlter(ctx, authorID, person.SteamID)
	if errAlter != nil {
		return errAlter
	}

	if !allowed {
		return permission.ErrPermissionDenied
	}

	if person.PlayerqueueChatStatus == status {
		return database.ErrDuplicate
	}

	previousStatus := person.PlayerqueueChatStatus
	person.PlayerqueueChatStatus = status

	if errSave := p.persons.SavePerson(ctx, nil, &person); errSave != nil {
		return errSave
	}

	p.queue.UpdateChatStatus(steamID, status, reason, previousStatus)

	// author, errGetProfile := p.persons.GetOrCreatePersonBySteamID(ctx, nil, authorID)
	// if errGetProfile != nil {
	// 	return errGetProfile
	// }
	//
	// p.notif.Enqueue(ctx, notification.NewDiscordNotification(
	// 	discord.ChannelPlayerqueue, discord.NewPlayerqueueChatStatus(author.ToUserProfile(), person.ToUserProfile(), status, reason)))

	slog.Info("Set chat status", slog.String("steam_id", person.SteamID.String()), slog.String("status", string(status)))

	return nil
}

func sanitizeUserMessage(msg string) string {
	s := removeNonPrintable(strings.TrimSpace(msg))
	s = stringutil.SanitizeUGC(s)
	// TODO 1984
	return s
}

func removeNonPrintable(input string) string {
	out := strings.Map(func(r rune) rune {
		if unicode.IsGraphic(r) && unicode.IsPrint(r) || r == ' ' {
			return r
		}

		return -1
	}, input)

	return out
}

func (p PlayerqueueUsecase) AddMessage(ctx context.Context, bodyMD string, user domain.PersonInfo) error {
	bodyMD = sanitizeUserMessage(bodyMD)
	if len(bodyMD) == 0 {
		return ErrBadInput
	}

	sid := user.GetSteamID()
	if !sid.Valid() {
		return ErrBadInput
	}

	newMessage := ChatLog{
		SteamID:         user.GetSteamID(),
		CreatedOn:       time.Now(),
		Personaname:     user.GetName(),
		Avatarhash:      user.GetAvatar().Hash(),
		PermissionLevel: int(user.Permissions()),
		BodyMD:          bodyMD,
		Deleted:         false,
	}

	message, err := p.repo.Save(ctx, newMessage)
	if err != nil {
		return err
	}

	p.queue.Message(message)

	// p.notif.Enqueue(ctx,
	// 	notification.NewDiscordNotification(discord.ChannelPlayerqueue, discord.NewPlayerqueueMessage(user, bodyMD)))

	return nil
}

func (p PlayerqueueUsecase) Recent(ctx context.Context, limit uint64) ([]ChatLog, error) {
	if limit == 0 {
		limit = 50
	}

	return p.repo.Query(ctx, PlayerqueueQueryOpts{
		QueryFilter: domain.QueryFilter{
			Limit:   limit,
			Desc:    true,
			OrderBy: "message_id",
			Deleted: false,
		},
	})
}
