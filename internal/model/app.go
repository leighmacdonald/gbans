package model

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/discordutil"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"go.uber.org/zap"
	"gopkg.in/mxpv/patreon-go.v1"
	"net"
)

type Application interface {
	Store() store.Store
	Start() error
	Logger() *zap.Logger
	Ctx() context.Context // TODO remove
	Find(ctx context.Context, playerStr store.StringSID, ip string, playerInfo *state.PlayerInfo) error
	FindPlayerByCIDR(ctx context.Context, ipNet *net.IPNet, playerInfo *state.PlayerInfo) error
	ServerState() state.ServerStateCollection
	PersonBySID(ctx context.Context, sid steamid.SID64, person *store.Person) error
	Kick(ctx context.Context, origin store.Origin, target store.StringSID, author store.StringSID,
		reason store.Reason, playerInfo *state.PlayerInfo) error
	Silence(ctx context.Context, origin store.Origin, target store.StringSID, author store.StringSID,
		reason store.Reason, playerInfo *state.PlayerInfo) error
	SetSteam(ctx context.Context, sid64 steamid.SID64, discordId string) error
	Say(ctx context.Context, author steamid.SID64, serverName string, message string) error
	CSay(ctx context.Context, author steamid.SID64, serverName string, message string) error
	PSay(ctx context.Context, author steamid.SID64, target store.StringSID, message string, server *store.Server) error
	FilterAdd(ctx context.Context, filter *store.Filter) error
	FilterDel(ctx context.Context, database store.Store, filterId int64) (bool, error)
	FilterCheck(message string) []store.Filter
	BanSteam(ctx context.Context, banSteam *store.BanSteam) error
	BanASN(ctx context.Context, banASN *store.BanASN) error
	BanCIDR(ctx context.Context, banNet *store.BanCIDR) error
	BanSteamGroup(ctx context.Context, banGroup *store.BanGroup) error
	Unban(ctx context.Context, target steamid.SID64, reason string) (bool, error)
	UnbanASN(ctx context.Context, asnNum string) (bool, error)
	PatreonPledges() []patreon.Pledge
	PatreonCampaigns() []patreon.Campaign
	SendDiscordPayload(payload discordutil.Payload)
	IsSteamGroupBanned(steamId steamid.SID64) bool
	MasterServerList() []ServerLocation
	LogFileChan() chan *LogFilePayload
	SendUserNotification(pl NotificationPayload)
}
