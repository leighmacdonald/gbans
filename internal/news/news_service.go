package news

import (
	"context"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	v1 "github.com/leighmacdonald/gbans/internal/news/v1"
	"github.com/leighmacdonald/gbans/internal/news/v1/newsv1connect"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	newsv1connect.UnimplementedNewsServiceHandler

	news News
}

func NewService(news News, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := newsv1connect.NewNewsServiceHandler(Service{news: news}, option...)

	authMiddleware.UserRoute(newsv1connect.NewsServiceEditProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.UserRoute(newsv1connect.NewsServiceCreateProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.UserRoute(newsv1connect.NewsServiceDeleteProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.UserRoute(newsv1connect.NewsServiceAllProcedure, rpc.WithMinPermissions(permission.Moderator))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) Latest(ctx context.Context, req *v1.LatestRequest) (*v1.LatestResponse, error) {
	news, errNews := s.news.GetNewsLatest(ctx, req.GetLimit(), false)
	if errNews != nil {
		return nil, connect.NewError(connect.CodeInternal, errNews)
	}

	resp := v1.LatestResponse{Article: make([]*v1.Article, len(news))}
	for idx, entry := range news {
		resp.Article[idx] = &v1.Article{
			NewsId:      &entry.NewsID,
			Title:       &entry.Title,
			BodyMd:      &entry.BodyMD,
			IsPublished: &entry.IsPublished,
			CreatedOn:   timestamppb.New(entry.CreatedOn),
			UpdatedOn:   timestamppb.New(entry.UpdatedOn),
		}
	}

	return &resp, nil
}

func (s Service) Edit(ctx context.Context, req *v1.EditRequest) (*v1.EditResponse, error) {
	entry := &Article{
		NewsID:      req.GetNewsId(),
		Title:       req.GetTitle(),
		BodyMD:      req.GetBodyMd(),
		IsPublished: req.GetIsPublished(),
		CreatedOn:   req.CreatedOn.AsTime(),
		UpdatedOn:   req.CreatedOn.AsTime(),
	}
	if errEntry := s.news.Save(ctx, entry); errEntry != nil {
		return nil, connect.NewError(connect.CodeInternal, errEntry)
	}

	resp := &v1.EditResponse{
		Article: &v1.Article{
			NewsId:      &entry.NewsID,
			Title:       &entry.Title,
			BodyMd:      &entry.BodyMD,
			IsPublished: &entry.IsPublished,
			CreatedOn:   timestamppb.New(entry.CreatedOn),
			UpdatedOn:   timestamppb.New(entry.UpdatedOn),
		},
	}

	return resp, nil
}

func toArticle(entry *Article) *v1.Article {
	return &v1.Article{
		NewsId:      &entry.NewsID,
		Title:       &entry.Title,
		BodyMd:      &entry.BodyMD,
		IsPublished: &entry.IsPublished,
		CreatedOn:   timestamppb.New(entry.CreatedOn),
		UpdatedOn:   timestamppb.New(entry.UpdatedOn),
	}
}

func (s Service) Create(ctx context.Context, req *v1.CreateRequest) (*v1.CreateResponse, error) {
	entry := &Article{
		Title:       req.GetTitle(),
		BodyMD:      req.GetBodyMd(),
		IsPublished: req.GetIsPublished(),
		CreatedOn:   req.CreatedOn.AsTime(),
		UpdatedOn:   req.CreatedOn.AsTime(),
	}
	if errEntry := s.news.Save(ctx, entry); errEntry != nil {
		return nil, connect.NewError(connect.CodeInternal, errEntry)
	}

	resp := &v1.CreateResponse{
		Article: toArticle(entry),
	}

	return resp, nil
}

func (s Service) Delete(ctx context.Context, req *v1.DeleteRequest) (*emptypb.Empty, error) {
	if err := s.news.DropNewsArticle(ctx, req.GetNewsId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return &emptypb.Empty{}, nil
}

func (s Service) All(ctx context.Context, _ *emptypb.Empty) (*v1.AllResponse, error) {
	entries, err := s.news.GetNewsLatest(ctx, 100000, true)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := v1.AllResponse{Articles: make([]*v1.Article, len(entries))}
	for idx, entry := range entries {
		resp.Articles[idx] = toArticle(&entry)
	}

	return &resp, nil
}
