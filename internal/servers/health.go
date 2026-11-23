package servers

import (
	"github.com/leighmacdonald/gbans/internal/health"
)

// rconRecorder receives the outcome of RCON commands. It is structurally
// implemented by *health.Tracker, keeping this package independent of the
// health service.
type rconRecorder interface {
	RecordRCON(serverID int32, err error)
}

// SetRCONRecorder wires health tracking for all current and future servers.
func (s *Servers) SetRCONRecorder(recorder rconRecorder) {
	s.serversMu.Lock()
	defer s.serversMu.Unlock()

	s.rconRecorder = recorder
	for _, server := range s.servers {
		server.rconRecorder = recorder
	}
}

// HealthSnapshot returns the dynamic state needed by the health overview.
func (s *Servers) HealthSnapshot() []health.ServerSnapshot {
	s.serversMu.RLock()
	defer s.serversMu.RUnlock()

	snapshots := make([]health.ServerSnapshot, 0, len(s.servers))
	for _, server := range s.servers {
		server.RLock()
		snapshots = append(snapshots, health.ServerSnapshot{
			ServerID:         server.ServerID,
			Name:             server.ShortName,
			LastStatusUpdate: server.state.LastUpdate,
			TokenCreatedOn:   server.TokenCreatedOn,
			STVPort:          server.state.STVPort,
			Enabled:          !server.Deleted && server.IsEnabled,
		})
		server.RUnlock()
	}

	return snapshots
}

// recordRCON reports an RCON outcome without holding any server lock.
func (s *Server) recordRCON(err error) {
	if s.rconRecorder != nil {
		s.rconRecorder.RecordRCON(s.ServerID, err)
	}
}

// applyConfig refreshes admin-editable static fields from a freshly loaded
// database row while preserving dynamic state and resolved IP addresses.
func (s *Server) applyConfig(from Server) {
	s.Lock()
	defer s.Unlock()

	s.ShortName = from.ShortName
	s.Name = from.Name
	s.Address = from.Address
	s.AddressInternal = from.AddressInternal
	s.SDREnabled = from.SDREnabled
	s.Port = from.Port
	s.RCON = from.RCON
	s.Password = from.Password
	s.STVPassword = from.STVPassword
	s.IsEnabled = from.IsEnabled
	s.Deleted = from.Deleted
	s.Region = from.Region
	s.CC = from.CC
	s.Latitude = from.Latitude
	s.Longitude = from.Longitude
	s.LogSecret = from.LogSecret
	s.EnableStats = from.EnableStats
	s.StatsBucketID = from.StatsBucketID
	s.TokenCreatedOn = from.TokenCreatedOn
	s.DiscordSeedRoleIDs = from.DiscordSeedRoleIDs
	s.DiscordSeedChannelID = from.DiscordSeedChannelID
}
