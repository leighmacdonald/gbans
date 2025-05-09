package tests_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
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
		PrintBuildLog: true,
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

type MockConfigRepository struct {
	config domain.Config
}

func newConfigRepo(config domain.Config) domain.ConfigRepository {
	return &MockConfigRepository{
		config: config,
	}
}

func (m *MockConfigRepository) Read(_ context.Context) (domain.Config, error) {
	return m.config, nil
}

func (m *MockConfigRepository) Write(_ context.Context, config domain.Config) error {
	m.config = config

	return nil
}

func (m *MockConfigRepository) Init(_ context.Context) error {
	return nil
}

type permTestValues struct {
	method string
	code   int
	path   string
	levels []domain.Privilege
}

var (
	authed     = []domain.Privilege{domain.PGuest}                                  //nolint:gochecknoglobals
	moderators = []domain.Privilege{domain.PGuest, domain.PUser}                    //nolint:gochecknoglobals
	admin      = []domain.Privilege{domain.PGuest, domain.PUser, domain.PModerator} //nolint:gochecknoglobals
)

func testPermissions(t *testing.T, router *gin.Engine, testCases []permTestValues) {
	t.Helper()

	for _, testCase := range testCases {
		for _, level := range testCase.levels {
			var tokens *domain.UserTokens

			switch level {
			case domain.PUser:
				tokens = loginUser(getUser())
			case domain.PModerator:
				tokens = loginUser(getModerator())
			}

			testEndpoint(t, router, testCase.method, testCase.path, nil, testCase.code, &authTokens{user: tokens})
		}
	}
}
