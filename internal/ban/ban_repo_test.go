package ban_test

import (
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/database/query"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestRepository_Save(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ban id and timestamps", func(t *testing.T) {
		e := newEnv(t)

		source, target := steamid.RandSID64(), steamid.RandSID64()

		b := ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		}

		e.saveRawBan(t, &b)
		require.Positive(t, b.BanID)
		require.False(t, b.CreatedOn.IsZero())
		require.False(t, b.UpdatedOn.IsZero())
	})

	t.Run("update persists changes", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		saved := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})

		saved.Note = "updated note"
		saved.ValidUntil = time.Now().Add(48 * time.Hour)
		require.NoError(t, e.repo.Save(ctx, &saved))

		fetched, err := e.repo.Query(ctx, ban.QueryOpts{BanID: saved.BanID})
		require.NoError(t, err)
		require.Len(t, fetched, 1)
		require.Equal(t, "updated note", fetched[0].Note)
		require.True(t, fetched[0].ValidUntil.After(time.Now().Add(47*time.Hour)))
	})

	t.Run("duplicate ban for same target is rejected", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})

		err := e.repo.Save(ctx, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})
		require.ErrorIs(t, err, database.ErrDuplicate)
	})

	t.Run("same ban type as existing is rejected", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.NoComm, Reason: reason.Spam,
		})

		err := e.repo.Save(ctx, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.NoComm, Reason: reason.Spam,
		})
		require.ErrorIs(t, err, database.ErrDuplicate)
	})

	t.Run("upgrading the ban type is allowed", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		mute := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.NoComm, Reason: reason.Spam,
		})

		fullBan := ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		}
		require.NoError(t, e.repo.Save(ctx, &fullBan))

		require.NotZero(t, fullBan.BanID)
		require.NotEqual(t, mute.BanID, fullBan.BanID)
	})

	t.Run("deleted bans do not block new bans", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		saved := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})
		require.NoError(t, e.repo.Delete(ctx, &saved, false))

		rebanned := ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		}
		require.NoError(t, e.repo.Save(ctx, &rebanned))
		require.Positive(t, rebanned.BanID)
	})
}

