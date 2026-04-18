package network

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	v1 "github.com/leighmacdonald/gbans/internal/network/v1"
	"github.com/leighmacdonald/gbans/internal/network/v1/networkv1connect"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type BlocklistService struct {
	networkv1connect.UnimplementedBlocklistServiceHandler

	blocklists Blocklists
}

func NewBlocklistService(blocklists Blocklists) BlocklistService {
	return BlocklistService{blocklists: blocklists}
}

func (s BlocklistService) BlocklistSources(ctx context.Context, _ *emptypb.Empty) (*v1.BlocklistSourcesResponse, error) {
	blockLists, err := s.blocklists.GetCIDRBlockSources(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.BlocklistSourcesResponse{BlocklistSource: make([]*v1.CIDRBlockSource, len(blockLists))}
	for idx, source := range blockLists {
		resp.BlocklistSource[idx] = toBlocklistSource(source)
	}

	return &resp, nil
}

func toBlocklistSource(source CIDRBlockSource) *v1.CIDRBlockSource {
	return &v1.CIDRBlockSource{
		CidrBlockSourceId: &source.CIDRBlockSourceID,
		Name:              &source.Name,
		Url:               &source.URL,
		Enabled:           &source.Enabled,
		CreatedOn:         timestamppb.New(source.CreatedOn),
		UpdatedOn:         timestamppb.New(source.UpdatedOn),
	}
}

func (s BlocklistService) BlocklistSourcesCreate(ctx context.Context, req *v1.BlocklistSourcesCreateRequest) (*v1.BlocklistSourcesCreateResponse, error) {
	blockList, errSave := s.blocklists.CreateCIDRBlockSources(ctx, req.GetName(), req.GetUrl(), req.GetEnabled())
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.BlocklistSourcesCreateResponse{BlockSource: toBlocklistSource(blockList)}, nil
}

func (s BlocklistService) BlocklistSourcesEdit(ctx context.Context, req *v1.BlocklistSourcesEditRequest) (*v1.BlocklistSourcesEditResponse, error) {
	blockSource, errUpdate := s.blocklists.UpdateCIDRBlockSource(ctx, req.GetCidrBlockSourceId(), req.GetName(), req.GetUrl(), req.GetEnabled())
	if errUpdate != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.BlocklistSourcesEditResponse{BlockSource: toBlocklistSource(blockSource)}, nil
}

func (s BlocklistService) BlocklistSourcesDelete(ctx context.Context, req *v1.BlocklistSourcesDeleteRequest) (*emptypb.Empty, error) {
	if err := s.blocklists.DeleteCIDRBlockSources(ctx, req.GetCidrBlockSourceId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func (s BlocklistService) WhitelistAddress(ctx context.Context, _ *emptypb.Empty) (*v1.WhitelistAddressResponse, error) {
	whiteLists, errWl := s.blocklists.GetCIDRBlockWhitelists(ctx)
	if errWl != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.WhitelistAddressResponse{Whitelisted: make([]*v1.CIDRBlockWhitelist, len(whiteLists))}
	for idx, whitelist := range whiteLists {
		resp.Whitelisted[idx] = toCIDRBlockWhitelist(whitelist)
	}

	return &resp, nil
}

func (s BlocklistService) WhitelistAddressCreate(ctx context.Context, req *v1.WhitelistAddressCreateRequest) (*v1.WhitelistAddressCreateResponse, error) {
	whitelist, errSave := s.blocklists.CreateCIDRBlockWhitelist(ctx, req.GetAddress())
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.WhitelistAddressCreateResponse{Whitelist: toCIDRBlockWhitelist(whitelist)}, nil
}

func (s BlocklistService) WhitelistAddressDelete(ctx context.Context, req *v1.WhitelistAddressDeleteRequest) (*emptypb.Empty, error) {
	errSave := s.blocklists.DeleteCIDRBlockWhitelist(ctx, req.GetCidrBlockWhitelistId())
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func (s BlocklistService) WhitelistAddressEdit(ctx context.Context, req *v1.WhitelistAddressEditRequest) (*v1.WhitelistAddressEditResponse, error) {
	addr := req.GetAddress()
	if !strings.Contains(addr, "/") {
		addr += maskSingleHost
	}

	whiteList, errSave := s.blocklists.UpdateCIDRBlockWhitelist(ctx, req.GetCidrBlockWhitelistId(), addr)
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.WhitelistAddressEditResponse{Whitelist: toWhitelistIP(whiteList)}, nil
}

func (s BlocklistService) WhitelistSteam(ctx context.Context, _ *emptypb.Empty) (*v1.WhitelistSteamResponse, error) {
	whiteLists, errWl := s.blocklists.GetSteamBlockWhitelists(ctx)
	if errWl != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.WhitelistSteamResponse{Whitelists: make([]*v1.WhitelistSteam, len(whiteLists))}
	for idx, whitelist := range whiteLists {
		resp.Whitelists[idx] = toWhitelistSteam(whitelist)
	}

	return &resp, nil
}

func (s BlocklistService) WhitelistSteamDelete(ctx context.Context, req *v1.WhitelistSteamDeleteRequest) (*emptypb.Empty, error) {
	errSave := s.blocklists.DeleteSteamBlockWhitelists(ctx, steamid.New(req.GetSteamId()))
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func (s BlocklistService) WhitelistSteamCreate(ctx context.Context, req *v1.WhitelistSteamCreateRequest) (*v1.WhitelistSteamCreateResponse, error) {
	steamID := steamid.New(req.GetSteamId())
	if !steamID.Valid() {
		return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
	}

	whitelist, errSave := s.blocklists.CreateSteamBlockWhitelists(ctx, steamID)
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.WhitelistSteamCreateResponse{Whitelist: toWhitelistSteam(whitelist)}, nil
}

func (s BlocklistService) CheckBlock(_ context.Context, _ *v1.CheckBlockRequest) (*v1.CheckBlockResponse, error) {
	return nil, connect.NewError(connect.CodeUnimplemented, rpc.ErrInternal)
}

func toCIDRBlockWhitelist(whitelist WhitelistIP) *v1.CIDRBlockWhitelist {
	return &v1.CIDRBlockWhitelist{
		CidrBlockWhitelistId: &whitelist.CIDRBlockWhitelistID,
		Address:              ptr.To(whitelist.Address.String()),
		CreatedOn:            timestamppb.New(whitelist.CreatedOn),
		UpdatedOn:            timestamppb.New(whitelist.UpdatedOn),
	}
}

func toWhitelistIP(whitelist WhitelistIP) *v1.WhitelistIP {
	return &v1.WhitelistIP{
		CidrBlockWhitelistId: &whitelist.CIDRBlockWhitelistID,
		Address:              ptr.To(whitelist.Address.String()),
		CreatedOn:            timestamppb.New(whitelist.CreatedOn),
		UpdatedOn:            timestamppb.New(whitelist.UpdatedOn),
	}
}

func toWhitelistSteam(whitelist WhitelistSteam) *v1.WhitelistSteam {
	sid := steamid.New(whitelist.SteamIDValue)
	return &v1.WhitelistSteam{
		SteamId:     ptr.To(sid.Int64()),
		PersonaName: &whitelist.Personaname,
		AvatarHash:  &whitelist.AvatarHash,
		CreatedOn:   timestamppb.New(whitelist.CreatedOn),
		UpdatedOn:   timestamppb.New(whitelist.UpdatedOn),
	}
}
