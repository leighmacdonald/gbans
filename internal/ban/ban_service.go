package ban

import (
	"context"
	"errors"
	"strconv"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	v1 "github.com/leighmacdonald/gbans/internal/ban/v1"
	"github.com/leighmacdonald/gbans/internal/ban/v1/banv1connect"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	banv1connect.UnimplementedBanServiceHandler

	client thirdparty.ClientWithResponsesInterface
	bans   Bans
}

func NewBanService(bans Bans, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	client, errClient := thirdparty.NewClientWithResponses("https://tf-api.roto.lol")
	if errClient != nil {
		panic(errClient)
	}

	pattern, handler := banv1connect.NewBanServiceHandler(Service{bans: bans, client: client}, option...)

	authMiddleware.UserRoute(banv1connect.BanServiceQueryProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.UserRoute(banv1connect.BanServiceDeleteProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.UserRoute(banv1connect.BanServiceGetProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(banv1connect.BanServiceQuerySourceBansProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.UserRoute(banv1connect.BanServiceUpdateProcedure, rpc.WithMinPermissions(permission.Moderator))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) Query(ctx context.Context, req *v1.QueryRequest) (*v1.QueryResponse, error) {
	reasons := make([]reason.Reason, len(req.GetReason()))
	for idx, reqReason := range req.GetReason() {
		reasons[idx] = reason.Reason(reqReason)
	}

	opts := QueryOpts{
		Deleted:       req.GetDeleted(),
		GroupsOnly:    req.GetGroupsOnly(),
		CIDR:          req.GetCidr(),
		CIDROnly:      req.GetCidrOnly(),
		Reasons:       reasons,
		IncludeGroups: req.GetGroupsOnly(),
	}

	if req.SourceId != nil {
		opts.SourceID = steamid.New(req.GetSourceId())
	}

	if req.TargetId != nil {
		opts.TargetID = steamid.New(req.GetTargetId())
	}

	bans, errBans := s.bans.Query(ctx, opts)
	if errBans != nil {
		return nil, connect.NewError(connect.CodeInternal, errBans)
	}

	resp := &v1.QueryResponse{Bans: make([]*v1.Ban, len(bans))}
	for idx, ban := range bans {
		resp.Bans[idx] = toBan(ban)
	}

	return resp, nil
}

func (s Service) Delete(ctx context.Context, req *v1.DeleteRequest) (*emptypb.Empty, error) {
	bannedPerson, errBan := s.bans.QueryOne(ctx, QueryOpts{BanID: ptr.From(req.BanId), EvadeOk: true})
	if errBan != nil {
		if errors.Is(errBan, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	user := rpc.UserInfoFromCtx(ctx)
	changed, errSave := s.bans.Unban(ctx, bannedPerson.TargetID, ptr.From(req.Reason), user)
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, errSave)
	}

	if !changed {
		return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
	}

	return &emptypb.Empty{}, nil
}

func (s Service) Get(ctx context.Context, req *v1.GetRequest) (*v1.GetResponse, error) {
	user := rpc.UserInfoFromCtx(ctx)

	bannedPerson, errGet := s.bans.QueryOne(ctx, QueryOpts{BanID: req.GetBanId(), Deleted: false, EvadeOk: true})
	if errGet != nil {
		if errors.Is(errGet, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	if !user.HasPermission(permission.Moderator) && !bannedPerson.TargetID.Equal(user.GetSteamID()) {
		return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
	}

	return &v1.GetResponse{Ban: toBan(bannedPerson)}, nil
}

func (s Service) QuerySourceBans(ctx context.Context, req *v1.QuerySourceBansRequest) (*v1.QuerySourceBansResponse, error) {
	sid := req.GetSteamId()

	queryResp, errResp := s.client.BansSearchWithResponse(ctx, &thirdparty.BansSearchParams{Steamids: strconv.FormatInt(sid, 10)})
	if errResp != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}
	if queryResp.JSON200 == nil {
		return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
	}

	resp := v1.QuerySourceBansResponse{Bans: make([]*v1.SourceBanRecord, len(*queryResp.JSON200))}

	for idx, ban := range *queryResp.JSON200 {
		sid := steamid.New(ban.SteamId)
		resp.Bans[idx] = &v1.SourceBanRecord{
			SiteName:    &ban.SiteName,
			PersonaName: &ban.Name,
			SteamId:     new(sid.Int64()),
			Reason:      &ban.Reason,
			Duration:    durationpb.New(ban.ExpiresOn.Sub(ban.CreatedOn)),
			Permanent:   &ban.Permanent,
			CreatedOn:   timestamppb.New(ban.CreatedOn),
		}
	}

	return &resp, nil
}

func (s Service) Update(ctx context.Context, req *v1.UpdateRequest) (*v1.UpdateResponse, error) {
	bannedPerson, banErr := s.bans.QueryOne(ctx, QueryOpts{BanID: req.GetBanId(), Deleted: true, EvadeOk: true})
	if banErr != nil {
		return nil, connect.NewError(connect.CodeNotFound, banErr)
	}

	if reason.Reason(req.GetReason()) == reason.Custom {
		if req.GetReasonText() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
		}

		bannedPerson.ReasonText = req.GetReasonText()
	} else {
		bannedPerson.ReasonText = ""
	}

	if validUntil := req.GetValidUntil(); validUntil.IsValid() && !validUntil.AsTime().IsZero() {
		bannedPerson.ValidUntil = validUntil.AsTime()
	}

	bannedPerson.AppealState = AppealState(req.GetAppealState())
	bannedPerson.Note = req.GetNote()
	bannedPerson.BanType = bantype.Type(req.GetBanType())
	bannedPerson.Reason = reason.Reason(req.GetReason())
	bannedPerson.EvadeOk = req.GetEvadeOk()

	if cidr := req.GetCidr(); cidr != "" {
		bannedPerson.CIDR = &cidr
	}

	if errSave := s.bans.Save(ctx, &bannedPerson); errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, ErrSaveBan)
	}

	return &v1.UpdateResponse{Ban: toBan(bannedPerson)}, nil
}

func toBan(ban Ban) *v1.Ban {
	return &v1.Ban{
		TargetId:          new(ban.TargetID.Int64()),
		SourceId:          new(ban.SourceID.Int64()),
		BanId:             &ban.BanID,
		ReportId:          &ban.ReportID,
		LastIp:            ban.LastIP,
		EvadeOk:           &ban.EvadeOk,
		BanType:           new(v1.BanType(ban.BanType)),  //nolint:gosec
		Reason:            new(v1.BanReason(ban.Reason)), //nolint:gosec
		ReasonText:        &ban.ReasonText,
		UnbanReasonText:   &ban.UnbanReasonText,
		Note:              &ban.Note,
		Origin:            new(v1.Origin(ban.Origin)), //nolint:gosec
		Cidr:              ban.CIDR,
		AppealState:       new(v1.AppealState(ban.AppealState)), //nolint:gosec
		Name:              &ban.Name,
		Deleted:           &ban.Deleted,
		IsEnabled:         &ban.IsEnabled,
		ValidUntil:        timestamppb.New(ban.ValidUntil),
		CreatedOn:         timestamppb.New(ban.CreatedOn),
		UpdatedOn:         timestamppb.New(ban.UpdatedOn),
		SourcePersonaName: &ban.SourcePersonaname,
		SourceAvatarHash:  &ban.SourceAvatarhash,
		TargetPersonaName: &ban.TargetPersonaname,
		TargetAvatarHash:  &ban.TargetAvatarhash,
	}
}
