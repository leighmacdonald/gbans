package chat

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/leighmacdonald/gbans/internal/chat/v1"
	"github.com/leighmacdonald/gbans/internal/chat/v1/chatv1connect"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/opengraph"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	// chatv1connect.UnimplementedChatServiceHandler

	chat     *Chat
	roleAuth *rpc.RoleAuth
	// keepAliveInterval controls how often an empty keepalive event is sent while
	// no chat messages are flowing. It must stay comfortably below the idle
	// timeouts of intermediaries (Cloudflare closes idle proxied streams).
	keepAliveInterval time.Duration
}

// defaultStreamKeepAliveInterval is well under Cloudflare's ~100s idle cut-off
// for proxied streaming responses.
const defaultStreamKeepAliveInterval = 20 * time.Second

func NewService(chat *Chat, openGraph *opengraph.Repository, roleAuth *rpc.RoleAuth, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := chatv1connect.NewChatServiceHandler(Service{
		chat:              chat,
		roleAuth:          roleAuth,
		keepAliveInterval: defaultStreamKeepAliveInterval,
	}, option...)

	openGraph.Register("/chatlogs", opengraph.Static("Chat Logs", "Browse in-game chat logs."))

	authMiddleware.UserRoute(chatv1connect.ChatServiceQueryProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_CHATLOG_READ))
	authMiddleware.UserRoute(chatv1connect.ChatServiceQueryContextProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_CHATLOG_READ))
	authMiddleware.UserRoute(chatv1connect.ChatServiceStreamChatProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_CHATLOG_READ))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) Query(ctx context.Context, req *v1.QueryRequest) (*v1.QueryResponse, error) {
	ctxUser := rpc.UserInfoFromCtx(ctx)
	moderator := s.roleAuth.HasPermission(ctx, *ctxUser, rolesv1.Permission_PERMISSION_BAN_READ)

	chatQuery := HistoryQueryFilter{
		Filter:        rpc.FromRPC(req.GetFilter()),
		Query:         req.GetQuery(),
		Personaname:   "",
		Unrestricted:  moderator,
		DontCalcTotal: false,
		FlaggedOnly:   req.GetFlaggedOnly(),
	}

	chatQuery.ServerIDs = req.GetServerIds()

	if dateStart := req.GetDateStart(); dateStart.IsValid() {
		chatQuery.DateStart = new(req.GetDateStart().AsTime())
	}

	if dateEnd := req.GetDateEnd(); dateEnd.IsValid() {
		chatQuery.DateEnd = new(req.GetDateEnd().AsTime())
	}

	if steamID := req.GetSteamId(); steamID > 0 {
		chatQuery.SourceIDField = httphelper.SourceIDField{SourceID: strconv.FormatInt(steamID, 10)}
	}

	messages, errChat := s.chat.QueryChatHistory(ctx, moderator, chatQuery)
	if errChat != nil && !errors.Is(errChat, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, errChat)
	}

	resp := v1.QueryResponse{Messages: make([]*v1.Message, len(messages))}
	for idx, msg := range messages {
		resp.Messages[idx] = toMessage(msg)
	}

	return &resp, nil
}

func (s Service) QueryContext(ctx context.Context, req *v1.QueryContextRequest) (*v1.QueryContextResponse, error) {
	messages, errQuery := s.chat.GetPersonMessageContext(ctx, req.GetPersonMessageId(), req.GetPadding())
	if errQuery != nil {
		return nil, connect.NewError(connect.CodeInternal, errQuery)
	}

	resp := v1.QueryContextResponse{Messages: make([]*v1.Message, len(messages))}
	for idx, msg := range messages {
		resp.Messages[idx] = toMessage(&msg)
	}

	return &resp, nil
}

