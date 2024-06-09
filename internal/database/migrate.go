package database

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/golang-migrate/migrate/v4"
	pgxMigrate "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"github.com/leighmacdonald/gbans/pkg/util"
)

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

var (
	ErrOpenDB          = errors.New("failed to open database driver")
	ErrPing            = errors.New("failed to ping database")
	ErrMigrationDriver = errors.New("failed to setup migration driver")
	ErrMigrateFS       = errors.New("could not setup http.FS migration source")
	ErrMigrateCreate   = errors.New("failed to setup migration instance")
	ErrMigrate         = errors.New("migration failed to complete")
)

// migrate database schema.
func (db *postgresStore) migrate(action MigrationAction, dsn string) error {
	defer func() {
		db.migrated = true
	}()

	instance, errOpen := sql.Open("pgx", dsn)
	if errOpen != nil {
		return errors.Join(errOpen, ErrOpenDB)
	}

	if errPing := instance.Ping(); errPing != nil {
		return errors.Join(errPing, ErrPing)
	}

	driver, errMigrate := pgxMigrate.WithInstance(instance, &pgxMigrate.Config{
		MigrationsTable:       "_migration",
		SchemaName:            "public",
		MultiStatementEnabled: false,
	})
	if errMigrate != nil {
		return errors.Join(errMigrate, ErrMigrationDriver)
	}

	defer util.LogCloser(driver)

	source, errHTTPFS := httpfs.New(http.FS(migrations), "migrations")
	if errHTTPFS != nil {
		return errors.Join(errHTTPFS, ErrMigrateFS)
	}

	migrator, errMigrateInstance := migrate.NewWithInstance("iofs", source, "pgx", driver)
	if errMigrateInstance != nil {
		return errors.Join(errMigrateInstance, ErrMigrateCreate)
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
		return errors.Join(errMigration, ErrMigrate)
	}

	return nil
}
