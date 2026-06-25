package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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
	GetCount(ctx context.Context, builder sq.SelectBuilder) (uint64, error)
	TruncateTable(ctx context.Context, table string) error
	WrapTx(ctx context.Context, fn func(pgx.Tx) error) error
	Migrate(ctx context.Context, action MigrationAction, dsn string) error
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
	sb          sq.StatementBuilderType
	dsn         string
	autoMigrate bool
	migrated    bool
	logQueries  bool
}

func (db *PgStore) WrapTx(ctx context.Context, txFunc func(pgx.Tx) error) error {
	transaction, errTx := db.Begin(ctx)
	if errTx != nil {
		return Err(errTx)
	}

	if err := txFunc(transaction); err != nil {
		if errRollback := transaction.Rollback(ctx); errRollback != nil {
			return Err(errRollback)
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

func (db *PgStore) RefreshMaterializedView(ctx context.Context, viewName string) error {
	timeStart := time.Now()
	if err := db.Exec(ctx, "refresh materialized view concurrently "+viewName); err != nil {
		return Err(err)
	}

	slog.Debug("Refreshed view successfully", slog.String("view", viewName), slog.Duration("duration", time.Since(timeStart)))

	return nil
}

// Connect sets up underlying required services.
func (db *PgStore) Connect(ctx context.Context) error {
	cfg, errConfig := pgxpool.ParseConfig(db.dsn)
	if errConfig != nil {
		return fmt.Errorf("unable to parse db config/dsn: %w", errConfig)
	}

	cfg.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		registerUUIDCodec(conn.TypeMap())

		return nil
	}

	if db.logQueries {
		cfg.ConnConfig.Tracer = &dbQueryTracer{}
	}

	if db.autoMigrate && !db.migrated {
		if errMigrate := db.Migrate(ctx, MigrateUp, db.dsn); errMigrate != nil {
			return fmt.Errorf("could not migrate schema: %w", errMigrate)
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
	return db.conn.SendBatch(ctx, batch)
}

func (db *PgStore) Builder() sq.StatementBuilderType {
	return db.sb
}

//nolint:ireturn
func (db *PgStore) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return db.conn.Query(ctx, query, args...) //nolint:wrapcheck
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
	return db.conn.QueryRow(ctx, query, args...)
}

func (db *PgStore) QueryRowBuilder(ctx context.Context, builder sq.SelectBuilder) (pgx.Row, error) { //nolint:ireturn
	query, args, errQuery := builder.ToSql()
	if errQuery != nil {
		return nil, errQuery //nolint:wrapcheck
	}

	return db.conn.QueryRow(ctx, query, args...), nil
}

func (db *PgStore) Exec(ctx context.Context, query string, args ...any) error {
	var err error
	_, err = db.conn.Exec(ctx, query, args...)

	return err //nolint:wrapcheck
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
	return db.conn.Begin(ctx) //nolint:wrapcheck
}

func (db *PgStore) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error) { //nolint:ireturn
	return db.conn.BeginTx(ctx, txOptions) //nolint:wrapcheck
}

// Close will close the underlying database connection if it exists.
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
	query, args, errQueryArgs := sq.Delete(table).ToSql()
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

// errUUIDPlanScan is returned when pgx cannot find a scan plan for UUID.
var errUUIDPlanScan = errors.New("uuid: PlanScan did not find a plan")

// errUUIDNullScan is returned when scanning NULL into a non-null UUID pointer.
var errUUIDNullScan = errors.New("cannot scan NULL into *uuid.UUID")

// registerUUIDCodec registers gofrs/uuid/v5 types with the pgx type map.
// Replaces github.com/jackc/pgx-gofrs-uuid to reduce dependencies.
func registerUUIDCodec(tm *pgtype.Map) {
	tm.TryWrapEncodePlanFuncs = append([]pgtype.TryWrapEncodePlanFunc{tryWrapUUIDEncodePlan}, tm.TryWrapEncodePlanFuncs...)
	tm.TryWrapScanPlanFuncs = append([]pgtype.TryWrapScanPlanFunc{tryWrapUUIDScanPlan}, tm.TryWrapScanPlanFuncs...)

	tm.RegisterType(&pgtype.Type{
		Name:  "uuid",
		OID:   pgtype.UUIDOID,
		Codec: uuidCodec{},
	})
}

// uuidCodec wraps pgtype.UUIDCodec to decode directly to uuid.UUID.
type uuidCodec struct{ pgtype.UUIDCodec }

func (c uuidCodec) DecodeValue(typeMap *pgtype.Map, oid uint32, format int16, src []byte) (any, error) {
	if src == nil {
		return nil, nil //nolint:nilnil // null database value returns nil
	}

	var target uuid.UUID
	plan := typeMap.PlanScan(oid, format, &target)
	if plan == nil {
		return nil, errUUIDPlanScan //nolint:err113
	}

	if err := plan.Scan(src, &target); err != nil {
		return nil, err //nolint:wrapcheck // passthrough pgx scan error
	}

	return target, nil
}

// uuidPG is an alias for pgtype.UUID encoding compatibility.
// nolint:recvcheck // matches pgx-gofrs-uuid pattern (pointer ScanUUID, value UUIDValue)
type uuidPG struct {
	data [16]byte
}

//nolint:recvcheck // matches pgx-gofrs-uuid pattern (pointer ScanUUID, value UUIDValue)
func (u *uuidPG) ScanUUID(v pgtype.UUID) error {
	if !v.Valid {
		return errUUIDNullScan //nolint:err113
	}

	copy(u.data[:], v.Bytes[:])

	return nil
}

func (u uuidPG) UUIDValue() (pgtype.UUID, error) {
	return pgtype.UUID{Bytes: u.data, Valid: true}, nil
}

// nullUUIDPG handles nullable UUIDs.
// nolint:recvcheck // matches pgx-gofrs-uuid pattern (pointer ScanUUID, value UUIDValue)
type nullUUIDPG struct{ uuid.NullUUID }

//nolint:recvcheck // matches pgx-gofrs-uuid pattern (pointer ScanUUID, value UUIDValue)
func (u *nullUUIDPG) ScanUUID(v pgtype.UUID) error {
	*u = nullUUIDPG{NullUUID: uuid.NullUUID{UUID: uuid.UUID(v.Bytes), Valid: v.Valid}}

	return nil
}

func (u nullUUIDPG) UUIDValue() (pgtype.UUID, error) {
	return pgtype.UUID{Bytes: [16]byte(u.UUID), Valid: u.Valid}, nil
}

// wrap plans for encoding.
type wrapUUIDEncodePlan struct {
	plan pgtype.EncodePlan
}

func (p *wrapUUIDEncodePlan) SetNext(next pgtype.EncodePlan) { p.plan = next }

func (p *wrapUUIDEncodePlan) Encode(value any, buf []byte) ([]byte, error) {
	//nolint:forcetypeassert // type guaranteed by tryWrapUUIDEncodePlan
	buf, err := p.plan.Encode(uuidPG{data: [16]byte(value.(uuid.UUID))}, buf)

	return buf, err //nolint:wrapcheck // passthrough pgx encode error
}

type wrapNullUUIDEncodePlan struct {
	plan pgtype.EncodePlan
}

func (p *wrapNullUUIDEncodePlan) SetNext(next pgtype.EncodePlan) { p.plan = next }

func (p *wrapNullUUIDEncodePlan) Encode(value any, buf []byte) ([]byte, error) {
	//nolint:forcetypeassert // type guaranteed by tryWrapUUIDEncodePlan
	buf, err := p.plan.Encode(nullUUIDPG{NullUUID: value.(uuid.NullUUID)}, buf)

	return buf, err //nolint:wrapcheck // passthrough pgx encode error
}

// wrap plans for scanning.
type wrapUUIDScanPlan struct {
	plan pgtype.ScanPlan
}

func (p *wrapUUIDScanPlan) SetNext(next pgtype.ScanPlan) { p.plan = next }

func (p *wrapUUIDScanPlan) Scan(src []byte, dst any) error {
	//nolint:forcetypeassert // type guaranteed by tryWrapUUIDScanPlan
	target := dst.(*uuid.UUID) //nolint:varnamelen // local temporary
	var pg uuidPG
	if err := p.plan.Scan(src, &pg); err != nil {
		return err //nolint:wrapcheck // passthrough pgx scan error
	}

	copy((*target)[:], pg.data[:])

	return nil
}

type wrapNullUUIDScanPlan struct {
	plan pgtype.ScanPlan
}

func (p *wrapNullUUIDScanPlan) SetNext(next pgtype.ScanPlan) { p.plan = next }

func (p *wrapNullUUIDScanPlan) Scan(src []byte, dst any) error {
	//nolint:forcetypeassert // type guaranteed by tryWrapUUIDScanPlan
	target := dst.(*uuid.NullUUID) //nolint:varnamelen // local temporary
	var pg nullUUIDPG
	if err := p.plan.Scan(src, &pg); err != nil {
		return err //nolint:wrapcheck // passthrough pgx scan error
	}

	*target = pg.NullUUID

	return nil
}

func tryWrapUUIDEncodePlan(value any) (pgtype.WrappedEncodePlanNextSetter, any, bool) {
	switch value := value.(type) {
	case uuid.UUID:
		return &wrapUUIDEncodePlan{}, value, true
	case uuid.NullUUID:
		return &wrapNullUUIDEncodePlan{}, value, true
	}

	return nil, nil, false
}

func tryWrapUUIDScanPlan(target any) (pgtype.WrappedScanPlanNextSetter, any, bool) {
	switch target := target.(type) {
	case *uuid.UUID:
		return &wrapUUIDScanPlan{}, target, true
	case *uuid.NullUUID:
		return &wrapNullUUIDScanPlan{}, target, true
	}

	return nil, nil, false
}
