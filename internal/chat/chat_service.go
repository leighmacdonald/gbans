package chat

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	v1 "github.com/leighmacdonald/gbans/internal/chat/v1"
	"github.com/leighmacdonald/gbans/internal/chat/v1/chatv1connect"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	chatv1connect.UnimplementedChatServiceHandler

	chat *Chat
}

func NewService(chat *Chat, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := chatv1connect.NewChatServiceHandler(Service{chat: chat}, option...)

	authMiddleware.UserRoute(chatv1connect.ChatServiceQueryProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(chatv1connect.ChatServiceQueryContextProcedure, rpc.WithMinPermissions(permission.User))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) Query(ctx context.Context, req *v1.QueryRequest) (*v1.QueryResponse, error) {
	ctxUser, _ := rpc.UserInfoFromCtx(ctx)

	chatQuery := HistoryQueryFilter{
		Filter:        rpc.FromRPC(req.Filter),
		SourceIDField: httphelper.SourceIDField{SourceID: fmt.Sprintf("%d", req.GetSteamId())},
		Query:         req.GetQuery(),
		Personaname:   "",
		ServerID:      req.GetServerId(),
		DateStart:     ptr.To(req.GetDateStart().AsTime()),
		DateEnd:       ptr.To(req.GetDateEnd().AsTime()),
		Unrestricted:  ctxUser.HasPermission(permission.Moderator),
		DontCalcTotal: false,
		FlaggedOnly:   req.GetFlaggedOnly(),
	}

	messages, count, errChat := s.chat.QueryChatHistory(ctx, ctxUser.Privilege, chatQuery)
	if errChat != nil && !errors.Is(errChat, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, errChat)
	}

	resp := v1.QueryResponse{Messages: make([]*v1.Message, len(messages)), Count: &count}
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
		resp.Messages[idx] = toMessage(msg)
	}

	return &resp, nil
}

func toMessage(msg QueryChatHistoryResult) *v1.Message {
	return &v1.Message{
		PersonMessageId:   &msg.PersonMessageID,
		MatchId:           ptr.To(msg.MatchID.String()),
		SteamId:           ptr.To(msg.SteamID.Int64()),
		AvatarHash:        &msg.AvatarHash,
		PersonaName:       &msg.PersonaName,
		ServerName:        &msg.ServerName,
		ServerId:          &msg.ServerID,
		Body:              &msg.Body,
		Team:              &msg.Team,
		CreatedOn:         timestamppb.New(msg.CreatedOn),
		AutoFilterFlagged: &msg.AutoFilterFlagged,
	}
}
