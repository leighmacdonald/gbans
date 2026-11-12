package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgerrcode"
	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
)

const (
	// DefaultMaxConns is the size of the shared database pool when the DSN
	// does not explicitly set pool_max_conns. pgxpool's own default is
	// max(4, NumCPU), which would silently grow with the host's CPU count
	// (and can exceed the database's max_connections on large machines), so
	// a fixed default is pinned here instead. Operators can still override
	// it per-DSN via pool_max_conns.
	DefaultMaxConns int32 = 16
	// queryAcquireTimeout bounds how long a query waits for a free pooled
	// connection before failing. A saturated pool then surfaces as a quick
	// error instead of requests (or background workers) queueing
	// indefinitely behind it.
	queryAcquireTimeout = 30 * time.Second
)

var (
	// ErrNoResult is returned on successful queries which return no rows.
	ErrNoResult = errors.New("no results found")
	// ErrDuplicate is returned when a duplicate row result is attempted to be inserted.
	ErrDuplicate   = errors.New("entity already exists")
	ErrPoolFailed  = errors.New("could not create store pool")
	ErrCreateQuery = errors.New("failed to generate query")
	ErrSaveChanges = errors.New("cannot save changes")
	ErrCloseBatch  = errors.New("failed to close batch request")
	ErrScanResult  = errors.New("failed to scan result")
	ErrInvalidView = errors.New("invalid view name")
)

//go:embed migrations
var migrations embed.FS

// Database is the common database interface. All errors from callers should be wrapped in Err as they
// are not automatically wrapped.
type Database interface {
	Pool() *pgxpool.Pool
	Connect(ctx context.Context) error
	Close() error
	RefreshMaterializedView(ctx context.Context, viewName string) error
	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) pgx.Row
	Exec(ctx context.Context, query string, args ...any) error
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	GetCount(ctx context.Context, builder sq.SelectBuilder) (uint64, error)
	TruncateTable(ctx context.Context, table string) error
	WrapTx(ctx context.Context, fn func(pgx.Tx) error) error

	Builder() sq.StatementBuilderType
	QueryBuilder(ctx context.Context, builder sq.SelectBuilder) (pgx.Rows, error)
	QueryRowBuilder(ctx context.Context, builder sq.SelectBuilder) (pgx.Row, error)
	ExecInsertBuilder(ctx context.Context, builder sq.InsertBuilder) error
	ExecDeleteBuilder(ctx context.Context, builder sq.DeleteBuilder) error
	ExecUpdateBuilder(ctx context.Context, builder sq.UpdateBuilder) error
	ExecInsertBuilderWithReturnValue(ctx context.Context, builder sq.InsertBuilder, outID any) error
}

type dbQueryTracer struct{}

func (tracer *dbQueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	slog.Info("Executing command", slog.String("sql", data.SQL), slog.Any("args", data.Args))

	return ctx
}

func (tracer *dbQueryTracer) TraceQueryEnd(_ context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {}

type PgStore struct {
	conn *pgxpool.Pool
	// Use $ for pg based queries.
	sb             sq.StatementBuilderType
	dsn            string
	autoMigrate    bool
	migrateOnce    sync.Once
	logQueries     bool
	validViewNames []string
}

func (db *PgStore) WrapTx(ctx context.Context, txFunc func(pgx.Tx) error) error {
	transaction, errTx := db.Begin(ctx)
	if errTx != nil {
		return Err(errTx)
	}

	if err := txFunc(transaction); err != nil {
		if errRollback := transaction.Rollback(ctx); errRollback != nil {
			slog.Error("Failed to rollback transaction", slog.String("error", errRollback.Error()))
		}

		return err
	}

	if err := transaction.Commit(ctx); err != nil {
		return Err(err)
	}

	return nil
}

func New(dsn string, autoMigrate bool, logQueries bool) *PgStore {
	return &PgStore{
		sb:          sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
		dsn:         dsn,
		autoMigrate: autoMigrate,
		logQueries:  logQueries,
		validViewNames: []string{
			"stats_summary_daily_overall_view",
			"stats_summary_daily_variants_view",
			"stats_summary_weekly_overall_view",
			"stats_summary_weekly_variants_view",
			"stats_summary_monthly_overall_view",
			"stats_summary_monthly_variants_view",
			"stats_summary_yearly_overall_view",
			"stats_summary_yearly_variants_view",
			"stats_summary_alltime_overall_view",  //nolint:misspell // name must match the database view.
			"stats_summary_alltime_variants_view", //nolint:misspell // name must match the database view.
			"stats_map_daily_view",
			"stats_map_daily_player_view",
			"stats_map_alltime_view", //nolint:misspell // name must match the database view.
			"stats_weapons_view",
		},
	}
}

// Err is used to wrap common database errors in owr own error types.
func Err(rootError error) error {
	if rootError == nil {
		return nil
	}

	if errors.Is(rootError, pgx.ErrNoRows) {
		return ErrNoResult
	}

	if pgErr, ok := errors.AsType[*pgconn.PgError](rootError); ok {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return ErrDuplicate
		default:
			return rootError
		}
	}

	return rootError
}

