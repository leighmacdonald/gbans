package queue

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/riverqueue/river/rivershared/util/slogutil"
	"github.com/riverqueue/river/rivertype"
)

var (
	ErrMigrateQueue = errors.New("failed to migrate queue database tables")
	ErrSetupQueue   = errors.New("failed to setup queue client")
	ErrStartQueue   = errors.New("failed to start job client")
)

type JobPriority int

const (
	RealTime JobPriority = 1
	High     JobPriority = 2
	Normal   JobPriority = 3
	Slow     JobPriority = 4
)

type NamedQueue string

const (
	Default NamedQueue = "default"
	Demo    NamedQueue = "demo"
)

func Init(ctx context.Context, dbPool *pgxpool.Pool) error {
	migrator, errNew := rivermigrate.New[pgx.Tx](riverpgxv5.New(dbPool), nil)
	if errNew != nil {
		return errors.Join(errNew, ErrMigrateQueue)
	}

	res, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return errors.Join(err, ErrMigrateQueue)
	}

	for _, ver := range res.Versions {
		slog.Info("Migrated river version", slog.Int("version", ver.Version))
	}

	return nil
}

func Client(dbPool *pgxpool.Pool, workers *river.Workers, periodic []*river.PeriodicJob) (*river.Client[pgx.Tx], error) {
	newRiverClient, err := river.NewClient[pgx.Tx](riverpgxv5.New(dbPool), &river.Config{
		Logger:     slog.New(&slogutil.SlogMessageOnlyHandler{Level: slog.LevelWarn}),
		JobTimeout: time.Minute * 5,
		Queues: map[string]river.QueueConfig{
			string(Default): {MaxWorkers: 2},
			string(Demo):    {MaxWorkers: 1},
		},
		Workers:      workers,
		PeriodicJobs: periodic,
		ErrorHandler: &errorHandler{},
		MaxAttempts:  3,
	})
	if err != nil {
		return nil, errors.Join(err, ErrSetupQueue)
	}

	return newRiverClient, nil
}

type errorHandler struct{}

func (*errorHandler) HandleError(_ context.Context, job *rivertype.JobRow, err error) *river.ErrorHandlerResult {
	slog.Error("Job returned error", log.ErrAttr(err),
		slog.String("queue", job.Queue), slog.String("kind", job.Kind),
		slog.String("args", string(job.EncodedArgs)))

	return nil
}

func (*errorHandler) HandlePanic(_ context.Context, job *rivertype.JobRow, panicVal any, trace string) *river.ErrorHandlerResult {
	slog.Error("Job panic",
		slog.String("trace", trace), slog.Any("value", panicVal),
		slog.String("queue", job.Queue), slog.String("kind", job.Kind),
		slog.String("args", string(job.EncodedArgs)))

	return nil
}
