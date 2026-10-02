package ban_test

import (
	"context"
	"net"
	"net/netip"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/demo"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/maps"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/roles"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/stats"
	"github.com/leighmacdonald/gbans/internal/tests"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

// fixture provides a shared set of common dependencies that can be used for integration testing.
var fixture *tests.Fixture //nolint:gochecknoglobals

func TestMain(m *testing.M) {
	fixture = tests.NewFixture()
	defer fixture.Close()

	m.Run()
}

// env bundles the dependencies needed to exercise the ban package.
type env struct {
	bans       ban.Bans
	repo       ban.Repository
	appeals    ban.Appeals
	appealRepo ban.AppealRepository
	reports    ban.Reports
	reportRepo ban.ReportRepository
	roleAuth   *rpc.RoleAuth
	roleSvc    *roles.Roles
	owner      steamid.SteamID
}

type envConfig struct {
	notif      notification.Notifier
	ipProvider ban.LastIPProvider
}

type envOpt func(*envConfig)

func withNotifier(notif notification.Notifier) envOpt {
	return func(c *envConfig) { c.notif = notif }
}

func withIPProvider(provider ban.LastIPProvider) envOpt {
	return func(c *envConfig) { c.ipProvider = provider }
}

func newEnv(t *testing.T, opts ...envOpt) *env {
	t.Helper()

	cfg := envConfig{
		notif:      notification.NewDiscard(),
		ipProvider: tests.EmptyIPProvider{},
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	roleSvc := roles.NewRoles(roles.NewRepository(fixture.Database), tests.OwnerSID)
	roleAuth := rpc.NewRoleAuth(roleSvc)

	assets := asset.NewAssets(asset.NewLocalRepository(fixture.Database, t.TempDir()))
	filters := chat.NewWordFilters(chat.NewWordFilterRepository(fixture.Database), notification.NewDiscard(), fixture.Config.Config().Filters)
	chatCase := chat.New(chat.NewRepository(fixture.Database), fixture.Config.Config().Filters, filters, fixture.Persons, notification.NewDiscard(), nil, "")
	statsCase := stats.New(stats.NewRepository(fixture.Database), maps.New(maps.NewRepository(fixture.Database)))
	demos := demo.NewDemos(asset.BucketDemo, demo.NewRepository(fixture.Database),
		assets, statsCase, chatCase, fixture.Persons, fixture.Config.Config().Demo, tests.OwnerSID)

	reportRepo := ban.NewReportRepository(fixture.Database)
	reports := ban.NewReports(reportRepo,
		person.NewPersons(person.NewRepository(fixture.Database, true), fixture.TFApi),
		demos, fixture.TFApi, cfg.notif, "", roleAuth)

	serversCase, errServers := servers.New(servers.NewRepository(fixture.Database), nil, "")
	require.NoError(t, errServers)

	// The fixture config loses its Owner (Configuration.reload overwrites the
	// embedded Static with an empty one), so use the known test owner instead.
	owner := tests.OwnerSID

	repo := ban.NewRepository(fixture.Database)
	bans := ban.New(repo, fixture.Persons,
		fixture.Config.Config().Discord.BanLogChannelID, fixture.Config.Config().Discord.KickLogChannelID,
		owner, reports, cfg.notif, serversCase, cfg.ipProvider)

	appealRepo := ban.NewAppealRepository(fixture.Database)
	appeals := ban.NewAppeals(appealRepo, bans, fixture.Persons, cfg.notif, fixture.Config.Config().Discord.BanLogChannelID, roleAuth)

	return &env{
		bans:       bans,
		repo:       repo,
		appeals:    appeals,
		appealRepo: appealRepo,
		reports:    reports,
		reportRepo: reportRepo,
		roleAuth:   roleAuth,
		roleSvc:    roleSvc,
		owner:      owner,
	}
}

// grantRole assigns an existing role (e.g. "admin", "moderator") to the steam id.
func (e *env) grantRole(t *testing.T, sid steamid.SteamID, roleName string) {
	t.Helper()

	all, err := e.roleSvc.GetAll(t.Context())
	require.NoError(t, err)

	var roleID int32
	for _, role := range all {
		if role.RoleName == roleName {
			roleID = role.RoleID

			break
		}
	}

	require.NotZero(t, roleID, "role %q not found", roleName)
	require.NoError(t, e.roleSvc.Assign(t.Context(), sid, roleID))
}

// createAndGrantRole creates a new role with the given permission names and assigns it to the steam id.
func (e *env) createAndGrantRole(t *testing.T, sid steamid.SteamID, perms ...string) {
	t.Helper()

	roleName := stringutil.SecureRandomString(8)
	role, err := e.roleSvc.Create(t.Context(), roleName, perms)
	require.NoError(t, err)
	require.NoError(t, e.roleSvc.Assign(t.Context(), sid, role.RoleID))
}

// createPerson creates a person row for the steam id and returns the person.
func createPerson(t *testing.T, sid steamid.SteamID) personDomain.Core {
	t.Helper()

	return fixture.CreateTestPerson(t.Context(), sid)
}

// setPersonName sets the person row name and avatar. Person profile data is not
// persisted by GetOrCreatePersonBySteamID (person.updatePerson discards the
// ApplySteamInfo result), so tests that assert on joined person names set them
// explicitly.
func setPersonName(t *testing.T, sid steamid.SteamID, name, avatar string) {
	t.Helper()

	require.NoError(t, fixture.Database.Exec(t.Context(),
		"UPDATE person SET personaname = $1, avatarhash = $2 WHERE steam_id = $3",
		name, avatar, sid.Int64()))
}

// saveRawBan saves a ban directly through the repository, bypassing service level validation.
func (e *env) saveRawBan(t *testing.T, subject *ban.Ban) ban.Ban {
	t.Helper()

	createPerson(t, subject.TargetID)
	createPerson(t, subject.SourceID)

	require.NoError(t, e.repo.Save(t.Context(), subject))

	return *subject
}

// banIDs returns the sorted ban ids of the given bans.
func banIDs(bans []ban.Ban) []int32 {
	ids := make([]int32, 0, len(bans))
	for _, b := range bans {
		ids = append(ids, b.BanID)
	}

	slices.Sort(ids)

	return ids
}

type capturingNotifier struct {
	sync.Mutex //nolint:bytestring

	payloads []notification.Payload
}

func (c *capturingNotifier) Send(payload notification.Payload) {
	c.Lock()
	defer c.Unlock()

	c.payloads = append(c.payloads, payload)
}

func (c *capturingNotifier) List() []notification.Payload {
	c.Lock()
	defer c.Unlock()

	return slices.Clone(c.payloads)
}

type staticIPProvider struct {
	ip net.IP
}

func (s staticIPProvider) GetPlayerMostRecentIP(_ context.Context, _ steamid.SteamID) net.IP {
	return s.ip
}

func TestOrigin_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		origin ban.Origin
		want   string
	}{
		{name: "System", origin: ban.System, want: "System"},
		{name: "Bot", origin: ban.Bot, want: "Bot"},
		{name: "Web", origin: ban.Web, want: "Web"},
		{name: "InGame", origin: ban.InGame, want: "In-Game"},
		{name: "Reported", origin: ban.Reported, want: "Reported"},
		{name: "unknown", origin: ban.Origin(99), want: "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.origin.String())
		})
	}
}

