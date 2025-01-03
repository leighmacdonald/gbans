package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"

	sq "github.com/Masterminds/squirrel"
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
	Pool() *pgxpool.Pool
	Connect(ctx context.Context) error
	Close() error
	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Query(ctx context.Context, transaction pgx.Tx, query string, args ...any) (pgx.Rows, error)
	QueryBuilder(ctx context.Context, transaction pgx.Tx, builder sq.SelectBuilder) (pgx.Rows, error)
	QueryRow(ctx context.Context, transaction pgx.Tx, query string, args ...any) pgx.Row
	QueryRowBuilder(ctx context.Context, transaction pgx.Tx, builder sq.SelectBuilder) (pgx.Row, error)
	Exec(ctx context.Context, transaction pgx.Tx, query string, args ...any) error
	ExecInsertBuilder(ctx context.Context, transaction pgx.Tx, builder sq.InsertBuilder) error
	ExecDeleteBuilder(ctx context.Context, transaction pgx.Tx, builder sq.DeleteBuilder) error
	ExecUpdateBuilder(ctx context.Context, transaction pgx.Tx, builder sq.UpdateBuilder) error
	ExecInsertBuilderWithReturnValue(ctx context.Context, transaction pgx.Tx, builder sq.InsertBuilder, outID any) error
	Builder() sq.StatementBuilderType
	SendBatch(ctx context.Context, transaction pgx.Tx, b *pgx.Batch) pgx.BatchResults
	GetCount(ctx context.Context, transaction pgx.Tx, builder sq.SelectBuilder) (int64, error)
	DBErr(rootError error) error
	TruncateTable(ctx context.Context, table string) error
	WrapTx(ctx context.Context, fn func(pgx.Tx) error) error
}

type dbQueryTracer struct{}

func (tracer *dbQueryTracer) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	slog.Info("Executing command", slog.String("sql", data.SQL), slog.Any("args", data.Args))

	return ctx
}

func (tracer *dbQueryTracer) TraceQueryEnd(_ context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
}

type postgresStore struct {
	conn *pgxpool.Pool
	// Use $ for pg based queries.
	sb          sq.StatementBuilderType
	dsn         string
	autoMigrate bool
	migrated    bool
	logQueries  bool
}

func (db *postgresStore) WrapTx(ctx context.Context, txFunc func(pgx.Tx) error) error {
	transaction, errTx := db.Begin(ctx)
	if errTx != nil {
		return db.DBErr(errTx)
	}

	if err := txFunc(transaction); err != nil {
		if errRollback := transaction.Rollback(ctx); errRollback != nil {
			return db.DBErr(errRollback)
		}

		return err
	}

	if err := transaction.Commit(ctx); err != nil {
		return db.DBErr(err)
	}

	return nil
}

func New(dsn string, autoMigrate bool, logQueries bool) Database {
	return &postgresStore{
		sb:          sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
		dsn:         dsn,
		autoMigrate: autoMigrate,
		logQueries:  logQueries,
	}
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

func (db *postgresStore) Pool() *pgxpool.Pool {
	return db.conn
}

// Connect sets up underlying required services.
func (db *postgresStore) Connect(ctx context.Context) error {
	cfg, errConfig := pgxpool.ParseConfig(db.dsn)
	if errConfig != nil {
		return fmt.Errorf("unable to parse db config/dsn: %w", errConfig)
	}

	cfg.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		pgxuuid.Register(conn.TypeMap())

		return nil
	}

	if db.logQueries {
		cfg.ConnConfig.Tracer = &dbQueryTracer{}
	}

	if db.autoMigrate && !db.migrated {
		if errMigrate := db.migrate(MigrateUp, db.dsn); errMigrate != nil {
			return fmt.Errorf("could not migrate schema: %w", errMigrate)
		}

		slog.Info("Migration completed successfully")
	}

	dbConn, errConnectConfig := pgxpool.NewWithConfig(ctx, cfg)
	if errConnectConfig != nil {
		return errors.Join(errConnectConfig, domain.ErrPoolFailed)
	}

	db.conn = dbConn

	return nil
}

func (db *postgresStore) SendBatch(ctx context.Context, transaction pgx.Tx, batch *pgx.Batch) pgx.BatchResults { //nolint:ireturn
	if transaction != nil {
		return transaction.SendBatch(ctx, batch)
	}

	return db.conn.SendBatch(ctx, batch)
}

