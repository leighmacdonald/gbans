package dns

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/servers/state"
)

var (
	ErrDNSUpdate      = errors.New("failed to update DNS record")
	ErrRecordNotFound = errors.New("record not found")
	errNoChange       = errors.New("no change in DNS record")
)

type Provider interface {
	// Update is responsible for updating the provided A record with the underlying DNS provider.
	// TODO Enable the automatic creation of the record if it doesn't already exist.
	Update(ctx context.Context, ip net.IP, addr string) error
}

type StateProvider interface {
	Current() []state.ServerState
}

type ServerProvider interface {
	Servers(ctx context.Context, filter servers.Query) ([]servers.Server, error)
}

func MonitorChanges(ctx context.Context, conf config.Config, state StateProvider, server ServerProvider) {
	if conf.Network.CFKey == "" || conf.Network.CFEmail == "" || conf.Network.CFZoneID == "" {
		slog.Warn("Cloudflare DNS configuration is missing, unable to update DNS records")

		return
	}

	dnsProvider := NewCloudflare(conf.Network.CFZoneID, conf.Network.CFKey, conf.Network.CFEmail)
	detector := NewChangeDetector(dnsProvider, state, server)
	detector.Start(ctx, time.Second*10)
}

type hostState struct {
	addressSDR string
	idAddr     net.IP
	lastUpdate time.Time
}

type ChangeDetector struct {
	provider     Provider
	state        StateProvider
	currentState []state.ServerState
	servers      ServerProvider
	current      map[int]hostState
	started      bool
}

func NewChangeDetector(dnsProvider Provider, state StateProvider, servers ServerProvider) *ChangeDetector {
	return &ChangeDetector{provider: dnsProvider, state: state, servers: servers, current: map[int]hostState{}}
}

func (c *ChangeDetector) findIP(serverID int) net.IP {
	for _, state := range c.currentState {
		if state.ServerID == serverID {
			return net.ParseIP(state.IP)
		}
	}

	return nil
}

// sync takes care of checking if the SDR ip of the game servers changes, and if so, it updates the DNS with the
// new ip.
func (c *ChangeDetector) sync(ctx context.Context) error {
	servers, errServers := c.servers.Servers(ctx, servers.Query{SDROnly: true})
	if errServers != nil {
		return errServers
	}

	for _, server := range servers {
		currentIP := c.findIP(server.ServerID)
		if currentIP == nil {
			continue
		}

		// Updated either on the first invocation, or on changes only.
		curHostState, found := c.current[server.ServerID]
		if !found || !curHostState.idAddr.Equal(currentIP) {
			if err := c.provider.Update(ctx, currentIP, server.Address); err != nil && !errors.Is(err, errNoChange) {
				slog.Error("Failed to update DNS record", slog.Int("server_id", server.ServerID), slog.String("error", err.Error()))

				continue
			}

			if !found {
				c.current[server.ServerID] = hostState{
					addressSDR: server.Address,
					idAddr:     currentIP,
					lastUpdate: time.Now(),
				}
			} else {
				edit := c.current[server.ServerID]
				edit.idAddr = currentIP
				edit.lastUpdate = time.Now()
				c.current[server.ServerID] = edit
			}

			slog.Info("Updated SDR DNS record", slog.String("ip", currentIP.String()), slog.String("dns", server.Address))
		}
	}

	return nil
}

func (c *ChangeDetector) Start(ctx context.Context, updateFrequency time.Duration) {
	if c.started {
		return
	}

	c.started = true
	ticker := time.NewTicker(updateFrequency)

	for {
		select {
		case <-ticker.C:
			c.currentState = c.state.Current()
			if err := c.sync(ctx); err != nil {
				slog.Error("Failed to update DNS record", slog.String("error", err.Error()))
			}
		case <-ctx.Done():
			return
		}
	}
}

type Cloudflare struct {
	client *cloudflare.Client
	zoneID string
}

func NewCloudflare(zoneID string, apiKey string, email string) *Cloudflare {
	return &Cloudflare{
		client: cloudflare.NewClient(option.WithAPIToken(apiKey), option.WithAPIEmail(email)),
		zoneID: zoneID,
	}
}

func (c *Cloudflare) findRecord(ctx context.Context, addr string) (string, net.IP, error) {
	recordResponse, err := c.client.DNS.Records.List(ctx, dns.RecordListParams{ZoneID: cloudflare.F(c.zoneID)})
	if err != nil {
		return "", nil, errors.Join(err, ErrDNSUpdate)
	}

	for _, record := range recordResponse.Result {
		if record.Name == addr {
			return record.ID, net.ParseIP(record.Content), nil
		}
	}

	return "", nil, ErrRecordNotFound
}

// https://developers.cloudflare.com/api/go/resources/dns/subresources/records/methods/update/
func (c *Cloudflare) Update(ctx context.Context, newIP net.IP, dnsAddr string) error {
	recordID, recordIP, errRecordID := c.findRecord(ctx, dnsAddr)
	if errRecordID != nil {
		return errors.Join(errRecordID, ErrDNSUpdate)
	}

	if recordIP.Equal(newIP) {
		// Record is already correct, skip update.
		return errNoChange
	}

	recordResponse, err := c.client.DNS.Records.Update(ctx, recordID, dns.RecordUpdateParams{
		ZoneID: cloudflare.F(c.zoneID),
		Body: dns.ARecordParam{
			Content: cloudflare.F(newIP.String()),
			Name:    cloudflare.F(dnsAddr),
			Type:    cloudflare.F(dns.ARecordTypeA),
		},
	},
	)
	if err != nil {
		return errors.Join(err, ErrDNSUpdate)
	}

	if recordResponse.Content != newIP.String() {
		return ErrDNSUpdate
	}

	return nil
}
