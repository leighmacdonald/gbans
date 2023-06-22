package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net/http"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang-migrate/migrate/v4"
	pgxMigrate "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	// ErrNoResult is returned on successful queries which return no rows.
	ErrNoResult = errors.New("No results found")
	// ErrDuplicate is returned when a duplicate row result is attempted to be inserted.
	ErrDuplicate = errors.New("Duplicate entity")
	// Use $ for pg based queries.
	sb = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	//go:embed migrations
	migrations embed.FS
	conn       *pgxpool.Pool
	logger     *zap.Logger
)

type tableName string

const (
	tableNetLocation tableName = "net_location"
	tableNetProxy    tableName = "net_proxy"
	tableNetASN      tableName = "net_asn"

	tableServer tableName = "server"
	tableDemo   tableName = "demo"
)

// QueryFilter provides a structure for common query parameters.
type QueryFilter struct {
	Offset   uint64 `json:"offset,omitempty" uri:"offset" binding:"gte=0"`
	Limit    uint64 `json:"limit,omitempty" uri:"limit" binding:"gte=0,lte=1000"`
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

func NewQueryFilter(query string) QueryFilter {
	return QueryFilter{
		Limit:    1000,
		Offset:   0,
		SortDesc: true,
		OrderBy:  "created_on",
		Query:    query,
		Deleted:  false,
	}
}

// Init sets up underlying required services.
func Init(ctx context.Context, l *zap.Logger) error {
	logger = l.Named("store")
	cfg, errConfig := pgxpool.ParseConfig(config.DB.DSN)
	if errConfig != nil {
		return errors.Errorf("Unable to parse config: %v", errConfig)
	}
	if config.DB.AutoMigrate {
		if errMigrate := Migrate(MigrateUp); errMigrate != nil {
			if errMigrate.Error() == "no change" {
				logger.Info("Migration at latest version")
			} else {
				return errors.Errorf("Could not migrate schema: %v", errMigrate)
			}
		} else {
			logger.Info("Migration completed successfully")
		}
	}
	dbConn, errConnectConfig := pgxpool.ConnectConfig(ctx, cfg)
	if errConnectConfig != nil {
		return errors.Wrap(errConnectConfig, "Failed to connect to database")
	}
	conn = dbConn
	return nil
}

func Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	rows, err := conn.Query(ctx, query, args...)
	return rows, Err(err)
}

func QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return conn.QueryRow(ctx, query, args...)
}

func Exec(ctx context.Context, query string, args ...any) error {
	_, err := conn.Exec(ctx, query, args...)
	return Err(err)
}

// Close will close the underlying database connection if it exists.
func Close() error {
	if conn != nil {
		conn.Close()
	}
	return nil
}

func truncateTable(ctx context.Context, table tableName) error {
	if _, errExec := conn.Exec(ctx, fmt.Sprintf("TRUNCATE %s;", table)); errExec != nil {
		return Err(errExec)
	}
	return nil
}

// Err is used to wrap common database errors in owr own error types.
func Err(rootError error) error {
	if rootError == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(rootError, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return ErrDuplicate
		default:
			return rootError
		}
	}
	if rootError.Error() == "no rows in result set" {
		return ErrNoResult
	}
	return rootError
}

// MigrationAction is the type of migration to perform.
type MigrationAction int

const (
	// MigrateUp Fully upgrades the schema.
	MigrateUp = iota
	// MigrateDn Fully downgrades the schema.
	MigrateDn
	// MigrateUpOne Upgrade the schema by one revision.
	MigrateUpOne
	// MigrateDownOne Downgrade the schema by one revision.
	MigrateDownOne
)

// Migrate database schema.
func Migrate(action MigrationAction) error {
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
	defer util.LogCloser(driver, logger)
	source, errHTTPFS := httpfs.New(http.FS(migrations), "migrations")
	if errHTTPFS != nil {
		return errHTTPFS
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
