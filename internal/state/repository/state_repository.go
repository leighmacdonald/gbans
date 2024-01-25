package repository

import (
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/state"
)

type stateRepository struct {
	collector *state.Collector
}

func NewStateRepository(collector *state.Collector) domain.StateRepository {
	return &stateRepository{collector: collector}
}

func (s *stateRepository) Update(serverID int, update domain.PartialStateUpdate) error {
	return s.collector.Update(serverID, update)
}

func (s *stateRepository) Current() []domain.ServerState {
	return s.collector.Current()
}
