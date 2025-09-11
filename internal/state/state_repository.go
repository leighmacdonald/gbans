package state

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/servers"
)

type StateRepository struct {
	collector *Collector
}

func NewStateRepository(collector *Collector) *StateRepository {
	return &StateRepository{collector: collector}
}

func (s *StateRepository) Start(ctx context.Context) {
	s.collector.Start(ctx)
}

func (s *StateRepository) GetServer(serverID int) (ServerConfig, error) {
	return s.collector.GetServer(serverID)
}

func (s *StateRepository) Configs() []ServerConfig {
	return s.collector.Configs()
}

func (s *StateRepository) ExecRaw(ctx context.Context, addr string, password string, cmd string) (string, error) {
	return s.collector.ExecRaw(ctx, addr, password, cmd)
}

func (s *StateRepository) Update(serverID int, update servers.PartialStateUpdate) error {
	return s.collector.Update(serverID, update)
}

func (s *StateRepository) Current() []ServerState {
	return s.collector.Current()
}
