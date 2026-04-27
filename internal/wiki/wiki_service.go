package wiki

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	personv1 "github.com/leighmacdonald/gbans/internal/person/v1"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	v1 "github.com/leighmacdonald/gbans/internal/wiki/v1"
	"github.com/leighmacdonald/gbans/internal/wiki/v1/wikiv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	wikiv1connect.UnimplementedWikiServiceHandler

	wiki Wiki
}

func NewService(wiki Wiki, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := wikiv1connect.NewWikiServiceHandler(Service{wiki: wiki}, option...)

	authMiddleware.AuthedRoute(wikiv1connect.WikiServiceGetProcedure, rpc.WithMinPermissions(permission.Guest))
	authMiddleware.AuthedRoute(wikiv1connect.WikiServiceUpdateProcedure, rpc.WithMinPermissions(permission.Moderator))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) Get(ctx context.Context, request *v1.GetRequest) (*v1.GetResponse, error) {
	slug := request.GetSlug()
	page, err := s.wiki.Page(ctx, slug)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNoResult):
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		case errors.Is(err, permission.ErrDenied):
			return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
		default:
			return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
		}
	}

	if _, errAuth := rpc.UserInfoFromCtxWithCheck(ctx, page.PermissionLevel); errAuth != nil {
		return nil, errAuth
	}

	return &v1.GetResponse{
		Wiki: &v1.Wiki{
			Slug:            &page.Slug,
			BodyMd:          &page.BodyMD,
			Revision:        &page.Revision,
			PermissionLevel: ptr.To(personv1.Privilege(page.PermissionLevel)),
			CreatedOn:       timestamppb.New(page.CreatedOn),
			UpdatedOn:       timestamppb.New(page.UpdatedOn),
		},
	}, nil
}

func (s Service) Update(ctx context.Context, request *v1.UpdateRequest) (*v1.UpdateResponse, error) {
	update := request.GetWiki()
	page, err := s.wiki.Page(ctx, update.GetSlug())
	if err != nil && errors.Is(err, ErrSlugUnknown) {
		return nil, connect.NewError(connect.CodeInternal, ErrSlugUnknown)
	}

	page.Slug = update.GetSlug()
	page.BodyMD = update.GetBodyMd()
	page.PermissionLevel = permission.Privilege(update.GetPermissionLevel())

	updatedPage, errSave := s.wiki.Save(ctx, page)
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, errSave)
	}

	return &v1.UpdateResponse{Wiki: &v1.Wiki{
		Slug:            &updatedPage.Slug,
		BodyMd:          &updatedPage.BodyMD,
		Revision:        &updatedPage.Revision,
		PermissionLevel: ptr.To(personv1.Privilege(updatedPage.PermissionLevel)),
		CreatedOn:       timestamppb.New(updatedPage.CreatedOn),
		UpdatedOn:       timestamppb.New(updatedPage.UpdatedOn),
	}}, nil
}
