package ban_test

import (
	"slices"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestReportStatus_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status ban.ReportStatus
		want   string
	}{
		{name: "opened", status: ban.Opened, want: "Opened"},
		{name: "need more info", status: ban.NeedMoreInfo, want: "Need more information"},
		{name: "closed without action", status: ban.ClosedWithoutAction, want: "Closed without action"},
		{name: "closed with action", status: ban.ClosedWithAction, want: "Closed with action"},
		{name: "any status", status: ban.ReportStatus(-1), want: "Need more information"},
		{name: "unknown", status: ban.ReportStatus(99), want: "Need more information"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.status.String())
		})
	}
}

func TestNewReport(t *testing.T) {
	report := ban.NewReport()

	require.Zero(t, report.ReportID)
	require.Equal(t, ban.Opened, report.ReportStatus)
	require.Equal(t, int32(-1), report.DemoTick)
	require.Zero(t, report.DemoID)
	require.False(t, report.CreatedOn.IsZero())
	require.False(t, report.UpdatedOn.IsZero())
}

func TestReport_Path(t *testing.T) {
	require.Equal(t, "/report/42", ban.Report{ReportID: 42}.Path())
}

func TestReportMessage_Path(t *testing.T) {
	require.Equal(t, "/report/42#7", ban.ReportMessage{ReportID: 42, ReportMessageID: 7}.Path())
}

func TestNewReportMessage(t *testing.T) {
	msg := ban.NewReportMessage(42, steamid.New(76561198000000001), "body")

	require.Equal(t, int32(42), msg.ReportID)
	require.Equal(t, steamid.New(76561198000000001), msg.AuthorID)
	require.Equal(t, "body", msg.MessageMD)
	require.False(t, msg.Deleted)
	require.False(t, msg.CreatedOn.IsZero())
	require.False(t, msg.UpdatedOn.IsZero())
}

func TestReports_Save(t *testing.T) {
	t.Parallel()

	t.Run("creates a report", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		saved, err := e.reports.Save(ctx, author, ban.RequestReportCreate{
			TargetID:    target,
			Description: "cheating with aimbot",
			Reason:      reason.Cheating,
		})
		require.NoError(t, err)
		require.Positive(t, saved.ReportID)
		require.Equal(t, ban.Opened, saved.ReportStatus)
		require.Equal(t, author.SteamID, saved.SourceID)
		require.Equal(t, target, saved.TargetID)
		require.Equal(t, author.SteamID, saved.Author.SteamID)
		require.Equal(t, target, saved.Subject.SteamID)
	})

	t.Run("short description is rejected", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)

		author := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		_, err := e.reports.Save(t.Context(), author, ban.RequestReportCreate{
			TargetID:    target,
			Description: "too short",
		})
		require.Error(t, err)
		require.ErrorIs(t, err, httphelper.ErrParamInvalid)
	})

	t.Run("missing target is rejected", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)

		author := createPerson(t, steamid.RandSID64())
		_, err := e.reports.Save(t.Context(), author, ban.RequestReportCreate{
			Description: "cheating with aimbot",
		})
		require.Error(t, err)
		require.ErrorIs(t, err, httphelper.ErrParamInvalid)
	})

	t.Run("self report is rejected", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)

		author := createPerson(t, steamid.RandSID64())
		_, err := e.reports.Save(t.Context(), author, ban.RequestReportCreate{
			TargetID:    author.SteamID,
			Description: "cheating with aimbot",
		})
		require.Error(t, err)
		require.ErrorIs(t, err, httphelper.ErrParamInvalid)
	})

	t.Run("duplicate open report is rejected", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		req := ban.RequestReportCreate{
			TargetID:    target,
			Description: "cheating with aimbot",
			Reason:      reason.Cheating,
		}

		_, err := e.reports.Save(ctx, author, req)
		require.NoError(t, err)

		_, err = e.reports.Save(ctx, author, req)
		require.Error(t, err)
		require.ErrorIs(t, err, ban.ErrReportExists)
	})

	t.Run("a closed report allows a new report", func(t *testing.T) {
		t.Parallel()

		harness := newEnv(t)
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		req := ban.RequestReportCreate{
			TargetID:    target,
			Description: "cheating with aimbot",
			Reason:      reason.Cheating,
		}

		first, err := harness.reports.Save(ctx, author, req)
		require.NoError(t, err)

		_, err = harness.reports.SetReportStatus(ctx, first.ReportID, author, ban.ClosedWithAction)
		require.NoError(t, err)

		second, err := harness.reports.Save(ctx, author, req)
		require.NoError(t, err)
		require.NotEqual(t, first.ReportID, second.ReportID)
	})

	t.Run("unknown demo is rejected", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)

		author := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		_, err := e.reports.Save(t.Context(), author, ban.RequestReportCreate{
			TargetID:    target,
			Description: "cheating with aimbot",
			DemoID:      999999,
		})
		require.Error(t, err)
	})

	t.Run("sends notifications", func(t *testing.T) {
		t.Parallel()

		capt := &capturingNotifier{}
		e := newEnv(t, withNotifier(capt))
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		_, err := e.reports.Save(ctx, author, ban.RequestReportCreate{
			TargetID:    target,
			Description: "cheating with aimbot",
		})
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			return len(capt.List()) >= 2
		}, time.Second*3, time.Millisecond*10)

		var discordSent, groupSent bool
		for _, payload := range capt.List() {
			switch {
			// The report_new discord template is not initialised in tests, so
			// the payload carries a nil MessageSend; assert on the type only.
			case slices.Contains(payload.Types, notification.Discord):
				discordSent = true
			case slices.Contains(payload.Groups, rolesv1.Permission_PERMISSION_BAN_READ) &&
				payload.Message == "A new report was created. Author: "+author.GetName()+", Target: name-"+target.String():
				// The author person comes from the creation path, whose name is
				// never persisted (person.updatePerson discards ApplySteamInfo),
				// so GetName falls back to the steam id. The target is re-fetched
				// and carries the fake API name in memory.
				groupSent = true
			}
		}

		require.True(t, discordSent, "expected discord notification, got %+v", capt.List())
		require.True(t, groupSent, "expected group notification, got %+v", capt.List())
	})
}

