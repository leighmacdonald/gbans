// Package store provides functionality for communicating with the backend database. The database
// must implement the Store interface.
package store

import (
	"context"
	"github.com/jackc/pgx/v4"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"io"
	"net"
	"time"
)

type GenericStore interface {
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, query string, args ...any) error
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
}

type AuthStore interface {
	GetPersonAuthByRefreshToken(ctx context.Context, token string, auth *PersonAuth) error
	GetPersonAuth(ctx context.Context, sid64 steamid.SID64, ipAddr net.IP, auth *PersonAuth) error
	SavePersonAuth(ctx context.Context, auth *PersonAuth) error
	DeletePersonAuth(ctx context.Context, authId int64) error
	PrunePersonAuth(ctx context.Context) error
}

type ServerStore interface {
	GetServer(ctx context.Context, serverID int, server *Server) error
	GetServers(ctx context.Context, includeDisabled bool) ([]Server, error)
	GetServerByName(ctx context.Context, serverName string, server *Server) error
	SaveServer(ctx context.Context, server *Server) error
	DropServer(ctx context.Context, serverID int) error
}

type DemoStore interface {
	GetDemoById(ctx context.Context, demoId int64, demoFile *DemoFile) error
	GetDemoByName(ctx context.Context, demoName string, demoFile *DemoFile) error
	GetDemos(ctx context.Context, opts GetDemosOptions) ([]DemoFile, error)
	SaveDemo(ctx context.Context, d *DemoFile) error
	DropDemo(ctx context.Context, d *DemoFile) error
	FlushDemos(ctx context.Context) error
}

type NewsStore interface {
	GetNewsLatest(ctx context.Context, limit int, includeUnpublished bool) ([]NewsEntry, error)
	GetNewsLatestArticle(ctx context.Context, includeUnpublished bool, entry *NewsEntry) error
	GetNewsById(ctx context.Context, newsId int, entry *NewsEntry) error
	SaveNewsArticle(ctx context.Context, entry *NewsEntry) error
	DropNewsArticle(ctx context.Context, newsId int) error
}

type BanStore interface {
	GetBanBySteamID(ctx context.Context, steamID steamid.SID64, bannedPerson *BannedPerson, deletedOk bool) error
	GetBanByBanID(ctx context.Context, banID int64, bannedPerson *BannedPerson, deletedOk bool) error
	SaveBan(ctx context.Context, ban *BanSteam) error
	DropBan(ctx context.Context, ban *BanSteam, hardDelete bool) error
	GetBansSteam(ctx context.Context, queryFilter BansQueryFilter) ([]BannedPerson, error)
	GetBansOlderThan(ctx context.Context, queryFilter QueryFilter, time time.Time) ([]BanSteam, error)
	GetExpiredBans(ctx context.Context) ([]BanSteam, error)
	GetAppealsByActivity(ctx context.Context, queryFilter QueryFilter) ([]AppealOverview, error)

	GetBansNet(ctx context.Context) ([]BanCIDR, error)
	GetBanNetById(ctx context.Context, netId int64, banCidr *BanCIDR) error
	GetBanNetByAddress(ctx context.Context, ip net.IP) ([]BanCIDR, error)
	SaveBanNet(ctx context.Context, banNet *BanCIDR) error
	DropBanNet(ctx context.Context, ban *BanCIDR) error
	GetExpiredNetBans(ctx context.Context) ([]BanCIDR, error)

	GetBansASN(ctx context.Context) ([]BanASN, error)
	GetBanASN(ctx context.Context, asNum int64, banASN *BanASN) error
	SaveBanASN(ctx context.Context, banASN *BanASN) error
	DropBanASN(ctx context.Context, ban *BanASN) error
	GetExpiredASNBans(ctx context.Context) ([]BanASN, error)

	GetBanGroups(ctx context.Context) ([]BanGroup, error)
	GetBanGroup(ctx context.Context, groupId steamid.GID, banGroup *BanGroup) error
	GetBanGroupById(ctx context.Context, banGroupId int64, banGroup *BanGroup) error
	SaveBanGroup(ctx context.Context, banGroup *BanGroup) error
	DropBanGroup(ctx context.Context, banGroup *BanGroup) error

	SaveBanMessage(ctx context.Context, message *UserMessage) error
	DropBanMessage(ctx context.Context, message *UserMessage) error
	GetBanMessages(ctx context.Context, banId int64) ([]UserMessage, error)
	GetBanMessageById(ctx context.Context, banMessageId int, message *UserMessage) error
}

