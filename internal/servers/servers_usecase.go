package servers

import (
	"context"
	"log/slog"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
)

func NewServersUsecase(repository domain.ServersRepository) domain.ServersUsecase {
	return &serversUsecase{repository: repository}
}

type serversUsecase struct {
	repository domain.ServersRepository
}

// Delete performs a soft delete of the server. We use soft deleted because we dont wand to delete all the relationships
// that rely on this suchs a stats.
func (s *serversUsecase) Delete(ctx context.Context, serverID int) error {
	if serverID <= 0 {
		return domain.ErrInvalidParameter
	}

	server, errServer := s.Server(ctx, serverID)
	if errServer != nil {
		return errServer
	}

	server.Deleted = true

	if err := s.repository.SaveServer(ctx, &server); err != nil {
		return err
	}

	slog.Info("Deleted server", slog.Int("server_id", serverID))

	return nil
}

func (s *serversUsecase) Server(ctx context.Context, serverID int) (domain.Server, error) {
	if serverID <= 0 {
		return domain.Server{}, domain.ErrGetServer
	}

	return s.repository.GetServer(ctx, serverID)
}

func (s *serversUsecase) ServerPermissions(ctx context.Context) ([]domain.ServerPermission, error) {
	return s.repository.GetServerPermissions(ctx)
}

func (s *serversUsecase) Servers(ctx context.Context, filter domain.ServerQueryFilter) ([]domain.Server, int64, error) {
	return s.repository.GetServers(ctx, filter)
}

func (s *serversUsecase) GetByName(ctx context.Context, serverName string, server *domain.Server, disabledOk bool, deletedOk bool) error {
	return s.repository.GetServerByName(ctx, serverName, server, disabledOk, deletedOk)
}

func (s *serversUsecase) GetByPassword(ctx context.Context, serverPassword string, server *domain.Server, disabledOk bool, deletedOk bool) error {
	return s.repository.GetServerByPassword(ctx, serverPassword, server, disabledOk, deletedOk)
}

func (s *serversUsecase) Save(ctx context.Context, req domain.RequestServerUpdate) (domain.Server, error) {
	var server domain.Server

	if req.ServerID > 0 {
		existingServer, errServer := s.Server(ctx, req.ServerID)
		if errServer != nil {
			return domain.Server{}, errServer
		}
		server = existingServer
		server.UpdatedOn = time.Now()
	} else {
		server = domain.NewServer(req.ServerNameShort, req.Host, req.Port)
	}

	server.ShortName = req.ServerNameShort
	server.Name = req.ServerName
	server.Address = req.Host
	server.Port = req.Port
	server.ReservedSlots = req.ReservedSlots
	server.RCON = req.RCON
	server.Password = req.Password
	server.Latitude = req.Lat
	server.Longitude = req.Lon
	server.CC = req.CC
	server.Region = req.Region
	server.IsEnabled = req.IsEnabled
	server.LogSecret = req.LogSecret
	server.EnableStats = req.EnableStats
	server.AddressInternal = req.AddressInternal
	server.SDREnabled = req.SDREnabled

	if err := s.repository.SaveServer(ctx, &server); err != nil {
		return domain.Server{}, err
	}

	if req.ServerID > 0 {
		slog.Info("Updated server successfully", slog.String("name", server.ShortName))
	} else {
		slog.Info("Created new server", slog.String("name", server.ShortName), slog.Int("server_id", server.ServerID))
	}

	return s.Server(ctx, server.ServerID)
}
