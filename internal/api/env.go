package api

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/activity"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/s3"
	"github.com/leighmacdonald/gbans/internal/state"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/ip2location"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/wiki"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"go.uber.org/zap"
)

type Env interface {
	Log() *zap.Logger
	Config() config.Config
	Store() store.Stores
	SendPayload(channelID string, message *discordgo.MessageEmbed)
	Version() model.BuildInfo
	Assets() s3.AssetStore
	Activity() *activity.Tracker
	State() *state.Collector
	NetBlocks() model.NetBLocker
	Patreon() model.Patreon
	Groups() model.Groups
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

type Store interface {
	BanStore
	ContestStore
	DemoStore
	FilterStore
	ForumStore
	MatchStore
	NetStore
	NewsStore
	PatreonStore
	PeopleStore
	ReportStore
	ServerStore
	StatsStore
	WikiStore
}

type BanStore interface {
	DropBan(ctx context.Context, ban *model.BanSteam, hardDelete bool) error
	GetBanBySteamID(ctx context.Context, sid64 steamid.SID64, bannedPerson *model.BannedSteamPerson, deletedOk bool) error
	GetBanByBanID(ctx context.Context, banID int64, bannedPerson *model.BannedSteamPerson, deletedOk bool) error
	GetBanByLastIP(ctx context.Context, lastIP net.IP, bannedPerson *model.BannedSteamPerson, deletedOk bool) error
	SaveBan(ctx context.Context, ban *model.BanSteam) error
	GetExpiredBans(ctx context.Context) ([]model.BanSteam, error)
	GetAppealsByActivity(ctx context.Context, opts model.AppealQueryFilter) ([]model.AppealOverview, int64, error)
	GetBansSteam(ctx context.Context, filter model.SteamBansQueryFilter) ([]model.BannedSteamPerson, int64, error)
	GetBansOlderThan(ctx context.Context, filter model.QueryFilter, since time.Time) ([]model.BanSteam, error)
	SaveBanMessage(ctx context.Context, message *model.BanAppealMessage) error
	GetBanMessages(ctx context.Context, banID int64) ([]model.BanAppealMessage, error)
	GetBanMessageByID(ctx context.Context, banMessageID int, message *model.BanAppealMessage) error
	DropBanMessage(ctx context.Context, message *model.BanAppealMessage) error
	GetBanGroup(ctx context.Context, groupID steamid.GID, banGroup *model.BanGroup) error
	GetBanGroupByID(ctx context.Context, banGroupID int64, banGroup *model.BanGroup) error
	GetBanGroups(ctx context.Context, filter model.GroupBansQueryFilter) ([]model.BannedGroupPerson, int64, error)
	GetMembersList(ctx context.Context, parentID int64, list *model.MembersList) error
	SaveMembersList(ctx context.Context, list *model.MembersList) error
	SaveBanGroup(ctx context.Context, banGroup *model.BanGroup) error
	DropBanGroup(ctx context.Context, banGroup *model.BanGroup) error
}

type ServerStore interface {
	SaveServer(ctx context.Context, server *model.Server) error
	GetServer(ctx context.Context, serverID int, server *model.Server) error
	GetServerPermissions(ctx context.Context) ([]model.ServerPermission, error)
	GetServers(ctx context.Context, filter model.ServerQueryFilter) ([]model.Server, int64, error)
	GetServerByName(ctx context.Context, serverName string, server *model.Server, disabledOk bool, deletedOk bool) error
	GetServerByPassword(ctx context.Context, serverPassword string, server *model.Server, disabledOk bool, deletedOk bool) error
	DropServer(ctx context.Context, serverID int) error
}

type PeopleStore interface {
	DropPerson(ctx context.Context, steamID steamid.SID64) error
	SavePerson(ctx context.Context, person *model.Person) error
	GetPersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *model.Person) error
	GetPeopleBySteamID(ctx context.Context, steamIds steamid.Collection) (model.People, error)
	GetSteamsAtAddress(ctx context.Context, addr net.IP) (steamid.Collection, error)
	GetPeople(ctx context.Context, filter model.PlayerQuery) (model.People, int64, error)
	GetOrCreatePersonBySteamID(ctx context.Context, sid64 steamid.SID64, person *model.Person) error
	GetPersonByDiscordID(ctx context.Context, discordID string, person *model.Person) error
	GetExpiredProfiles(ctx context.Context, limit uint64) ([]model.Person, error)
	AddChatHistory(ctx context.Context, message *model.PersonMessage) error
	GetPersonMessageByID(ctx context.Context, personMessageID int64, msg *model.PersonMessage) error
	QueryConnectionHistory(ctx context.Context, opts model.ConnectionHistoryQueryFilter) ([]model.PersonConnection, int64, error)
	QueryChatHistory(ctx context.Context, filters model.ChatHistoryQueryFilter) ([]model.QueryChatHistoryResult, int64, error)
	GetPersonMessage(ctx context.Context, messageID int64, msg *model.QueryChatHistoryResult) error
	GetPersonMessageContext(ctx context.Context, serverID int, messageID int64, paddedMessageCount int) ([]model.QueryChatHistoryResult, error)
	GetPersonIPHistory(ctx context.Context, sid64 steamid.SID64, limit uint64) (model.PersonConnections, error)
	AddConnectionHistory(ctx context.Context, conn *model.PersonConnection) error
	GetPersonAuthByRefreshToken(ctx context.Context, token string, auth *model.PersonAuth) error
	SavePersonAuth(ctx context.Context, auth *model.PersonAuth) error
	DeletePersonAuth(ctx context.Context, authID int64) error
	PrunePersonAuth(ctx context.Context, database Store) error
	SendNotification(ctx context.Context, targetID steamid.SID64, severity model.NotificationSeverity, message string, link string) error
	GetPersonNotifications(ctx context.Context, filters model.NotificationQuery) ([]model.UserNotification, int64, error)
	GetSteamIdsAbove(ctx context.Context, privilege model.Privilege) (steamid.Collection, error)
	GetPersonSettings(ctx context.Context, steamID steamid.SID64, settings *model.PersonSettings) error
	SavePersonSettings(ctx context.Context, settings *model.PersonSettings) error
}

