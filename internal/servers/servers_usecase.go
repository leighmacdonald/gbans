package servers

import (
	"context"

	"github.com/leighmacdonald/gbans/internal/domain"
)

type serversUsecase struct {
	serversRepo domain.ServersRepository
}

func NewServersUsecase(sr domain.ServersRepository) domain.ServersUsecase {
	return &serversUsecase{serversRepo: sr}
}

func (s *serversUsecase) GetServer(ctx context.Context, serverID int, server *domain.Server) error {
	return s.serversRepo.GetServer(ctx, serverID, server)
}

func (s *serversUsecase) GetServerPermissions(ctx context.Context) ([]domain.ServerPermission, error) {
	return s.serversRepo.GetServerPermissions(ctx)
}

func (s *serversUsecase) GetServers(ctx context.Context, filter domain.ServerQueryFilter) ([]domain.Server, int64, error) {
	return s.serversRepo.GetServers(ctx, filter)
}

func (s *serversUsecase) GetServerByName(ctx context.Context, serverName string, server *domain.Server, disabledOk bool, deletedOk bool) error {
	return s.serversRepo.GetServerByName(ctx, serverName, server, disabledOk, deletedOk)
}

func (s *serversUsecase) GetServerByPassword(ctx context.Context, serverPassword string, server *domain.Server, disabledOk bool, deletedOk bool) error {
	return s.serversRepo.GetServerByPassword(ctx, serverPassword, server, disabledOk, deletedOk)
}

func (s *serversUsecase) SaveServer(ctx context.Context, server *domain.Server) error {
	return s.serversRepo.SaveServer(ctx, server)
}

func (s *serversUsecase) DropServer(ctx context.Context, serverID int) error {
	return s.serversRepo.DropServer(ctx, serverID)
}
