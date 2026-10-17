package discord_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/authn"
	"connectrpc.com/connect"
	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
	discordv1 "github.com/leighmacdonald/gbans/internal/discord/v1"
	"github.com/leighmacdonald/gbans/internal/discord/v1/discordv1connect"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

type fakeRoleResolver struct{}

func (fakeRoleResolver) PermissionsBySteamID(_ context.Context, _ steamid.SteamID) []rolesv1.Permission {
	return nil
}

type recordingConnection struct {
	config   *discord.Config
	messages []*discordgo.MessageSend
	channels []string
}

func (r *recordingConnection) Send(channelID string, message *discordgo.MessageSend) error {
	r.channels = append(r.channels, channelID)
	r.messages = append(r.messages, message)

	return nil
}
func (r *recordingConnection) Start() error { return nil }
func (r *recordingConnection) Close()       {}
func (r *recordingConnection) Config() *discord.Config {
	return r.config
}

func (r *recordingConnection) MustRegisterCommandHandler(_ *discordgo.ApplicationCommand, _ discord.Handler) {
}
func (r *recordingConnection) MustRegisterPrefixHandler(_ string, _ discord.Handler) {}
func (r *recordingConnection) CreateRole(_ string) (string, error)                   { return "", nil }

func (r *recordingConnection) Roles() ([]*discordgo.Role, error) { return nil, nil }

const (
	testSiteName = "gbans-test"
	testCookie   = "test-cookie-secret"
)

func newDiscordHarness(t *testing.T) (*recordingConnection, *httptest.Server) {
	t.Helper()

	connection := &recordingConnection{config: &discord.Config{LogChannelID: "log-channel"}}
	resolver := fakeRoleResolver{}
	middleware := rpc.NewMiddleware(testSiteName, testCookie, resolver)
	service := discord.NewService(connection, rpc.NewRoleAuth(resolver), middleware)

	api := http.NewServeMux()
	api.Handle(service.Pattern, service.Handler)

	server := httptest.NewServer(authn.NewMiddleware(middleware.Authenticate).Wrap(api))
	t.Cleanup(server.Close)

	return connection, server
}

func serverTokenClient(server *httptest.Server) (discordv1connect.DiscordServiceClient, error) {
	token, errToken := rpc.NewServerTokenGenerator(testSiteName, []byte(testCookie))(7, "Pl-UpToDate")
	if errToken != nil {
		return nil, errToken
	}

	interceptor := connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+token)

			return next(ctx, req)
		}
	}))

	return discordv1connect.NewDiscordServiceClient(server.Client(), server.URL, interceptor), nil
}

func TestSendMessageCrash(t *testing.T) {
	connection, server := newDiscordHarness(t)

	client, errClient := serverTokenClient(server)
	require.NoError(t, errClient)

	_, err := client.SendMessageCrash(t.Context(), &discordv1.SendMessageCrashRequest{CrashId: new("19B4094E")})
	require.NoError(t, err)

	require.Len(t, connection.messages, 1)
	require.Equal(t, "log-channel", connection.channels[0])

	message := connection.messages[0]
	require.Len(t, message.Components, 3)

	heading, ok := message.Components[0].(discordgo.TextDisplay)
	require.True(t, ok)
	require.Contains(t, heading.Content, "Pl-UpToDate")

	link, ok := message.Components[1].(discordgo.Button)
	require.True(t, ok)
	require.Equal(t, "https://crash.limetech.org/?id=19B4094E", link.URL)
}

func TestSendMessageCrashUnauthenticated(t *testing.T) {
	connection, server := newDiscordHarness(t)

	client := discordv1connect.NewDiscordServiceClient(server.Client(), server.URL)

	_, err := client.SendMessageCrash(t.Context(), &discordv1.SendMessageCrashRequest{CrashId: new("19B4094E")})
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Empty(t, connection.messages)
}
