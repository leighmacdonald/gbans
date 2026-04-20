package chat

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	v1 "github.com/leighmacdonald/gbans/internal/chat/v1"
	"github.com/leighmacdonald/gbans/internal/chat/v1/chatv1connect"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type WordfilterService struct {
	chatv1connect.UnimplementedWordfilterServiceHandler

	chat    *Chat
	config  Config
	filters WordFilters
}

func NewWordfilterService(filters WordFilters, chat *Chat, config Config, authMiddleware *rpc.Middleware, options ...connect.HandlerOption) rpc.Service {
	pattern, handler := chatv1connect.NewWordfilterServiceHandler(WordfilterService{filters: filters, chat: chat, config: config}, options...)

	authMiddleware.AuthedRoute(chatv1connect.WordfilterServiceFiltersProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.AuthedRoute(chatv1connect.WordfilterServiceWarningStateProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.AuthedRoute(chatv1connect.WordfilterServiceFilterCreateProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.AuthedRoute(chatv1connect.WordfilterServiceFilterEditProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.AuthedRoute(chatv1connect.WordfilterServiceFilterDeleteProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.AuthedRoute(chatv1connect.WordfilterServiceFilterMatchProcedure, rpc.WithMinPermissions(permission.Moderator))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s WordfilterService) Filters(ctx context.Context, _ *emptypb.Empty) (*v1.FiltersResponse, error) {
	words, errGetFilters := s.filters.GetFilters(ctx)
	if errGetFilters != nil {
		return nil, connect.NewError(connect.CodeInternal, errGetFilters)
	}

	resp := v1.FiltersResponse{Filters: make([]*v1.Filter, len(words))}
	for idx, word := range words {
		resp.Filters[idx] = toFilter(word)
	}

	return &resp, nil
}

func toFilter(filter Filter) *v1.Filter {
	return &v1.Filter{
		FilterId:     &filter.FilterID,
		AuthorId:     ptr.To(filter.AuthorID.Int64()),
		Pattern:      &filter.Pattern,
		IsRegex:      &filter.IsRegex,
		IsEnabled:    &filter.IsEnabled,
		Action:       ptr.To(v1.FilterAction(filter.Action)),
		Duration:     &filter.Duration,
		TriggerCount: &filter.TriggerCount,
		Weight:       &filter.Weight,
		CreatedOn:    timestamppb.New(filter.CreatedOn),
		UpdatedOn:    timestamppb.New(filter.UpdatedOn),
	}
}

func (s WordfilterService) WarningState(_ context.Context, _ *emptypb.Empty) (*v1.WarningStateResponse, error) {
	state := s.chat.WarningState()

	resp := v1.WarningStateResponse{Current: make([]*v1.UserWarning, 0), MaxWeight: &s.config.MaxWeight}
	for _, warnings := range state {
		for _, warn := range warnings {
			resp.Current = append(resp.Current, &v1.UserWarning{
				Reason:       ptr.To(v1.BanReason(warn.WarnReason)),
				Message:      &warn.Message,
				Matched:      &warn.Matched,
				Filter:       toFilter(warn.MatchedFilter),
				CreatedOn:    timestamppb.New(warn.CreatedOn),
				PersonaName:  &warn.Personaname,
				AvatarHash:   &warn.Avatar,
				ServerName:   &warn.ServerName,
				ServerId:     &warn.ServerID,
				SteamId:      &warn.SteamID,
				CurrentTotal: &warn.CurrentTotal,
			})
		}
	}

	return &resp, nil
}

func (s WordfilterService) FilterCreate(ctx context.Context, req *v1.FilterCreateRequest) (*v1.FilterCreateResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)

	reqFilter := req.Filter

	filter := Filter{
		AuthorID:     user.SteamID,
		Pattern:      reqFilter.GetPattern(),
		IsRegex:      reqFilter.GetIsRegex(),
		IsEnabled:    reqFilter.GetIsEnabled(),
		Action:       FilterAction(reqFilter.GetAction()),
		Duration:     reqFilter.GetDuration(),
		TriggerCount: reqFilter.GetTriggerCount(),
		Weight:       reqFilter.GetWeight(),
	}

	wordFilter, errCreate := s.filters.Create(ctx, user.SteamID, filter)
	if errCreate != nil {
		return nil, connect.NewError(connect.CodeInternal, errCreate)
	}

	return &v1.FilterCreateResponse{Filter: toFilter(wordFilter)}, nil
}

func (s WordfilterService) FilterEdit(ctx context.Context, req *v1.FilterEditRequest) (*v1.FilterEditResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)

	reqFilter := req.Filter

	existingFilter, errGet := s.filters.repository.GetFilterByID(ctx, *reqFilter.FilterId)
	if errGet != nil {
		return nil, connect.NewError(connect.CodeInternal, errGet)
	}

	existingFilter.AuthorID = user.SteamID
	existingFilter.UpdatedOn = time.Now()
	existingFilter.Pattern = ptr.From(reqFilter.Pattern)
	existingFilter.IsRegex = ptr.From(reqFilter.IsRegex)
	existingFilter.IsEnabled = ptr.From(reqFilter.IsEnabled)
	existingFilter.Action = FilterAction(ptr.From(reqFilter.Action))
	existingFilter.Duration = ptr.From(reqFilter.Duration)
	existingFilter.Weight = ptr.From(reqFilter.Weight)

	if errSave := s.filters.repository.SaveFilter(ctx, &existingFilter); errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, errSave)
	}

	filter := Filter{
		AuthorID:     user.SteamID,
		Pattern:      reqFilter.GetPattern(),
		IsRegex:      reqFilter.GetIsRegex(),
		IsEnabled:    reqFilter.GetIsEnabled(),
		Action:       FilterAction(reqFilter.GetAction()),
		Duration:     reqFilter.GetDuration(),
		TriggerCount: reqFilter.GetTriggerCount(),
		Weight:       reqFilter.GetWeight(),
	}

	return &v1.FilterEditResponse{Filter: toFilter(filter)}, nil
}

func (s WordfilterService) FilterDelete(ctx context.Context, req *v1.FilterDeleteRequest) (*emptypb.Empty, error) {
	if errDrop := s.filters.DropFilter(ctx, req.GetFilterId()); errDrop != nil {
		return nil, connect.NewError(connect.CodeInternal, errDrop)
	}

	return &emptypb.Empty{}, nil
}

func (s WordfilterService) FilterMatch(_ context.Context, req *v1.FilterMatchRequest) (*v1.FilterMatchResponse, error) {
	matches := s.filters.Check(ptr.From(req.Query))

	resp := v1.FilterMatchResponse{Filters: make([]*v1.Filter, len(matches))}
	for i, m := range matches {
		resp.Filters[i] = toFilter(m)
	}

	return &resp, nil
}
