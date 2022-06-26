package store

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	cache "github.com/Code-Hex/go-generics-cache"
	sq "github.com/Masterminds/squirrel"
	"github.com/golang-migrate/migrate/v4"
	pgxMigrate "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/logrusadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"time"
)

var (
	// ErrNoResult is returned on successful queries which return no rows
	ErrNoResult = errors.New("No results found")
	// ErrDuplicate is returned when a duplicate row result is attempted to be inserted
	ErrDuplicate = errors.New("Duplicate entity")
	// Use $ for pg based queries
	sb = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	//go:embed migrations
	migrations embed.FS
)

type tableName string

const (
	tableNetLocation tableName = "net_location"
	tableNetProxy    tableName = "net_proxy"
	tableNetASN      tableName = "net_asn"
	//tablePersonIP    tableName = "person_ip"
	tableServer tableName = "server"
	tableDemo   tableName = "demo"
)

// QueryFilter provides a structure for common query parameters
type QueryFilter struct {
	Offset   uint64 `json:"offset,omitempty" uri:"offset" binding:"gte=0"`
	Limit    int    `json:"limit,omitempty" uri:"limit" binding:"gte=0,lte=1000"`
	SortDesc bool   `json:"desc,omitempty" uri:"desc"`
	Query    string `json:"query,omitempty" uri:"query"`
	OrderBy  string `json:"order_by,omitempty" uri:"order_by"`
	Deleted  bool   `json:"deleted,omitempty" uri:"deleted"`
}

func (queryFilter *QueryFilter) orderString() string {
	dir := "DESC"
	if !queryFilter.SortDesc {
		dir = "ASC"
	}
	return fmt.Sprintf("%s %s", queryFilter.OrderBy, dir)
}

func NewQueryFilter(query string) *QueryFilter {
	return &QueryFilter{
		Limit:    1000,
		Offset:   0,
		SortDesc: true,
		OrderBy:  "created_on",
		Query:    query,
	}
}

// New sets up underlying required services.
func New(ctx context.Context, dsn string) (Store, error) {
	cfg, errConfig := pgxpool.ParseConfig(dsn)
	if errConfig != nil {
		return nil, errors.Errorf("Unable to parse config: %v", errConfig)
	}
	newDatabase := pgStore{}
	if config.DB.AutoMigrate {
		if errMigrate := newDatabase.Migrate(MigrateUp); errMigrate != nil {
			if errMigrate.Error() == "no change" {
				log.Debugf("Migration at latest version")
			} else {
				return nil, errors.Errorf("Could not migrate schema: %v", errMigrate)
			}
		} else {
			log.Infof("Migration completed successfully")
		}
	}
	if config.DB.LogQueries {
		logger := log.New()
		logLevel, errLevel := log.ParseLevel(config.Log.Level)
		if errLevel != nil {
			log.Fatalf("Invalid log level: %s (%v)", config.Log.Level, errLevel)
		}
		logger.SetLevel(logLevel)
		logger.SetFormatter(&log.TextFormatter{
			ForceColors:   config.Log.ForceColours,
			DisableColors: config.Log.DisableColours,
			FullTimestamp: config.Log.FullTimestamp,
		})
		logger.SetReportCaller(config.Log.ReportCaller)
		cfg.ConnConfig.Logger = logrusadapter.NewLogger(logger)
	}
	dbConn, errConnectConfig := pgxpool.ConnectConfig(ctx, cfg)
	if errConnectConfig != nil {
		log.Fatalf("Failed to connect to database: %v", errConnectConfig)
	}
	return &pgStore{
		conn:        dbConn,
		playerCache: cache.New[steamid.SID64, model.Person](),
		serverCache: cache.New[string, model.Server](),
	}, nil
}

// pgStore implements Store against a postgresql database
type pgStore struct {
	conn        *pgxpool.Pool
	playerCache *cache.Cache[steamid.SID64, model.Person]
	serverCache *cache.Cache[string, model.Server]
}

func (database *pgStore) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	rows, err := database.conn.Query(ctx, query, args...)
	return rows, Err(err)
}

func (database *pgStore) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return database.conn.QueryRow(ctx, query, args...)
}

func (database *pgStore) Exec(ctx context.Context, query string, args ...any) error {
	_, err := database.conn.Exec(ctx, query, args...)
	return Err(err)
}

// Close will close the underlying database connection if it exists
func (database *pgStore) Close() error {
	if database.conn != nil {
		database.conn.Close()
	}
	return nil
}

func (database *pgStore) truncateTable(ctx context.Context, table tableName) error {
	if _, errExec := database.conn.Exec(ctx, fmt.Sprintf("TRUNCATE %s;", table)); errExec != nil {
		return Err(errExec)
	}
	return nil
}

// Err is used to wrap common database errors in owr own error types
func Err(rootError error) error {
	if rootError == nil {
		return rootError
	}
	var pgErr *pgconn.PgError
	if errors.As(rootError, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return ErrDuplicate
		default:
			log.Errorf("Unhandled store error: (%s) %s", pgErr.Code, pgErr.Message)
			return rootError
		}
	}
	if rootError.Error() == "no rows in result set" {
		return ErrNoResult
	}
	return rootError
}

