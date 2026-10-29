package servers_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"connectrpc.com/authn"
	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/opengraph"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/internal/servers"
	v1 "github.com/leighmacdonald/gbans/internal/servers/v1"
	"github.com/leighmacdonald/gbans/internal/servers/v1/serversv1connect"
	"github.com/leighmacdonald/gbans/pkg/broadcaster"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/stretchr/testify/require"
)

type testPermissionResolver struct {
	perms []rolesv1.Permission
}

func (r testPermissionResolver) PermissionsBySteamID(context.Context, steamid.SteamID) []rolesv1.Permission {
	return r.perms
}

// authRoundTripper injects the bearer token and fingerprint cookie into every
// outgoing request, which covers both unary and streaming calls.
type authRoundTripper struct {
	token       string
	fingerprint string
}

func (h authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+h.token)
	req.Header.Set("Cookie", rpc.FingerprintCookieName+"="+h.fingerprint)

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("round trip: %w", err)
	}

	return resp, nil
}

func TestServersServiceStreamLogs(t *testing.T) {
	serverCase, err := servers.New(servers.NewRepository(fixture.Database),
		broadcaster.New[logparse.EventType, logparse.ServerEvent](), "")
	require.NoError(t, err)

	saved, err := serverCase.Save(t.Context(), servers.NewServer("stream-logs", "stream-logs.example.com", 27015))
	require.NoError(t, err)

	// The fixture database is shared by all tests in this package.
	t.Cleanup(func() {
		require.NoError(t, serverCase.Delete(context.Background(), saved.ServerID))
	})

	const history = 3
	for n := 1; n <= history; n++ {
		serverCase.LogBuffer().Push(saved.ServerID, makeEvent(saved.ServerID, n))
	}

	resolver := testPermissionResolver{perms: []rolesv1.Permission{rolesv1.Permission_PERMISSION_SERVER_READ}}
	authMiddleware := rpc.NewMiddleware("test-site", "test-cookie-secret", resolver)
	roleAuth := rpc.NewRoleAuth(resolver)

	svc := servers.NewServersService(serverCase, &network.Networks{},
		opengraph.NewRepository(opengraph.Options{}), roleAuth, authMiddleware)

	apiMux := http.NewServeMux()
	apiMux.Handle(svc.Pattern, svc.Handler)
	testServer := httptest.NewServer(authn.NewMiddleware(authMiddleware.Authenticate).Wrap(apiMux))
	t.Cleanup(testServer.Close)

	user := person.Core{SteamID: steamid.New(76561198000000001), Name: "Tester"}
	token, fingerprint, err := authMiddleware.MakeUserToken(user)
	require.NoError(t, err)

	client := serversv1connect.NewServersServiceClient(
		&http.Client{Transport: authRoundTripper{token: token, fingerprint: fingerprint}}, testServer.URL)

	t.Run("streams history and live lines", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()

		stream, errStream := client.StreamLogs(ctx, &v1.StreamLogsRequest{ServerId: &saved.ServerID})
		require.NoError(t, errStream)

		for n := 1; n <= history; n++ {
			require.True(t, stream.Receive(), "expected line %d", n)
			msg := stream.Msg()
			require.Contains(t, msg.GetBody(), fmt.Sprintf("line %d", n))
			require.Equal(t, saved.ServerID, msg.GetServerId())
			require.Equal(t, "stream-logs", msg.GetServerName())
			require.Equal(t, v1.LogEventType_LOG_EVENT_TYPE_SAY, msg.GetEventType())
		}

		serverCase.LogBuffer().Push(saved.ServerID, makeEvent(saved.ServerID, 4))
		require.True(t, stream.Receive())
		require.Contains(t, stream.Msg().GetBody(), "line 4")
	})

	t.Run("history limit trims the buffer", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()

		limit := int32(2)
		stream, errStream := client.StreamLogs(ctx, &v1.StreamLogsRequest{
			ServerId:     &saved.ServerID,
			HistoryLimit: &limit,
		})
		require.NoError(t, errStream)

		// The buffer now holds 4 lines, only the last 2 are sent as history.
		for _, wantLine := range []string{"line 3", "line 4"} {
			require.True(t, stream.Receive(), "expected %s", wantLine)
			require.Contains(t, stream.Msg().GetBody(), wantLine)
		}
	})

	t.Run("unknown server is not found", func(t *testing.T) {
		missing := int32(999999)
		stream, errStream := client.StreamLogs(t.Context(), &v1.StreamLogsRequest{ServerId: &missing})
		require.NoError(t, errStream)

		require.False(t, stream.Receive())

		var connectErr *connect.Error
		require.ErrorAs(t, stream.Err(), &connectErr)
		require.Equal(t, connect.CodeNotFound, connectErr.Code())
	})

	t.Run("unauthenticated request is rejected", func(t *testing.T) {
		unauthClient := serversv1connect.NewServersServiceClient(http.DefaultClient, testServer.URL)

		stream, errStream := unauthClient.StreamLogs(t.Context(), &v1.StreamLogsRequest{ServerId: &saved.ServerID})
		require.NoError(t, errStream)

		require.False(t, stream.Receive())
		require.Error(t, stream.Err())
	})
}

