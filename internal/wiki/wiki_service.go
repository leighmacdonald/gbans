package wiki

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/database"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	v1 "github.com/leighmacdonald/gbans/internal/wiki/v1"
	"github.com/leighmacdonald/gbans/internal/wiki/v1/wikiv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	// wikiv1connect.UnimplementedWikiServiceHandler

	wiki     Wiki
	roleAuth *rpc.RoleAuth
}

func NewService(wiki Wiki, roleAuth *rpc.RoleAuth, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := wikiv1connect.NewWikiServiceHandler(Service{wiki: wiki, roleAuth: roleAuth}, option...)

	authMiddleware.PublicRoute(wikiv1connect.WikiServiceGetProcedure)
	authMiddleware.UserRoute(wikiv1connect.WikiServiceUpdateProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_WIKI_EDIT))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) Get(ctx context.Context, request *v1.GetRequest) (*v1.GetResponse, error) {
	slug := request.GetSlug()
	page, err := s.wiki.Page(ctx, slug)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNoResult):
			return &v1.GetResponse{
				Wiki: &v1.Wiki{
					Slug:               &slug,
					BodyMd:             new(fmt.Sprintf("# New %s Wiki", slug)),
					Revision:           new(int32(0)),
					RequiredPermission: new(rolesv1.Permission(rolesv1.Permission_PERMISSION_UNSPECIFIED)),
					CreatedOn:          timestamppb.Now(),
					UpdatedOn:          timestamppb.Now(),
				},
			}, nil
		default:
			return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
		}
	}

	if page.RequiredPermission != rolesv1.Permission_PERMISSION_UNSPECIFIED {
		user := rpc.UserInfoFromCtx(ctx)
		if !s.roleAuth.HasPermission(ctx, *user, page.RequiredPermission) {
			return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
		}
	}

	return &v1.GetResponse{
		Wiki: &v1.Wiki{
			Slug:               &page.Slug,
			BodyMd:             &page.BodyMD,
			Revision:           &page.Revision,
			RequiredPermission: new(rolesv1.Permission(page.RequiredPermission)),
			CreatedOn:          timestamppb.New(page.CreatedOn),
			UpdatedOn:          timestamppb.New(page.UpdatedOn),
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
	page.RequiredPermission = update.GetRequiredPermission()

	updatedPage, errSave := s.wiki.Save(ctx, page)
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, errSave)
	}

	return &v1.UpdateResponse{Wiki: &v1.Wiki{
		Slug:               &updatedPage.Slug,
		BodyMd:             &updatedPage.BodyMD,
		Revision:           &updatedPage.Revision,
		RequiredPermission: new(rolesv1.Permission(updatedPage.RequiredPermission)),
		CreatedOn:          timestamppb.New(updatedPage.CreatedOn),
		UpdatedOn:          timestamppb.New(updatedPage.UpdatedOn),
	}}, nil
}
