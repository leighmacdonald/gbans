package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"fmt"
	"math/big"
	"net/http"
	"strings"
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

const maxResultsDefault = 100

type tableName string

const (
	tableNetLocation tableName = "net_location"
	tableNetProxy    tableName = "net_proxy"
	tableNetASN      tableName = "net_asn"

	tableServer tableName = "server"
	tableDemo   tableName = "demo"
)

// EmptyUUID is used as a placeholder value for signaling the entity is new.
const EmptyUUID = "feb4bf16-7f55-4cb4-923c-4de69a093b79"

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

// applySafeOrder is used to ensure that a user requested column is valid. This
// is used to prevent potential injection attacks as there is no parameterized
// order by value.
func (qf QueryFilter) applySafeOrder(builder sq.SelectBuilder, validColumns map[string][]string, fallback string) sq.SelectBuilder {
	if qf.OrderBy == "" {
		qf.OrderBy = fallback
	}

	qf.OrderBy = strings.ToLower(qf.OrderBy)

	isValid := false

	for prefix, columns := range validColumns {
		for _, name := range columns {
			if name == qf.OrderBy {
				qf.OrderBy = prefix + qf.OrderBy
				isValid = true

				break
			}
		}

		if isValid {
			break
		}
	}

	if qf.Desc {
		builder = builder.OrderBy(fmt.Sprintf("%s DESC", qf.OrderBy))
	} else {
		builder = builder.OrderBy(fmt.Sprintf("%s ASC", qf.OrderBy))
	}

	return builder
}

func (qf QueryFilter) applyLimitOffsetDefault(builder sq.SelectBuilder) sq.SelectBuilder {
	return qf.applyLimitOffset(builder, maxResultsDefault)
}

func (qf QueryFilter) applyLimitOffset(builder sq.SelectBuilder, maxLimit uint64) sq.SelectBuilder {
	if qf.Limit > maxLimit {
		qf.Limit = maxLimit
	}

	if qf.Limit > 0 {
		builder = builder.Limit(qf.Limit)
	}

	if qf.Offset > 0 {
		builder = builder.Offset(qf.Offset)
	}

	return builder
}

func NewTimeStamped() TimeStamped {
	now := time.Now()

	return TimeStamped{
		CreatedOn: now,
		UpdatedOn: now,
	}
}

type TimeStamped struct {
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
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
func (db *Store) QueryBuilder(ctx context.Context, builder sq.SelectBuilder) (pgx.Rows, error) {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return nil, Err(errQuery)
	}

	rows, err := db.conn.Query(ctx, query, args...)

	return rows, Err(err)
}

//nolint:ireturn
func (db *Store) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return db.conn.QueryRow(ctx, query, args...)
}

//nolint:ireturn
func (db *Store) QueryRowBuilder(ctx context.Context, builder sq.SelectBuilder) (pgx.Row, error) {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return nil, Err(errQuery)
	}

	return db.conn.QueryRow(ctx, query, args...), nil
}

func (db *Store) Exec(ctx context.Context, query string, args ...any) error {
	_, err := db.conn.Exec(ctx, query, args...)

	return Err(err)
}

func (db *Store) ExecInsertBuilder(ctx context.Context, builder sq.InsertBuilder) error {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}

	_, err := db.conn.Exec(ctx, query, args...)

	return Err(err)
}

func (db *Store) ExecDeleteBuilder(ctx context.Context, builder sq.DeleteBuilder) error {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}

	_, err := db.conn.Exec(ctx, query, args...)

	return Err(err)
}

func (db *Store) ExecUpdateBuilder(ctx context.Context, builder sq.UpdateBuilder) error {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}

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

func (db *Store) GetCount(ctx context.Context, builder sq.SelectBuilder) (int64, error) {
	countQuery, argsCount, errCountQuery := builder.ToSql()
	if errCountQuery != nil {
		return 0, errors.Wrap(errCountQuery, "Failed to create count query")
	}

	var count int64
	if errCount := db.
		QueryRow(ctx, countQuery, argsCount...).
		Scan(&count); errCount != nil {
		return 0, Err(errCount)
	}

	return count, nil
}

func (db *Store) truncateTable(ctx context.Context, table tableName) error {
	query, args, errQueryArgs := sq.Delete(string(table)).ToSql()
	if errQueryArgs != nil {
		return Err(errQueryArgs)
	}

	rows, errExec := db.Query(ctx, query, args...)
	if errExec != nil {
		return Err(errExec)
	}

	rows.Close()

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

	if errMigration != nil && !errors.Is(errMigration, migrate.ErrNoChange) {
		return errors.Wrapf(errMigration, "Failed to perform migration")
	}

	return nil
}

func SecureRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-"

	ret := make([]byte, n)

	for currentChar := 0; currentChar < n; currentChar++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return ""
		}

		ret[currentChar] = letters[num.Int64()]
	}

	return string(ret)
}
