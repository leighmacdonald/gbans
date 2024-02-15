package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgerrcode"
	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leighmacdonald/gbans/internal/domain"
)

// // ErrNoResult is returned on successful queries which return no rows.
// ErrNoResult = errors.New("No results found")
// // ErrDuplicate is returned when a duplicate row result is attempted to be inserted.
// ErrDuplicate = errors.New("Duplicate entity")

//go:embed migrations
var migrations embed.FS

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
	GetCount(ctx context.Context, builder sq.SelectBuilder) (int64, error)
	DBErr(rootError error) error
	TruncateTable(ctx context.Context, table string) error
}

type postgresStore struct {
	conn *pgxpool.Pool
	log  *slog.Logger
	// Use $ for pg based queries.
	sb          sq.StatementBuilderType
	dsn         string
	autoMigrate bool
	migrated    bool
	logQueries  bool
}

func New(dsn string, autoMigrate bool, logQueries bool) Database {
	return &postgresStore{
		sb:          sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
		log:         slog.Default().WithGroup("db"),
		dsn:         dsn,
		autoMigrate: autoMigrate,
		logQueries:  logQueries,
	}
}

type dbQueryTracer struct {
	log *slog.Logger
}

func (tracer *dbQueryTracer) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	tracer.log.Info("Executing command", slog.String("sql", data.SQL), slog.Any("args", data.Args))

	return ctx
}

func (tracer *dbQueryTracer) TraceQueryEnd(_ context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
}

// DBErr is used to wrap common database errors in owr own error types.
func (db *postgresStore) DBErr(rootError error) error {
	if rootError == nil {
		return nil
	}

	if errors.Is(rootError, pgx.ErrNoRows) {
		return domain.ErrNoResult
	}

	var pgErr *pgconn.PgError

	if errors.As(rootError, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return domain.ErrDuplicate
		default:
			return rootError
		}
	}

	return rootError
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
		cfg.ConnConfig.Tracer = &dbQueryTracer{log: db.log}
	}

	if db.autoMigrate && !db.migrated {
		if errMigrate := db.migrate(MigrateUp, db.dsn); errMigrate != nil {
			return fmt.Errorf("could not migrate schema: %w", errMigrate)
		}

		db.log.Info("Migration completed successfully")
	}

	dbConn, errConnectConfig := pgxpool.NewWithConfig(ctx, cfg)
	if errConnectConfig != nil {
		return errors.Join(errConnectConfig, domain.ErrPoolFailed)
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
		return nil, db.DBErr(errQuery)
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
		return db.DBErr(errQuery)
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

func (db *postgresStore) GetCount(ctx context.Context, builder sq.SelectBuilder) (int64, error) {
	countQuery, argsCount, errCountQuery := builder.ToSql()
	if errCountQuery != nil {
		return 0, errors.Join(errCountQuery, domain.ErrCreateQuery)
	}

	var count int64
	if errCount := db.
		QueryRow(ctx, countQuery, argsCount...).
		Scan(&count); errCount != nil {
		return 0, errCount //nolint:wrapcheck
	}

	return count, nil
}

func (db *postgresStore) TruncateTable(ctx context.Context, table string) error {
	query, args, errQueryArgs := sq.Delete(table).ToSql()
	if errQueryArgs != nil {
		return db.DBErr(errQueryArgs)
	}

	rows, errExec := db.Query(ctx, query, args...)
	if errExec != nil {
		return db.DBErr(errExec)
	}

	rows.Close()

	return nil
}
