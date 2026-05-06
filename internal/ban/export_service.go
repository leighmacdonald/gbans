package ban

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	v1 "github.com/leighmacdonald/gbans/internal/ban/v1"
	"github.com/leighmacdonald/gbans/internal/ban/v1/banv1connect"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
)

type ExportService struct {
	banv1connect.UnimplementedExportServiceHandler

	bans           Bans
	authorizedKeys []string
	siteName       string
}

func NewExportService(bans Bans, authorizedKeys []string, siteName string, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := banv1connect.NewExportServiceHandler(ExportService{
		bans:           bans,
		authorizedKeys: authorizedKeys,
		siteName:       siteName,
	}, option...)

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s ExportService) GetTF2BD(ctx context.Context, req *v1.GetTF2BDRequest) (*v1.TF2BDSchema, error) {
	if len(s.authorizedKeys) > 0 {
		key := req.GetKey()
		if key == "" || !slices.Contains(s.authorizedKeys, key) {
			return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
		}
	}

	bans, errBans := s.bans.Query(ctx, QueryOpts{})
	if errBans != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
	}

	var filtered []Ban

	for _, curBan := range bans {
		if curBan.Reason != reason.Cheating || curBan.Deleted || !curBan.IsEnabled {
			continue
		}

		filtered = append(filtered, curBan)
	}

	resp := v1.TF2BDSchema{
		Schema: ptr.To("https://raw.githubusercontent.com/PazerOP/tf2_bot_detector/master/schemas/v3/playerlist.schema.json"),
		FileInfo: &v1.FileInfo{
			Authors:     []string{s.siteName},
			Description: ptr.To("Players permanently banned for cheating"),
			Title:       ptr.To(s.siteName + " Cheater List"),
			UpdateUrl:   ptr.To("/export/bans/tf2bd"),
		},
		Players: []*v1.Player{},
	}

	for _, ban := range filtered {
		resp.Players = append(resp.Players, &v1.Player{
			Attributes: []string{"cheater"},
			SteamId:    ptr.To(string(ban.TargetID.Steam3())),
			LastSeen: &v1.LastSeen{
				PlayerName: ptr.To(ban.TargetID.String()),
				Time:       ptr.To(int32(ban.UpdatedOn.Unix())),
			},
		})
	}

	return &resp, nil
}

func (s ExportService) GetValveSteamID(ctx context.Context, req *v1.GetValveSteamIDRequest) (*v1.GetValveSteamIDResponse, error) {
	if len(s.authorizedKeys) > 0 {
		key := req.GetKey()
		if key == "key" || !slices.Contains(s.authorizedKeys, key) {
			return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
		}
	}

	// TODO limit to perm?
	bans, errBans := s.bans.Query(ctx, QueryOpts{})
	if errBans != nil && !errors.Is(errBans, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.GetValveSteamIDResponse{}

	for _, ban := range bans {
		if ban.Deleted || !ban.IsEnabled {
			continue
		}

		resp.BanLines = append(resp.BanLines, fmt.Sprintf("banid 0 %s", ban.TargetID.Steam(false)))
	}

	return &resp, nil
}
