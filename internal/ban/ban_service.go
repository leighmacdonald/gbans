package ban

import (
	"context"
	"errors"
	"fmt"
	"time"

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

type BanService struct {
	banv1connect.UnimplementedBanServiceHandler

	client thirdparty.ClientWithResponsesInterface
	bans   Bans
}

func NewBanService(bans Bans, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	client, errClient := thirdparty.NewClientWithResponses("https://tf-api.roto.lol")
	if errClient != nil {
		panic(errClient)
	}

	pattern, handler := banv1connect.NewBanServiceHandler(BanService{bans: bans, client: client}, option...)

	authMiddleware.UserRoute(banv1connect.BanServiceQueryProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.UserRoute(banv1connect.BanServiceDeleteProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.UserRoute(banv1connect.BanServiceGetProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(banv1connect.BanServiceQuerySourceBansProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.UserRoute(banv1connect.BanServiceUpdateProcedure, rpc.WithMinPermissions(permission.Moderator))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s BanService) Query(ctx context.Context, req *v1.QueryRequest) (*v1.QueryResponse, error) {
	var reasons []reason.Reason
	for _, reqReason := range req.GetReason() {
		reasons = append(reasons, reason.Reason(reqReason))
	}

	bans, errBans := s.bans.Query(ctx, QueryOpts{
		Deleted:       req.GetDeleted(),
		SourceID:      steamid.New(req.GetSourceId()),
		TargetID:      steamid.New(req.GetTargetId()),
		GroupsOnly:    req.GetGroupsOnly(),
		CIDR:          req.GetCidr(),
		CIDROnly:      req.GetCidrOnly(),
		Reasons:       reasons,
		IncludeGroups: req.GetGroupsOnly(),
	})
	if errBans != nil {
		return nil, connect.NewError(connect.CodeInternal, errBans)
	}

	resp := &v1.QueryResponse{Bans: make([]*v1.Ban, len(bans))}
	for idx, ban := range bans {
		resp.Bans[idx] = toBan(ban)
	}

	return resp, nil
}

func (s BanService) Delete(ctx context.Context, req *v1.DeleteRequest) (*emptypb.Empty, error) {
	bannedPerson, errBan := s.bans.QueryOne(ctx, QueryOpts{BanID: ptr.From(req.BanId), EvadeOk: true})
	if errBan != nil {
		if errors.Is(errBan, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	user, _ := rpc.UserInfoFromCtx(ctx)
	changed, errSave := s.bans.Unban(ctx, bannedPerson.TargetID, ptr.From(req.Reason), user)
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, errSave)
	}

	if !changed {
		return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
	}

	return &emptypb.Empty{}, nil
}

func (s BanService) Get(ctx context.Context, req *v1.GetRequest) (*v1.GetResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)

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

func (s BanService) QuerySourceBans(ctx context.Context, req *v1.QuerySourceBansRequest) (*v1.QuerySourceBansResponse, error) {
	sid := req.GetSteamId()
	queryResp, errResp := s.client.BansSearchWithResponse(ctx, &thirdparty.BansSearchParams{Steamids: fmt.Sprintf("%d", sid)})
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
			SteamId:     ptr.To(sid.Int64()),
			Reason:      &ban.Reason,
			Duration:    durationpb.New(ban.ExpiresOn.Sub(ban.CreatedOn)),
			Permanent:   &ban.Permanent,
			CreatedOn:   timestamppb.New(ban.CreatedOn),
		}
	}

	return &resp, nil
}

func (s BanService) Update(ctx context.Context, req *v1.UpdateRequest) (*v1.UpdateResponse, error) {
	bannedPerson, banErr := s.bans.QueryOne(ctx, QueryOpts{BanID: req.GetBanId(), Deleted: true, EvadeOk: true})
	if banErr != nil {
		return nil, connect.NewError(connect.CodeNotFound, banErr)
	}

	if reason.Reason(ptr.From(req.Reason)) == reason.Custom {
		if req.GetReasonText() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
		}

		bannedPerson.ReasonText = req.GetReasonText()
	} else {
		bannedPerson.ReasonText = ""
	}

	if req.GetDuration() != nil {
		dur := req.GetDuration()
		bannedPerson.ValidUntil = time.Now().Add(dur.AsDuration())
	}

	bannedPerson.Note = ptr.From(req.Note)
	bannedPerson.BanType = bantype.Type(ptr.From(req.BanType))
	bannedPerson.Reason = reason.Reason(ptr.From(req.Reason))
	bannedPerson.EvadeOk = ptr.From(req.EvadeOk)

	if req.Cidr != nil {
		bannedPerson.CIDR = req.Cidr
	}

	if errSave := s.bans.Save(ctx, &bannedPerson); errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, ErrSaveBan)
	}

	return &v1.UpdateResponse{Ban: toBan(bannedPerson)}, nil
}

func toBan(ban Ban) *v1.Ban {
	return &v1.Ban{
		TargetId:          ptr.To(ban.TargetID.Int64()),
		SourceId:          ptr.To(ban.SourceID.Int64()),
		BanId:             &ban.BanID,
		ReportId:          &ban.ReportID,
		LastIp:            ban.LastIP,
		EvadeOk:           &ban.EvadeOk,
		BanType:           ptr.To(v1.BanType(ban.BanType)),
		Reason:            ptr.To(v1.BanReason(ban.Reason)),
		ReasonText:        &ban.ReasonText,
		UnbanReasonText:   &ban.UnbanReasonText,
		Note:              &ban.Note,
		Origin:            ptr.To(v1.Origin(ban.Origin)),
		Cidr:              ban.CIDR,
		AppealState:       ptr.To(v1.AppealState(ban.AppealState)),
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