func TestOpts_Validate(t *testing.T) {
	t.Parallel()

	future := time.Now().Add(time.Hour)

	tests := []struct {
		name        string
		opts        ban.Opts
		wantErr     error
		wantErrAlso error
	}{
		{
			name: "valid",
			opts: ban.Opts{ValidUntil: future, Reason: reason.Cheating},
		},
		{
			name: "valid with cidr",
			opts: ban.Opts{ValidUntil: future, Reason: reason.Cheating, CIDR: new("198.51.100.0/24")},
		},
		{
			name: "valid custom reason",
			opts: ban.Opts{ValidUntil: future, Reason: reason.Custom, ReasonText: "abc"},
		},
		{
			name:        "expired valid until",
			opts:        ban.Opts{ValidUntil: time.Now().Add(-time.Hour), Reason: reason.Cheating},
			wantErr:     ban.ErrInvalidBanOpts,
			wantErrAlso: ban.ErrInvalidBanDuration,
		},
		{
			name:    "custom reason too short",
			opts:    ban.Opts{ValidUntil: future, Reason: reason.Custom, ReasonText: "ab"},
			wantErr: ban.ErrInvalidBanOpts,
		},
		{
			name:    "invalid cidr",
			opts:    ban.Opts{ValidUntil: future, Reason: reason.Cheating, CIDR: new("not-a-cidr")},
			wantErr: ban.ErrInvalidBanOpts,
		},
		{
			name:    "cidr too many hosts",
			opts:    ban.Opts{ValidUntil: future, Reason: reason.Cheating, CIDR: new("10.0.0.0/15")},
			wantErr: ban.ErrInvalidBanOpts,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := testCase.opts.Validate()
			if testCase.wantErr == nil {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			require.ErrorIs(t, err, testCase.wantErr)

			if testCase.wantErrAlso != nil {
				require.ErrorIs(t, err, testCase.wantErrAlso)
			}
		})
	}
}

func TestAddressCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cidr string
		want uint64
	}{
		{name: "single host", cidr: "203.0.113.5/32", want: 1},
		{name: "class c", cidr: "198.51.100.0/24", want: 256},
		{name: "class b", cidr: "10.0.0.0/16", want: 65536},
		{name: "larger than class b", cidr: "10.0.0.0/15", want: 131072},
		{name: "ipv6 single host", cidr: "2001:db8::/128", want: 1},
		{name: "ipv6 /64 shift overflow", cidr: "2001:db8::/64", want: 0},
		{name: "ipv6 /48 shift overflow", cidr: "2001:db8:abcd::/48", want: 0},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, ipnet, err := net.ParseCIDR(testCase.cidr)
			require.NoError(t, err)
			require.Equal(t, testCase.want, ban.AddressCount(ipnet))
		})
	}
}

func TestBan_IsGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sid  steamid.SteamID
		want bool
	}{
		{name: "regular user sid", sid: steamid.New(76561198000000001), want: false},
		{name: "base gid", sid: steamid.New(steamid.BaseGID), want: true},
		{name: "group sid above base", sid: steamid.New(steamid.BaseGID + 42), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, ban.Ban{TargetID: tt.sid}.IsGroup())
		})
	}
}

func TestBan_Path(t *testing.T) {
	require.Equal(t, "/ban/42", ban.Ban{BanID: 42}.Path())
}

func TestBan_String(t *testing.T) {
	subject := ban.Ban{
		TargetID:   steamid.New(76561198000000001),
		Origin:     ban.System,
		ReasonText: "cheat engine",
		BanType:    bantype.Banned,
	}

	require.Equal(t, "SID: 76561198000000001 Origin: System Reason: cheat engine Type: banned", subject.String())
}

func TestBan_Expired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		validUntil time.Time
		want       bool
	}{
		{name: "expired", validUntil: time.Now().Add(-time.Minute), want: true},
		{name: "not expired", validUntil: time.Now().Add(time.Minute), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, ban.Ban{ValidUntil: tt.validUntil}.Expired())
		})
	}
}

