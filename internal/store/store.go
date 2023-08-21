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
	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leighmacdonald/gbans/pkg/fp"
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	// ErrNoResult is returned on successful queries which return no rows.
	ErrNoResult = errors.New("No results found")
	// ErrDuplicate is returned when a duplicate row result is attempted to be inserted.
	ErrDuplicate = errors.New("Duplicate entity")

	//go:embed migrations
	migrations embed.FS
)

type tableName string

const (
	tableNetLocation tableName = "net_location"
	tableNetProxy    tableName = "net_proxy"
	tableNetASN      tableName = "net_asn"

	tableServer tableName = "server"
	tableDemo   tableName = "demo"
)

type Store struct {
	conn *pgxpool.Pool
	log  *zap.Logger
	// Use $ for pg based queries.
	sb          sq.StatementBuilderType
	dsn         string
	autoMigrate bool
	migrated    bool
	logQueries  bool
	weaponMap   fp.MutexMap[logparse.Weapon, int]
}

func New(rootLogger *zap.Logger, dsn string, autoMigrate bool, logQueries bool) *Store {
	return &Store{
		sb:          sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
		log:         rootLogger.Named("db"),
		dsn:         dsn,
		autoMigrate: autoMigrate,
		logQueries:  logQueries,
		weaponMap:   fp.NewMutexMap[logparse.Weapon, int](),
	}
}

// QueryFilter provides a structure for common query parameters.
type QueryFilter struct {
	Offset  uint64 `json:"offset,omitempty" uri:"offset" binding:"gte=0"`
	Limit   uint64 `json:"limit,omitempty" uri:"limit" binding:"gte=0,lte=1000"`
	Desc    bool   `json:"desc,omitempty" uri:"desc"`
	Query   string `json:"query,omitempty" uri:"query"`
	OrderBy string `json:"order_by,omitempty" uri:"order_by"`
	Deleted bool   `json:"deleted,omitempty" uri:"deleted"`
}

func (queryFilter *QueryFilter) orderString() string {
	dir := "DESC"
	if !queryFilter.Desc {
		dir = "ASC"
	}

	return fmt.Sprintf("%s %s", queryFilter.OrderBy, dir)
}

const maxQuerySize = 1000

func NewQueryFilter(query string) QueryFilter {
	return QueryFilter{
		Limit:   maxQuerySize,
		Offset:  0,
		Desc:    true,
		OrderBy: "created_on",
		Query:   query,
		Deleted: false,
	}
}

type dbQueryTracer struct {
	log *zap.SugaredLogger
}

func (tracer *dbQueryTracer) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	tracer.log.Infow("Executing command", "sql", data.SQL, "args", data.Args)

	return ctx
}

func (tracer *dbQueryTracer) TraceQueryEnd(_ context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
}

// Connect sets up underlying required services.
func (db *Store) Connect(ctx context.Context) error {
	cfg, errConfig := pgxpool.ParseConfig(db.dsn)
	if errConfig != nil {
		return errors.Errorf("Unable to parse config: %v", errConfig)
	}

	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxuuid.Register(conn.TypeMap())

		return nil
	}

	if db.logQueries {
		cfg.ConnConfig.Tracer = &dbQueryTracer{log: db.log.Sugar()}
	}

	if db.autoMigrate && !db.migrated {
		if errMigrate := db.migrate(MigrateUp, db.dsn); errMigrate != nil {
			return errors.Errorf("Could not migrate schema: %v", errMigrate)
		}

		db.log.Info("Migration completed successfully")
	}

	dbConn, errConnectConfig := pgxpool.NewWithConfig(ctx, cfg)
	if errConnectConfig != nil {
		return errors.Wrap(errConnectConfig, "Failed to connect to database")
	}

	db.conn = dbConn

	return nil
}

//nolint:ireturn
func (db *Store) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	rows, err := db.conn.Query(ctx, query, args...)

	return rows, Err(err)
}

//nolint:ireturn
func (db *Store) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return db.conn.QueryRow(ctx, query, args...)
}

func (db *Store) Exec(ctx context.Context, query string, args ...any) error {
	_, err := db.conn.Exec(ctx, query, args...)

	return Err(err)
}

// Close will close the underlying database connection if it exists.
func (db *Store) Close() error {
	if db.conn != nil {
		db.conn.Close()
	}

	return nil
}

func (db *Store) truncateTable(ctx context.Context, table tableName) error {
	query, args, errQueryArgs := sq.Delete(string(table)).ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}

	if _, errExec := db.Query(ctx, query, args...); errExec != nil {
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

// migrate database schema.
func (db *Store) migrate(action MigrationAction, dsn string) error {
	defer func() {
		db.migrated = true
	}()

	instance, errOpen := sql.Open("pgx", dsn)
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

	defer util.LogCloser(driver, db.log)

	source, errHTTPFS := httpfs.New(http.FS(migrations), "migrations")
	if errHTTPFS != nil {
		return errors.Wrapf(errHTTPFS, "Failed to create migration httpfs")
	}

	migrator, errMigrateInstance := migrate.NewWithInstance("iofs", source, "pgx", driver)
	if errMigrateInstance != nil {
		return errors.Wrapf(errMigrateInstance, "Failed to migrator up")
	}

	var errMigration error

	switch action {
	case MigrateUpOne:
		errMigration = migrator.Steps(1)
	case MigrateDn:
		errMigration = migrator.Down()
	case MigrateDownOne:
		errMigration = migrator.Steps(-1)
	case MigrateUp:
		fallthrough
	default:
		errMigration = migrator.Up()
	}

	if errMigration != nil {
		if errMigration.Error() != "no change" {
			return errors.Wrapf(errMigration, "Failed to perform migration")
		}
	}

	return nil
}
