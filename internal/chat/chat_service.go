package chat

import (
	"context"
	"errors"
	"strconv"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	v1 "github.com/leighmacdonald/gbans/internal/chat/v1"
	"github.com/leighmacdonald/gbans/internal/chat/v1/chatv1connect"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	// chatv1connect.UnimplementedChatServiceHandler

	chat *Chat
}

func NewService(chat *Chat, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := chatv1connect.NewChatServiceHandler(Service{chat: chat}, option...)

	authMiddleware.UserRoute(chatv1connect.ChatServiceQueryProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(chatv1connect.ChatServiceQueryContextProcedure, rpc.WithMinPermissions(permission.User))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) Query(ctx context.Context, req *v1.QueryRequest) (*v1.QueryResponse, error) {
	ctxUser := rpc.UserInfoFromCtx(ctx)

	chatQuery := HistoryQueryFilter{
		Filter:        rpc.FromRPC(req.GetFilter()),
		Query:         req.GetQuery(),
		Personaname:   "",
		Unrestricted:  ctxUser.HasPermission(permission.Moderator),
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

	messages, errChat := s.chat.QueryChatHistory(ctx, ctxUser.Privilege, chatQuery)
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

	return &v1.Message{
		PersonMessageId:   &msg.PersonMessageID,
		AssestId:          assetID,
		SteamId:           new(msg.SteamID.Int64()),
		AvatarHash:        &msg.AvatarHash,
		PersonaName:       &msg.PersonaName,
		ServerName:        &msg.ServerName,
		ServerId:          &msg.ServerID,
		MatchId:           new(msg.MatchID.String()),
		DemoId:            msg.DemoID,
		DemoTick:          msg.DemoTick,
		Body:              &msg.Body,
		CreatedOn:         timestamppb.New(msg.CreatedOn),
		AutoFilterFlagged: &msg.AutoFilterFlagged,
	}
}