func TestBans_Query(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	e := newEnv(t)

	var (
		srcA, srcB = steamid.RandSID64(), steamid.RandSID64()
		tgtA, tgtB = steamid.RandSID64(), steamid.RandSID64()
		tgtC       = steamid.RandSID64()
		tgtD, tgtE = steamid.RandSID64(), steamid.RandSID64()
		groupSID   = steamid.New(steamid.BaseGID + 1337)
		cidr       = "192.0.2.0/28"
	)

	for _, sid := range []steamid.SteamID{srcA, srcB, tgtA, tgtB, tgtC, tgtD, tgtE, groupSID} {
		createPerson(t, sid)
	}

	banA, err := e.bans.Create(ctx, ban.Opts{
		SourceID: srcA, TargetID: tgtA, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating,
	})
	require.NoError(t, err)

	banB, err := e.bans.Create(ctx, ban.Opts{
		SourceID: srcA, TargetID: tgtB, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.NoComm, Reason: reason.Spam,
	})
	require.NoError(t, err)

	// tgtC is soft deleted below
	banC, err := e.bans.Create(ctx, ban.Opts{
		SourceID: srcB, TargetID: tgtC, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating,
	})
	require.NoError(t, err)
	require.NoError(t, e.bans.Delete(ctx, &banC, false))

	// Bans.Create cannot be used for group targets: its final re-fetch uses the
	// default group exclusion (target_id < BaseGID), so creation fails with
	// "ban does not exist". Save at repository level instead.
	banD := e.saveRawBan(t, &ban.Ban{
		SourceID: srcB, TargetID: groupSID, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating,
	})

	banE, err := e.bans.Create(ctx, ban.Opts{
		SourceID: srcB, TargetID: tgtD, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Network, Reason: reason.External, CIDR: &cidr,
	})
	require.NoError(t, err)

	// banF is attached to a report
	report := ban.Report{
		SourceID: srcB, TargetID: tgtE, Description: "cheating evidence",
		ReportStatus: ban.Opened, CreatedOn: time.Now(), UpdatedOn: time.Now(),
	}
	require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

	banF, err := e.bans.Create(ctx, ban.Opts{
		SourceID: srcB, TargetID: tgtE, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating, ReportID: &report.ReportID,
	})
	require.NoError(t, err)

	t.Run("by target id", func(t *testing.T) {
		t.Parallel()

		got, err := e.bans.Query(ctx, ban.QueryOpts{TargetID: tgtA})
		require.NoError(t, err)
		require.Equal(t, []int32{banA.BanID}, banIDs(got))
	})

	t.Run("by source id excludes deleted", func(t *testing.T) {
		t.Parallel()

		got, err := e.bans.Query(ctx, ban.QueryOpts{SourceID: srcA})
		require.NoError(t, err)
		require.Equal(t, banIDs([]ban.Ban{banA, banB}), banIDs(got))
	})

	t.Run("by source id including deleted", func(t *testing.T) {
		t.Parallel()

		got, err := e.bans.Query(ctx, ban.QueryOpts{SourceID: srcA, Deleted: true})
		require.NoError(t, err)
		require.Len(t, got, 2)
	})

	t.Run("by source and reason", func(t *testing.T) {
		t.Parallel()

		got, err := e.bans.Query(ctx, ban.QueryOpts{SourceID: srcB, Reasons: []reason.Reason{reason.Cheating}})
		require.NoError(t, err)
		require.Equal(t, []int32{banF.BanID}, banIDs(got))
	})

	t.Run("by source, reason, including deleted and groups", func(t *testing.T) {
		t.Parallel()

		got, err := e.bans.Query(ctx, ban.QueryOpts{
			SourceID: srcB, Reasons: []reason.Reason{reason.Cheating}, Deleted: true, IncludeGroups: true,
		})
		require.NoError(t, err)
		require.Equal(t, banIDs([]ban.Ban{banC, banD, banF}), banIDs(got))
	})

	t.Run("by ban id", func(t *testing.T) {
		t.Parallel()

		got, err := e.bans.Query(ctx, ban.QueryOpts{BanID: banB.BanID})
		require.NoError(t, err)
		require.Equal(t, []int32{banB.BanID}, banIDs(got))
	})

	t.Run("by report id", func(t *testing.T) {
		t.Parallel()

		got, err := e.bans.Query(ctx, ban.QueryOpts{ReportID: report.ReportID})
		require.NoError(t, err)
		require.Equal(t, []int32{banF.BanID}, banIDs(got))
	})

	t.Run("excludes groups by default", func(t *testing.T) {
		t.Parallel()

		got, err := e.bans.Query(ctx, ban.QueryOpts{SourceID: srcB})
		require.NoError(t, err)
		require.NotContains(t, banIDs(got), banD.BanID)
	})

	t.Run("groups only", func(t *testing.T) {
		t.Parallel()

		got, err := e.bans.Query(ctx, ban.QueryOpts{SourceID: srcB, GroupsOnly: true, IncludeGroups: true})
		require.NoError(t, err)
		require.Equal(t, []int32{banD.BanID}, banIDs(got))
	})

	t.Run("include groups", func(t *testing.T) {
		t.Parallel()

		got, err := e.bans.Query(ctx, ban.QueryOpts{SourceID: srcB, IncludeGroups: true})
		require.NoError(t, err)
		require.Contains(t, banIDs(got), banD.BanID)
	})

	t.Run("by cidr range", func(t *testing.T) {
		t.Parallel()

		// 192.0.2.5 is inside banE's 192.0.2.0/28 range.
		got, err := e.bans.Query(ctx, ban.QueryOpts{CIDR: "192.0.2.5"})
		require.NoError(t, err)
		require.Contains(t, banIDs(got), banE.BanID)
	})

	t.Run("cidr only", func(t *testing.T) {
		t.Parallel()

		got, err := e.bans.Query(ctx, ban.QueryOpts{SourceID: srcB, CIDROnly: true})
		require.NoError(t, err)
		require.Equal(t, []int32{banE.BanID}, banIDs(got))
	})

	t.Run("query one", func(t *testing.T) {
		t.Parallel()

		got, err := e.bans.QueryOne(ctx, ban.QueryOpts{BanID: banA.BanID})
		require.NoError(t, err)
		require.Equal(t, banA, got)
	})

	t.Run("query one not found", func(t *testing.T) {
		t.Parallel()

		_, err := e.bans.QueryOne(ctx, ban.QueryOpts{BanID: 99999999})
		require.ErrorIs(t, err, ban.ErrBanDoesNotExist)
	})
}

