package servers

import (
	"context"
	"log/slog"
	"time"

	"github.com/leighmacdonald/gbans/internal/domain"
)

func NewServersUsecase(repository ServersRepository) *ServersUsecase {
	return &ServersUsecase{repository: repository}
}

type ServersUsecase struct {
	repository ServersRepository
}

// Delete performs a soft delete of the server. We use soft deleted because we dont wand to delete all the relationships
// that rely on this suchs a stats.
func (s *ServersUsecase) Delete(ctx context.Context, serverID int) error {
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

func (s *ServersUsecase) Server(ctx context.Context, serverID int) (Server, error) {
	if serverID <= 0 {
		return Server{}, domain.ErrGetServer
	}

	return s.repository.GetServer(ctx, serverID)
}

func (s *ServersUsecase) ServerPermissions(ctx context.Context) ([]ServerPermission, error) {
	return s.repository.GetServerPermissions(ctx)
}

func (s *ServersUsecase) Servers(ctx context.Context, filter ServerQueryFilter) ([]Server, int64, error) {
	return s.repository.GetServers(ctx, filter)
}

func (s *ServersUsecase) GetByName(ctx context.Context, serverName string, server *Server, disabledOk bool, deletedOk bool) error {
	return s.repository.GetServerByName(ctx, serverName, server, disabledOk, deletedOk)
}

func (s *ServersUsecase) GetByPassword(ctx context.Context, serverPassword string, server *Server, disabledOk bool, deletedOk bool) error {
	return s.repository.GetServerByPassword(ctx, serverPassword, server, disabledOk, deletedOk)
}

func (s *ServersUsecase) Save(ctx context.Context, req RequestServerUpdate) (Server, error) {
	var server Server

	if req.ServerID > 0 {
		existingServer, errServer := s.Server(ctx, req.ServerID)
		if errServer != nil {
			return Server{}, errServer
		}
		server = existingServer
		server.UpdatedOn = time.Now()
	} else {
		server = NewServer(req.ServerNameShort, req.Host, req.Port)
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
		return Server{}, err
	}

	if req.ServerID > 0 {
		slog.Info("Updated server successfully", slog.String("name", server.ShortName))
	} else {
		slog.Info("Created new server", slog.String("name", server.ShortName), slog.Int("server_id", server.ServerID))
	}

	return s.Server(ctx, server.ServerID)
}
