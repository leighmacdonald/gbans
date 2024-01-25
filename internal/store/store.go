package store

import (
	"context"
	"embed"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leighmacdonald/gbans/internal/errs"
	"go.uber.org/zap"
)

// // ErrNoResult is returned on successful queries which return no rows.
// ErrNoResult = errors.New("No results found")
// // ErrDuplicate is returned when a duplicate row result is attempted to be inserted.
// ErrDuplicate = errors.New("Duplicate entity")

//go:embed migrations
var migrations embed.FS

var (
	ErrRowResults  = errors.New("resulting rows contain error")
	ErrTxStart     = errors.New("could not start transaction")
	ErrTxCommit    = errors.New("failed to commit tx changes")
	ErrTxRollback  = errors.New("could not rollback transaction")
	ErrPoolFailed  = errors.New("could not create store pool")
	ErrUUIDGen     = errors.New("could not generate uuid")
	ErrCreateQuery = errors.New("failed to generate query")
	ErrCountQuery  = errors.New("failed to get count result")
	ErrTooShort    = errors.New("value too short")
)

// Database is the common database interface. All errors from callers should be wrapped in errs.DBErr as they
// are not automatically wrapped.
type Database interface {
	Connect(ctx context.Context) error
	Close() error
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
	GetCount(ctx context.Context, store Database, builder sq.SelectBuilder) (int64, error)
}

type Stores struct {
	Database
}

type postgresStore struct {
	conn *pgxpool.Pool
	log  *zap.Logger
	// Use $ for pg based queries.
	sb          sq.StatementBuilderType
	dsn         string
	autoMigrate bool
	migrated    bool
	logQueries  bool
}

func New(rootLogger *zap.Logger, dsn string, autoMigrate bool, logQueries bool) Stores {
	return Stores{Database: &postgresStore{
		sb:          sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
		log:         rootLogger.Named("db"),
		dsn:         dsn,
		autoMigrate: autoMigrate,
		logQueries:  logQueries,
	}}
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
func (db *postgresStore) Connect(ctx context.Context) error {
	cfg, errConfig := pgxpool.ParseConfig(db.dsn)
	if errConfig != nil {
		return fmt.Errorf("unable to parse db config/dsn: %w", errConfig)
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
			return fmt.Errorf("could not migrate schema: %w", errMigrate)
		}

		db.log.Info("Migration completed successfully")
	}

	dbConn, errConnectConfig := pgxpool.NewWithConfig(ctx, cfg)
	if errConnectConfig != nil {
		return errors.Join(errConnectConfig, ErrPoolFailed)
	}

	db.conn = dbConn

	return nil
}

func (db *postgresStore) SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults { //nolint:ireturn
	return db.conn.SendBatch(ctx, batch)
}

func (db *postgresStore) Builder() sq.StatementBuilderType {
	return db.sb
}

//nolint:ireturn
func (db *postgresStore) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	rows, err := db.conn.Query(ctx, query, args...)

	return rows, err //nolint:wrapcheck
}

func (db *postgresStore) QueryBuilder(ctx context.Context, builder sq.SelectBuilder) (pgx.Rows, error) { //nolint:ireturn
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return nil, errs.DBErr(errQuery)
	}

	rows, err := db.conn.Query(ctx, query, args...)

	return rows, err //nolint:wrapcheck
}

func (db *postgresStore) QueryRow(ctx context.Context, query string, args ...any) pgx.Row { //nolint:ireturn
	return db.conn.QueryRow(ctx, query, args...)
}

func (db *postgresStore) QueryRowBuilder(ctx context.Context, builder sq.SelectBuilder) (pgx.Row, error) { //nolint:ireturn
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return nil, errQuery //nolint:wrapcheck
	}

	return db.conn.QueryRow(ctx, query, args...), nil
}

func (db *postgresStore) Exec(ctx context.Context, query string, args ...any) error {
	_, err := db.conn.Exec(ctx, query, args...)

	return err //nolint:wrapcheck
}

func (db *postgresStore) ExecInsertBuilder(ctx context.Context, builder sq.InsertBuilder) error {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return errs.DBErr(errQuery)
	}

	_, err := db.conn.Exec(ctx, query, args...)

	return err //nolint:wrapcheck
}

func (db *postgresStore) ExecDeleteBuilder(ctx context.Context, builder sq.DeleteBuilder) error {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return errQuery //nolint:wrapcheck
	}

	_, err := db.conn.Exec(ctx, query, args...)

	return err //nolint:wrapcheck
}

func (db *postgresStore) ExecUpdateBuilder(ctx context.Context, builder sq.UpdateBuilder) error {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return errQuery //nolint:wrapcheck
	}

	_, err := db.conn.Exec(ctx, query, args...)

	return err //nolint:wrapcheck
}

func (db *postgresStore) ExecInsertBuilderWithReturnValue(ctx context.Context, builder sq.InsertBuilder, outID any) error {
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

func (db *postgresStore) Begin(ctx context.Context) (pgx.Tx, error) { //nolint:ireturn
	return db.conn.Begin(ctx) //nolint:wrapcheck
}

// Close will close the underlying database connection if it exists.
func (db *postgresStore) Close() error {
	if db.conn != nil {
		db.conn.Close()
	}

	return nil
}

func (db *postgresStore) GetCount(ctx context.Context, store Database, builder sq.SelectBuilder) (int64, error) {
	countQuery, argsCount, errCountQuery := builder.ToSql()
	if errCountQuery != nil {
		return 0, errors.Join(errCountQuery, ErrCreateQuery)
	}

	var count int64
	if errCount := store.
		QueryRow(ctx, countQuery, argsCount...).
		Scan(&count); errCount != nil {
		return 0, errCount //nolint:wrapcheck
	}

	return count, nil
}

func getCount(ctx context.Context, store Database, builder sq.SelectBuilder) (int64, error) {
	countQuery, argsCount, errCountQuery := builder.ToSql()
	if errCountQuery != nil {
		return 0, errors.Join(errCountQuery, ErrCreateQuery)
	}

	var count int64
	if errCount := store.
		QueryRow(ctx, countQuery, argsCount...).
		Scan(&count); errCount != nil {
		return 0, errCount //nolint:wrapcheck
	}

	return count, nil
}

func truncateTable(ctx context.Context, database Database, table string) error {
	query, args, errQueryArgs := sq.Delete(table).ToSql()
	if errQueryArgs != nil {
		return errs.DBErr(errQueryArgs)
	}

	rows, errExec := database.Query(ctx, query, args...)
	if errExec != nil {
		return errs.DBErr(errExec)
	}

	rows.Close()

	return nil
}
