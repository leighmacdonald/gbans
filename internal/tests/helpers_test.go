package tests_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/pkg/fs"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var ErrContainer = errors.New("failed to bring up test container")

// postgresContainer is used instead of the postgres.PostgresContainer one since
// we need to build our custom image with extra extensions.
type postgresContainer struct {
	testcontainers.Container
	dbName   string
	user     string
	password string
	dsn      string
}

func newDB(ctx context.Context) (*postgresContainer, error) {
	if dbContainer != nil {
		return dbContainer, nil
	}

	const testInfo = "gbans-test"
	username, password, dbName := testInfo, testInfo, testInfo

	// Naively look for the docker directory. Assumes the project root directory is named "gbans"
	dockerRoot := fs.FindFile("docker", "gbans")

	fromDockerfile := testcontainers.FromDockerfile{
		Dockerfile:    "postgres-ip4r.Dockerfile",
		Context:       dockerRoot,
		PrintBuildLog: false,
	}

	cont, errContainer := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: fromDockerfile,
			HostConfigModifier: func(config *container.HostConfig) {
				config.AutoRemove = false
			},
			Env: map[string]string{
				"POSTGRES_DB":       dbName,
				"POSTGRES_USER":     username,
				"POSTGRES_PASSWORD": password,
			},
			AlwaysPullImage: false,
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

	pgContainer := postgresContainer{
		Container: cont,
		dbName:    dbName,
		user:      username,
		password:  password,
		dsn:       dsn,
	}

	return &pgContainer, nil
}

type permTestValues struct {
	method string
	code   int
	path   string
	levels []permission.Privilege
}

var (
	authed     = []permission.Privilege{permission.Guest}                                        //nolint:gochecknoglobals
	moderators = []permission.Privilege{permission.Guest, permission.User}                       //nolint:gochecknoglobals
	admin      = []permission.Privilege{permission.Guest, permission.User, permission.Moderator} //nolint:gochecknoglobals
)

func testPermissions(t *testing.T, router *gin.Engine, testCases []permTestValues) {
	t.Helper()

	for _, testCase := range testCases {
		for _, level := range testCase.levels {
			var tokens *auth.UserTokens

			switch level {
			case permission.User:
				tokens = loginUser(getUser())
			case permission.Moderator:
				tokens = loginUser(getModerator())
			}

			testEndpoint(t, router, testCase.method, testCase.path, nil, testCase.code, &authTokens{user: tokens})
		}
	}
}