func (db *PgStore) Pool() *pgxpool.Pool {
	return db.conn
}

// acquireConn grabs a pooled connection for a single statement, failing fast
// (queryAcquireTimeout) when the pool is saturated instead of blocking the
// caller indefinitely.
func (db *PgStore) acquireConn(ctx context.Context) (*pgxpool.Conn, error) {
	acquireCtx, cancel := context.WithTimeout(ctx, queryAcquireTimeout)
	defer cancel()

	conn, err := db.conn.Acquire(acquireCtx)
	if err != nil {
		return nil, fmt.Errorf("acquiring database connection: %w", err)
	}

	return conn, nil
}

// poolRows releases its pooled connection when the rows are closed, or
// automatically once they are exhausted or a read fails — mirroring
// pgxpool's own poolRows, so callers that never call Close do not leak a
// connection.
type poolRows struct {
	pgx.Rows

	release func()
}

func (r *poolRows) Next() bool {
	next := r.Rows.Next()
	if !next {
		r.Close()
	}

	return next
}

func (r *poolRows) Scan(dest ...any) error {
	err := r.Rows.Scan(dest...)
	if err != nil {
		r.Close()
	}

	return err //nolint:wrapcheck
}

func (r *poolRows) Values() ([]any, error) {
	values, errValues := r.Rows.Values()
	if errValues != nil {
		r.Close()
	}

	return values, errValues //nolint:wrapcheck
}

func (r *poolRows) Close() {
	r.Rows.Close()
	r.release()
}

// poolRow releases its pooled connection when the row is scanned.
type poolRow struct {
	pgx.Row

	release func()
}

func (r *poolRow) Scan(dest ...any) error {
	panicked := true
	defer func() {
		if panicked {
			r.release()
		}
	}()

	err := r.Row.Scan(dest...)
	panicked = false
	r.release()

	return err //nolint:wrapcheck
}

// poolTx releases its pooled connection when the transaction ends.
type poolTx struct {
	pgx.Tx

	release func()
}

func (t *poolTx) Commit(ctx context.Context) error {
	err := t.Tx.Commit(ctx)
	t.release()

	return err //nolint:wrapcheck
}

func (t *poolTx) Rollback(ctx context.Context) error {
	err := t.Tx.Rollback(ctx)
	t.release()

	return err //nolint:wrapcheck
}

// poolBatchResults releases its pooled connection when the batch results are
// closed.
type poolBatchResults struct {
	pgx.BatchResults

	release func()
}

func (b *poolBatchResults) Close() error {
	err := b.BatchResults.Close()
	b.release()

	return err //nolint:wrapcheck
}

// errorRow is a pgx.Row whose Scan always returns err (used when acquiring
// the underlying connection fails).
type errorRow struct{ err error }

func (r errorRow) Scan(...any) error { return r.err }

// errorBatchResults is a pgx.BatchResults that always returns err (used when
// acquiring the underlying connection fails).
type errorBatchResults struct{ err error }

