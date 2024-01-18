//go:build integration

package app

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

type mockAsset struct {
	name        string
	body        []byte
	size        int64
	contentType string
}

type MockAssetStore struct {
	buckets map[string][]mockAsset
}

func (s *MockAssetStore) Remove(_ context.Context, _ string, _ string) error {
	return nil
}

func (s *MockAssetStore) Put(_ context.Context, bucket string, name string, body io.Reader, size int64, contentType string) error {
	_, ok := s.buckets[bucket]
	if !ok {
		s.buckets[bucket] = []mockAsset{}
	}
	data, _ := io.ReadAll(body)
	s.buckets[bucket] = append(s.buckets[bucket], mockAsset{
		name:        name,
		body:        data,
		size:        size,
		contentType: contentType,
	})

	return nil
}

func newTestDB(ctx context.Context) (string, *postgres.PostgresContainer, error) {
	const testInfo = "gbans-test"
	username, password, dbName := testInfo, testInfo, testInfo
	cont, errContainer := postgres.RunContainer(
		ctx,
		testcontainers.WithImage("docker.io/postgis/postgis:15-3.3"),
		postgres.WithDatabase(dbName),
		postgres.WithUsername(username),
		postgres.WithPassword(password),
		testcontainers.WithWaitStrategy(wait.
			ForLog("database system is ready to accept connections").
			WithOccurrence(2)),
	)

	if errContainer != nil {
		return "", nil, errors.Wrap(errContainer, "Failed to bring up test container")
	}

	port, _ := cont.MappedPort(ctx, "5432")
	dsn := fmt.Sprintf("postgresql://%s:%s@localhost:%s/%s", username, password, port.Port(), dbName)

	return dsn, cont, nil
}

func TestApp(t *testing.T) {
	ctx := context.Background()

	setDefaultConfigValues()

	var config Config

	require.NoError(t, ReadConfig(&config, true))

	config.General.Mode = TestMode
	config.General.Owner = "76561198084134025"
	config.Discord.Enabled = false

	dsn, databaseContainer, errDB := newTestDB(ctx)
	if errDB != nil {
		t.Skipf("Failed to bring up testcontainer db: %v", errDB)
	}

	database := store.New(zap.NewNop(), dsn, true, false)
	if dbErr := database.Connect(ctx); dbErr != nil {
		t.Fatalf("Failed to setup db: %v", dbErr)
	}

	t.Cleanup(func() {
		if errTerm := databaseContainer.Terminate(ctx); errTerm != nil {
			t.Error("Failed to terminate test container")
		}
	})

	app := New(&config, database, nil, zap.NewNop(), &MockAssetStore{
		map[string][]mockAsset{},
	})

	require.NoError(t, firstTimeSetup(ctx, &config, database))

	t.Run("api_server", testServerAPI(&app))
	t.Run("api_frontend", testFrontendAPI(&app))
	t.Run("match_sum", testMatchSum(&app))
}

func newTestReq(method string, route string, body any, token string) *http.Request {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, route, bytes.NewReader(b))
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	return req
}

func decodeTestResponse(r *http.Response, target any) error {
	defer func() {
		_ = r.Body.Close()
	}()
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(&target)
}

