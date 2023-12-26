package app

import (
	"context"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/pkg/errors"
)

// NetworkBlocker provides a simple interface for blocking users connecting from banned IPs. Its designed to
// download list of, for example, VPN CIDR blocks, parse them and block any ip that is contained within any of those
// network blocks.
//
// IPs can be individually whitelisted if a remote/3rd party source cannot be changed.
type NetworkBlocker struct {
	cidrRx      *regexp.Regexp
	blocks      map[string][]*net.IPNet
	whitelisted []*net.IPNet
	sync.RWMutex
}

func NewNetworkBlocker() *NetworkBlocker {
	return &NetworkBlocker{
		blocks: make(map[string][]*net.IPNet),
		cidrRx: regexp.MustCompile(`^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(/(3[0-2]|2[0-9]|1[0-9]|[0-9]))?$`),
	}
}

func (b *NetworkBlocker) IsMatch(addr net.IP) (bool, string) {
	b.RLock()
	defer b.RUnlock()

	for _, whitelisted := range b.whitelisted {
		if whitelisted.Contains(addr) {
			return false, ""
		}
	}

	for name, networks := range b.blocks {
		for _, block := range networks {
			if block.Contains(addr) {
				return true, name
			}
		}
	}

	return false, ""
}

func (b *NetworkBlocker) RemoveSource(name string) {
	b.Lock()
	defer b.Unlock()

	delete(b.blocks, name)
}

func (b *NetworkBlocker) AddWhitelist(network *net.IPNet) {
	b.RLock()
	defer b.RUnlock()

	for _, existing := range b.whitelisted {
		if existing.String() == network.String() {
			return
		}
	}

	b.whitelisted = append(b.whitelisted, network)
}

func (b *NetworkBlocker) AddRemoteSource(ctx context.Context, name string, url string) (int64, error) {
	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if errReq != nil {
		return 0, errors.Wrap(errReq, "Invalid request")
	}

	client := util.NewHTTPClient()

	resp, errResp := client.Do(req)
	if errResp != nil {
		return 0, errors.Wrap(errResp, "Failed to fetch remote block source")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return 0, errors.Errorf("Invalid response code; %d", resp.StatusCode)
	}

	bodyBytes, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		return 0, errors.Wrap(errRead, "Failed to read response body")
	}

	var blocks []*net.IPNet //nolint:prealloc

	for _, line := range strings.Split(string(bodyBytes), "\n") {
		trimmed := strings.TrimSpace(line)
		if !b.cidrRx.MatchString(trimmed) {
			continue
		}

		_, cidrBlock, errBlock := net.ParseCIDR(trimmed)
		if errBlock != nil {
			continue
		}

		blocks = append(blocks, cidrBlock)
	}

	b.Lock()
	b.blocks[name] = blocks
	b.Unlock()

	return int64(len(blocks)), nil
}