func (b errorBatchResults) Exec() (pgconn.CommandTag, error) { return pgconn.CommandTag{}, b.err }
func (b errorBatchResults) Query() (pgx.Rows, error)         { return nil, b.err }
func (b errorBatchResults) QueryRow() pgx.Row {
	//nolint:staticcheck // keyed literal: errorBatchResults is not convertible to errorRow
	return errorRow{err: b.err}
}
func (b errorBatchResults) Close() error { return b.err }
func (b errorBatchResults) Tx() pgx.Tx   { return nil }

func (db *PgStore) RefreshMaterializedView(ctx context.Context, viewName string) error {
	timeStart := time.Now()
	if !slices.Contains(db.validViewNames, viewName) {
		return ErrInvalidView
	}

	if err := db.Exec(ctx, "refresh materialized view concurrently "+pgx.Identifier([]string{viewName}).Sanitize()); err != nil {
		return Err(err)
	}

	slog.Debug("Refreshed view successfully", slog.String("view", viewName), slog.Duration("duration", time.Since(timeStart)))

	return nil
}

// dsnSetsPoolParam reports whether dsn sets the given pgxpool DSN parameter
// (e.g. pool_max_conns), meaning the operator explicitly chose the value.
func dsnSetsPoolParam(dsn, param string) bool {
	u, err := url.Parse(dsn)
	if err != nil {
		return false
	}

	return u.Query().Get(param) != ""
}

// Connect sets up underlying required services.
func (db *PgStore) Connect(ctx context.Context) error {
	cfg, errConfig := pgxpool.ParseConfig(db.dsn)
	if errConfig != nil {
		return fmt.Errorf("unable to parse db config/dsn: %w", errConfig)
	}

	// pgxpool.ParseConfig already defaults MaxConns to max(4, NumCPU) when
	// the DSN does not set pool_max_conns, so an "is it zero?" check here
	// would never fire. Pin the documented default instead, but keep the
	// operator's explicit DSN value when one was given.
	if !dsnSetsPoolParam(db.dsn, "pool_max_conns") {
		cfg.MaxConns = DefaultMaxConns
	}

	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxuuid.Register(conn.TypeMap())

		return registerPermissionType(ctx, conn)
	}

	if db.logQueries {
		cfg.ConnConfig.Tracer = &dbQueryTracer{}
	}

	if db.autoMigrate {
		var err error
		db.migrateOnce.Do(func() {
			if errMigrate := Migrate(MigrateUp, db.dsn); errMigrate != nil {
				err = fmt.Errorf("could not migrate schema: %w", errMigrate)
			}
		})
		if err != nil {
			return err
		}
	}

	dbConn, errConnectConfig := pgxpool.NewWithConfig(ctx, cfg)
	if errConnectConfig != nil {
		return errors.Join(errConnectConfig, ErrPoolFailed)
	}

	db.conn = dbConn

	return nil
}

func (db *PgStore) SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults { //nolint:ireturn
	conn, err := db.acquireConn(ctx)
	if err != nil {
		return errorBatchResults{err: err}
	}

	return &poolBatchResults{BatchResults: conn.SendBatch(ctx, batch), release: conn.Release}
}

func (db *PgStore) Builder() sq.StatementBuilderType {
	return db.sb
}

//nolint:ireturn
func (db *PgStore) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	conn, err := db.acquireConn(ctx)
	if err != nil {
		return nil, err
	}

	rows, errQuery := conn.Query(ctx, query, args...)
	if errQuery != nil {
		conn.Release()

		return nil, errQuery //nolint:wrapcheck
	}

	return &poolRows{Rows: rows, release: conn.Release}, nil
}

func (db *PgStore) QueryBuilder(ctx context.Context, builder sq.SelectBuilder) (pgx.Rows, error) { //nolint:ireturn
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return nil, Err(errQuery)
	}

	rows, err := db.Query(ctx, query, args...)

	return rows, err //nolint:wrapcheck
}