// fakeRCONServer speaks the packet dialect of the rcon client library: a
// little-endian size, request id and 4 byte command type followed by the body
// and two null bytes. Logins (type 3) are answered with an auth response
// (type 2), commands (type 2) with a data response (type 0).
type fakeRCONServer struct {
	listener net.Listener
	respond  func(command string) string
}

var errUnexpectedPacket = errors.New("unexpected rcon packet")

func startFakeRCONServer(t *testing.T, respond func(command string) string) string {
	t.Helper()

	var lc net.ListenConfig
	listener, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	fake := &fakeRCONServer{listener, respond}
	t.Cleanup(func() { require.NoError(t, listener.Close()) })

	go func() {
		for {
			conn, errAccept := listener.Accept()
			if errAccept != nil {
				return
			}

			go fake.handle(conn)
		}
	}()

	return listener.Addr().String()
}

func (f *fakeRCONServer) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	reqID, cmdType, body, err := f.readRequest(conn)
	if err != nil || cmdType != 3 || body == "" {
		return
	}
	if err = f.writeResponse(conn, reqID, 2, ""); err != nil {
		return
	}

	for {
		reqID, cmdType, body, err = f.readRequest(conn)
		if err != nil || cmdType != 2 {
			return
		}
		if err = f.writeResponse(conn, reqID, 0, f.respond(body)); err != nil {
			return
		}
	}
}

func (f *fakeRCONServer) readRequest(conn net.Conn) (int32, int32, string, error) {
	var size int32
	if err := binary.Read(conn, binary.LittleEndian, &size); err != nil {
		return 0, 0, "", fmt.Errorf("read packet size: %w", err)
	}
	if size < 10 || size > 4110 {
		return 0, 0, "", fmt.Errorf("%w: size %d", errUnexpectedPacket, size)
	}

	buf := make([]byte, size)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return 0, 0, "", fmt.Errorf("read packet: %w", err)
	}

	var reqID, cmdType int32
	if err := binary.Read(bytes.NewReader(buf[:4]), binary.LittleEndian, &reqID); err != nil {
		return 0, 0, "", fmt.Errorf("read request id: %w", err)
	}
	if err := binary.Read(bytes.NewReader(buf[4:8]), binary.LittleEndian, &cmdType); err != nil {
		return 0, 0, "", fmt.Errorf("read command type: %w", err)
	}

	return reqID, cmdType, strings.TrimRight(string(buf[8:]), "\x00"), nil
}

func (f *fakeRCONServer) writeResponse(conn net.Conn, reqID, cmdType int32, body string) error {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, int32(10+len(body))); err != nil { //nolint:gosec // bounded by the protocol
		return fmt.Errorf("write packet size: %w", err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, reqID); err != nil {
		return fmt.Errorf("write request id: %w", err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, cmdType); err != nil {
		return fmt.Errorf("write command type: %w", err)
	}
	buf.WriteString(body)
	buf.WriteByte(0)
	buf.WriteByte(0)

	if _, err := conn.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("write response: %w", err)
	}

	return nil
}

