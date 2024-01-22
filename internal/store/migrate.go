package store

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

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

// migrate database schema.
func (db *postgresStore) migrate(action MigrationAction, dsn string) error {
	defer func() {
		db.migrated = true
	}()

	instance, errOpen := sql.Open("pgx", dsn)
	if errOpen != nil {
		return errors.Join(errOpen, errors.New("Failed to open database for migration"))
	}

	if errPing := instance.Ping(); errPing != nil {
		return errors.Join(errPing, errors.New("Cannot migrate, failed to connect to target server"))
	}

	driver, errMigrate := pgxMigrate.WithInstance(instance, &pgxMigrate.Config{
		MigrationsTable:       "_migration",
		SchemaName:            "public",
		StatementTimeout:      60 * time.Second,
		MultiStatementEnabled: true,
	})
	if errMigrate != nil {
		return errors.Join(errMigrate, errors.New("failed to create migration driver"))
	}

	defer util.LogCloser(driver, db.log)

	source, errHTTPFS := httpfs.New(http.FS(migrations), "migrations")
	if errHTTPFS != nil {
		return errors.Join(errHTTPFS, errors.New("Failed to create migration httpfs"))
	}

	migrator, errMigrateInstance := migrate.NewWithInstance("iofs", source, "pgx", driver)
	if errMigrateInstance != nil {
		return errors.Join(errMigrateInstance, errors.New("Failed to migrator up"))
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
		return errors.Join(errMigration, errors.New("Failed to perform migration"))
	}

	return nil
}
