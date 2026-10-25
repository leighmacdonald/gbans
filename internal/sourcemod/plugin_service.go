package sourcemod

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	banv1 "github.com/leighmacdonald/gbans/internal/ban/v1"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/rpc"
	v1 "github.com/leighmacdonald/gbans/internal/sourcemod/v1"
	"github.com/leighmacdonald/gbans/internal/sourcemod/v1/sourcemodv1connect"
	"github.com/leighmacdonald/gbans/internal/stats"
	statsv1 "github.com/leighmacdonald/gbans/internal/stats/v1"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	errTooOften      = errors.New("you must wait before trying to mod ping again")
	errStatsTooOften = errors.New("you must wait before viewing your stats again")
)

// playerStatsTimeBuckets is the fixed set of time windows the in-game /stats
// command reports, from shortest to longest.
func playerStatsTimeBuckets() []stats.TimeBucket {
	return []stats.TimeBucket{
		stats.TimeBucketDaily,
		stats.TimeBucketWeekly,
		stats.TimeBucketMonthly,
		stats.TimeBucketYearly,
		stats.TimeBucketAlltime,
	}
}

type TokenGeneratorFn func(serverID int32, serverName string) (string, error)

// PluginService is responsible for all communication with the game servers using the sourcemod plugin.
type PluginService struct {
	sourcemod               Sourcemod
	notifier                notification.Notifier
	persons                 *person.Persons
	serverAuth              rpc.ServerAuthenticator
	tokenGenerator          TokenGeneratorFn
	evades                  EvadeChecker
	stats                   stats.Stats
	logChannelID            string
	pingHistory             map[steamid.SteamID]time.Time
	pingHistoryMu           *sync.Mutex
	minPingModRetryInterval time.Duration
	statsHistory            map[steamid.SteamID]time.Time
	statsHistoryMu          *sync.Mutex
	minStatsRetryInterval   time.Duration
}

func NewPluginService(sourcemod Sourcemod, persons *person.Persons, serverAuthenticator rpc.ServerAuthenticator, evades EvadeChecker, stats stats.Stats, tokenGenerator TokenGeneratorFn, notifier notification.Notifier, logChannelID string, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := sourcemodv1connect.NewPluginServiceHandler(PluginService{
		sourcemod:               sourcemod,
		persons:                 persons,
		tokenGenerator:          tokenGenerator,
		notifier:                notifier,
		evades:                  evades,
		stats:                   stats,
		logChannelID:            logChannelID,
		serverAuth:              serverAuthenticator,
		pingHistory:             map[steamid.SteamID]time.Time{},
		pingHistoryMu:           &sync.Mutex{},
		minPingModRetryInterval: time.Minute * 5,
		statsHistory:            map[steamid.SteamID]time.Time{},
		statsHistoryMu:          &sync.Mutex{},
		minStatsRetryInterval:   time.Minute * 5,
	}, option...)

	serverAuth := rpc.NewServerAuthenticator()

	authMiddleware.ServerRoute(sourcemodv1connect.PluginServiceSMCheckProcedure, serverAuth)
	authMiddleware.ServerRoute(sourcemodv1connect.PluginServiceSMOverridesProcedure, serverAuth)
	authMiddleware.ServerRoute(sourcemodv1connect.PluginServiceSMUsersProcedure, serverAuth)
	authMiddleware.ServerRoute(sourcemodv1connect.PluginServiceSMGroupsProcedure, serverAuth)
	authMiddleware.ServerRoute(sourcemodv1connect.PluginServiceSMSeedProcedure, serverAuth)
	authMiddleware.ServerRoute(sourcemodv1connect.PluginServiceSMPingModProcedure, serverAuth)
	authMiddleware.ServerRoute(sourcemodv1connect.PluginServiceSMPlayerStatsProcedure, serverAuth)

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s PluginService) SMPingMod(ctx context.Context, req *v1.SMPingModRequest) (*emptypb.Empty, error) {
	serverInfo := rpc.ServerInfoFromCtx(ctx)
	steamID := steamid.New(req.GetSteamId())

	if !steamID.Valid() {
		return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
	}

	s.pingHistoryMu.Lock()
	defer s.pingHistoryMu.Unlock()
	lastTry, ok := s.pingHistory[steamID]
	if ok && time.Since(lastTry) < s.minPingModRetryInterval {
		return nil, connect.NewError(connect.CodeResourceExhausted, errTooOften)
	}

	if err := s.sourcemod.PingMod(ctx, steamID, req.GetName(), req.GetReason(), req.GetClientId(), serverInfo.ServerName); err != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}
	s.pingHistory[steamID] = time.Now()

	return &emptypb.Empty{}, nil
}

