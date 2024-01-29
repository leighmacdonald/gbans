package state

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/domain"
)

type stateRepository struct {
	collector *Collector
}

func NewStateRepository(collector *Collector) domain.StateRepository {
	return &stateRepository{collector: collector}
}

func (s *stateRepository) GetServer(serverID int) (domain.ServerConfig, error) {
	return s.collector.GetServer(serverID)
}

func (s *stateRepository) Configs() []domain.ServerConfig {
	return s.collector.Configs()
}

func (s *stateRepository) ExecRaw(ctx context.Context, addr string, password string, cmd string) (string, error) {
	return s.ExecRaw(ctx, addr, password, cmd)
}

func (s *stateRepository) Update(serverID int, update domain.PartialStateUpdate) error {
	return s.collector.Update(serverID, update)
}

func (s *stateRepository) Current() []domain.ServerState {
	return s.collector.Current()
}
