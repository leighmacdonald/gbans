package store

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"io"
	"net"
	"time"
)

type ServerStore interface {
	GetServer(ctx context.Context, serverID int64, s *model.Server) error
	GetServers(ctx context.Context, includeDisabled bool) ([]model.Server, error)
	GetServerByName(ctx context.Context, serverName string, s *model.Server) error
	SaveServer(ctx context.Context, server *model.Server) error
	DropServer(ctx context.Context, serverID int64) error
}

type DemoStore interface {
	GetDemo(ctx context.Context, demoId int64, d *model.DemoFile) error
	GetDemos(ctx context.Context) ([]model.DemoFile, error)
	SaveDemo(ctx context.Context, d *model.DemoFile) error
	DropDemo(ctx context.Context, d *model.DemoFile) error
}

type BanStore interface {
	GetBanBySteamID(ctx context.Context, steamID steamid.SID64, full bool, b *model.BannedPerson) error
	GetBanByBanID(ctx context.Context, banID uint64, full bool, b *model.BannedPerson) error
	GetAppeal(ctx context.Context, banID uint64, a *model.Appeal) error
	SaveAppeal(ctx context.Context, appeal *model.Appeal) error
	SaveBan(ctx context.Context, ban *model.Ban) error
	GetBanNet(ctx context.Context, ip net.IP) ([]model.BanNet, error)
	SaveBanNet(ctx context.Context, banNet *model.BanNet) error
	DropNetBan(ctx context.Context, ban *model.BanNet) error
	DropBan(ctx context.Context, ban *model.Ban) error
	GetExpiredBans(ctx context.Context) ([]model.Ban, error)
	GetBans(ctx context.Context, o *QueryFilter) ([]model.BannedPerson, error)
	GetBansOlderThan(ctx context.Context, o *QueryFilter, t time.Time) ([]model.Ban, error)
	GetExpiredNetBans(ctx context.Context) ([]model.BanNet, error)
	Import(ctx context.Context, root string) error
}

type PersonStore interface {
	DropPerson(ctx context.Context, steamID steamid.SID64) error
	SavePerson(ctx context.Context, person *model.Person) error
	GetPersonBySteamID(ctx context.Context, sid steamid.SID64, p *model.Person) error
	GetPeople(ctx context.Context, qf *QueryFilter) ([]model.Person, error)
	GetOrCreatePersonBySteamID(ctx context.Context, sid steamid.SID64, p *model.Person) error
	GetPersonByDiscordID(ctx context.Context, did string, p *model.Person) error
	AddPersonIP(ctx context.Context, p *model.Person, ip string) error
	GetIPHistory(ctx context.Context, sid64 steamid.SID64) ([]model.PersonIPRecord, error)
	GetExpiredProfiles(ctx context.Context, limit int) ([]model.Person, error)
}

type FilterStore interface {
	InsertFilter(ctx context.Context, rx string) (*model.Filter, error)
	DropFilter(ctx context.Context, filter *model.Filter) error
	GetFilterByID(ctx context.Context, wordId int, f *model.Filter) error
	GetFilters(ctx context.Context) ([]*model.Filter, error)
}

type MigrationStore interface {
	Migrate(action MigrationAction) error
}

type StatStore interface {
	GetStats(ctx context.Context, s *model.Stats) error
	GetChatHistory(ctx context.Context, sid64 steamid.SID64, limit int) ([]logparse.SayEvt, error)
	FindLogEvents(ctx context.Context, opts model.LogQueryOpts) ([]model.ServerEvent, error)
	BatchInsertServerLogs(ctx context.Context, logs []model.ServerEvent) error
}

type NetworkStore interface {
	InsertBlockListData(ctx context.Context, d *ip2location.BlockListData) error
	GetASNRecord(ctx context.Context, ip net.IP, r *ip2location.ASNRecord) error
	GetLocationRecord(ctx context.Context, ip net.IP, l *ip2location.LocationRecord) error
	GetProxyRecord(ctx context.Context, ip net.IP, l *ip2location.ProxyRecord) error
	GetPersonIPHistory(ctx context.Context, sid steamid.SID64) ([]model.PersonIPRecord, error)
}

// Store defines our composite store interface encapsulating all store interfaces
type Store interface {
	BanStore
	DemoStore
	FilterStore
	MigrationStore
	NetworkStore
	PersonStore
	ServerStore
	StatStore
	io.Closer
}