func TestReports_Report(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	ctx := t.Context()

	author := createPerson(t, steamid.RandSID64())
	stranger := createPerson(t, steamid.RandSID64())
	target := steamid.RandSID64()
	createPerson(t, target)

	admin := steamid.RandSID64()
	createPerson(t, admin)
	e.createAndGrantRole(t, admin, "PERMISSION_REPORT_ADMIN")
	adminPerson := createPerson(t, admin)

	report := newTestReport(t, author.SteamID, target, ban.Opened)
	require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

	t.Run("author can read", func(t *testing.T) {
		t.Parallel()

		got, err := e.reports.Report(ctx, author, report.ReportID)
		require.NoError(t, err)
		require.Equal(t, report.ReportID, got.ReportID)
		require.Equal(t, author.SteamID, got.Author.SteamID)
		require.Equal(t, target, got.Subject.SteamID)
	})

	t.Run("report admin can read", func(t *testing.T) {
		t.Parallel()

		got, err := e.reports.Report(ctx, adminPerson, report.ReportID)
		require.NoError(t, err)
		require.Equal(t, report.ReportID, got.ReportID)
	})

	t.Run("other users are denied", func(t *testing.T) {
		t.Parallel()

		_, err := e.reports.Report(ctx, stranger, report.ReportID)
		require.ErrorIs(t, err, rpc.ErrPermission)
	})

	t.Run("unknown report is rejected", func(t *testing.T) {
		t.Parallel()

		_, err := e.reports.Report(ctx, author, 99999999)
		require.Error(t, err)
		require.ErrorIs(t, err, database.ErrNoResult)
	})

	t.Run("dropped report is rejected", func(t *testing.T) {
		droppedTarget := steamid.RandSID64()
		createPerson(t, droppedTarget)
		dropped := newTestReport(t, author.SteamID, droppedTarget, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &dropped))
		require.NoError(t, e.reportRepo.DropReport(ctx, &dropped))

		_, err := e.reports.Report(ctx, author, dropped.ReportID)
		require.Error(t, err)
		require.ErrorIs(t, err, database.ErrNoResult)
	})
}

