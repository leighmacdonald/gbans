package api

import (
	"context"
	"io"
	"net"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/activity"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/s3"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type Env interface {
	Log() *zap.Logger
	Config() config.Config
	Store() store.Stores
	SendPayload(channelID string, message *discordgo.MessageEmbed)
	Version() model.BuildInfo
	Assets() *s3.Client
	Activity() *activity.Tracker
	State() *state.Collector
	NetBlocks() model.NetBLocker
	Patreon() model.Patreon
	Groups() *thirdparty.SteamGroupMemberships
	Friends() *thirdparty.SteamFriends
	Warnings() model.Warnings

	BanASN(ctx context.Context, banASN *model.BanASN) error
	BanCIDR(ctx context.Context, banNet *model.BanCIDR) error
	BanSteam(ctx context.Context, banSteam *model.BanSteam) error
	BanSteamGroup(ctx context.Context, banGroup *model.BanGroup) error

	FilterAdd(ctx context.Context, filter *model.Filter) error
	Unban(ctx context.Context, targetSID steamid.SID64, reason string) (bool, error)
}

type ServerState interface {
	Current() []state.ServerState
	Find(name string, steamID steamid.SID64, ip net.IP, cidr *net.IPNet) []model.PlayerServerInfo
	Update(serverID int, update model.PartialStateUpdate) error
}

type ActivityTracker interface {
	Touch(person model.UserProfile)
	Current() []model.ForumActivity
}

type AssetStore interface {
	Put(ctx context.Context, bucket string, name string, body io.Reader, size int64, contentType string) error
	Remove(ctx context.Context, bucket string, name string) error
}