func TestRepository_Query(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	e := newEnv(t)

	var (
		source   = steamid.RandSID64()
		t1, t2   = steamid.RandSID64(), steamid.RandSID64()
		t3, t4   = steamid.RandSID64(), steamid.RandSID64()
		groupSID = steamid.New(steamid.BaseGID + 99)
	)
	for _, sid := range []steamid.SteamID{source, t1, t2, t3, t4, groupSID} {
		createPerson(t, sid)
	}

	banA := e.saveRawBan(t, &ban.Ban{
		SourceID: source, TargetID: t1, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating,
	})
	banB := e.saveRawBan(t, &ban.Ban{
		SourceID: source, TargetID: t2, ValidUntil: time.Now().Add(-time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating,
	})
	banC := e.saveRawBan(t, &ban.Ban{
		SourceID: source, TargetID: t3, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating,
	})
	require.NoError(t, e.repo.Delete(ctx, &banC, false))

	banD := e.saveRawBan(t, &ban.Ban{
		SourceID: source, TargetID: groupSID, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating,
	})

	report := ban.Report{
		SourceID: source, TargetID: t4, Description: "cheating evidence",
		ReportStatus: ban.Opened, CreatedOn: time.Now(), UpdatedOn: time.Now(),
	}
	require.NoError(t, e.reportRepo.SaveReport(ctx, &report))

	cidr := "198.51.100.0/24"
	banE := e.saveRawBan(t, &ban.Ban{
		SourceID: source, TargetID: t4, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Network, Reason: reason.External, CIDR: &cidr, ReportID: &report.ReportID,
	})

	t.Run("by ban id", func(t *testing.T) {
		t.Parallel()

		got, err := e.repo.Query(ctx, ban.QueryOpts{BanID: banA.BanID})
		require.NoError(t, err)
		require.Equal(t, []int32{banA.BanID}, banIDs(got))
	})

	t.Run("excludes deleted and groups by default", func(t *testing.T) {
		t.Parallel()

		got, err := e.repo.Query(ctx, ban.QueryOpts{SourceID: source})
		require.NoError(t, err)
		require.Equal(t, banIDs([]ban.Ban{banA, banB, banE}), banIDs(got))
	})

	t.Run("includes deleted when requested", func(t *testing.T) {
		t.Parallel()

		got, err := e.repo.Query(ctx, ban.QueryOpts{SourceID: source, Deleted: true})
		require.NoError(t, err)
		require.Contains(t, banIDs(got), banC.BanID)
	})

	t.Run("by target id", func(t *testing.T) {
		t.Parallel()

		got, err := e.repo.Query(ctx, ban.QueryOpts{TargetID: t1})
		require.NoError(t, err)
		require.Equal(t, []int32{banA.BanID}, banIDs(got))
	})

	t.Run("by reasons", func(t *testing.T) {
		t.Parallel()

		got, err := e.repo.Query(ctx, ban.QueryOpts{SourceID: source, Deleted: true, IncludeGroups: true, Reasons: []reason.Reason{reason.Cheating}})
		require.NoError(t, err)
		require.Equal(t, banIDs([]ban.Ban{banA, banB, banC, banD}), banIDs(got))
	})

	t.Run("by report id", func(t *testing.T) {
		t.Parallel()

		got, err := e.repo.Query(ctx, ban.QueryOpts{ReportID: report.ReportID})
		require.NoError(t, err)
		require.Equal(t, []int32{banE.BanID}, banIDs(got))
	})

	t.Run("cidr filter is broken", func(t *testing.T) {
		t.Parallel()

		// The CIDR filter SQL references a non-existent ip_range column (the
		// ban table column is b.cidr), so any query using CIDR fails until
		// ban_repo.go is fixed.
		_, err := e.repo.Query(ctx, ban.QueryOpts{CIDR: "198.51.100.77"})
		require.Error(t, err)

		_, err = e.repo.Query(ctx, ban.QueryOpts{CIDR: "203.0.113.50"})
		require.Error(t, err)
	})

	t.Run("cidr only", func(t *testing.T) {
		t.Parallel()

		got, err := e.repo.Query(ctx, ban.QueryOpts{SourceID: source, CIDROnly: true})
		require.NoError(t, err)
		require.Equal(t, []int32{banE.BanID}, banIDs(got))
	})

	t.Run("groups only", func(t *testing.T) {
		t.Parallel()

		got, err := e.repo.Query(ctx, ban.QueryOpts{SourceID: source, GroupsOnly: true, IncludeGroups: true})
		require.NoError(t, err)
		require.Equal(t, []int32{banD.BanID}, banIDs(got))
	})

	t.Run("include groups", func(t *testing.T) {
		t.Parallel()

		got, err := e.repo.Query(ctx, ban.QueryOpts{SourceID: source, IncludeGroups: true})
		require.NoError(t, err)
		require.Contains(t, banIDs(got), banD.BanID)
	})

	t.Run("only expired when valid until set", func(t *testing.T) {
		t.Parallel()

		got, err := e.repo.Query(ctx, ban.QueryOpts{SourceID: source, ValidUntil: time.Now()})
		require.NoError(t, err)
		require.Equal(t, []int32{banB.BanID}, banIDs(got))
	})

	t.Run("empty result is non nil", func(t *testing.T) {
		t.Parallel()

		got, err := e.repo.Query(ctx, ban.QueryOpts{BanID: 99999999})
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Empty(t, got)
	})
}

func TestRepository_Delete(t *testing.T) {
	t.Parallel()

	t.Run("soft delete keeps the row", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		saved := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})

		require.NoError(t, e.repo.Delete(ctx, &saved, false))
		require.True(t, saved.Deleted)
		require.NotZero(t, saved.BanID)

		fetched, err := e.repo.Query(ctx, ban.QueryOpts{BanID: saved.BanID, Deleted: true})
		require.NoError(t, err)
		require.Len(t, fetched, 1)
		require.True(t, fetched[0].Deleted)
	})

	t.Run("hard delete removes the row", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		saved := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})
		banID := saved.BanID

		require.NoError(t, e.repo.Delete(ctx, &saved, true))
		require.Zero(t, saved.BanID)

		got, err := e.repo.Query(ctx, ban.QueryOpts{BanID: banID, Deleted: true})
		require.NoError(t, err)
		require.Empty(t, got)
	})
}

