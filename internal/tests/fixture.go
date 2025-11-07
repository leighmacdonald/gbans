package tests

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/database"
	personDomain "github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/internal/person"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type Fixture struct {
	container *postgresContainer
	Database  database.Database
	Config    *config.Configuration
	Persons   personDomain.Provider
	TFApi     *thirdparty.TFAPI
	DSN       string
	Close     func()
}

func NewFixture() *Fixture {
	testCtx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	// slog.SetDefault(slog.New(slog.DiscardHandler))
	testDB, errStore := newDB(testCtx)
	if errStore != nil {
		panic(errStore)
	}

	databaseConn := database.New(testDB.dsn, true, false)
	if err := databaseConn.Connect(testCtx); err != nil {
		panic(err)
	}

	api, _ := thirdparty.NewTFAPI("https://tf-api.roto.lol", &http.Client{Timeout: time.Second * 15})

	conf := TestConfig(testDB.dsn)

	return &Fixture{
		container: testDB,
		Database:  databaseConn,
		TFApi:     api,
		Config:    conf,
		DSN:       testDB.dsn,
		Persons:   person.NewPersons(person.NewRepository(conf.Config(), databaseConn), steamid.New(conf.Config().Owner), nil),
		Close: func() {
			termCtx, termCancel := context.WithTimeout(context.Background(), time.Second*30)
			defer termCancel()

			if errTerm := testDB.Terminate(termCtx); errTerm != nil {
				panic(fmt.Sprintf("Failed to terminate test container: %v", errTerm))
			}
		},
	}
}

func (f Fixture) CreateRouter() *gin.Engine {
	router, err := httphelper.CreateRouter(httphelper.RouterOpts{LogLevel: log.Error, Mode: gin.TestMode})
	if err != nil {
		panic(err)
	}

	return router
}

func (f Fixture) Reset(ctx context.Context) {
	const query = `DO
$do$
BEGIN
   EXECUTE
   (SELECT 'TRUNCATE TABLE ' || string_agg(oid::regclass::text, ', ') || ' CASCADE'
    FROM   pg_class
    WHERE  relkind = 'r'
    AND    relnamespace = 'public'::regnamespace
   );
END
$do$;`

	if err := f.Database.Exec(ctx, query); err != nil {
		panic(err)
	}

	if err := f.Database.Migrate(ctx, database.MigrateUp, f.DSN); err != nil {
		panic(err)
	}
}

func (f Fixture) CreateTestPerson(ctx context.Context, steamID steamid.SteamID, perm permission.Privilege) personDomain.Core {
	people := person.NewPersons(person.NewRepository(f.Config.Config(), f.Database), OwnerSID, nil)
	person, errPerson := people.GetOrCreatePersonBySteamID(ctx, steamID)
	if errPerson != nil {
		panic(errPerson)
	}
	full, _ := people.BySteamID(ctx, steamID)
	full.PermissionLevel = perm
	person.PermissionLevel = perm
	_ = people.Save(ctx, &full)

	return person
}

func (f Fixture) CreateTestServer(ctx context.Context) servers.Server {
	serverCase := servers.NewServers(servers.NewRepository(f.Database))
	server, errServer := serverCase.Save(ctx, servers.Server{
		Name:      stringutil.SecureRandomString(10),
		ShortName: stringutil.SecureRandomString(3),
		Address:   stringutil.SecureRandomString(10),
		Password:  stringutil.SecureRandomString(10),
		LogSecret: 12345678,
		Port:      27015,
		RCON:      stringutil.SecureRandomString(10),
		IsEnabled: true,
		Region:    "eu",
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	})
	if errServer != nil {
		panic(errServer)
	}

	return server
}
