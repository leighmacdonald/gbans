package store

import (
	"context"
	"embed"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

// EmptyUUID is used as a placeholder value for signaling the entity is new.
const EmptyUUID = "feb4bf16-7f55-4cb4-923c-4de69a093b79"

// Store is the common database interface. All errors from callers should be wrapped in DBErr as they
// are not automatically wrapped.
type Store interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryBuilder(ctx context.Context, builder sq.SelectBuilder) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	QueryRowBuilder(ctx context.Context, builder sq.SelectBuilder) (pgx.Row, error)
	Exec(ctx context.Context, query string, args ...any) error
	ExecInsertBuilder(ctx context.Context, builder sq.InsertBuilder) error
	ExecDeleteBuilder(ctx context.Context, builder sq.DeleteBuilder) error
	ExecUpdateBuilder(ctx context.Context, builder sq.UpdateBuilder) error
	ExecInsertBuilderWithReturnValue(ctx context.Context, builder sq.InsertBuilder, outID any) error
	Builder() sq.StatementBuilderType
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

type Database struct {
	conn *pgxpool.Pool
	log  *zap.Logger
	// Use $ for pg based queries.
	sb          sq.StatementBuilderType
	dsn         string
	autoMigrate bool
	migrated    bool
	logQueries  bool
}

func New(rootLogger *zap.Logger, dsn string, autoMigrate bool, logQueries bool) *Database {
	return &Database{
		sb:          sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
		log:         rootLogger.Named("db"),
		dsn:         dsn,
		autoMigrate: autoMigrate,
		logQueries:  logQueries,
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
func (db *Database) Connect(ctx context.Context) error {
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

func (db *Database) SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults { //nolint:ireturn
	return db.conn.SendBatch(ctx, batch)
}

func (db *Database) Builder() sq.StatementBuilderType {
	return db.sb
}

//nolint:ireturn
func (db *Database) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	rows, err := db.conn.Query(ctx, query, args...)

	return rows, err //nolint:wrapcheck
}

func (db *Database) QueryBuilder(ctx context.Context, builder sq.SelectBuilder) (pgx.Rows, error) { //nolint:ireturn
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return nil, DBErr(errQuery)
	}

	rows, err := db.conn.Query(ctx, query, args...)

	return rows, err //nolint:wrapcheck
}

func (db *Database) QueryRow(ctx context.Context, query string, args ...any) pgx.Row { //nolint:ireturn
	return db.conn.QueryRow(ctx, query, args...)
}

func (db *Database) QueryRowBuilder(ctx context.Context, builder sq.SelectBuilder) (pgx.Row, error) { //nolint:ireturn
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return nil, errQuery //nolint:wrapcheck
	}

	return db.conn.QueryRow(ctx, query, args...), nil
}

func (db *Database) Exec(ctx context.Context, query string, args ...any) error {
	_, err := db.conn.Exec(ctx, query, args...)

	return err //nolint:wrapcheck
}

func (db *Database) ExecInsertBuilder(ctx context.Context, builder sq.InsertBuilder) error {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return DBErr(errQuery)
	}

	_, err := db.conn.Exec(ctx, query, args...)

	return err //nolint:wrapcheck
}

func (db *Database) ExecDeleteBuilder(ctx context.Context, builder sq.DeleteBuilder) error {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return errQuery //nolint:wrapcheck
	}

	_, err := db.conn.Exec(ctx, query, args...)

	return err //nolint:wrapcheck
}

func (db *Database) ExecUpdateBuilder(ctx context.Context, builder sq.UpdateBuilder) error {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return errQuery //nolint:wrapcheck
	}

	_, err := db.conn.Exec(ctx, query, args...)

	return err //nolint:wrapcheck
}

func (db *Database) ExecInsertBuilderWithReturnValue(ctx context.Context, builder sq.InsertBuilder, outID any) error {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return errQuery //nolint:wrapcheck
	}

	if errScan := db.
		QueryRow(ctx, query, args...).
		Scan(outID); errScan != nil {
		return errScan //nolint:wrapcheck
	}

	return nil
}

func (db *Database) Begin(ctx context.Context) (pgx.Tx, error) { //nolint:ireturn
	return db.conn.Begin(ctx) //nolint:wrapcheck
}

// Close will close the underlying database connection if it exists.
func (db *Database) Close() error {
	if db.conn != nil {
		db.conn.Close()
	}

	return nil
}

func getCount(ctx context.Context, database Store, builder sq.SelectBuilder) (int64, error) {
	countQuery, argsCount, errCountQuery := builder.ToSql()
	if errCountQuery != nil {
		return 0, errors.Wrap(errCountQuery, "Failed to create count query")
	}

	var count int64
	if errCount := database.
		QueryRow(ctx, countQuery, argsCount...).
		Scan(&count); errCount != nil {
		return 0, errCount //nolint:wrapcheck
	}

	return count, nil
}

func truncateTable(ctx context.Context, database Store, table string) error {
	query, args, errQueryArgs := sq.Delete(table).ToSql()
	if errQueryArgs != nil {
		return DBErr(errQueryArgs)
	}

	rows, errExec := database.Query(ctx, query, args...)
	if errExec != nil {
		return DBErr(errExec)
	}

	rows.Close()

	return nil
}

// DBErr is used to wrap common database errors in owr own error types.
func DBErr(rootError error) error {
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