func (db *postgresStore) Builder() sq.StatementBuilderType {
	return db.sb
}

//nolint:ireturn
func (db *postgresStore) Query(ctx context.Context, transaction pgx.Tx, query string, args ...any) (pgx.Rows, error) {
	var (
		rows pgx.Rows
		err  error
	)
	if transaction != nil {
		rows, err = transaction.Query(ctx, query, args...)
	} else {
		rows, err = db.conn.Query(ctx, query, args...)
	}

	return rows, err //nolint:wrapcheck
}

func (db *postgresStore) QueryBuilder(ctx context.Context, transaction pgx.Tx, builder sq.SelectBuilder) (pgx.Rows, error) { //nolint:ireturn
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return nil, db.DBErr(errQuery)
	}

	rows, err := db.Query(ctx, transaction, query, args...)

	return rows, err //nolint:wrapcheck
}

func (db *postgresStore) QueryRow(ctx context.Context, transaction pgx.Tx, query string, args ...any) pgx.Row { //nolint:ireturn
	if transaction != nil {
		return transaction.QueryRow(ctx, query, args...)
	}

	return db.conn.QueryRow(ctx, query, args...)
}

func (db *postgresStore) QueryRowBuilder(ctx context.Context, transaction pgx.Tx, builder sq.SelectBuilder) (pgx.Row, error) { //nolint:ireturn
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return nil, errQuery //nolint:wrapcheck
	}

	if transaction != nil {
		return transaction.QueryRow(ctx, query, args...), nil
	}

	return db.conn.QueryRow(ctx, query, args...), nil
}

func (db *postgresStore) Exec(ctx context.Context, transaction pgx.Tx, query string, args ...any) error {
	var err error
	if transaction != nil {
		_, err = transaction.Exec(ctx, query, args...)
	} else {
		_, err = db.conn.Exec(ctx, query, args...)
	}

	return err //nolint:wrapcheck
}

func (db *postgresStore) ExecInsertBuilder(ctx context.Context, transaction pgx.Tx, builder sq.InsertBuilder) error {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return db.DBErr(errQuery)
	}

	return db.Exec(ctx, transaction, query, args...) //nolint:wrapcheck
}

func (db *postgresStore) ExecDeleteBuilder(ctx context.Context, transaction pgx.Tx, builder sq.DeleteBuilder) error {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return errQuery //nolint:wrapcheck
	}

	return db.Exec(ctx, transaction, query, args...) //nolint:wrapcheck
}

func (db *postgresStore) ExecUpdateBuilder(ctx context.Context, transaction pgx.Tx, builder sq.UpdateBuilder) error {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return errQuery //nolint:wrapcheck
	}

	return db.Exec(ctx, transaction, query, args...) //nolint:wrapcheck
}

func (db *postgresStore) ExecInsertBuilderWithReturnValue(ctx context.Context, transaction pgx.Tx, builder sq.InsertBuilder, outID any) error {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return errQuery //nolint:wrapcheck
	}

	if errScan := db.
		QueryRow(ctx, transaction, query, args...).
		Scan(outID); errScan != nil {
		return errScan //nolint:wrapcheck
	}

	return nil
}

func (db *postgresStore) Begin(ctx context.Context) (pgx.Tx, error) { //nolint:ireturn
	return db.conn.Begin(ctx) //nolint:wrapcheck
}

func (db *postgresStore) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) { //nolint:ireturn
	return db.conn.BeginTx(ctx, txOptions) //nolint:wrapcheck
}

// Close will close the underlying database connection if it exists.
func (db *postgresStore) Close() error {
	if db.conn != nil {
		db.conn.Close()
	}

	return nil
}

func (db *postgresStore) GetCount(ctx context.Context, transaction pgx.Tx, builder sq.SelectBuilder) (int64, error) {
	countQuery, argsCount, errCountQuery := builder.ToSql()
	if errCountQuery != nil {
		return 0, errors.Join(errCountQuery, domain.ErrCreateQuery)
	}

	var count int64
	if errCount := db.
		QueryRow(ctx, transaction, countQuery, argsCount...).
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

	rows, errExec := db.Query(ctx, nil, query, args...)
	if errExec != nil {
		return db.DBErr(errExec)
	}

	rows.Close()

	return nil
}