func TestReports_SetReportStatus(t *testing.T) {
	t.Parallel()

	t.Run("author can change the status", func(t *testing.T) {
		t.Parallel()

		capt := &capturingNotifier{}
		e := newEnv(t, withNotifier(capt))
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		report := newTestReport(t, author.SteamID, target, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

		updated, err := e.reports.SetReportStatus(ctx, report.ReportID, author, ban.ClosedWithAction)
		require.NoError(t, err)
		require.Equal(t, ban.ClosedWithAction, updated.ReportStatus)

		fetched, err := e.reportRepo.GetReport(ctx, report.ReportID)
		require.NoError(t, err)
		require.Equal(t, ban.ClosedWithAction, fetched.ReportStatus)

		require.Eventually(t, func() bool {
			var userNotified bool
			for _, payload := range capt.List() {
				if slices.Contains(payload.Sids, author.SteamID) &&
					payload.Message == "Your report status has changed: Opened -> Closed with action" {
					userNotified = true
				}
			}

			return userNotified
		}, time.Second*3, time.Millisecond*10)
	})

	t.Run("same status is a no op", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		report := newTestReport(t, author.SteamID, target, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

		updated, err := e.reports.SetReportStatus(ctx, report.ReportID, author, ban.Opened)
		require.NoError(t, err)
		require.Equal(t, ban.Opened, updated.ReportStatus)
	})

	t.Run("other users are denied", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		stranger := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		report := newTestReport(t, author.SteamID, target, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

		_, err := e.reports.SetReportStatus(ctx, report.ReportID, stranger, ban.ClosedWithAction)
		require.ErrorIs(t, err, rpc.ErrPermission)

		fetched, err := e.reportRepo.GetReport(ctx, report.ReportID)
		require.NoError(t, err)
		require.Equal(t, ban.Opened, fetched.ReportStatus)
	})
}

func TestReports_BySteamID(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	ctx := t.Context()

	author := steamid.RandSID64()
	target := steamid.RandSID64()
	createPerson(t, author)
	createPerson(t, target)

	report := newTestReport(t, author, target, ban.Opened)
	require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

	t.Run("reports for the steam id", func(t *testing.T) {
		t.Parallel()

		got, err := e.reports.BySteamID(ctx, author)
		require.NoError(t, err)
		require.Contains(t, reportIDs(gotReports(got)), report.ReportID)
	})

	t.Run("unknown user has no reports", func(t *testing.T) {
		t.Parallel()

		got, err := e.reports.BySteamID(ctx, steamid.RandSID64())
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("invalid steam id is rejected", func(t *testing.T) {
		t.Parallel()

		_, err := e.reports.BySteamID(ctx, steamid.SteamID{})
		require.ErrorIs(t, err, steamid.ErrInvalidSID)
	})
}

func TestReports_Reports(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	ctx := t.Context()

	author := steamid.RandSID64()
	target := steamid.RandSID64()
	createPerson(t, author)
	createPerson(t, target)

	report := newTestReport(t, author, target, ban.Opened)
	require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

	got, err := e.reports.Reports(ctx)
	require.NoError(t, err)
	require.Contains(t, reportIDs(gotReports(got)), report.ReportID)
}

func TestReports_CreateMessage(t *testing.T) {
	t.Parallel()

	t.Run("author can create a message", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		report := newTestReport(t, author.SteamID, target, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

		msg, err := e.reports.CreateMessage(ctx, report.ReportID, author, ban.RequestMessageBodyMD{BodyMD: "  more detail here  "})
		require.NoError(t, err)
		require.Positive(t, msg.ReportMessageID)
		require.Equal(t, "more detail here", msg.MessageMD)

		messages, err := e.reportRepo.GetReportMessages(ctx, report.ReportID)
		require.NoError(t, err)
		require.Len(t, messages, 1)
	})

	t.Run("empty body is rejected", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		report := newTestReport(t, author.SteamID, target, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

		_, err := e.reports.CreateMessage(ctx, report.ReportID, author, ban.RequestMessageBodyMD{BodyMD: "  "})
		require.ErrorIs(t, err, httphelper.ErrParamInvalid)
	})

	t.Run("other users are denied", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		stranger := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		report := newTestReport(t, author.SteamID, target, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

		_, err := e.reports.CreateMessage(ctx, report.ReportID, stranger, ban.RequestMessageBodyMD{BodyMD: "sneaky reply"})
		require.ErrorIs(t, err, rpc.ErrPermission)
	})

	t.Run("notifies group and author for foreign replies", func(t *testing.T) {
		t.Parallel()

		capt := &capturingNotifier{}
		e := newEnv(t, withNotifier(capt))
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		admin := steamid.RandSID64()
		createPerson(t, admin)
		e.createAndGrantRole(t, admin, "PERMISSION_REPORT_ADMIN")
		adminPerson := createPerson(t, admin)

		report := newTestReport(t, author.SteamID, target, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

		_, err := e.reports.CreateMessage(ctx, report.ReportID, adminPerson, ban.RequestMessageBodyMD{BodyMD: "official reply"})
		require.NoError(t, err)

		var groupSent, authorSent bool
		for _, payload := range capt.List() {
			switch {
			case slices.Contains(payload.Groups, rolesv1.Permission_PERMISSION_BAN_READ) &&
				payload.Message == "A new report reply has been posted. Author: "+adminPerson.GetName():
				groupSent = true
			case slices.Contains(payload.Sids, author.SteamID) && payload.Message == "A new report reply has been posted":
				authorSent = true
			}
		}

		require.True(t, groupSent, "expected group notification, got %+v", capt.List())
		require.True(t, authorSent, "expected author notification, got %+v", capt.List())
	})
}

func TestReports_EditMessage(t *testing.T) {
	t.Parallel()

	t.Run("author can edit their message", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		report := newTestReport(t, author.SteamID, target, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

		msg, err := e.reports.CreateMessage(ctx, report.ReportID, author, ban.RequestMessageBodyMD{BodyMD: "original"})
		require.NoError(t, err)

		edited, err := e.reports.EditMessage(ctx, msg.ReportMessageID, author, ban.RequestMessageBodyMD{BodyMD: "edited"})
		require.NoError(t, err)
		require.Equal(t, "edited", edited.MessageMD)
	})

	t.Run("invalid id is rejected", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)

		author := createPerson(t, steamid.RandSID64())
		_, err := e.reports.EditMessage(t.Context(), 0, author, ban.RequestMessageBodyMD{BodyMD: "edited"})
		require.ErrorIs(t, err, httphelper.ErrParamInvalid)
	})

	t.Run("unknown message is rejected", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)

		author := createPerson(t, steamid.RandSID64())
		_, err := e.reports.EditMessage(t.Context(), 99999999, author, ban.RequestMessageBodyMD{BodyMD: "edited"})
		require.Error(t, err)
		require.ErrorIs(t, err, database.ErrNoResult)
	})

	t.Run("empty body is rejected", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		report := newTestReport(t, author.SteamID, target, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

		msg, err := e.reports.CreateMessage(ctx, report.ReportID, author, ban.RequestMessageBodyMD{BodyMD: "original"})
		require.NoError(t, err)

		_, err = e.reports.EditMessage(ctx, msg.ReportMessageID, author, ban.RequestMessageBodyMD{BodyMD: "   "})
		require.ErrorIs(t, err, httphelper.ErrInvalidParameter)
	})

	t.Run("unchanged body is rejected", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		report := newTestReport(t, author.SteamID, target, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

		msg, err := e.reports.CreateMessage(ctx, report.ReportID, author, ban.RequestMessageBodyMD{BodyMD: "original"})
		require.NoError(t, err)

		_, err = e.reports.EditMessage(ctx, msg.ReportMessageID, author, ban.RequestMessageBodyMD{BodyMD: "original"})
		require.ErrorIs(t, err, database.ErrDuplicate)
	})

	t.Run("other users are denied", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		stranger := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		report := newTestReport(t, author.SteamID, target, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

		msg, err := e.reports.CreateMessage(ctx, report.ReportID, author, ban.RequestMessageBodyMD{BodyMD: "original"})
		require.NoError(t, err)

		_, err = e.reports.EditMessage(ctx, msg.ReportMessageID, stranger, ban.RequestMessageBodyMD{BodyMD: "tampered"})
		require.ErrorIs(t, err, rpc.ErrPermission)
	})

	t.Run("report admin can edit any message", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		admin := steamid.RandSID64()
		createPerson(t, admin)
		e.createAndGrantRole(t, admin, "PERMISSION_REPORT_ADMIN")
		adminPerson := createPerson(t, admin)

		report := newTestReport(t, author.SteamID, target, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

		msg, err := e.reports.CreateMessage(ctx, report.ReportID, author, ban.RequestMessageBodyMD{BodyMD: "original"})
		require.NoError(t, err)

		edited, err := e.reports.EditMessage(ctx, msg.ReportMessageID, adminPerson, ban.RequestMessageBodyMD{BodyMD: "admin edit"})
		require.NoError(t, err)
		require.Equal(t, "admin edit", edited.MessageMD)
	})
}