func (s PluginService) SMAuthenticate(ctx context.Context, req *v1.SMAuthenticateRequest) (*v1.SMAuthenticateResponse, error) {
	password := req.GetPassword()
	if password == "" {
		return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
	}

	serverID, name, err := s.serverAuth.GetByPassword(ctx, password)
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
	}

	token, errToken := s.tokenGenerator(serverID, name)
	if errToken != nil || token == "" {
		return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
	}

	slog.Debug("Server authentication", slog.Int("serverId", int(serverID)), slog.String("name", name))

	return &v1.SMAuthenticateResponse{Token: &token}, nil
}

// SMCheck verifies whether a connecting player is banned by their SteamID or IP address.
// It fails open on errors, allowing the player to connect rather than causing a denial-of-service.
// If a ban is found that doesn't match the provided SteamID directly, it checks for evasion
// bans before returning the result. Discord notifications are sent asynchronously on denials.
func (s PluginService) SMCheck(ctx context.Context, req *v1.SMCheckRequest) (*v1.SMCheckResponse, error) {
	defaultResponse := &v1.SMCheckResponse{
		ClientId: new(req.GetClientId()),
		BanType:  new(banv1.BanType_BAN_TYPE_OK_UNSPECIFIED),
		Msg:      new(""),
	}
	steamID := steamid.New(req.GetSteamId())
	ipAddr, errIP := netip.ParseAddr(req.GetIp())
	if errIP != nil {
		slog.Error("Failed to parse IP", slog.String("error", errIP.Error()))

		return defaultResponse, nil //nolint:nilerr
	}

	banState, msg, errBS := s.sourcemod.GetBanState(ctx, steamID, ipAddr)
	if errBS != nil {
		slog.Error("failed to get ban state", slog.String("error", errBS.Error()))

		// Fail Open
		return defaultResponse, nil //nolint:nilerr
	}

	if banState.BanID == 0 {
		slog.Debug("Player connect check", slog.String("steam_id", steamID.String()), slog.Bool("success", true))

		return defaultResponse, nil
	}

	if errPlayer := s.persons.EnsurePerson(ctx, steamID); errPlayer != nil {
		slog.Error("Failed to load or create player on connect")

		return defaultResponse, nil //nolint:nilerr
	}

	if banState.SteamID != steamID && !banState.EvadeOK {
		evadeBanned, err := s.evades.CheckEvadeStatus(ctx, steamID, ipAddr)
		if err != nil {
			slog.Error("Failed to check evade status", slog.String("error", err.Error()))

			return defaultResponse, nil //nolint:nilerr
		}

		if evadeBanned {
			go s.notifier.Send(notification.NewDiscord(s.logChannelID, newCheckDenyMessage(banState)))

			return &v1.SMCheckResponse{ClientId: defaultResponse.ClientId, BanType: toBanType(bantype.Banned), Msg: new("Evasion ban")}, nil
		}
	}

	if banState.SteamID != steamID && banState.EvadeOK {
		return defaultResponse, nil
	}

	go s.notifier.Send(notification.NewDiscord(s.logChannelID, newCheckDenyMessage(banState)))

	return &v1.SMCheckResponse{
		ClientId: defaultResponse.ClientId,
		BanType:  toBanType(banState.BanType),
		Msg:      &msg,
	}, nil
}

func (s PluginService) SMOverrides(ctx context.Context, _ *emptypb.Empty) (*v1.SMOverridesResponse, error) {
	overrides, errOverrides := s.sourcemod.Overrides(ctx)
	if errOverrides != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.SMOverridesResponse{Overrides: make([]*v1.SMOverride, len(overrides))}
	for idx, override := range overrides {
		resp.Overrides[idx] = &v1.SMOverride{
			OverrideType: toOverrideType(override.Type),
			Name:         &override.Name,
			Permissions:  override.Permissions,
		}
	}

	return &resp, nil
}

