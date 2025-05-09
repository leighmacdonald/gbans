package network

import (
	"context"
	"errors"
	"net"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
)

var ErrDNSUpdate = errors.New("failed to update DNS record")

type ServiceProvider interface {
	Update(ip net.IP, addr struct{}) error
}

type Cloudflare struct {
	client *cloudflare.Client
	zoneID string
}

func NewCloudflare(zoneID string, apiKey string, email string) *Cloudflare {
	return &Cloudflare{
		client: cloudflare.NewClient(
			option.WithAPIKey("144c9defac04969c7bfad8efaa8ea194"), // defaults to os.LookupEnv("CLOUDFLARE_API_KEY")
			option.WithAPIEmail("user@example.com"),               // defaults to os.LookupEnv("CLOUDFLARE_EMAIL")
		),
		zoneID: zoneID,
	}
}

func (c *Cloudflare) findRecord(ctx context.Context, zoneID string, addr string) (string, error) {
	recordResponse, err := c.client.DNS.Records.List(ctx, dns.RecordListParams{ZoneID: cloudflare.F(zoneID)})
	if err != nil {
		return "", err
	}

	for _, record := range recordResponse.Result {
		if record.Name == addr {
			return record.ID, nil
		}
	}

	return "", errors.New("record not found")
}

// https://developers.cloudflare.com/api/go/resources/dns/subresources/records/methods/update/
func (c *Cloudflare) Update(ctx context.Context, zoneID string, ip net.IP, addr string) error {
	recordID, errRecordID := c.findRecord(ctx, zoneID, addr)
	if errRecordID != nil {
		return errors.Join(errRecordID, ErrDNSUpdate)
	}

	recordResponse, err := c.client.DNS.Records.Update(ctx, recordID, dns.RecordUpdateParams{
		ZoneID: cloudflare.F(zoneID),
		Record: dns.ARecordParam{
			Content: cloudflare.F(ip.String()),
			// Name:    cloudflare.F(addr),
			Type: cloudflare.F(dns.ARecordTypeA),
		},
	},
	)
	if err != nil {
		return errors.Join(err, ErrDNSUpdate)
	}

	if recordResponse.Content != ip.String() {
		return ErrDNSUpdate
	}

	return nil
}
