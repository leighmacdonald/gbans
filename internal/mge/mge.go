package mge

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
}
