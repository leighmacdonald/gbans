package mge

import "context"

type MGE struct {
	repo Repository
}

func NewMGE(repo Repository) MGE {
	return MGE{repo: repo}
}

func (m *MGE) Query(ctx context.Context, opts QueryOpts) ([]PlayerStats, int64, error) {
	return m.repo.Query(ctx, opts)
}
