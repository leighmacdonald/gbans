package test

import (
	"context"
	"errors"
	"fmt"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	ErrContainer = errors.New("failed to bring up test container")

	container *PostgresContainer //nolint:gochecknoglobals
)

func NewDB(ctx context.Context) (*PostgresContainer, error) {
	if container != nil {
		return container, nil
	}

	const testInfo = "gbans-test"
	username, password, dbName := testInfo, testInfo, testInfo

	fromDockerfile := testcontainers.FromDockerfile{
		Dockerfile:    "postgres-ip4r.Dockerfile",
		Context:       "docker",
		PrintBuildLog: true,
	}

	cont, errContainer := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: fromDockerfile,
			Env: map[string]string{
				"POSTGRES_DB":       dbName,
				"POSTGRES_USER":     username,
				"POSTGRES_PASSWORD": password,
			},
			WaitingFor: wait.
				ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		},
		Started: true,
	})

	if errContainer != nil {
		return nil, errors.Join(errContainer, ErrContainer)
	}

	port, _ := cont.MappedPort(ctx, "5432")
	dsn := fmt.Sprintf("postgresql://%s:%s@localhost:%s/%s", username, password, port.Port(), dbName)

	pgContainer := PostgresContainer{
		Container: cont,
		dbName:    dbName,
		user:      username,
		password:  password,
		dsn:       dsn,
	}

	container = &pgContainer

	return container, nil
}

// PostgresContainer is used instead of the postgres.PostgresContainer one since
// we need to build our custom image with extra extensions.
type PostgresContainer struct {
	testcontainers.Container
	dbName   string
	user     string
	password string
	dsn      string
}