func (db *PgStore) QueryRow(ctx context.Context, query string, args ...any) pgx.Row { //nolint:ireturn
	conn, err := db.acquireConn(ctx)
	if err != nil {
		return errorRow{err: err}
	}

	return &poolRow{Row: conn.QueryRow(ctx, query, args...), release: conn.Release}
}

func (db *PgStore) QueryRowBuilder(ctx context.Context, builder sq.SelectBuilder) (pgx.Row, error) { //nolint:ireturn
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return nil, errQuery //nolint:wrapcheck
	}

	return db.QueryRow(ctx, query, args...), nil
}

func (db *PgStore) Exec(ctx context.Context, query string, args ...any) error {
	conn, err := db.acquireConn(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	_, errExec := conn.Exec(ctx, query, args...)

	return errExec //nolint:wrapcheck
}

func (db *PgStore) ExecInsertBuilder(ctx context.Context, builder sq.InsertBuilder) error {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return Err(errQuery)
	}

	return db.Exec(ctx, query, args...) //nolint:wrapcheck
}

func (db *PgStore) ExecDeleteBuilder(ctx context.Context, builder sq.DeleteBuilder) error {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return errQuery //nolint:wrapcheck
	}

	return db.Exec(ctx, query, args...) //nolint:wrapcheck
}

func (db *PgStore) ExecUpdateBuilder(ctx context.Context, builder sq.UpdateBuilder) error {
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return errQuery //nolint:wrapcheck
	}

	return db.Exec(ctx, query, args...) //nolint:wrapcheck
}

func (db *PgStore) ExecInsertBuilderWithReturnValue(ctx context.Context, builder sq.InsertBuilder, outID any) error {
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

func (db *PgStore) Begin(ctx context.Context) (pgx.Tx, error) { //nolint:ireturn
	conn, err := db.acquireConn(ctx)
	if err != nil {
		return nil, err
	}

	transaction, errTx := conn.Begin(ctx)
	if errTx != nil {
		conn.Release()

		return nil, errTx //nolint:wrapcheck
	}

	return &poolTx{Tx: transaction, release: conn.Release}, nil
}

func (db *PgStore) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) { //nolint:ireturn
	conn, err := db.acquireConn(ctx)
	if err != nil {
		return nil, err
	}

	transaction, errTx := conn.BeginTx(ctx, txOptions)
	if errTx != nil {
		conn.Release()

		return nil, errTx //nolint:wrapcheck
	}

	return &poolTx{Tx: transaction, release: conn.Release}, nil
}

// Close will close the underlying database connection if it exists.
// registerPermissionType teaches pgx to encode and scan the roles.v1.Permission Go enum as the
// database `permission` enum type, for connections where the parameter OID is unknown as well as
// result columns.
func registerPermissionType(ctx context.Context, conn *pgx.Conn) error {
	var oid uint32
	if errOid := conn.QueryRow(ctx, "select oid from pg_type where typname = 'permission'").Scan(&oid); errOid != nil {
		return fmt.Errorf("querying permission type oid: %w", errOid)
	}

	typeMap := conn.TypeMap()
	typeMap.RegisterType(&pgtype.Type{OID: oid, Name: "permission", Codec: permissionCodec{}})
	typeMap.RegisterDefaultPgType(rolesv1.Permission(0), "permission")

	return nil
}

func (db *PgStore) Close() error {
	if db.conn != nil {
		db.conn.Close()
	}

	return nil
}

func (db *PgStore) GetCount(ctx context.Context, builder sq.SelectBuilder) (uint64, error) {
	countQuery, argsCount, errCountQuery := builder.ToSql()
	if errCountQuery != nil {
		return 0, errors.Join(errCountQuery, ErrCreateQuery)
	}

	var count uint64
	if errCount := db.
		QueryRow(ctx, countQuery, argsCount...).
		Scan(&count); errCount != nil {
		return 0, errCount //nolint:wrapcheck
	}

	return count, nil
}

func (db *PgStore) TruncateTable(ctx context.Context, table string) error {
	if err := db.Exec(ctx, "TRUNCATE TABLE "+pgx.Identifier([]string{table}).Sanitize()); err != nil {
		return Err(err)
	}

	return nil
}
