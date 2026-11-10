//nolint:testpackage // exercises the unexported toMessage conversion directly.
package chat

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	v1 "github.com/leighmacdonald/gbans/internal/chat/v1"
	"github.com/leighmacdonald/gbans/internal/chat/v1/chatv1connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToMessageMatchID(t *testing.T) {
	t.Parallel()

	t.Run("nil match id", func(t *testing.T) {
		t.Parallel()

		msg := toMessage(&QueryChatHistoryResult{})

		require.NotNil(t, msg)
		assert.Nil(t, msg.MatchId)
	})

	t.Run("match id", func(t *testing.T) {
		t.Parallel()

		matchID, errID := uuid.NewV4()
		require.NoError(t, errID)

		result := &QueryChatHistoryResult{}
		result.MatchID = &matchID

		msg := toMessage(result)

		require.NotNil(t, msg.MatchId)
		assert.Equal(t, matchID.String(), msg.GetMatchId())
	})
}

//nolint:paralleltest // Uses a short keepalive interval to avoid a slow test.
func TestStreamChatKeepAlive(t *testing.T) {
	buffer := NewBuffer()
	buffer.Push(BufferedChatMessage{ServerID: 1, ServerName: "test", Body: "hello", PersonaName: "player"})

	_, handler := chatv1connect.NewChatServiceHandler(Service{
		chat:              &Chat{buffer: buffer},
		keepAliveInterval: 10 * time.Millisecond,
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := chatv1connect.NewChatServiceClient(server.Client(), server.URL)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	stream, errStream := client.StreamChat(ctx, &v1.StreamChatRequest{HistoryLimit: new(int32(10))})
	require.NoError(t, errStream)

	t.Cleanup(func() {
		_ = stream.Close()
	})

	require.True(t, stream.Receive())
	assert.False(t, stream.Msg().GetKeepalive())
	assert.Equal(t, "hello", stream.Msg().GetBody())

	for range 3 {
		require.True(t, stream.Receive())
		assert.True(t, stream.Msg().GetKeepalive())
	}
}
