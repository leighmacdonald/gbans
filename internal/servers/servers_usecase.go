package servers

import (
	"context"
	"log/slog"

	"github.com/leighmacdonald/gbans/internal/domain"
)

type serversUsecase struct {
	serversRepo domain.ServersRepository
}

func NewServersUsecase(repository domain.ServersRepository) domain.ServersUsecase {
	return &serversUsecase{serversRepo: repository}
}

func (s *serversUsecase) GetServer(ctx context.Context, serverID int) (domain.Server, error) {
	if serverID <= 0 {
		return domain.Server{}, domain.ErrGetServer
	}

	return s.serversRepo.GetServer(ctx, serverID)
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
	isNew := server.ServerID == 0

	if err := s.serversRepo.SaveServer(ctx, server); err != nil {
		return err
	}

	if isNew {
		slog.Info("Server config created", slog.Int("server_id", server.ServerID), slog.String("name", server.ShortName))
	} else {
		slog.Info("Server config updated", slog.Int("server_id", server.ServerID), slog.String("name", server.ShortName), slog.Bool("deleted", server.Deleted))
	}

	return nil
}