func (s PluginService) SMUsers(ctx context.Context, _ *emptypb.Empty) (*v1.SMUsersResponse, error) {
	users, errUsers := s.sourcemod.Admins(ctx)
	if errUsers != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	// UserGroups is always a non-nil slice so the JSON response contains the
	// key even when empty; the sourcemod plugin dereferences it unconditionally.
	resp := v1.SMUsersResponse{
		Users:      make([]*v1.SMUser, len(users)),
		UserGroups: make([]*v1.SMUserGroup, 0),
	}

	for idx, user := range users {
		resp.Users[idx] = &v1.SMUser{
			Id:          &user.AdminID,
			AuthType:    toAuthType(user.AuthType),
			Identity:    &user.Identity,
			Password:    &user.Password,
			Name:        &user.Name,
			Immunity:    &user.Immunity,
			Permissions: user.Permissions,
		}

		for _, ug := range user.Groups {
			resp.UserGroups = append(resp.UserGroups, &v1.SMUserGroup{
				AdminId:   &user.AdminID,
				GroupName: &ug.Name,
			})
		}
	}

	return &resp, nil
}

func (s PluginService) SMGroups(ctx context.Context, _ *emptypb.Empty) (*v1.SMGroupsResponse, error) {
	groups, errGroups := s.sourcemod.Groups(ctx)
	if errGroups != nil && !errors.Is(errGroups, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
	}

	immunities, errImmunities := s.sourcemod.GroupImmunities(ctx)
	if errImmunities != nil && !errors.Is(errImmunities, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.SMGroupsResponse{
		Groups:     make([]*v1.Group, len(groups)),
		Immunities: make([]*v1.SMGroupImmunity, len(immunities)),
	}

	//goland:noinspection ALL
	for idx, group := range groups {
		resp.Groups[idx] = &v1.Group{
			Name:          &group.Name,
			ImmunityLevel: &group.ImmunityLevel,
			Permissions:   group.Permissions,
		}
	}

	for idx, immunity := range immunities {
		resp.Immunities[idx] = &v1.SMGroupImmunity{
			GroupName: &immunity.Group.Name,
			OtherName: &immunity.Other.Name,
		}
	}

	if resp.Groups == nil {
		resp.Groups = []*v1.Group{}
	}

	if resp.Immunities == nil {
		resp.Immunities = []*v1.SMGroupImmunity{}
	}

	return &resp, nil
}

func (s PluginService) SMSeed(ctx context.Context, req *v1.SMSeedRequest) (*v1.SMSeedResponse, error) {
	serverInfo := rpc.ServerInfoFromCtx(ctx)
	steamID := steamid.New(req.GetSteamId())
	if !steamID.Valid() {
		return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
	}

	server, errServer := s.sourcemod.servers.Server(ctx, serverInfo.ServerID)
	if errServer != nil {
		return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
	}

	serverState, errState := s.sourcemod.servers.SafeState(ctx, server.ServerID)
	if errState != nil {
		return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
	}

	if ok, errSeed := s.sourcemod.seedRequest(
		server.DiscordSeedRoleIDs, serverState, steamID.String(), "", server.DiscordSeedChannelID,
	); !ok {
		if errors.Is(errSeed, ErrReqTooSoon) {
			// The plugin maps this error to HTTP 429 and tells the requesting player to wait.
			return nil, connect.NewError(connect.CodeResourceExhausted, errSeed)
		}

		if errors.Is(errSeed, ErrNoSeedRoles) || errors.Is(errSeed, ErrSeedDisabled) {
			// These are configuration errors the server owner can fix, so surface
			// the reason instead of collapsing it into a generic internal error.
			return nil, connect.NewError(connect.CodeFailedPrecondition, errSeed)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.SMSeedResponse{Message: new("Successfully sent request")}, nil
}

// SMPlayerStats returns a player's overall (points) rank and kills for every
// time window in the default stats bucket, for the in-game /stats menu. The
// rank matches the website's default overall leaderboard so the two agree.
// Each player may call it at most once every five minutes.
func (s PluginService) SMPlayerStats(ctx context.Context, req *v1.SMPlayerStatsRequest) (*v1.SMPlayerStatsResponse, error) {
	steamID := steamid.New(req.GetSteamId())
	if !steamID.Valid() {
		return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
	}

	s.statsHistoryMu.Lock()
	lastTry, ok := s.statsHistory[steamID]
	if ok && time.Since(lastTry) < s.minStatsRetryInterval {
		s.statsHistoryMu.Unlock()

		return nil, connect.NewError(connect.CodeResourceExhausted, errStatsTooOften)
	}

	buckets, errBuckets := s.stats.Buckets(ctx)
	if errBuckets != nil {
		s.statsHistoryMu.Unlock()
		slog.Error("Failed to load stats buckets", slog.String("error", errBuckets.Error()))

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	defaultBucket := defaultStatsBucket(buckets)
	if defaultBucket == nil {
		s.statsHistoryMu.Unlock()
		slog.Error("No default or enabled stats bucket configured")

		return nil, connect.NewError(connect.CodeFailedPrecondition, rpc.ErrInternal)
	}

	timeBuckets := playerStatsTimeBuckets()
	resp := &v1.SMPlayerStatsResponse{
		StatsBucketId:   new(uint32(defaultBucket.BucketID)), //nolint:gosec // bucket ids are positive.
		StatsBucketName: &defaultBucket.BucketName,
		Buckets:         make([]*v1.SMPlayerStatsBucket, 0, len(timeBuckets)),
	}

	now := time.Now()
	for _, timeBucket := range timeBuckets {
		// VariantOverallUnspecified ranks by points, the same column the
		// website's default overall leaderboard shows, so the in-game rank
		// matches the frontend for the same player and window.
		rank, kills, errRank := s.stats.PlayerKillsAndRank(ctx, stats.Opts{
			StatsBucketID: uint32(defaultBucket.BucketID), //nolint:gosec // bucket ids are positive.
			Variant:       stats.VariantOverallUnspecified,
			TimeBucket:    timeBucket,
			TimeStamp:     now,
		}, steamID)
		if errRank != nil {
			s.statsHistoryMu.Unlock()
			slog.Error("Failed to load player stats",
				slog.String("steamID", steamID.String()),
				slog.String("timeBucket", timeBucket.String()),
				slog.String("error", errRank.Error()))

			return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
		}

		resp.Buckets = append(resp.Buckets, &v1.SMPlayerStatsBucket{
			TimeBucket: new(toTimeBucket(timeBucket)),
			Rank:       new(rank),
			Kills:      new(kills),
		})
	}

	// Only record a successful lookup so a transient backend failure does not
	// lock the player out for the whole cooldown window.
	s.statsHistory[steamID] = time.Now()
	s.statsHistoryMu.Unlock()

	return resp, nil
}

// defaultStatsBucket returns the default stats bucket, falling back to the
// first enabled bucket, or nil when neither exists.
func defaultStatsBucket(buckets []stats.Bucket) *stats.Bucket {
	for idx := range buckets {
		if buckets[idx].IsDefault {
			return &buckets[idx]
		}
	}

	for idx := range buckets {
		if buckets[idx].IsEnabled {
			return &buckets[idx]
		}
	}

	return nil
}

// toTimeBucket maps the internal stats time bucket to its wire enum value.
func toTimeBucket(tb stats.TimeBucket) statsv1.TimeBucket {
	switch tb {
	case stats.TimeBucketDaily:
		return statsv1.TimeBucket_TIME_BUCKET_DAILY
	case stats.TimeBucketWeekly:
		return statsv1.TimeBucket_TIME_BUCKET_WEEKLY
	case stats.TimeBucketMonthly:
		return statsv1.TimeBucket_TIME_BUCKET_MONTHLY
	case stats.TimeBucketYearly:
		return statsv1.TimeBucket_TIME_BUCKET_YEARLY
	default:
		return statsv1.TimeBucket_TIME_BUCKET_ALLTIME //nolint:misspell // must match the generated enum name.
	}
}
