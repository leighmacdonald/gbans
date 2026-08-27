package ban_test

import (
	"slices"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/ban/bantype"
	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

func TestAppealState_String(t *testing.T) {
	tests := []struct {
		name  string
		state ban.AppealState
		want  string
	}{
		{name: "open", state: ban.Open, want: "Open"},
		{name: "denied", state: ban.Denied, want: "Denied"},
		{name: "accepted", state: ban.Accepted, want: "Accepted"},
		{name: "reduced", state: ban.Reduced, want: "Reduced"},
		{name: "no appeal", state: ban.NoAppeal, want: "No Appeal"},
		{name: "any state", state: ban.AnyState, want: "Open"},
		{name: "unknown", state: ban.AppealState(99), want: "Open"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.state.String())
		})
	}
}

func TestAppealMessage_Path(t *testing.T) {
	require.Equal(t, "/ban/42#msg-7", ban.AppealMessage{BanID: 42, BanMessageID: 7}.Path())
}

func TestAppeals_CreateBanMessage(t *testing.T) {
	t.Parallel()

	t.Run("target can reply on an open appeal", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		author := createPerson(t, source)
		createPerson(t, target)

		created := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})

		targetPerson := createPerson(t, target)

		msg, err := e.appeals.CreateBanMessage(ctx, targetPerson, created.BanID, "I did not cheat, I promise")
		require.NoError(t, err)
		require.Positive(t, msg.BanMessageID)
		require.Equal(t, created.BanID, msg.BanID)
		require.Equal(t, target, msg.AuthorID)

		messages, err := e.appealRepo.Messages(ctx, created.BanID)
		require.NoError(t, err)
		require.Len(t, messages, 1)

		_ = author
	})

	t.Run("invalid ban id is rejected", func(t *testing.T) {
		e := newEnv(t)

		user := createPerson(t, steamid.RandSID64())
		_, err := e.appeals.CreateBanMessage(t.Context(), user, 0, "hello")
		require.ErrorIs(t, err, httphelper.ErrInvalidParameter)
	})

	t.Run("unknown ban is rejected", func(t *testing.T) {
		e := newEnv(t)

		user := createPerson(t, steamid.RandSID64())
		_, err := e.appeals.CreateBanMessage(t.Context(), user, 99999999, "hello")
		require.ErrorIs(t, err, httphelper.ErrInvalidParameter)
	})

	t.Run("empty message is rejected", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		targetPerson := createPerson(t, target)

		created := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})

		_, err := e.appeals.CreateBanMessage(ctx, targetPerson, created.BanID, "")
		require.ErrorIs(t, err, httphelper.ErrInvalidParameter)
	})

	t.Run("closed appeal denies non admin", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target, other := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		createPerson(t, target)
		otherPerson := createPerson(t, other)

		created := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})
		created.AppealState = ban.Denied
		require.NoError(t, e.bans.Save(ctx, &created))

		// "other" holds only the baseline user role, no appeal admin.
		_, err := e.appeals.CreateBanMessage(ctx, otherPerson, created.BanID, "let me explain")
		require.ErrorIs(t, err, rpc.ErrPermission)
	})

	t.Run("appeal admin can reply on a closed appeal", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target, admin := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		createPerson(t, target)
		adminPerson := createPerson(t, admin)
		// CreateBanMessage requires BAN_CREATE or being the target before the
		// APPEAL_ADMIN check on closed appeals.
		e.createAndGrantRole(t, admin, "PERMISSION_APPEAL_ADMIN", "PERMISSION_BAN_CREATE")

		created := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})
		created.AppealState = ban.Denied
		require.NoError(t, e.bans.Save(ctx, &created))

		msg, err := e.appeals.CreateBanMessage(ctx, adminPerson, created.BanID, "reviewing this ban")
		require.NoError(t, err)
		require.Positive(t, msg.BanMessageID)
	})

	t.Run("sends notifications to group and target", func(t *testing.T) {
		capt := &capturingNotifier{}
		e := newEnv(t, withNotifier(capt))
		ctx := t.Context()

		source, target, admin := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		createPerson(t, target)
		adminPerson := createPerson(t, admin)
		// CreateBanMessage requires BAN_CREATE or being the target.
		e.createAndGrantRole(t, admin, "PERMISSION_APPEAL_ADMIN", "PERMISSION_BAN_CREATE")

		created := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})

		_, err := e.appeals.CreateBanMessage(ctx, adminPerson, created.BanID, "reviewing this ban")
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			return len(capt.List()) >= 3
		}, time.Second*3, time.Millisecond*10)

		var discordSent, groupSent, userSent bool
		for _, payload := range capt.List() {
			switch {
			case slices.Contains(payload.Types, notification.Discord) && payload.MessageSend != nil:
				discordSent = true
			case slices.Contains(payload.Groups, rolesv1.Permission_PERMISSION_BAN_READ) && payload.Message == "A new ban appeal message":
				groupSent = true
			case slices.Contains(payload.Sids, target) && payload.Message == "A new ban appeal message":
				userSent = true
			}
		}

		require.True(t, discordSent, "expected discord notification, got %+v", capt.List())
		require.True(t, groupSent, "expected group notification, got %+v", capt.List())
		require.True(t, userSent, "expected target user notification, got %+v", capt.List())
	})
}

