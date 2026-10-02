package ban_test

import (
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestAppealRepository_SaveMessage(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	ctx := t.Context()

	source, target, author := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()
	createPerson(t, source)
	createPerson(t, target)
	createPerson(t, author)

	saved := e.saveRawBan(t, &ban.Ban{
		SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating,
	})

	setPersonName(t, author, "appealer", "avatar-hash")

	msg := ban.NewBanAppealMessage(saved.BanID, author, "I swear I did not cheat")
	require.NoError(t, e.appealRepo.SaveMessage(ctx, &msg))
	require.Positive(t, msg.BanMessageID)

	fetched, err := e.appealRepo.MessageByID(ctx, msg.BanMessageID)
	require.NoError(t, err)
	require.Equal(t, msg.BanMessageID, fetched.BanMessageID)
	require.Equal(t, saved.BanID, fetched.BanID)
	require.Equal(t, author, fetched.AuthorID)
	require.Equal(t, "I swear I did not cheat", fetched.MessageMD)
	require.False(t, fetched.Deleted)
	require.Equal(t, "appealer", fetched.Personaname)
	require.Equal(t, "avatar-hash", fetched.Avatarhash)
}

//nolint:tparallel // subtests intentionally share one env: a later subtest drops a message the earlier ones read
func TestAppealRepository_Messages(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	ctx := t.Context()

	source, target, author1, author2 := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()
	for _, sid := range []steamid.SteamID{source, target, author1, author2} {
		createPerson(t, sid)
	}

	ban1 := e.saveRawBan(t, &ban.Ban{
		SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating,
	})
	ban2 := e.saveRawBan(t, &ban.Ban{
		SourceID: source, TargetID: author2, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating,
	})

	base := time.Now().Add(-time.Hour)

	msgs := make([]ban.AppealMessage, 3)
	for i, author := range []steamid.SteamID{author1, author2, author1} {
		msgs[i] = ban.NewBanAppealMessage(ban1.BanID, author, "message number "+string(rune('1'+i)))
		msgs[i].CreatedOn = base.Add(time.Duration(i) * time.Second)
		msgs[i].UpdatedOn = base.Add(time.Duration(i) * time.Second)
		require.NoError(t, e.appealRepo.SaveMessage(ctx, &msgs[i]))
	}

	other := ban.NewBanAppealMessage(ban2.BanID, author2, "different ban")
	require.NoError(t, e.appealRepo.SaveMessage(ctx, &other))

	// Sequential on purpose: later subtests mutate the shared messages.
	t.Run("returns only messages for the ban in created order", func(t *testing.T) {
		got, err := e.appealRepo.Messages(ctx, ban1.BanID)
		require.NoError(t, err)
		require.Len(t, got, 3)
		require.Equal(t, msgs[0].BanMessageID, got[0].BanMessageID)
		require.Equal(t, msgs[1].BanMessageID, got[1].BanMessageID)
		require.Equal(t, msgs[2].BanMessageID, got[2].BanMessageID)
	})

	t.Run("other bans are not leaked", func(t *testing.T) {
		got, err := e.appealRepo.Messages(ctx, ban2.BanID)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, other.BanMessageID, got[0].BanMessageID)
	})

	t.Run("deleted messages are excluded", func(t *testing.T) {
		require.NoError(t, e.appealRepo.DropMessage(ctx, &msgs[0]))

		got, err := e.appealRepo.Messages(ctx, ban1.BanID)
		require.NoError(t, err)
		require.Len(t, got, 2)
	})

	t.Run("empty result is non nil", func(t *testing.T) {
		empty := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: steamid.RandSID64(), ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})

		got, err := e.appealRepo.Messages(ctx, empty.BanID)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Empty(t, got)
	})
}

func TestAppealRepository_MessageByID(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	ctx := t.Context()

	_, err := e.appealRepo.MessageByID(ctx, 99999999)
	require.Error(t, err)
	require.ErrorIs(t, err, database.ErrNoResult)
}

//nolint:tparallel // subtests intentionally share one env: the "deleted filter" subtest deletes the ban the earlier ones read
func TestAppealRepository_ByActivity(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	ctx := t.Context()

	source, target, noAppealTarget, author := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()
	for _, sid := range []steamid.SteamID{source, target, noAppealTarget, author} {
		createPerson(t, sid)
	}

	active := e.saveRawBan(t, &ban.Ban{
		SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating,
	})
	inactive := e.saveRawBan(t, &ban.Ban{
		SourceID: source, TargetID: noAppealTarget, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating,
	})

	msg := ban.NewBanAppealMessage(active.BanID, author, "I appeal this ban")
	require.NoError(t, e.appealRepo.SaveMessage(ctx, &msg))

	setPersonName(t, source, "bannerman", "source-avatar")
	setPersonName(t, target, "banned player", "target-avatar")

	// Sequential on purpose: the "deleted filter" subtest mutates the ban.
	t.Run("only bans with appeal activity", func(t *testing.T) {
		got, err := e.appealRepo.ByActivity(ctx, ban.AppealQueryFilter{})
		require.NoError(t, err)

		ids := make([]int32, 0, len(got))
		for _, overview := range got {
			ids = append(ids, overview.BanID)
		}

		require.Contains(t, ids, active.BanID)
		require.NotContains(t, ids, inactive.BanID)
	})

	t.Run("populates person names", func(t *testing.T) {
		got, err := e.appealRepo.ByActivity(ctx, ban.AppealQueryFilter{})
		require.NoError(t, err)

		var found ban.AppealOverview
		for _, overview := range got {
			if overview.BanID == active.BanID {
				found = overview
			}
		}

		require.Equal(t, active.BanID, found.BanID)
		require.Equal(t, target, found.TargetID)
		require.Equal(t, "banned player", found.TargetPersonaname)
		require.Equal(t, "bannerman", found.SourcePersonaname)
	})

	t.Run("deleted filter", func(t *testing.T) {
		require.NoError(t, e.bans.Delete(ctx, &active, false))

		got, err := e.appealRepo.ByActivity(ctx, ban.AppealQueryFilter{})
		require.NoError(t, err)
		for _, overview := range got {
			require.NotEqual(t, active.BanID, overview.BanID)
		}

		withDeleted, err := e.appealRepo.ByActivity(ctx, ban.AppealQueryFilter{Deleted: true})
		require.NoError(t, err)
		var found bool
		for _, overview := range withDeleted {
			if overview.BanID == active.BanID {
				found = true

				break
			}
		}

		require.True(t, found)
	})
}
