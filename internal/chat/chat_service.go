package chat

import (
	"context"
	"errors"
	"strconv"

	"connectrpc.com/connect"
	v1 "github.com/leighmacdonald/gbans/internal/chat/v1"
	"github.com/leighmacdonald/gbans/internal/chat/v1/chatv1connect"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	// chatv1connect.UnimplementedChatServiceHandler

	chat     *Chat
	roleAuth *rpc.RoleAuth
}

func NewService(chat *Chat, roleAuth *rpc.RoleAuth, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := chatv1connect.NewChatServiceHandler(Service{chat: chat, roleAuth: roleAuth}, option...)

	authMiddleware.UserRoute(chatv1connect.ChatServiceQueryProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_CHATLOG_READ))
	authMiddleware.UserRoute(chatv1connect.ChatServiceQueryContextProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_CHATLOG_READ))

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

	return &v1.Message{
		PersonMessageId:   &msg.PersonMessageID,
		AssetId:           assetID,
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