func TestBans_Save(t *testing.T) {
	t.Parallel()

	t.Run("appeal state change sends notifications", func(t *testing.T) {
		t.Parallel()

		capt := &capturingNotifier{}
		e := newEnv(t, withNotifier(capt))

		source, target := steamid.RandSID64(), steamid.RandSID64()

		created := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})
		require.Equal(t, ban.Open, created.AppealState)

		created.AppealState = ban.Denied
		require.NoError(t, e.bans.Save(t.Context(), &created))

		payloads := capt.List()
		require.Len(t, payloads, 2)

		var stateChanged, userNotified bool
		for _, payload := range payloads {
			switch {
			case slices.Contains(payload.Groups, rolesv1.Permission_PERMISSION_BAN_READ) && payload.Message == "Ban appeal state changed: Open -> Denied":
				stateChanged = true
			case slices.Contains(payload.Sids, target) && payload.Message == "Your mute/ban appeal status has changed: Open -> Denied":
				userNotified = true
			}
		}

		require.True(t, stateChanged, "expected appeal state group notification, got %+v", payloads)
		require.True(t, userNotified, "expected appeal state user notification, got %+v", payloads)
	})

	t.Run("no notification when appeal state unchanged", func(t *testing.T) {
		t.Parallel()

		capt := &capturingNotifier{}
		e := newEnv(t, withNotifier(capt))

		source, target := steamid.RandSID64(), steamid.RandSID64()

		created := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})

		require.NoError(t, e.bans.Save(t.Context(), &created))
		require.Empty(t, capt.List())
	})
}