func toMessage(msg *QueryChatHistoryResult) *v1.Message {
	var assetID *string
	if !msg.AssetID.IsNil() {
		assetID = new(msg.AssetID.String())
	}

	var matchID *string
	if msg.MatchID != nil {
		matchID = new(msg.MatchID.String())
	}

	return &v1.Message{
		PersonMessageId:   &msg.PersonMessageID,
		AssetId:           assetID,
		SteamId:           new(msg.SteamID.Int64()),
		AvatarHash:        &msg.AvatarHash,
		PersonaName:       &msg.PersonaName,
		ServerName:        &msg.ServerName,
		ServerId:          &msg.ServerID,
		MatchId:           matchID,
		DemoId:            msg.DemoID,
		DemoTick:          msg.DemoTick,
		Body:              &msg.Body,
		CreatedOn:         timestamppb.New(msg.CreatedOn),
		AutoFilterFlagged: &msg.AutoFilterFlagged,
	}
}

// StreamChat sends the buffered chat history and then streams live chat
// messages from all servers, or the requested subset, until the client
// disconnects. Keepalive pings are interleaved so that proxies (Cloudflare in
// particular) do not close the stream when chat is quiet.
func (s Service) StreamChat(ctx context.Context, req *v1.StreamChatRequest, stream *connect.ServerStream[v1.StreamChatResponse]) error {
	buffer := s.chat.buffer
	sub := make(chan BufferedChatMessage, 128)
	buffer.Consume(sub)
	defer buffer.Unregister(sub)

	// The snapshot is taken after subscribing, so messages pushed in between are
	// present in both. The seq watermark dedupes them and keeps the stream in
	// order across the history/live boundary.
	snap := buffer.Snapshot()

	var watermark uint64
	if len(snap) > 0 {
		watermark = snap[len(snap)-1].Seq
	}
	if limit := int(req.GetHistoryLimit()); limit > 0 && len(snap) > limit {
		snap = snap[len(snap)-limit:]
	}

	serverIDs := req.GetServerIds()

	send := func(msg BufferedChatMessage) error {
		if errSend := stream.Send(msg.toStreamChatResponse()); errSend != nil {
			return fmt.Errorf("send chat message: %w", errSend)
		}

		return nil
	}

	for _, msg := range snap {
		if !messageInServers(serverIDs, msg.ServerID) {
			continue
		}

		if errSend := send(msg); errSend != nil {
			return errSend
		}
	}

	keepAliveInterval := s.keepAliveInterval
	if keepAliveInterval <= 0 {
		keepAliveInterval = defaultStreamKeepAliveInterval
	}

	keepAlive := time.NewTicker(keepAliveInterval)
	defer keepAlive.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-keepAlive.C:
			if errSend := stream.Send(&v1.StreamChatResponse{Keepalive: new(true)}); errSend != nil {
				return fmt.Errorf("send keepalive: %w", errSend)
			}
		case msg, ok := <-sub:
			if !ok {
				return nil
			}
			if msg.Seq <= watermark {
				continue
			}
			watermark = msg.Seq
			if !messageInServers(serverIDs, msg.ServerID) {
				continue
			}

			if errSend := send(msg); errSend != nil {
				return errSend
			}
		}
	}
}

// messageInServers returns true when the server filter is empty (all servers)
// or when the message's server is part of the requested subset.
func messageInServers(serverIDs []int32, serverID int32) bool {
	if len(serverIDs) == 0 {
		return true
	}

	return slices.Contains(serverIDs, serverID)
}

func (b BufferedChatMessage) toStreamChatResponse() *v1.StreamChatResponse {
	steamID := b.SteamID.Int64()

	return &v1.StreamChatResponse{
		Seq:         new(b.Seq),
		SteamId:     new(steamID),
		AvatarHash:  &b.AvatarHash,
		PersonaName: &b.PersonaName,
		ServerId:    &b.ServerID,
		ServerName:  &b.ServerName,
		Body:        &b.Body,
		IsTeam:      &b.Team,
		CreatedOn:   timestamppb.New(b.CreatedOn),
	}
}