type ContestStore interface {
	ContestByID(ctx context.Context, contestID uuid.UUID, contest *model.Contest) error
	ContestDelete(ctx context.Context, contestID uuid.UUID) error
	ContestEntryDelete(ctx context.Context, contestEntryID uuid.UUID) error
	Contests(ctx context.Context, publicOnly bool) ([]model.Contest, error)
	ContestEntrySave(ctx context.Context, entry model.ContestEntry) error
	ContestSave(ctx context.Context, contest *model.Contest) error
	ContestEntry(ctx context.Context, contestID uuid.UUID, entry *model.ContestEntry) error
	ContestEntries(ctx context.Context, contestID uuid.UUID) ([]*model.ContestEntry, error)
	ContestEntryVoteGet(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SID64, record *model.ContentVoteRecord) error
	ContestEntryVote(ctx context.Context, contestEntryID uuid.UUID, steamID steamid.SID64, vote bool) error
	ContestEntryVoteDelete(ctx context.Context, contestEntryVoteID int64) error
	ContestEntryVoteUpdate(ctx context.Context, contestEntryVoteID int64, newVote bool) error
}

type DemoStore interface {
	ExpiredDemos(ctx context.Context, limit uint64) ([]model.DemoInfo, error)
	GetDemoByID(ctx context.Context, demoID int64, demoFile *model.DemoFile) error
	GetDemoByName(ctx context.Context, demoName string, demoFile *model.DemoFile) error
	GetDemos(ctx context.Context, opts model.DemoFilter) ([]model.DemoFile, int64, error)
	SaveDemo(ctx context.Context, demoFile *model.DemoFile) error
	DropDemo(ctx context.Context, demoFile *model.DemoFile) error
	SaveAsset(ctx context.Context, asset *model.Asset) error
}

type FilterStore interface {
	SaveFilter(ctx context.Context, filter *model.Filter) error
	DropFilter(ctx context.Context, filter *model.Filter) error
	GetFilterByID(ctx context.Context, filterID int64, filter *model.Filter) error
	GetFilters(ctx context.Context, opts model.FiltersQueryFilter) ([]model.Filter, int64, error)
	AddMessageFilterMatch(ctx context.Context, messageID int64, filterID int64) error
}