func TestAppeals_EditBanMessage(t *testing.T) {
	t.Parallel()

	t.Run("author can edit their message", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		targetPerson := createPerson(t, target)

		created := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})

		msg := ban.NewBanAppealMessage(created.BanID, target, "original text")
		require.NoError(t, e.appealRepo.SaveMessage(ctx, &msg))

		edited, err := e.appeals.EditBanMessage(ctx, targetPerson, msg.BanMessageID, "updated text")
		require.NoError(t, err)
		require.Equal(t, "updated text", edited.MessageMD)

		fetched, err := e.appealRepo.MessageByID(ctx, msg.BanMessageID)
		require.NoError(t, err)
		require.Equal(t, "updated text", fetched.MessageMD)
	})

	t.Run("unknown message is rejected", func(t *testing.T) {
		e := newEnv(t)

		user := createPerson(t, steamid.RandSID64())
		_, err := e.appeals.EditBanMessage(t.Context(), user, 99999999, "text")
		require.Error(t, err)
		require.ErrorIs(t, err, database.ErrNoResult)
	})

	t.Run("empty message is rejected", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		targetPerson := createPerson(t, target)

		created := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})

		msg := ban.NewBanAppealMessage(created.BanID, target, "original text")
		require.NoError(t, e.appealRepo.SaveMessage(ctx, &msg))

		_, err := e.appeals.EditBanMessage(ctx, targetPerson, msg.BanMessageID, "")
		require.ErrorIs(t, err, httphelper.ErrInvalidParameter)
	})

	t.Run("unchanged message is rejected", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		targetPerson := createPerson(t, target)

		created := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})

		msg := ban.NewBanAppealMessage(created.BanID, target, "original text")
		require.NoError(t, e.appealRepo.SaveMessage(ctx, &msg))

		_, err := e.appeals.EditBanMessage(ctx, targetPerson, msg.BanMessageID, "original text")
		require.ErrorIs(t, err, database.ErrDuplicate)
	})

	t.Run("non author without appeal admin is rejected", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target, other := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		createPerson(t, target)
		otherPerson := createPerson(t, other)

		created := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})

		msg := ban.NewBanAppealMessage(created.BanID, target, "original text")
		require.NoError(t, e.appealRepo.SaveMessage(ctx, &msg))

		_, err := e.appeals.EditBanMessage(ctx, otherPerson, msg.BanMessageID, "tweaked")
		require.ErrorIs(t, err, rpc.ErrPermission)
	})

	t.Run("appeal admin can edit any message", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target, admin := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		createPerson(t, target)
		adminPerson := createPerson(t, admin)
		e.createAndGrantRole(t, admin, "PERMISSION_APPEAL_ADMIN")

		created := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})

		msg := ban.NewBanAppealMessage(created.BanID, target, "original text")
		require.NoError(t, e.appealRepo.SaveMessage(ctx, &msg))

		edited, err := e.appeals.EditBanMessage(ctx, adminPerson, msg.BanMessageID, "admin edited")
		require.NoError(t, err)
		require.Equal(t, "admin edited", edited.MessageMD)
	})
}