func TestBans_Create(t *testing.T) {
	t.Parallel()

	t.Run("creates a ban", func(t *testing.T) {
		t.Parallel()

		harness := newEnv(t)

		source, target := steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		createPerson(t, target)

		cidr := "198.51.100.0/30"
		created, err := harness.bans.Create(t.Context(), ban.Opts{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(10 * time.Hour),
			BanType: bantype.Banned, Reason: reason.Custom, ReasonText: "aimbot",
			Origin: ban.Web, Note: "admin note", Name: "cheater", CIDR: &cidr,
		})
		require.NoError(t, err)
		require.Positive(t, created.BanID)
		require.Equal(t, source, created.SourceID)
		require.Equal(t, target, created.TargetID)
		require.Equal(t, bantype.Banned, created.BanType)
		require.Equal(t, reason.Custom, created.Reason)
		require.Equal(t, "aimbot", created.ReasonText)
		require.Equal(t, ban.Web, created.Origin)
		require.Equal(t, "admin note", created.Note)
		require.NotNil(t, created.CIDR)
		require.Equal(t, "198.51.100.0/30", *created.CIDR)
		require.False(t, created.CreatedOn.IsZero())
		require.False(t, created.UpdatedOn.IsZero())

		fetched, err := harness.bans.QueryOne(t.Context(), ban.QueryOpts{BanID: created.BanID})
		require.NoError(t, err)
		require.Equal(t, created, fetched)
	})

	t.Run("stores the most recent player ip", func(t *testing.T) {
		t.Parallel()

		harness := newEnv(t, withIPProvider(staticIPProvider{ip: net.ParseIP("1.2.3.4")}))

		source, target := steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		createPerson(t, target)

		created, err := harness.bans.Create(t.Context(), ban.Opts{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})
		require.NoError(t, err)

		// host() strips the /32 netmask the driver stores on the inet column.
		var lastIP *string
		require.NoError(t, fixture.Database.QueryRow(t.Context(),
			"SELECT host(last_ip) FROM ban WHERE ban_id = $1", created.BanID).Scan(&lastIP))
		require.NotNil(t, lastIP)
		require.Equal(t, "1.2.3.4", *lastIP)
	})

	t.Run("no last ip when provider has none", func(t *testing.T) {
		t.Parallel()

		harness := newEnv(t)

		source, target := steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		createPerson(t, target)

		created, err := harness.bans.Create(t.Context(), ban.Opts{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})
		require.NoError(t, err)

		var lastIP *string
		require.NoError(t, fixture.Database.QueryRow(t.Context(),
			"SELECT last_ip FROM ban WHERE ban_id = $1", created.BanID).Scan(&lastIP))
		require.Nil(t, lastIP)
	})

	t.Run("invalid options are rejected", func(t *testing.T) {
		t.Parallel()

		harness := newEnv(t)

		source, target := steamid.RandSID64(), steamid.RandSID64()
		_, err := harness.bans.Create(t.Context(), ban.Opts{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(-time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})
		require.Error(t, err)
		require.ErrorIs(t, err, ban.ErrInvalidBanOpts)
		require.ErrorIs(t, err, ban.ErrInvalidBanDuration)
	})

	t.Run("duplicate ban is rejected", func(t *testing.T) {
		t.Parallel()

		harness := newEnv(t)

		source, target := steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		createPerson(t, target)

		opts := ban.Opts{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		}

		created, err := harness.bans.Create(t.Context(), opts)
		require.NoError(t, err)
		require.Positive(t, created.BanID)

		_, err = harness.bans.Create(t.Context(), opts)
		require.Error(t, err)
		require.ErrorIs(t, err, database.ErrDuplicate)
		require.ErrorIs(t, err, ban.ErrSaveBan)
	})

	t.Run("closes the attached report", func(t *testing.T) {
		t.Parallel()

		harness := newEnv(t)

		source, target := steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		createPerson(t, target)

		report := ban.Report{
			SourceID: source, TargetID: target, Description: "cheating evidence",
			ReportStatus: ban.Opened, CreatedOn: time.Now(), UpdatedOn: time.Now(),
		}
		require.NoError(t, harness.reportRepo.SaveReport(t.Context(), &report))

		created, err := harness.bans.Create(t.Context(), ban.Opts{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating, ReportID: &report.ReportID,
		})
		require.NoError(t, err)
		require.NotNil(t, created.ReportID)
		require.Equal(t, report.ReportID, *created.ReportID)

		savedReport, err := harness.reportRepo.GetReport(t.Context(), report.ReportID)
		require.NoError(t, err)
		require.Equal(t, ban.ClosedWithAction, savedReport.ReportStatus)
	})

	t.Run("sends ban notifications", func(t *testing.T) {
		t.Parallel()

		capt := &capturingNotifier{}
		harness := newEnv(t, withNotifier(capt))

		source, target := steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		createPerson(t, target)

		_, err := harness.bans.Create(t.Context(), ban.Opts{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating, Name: "cheater",
		})
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			return len(capt.List()) >= 3
		}, time.Second*3, time.Millisecond*10)

		var discordLogged, userNotified bool
		for _, payload := range capt.List() {
			if slices.Contains(payload.Types, notification.Discord) && payload.MessageSend != nil {
				discordLogged = true
			}

			if slices.Contains(payload.Sids, target) && payload.Severity == notification.Warn {
				userNotified = true
			}
		}

		require.True(t, discordLogged, "expected a discord log notification, got %+v", capt.List())
		require.True(t, userNotified, "expected a notification for the banned user, got %+v", capt.List())
	})
}

