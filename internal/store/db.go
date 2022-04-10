package store

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/golang-migrate/migrate/v4"
	pgxMigrate "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/logrusadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"time"
)

var (
	// ErrNoResult is returned on successful queries which return no rows
	ErrNoResult = errors.New("No results found")
	// ErrDuplicate is returned when a duplicate row result is attempted to be inserted
	ErrDuplicate = errors.New("Duplicate entity")
	// Use $ for pg based queries
	sb = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	//go:embed migrations
	migrations embed.FS
)

type tableName string

const (
	tableNetLocation tableName = "net_location"
	tableNetProxy    tableName = "net_proxy"
	tableNetASN      tableName = "net_asn"
	//tablePersonIP    tableName = "person_ip"
	tableServer tableName = "server"
	tableDemo   tableName = "demo"
)

// QueryFilter provides a structure for common query parameters
type QueryFilter struct {
	Offset   uint64 `json:"offset,omitempty" uri:"offset" binding:"gte=0"`
	Limit    int    `json:"limit,omitempty" uri:"limit" binding:"gte=0,lte=1000"`
	SortDesc bool   `json:"desc,omitempty" uri:"desc"`
	Query    string `json:"query,omitempty" uri:"query"`
	OrderBy  string `json:"order_by,omitempty" uri:"order_by"`
	Deleted  bool   `json:"deleted,omitempty" uri:"deleted"`
}

func (qf *QueryFilter) orderString() string {
	dir := "DESC"
	if !qf.SortDesc {
		dir = "ASC"
	}
	return fmt.Sprintf("%s %s", qf.OrderBy, dir)
}

func NewQueryFilter(query string) *QueryFilter {
	return &QueryFilter{
		Limit:    1000,
		Offset:   0,
		SortDesc: true,
		OrderBy:  "created_on",
		Query:    query,
	}
}

// New sets up underlying required services.
func New(dsn string) (Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Unable to parse config: %v", err)
	}
	ndb := pgStore{}
	if config.DB.AutoMigrate {
		if errM := ndb.Migrate(MigrateUp); errM != nil {
			if errM.Error() == "no change" {
				log.Debugf("Migration at latest version")
			} else {
				log.Fatalf("Could not migrate schema: %v", errM)
			}
		} else {
			log.Infof("Migration completed successfully")
		}
	}
	if config.DB.LogQueries {
		lgr := log.New()
		lvl, err2 := log.ParseLevel(config.Log.Level)
		if err2 != nil {
			log.Fatalf("Invalid log level: %s (%v)", config.Log.Level, err2)
		}
		lgr.SetLevel(lvl)
		lgr.SetFormatter(&log.TextFormatter{
			ForceColors:   config.Log.ForceColours,
			DisableColors: config.Log.DisableColours,
			FullTimestamp: config.Log.FullTimestamp,
		})
		lgr.SetReportCaller(config.Log.ReportCaller)
		cfg.ConnConfig.Logger = logrusadapter.NewLogger(lgr)
	}
	dbConn, err3 := pgxpool.ConnectConfig(context.Background(), cfg)
	if err3 != nil {
		log.Fatalf("Failed to connect to database: %v", err3)
	}
	return &pgStore{c: dbConn}, nil
}

// pgStore implements Store against a postgresql database
type pgStore struct {
	c *pgxpool.Pool
}

func (db *pgStore) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return db.c.Query(ctx, query, args...)
}

// Close will close the underlying database connection if it exists
func (db *pgStore) Close() error {
	if db.c != nil {
		db.c.Close()
	}
	return nil
}

func (db *pgStore) truncateTable(ctx context.Context, table tableName) error {
	if _, err := db.c.Exec(ctx, fmt.Sprintf("TRUNCATE %s;", table)); err != nil {
		return Err(err)
	}
	return nil
}

// Err is used to wrap common database errors in own own error types
func Err(err error) error {
	if err == nil {
		return err
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return ErrDuplicate
		default:
			log.Errorf("Unhandled store error: (%s) %s", pgErr.Code, pgErr.Message)
			return err
		}
	}
	if err.Error() == "no rows in result set" {
		return ErrNoResult
	}
	return err
}

// MigrationAction is the type of migration to perform
type MigrationAction int

