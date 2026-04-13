package anticheat

import (
	"context"
	"fmt"

	v1 "github.com/leighmacdonald/gbans/internal/anticheat/v1"
	"github.com/leighmacdonald/gbans/internal/anticheat/v1/anticheatv1connect"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewService(anticheat AntiCheat) Service {
	return Service{anticheat: anticheat}
}

type Service struct {
	anticheatv1connect.UnimplementedAnticheatServiceHandler

	anticheat AntiCheat
}

func (s Service) Query(ctx context.Context, request *v1.QueryRequest) (*v1.QueryResponse, error) {
	opts := Query{
		Filter:    query.FromRPC(request.Filter),
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
		return ptr.To(v1.Detection_SilentAim)
	case logparse.AimSnap:
		return ptr.To(v1.Detection_AimSnap)
	case logparse.TooManyConnectiona:
		return ptr.To(v1.Detection_TooManyConnections)
	case logparse.Interp:
		return ptr.To(v1.Detection_Interp)
	case logparse.BHop:
		return ptr.To(v1.Detection_BHop)
	case logparse.CmdNumSpike:
		return ptr.To(v1.Detection_CmdNumSpike)
	case logparse.EyeAngles:
		return ptr.To(v1.Detection_EyeAngles)
	case logparse.InvalidUserCmd:
		return ptr.To(v1.Detection_InvalidUserCmd)
	case logparse.OOBCVar:
		return ptr.To(v1.Detection_OOBCVar)
	case logparse.CheatCVar:
		return ptr.To(v1.Detection_CheatCVar)
	case logparse.Unknown:
		fallthrough
	default:
		return ptr.To(v1.Detection_Unknown)
	}
}
