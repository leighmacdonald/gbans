package wiki

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal"
	"github.com/leighmacdonald/gbans/internal/auth"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/ptr"
	v1 "github.com/leighmacdonald/gbans/internal/wiki/v1"
	"github.com/leighmacdonald/gbans/internal/wiki/v1/wikiv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	wikiv1connect.UnimplementedWikiServiceHandler

	wiki Wiki
}

func NewService(wiki Wiki) Service {
	return Service{wiki: wiki}
}

func (s Service) Get(ctx context.Context, request *v1.GetRequest) (*v1.GetResponse, error) {
	slug := request.GetSlug()
	page, err := s.wiki.Page(ctx, slug)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrNoResult):
			return nil, connect.NewError(connect.CodeNotFound, httphelper.ErrNotFound)
		case errors.Is(err, permission.ErrDenied):
			return nil, connect.NewError(connect.CodePermissionDenied, permission.ErrDenied)
		default:
			return nil, connect.NewError(connect.CodeInternal, httphelper.ErrInternal)
		}
	}

	if _, errAuth := auth.UserInfoFromCtxWithCheck(ctx, page.PermissionLevel); errAuth != nil {
		return nil, errAuth
	}

	return &v1.GetResponse{
		Wiki: &v1.Wiki{
			Slug:            &page.Slug,
			BodyMd:          &page.BodyMD,
			Revision:        &page.Revision,
			PermissionLevel: ptr.To(internal.Privilege(page.PermissionLevel)),
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
		PermissionLevel: ptr.To(internal.Privilege(updatedPage.PermissionLevel)),
		CreatedOn:       timestamppb.New(updatedPage.CreatedOn),
		UpdatedOn:       timestamppb.New(updatedPage.UpdatedOn),
	}}, nil
}