const (
	// MigrateUp Fully upgrades the schema
	MigrateUp = iota
	// MigrateDn Fully downgrades the schema
	MigrateDn
	// MigrateUpOne Upgrade the schema by one revision
	// MigrateUpOne
	// MigrateDownOne Downgrade the schema by one revision
	// MigrateDownOne
)

// Migrate e
func (db *pgStore) Migrate(action MigrationAction) error {
	instance, err := sql.Open("pgx", config.DB.DSN)
	if err != nil {
		return errors.Wrapf(err, "Failed to open database for migration")
	}
	if errPing := instance.Ping(); errPing != nil {
		return errors.Wrapf(errPing, "Cannot migrate, failed to connect to target server")
	}
	driver, err2 := pgxMigrate.WithInstance(instance, &pgxMigrate.Config{
		MigrationsTable:       "_migration",
		SchemaName:            "public",
		StatementTimeout:      60 * time.Second,
		MultiStatementEnabled: true,
	})
	if err2 != nil {
		return errors.Wrapf(err2, "failed to create migration driver")
	}
	defer func() {
		if e := driver.Close(); e != nil {
			log.Errorf("Failed to close migrate driver: %v", e)
		}
	}()
	source, err3 := httpfs.New(http.FS(migrations), "migrations")
	if err3 != nil {
		return err3
	}
	m, err4 := migrate.NewWithInstance("iofs", source, "pgx", driver)
	if err4 != nil {
		return errors.Wrapf(err4, "Failed to migrate up")
	}
	switch action {
	//case MigrateUpOne:
	//	return m.Steps(1)
	case MigrateDn:
		return m.Down()
	//case MigrateDownOne:
	//	return m.Steps(-1)
	case MigrateUp:
		fallthrough
	default:
		return m.Up()
	}
}

// Import will import bans from a root folder.
// The formatting is JSON with the importedBan schema defined inline
// Valid filenames are: main_ban.json
func (db *pgStore) Import(ctx context.Context, root string) error {
	type importedBan struct {
		BanID      int    `json:"ban_id"`
		SteamID    uint64 `json:"steam_id"`
		AuthorID   uint64 `json:"author_id"`
		BanType    int    `json:"ban_type"`
		Reason     int    `json:"reason"`
		ReasonText string `json:"reason_text"`
		Note       string `json:"note"`
		Until      int    `json:"until"`
		CreatedOn  int    `json:"created_on"`
		UpdatedOn  int    `json:"updated_on"`
		BanSource  int    `json:"ban_source"`
	}

	return filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
		switch d.Name() {
		case "main_ban.json":
			b, err := ioutil.ReadFile(path.Join(root, d.Name()))
			if err != nil {
				return err
			}
			var imported []importedBan
			if err2 := json.Unmarshal(b, &imported); err2 != nil {
				return err2
			}
			for _, im := range imported {
				b1 := model.NewPerson(steamid.SID64(im.SteamID))
				b2 := model.NewPerson(steamid.SID64(im.AuthorID))
				if e1 := db.GetOrCreatePersonBySteamID(ctx, steamid.SID64(im.SteamID), &b1); e1 != nil {
					return e1
				}
				if e2 := db.GetOrCreatePersonBySteamID(ctx, steamid.SID64(im.AuthorID), &b2); e2 != nil {
					return e2
				}
				sum, err3 := steamweb.PlayerSummaries(steamid.Collection{b1.SteamID, b2.SteamID})
				if err3 != nil {
					log.Errorf("Failed to get player summary: %v", err3)
					return err3
				}
				if len(sum) > 0 {
					b1.PlayerSummary = &sum[0]
					if err4 := db.SavePerson(ctx, &b1); err4 != nil {
						return err4
					}
					if b2.SteamID.Valid() && len(sum) > 1 {
						b2.PlayerSummary = &sum[1]
						if err5 := db.SavePerson(ctx, &b2); err5 != nil {
							return err5
						}
					}
				}
				bn := model.NewBan(b1.SteamID, b2.SteamID, 0)
				bn.ValidUntil = time.Unix(int64(im.Until), 0)
				bn.ReasonText = im.ReasonText
				bn.CreatedOn = time.Unix(int64(im.CreatedOn), 0)
				bn.UpdatedOn = time.Unix(int64(im.UpdatedOn), 0)
				bn.Source = model.System
				if err4 := db.SaveBan(ctx, &bn); err4 != nil {
					return err4
				}
			}
		}
		return nil
	})
}
