package playerqueue

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type QueryOpts struct {
	query.Filter
}

type ChatLog struct {
	MessageID       int64           `json:"message_id"`
	SteamID         steamid.SteamID `json:"steam_id"`
	CreatedOn       time.Time       `json:"created_on"`
	Personaname     string          `json:"personaname"`
	Avatarhash      string          `json:"avatarhash"`
	PermissionLevel int             `json:"permission_level"`
	BodyMD          string          `json:"body_md"`
	Deleted         bool            `json:"deleted"`
}

type Client interface {
	// ID generates a unique identifier for the client connection instance
	ID() string
	// Next handles the incoming operation request
	Next(r *Request) error
	SteamID() steamid.SteamID
	Name() string
	Avatarhash() string
	// Close disconnects the underlying connection
	Close()
	// Start begins the clients response sender worker
	Start(ctx context.Context)
	Send(response Response)
	// HasMessageAccess checks if the user has at least readonly access to chat logs
	HasMessageAccess() bool
	// Limit slows down incoming messages, similar to "slow mode", but much dumber, for now.
	Limit()
}

type ChatStatus string

const (
	Readwrite ChatStatus = "readwrite"
	Readonly  ChatStatus = "readonly"
	Noaccess  ChatStatus = "noaccess"
)

type Op int

const (
	JoinQueue Op = iota
	LeaveQueue
	Message
	StateUpdate
	StartGame
	Purge
	Bye
	ChatStatusChange
)

type Request struct {
	Op      Op              `json:"op"`
	Payload json.RawMessage `json:"payload"`
}

type Response struct {
	Op      Op  `json:"op"`
	Payload any `json:"payload"`
}

type ChatStatusChangePayload struct {
	Status ChatStatus `json:"status"`
	Reason string     `json:"reason"`
}

func NewPlayerqueue(repo Repository, persons person.Provider, serversUC servers.Servers,
	state *servers.State, chatLogs []ChatLog, config *config.Configuration, notif notification.Notifier,
) *Playerqueue {
	return &Playerqueue{
		config:  config,
		repo:    repo,
		persons: persons,
		notif:   notif,
		queue: New(100, 2, chatLogs, func() ([]Lobby, error) {
			currentState := state.Current()

			srvs, errServers := serversUC.Servers(context.Background(), servers.Query{
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

type Playerqueue struct {
	repo    Repository
	persons person.Provider
	notif   notification.Notifier
	config  *config.Configuration
	queue   *Coordinator
}

func (p Playerqueue) Start(ctx context.Context) {
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

func (p Playerqueue) JoinLobbies(client Client, servers []int) error {
	return p.queue.Join(client, servers)
}

func (p Playerqueue) LeaveLobbies(client Client, servers []int) error {
	return p.queue.Leave(client, servers)
}

func (p Playerqueue) Connect(ctx context.Context, user person.Info, conn *websocket.Conn) Client {
	return p.queue.Connect(ctx, user.GetSteamID(), user.GetName(), user.GetAvatar().Hash(), conn)
}

func (p Playerqueue) Disconnect(client Client) {
	p.queue.Disconnect(client)
}

func (p Playerqueue) Purge(ctx context.Context, authorID steamid.SteamID, messageID int64, count int) error {
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

	author, errGetProfile := p.persons.GetOrCreatePersonBySteamID(ctx, authorID)
	if errGetProfile != nil {
		return errGetProfile
	}

	target, errGetTarget := p.persons.GetOrCreatePersonBySteamID(ctx, message.SteamID)
	if errGetTarget != nil {
		return errGetTarget
	}

	p.notif.Send(notification.NewDiscord(p.config.Config().Discord.PlayerqueueChannelID,
		NewPlayerqueuePurge(author, target, message, count)))

	return nil
}

func (p Playerqueue) Message(ctx context.Context, messageID int64) (ChatLog, error) {
	return p.repo.Message(ctx, messageID)
}

func (p Playerqueue) Delete(ctx context.Context, messageID ...int64) error {
	if len(messageID) == 0 {
		return nil
	}

	return p.repo.Delete(ctx, messageID...)
}

func (p Playerqueue) SetChatStatus(ctx context.Context, authorID steamid.SteamID, steamID steamid.SteamID, status ChatStatus, reason string) error {
	if !steamID.Valid() {
		return domain.ErrInvalidSID
	}

	author, errAuthor := p.persons.GetOrCreatePersonBySteamID(ctx, authorID)
	if errAuthor != nil {
		return errAuthor
	}

	person, errPerson := p.persons.GetOrCreatePersonBySteamID(ctx, steamID)
	if errPerson != nil {
		return errPerson
	}

	if author.PermissionLevel <= person.PermissionLevel {
		return permission.ErrDenied
	}

	if errSave := p.repo.SetChatStatus(ctx, person.SteamID, status); errSave != nil {
		return errSave
	}

	p.queue.UpdateChatStatus(steamID, status, reason, Readwrite)

	author, errGetProfile := p.persons.GetOrCreatePersonBySteamID(ctx, authorID)
	if errGetProfile != nil {
		return errGetProfile
	}

	p.notif.Send(notification.NewDiscord(p.config.Config().Discord.PlayerqueueChannelID, NewPlayerqueueChatStatus(author, person, status, reason)))

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

func (p Playerqueue) AddMessage(ctx context.Context, bodyMD string, user person.Info) error {
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

func (p Playerqueue) Recent(ctx context.Context, limit uint64) ([]ChatLog, error) {
	if limit == 0 {
		limit = 50
	}

	return p.repo.Query(ctx, QueryOpts{
		Filter: query.Filter{
			Limit:   limit,
			Desc:    true,
			OrderBy: "message_id",
			Deleted: false,
		},
	})
}
