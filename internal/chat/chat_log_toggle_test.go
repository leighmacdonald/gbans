//nolint:testpackage // exercises the unexported chat message handler directly.
package chat

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/leighmacdonald/gbans/internal/ban/reason"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

type countingNotifier struct {
	mu    sync.Mutex
	count int
}

func (n *countingNotifier) Send(notification.Payload) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.count++
}

func (n *countingNotifier) sent() int {
	n.mu.Lock()
	defer n.mu.Unlock()

	return n.count
}

type fakePersons struct{}

func (fakePersons) GetOrCreatePersonBySteamID(_ context.Context, sid steamid.SteamID) (person.Core, error) {
	return person.Core{SteamID: sid, Name: "Tester", Avatarhash: "avatar"}, nil
}

func (fakePersons) EnsurePerson(context.Context, steamid.SteamID) error { return nil }

func TestChatLogEnabledToggle(t *testing.T) {
	ctx := t.Context()
	notifier := &countingNotifier{}
	cfg := &Config{Enabled: false} // word filtering is independent of chat log forwarding

	discordCfg := &discord.Config{ChatLogChannelID: "chat-log-channel"}
	discordCfg.SetChatLogEnabled(true)
	chatCase := New(NewRepository(nil), cfg, WordFilters{}, fakePersons{}, notifier, nil,
		discordCfg, steamid.New(76561198000000001))

	player := logparse.SourcePlayer{SID: steamid.New(76561198000000002), Name: "Tester"}
	results := logparse.Results{
		EventType: logparse.Say,
		Event:     logparse.SayEvt{SourcePlayer: player, Msg: "hello world"},
	}
	evt := logparse.ServerEvent{
		ServerID:   1,
		ServerName: "test",
		Results:    results,
	}
	sendMessage := func() {
		require.NoError(t, chatCase.handleMessage(ctx, evt, player, "hello world", time.Now(), false, reason.Language))
	}

	sendMessage()
	require.Equal(t, 1, notifier.sent())

	discordCfg.SetChatLogEnabled(false)
	sendMessage()
	require.Equal(t, 1, notifier.sent(), "disabled toggle must not send to Discord")

	discordCfg.SetChatLogEnabled(true)
	sendMessage()
	require.Equal(t, 2, notifier.sent(), "enabled toggle must send to Discord")

	require.Equal(t, "chat-log-channel", chatCase.logChannelID, "channel comes from the config instance")
}