// newPermissionClient returns a ServersService client authenticated as a user
// holding exactly the given permissions.
func newPermissionClient(t *testing.T, serverCase *servers.Servers, perms []rolesv1.Permission) serversv1connect.ServersServiceClient {
	t.Helper()

	resolver := testPermissionResolver{perms: perms}
	authMiddleware := rpc.NewMiddleware("test-site", "test-cookie-secret", resolver)
	roleAuth := rpc.NewRoleAuth(resolver)

	svc := servers.NewServersService(serverCase, &network.Networks{},
		opengraph.NewRepository(opengraph.Options{}), roleAuth, authMiddleware)

	apiMux := http.NewServeMux()
	apiMux.Handle(svc.Pattern, svc.Handler)
	testServer := httptest.NewServer(authn.NewMiddleware(authMiddleware.Authenticate).Wrap(apiMux))
	t.Cleanup(testServer.Close)

	user := person.Core{SteamID: steamid.New(76561198000000002), Name: "Tester"}
	token, fingerprint, err := authMiddleware.MakeUserToken(user)
	require.NoError(t, err)

	return serversv1connect.NewServersServiceClient(
		&http.Client{Transport: authRoundTripper{token: token, fingerprint: fingerprint}}, testServer.URL)
}

func TestServersServiceExecCommand(t *testing.T) {
	serverCase, err := servers.New(servers.NewRepository(fixture.Database),
		broadcaster.New[logparse.EventType, logparse.ServerEvent](), "")
	require.NoError(t, err)

	fakeAddr := startFakeRCONServer(t, func(command string) string { return "ok: " + command })
	host, portStr, err := net.SplitHostPort(fakeAddr)
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	saved, err := serverCase.Save(t.Context(),
		servers.NewServer("exec-test", host, uint16(port))) //nolint:gosec
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, serverCase.Delete(context.Background(), saved.ServerID))
	})

	client := newPermissionClient(t, serverCase, []rolesv1.Permission{rolesv1.Permission_PERMISSION_SERVER_WRITE})

	t.Run("executes a command and returns the response", func(t *testing.T) {
		command := "sm_say hello"
		resp, errExec := client.ExecCommand(t.Context(), &v1.ExecCommandRequest{
			ServerId: &saved.ServerID,
			Command:  &command,
		})
		require.NoError(t, errExec)
		require.Equal(t, "ok: sm_say hello", resp.GetResponse())
	})

	t.Run("unknown server is not found", func(t *testing.T) {
		missing := int32(999999)
		command := "status"
		_, errExec := client.ExecCommand(t.Context(), &v1.ExecCommandRequest{ServerId: &missing, Command: &command})

		var connectErr *connect.Error
		require.ErrorAs(t, errExec, &connectErr)
		require.Equal(t, connect.CodeNotFound, connectErr.Code())
	})

	t.Run("disabled server is rejected", func(t *testing.T) {
		disabled, errSave := serverCase.Save(t.Context(), servers.NewServer("exec-disabled", "127.0.0.1", 27015))
		require.NoError(t, errSave)
		t.Cleanup(func() {
			require.NoError(t, serverCase.Delete(context.Background(), disabled.ServerID))
		})

		disabled.IsEnabled = false
		_, errSave = serverCase.Save(t.Context(), disabled)
		require.NoError(t, errSave)

		command := "status"
		_, errExec := client.ExecCommand(t.Context(), &v1.ExecCommandRequest{ServerId: &disabled.ServerID, Command: &command})

		var connectErr *connect.Error
		require.ErrorAs(t, errExec, &connectErr)
		require.Equal(t, connect.CodeFailedPrecondition, connectErr.Code())
	})

	t.Run("requires the server write permission", func(t *testing.T) {
		readOnlyClient := newPermissionClient(t, serverCase, []rolesv1.Permission{rolesv1.Permission_PERMISSION_SERVER_READ})

		command := "status"
		_, errExec := readOnlyClient.ExecCommand(t.Context(), &v1.ExecCommandRequest{ServerId: &saved.ServerID, Command: &command})

		var connectErr *connect.Error
		require.ErrorAs(t, errExec, &connectErr)
		require.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
	})
}
