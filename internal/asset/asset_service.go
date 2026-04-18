package asset

import (
	"bytes"
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/gofrs/uuid/v5"
	v1 "github.com/leighmacdonald/gbans/internal/asset/v1"
	"github.com/leighmacdonald/gbans/internal/asset/v1/assetv1connect"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	assetv1connect.UnimplementedAssetServiceHandler

	assets Assets
}

func NewService(assets Assets) Service {
	return Service{assets: assets}
}

func (s Service) Create(ctx context.Context, req *v1.CreateRequest) (*v1.CreateResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	asset, errAsset := s.assets.Create(ctx, user.SteamID, "media", req.GetName(), bytes.NewReader(req.GetContents()), false)
	if errAsset != nil {
		return nil, connect.NewError(connect.CodeInternal, errAsset)
	}

	return &v1.CreateResponse{Asset: &v1.Asset{
		AssetId:   ptr.To(asset.AssetID.String()),
		Bucket:    ptr.To(string(asset.Bucket)),
		AuthorId:  ptr.To(asset.AuthorID.Int64()),
		Hash:      ptr.To(fmt.Sprintf("%x", asset.Hash)),
		IsPrivate: &asset.IsPrivate,
		MimeType:  &asset.MimeType,
		Name:      &asset.Name,
		Size:      &asset.Size,
		CreatedOn: timestamppb.New(asset.CreatedOn),
		UpdatedOn: timestamppb.New(asset.UpdatedOn),
	}}, nil
}

func (s Service) Delete(ctx context.Context, req *v1.DeleteRequest) (*v1.DeleteResponse, error) {
	id, errID := uuid.FromString(req.GetAssetId())
	if errID != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errID)
	}

	size, errDelete := s.assets.Delete(ctx, id)
	if errDelete != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errID)
	}

	return &v1.DeleteResponse{Size: &size}, nil
}