func TestAppeals_Messages(t *testing.T) {
	t.Parallel()

	e := newEnv(t)
	ctx := t.Context()

	source, target, reader := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()
	createPerson(t, source)
	targetPerson := createPerson(t, target)
	readerPerson := createPerson(t, reader)
	e.grantRole(t, reader, "moderator")

	created := e.saveRawBan(t, &ban.Ban{
		SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
		BanType: bantype.Banned, Reason: reason.Cheating,
	})

	msg := ban.NewBanAppealMessage(created.BanID, target, "I appeal")
	require.NoError(t, e.appealRepo.SaveMessage(ctx, &msg))

	t.Run("target can read", func(t *testing.T) {
		t.Parallel()

		got, err := e.appeals.Messages(ctx, targetPerson, created.BanID)
		require.NoError(t, err)
		require.Len(t, got, 1)
	})

	t.Run("appeal reader can read", func(t *testing.T) {
		t.Parallel()

		got, err := e.appeals.Messages(ctx, readerPerson, created.BanID)
		require.NoError(t, err)
		require.Len(t, got, 1)
	})

	t.Run("other users are denied", func(t *testing.T) {
		t.Parallel()

		stranger := createPerson(t, steamid.RandSID64())
		_, err := e.appeals.Messages(ctx, stranger, created.BanID)
		require.ErrorIs(t, err, rpc.ErrPermission)
	})
}

func TestAppeals_DropMessage(t *testing.T) {
	t.Parallel()

	t.Run("author can drop their message", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target := steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		targetPerson := createPerson(t, target)

		created := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})

		msg := ban.NewBanAppealMessage(created.BanID, target, "to be removed")
		require.NoError(t, e.appealRepo.SaveMessage(ctx, &msg))

		require.NoError(t, e.appeals.DropMessage(ctx, targetPerson, msg.BanMessageID))

		got, err := e.appealRepo.Messages(ctx, created.BanID)
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("non author is denied", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target, other := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		createPerson(t, target)
		otherPerson := createPerson(t, other)

		created := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})

		msg := ban.NewBanAppealMessage(created.BanID, target, "keep me")
		require.NoError(t, e.appealRepo.SaveMessage(ctx, &msg))

		err := e.appeals.DropMessage(ctx, otherPerson, msg.BanMessageID)
		require.ErrorIs(t, err, rpc.ErrPermission)
	})

	t.Run("appeal admin can drop any message", func(t *testing.T) {
		e := newEnv(t)
		ctx := t.Context()

		source, target, admin := steamid.RandSID64(), steamid.RandSID64(), steamid.RandSID64()
		createPerson(t, source)
		createPerson(t, target)
		adminPerson := createPerson(t, admin)
		e.createAndGrantRole(t, admin, "PERMISSION_APPEAL_ADMIN")

		created := e.saveRawBan(t, &ban.Ban{
			SourceID: source, TargetID: target, ValidUntil: time.Now().Add(time.Hour),
			BanType: bantype.Banned, Reason: reason.Cheating,
		})

		msg := ban.NewBanAppealMessage(created.BanID, target, "to be removed")
		require.NoError(t, e.appealRepo.SaveMessage(ctx, &msg))

		require.NoError(t, e.appeals.DropMessage(ctx, adminPerson, msg.BanMessageID))

		got, err := e.appealRepo.Messages(ctx, created.BanID)
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("unknown message is rejected", func(t *testing.T) {
		e := newEnv(t)

		user := createPerson(t, steamid.RandSID64())
		err := e.appeals.DropMessage(t.Context(), user, 99999999)
		require.Error(t, err)
		require.ErrorIs(t, err, database.ErrNoResult)
	})
}