type ForumStore interface {
	ForumCategories(ctx context.Context) ([]model.ForumCategory, error)
	ForumCategorySave(ctx context.Context, category *model.ForumCategory) error
	ForumCategory(ctx context.Context, categoryID int, category *model.ForumCategory) error
	ForumCategoryDelete(ctx context.Context, categoryID int) error
	Forums(ctx context.Context) ([]model.Forum, error)
	ForumSave(ctx context.Context, forum *model.Forum) error
	Forum(ctx context.Context, forumID int, forum *model.Forum) error
	ForumDelete(ctx context.Context, forumID int) error
	ForumThreadSave(ctx context.Context, thread *model.ForumThread) error
	ForumThread(ctx context.Context, forumThreadID int64, thread *model.ForumThread) error
	ForumThreadIncrView(ctx context.Context, forumThreadID int64) error
	ForumThreadDelete(ctx context.Context, forumThreadID int64) error
	ForumThreads(ctx context.Context, filter model.ThreadQueryFilter) ([]model.ThreadWithSource, int64, error)
	ForumIncrMessageCount(ctx context.Context, forumID int, incr bool) error
	ForumMessageSave(ctx context.Context, message *model.ForumMessage) error
	ForumRecentActivity(ctx context.Context, limit uint64, permissionLevel model.Privilege) ([]model.ForumMessage, error)
	ForumMessage(ctx context.Context, messageID int64, forumMessage *model.ForumMessage) error
	ForumMessages(ctx context.Context, filters model.ThreadMessagesQueryFilter) ([]model.ForumMessage, int64, error)
	ForumMessageDelete(ctx context.Context, messageID int64) error
	ForumMessageVoteApply(ctx context.Context, messageVote *model.ForumMessageVote) error
	ForumMessageVoteByID(ctx context.Context, messageVoteID int64, messageVote *model.ForumMessageVote) error
}

type MatchStore interface {
	MatchGetByID(ctx context.Context, matchID uuid.UUID, match *model.MatchResult) error
	MatchSave(ctx context.Context, match *logparse.Match, weaponMap fp.MutexMap[logparse.Weapon, int]) error
	StatsPlayerClass(ctx context.Context, sid64 steamid.SID64) (model.PlayerClassStatsCollection, error)
	StatsPlayerWeapons(ctx context.Context, sid64 steamid.SID64) ([]model.PlayerWeaponStats, error)
	StatsPlayerKillstreaks(ctx context.Context, sid64 steamid.SID64) ([]model.PlayerKillstreakStats, error)
	StatsPlayerMedic(ctx context.Context, sid64 steamid.SID64) ([]model.PlayerMedicStats, error)
	PlayerStats(ctx context.Context, steamID steamid.SID64, stats *model.PlayerStats) error
	Matches(ctx context.Context, opts model.MatchesQueryOpts) ([]model.MatchSummary, int64, error)
}

type NetStore interface {
	GetBanNetByAddress(ctx context.Context, ipAddr net.IP) ([]model.BanCIDR, error)
	GetBanNetByID(ctx context.Context, netID int64, banNet *model.BanCIDR) error
	GetPlayerMostRecentIP(ctx context.Context, steamID steamid.SID64) net.IP
	GetBansNet(ctx context.Context, filter model.CIDRBansQueryFilter) ([]model.BannedCIDRPerson, int64, error)
	SaveBanNet(ctx context.Context, banNet *model.BanCIDR) error
	DropBanNet(ctx context.Context, banNet *model.BanCIDR) error
	GetExpiredNetBans(ctx context.Context) ([]model.BanCIDR, error)
	GetExpiredASNBans(ctx context.Context) ([]model.BanASN, error)
	GetASNRecordsByNum(ctx context.Context, asNum int64) (ip2location.ASNRecords, error)
	GetASNRecordByIP(ctx context.Context, ipAddr net.IP, asnRecord *ip2location.ASNRecord) error
	GetLocationRecord(ctx context.Context, ipAddr net.IP, record *ip2location.LocationRecord) error
	GetProxyRecord(ctx context.Context, ipAddr net.IP, proxyRecord *ip2location.ProxyRecord) error
	InsertBlockListData(ctx context.Context, log *zap.Logger, blockListData *ip2location.BlockListData) error
	GetBanASN(ctx context.Context, asNum int64, banASN *model.BanASN) error
	GetBansASN(ctx context.Context, filter model.ASNBansQueryFilter) ([]model.BannedASNPerson, int64, error)
	SaveBanASN(ctx context.Context, banASN *model.BanASN) error
	DropBanASN(ctx context.Context, banASN *model.BanASN) error
	GetSteamIDsAtIP(ctx context.Context, ipNet *net.IPNet) (steamid.Collection, error)
	GetCIDRBlockSources(ctx context.Context) ([]model.CIDRBlockSource, error)
	GetCIDRBlockSource(ctx context.Context, sourceID int, block *model.CIDRBlockSource) error
	SaveCIDRBlockSources(ctx context.Context, block *model.CIDRBlockSource) error
	DeleteCIDRBlockSources(ctx context.Context, blockSourceID int) error
	GetCIDRBlockWhitelists(ctx context.Context) ([]model.CIDRBlockWhitelist, error)
	GetCIDRBlockWhitelist(ctx context.Context, whitelistID int, whitelist *model.CIDRBlockWhitelist) error
	SaveCIDRBlockWhitelist(ctx context.Context, whitelist *model.CIDRBlockWhitelist) error
	DeleteCIDRBlockWhitelist(ctx context.Context, whitelistID int) error
}

