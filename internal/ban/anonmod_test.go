package ban_test

import (
	"strings"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

// enableAnonmod flips the shared in-memory config to enable the anonymized
// moderator feature for the duration of the test. Configuration.Write cannot
// be used because the test config has no SteamKey (which reload requires).
func enableAnonmod(t *testing.T) {
	t.Helper()

	general := fixture.Config.Config().General

	general.Lock()
	general.AnonmodEnabled = true
	general.Unlock()

	t.Cleanup(func() {
		general.Lock()
		general.AnonmodEnabled = false
		general.Unlock()
	})
}

func findReportMessage(t *testing.T, messages []ban.ReportMessage, reportMessageID int32) ban.ReportMessage {
	t.Helper()

	for _, msg := range messages {
		if msg.ReportMessageID == reportMessageID {
			return msg
		}
	}

	t.Fatalf("report message %d not found in %d messages", reportMessageID, len(messages))

	return ban.ReportMessage{}
}

func findAppealMessage(t *testing.T, messages []ban.AppealMessage, banMessageID int64) ban.AppealMessage {
	t.Helper()

	for _, msg := range messages {
		if msg.BanMessageID == banMessageID {
			return msg
		}
	}

	t.Fatalf("appeal message %d not found in %d messages", banMessageID, len(messages))

	return ban.AppealMessage{}
}

func TestAnonmod_Identity(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	anonmod := ban.NewAnonmod(ban.NewAnonmodRepository(fixture.Database), fixture.Config)

	modA, modB := steamid.RandSID64(), steamid.RandSID64()

	t.Run("stable for the same author and issue", func(t *testing.T) {
		t.Parallel()

		first, err := anonmod.Identity(ctx, ban.AnonmodIssueTypeReport, 1, modA)
		require.NoError(t, err)
		require.NotEmpty(t, first.AnonName)
		require.Len(t, first.AnonAvatarHash, 40)

		second, err := anonmod.Identity(ctx, ban.AnonmodIssueTypeReport, 1, modA)
		require.NoError(t, err)
		require.Equal(t, first, second)
	})

	t.Run("distinct for different authors in the same issue", func(t *testing.T) {
		t.Parallel()

		identityA, errA := anonmod.Identity(ctx, ban.AnonmodIssueTypeAppeal, 2, modA)
		require.NoError(t, errA)
		identityB, errB := anonmod.Identity(ctx, ban.AnonmodIssueTypeAppeal, 2, modB)
		require.NoError(t, errB)

		require.NotEqual(t, strings.ToLower(identityA.AnonName), strings.ToLower(identityB.AnonName))
		require.NotEqual(t, identityA.AnonAvatarHash, identityB.AnonAvatarHash)
	})

	t.Run("independent per issue", func(t *testing.T) {
		t.Parallel()

		reportIdentity, err := anonmod.Identity(ctx, ban.AnonmodIssueTypeReport, 3, modA)
		require.NoError(t, err)
		appealIdentity, err := anonmod.Identity(ctx, ban.AnonmodIssueTypeAppeal, 3, modA)
		require.NoError(t, err)

		// Both rows exist and remain stable on re-read.
		reportRows, err := ban.NewAnonmodRepository(fixture.Database).Identities(ctx, ban.AnonmodIssueTypeReport, 3)
		require.NoError(t, err)
		require.Len(t, reportRows, 1)
		require.Equal(t, reportIdentity, reportRows[0])

		appealRows, err := ban.NewAnonmodRepository(fixture.Database).Identities(ctx, ban.AnonmodIssueTypeAppeal, 3)
		require.NoError(t, err)
		require.Len(t, appealRows, 1)
		require.Equal(t, appealIdentity, appealRows[0])
	})
}

// Not parallel on purpose: it toggles the shared in-memory config flag, which
// other tests observe through the same *config.Configuration.
func TestAnonmod_ReportMessages(t *testing.T) {
	ctx := t.Context()
	e := newEnv(t)

	anonmod := ban.NewAnonmod(ban.NewAnonmodRepository(fixture.Database), fixture.Config)

	reporter := createPerson(t, steamid.RandSID64())
	target := steamid.RandSID64()
	createPerson(t, target)

	mod1 := steamid.RandSID64()
	createPerson(t, mod1)
	// REPORT_ADMIN both allows replying to reports and marks the user as a
	// moderator for the anonymization views.
	e.createAndGrantRole(t, mod1, "PERMISSION_REPORT_ADMIN")
	mod1Person := createPerson(t, mod1)

	mod2 := steamid.RandSID64()
	createPerson(t, mod2)
	e.createAndGrantRole(t, mod2, "PERMISSION_REPORT_ADMIN")
	mod2Person := createPerson(t, mod2)

	savedReport, err := e.reports.Save(ctx, reporter, ban.RequestReportCreate{
		TargetID:    target,
		Description: "cheating with aimbot",
	})
	require.NoError(t, err)
	reportID := savedReport.ReportID

	reporterMsg, err := e.reports.CreateMessage(ctx, reportID, reporter, ban.RequestMessageBodyMD{BodyMD: "reporter detail"})
	require.NoError(t, err)
	mod1Msg, err := e.reports.CreateMessage(ctx, reportID, mod1Person, ban.RequestMessageBodyMD{BodyMD: "mod one reply"})
	require.NoError(t, err)
	mod2Msg, err := e.reports.CreateMessage(ctx, reportID, mod2Person, ban.RequestMessageBodyMD{BodyMD: "mod two reply"})
	require.NoError(t, err)

	t.Run("feature off passes messages through", func(t *testing.T) {
		got, err := e.reports.Messages(ctx, reportID, reporter)
		require.NoError(t, err)
		require.Len(t, got, 3)

		for _, msg := range got {
			require.True(t, msg.AuthorID.Valid())
			require.Empty(t, msg.AnonPersonaname)
			require.Empty(t, msg.AnonAvatarhash)
		}
	})

	enableAnonmod(t)

	identityMod1, err := anonmod.Identity(ctx, ban.AnonmodIssueTypeReport, reportID, mod1)
	require.NoError(t, err)
	identityMod2, err := anonmod.Identity(ctx, ban.AnonmodIssueTypeReport, reportID, mod2)
	require.NoError(t, err)
	require.NotEqual(t, identityMod1.AnonName, identityMod2.AnonName)

	t.Run("non-mod viewer sees anonymized mod replies", func(t *testing.T) {
		got, err := e.reports.Messages(ctx, reportID, reporter)
		require.NoError(t, err)

		gotReporter := findReportMessage(t, got, reporterMsg.ReportMessageID)
		require.Equal(t, reporter.SteamID, gotReporter.AuthorID)
		require.Empty(t, gotReporter.AnonPersonaname)

		gotMod1 := findReportMessage(t, got, mod1Msg.ReportMessageID)
		require.Zero(t, gotMod1.AuthorID)
		require.Equal(t, identityMod1.AnonName, gotMod1.Personaname)
		require.Equal(t, identityMod1.AnonAvatarHash, gotMod1.Avatarhash)
		require.Empty(t, gotMod1.AnonPersonaname)

		gotMod2 := findReportMessage(t, got, mod2Msg.ReportMessageID)
		require.Zero(t, gotMod2.AuthorID)
		require.Equal(t, identityMod2.AnonName, gotMod2.Personaname)
		require.Equal(t, identityMod2.AnonAvatarHash, gotMod2.Avatarhash)
	})

	t.Run("mod viewer sees real authors plus persona reference", func(t *testing.T) {
		got, err := e.reports.Messages(ctx, reportID, mod1Person)
		require.NoError(t, err)

		gotReporter := findReportMessage(t, got, reporterMsg.ReportMessageID)
		require.Equal(t, reporter.SteamID, gotReporter.AuthorID)
		require.Empty(t, gotReporter.AnonPersonaname)

		gotMod1 := findReportMessage(t, got, mod1Msg.ReportMessageID)
		require.Equal(t, mod1, gotMod1.AuthorID)
		require.Equal(t, identityMod1.AnonName, gotMod1.AnonPersonaname)
		require.Equal(t, identityMod1.AnonAvatarHash, gotMod1.AnonAvatarhash)

		gotMod2 := findReportMessage(t, got, mod2Msg.ReportMessageID)
		require.Equal(t, mod2, gotMod2.AuthorID)
		require.Equal(t, identityMod2.AnonName, gotMod2.AnonPersonaname)
		require.Equal(t, identityMod2.AnonAvatarHash, gotMod2.AnonAvatarhash)
	})
}

// Not parallel on purpose: it toggles the shared in-memory config flag, which
// other tests observe through the same *config.Configuration.
func TestAnonmod_AppealMessages(t *testing.T) {
	ctx := t.Context()
	e := newEnv(t)

	anonmod := ban.NewAnonmod(ban.NewAnonmodRepository(fixture.Database), fixture.Config)

	source, target := steamid.RandSID64(), steamid.RandSID64()
	createPerson(t, source)
	targetPerson := createPerson(t, target)

	mod1 := steamid.RandSID64()
	createPerson(t, mod1)
	// APPEAL_WRITE is the posting permission check; APPEAL_READ marks the
	// user as a moderator for the anonymization views.
	e.createAndGrantRole(t, mod1, "PERMISSION_APPEAL_WRITE", "PERMISSION_APPEAL_READ")
	mod1Person := createPerson(t, mod1)

	created := e.saveRawBan(t, &ban.Ban{
		SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating,
	})
	banID := created.BanID

	targetMsg, err := e.appeals.CreateBanMessage(ctx, targetPerson, banID, "I did not cheat")
	require.NoError(t, err)
	require.Empty(t, targetMsg.AnonPersonaname)

	t.Run("feature off passes messages through", func(t *testing.T) {
		got, err := e.appeals.Messages(ctx, targetPerson, banID)
		require.NoError(t, err)
		require.Len(t, got, 1)

		gotTarget := findAppealMessage(t, got, targetMsg.BanMessageID)
		require.Equal(t, target, gotTarget.AuthorID)
		require.Empty(t, gotTarget.AnonPersonaname)
	})

	enableAnonmod(t)

	// A mod reply created while the feature is on carries the persona
	// reference in the response.
	mod1Msg, err := e.appeals.CreateBanMessage(ctx, mod1Person, banID, "under review")
	require.NoError(t, err)
	require.Equal(t, mod1, mod1Msg.AuthorID)
	require.NotEmpty(t, mod1Msg.AnonPersonaname)
	require.Len(t, mod1Msg.AnonAvatarhash, 40)

	identityMod1, err := anonmod.Identity(ctx, ban.AnonmodIssueTypeAppeal, banID, mod1)
	require.NoError(t, err)
	require.Equal(t, identityMod1.AnonName, mod1Msg.AnonPersonaname)
	require.Equal(t, identityMod1.AnonAvatarHash, mod1Msg.AnonAvatarhash)

	t.Run("non-mod viewer sees anonymized mod replies", func(t *testing.T) {
		got, err := e.appeals.Messages(ctx, targetPerson, banID)
		require.NoError(t, err)
		require.Len(t, got, 2)

		gotTarget := findAppealMessage(t, got, targetMsg.BanMessageID)
		require.Equal(t, target, gotTarget.AuthorID)
		require.Empty(t, gotTarget.AnonPersonaname)

		gotMod1 := findAppealMessage(t, got, mod1Msg.BanMessageID)
		require.Zero(t, gotMod1.AuthorID)
		require.Equal(t, identityMod1.AnonName, gotMod1.Personaname)
		require.Equal(t, identityMod1.AnonAvatarHash, gotMod1.Avatarhash)
		require.Empty(t, gotMod1.AnonPersonaname)
	})

	t.Run("mod viewer sees real authors plus persona reference", func(t *testing.T) {
		got, err := e.appeals.Messages(ctx, mod1Person, banID)
		require.NoError(t, err)
		require.Len(t, got, 2)

		gotTarget := findAppealMessage(t, got, targetMsg.BanMessageID)
		require.Equal(t, target, gotTarget.AuthorID)
		require.Empty(t, gotTarget.AnonPersonaname)

		gotMod1 := findAppealMessage(t, got, mod1Msg.BanMessageID)
		require.Equal(t, mod1, gotMod1.AuthorID)
		require.Equal(t, identityMod1.AnonName, gotMod1.AnonPersonaname)
		require.Equal(t, identityMod1.AnonAvatarHash, gotMod1.AnonAvatarhash)
	})
}