func TestBans_Unban(t *testing.T) {
	t.Parallel()

	t.Run("unbans an existing ban", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		author := createPerson(t, source)
		createPerson(t, target)

		created, err := e.bans.Create(ctx, ban.Opts{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})
		require.NoError(t, err)

		didUnban, err := e.bans.Unban(ctx, target, "took a shower", author)
		require.NoError(t, err)
		require.True(t, didUnban)

		fetched, err := e.bans.QueryOne(ctx, ban.QueryOpts{BanID: created.BanID, Deleted: true})
		require.NoError(t, err)
		require.True(t, fetched.Deleted)
		require.Equal(t, "took a shower", fetched.UnbanReasonText)
	})

	t.Run("unbanning again returns false", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		author := createPerson(t, source)
		createPerson(t, target)

		_, err := e.bans.Create(ctx, ban.Opts{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})
		require.NoError(t, err)

		didUnban, err := e.bans.Unban(ctx, target, "first", author)
		require.NoError(t, err)
		require.True(t, didUnban)

		// QueryOne returns ErrBanDoesNotExist, but Unban only maps
		// database.ErrNoResult to (false, nil), so a missing ban surfaces as
		// an error. Pinned until ban.go:453 is fixed.
		didUnban, err = e.bans.Unban(ctx, target, "second", author)
		require.False(t, didUnban)
		require.ErrorIs(t, err, ban.ErrBanDoesNotExist)
	})

	t.Run("unbanning an unknown player returns false", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)

		author := createPerson(t, steamid.RandSID64())
		// See "unbanning again": the no-ban case is not mapped to (false, nil).
		didUnban, err := e.bans.Unban(t.Context(), steamid.RandSID64(), "no such ban", author)
		require.False(t, didUnban)
		require.ErrorIs(t, err, ban.ErrBanDoesNotExist)
	})
}

func TestBans_Delete(t *testing.T) {
	t.Parallel()

	t.Run("soft delete", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		createPerson(t, target)

		created, err := e.bans.Create(ctx, ban.Opts{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})
		require.NoError(t, err)

		require.NoError(t, e.bans.Delete(ctx, &created, false))

		fetched, err := e.bans.QueryOne(ctx, ban.QueryOpts{BanID: created.BanID, Deleted: true})
		require.NoError(t, err)
		require.True(t, fetched.Deleted)

		_, err = e.bans.QueryOne(ctx, ban.QueryOpts{BanID: created.BanID})
		require.ErrorIs(t, err, ban.ErrBanDoesNotExist)
	})

	t.Run("hard delete", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		createPerson(t, target)

		created, err := e.bans.Create(ctx, ban.Opts{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})
		require.NoError(t, err)
		banID := created.BanID

		require.NoError(t, e.bans.Delete(ctx, &created, true))
		require.Zero(t, created.BanID)

		_, err = e.bans.QueryOne(ctx, ban.QueryOpts{BanID: banID, Deleted: true})
		require.ErrorIs(t, err, ban.ErrBanDoesNotExist)
	})
}

