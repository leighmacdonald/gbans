package anticheat

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	v1 "github.com/leighmacdonald/gbans/internal/anticheat/v1"
	"github.com/leighmacdonald/gbans/internal/anticheat/v1/anticheatv1connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
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
	// anticheatv1connect.UnimplementedAnticheatServiceHandler

	anticheat AntiCheat
}

func (s Service) Query(ctx context.Context, request *v1.QueryRequest) (*v1.QueryResponse, error) {
	opts := Query{
		Filter:  rpc.FromRPC(request.Filter),
		Name:    request.GetName(),
		Summary: request.GetSummary(),
	}

	if request.Detection != nil {
		opts.Detection = toDetection(request.GetDetection())
	}

	if request.SteamId != nil {
		opts.SteamID = fmt.Sprintf("%d", request.GetSteamId())
	}

	entries, errResults := s.anticheat.Query(ctx, opts)
	if errResults != nil {
		return nil, errResults
	}

	results := v1.QueryResponse{Entries: make([]*v1.Entry, len(entries))}
	for idx, entry := range entries {
		results.Entries[idx] = &v1.Entry{
			AnticheatId: &entry.AnticheatID,
			SteamId:     new(entry.SteamID.Int64()),
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
		return new(v1.Detection_DETECTION_SILENT_AIM)
	case logparse.AimSnap:
		return new(v1.Detection_DETECTION_AIM_SNAP)
	case logparse.TooManyConnectiona:
		return new(v1.Detection_DETECTION_TOO_MANY_CONNECTIONS)
	case logparse.Interp:
		return new(v1.Detection_DETECTION_INTERP)
	case logparse.BHop:
		return new(v1.Detection_DETECTION_BHOP)
	case logparse.CmdNumSpike:
		return new(v1.Detection_DETECTION_CMD_NUM_SPIKE)
	case logparse.EyeAngles:
		return new(v1.Detection_DETECTION_EYE_ANGLES)
	case logparse.InvalidUserCmd:
		return new(v1.Detection_DETECTION_INVALID_USER_CMD)
	case logparse.OOBCVar:
		return new(v1.Detection_DETECTION_OOB_CVAR)
	case logparse.CheatCVar:
		return new(v1.Detection_DETECTION_CHEAT_CVAR)
	case logparse.Unknown:
		fallthrough
	default:
		return new(v1.Detection_DETECTION_UNSPECIFIED)
	}
}

func toDetection(detection v1.Detection) logparse.Detection {
	switch detection {
	case v1.Detection_DETECTION_AIM_SNAP:
		return logparse.AimSnap
	case v1.Detection_DETECTION_BHOP:
		return logparse.BHop
	case v1.Detection_DETECTION_CHEAT_CVAR:
		return logparse.CheatCVar
	case v1.Detection_DETECTION_CMD_NUM_SPIKE:
		return logparse.CmdNumSpike
	case v1.Detection_DETECTION_EYE_ANGLES:
		return logparse.EyeAngles
	case v1.Detection_DETECTION_INTERP:
		return logparse.Interp
	case v1.Detection_DETECTION_INVALID_USER_CMD:
		return logparse.InvalidUserCmd
	case v1.Detection_DETECTION_OOB_CVAR:
		return logparse.OOBCVar
	case v1.Detection_DETECTION_SILENT_AIM:
		return logparse.SilentAim
	case v1.Detection_DETECTION_TOO_MANY_CONNECTIONS:
		return logparse.TooManyConnectiona
	case v1.Detection_DETECTION_UNSPECIFIED:
		fallthrough
	default:
		return logparse.Any
	}
}
