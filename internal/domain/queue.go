package domain

import "github.com/riverqueue/river"

type JobWorkers interface {
	RegisterWorkers(workers *river.Workers)
}