func TestBans_Expired(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	ctx := t.Context()

	source, target, validTarget := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()
	createPerson(t, source)
	createPerson(t, target)
	createPerson(t, validTarget)

	expired := ban.Ban{
		TargetID: target, SourceID: source, BanType: bantype.Banned, Reason: reason.Cheating,
		ValidUntil: time.Now().Add(-time.Hour),
	}
	e.saveRawBan(t, &expired)

	_, err := e.bans.Create(ctx, ban.Opts{
		SourceID: source, TargetID: validTarget, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating,
	})
	require.NoError(t, err)

	got, err := e.bans.Expired(ctx)
	require.NoError(t, err)
	require.Contains(t, banIDs(got), expired.BanID)
}

// Not parallel on purpose: ExpirationMonitor.Update deletes every expired ban
// in the shared database, which would clobber expired bans created by other,
// concurrently running tests (e.g. TestRepository_Query, TestBans_Expired).
// Running it sequentially ensures it finishes before the parallel batch starts.
func TestExpirationMonitor_Update(t *testing.T) {
	e := newEnv(t)
	ctx := t.Context()

	source, target, validTarget := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()
	createPerson(t, source)
	createPerson(t, target)
	createPerson(t, validTarget)

	expired := ban.Ban{
		TargetID: target, SourceID: source, BanType: bantype.Banned, Reason: reason.Cheating,
		ValidUntil: time.Now().Add(-time.Hour),
	}
	e.saveRawBan(t, &expired)

	valid := ban.Ban{
		TargetID: validTarget, SourceID: source, BanType: bantype.Banned, Reason: reason.Cheating,
		ValidUntil: time.Now().Add(time.Hour),
	}
	e.saveRawBan(t, &valid)

	monitor := ban.NewExpirationMonitor(e.bans, fixture.Persons, notification.NewDiscard())
	monitor.Update(ctx)

	fetchedExpired, err := e.bans.QueryOne(ctx, ban.QueryOpts{BanID: expired.BanID, Deleted: true})
	require.NoError(t, err)
	require.True(t, fetchedExpired.Deleted)

	fetchedValid, err := e.bans.QueryOne(ctx, ban.QueryOpts{BanID: valid.BanID})
	require.NoError(t, err)
	require.False(t, fetchedValid.Deleted)
}

func TestBans_CheckEvadeStatus(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	ctx := t.Context()

	var (
		source       = steamid.RandSID64()
		bannedPlayer = steamid.RandSID64()
		evader       = steamid.RandSID64()
	)
	for _, sid := range []steamid.SteamID{source, bannedPlayer, evader} {
		createPerson(t, sid)
	}

	bannedCIDR := "192.0.2.16/28"
	_, err := e.bans.Create(ctx, ban.Opts{
		SourceID: source, TargetID: bannedPlayer, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating, CIDR: &bannedCIDR,
	})
	require.NoError(t, err)

	// 192.0.2.21 is inside the banned range, so the evader gets an evasion ban.
	t.Run("cidr match creates evade ban", func(t *testing.T) {
		t.Parallel()

		evadeBanned, err := e.bans.CheckEvadeStatus(ctx, evader, netip.MustParseAddr("192.0.2.21"))
		require.NoError(t, err)
		require.True(t, evadeBanned)

		evadeBan, err := e.bans.QueryOne(ctx, ban.QueryOpts{TargetID: evader})
		require.NoError(t, err)
		require.Equal(t, bantype.Banned, evadeBan.BanType)
		require.Equal(t, reason.Evading, evadeBan.Reason)
	})
}
