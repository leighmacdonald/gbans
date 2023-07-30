package app // nolint:testpackage

import (
	"context"
	"fmt"
	"testing"

	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

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

	dsn, databaseContainer, errDB := newTestDB(ctx)
	if errDB != nil {
		t.Skipf("Failed to bring up testcontainer db: %v", errDB)
	}

	database := store.New(zap.NewNop(), dsn, true, false)
	if dbErr := database.Connect(ctx); dbErr != nil {
		t.Fatalf("Failed to setup store: %v", dbErr)
	}

	t.Cleanup(func() {
		if errTerm := databaseContainer.Terminate(ctx); errTerm != nil {
			t.Error("Failed to terminate test container")
		}
	})

	app := New(&config, database, nil, zap.NewNop())

	t.Run("match_sum", testMatchSum(&app))
}

func testMatchSum(_ *App) func(t *testing.T) {
	return func(t *testing.T) {
	}
}
