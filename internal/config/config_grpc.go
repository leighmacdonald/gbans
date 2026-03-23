package config

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	configv1 "github.com/leighmacdonald/gbans/internal/rpc/config/v1"
	"github.com/leighmacdonald/gbans/internal/rpc/config/v1/configv1connect"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RPC struct {
	*Configuration
	configv1connect.UnimplementedConfigServiceHandler
	version string
}

func (r RPC) Info(context.Context, *emptypb.Empty) (*configv1.InfoResponse, error) {
	conf := r.Config()

	resp := configv1.InfoResponse{
		SiteName:           conf.General.SiteName,
		AssetUrl:           conf.General.AssetURL,
		Favicon:            conf.General.FaviconURL(),
		LinkId:             conf.Discord.LinkID,
		AppVersion:         r.version,
		DocumentPolicy:     "",
		SentryDsnWeb:       conf.General.SentryDSNWeb,
		SiteDescription:    conf.General.SiteDescription,
		PatreonClientId:    conf.Patreon.ClientID,
		DiscordClientId:    conf.Discord.AppID,
		DiscordEnabled:     conf.Discord.IntegrationsEnabled && conf.Discord.Enabled,
		PatreonEnabled:     conf.Patreon.IntegrationsEnabled && conf.Patreon.Enabled,
		DefaultRoute:       conf.General.DefaultRoute,
		NewsEnabled:        conf.General.NewsEnabled,
		ForumsEnabled:      conf.General.ForumsEnabled,
		ContestsEnabled:    conf.General.ContestsEnabled,
		WikiEnabled:        conf.General.WikiEnabled,
		StatsEnabled:       conf.General.StatsEnabled,
		ServersEnabled:     conf.General.ServersEnabled,
		ReportsEnabled:     conf.General.ReportsEnabled,
		ChatlogsEnabled:    conf.General.ChatlogsEnabled,
		DemosEnabled:       conf.General.DemosEnabled,
		SpeedrunsEnabled:   conf.General.SpeedrunsEnabled,
		PlayerqueueEnabled: conf.General.PlayerqueueEnabled,
	}
	return &resp, nil
}
func (r RPC) Get(context.Context, *emptypb.Empty) (*configv1.Config, error) {
	c := r.Config()
	return &configv1.Config{
		General:     &configv1.General{
			SiteName: c.General.SiteName,
			SiteDescription: c.General.SiteDescription,
			Mode: c.General.Mode,
			
		},
		Discord:     &configv1.Discord{},
		Patreon:     &configv1.Patreon{},
		Debug:       &configv1.Debug{},
		Demo:        &configv1.Demo{},
		Filters:     &configv1.Filters{},
		Sourcemd:    &configv1.Sourcemd{},
		Log:         &configv1.Log{},
		GeoLocation: &configv1.GeoLocation{},
		Ssh:         &configv1.SSH{},
		Network:     &configv1.Network{},
		LocalStore:  &configv1.LocalStore{},
		Exports:     &configv1.Exports{},
		Anticheat:   &configv1.Anticheat{},
	}, nil
}

func (r RPC) Update(context.Context, *configv1.Config) (*configv1.Config, error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("config.v1.ConfigService.Update is not implemented"))
}