type NewsStore interface {
	GetNewsLatest(ctx context.Context, limit int, includeUnpublished bool) ([]model.NewsEntry, error)
	GetNewsLatestArticle(ctx context.Context, includeUnpublished bool, entry *model.NewsEntry) error
	GetNewsByID(ctx context.Context, newsID int, entry *model.NewsEntry) error
	SaveNewsArticle(ctx context.Context, entry *model.NewsEntry) error
	DropNewsArticle(ctx context.Context, newsID int) error
}

type PatreonStore interface {
	SetPatreonAuth(ctx context.Context, accessToken string, refreshToken string) error
	GetPatreonAuth(ctx context.Context) (string, string, error)
}

type ReportStore interface {
	SaveReport(ctx context.Context, report *model.Report) error
	SaveReportMessage(ctx context.Context, message *model.ReportMessage) error
	DropReport(ctx context.Context, report *model.Report) error
	DropReportMessage(ctx context.Context, message *model.ReportMessage) error
	GetReports(ctx context.Context, opts model.ReportQueryFilter) ([]model.Report, int64, error)
	GetReportBySteamID(ctx context.Context, authorID steamid.SID64, steamID steamid.SID64, report *model.Report) error
	GetReport(ctx context.Context, reportID int64, report *model.Report) error
	GetReportMessages(ctx context.Context, reportID int64) ([]model.ReportMessage, error)
	GetReportMessageByID(ctx context.Context, reportMessageID int64, message *model.ReportMessage) error
}

type StatsStore interface {
	LoadWeapons(ctx context.Context, weaponMap fp.MutexMap[logparse.Weapon, int]) error
	GetWeaponByKey(ctx context.Context, key logparse.Weapon, weapon *model.Weapon) error
	GetWeaponByID(ctx context.Context, weaponID int, weapon *model.Weapon) error
	SaveWeapon(ctx context.Context, weapon *model.Weapon) error
	Weapons(ctx context.Context) ([]model.Weapon, error)
	GetStats(ctx context.Context, stats *model.Stats) error
	GetMapUsageStats(ctx context.Context) ([]model.MapUseDetail, error)
	TopChatters(ctx context.Context, count uint64) ([]model.TopChatterResult, error)
	WeaponsOverall(ctx context.Context) ([]model.WeaponsOverallResult, error)
	WeaponsOverallTopPlayers(ctx context.Context, weaponID int) ([]model.PlayerWeaponResult, error)
	WeaponsOverallByPlayer(ctx context.Context, steamID steamid.SID64) ([]model.WeaponsOverallResult, error)
	PlayersOverallByKills(ctx context.Context, count int) ([]model.PlayerWeaponResult, error)
	HealersOverallByHealing(ctx context.Context, count int) ([]model.HealingOverallResult, error)
	PlayerOverallClassStats(ctx context.Context, steamID steamid.SID64) ([]model.PlayerClassOverallResult, error)
	PlayerOverallStats(ctx context.Context, steamID steamid.SID64, por *model.PlayerOverallResult) error
}

type WikiStore interface {
	GetWikiPageBySlug(ctx context.Context, slug string, page *wiki.Page) error
	DeleteWikiPageBySlug(ctx context.Context, slug string) error
	SaveWikiPage(ctx context.Context, page *wiki.Page) error
	SaveMedia(ctx context.Context, media *model.Media) error
	GetMediaByAssetID(ctx context.Context, uuid uuid.UUID, media *model.Media) error
	GetMediaByName(ctx context.Context, name string, media *model.Media) error
	GetMediaByID(ctx context.Context, mediaID int, media *model.Media) error
}