type ReportStore interface {
	SaveReport(ctx context.Context, report *Report) error
	SaveReportMessage(ctx context.Context, message *UserMessage) error
	DropReport(ctx context.Context, report *Report) error
	DropReportMessage(ctx context.Context, message *UserMessage) error
	GetReport(ctx context.Context, reportId int64, report *Report) error
	GetReportBySteamId(ctx context.Context, authorId steamid.SID64, steamId steamid.SID64, report *Report) error
	GetReports(ctx context.Context, opts AuthorQueryFilter) ([]Report, error)
	GetReportMessages(ctx context.Context, reportId int64) ([]UserMessage, error)
	GetReportMessageById(ctx context.Context, reportMessageId int64, message *UserMessage) error
}

type PersonStore interface {
	DropPerson(ctx context.Context, steamID steamid.SID64) error
	SavePerson(ctx context.Context, person *Person) error
	GetServerPermissions(ctx context.Context) ([]ServerPermission, error)
	GetPersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *Person) error
	GetPeople(ctx context.Context, qf QueryFilter) (People, error)
	GetPeopleBySteamID(ctx context.Context, steamIds steamid.Collection) (People, error)
	GetOrCreatePersonBySteamID(ctx context.Context, sid64 steamid.SID64, p *Person) error
	GetPersonByDiscordID(ctx context.Context, discordId string, person *Person) error
	GetExpiredProfiles(ctx context.Context, limit uint64) ([]Person, error)
	GetPersonIPHistory(ctx context.Context, sid steamid.SID64, limit uint64) (PersonConnections, error)
	QueryChatHistory(ctx context.Context, query ChatHistoryQueryFilter) (PersonMessages, error)
	GetPersonMessageById(ctx context.Context, query int64, msg *PersonMessage) error
	AddChatHistory(ctx context.Context, message *PersonMessage) error
	AddConnectionHistory(ctx context.Context, conn *PersonConnection) error
	SendNotification(ctx context.Context, targets steamid.SID64, severity NotificationSeverity, message string, link string) error
	GetPersonNotifications(ctx context.Context, steamId steamid.SID64) ([]UserNotification, error)
	SetNotificationsRead(ctx context.Context, notificationIds []int64) error
	GetSteamIdsAbove(ctx context.Context, privilege Privilege) (steamid.Collection, error)
}

type FilterStore interface {
	SaveFilter(ctx context.Context, filter *Filter) error
	DropFilter(ctx context.Context, filter *Filter) error
	GetFilterByID(ctx context.Context, wordId int64, filter *Filter) error
	GetFilters(ctx context.Context) ([]Filter, error)
}

type MigrationStore interface {
	Migrate(action MigrationAction) error
}

type MediaStore interface {
	SaveMedia(ctx context.Context, media *Media) error
	GetMediaByName(ctx context.Context, name string, media *Media) error
	GetMediaById(ctx context.Context, mediaId int, media *Media) error
}

type WikiStore interface {
	GetWikiPageBySlug(ctx context.Context, slug string, page *wiki.Page) error
	DeleteWikiPageBySlug(ctx context.Context, slug string) error
	SaveWikiPage(ctx context.Context, page *wiki.Page) error
}

type StatStore interface {
	GetStats(ctx context.Context, stats *Stats) error
	MatchSave(ctx context.Context, match *logparse.Match) error
	MatchGetById(ctx context.Context, matchId int) (*logparse.Match, error)
	Matches(ctx context.Context, opts MatchesQueryOpts) (logparse.MatchSummaryCollection, error)
	SaveLocalTF2Stats(ctx context.Context, duration StatDuration, stats LocalTF2StatsSnapshot) error
	GetLocalTF2Stats(ctx context.Context, duration StatDuration) ([]LocalTF2StatsSnapshot, error)
	SaveGlobalTF2Stats(ctx context.Context, duration StatDuration, stats GlobalTF2StatsSnapshot) error
	GetGlobalTF2Stats(ctx context.Context, duration StatDuration) ([]GlobalTF2StatsSnapshot, error)
	BuildGlobalTF2Stats(ctx context.Context) error
	BuildLocalTF2Stats(ctx context.Context) error
}

type NetworkStore interface {
	InsertBlockListData(ctx context.Context, blockListData *ip2location.BlockListData) error
	GetASNRecordByIP(ctx context.Context, ip net.IP, asnRecord *ip2location.ASNRecord) error
	GetASNRecordsByNum(ctx context.Context, asNum int64) (ip2location.ASNRecords, error)
	GetLocationRecord(ctx context.Context, ip net.IP, locationRecord *ip2location.LocationRecord) error
	GetProxyRecord(ctx context.Context, ip net.IP, proxyRecord *ip2location.ProxyRecord) error
}

// Store defines our composite store interface encapsulating all store interfaces
type Store interface {
	GenericStore
	BanStore
	DemoStore
	FilterStore
	MigrationStore
	NetworkStore
	PersonStore
	ServerStore
	StatStore
	ReportStore
	NewsStore
	WikiStore
	MediaStore
	AuthStore
	thirdparty.PatreonStore
	io.Closer
}
