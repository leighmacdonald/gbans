package ban_test

import (
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func newTestReport(t *testing.T, source, target steamid.SteamID, status ban.ReportStatus) ban.Report {
	t.Helper()

	report := ban.NewReport()
	report.SourceID = source
	report.TargetID = target
	report.Description = "cheating with aimbot"
	report.ReportStatus = status
	report.Reason = reason.Cheating
	report.ReasonText = "aimbot"

	return report
}

func TestReportRepository_SaveReport(t *testing.T) {
	t.Parallel()

	t.Run("insert sets the report id", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		createPerson(t, target)

		report := newTestReport(t, source, target, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))
		require.Positive(t, report.ReportID)
	})

	t.Run("update persists changes", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		createPerson(t, target)

		report := newTestReport(t, source, target, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

		report.ReportStatus = ban.ClosedWithAction
		report.Description = "confirmed cheating"
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

		fetched, err := e.reportRepo.GetReport(ctx, report.ReportID)
		require.NoError(t, err)
		require.Equal(t, ban.ClosedWithAction, fetched.ReportStatus)
		require.Equal(t, "confirmed cheating", fetched.Description)
	})
}

func TestReportRepository_GetReports(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	ctx := t.Context()

	author, target, otherAuthor := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()
	createPerson(t, author)
	createPerson(t, target)
	createPerson(t, otherAuthor)

	open := newTestReport(t, author, target, ban.Opened)
	require.NoError(t, e.reportRepo.SaveReport(ctx, &open))

	closed := newTestReport(t, author, target, ban.ClosedWithAction)
	// GetReportBySteamID only considers open reports, so a second target keeps
	// the rows distinct from a "one open report per pair" perspective.
	closedTarget := steamid.RandSID64()
	createPerson(t, closedTarget)
	closed.TargetID = closedTarget
	require.NoError(t, e.reportRepo.SaveReport(ctx, &closed))

	otherReport := newTestReport(t, otherAuthor, target, ban.Opened)
	require.NoError(t, e.reportRepo.SaveReport(ctx, &otherReport))

	// Sequential on purpose: a later subtest drops a shared report.
	t.Run("reports for the author", func(t *testing.T) {
		got, err := e.reportRepo.GetReports(ctx, author)
		require.NoError(t, err)
		require.ElementsMatch(t, []int32{open.ReportID, closed.ReportID}, reportIDs(got))
	})

	t.Run("all reports for invalid steam id", func(t *testing.T) {
		got, err := e.reportRepo.GetReports(ctx, steamid.SteamID{})
		require.NoError(t, err)
		require.Contains(t, reportIDs(got), open.ReportID)
		require.Contains(t, reportIDs(got), otherReport.ReportID)
	})

	t.Run("dropped reports are excluded", func(t *testing.T) {
		require.NoError(t, e.reportRepo.DropReport(ctx, &open))

		got, err := e.reportRepo.GetReports(ctx, author)
		require.NoError(t, err)
		require.Equal(t, []int32{closed.ReportID}, reportIDs(got))
	})
}

func TestReportRepository_GetReport(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	ctx := t.Context()

	source, target := steamid.RandSID64(), steamid.RandSID64()
	createPerson(t, source)
	createPerson(t, target)

	server := fixture.CreateTestServer(ctx)
	require.NoError(t, fixture.Database.Exec(ctx,
		"INSERT INTO person_messages (person_message_id, steam_id, server_id, body, persona_name, created_on) "+
			"VALUES (77, $1, $2, 'chat line', 'chat name', now())",
		source.Int64(), server.ServerID))

	report := newTestReport(t, source, target, ban.Opened)
	report.DemoTick = 1234
	report.PersonMessageID = 77
	require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

	// Sequential on purpose: the last subtest drops the shared report.
	t.Run("round trip", func(t *testing.T) {
		fetched, err := e.reportRepo.GetReport(ctx, report.ReportID)
		require.NoError(t, err)
		require.Equal(t, source, fetched.SourceID)
		require.Equal(t, target, fetched.TargetID)
		require.Equal(t, "cheating with aimbot", fetched.Description)
		require.Equal(t, ban.Opened, fetched.ReportStatus)
		require.Equal(t, reason.Cheating, fetched.Reason)
		require.Equal(t, "aimbot", fetched.ReasonText)
		require.Equal(t, int32(1234), fetched.DemoTick)
		require.Equal(t, int64(77), fetched.PersonMessageID)
		require.False(t, fetched.Deleted)
	})

	t.Run("unknown report is rejected", func(t *testing.T) {
		_, err := e.reportRepo.GetReport(ctx, 99999999)
		require.Error(t, err)
		require.ErrorIs(t, err, database.ErrNoResult)
	})

	t.Run("dropped reports are rejected", func(t *testing.T) {
		require.NoError(t, e.reportRepo.DropReport(ctx, &report))

		_, err := e.reportRepo.GetReport(ctx, report.ReportID)
		require.Error(t, err)
		require.ErrorIs(t, err, database.ErrNoResult)
	})
}