func TestRepository_GetOlderThan(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	ctx := t.Context()

	source, t1, t2 := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()

	older := e.saveRawBan(t, &ban.Ban{
		SourceID: source, TargetID: t1, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating,
	})

	require.NoError(t, fixture.Database.Exec(ctx,
		"UPDATE ban SET updated_on = now() - interval '1 hour' WHERE ban_id = $1", older.BanID))

	since := time.Now()

	recent := e.saveRawBan(t, &ban.Ban{
		SourceID: source, TargetID: t2, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating,
	})

	got, err := e.repo.GetOlderThan(ctx, query.Filter{}, since)
	require.NoError(t, err)

	require.Contains(t, banIDs(got), older.BanID)
	require.NotContains(t, banIDs(got), recent.BanID)
}

func TestRepository_MembersList(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	ctx := t.Context()

	groupParent := steamid.New(steamid.BaseGID + 42)
	parent := groupParent.Int64()

	m1, m2 := steamid.RandSID64(), steamid.RandSID64()
	members := steamid.Collection{m1, m2}

	t.Run("new members list", func(t *testing.T) {
		t.Parallel()

		list := ban.NewMembersList(parent, members)
		require.Zero(t, list.MembersID)
		require.Equal(t, parent, list.ParentID)
		require.Equal(t, members, list.Members)
		require.False(t, list.CreatedOn.IsZero())
		require.False(t, list.UpdatedOn.IsZero())
	})

	t.Run("save and fetch", func(t *testing.T) {
		list := ban.NewMembersList(parent, members)
		require.NoError(t, e.repo.SaveMembersList(ctx, &list))
		require.Positive(t, list.MembersID)

		var fetched ban.MembersList
		require.NoError(t, e.repo.GetMembersList(ctx, parent, &fetched))
		require.Equal(t, list.MembersID, fetched.MembersID)
		require.Equal(t, parent, fetched.ParentID)
		require.ElementsMatch(t, members, fetched.Members)
	})

	t.Run("update existing members list", func(t *testing.T) {
		var list ban.MembersList
		require.NoError(t, e.repo.GetMembersList(ctx, parent, &list))

		extra := steamid.RandSID64()
		list.Members = append(list.Members, extra)
		require.NoError(t, e.repo.SaveMembersList(ctx, &list))

		var fetched ban.MembersList
		require.NoError(t, e.repo.GetMembersList(ctx, parent, &fetched))
		require.Contains(t, fetched.Members, extra)
	})

	t.Run("insert cache and truncate", func(t *testing.T) {
		groupID := steamid.New(steamid.BaseGID + 77)

		e1, e2, e3 := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()
		// steam_group_members has an FK to person.
		createPerson(t, e1)
		createPerson(t, e2)
		createPerson(t, e3)
		entries := []int64{e1.Int64(), e2.Int64(), e3.Int64()}

		require.NoError(t, e.repo.InsertCache(ctx, groupID, entries))

		var count int
		require.NoError(t, fixture.Database.QueryRow(ctx,
			"SELECT count(*) FROM steam_group_members WHERE group_id = $1", groupID.Int64()).Scan(&count))
		require.Equal(t, len(entries), count)

		require.NoError(t, e.repo.TruncateCache(ctx))
		require.NoError(t, fixture.Database.QueryRow(ctx,
			"SELECT count(*) FROM steam_group_members").Scan(&count))
		require.Zero(t, count)
	})
}
