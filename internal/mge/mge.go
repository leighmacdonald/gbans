package mge

<<<<<<< HEAD
import (
	"context"
	"errors"
)

var ErrInvalidMode = errors.New("invalid mode")

type MGE struct {
	repo Repository
}

func NewMGE(repo Repository) MGE {
	return MGE{repo: repo}
}

func (m MGE) Query(ctx context.Context, opts QueryOpts) ([]PlayerStats, int64, error) {
	return m.repo.Query(ctx, opts)
}

func (m MGE) History(ctx context.Context, opts HistoryOpts) ([]Duels, int64, error) {
	if opts.Mode != OneVsOne && opts.Mode != TwoVsTwo {
		return nil, 0, ErrInvalidMode
	}

	return m.repo.History(ctx, opts)
||||||| parent of 179f35e8 (Add overall ranking table)
=======
import "context"

type MGE struct {
	repo Repository
}

func NewMGE(repo Repository) MGE {
	return MGE{repo: repo}
}

func (m *MGE) Query(ctx context.Context, opts QueryOpts) ([]PlayerStats, int64, error) {
	return m.repo.Query(ctx, opts)
>>>>>>> 179f35e8 (Add overall ranking table)
}