func TestReportRepository_GetReportBySteamID(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	ctx := t.Context()

	author, target, otherAuthor := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()
	createPerson(t, author)
	createPerson(t, target)
	createPerson(t, otherAuthor)

	open := newTestReport(t, author, target, ban.Opened)
	require.NoError(t, e.reportRepo.SaveReport(ctx, &open))

	// Sequential on purpose: the subtests create reports for overlapping
	// author/target pairs, so order matters.
	t.Run("different author is not found", func(t *testing.T) {
		_, err := e.reportRepo.GetReportBySteamID(ctx, otherAuthor, target)
		require.Error(t, err)
		require.ErrorIs(t, err, database.ErrNoResult)
	})

	t.Run("open report is found", func(t *testing.T) {
		fetched, err := e.reportRepo.GetReportBySteamID(ctx, author, target)
		require.NoError(t, err)
		require.Equal(t, open.ReportID, fetched.ReportID)
	})

	t.Run("need more info report is found", func(t *testing.T) {
		needInfo := newTestReport(t, otherAuthor, target, ban.NeedMoreInfo)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &needInfo))

		fetched, err := e.reportRepo.GetReportBySteamID(ctx, otherAuthor, target)
		require.NoError(t, err)
		require.Equal(t, needInfo.ReportID, fetched.ReportID)
	})

	t.Run("closed report is not found", func(t *testing.T) {
		closedTarget := steamid.RandSID64()
		createPerson(t, closedTarget)

		closed := newTestReport(t, author, closedTarget, ban.ClosedWithoutAction)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &closed))

		_, err := e.reportRepo.GetReportBySteamID(ctx, author, closedTarget)
		require.Error(t, err)
		require.ErrorIs(t, err, database.ErrNoResult)
	})
}

func TestReportRepository_Messages(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	ctx := t.Context()

	source, target, author := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()
	createPerson(t, source)
	createPerson(t, target)
	createPerson(t, author)

	report := newTestReport(t, source, target, ban.Opened)
	require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

	base := time.Now().Add(-time.Hour)

	msg1 := ban.NewReportMessage(report.ReportID, author, "first message")
	msg1.CreatedOn = base
	msg1.UpdatedOn = base
	require.NoError(t, e.reportRepo.SaveReportMessage(ctx, &msg1))

	msg2 := ban.NewReportMessage(report.ReportID, author, "second message")
	msg2.CreatedOn = base.Add(time.Second)
	msg2.UpdatedOn = base.Add(time.Second)
	require.NoError(t, e.reportRepo.SaveReportMessage(ctx, &msg2))

	setPersonName(t, author, "reporter", "reporter-avatar")

	// Sequential on purpose: later subtests mutate the shared messages.
	t.Run("messages are returned in created order", func(t *testing.T) {
		got, err := e.reportRepo.GetReportMessages(ctx, report.ReportID)
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Equal(t, msg1.ReportMessageID, got[0].ReportMessageID)
		require.Equal(t, msg2.ReportMessageID, got[1].ReportMessageID)
		require.Equal(t, author, got[0].AuthorID)
		require.Equal(t, "reporter", got[0].Personaname)
		require.Equal(t, "reporter-avatar", got[0].Avatarhash)
	})

	t.Run("message by id round trips", func(t *testing.T) {
		fetched, err := e.reportRepo.GetReportMessageByID(ctx, msg2.ReportMessageID)
		require.NoError(t, err)
		require.Equal(t, msg2.ReportMessageID, fetched.ReportMessageID)
		require.Equal(t, report.ReportID, fetched.ReportID)
		require.Equal(t, "second message", fetched.MessageMD)
	})

	t.Run("dropped messages are excluded", func(t *testing.T) {
		require.NoError(t, e.reportRepo.DropReportMessage(ctx, &msg1))
		require.True(t, msg1.Deleted)

		got, err := e.reportRepo.GetReportMessages(ctx, report.ReportID)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, msg2.ReportMessageID, got[0].ReportMessageID)

		fetched, err := e.reportRepo.GetReportMessageByID(ctx, msg1.ReportMessageID)
		require.NoError(t, err)
		require.True(t, fetched.Deleted)
	})

	t.Run("update persists changes", func(t *testing.T) {
		msg2.MessageMD = "second message edited"
		require.NoError(t, e.reportRepo.SaveReportMessage(ctx, &msg2))

		fetched, err := e.reportRepo.GetReportMessageByID(ctx, msg2.ReportMessageID)
		require.NoError(t, err)
		require.Equal(t, "second message edited", fetched.MessageMD)
	})
}

func reportIDs(reports []ban.Report) []int32 {
	ids := make([]int32, 0, len(reports))
	for _, r := range reports {
		ids = append(ids, r.ReportID)
	}

	return ids
}