func testServerAPI(app *App) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		httpServer := newHTTPServer(ctx, app)
		testServer := store.NewServer("test-1", "127.0.0.1", 27015)
		testServer.Name = "Test Instance"
		require.NoError(t, app.db.SaveServer(ctx, &testServer))
		token, errToken := newServerToken(testServer.ServerID, app.config().HTTP.CookieKey)
		require.NoError(t, errToken)
		testBaddie := "76561197961279983"

		t.Run("auth_valid", func(t *testing.T) {
			t.Parallel()
			req := newTestReq("POST", "/api/server/auth", ServerAuthReq{
				Key: testServer.Password,
			}, "")
			w := httptest.NewRecorder()
			httpServer.Handler.ServeHTTP(w, req)
			var resp ServerAuthResp
			require.NoError(t, decodeTestResponse(w.Result(), &resp))
			require.True(t, resp.Status)
			require.True(t, len(resp.Token) > 8)
			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("auth_invalid", func(t *testing.T) {
			t.Parallel()
			req := newTestReq(http.MethodPost, "/api/server/auth", ServerAuthReq{
				Key: "xxxxx",
			}, "xxxxxxxxxxxxxxx")
			w := httptest.NewRecorder()
			httpServer.Handler.ServeHTTP(w, req)
			var resp ServerAuthResp
			require.NoError(t, decodeTestResponse(w.Result(), &resp))
			require.False(t, resp.Status)
			require.Equal(t, "", resp.Token)
			require.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("check_no_token", func(t *testing.T) {
			t.Parallel()
			req := newTestReq(http.MethodPost, "/api/check", CheckRequest{
				ClientID: 10,
				SteamID:  "76561197961279983",
				IP:       net.ParseIP("10.10.10.10"),
			}, "")
			w := httptest.NewRecorder()
			httpServer.Handler.ServeHTTP(w, req)
			require.Equal(t, http.StatusUnauthorized, w.Code)
		})

		t.Run("sm_admins", func(t *testing.T) {
			t.Parallel()
			req := newTestReq(http.MethodGet, "/api/server/admins", gin.H{}, token)
			w := httptest.NewRecorder()
			httpServer.Handler.ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code)
			var perms []store.ServerPermission
			require.NoError(t, decodeTestResponse(w.Result(), &perms))
			require.True(t, len(perms) >= 1)
			ownerFound := false
			for _, perm := range perms {
				if steamid.SIDToSID64(perm.SteamID) == app.config().General.Owner {
					ownerFound = true
					break
				}
			}
			require.True(t, ownerFound, "Failed to find owner sid")
		})

		t.Run("sm_ban", func(t *testing.T) {
			t.Parallel()
			req := newTestReq(http.MethodPost, "/api/sm/bans/steam/create", apiBanRequest{
				SourceID:       store.StringSID(app.config().General.Owner.String()),
				TargetID:       store.StringSID(testBaddie),
				Duration:       "custom",
				ValidUntil:     time.Now().Add(time.Hour * 11),
				BanType:        store.Banned,
				Reason:         store.Custom,
				ReasonText:     "Custom reason value",
				Note:           "A moderator note",
				ReportID:       0,
				DemoName:       "",
				DemoTick:       0,
				IncludeFriends: false,
			}, token)
			w := httptest.NewRecorder()
			httpServer.Handler.ServeHTTP(w, req)
			require.Equal(t, http.StatusCreated, w.Code)
			var banSteam store.BanSteam
			require.NoError(t, decodeTestResponse(w.Result(), &banSteam))
			require.True(t, banSteam.BanID > 0)
		})

		t.Run("sm_report", func(t *testing.T) {
			t.Parallel()
			req := newTestReq(http.MethodPost, "/api/sm/report/create", apiCreateReportReq{
				SourceID:        store.StringSID(app.config().General.Owner.String()),
				TargetID:        store.StringSID(testBaddie),
				Description:     "User report message",
				Reason:          store.Custom,
				ReasonText:      "Custom reason value",
				DemoName:        "",
				DemoTick:        0,
				PersonMessageID: 0,
			}, token)
			w := httptest.NewRecorder()
			httpServer.Handler.ServeHTTP(w, req)
			require.Equal(t, http.StatusCreated, w.Code)
			var report store.Report
			require.NoError(t, decodeTestResponse(w.Result(), &report))
			require.True(t, report.ReportID > 0)
		})

		t.Run("sm_ping", func(t *testing.T) {
			t.Parallel()
			req := newTestReq(http.MethodPost, "/api/ping_mod", pingReq{
				ServerName: testServer.ShortName,
				Name:       "Uncle Lame",
				SteamID:    steamid.SID64(testBaddie),
				Reason:     "cheating blah",
				Client:     11,
			}, token)
			w := httptest.NewRecorder()
			httpServer.Handler.ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("state_push", func(t *testing.T) {
			t.Parallel()
			app.state.stateMu.Lock()
			app.state.serverState[testServer.ServerID] = serverDetails{
				ServerID:  testServer.ServerID,
				NameShort: testServer.ShortName,
				Name:      testServer.Name,
			}
			app.state.stateMu.Unlock()
			req := newTestReq(http.MethodPost, "/api/state_update", partialStateUpdate{
				Hostname:       testServer.Name,
				ShortName:      testServer.ShortName,
				CurrentMap:     "pl_goodburger",
				PlayersReal:    25,
				PlayersTotal:   26,
				PlayersVisible: 24,
			}, token)
			w := httptest.NewRecorder()
			httpServer.Handler.ServeHTTP(w, req)
			require.Equal(t, http.StatusNoContent, w.Code)
		})

		t.Run("demo", func(t *testing.T) {
			t.Parallel()

			// Copied from srcdsup
			compressReaderBytes := func(log *zap.Logger, demoName string, demoBytes []byte, jsonBytes []byte) ([]byte, error) {
				var compressedDemo bytes.Buffer

				demoBufWriter := bufio.NewWriter(&compressedDemo)
				writer := zip.NewWriter(demoBufWriter)

				outFile, errWriter := writer.Create(demoName)
				if errWriter != nil {
					return nil, errors.Wrap(errWriter, "Failed to write body to xz")
				}

				if _, errWrite := outFile.Write(demoBytes); errWrite != nil {
					return nil, errors.Wrap(errWrite, "Failed to close writer")
				}

				jsonFile, errJSON := writer.Create("stats.json")
				if errJSON != nil {
					return nil, errors.Wrap(errJSON, "Failed to write json to zip")
				}

				if _, errWrite := jsonFile.Write(jsonBytes); errWrite != nil {
					return nil, errors.Wrap(errWrite, "Failed to close writer")
				}

				if errClose := writer.Close(); errClose != nil {
					return nil, errors.Wrap(errClose, "Failed to close writer")
				}

				log.Debug("Compressed size", zap.Int("size", compressedDemo.Len()))

				return compressedDemo.Bytes(), nil
			}
			demoFileName := "test-file.dem"
			stats, errStats := json.Marshal(gin.H{"76561198084134025": gin.H{"score": 0, "deaths": 0, "score_total": 0}})
			require.NoError(t, errStats)
			testDemoData := []byte(store.SecureRandomString(10000))
			compressedBody, errCompress := compressReaderBytes(zap.NewNop(), demoFileName, testDemoData, stats)
			require.NoError(t, errCompress)
			var (
				outBuffer       = new(bytes.Buffer)
				multiPartWriter = multipart.NewWriter(outBuffer)
			)

			h := make(textproto.MIMEHeader)
			h.Set("Content-Type", "application/octet-stream")
			h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="demo"; filename="%s"`, demoFileName))

			fileWriter, errCreatePart := multiPartWriter.CreatePart(h)
			require.NoError(t, errCreatePart)
			_, errWrite := fileWriter.Write(compressedBody)
			require.NoError(t, errWrite)
			require.NoError(t, multiPartWriter.WriteField("server_name", testServer.ShortName))
			require.NoError(t, multiPartWriter.WriteField("map_name", "pl_goodburger"))
			require.NoError(t, multiPartWriter.Close())

			req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, "/api/demo", outBuffer)
			require.NoError(t, errReq)

			req.Header.Set("Authorization", token)
			req.Header.Set("Content-Type", multiPartWriter.FormDataContentType())

			w := httptest.NewRecorder()
			httpServer.Handler.ServeHTTP(w, req)
			require.Equal(t, http.StatusCreated, w.Code)
		})
	}
}

func testMatchSum(_ *App) func(t *testing.T) {
	return func(t *testing.T) {
	}
}

func testFrontendAPI(_ *App) func(t *testing.T) {
	return func(t *testing.T) {
	}
}
