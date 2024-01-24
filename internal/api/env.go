package api

import (
	"context"
	"io"
	"net"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/activity"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/domain"
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
	Version() domain.BuildInfo
	Assets() *s3.Client
	Activity() *activity.Tracker
	State() *state.Collector
	NetBlocks() domain.NetBLocker
	Patreon() domain.Patreon
	Groups() *thirdparty.SteamGroupMemberships
	Friends() *thirdparty.SteamFriends
	Warnings() domain.Warnings

	BanASN(ctx context.Context, banASN *domain.BanASN) error
	BanCIDR(ctx context.Context, banNet *domain.BanCIDR) error
	BanSteam(ctx context.Context, banSteam *domain.BanSteam) error
	BanSteamGroup(ctx context.Context, banGroup *domain.BanGroup) error

	FilterAdd(ctx context.Context, filter *domain.Filter) error
	Unban(ctx context.Context, targetSID steamid.SID64, reason string) (bool, error)
}

type ServerState interface {
	Current() []state.ServerState
	Find(name string, steamID steamid.SID64, ip net.IP, cidr *net.IPNet) []domain.PlayerServerInfo
	Update(serverID int, update domain.PartialStateUpdate) error
}

type ActivityTracker interface {
	Touch(person domain.UserProfile)
	Current() []domain.ForumActivity
}

type AssetStore interface {
	Put(ctx context.Context, bucket string, name string, body io.Reader, size int64, contentType string) error
	Remove(ctx context.Context, bucket string, name string) error
}