// MigrationAction is the type of migration to perform
type MigrationAction int

const (
	// MigrateUp Fully upgrades the schema
	MigrateUp = iota
	// MigrateDn Fully downgrades the schema
	MigrateDn
	// MigrateUpOne Upgrade the schema by one revision
	MigrateUpOne
	// MigrateDownOne Downgrade the schema by one revision
	MigrateDownOne
)

// Migrate database schema
func (database *pgStore) Migrate(action MigrationAction) error {
	instance, errOpen := sql.Open("pgx", config.DB.DSN)
	if errOpen != nil {
		return errors.Wrapf(errOpen, "Failed to open database for migration")
	}
	if errPing := instance.Ping(); errPing != nil {
		return errors.Wrapf(errPing, "Cannot migrate, failed to connect to target server")
	}
	driver, errMigrate := pgxMigrate.WithInstance(instance, &pgxMigrate.Config{
		MigrationsTable:       "_migration",
		SchemaName:            "public",
		StatementTimeout:      60 * time.Second,
		MultiStatementEnabled: true,
	})
	if errMigrate != nil {
		return errors.Wrapf(errMigrate, "failed to create migration driver")
	}
	defer func() {
		if errClose := driver.Close(); errClose != nil {
			log.Errorf("Failed to close migrator driver: %v", errClose)
		}
	}()
	source, errHttpFS := httpfs.New(http.FS(migrations), "migrations")
	if errHttpFS != nil {
		return errHttpFS
	}
	migrator, errMigrateInstance := migrate.NewWithInstance("iofs", source, "pgx", driver)
	if errMigrateInstance != nil {
		return errors.Wrapf(errMigrateInstance, "Failed to migrator up")
	}
	switch action {
	case MigrateUpOne:
		return migrator.Steps(1)
	case MigrateDn:
		return migrator.Down()
	case MigrateDownOne:
		return migrator.Steps(-1)
	case MigrateUp:
		fallthrough
	default:
		return migrator.Up()
	}
}

// Import will import bans from a root folder.
// The formatting is JSON with the importedBan schema defined inline
// Valid filenames are: main_ban.json
func (database *pgStore) Import(ctx context.Context, root string) error {
	type importedBan struct {
		BanID      int    `json:"ban_id"`
		SteamID    uint64 `json:"steam_id"`
		AuthorID   uint64 `json:"author_id"`
		BanType    int    `json:"ban_type"`
		Reason     int    `json:"reason"`
		ReasonText string `json:"reason_text"`
		Note       string `json:"note"`
		Until      int    `json:"until"`
		CreatedOn  int    `json:"created_on"`
		UpdatedOn  int    `json:"updated_on"`
		BanSource  int    `json:"ban_source"`
	}

	return filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
		switch d.Name() {
		case "main_ban.json":
			body, errRead := ioutil.ReadFile(path.Join(root, d.Name()))
			if errRead != nil {
				return errRead
			}
			var importedBans []importedBan
			if errUnmarshal := json.Unmarshal(body, &importedBans); errUnmarshal != nil {
				return errUnmarshal
			}
			for _, imported := range importedBans {
				banTarget := model.NewPerson(steamid.SID64(imported.SteamID))
				author := model.NewPerson(steamid.SID64(imported.AuthorID))
				if errGetPersonA := database.GetOrCreatePersonBySteamID(ctx, steamid.SID64(imported.SteamID), &banTarget); errGetPersonA != nil {
					return errGetPersonA
				}
				if errGetPersonB := database.GetOrCreatePersonBySteamID(ctx, steamid.SID64(imported.AuthorID), &author); errGetPersonB != nil {
					return errGetPersonB
				}
				sum, errPlayerSummary := steamweb.PlayerSummaries(steamid.Collection{banTarget.SteamID, author.SteamID})
				if errPlayerSummary != nil {
					log.Errorf("Failed to get player summary: %v", errPlayerSummary)
					return errPlayerSummary
				}
				if len(sum) > 0 {
					banTarget.PlayerSummary = &sum[0]
					if errSavePerson := database.SavePerson(ctx, &banTarget); errSavePerson != nil {
						return errSavePerson
					}
					if author.SteamID.Valid() && len(sum) > 1 {
						author.PlayerSummary = &sum[1]
						if errSavePerson := database.SavePerson(ctx, &author); errSavePerson != nil {
							return errSavePerson
						}
					}
				}
				newBan := model.NewBan(banTarget.SteamID, author.SteamID, 0)
				newBan.ValidUntil = time.Unix(int64(imported.Until), 0)
				newBan.ReasonText = imported.ReasonText
				newBan.CreatedOn = time.Unix(int64(imported.CreatedOn), 0)
				newBan.UpdatedOn = time.Unix(int64(imported.UpdatedOn), 0)
				newBan.Source = model.System
				if errSaveBan := database.SaveBan(ctx, &newBan); errSaveBan != nil {
					return errSaveBan
				}
			}
		}
		return nil
	})
}
