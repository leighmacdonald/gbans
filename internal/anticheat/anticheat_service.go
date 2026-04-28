package anticheat

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	v1 "github.com/leighmacdonald/gbans/internal/anticheat/v1"
	"github.com/leighmacdonald/gbans/internal/anticheat/v1/anticheatv1connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewService(anticheat AntiCheat, authMiddleware *rpc.Middleware, interceptor ...connect.HandlerOption) rpc.Service {
	pattern, handler := anticheatv1connect.NewAnticheatServiceHandler(Service{anticheat: anticheat}, interceptor...)

	authMiddleware.UserRoute(anticheatv1connect.AnticheatServiceQueryProcedure, rpc.WithMinPermissions(permission.Moderator))

	return rpc.Service{
		Pattern: pattern,
		Handler: handler,
	}
}

type Service struct {
	anticheatv1connect.UnimplementedAnticheatServiceHandler

	anticheat AntiCheat
}

func (s Service) Query(ctx context.Context, request *v1.QueryRequest) (*v1.QueryResponse, error) {
	opts := Query{
		Filter:    rpc.FromRPC(request.Filter),
		Name:      request.GetName(),
		SteamID:   fmt.Sprintf("%d", request.GetSteamId()),
		ServerID:  0,
		Summary:   request.GetSummary(),
		Detection: "",
	}

	entries, errResults := s.anticheat.Query(ctx, opts)
	if errResults != nil {
		return nil, errResults
	}

	results := v1.QueryResponse{Entries: make([]*v1.Entry, len(entries))}
	for idx, entry := range entries {
		results.Entries[idx] = &v1.Entry{
			AnticheatId: &entry.AnticheatID,
			SteamId:     ptr.To(entry.SteamID.Int64()),
			ServerId:    &entry.ServerID,
			ServerName:  &entry.ServerName,
			DemoId:      entry.DemoID,
			DemoName:    &entry.DemoName,
			DemoTick:    &entry.DemoTick,
			Name:        &entry.Name,
			Detection:   detectionToRPC(entry.Detection),
			Summary:     &entry.Summary,
			RawLog:      &entry.RawLog,
			CreatedOn:   timestamppb.New(entry.CreatedOn),
			PersonaName: &entry.Personaname,
			AvatarHash:  &entry.AvatarHash,
			Triggered:   &entry.Triggered,
		}
	}

	return &results, nil
}

func detectionToRPC(detection logparse.Detection) *v1.Detection {
	switch detection {
	case logparse.SilentAim:
		return ptr.To(v1.Detection_DETECTION_SILENT_AIM)
	case logparse.AimSnap:
		return ptr.To(v1.Detection_DETECTION_AIM_SNAP)
	case logparse.TooManyConnectiona:
		return ptr.To(v1.Detection_DETECTION_TOO_MANY_CONNECTIONS)
	case logparse.Interp:
		return ptr.To(v1.Detection_DETECTION_INTERP)
	case logparse.BHop:
		return ptr.To(v1.Detection_DETECTION_BHOP)
	case logparse.CmdNumSpike:
		return ptr.To(v1.Detection_DETECTION_CMD_NUM_SPIKE)
	case logparse.EyeAngles:
		return ptr.To(v1.Detection_DETECTION_EYE_ANGLES)
	case logparse.InvalidUserCmd:
		return ptr.To(v1.Detection_DETECTION_INVALID_USER_CMD)
	case logparse.OOBCVar:
		return ptr.To(v1.Detection_DETECTION_OOB_CVAR)
	case logparse.CheatCVar:
		return ptr.To(v1.Detection_DETECTION_CHEAT_CVAR)
	case logparse.Unknown:
		fallthrough
	default:
		return ptr.To(v1.Detection_DETECTION_UNSPECIFIED)
	}
}