func TestReports_DropMessage(t *testing.T) {
	t.Parallel()

	t.Run("author can drop their message", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		report := newTestReport(t, author.SteamID, target, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

		msg, err := e.reports.CreateMessage(ctx, report.ReportID, author, ban.RequestMessageBodyMD{BodyMD: "to be dropped"})
		require.NoError(t, err)

		require.NoError(t, e.reports.DropMessage(ctx, author, msg.ReportMessageID))

		messages, err := e.reportRepo.GetReportMessages(ctx, report.ReportID)
		require.NoError(t, err)
		require.Empty(t, messages)
	})

	t.Run("other users are denied", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		stranger := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		report := newTestReport(t, author.SteamID, target, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

		msg, err := e.reports.CreateMessage(ctx, report.ReportID, author, ban.RequestMessageBodyMD{BodyMD: "keep me"})
		require.NoError(t, err)

		err = e.reports.DropMessage(ctx, stranger, msg.ReportMessageID)
		require.ErrorIs(t, err, rpc.ErrPermission)

		messages, err := e.reportRepo.GetReportMessages(ctx, report.ReportID)
		require.NoError(t, err)
		require.Len(t, messages, 1)
	})

	t.Run("report admin can drop any message", func(t *testing.T) {
		t.Parallel()

		e := newEnv(t)
		ctx := t.Context()

		author := createPerson(t, steamid.RandSID64())
		target := steamid.RandSID64()
		createPerson(t, target)

		admin := steamid.RandSID64()
		createPerson(t, admin)
		e.createAndGrantRole(t, admin, "PERMISSION_REPORT_ADMIN")
		adminPerson := createPerson(t, admin)

		report := newTestReport(t, author.SteamID, target, ban.Opened)
		require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

		msg, err := e.reports.CreateMessage(ctx, report.ReportID, author, ban.RequestMessageBodyMD{BodyMD: "to be dropped"})
		require.NoError(t, err)

		require.NoError(t, e.reports.DropMessage(ctx, adminPerson, msg.ReportMessageID))

		messages, err := e.reportRepo.GetReportMessages(ctx, report.ReportID)
		require.NoError(t, err)
		require.Empty(t, messages)
	})
}

func TestReports_MetaStats(t *testing.T) {
	t.Parallel()

	capt := &capturingNotifier{}
	e := newEnv(t, withNotifier(capt))

	require.NoError(t, e.reports.MetaStats(t.Context()))

	require.Eventually(t, func() bool {
		for _, payload := range capt.List() {
			if slices.Contains(payload.Types, notification.Discord) {
				return true
			}
		}

		return false
	}, time.Second*3, time.Millisecond*10)
}

func gotReports(reports []ban.ReportWithAuthor) []ban.Report {
	out := make([]ban.Report, 0, len(reports))
	for _, r := range reports {
		out = append(out, r.Report)
	}

	return out
}
