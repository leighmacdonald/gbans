package store

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/golang-migrate/migrate/v4"
	pgxMigrate "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/pkg/errors"
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

// migrate database schema.
func (db *Database) migrate(action MigrationAction, dsn string) error {
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
